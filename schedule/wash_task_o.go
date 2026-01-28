package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
	"math"

	solanaGo "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/programs/system"

	"marketcontrol/pkg/config"
	"marketcontrol/internal/models"
	"marketcontrol/pkg/solana"
)

func WaitForConfirmation(client *rpc.Client, sig solanaGo.Signature, timeout time.Duration) (bool, error) {
    deadline := time.Now().Add(timeout)

	fmt.Println("> 等待确认...")
    for time.Now().Before(deadline) {
        status, err := client.GetSignatureStatuses(context.Background(), true, sig)
        if err != nil {
            return false, err
        }

        if status.Value[0] != nil && status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
            return true, nil
        }

        time.Sleep(2 * time.Second)
    }

    return false, fmt.Errorf("超时未确认")
}

func ProcessTask(client *rpc.Client, task *models.WashTask, km *solana.KeyManager, minGas uint64) error {
	if task.IsSuccess {
		log.Printf("任务 %d 已完成，跳过", task.ID)
		return nil
	}

	// 更新任务状态为处理中
	if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).
		Update("status", "processing").Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %v", err)
	}

	var fromNode models.AddressNode
	if err := config.DB.Where("id = ?", task.FromNodeID).First(&fromNode).Error; err != nil {
		return fmt.Errorf("获取发送方节点信息失败: %v", err)
	}

	if fromNode.NodeType == models.NodeTypeLeaf {
		log.Printf("> 叶节点 跳过")
		return nil
	}

	var sendAddressManage models.AddressManage
	if err := config.DB.Where("address = ?", task.FromAddress).First(&sendAddressManage).Error; err != nil {
		return fmt.Errorf("获取地址 %s 的私钥失败: %v", task.FromAddress, err)
	}

	fromPubkey, err := solanaGo.PublicKeyFromBase58(task.FromAddress)
	if err != nil {
		return fmt.Errorf("无效的发送地址 %s: %v", task.FromAddress, err)
	}

	encryptPassword := os.Getenv("ENCRYPTPASSWORD")
	if encryptPassword == "" {
		return fmt.Errorf("未设置 ENCRYPTPASSWORD 环境变量")
	}

	decryptedPrivateKey, err := km.DecryptPrivateKey(sendAddressManage.PrivateKey, encryptPassword)
	if err != nil {
		return fmt.Errorf("解密私钥失败: %v", err)
	}

	privateKey := solanaGo.PrivateKey(decryptedPrivateKey)

	toPubkey, err := solanaGo.PublicKeyFromBase58(task.ToAddress)
	if err != nil {
		return fmt.Errorf("无效的接收地址 %s: %v", task.ToAddress, err)
	}

	var transferInstruction solanaGo.Instruction
	var toBalance uint64

	if task.SendToken == "sol" {
		balanceResult, err := client.GetBalance(
			context.Background(),
			fromPubkey,
			rpc.CommitmentFinalized,
		)
		if err != nil {
			return fmt.Errorf("获取账户余额失败: %v", err)
		}

		balance := balanceResult.Value

		toBalanceResult, err := client.GetBalance(
			context.Background(),
			toPubkey,
			rpc.CommitmentFinalized,
		)
		if err != nil {
			return fmt.Errorf("获取接收方余额失败: %v", err)
		}
		toBalance = toBalanceResult.Value
		log.Printf("> 接收方初始余额: %d lamports", toBalance)

		if balance < minGas {
			return fmt.Errorf("账户余额不足，当前余额: %d lamports", balance)
		}

		if fromNode.NodeType == models.NodeTypeIntermediate {
			task.SendAmount = balance - minGas
		} else if balance < task.SendAmount+minGas {
			if balance <= minGas {
				return fmt.Errorf("账户余额不足，当前余额: %d lamports", balance)
			}
			task.SendAmount = balance - minGas
		}

		transferInstruction = system.NewTransferInstruction(
			task.SendAmount,
			fromPubkey,
			toPubkey,
		).Build()
	}

	recent, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return fmt.Errorf("获取最新区块哈希失败: %v", err)
	}

	tx, err := solanaGo.NewTransaction(
		[]solanaGo.Instruction{transferInstruction},
		recent.Value.Blockhash,
		solanaGo.TransactionPayer(fromPubkey),
	)
	if err != nil {
		return fmt.Errorf("创建交易失败: %v", err)
	}

	_, err = tx.Sign(func(key solanaGo.PublicKey) *solanaGo.PrivateKey {
		if key.Equals(fromPubkey) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("交易签名失败: %v", err)
	}

	sig, err := client.SendTransaction(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("发送交易失败: %v", err)
	}
	fmt.Println("> sig:", sig.String())

	ok, err := WaitForConfirmation(client, sig, 30*time.Second)
	if err != nil || !ok {
		return fmt.Errorf("交易确认失败: %v", err)
	}

	toBalanceAfterResult, err := client.GetBalance(
		context.Background(),
		toPubkey,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return fmt.Errorf("获取接收方最终余额失败: %v", err)
	}
	toBalanceAfter := toBalanceAfterResult.Value
	log.Printf("> 接收方最终余额: %d lamports", toBalanceAfter)

	if toBalanceAfter > toBalance {
		if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"signature":   sig.String(),
			"is_success": true,
			"send_amount": task.SendAmount,
			"status":     "processed",
		}).Error; err != nil {
			return fmt.Errorf("更新任务状态失败: %v", err)
		}
		log.Printf("> 任务 %d 完成，交易签名: %s，接收方余额增加: %d lamports",
			task.ID, sig.String(), toBalanceAfter-toBalance)
	} else {
		// 更新任务状态为失败
		if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).
			Update("status", "failed").Error; err != nil {
			log.Printf("更新任务状态失败: %v", err)
		}
		return fmt.Errorf("交易可能失败，接收方余额未增加")
	}

	return nil
}

func verify_all_task_done(taskManageID uint) bool {
	var tasks []models.WashTask
	if err := config.DB.Where("wash_task_manage_id = ?", taskManageID).Find(&tasks).Error; err != nil {
		log.Printf("验证任务完成状态失败: %v", err)
		return false
	}

	if len(tasks) == 0 {
		return false
	}

	for _, task := range tasks {
		if !task.IsSuccess || task.Status != "processed" {
			return false
		}
	}

	// 更新任务管理的状态
	if err := config.DB.Model(&models.WashTaskManage{}).
		Where("id = ?", taskManageID).
		Updates(map[string]interface{}{
			"enabled": false,
			"status": "processed",
		}).Error; err != nil {
		log.Printf("更新任务管理状态失败: %v", err)
	}

	return true
}

func main() {
	config.InitDB()

	for {
		log.Println("> 开始 wash_wask 任务...")
		var taskManages []models.WashTaskManage
		if err := config.DB.Where("enabled = ?", true).Find(&taskManages).Error; err != nil {
			log.Printf("获取任务管理列表失败: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, taskManage := range taskManages {
			// 更新任务管理状态为处理中
			if err := config.DB.Model(&models.WashTaskManage{}).
				Where("id = ?", taskManage.ID).
				Update("status", "processing").Error; err != nil {
				log.Printf("更新任务管理状态失败: %v", err)
				continue
			}

			client := rpc.New(taskManage.Endpoint)

			var tasks []models.WashTask
			if err := config.DB.Where("wash_task_manage_id = ?", taskManage.ID).
				Order("sort_id asc").Find(&tasks).Error; err != nil {
				log.Printf("获取任务列表失败: %v", err)
				continue
			}

			if len(tasks) == 0 {
				continue
			}

			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].SortID < tasks[j].SortID
			})

			minGas := uint64(taskManage.TaskGas * math.Pow10(int(taskManage.TokenDecimals)))

			km := solana.NewKeyManager()

			for _, task := range tasks {
				if err := ProcessTask(client, &task, km, minGas); err != nil {
					log.Printf("处理任务失败: %v", err)
					if updateErr := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).Update("status", "failed").Error; updateErr != nil {
						log.Printf("设置任务失败状态时出错: %v", updateErr)
					}
					continue
				}
			}

			// 检查所有任务是否完成，如果完成则禁用该任务管理
			if verify_all_task_done(taskManage.ID) {
				if err := config.DB.Model(&models.WashTaskManage{}).
					Where("id = ?", taskManage.ID).
					Update("enabled", false).Error; err != nil {
					log.Printf("更新任务管理状态失败: %v", err)
				} else {
					log.Printf("任务管理 %d 的所有任务已完成，已禁用", taskManage.ID)
				}
			}
		}

		log.Println("> 任务执行完毕， sleep:  ", time.Minute)
		time.Sleep(time.Minute)
	}
}