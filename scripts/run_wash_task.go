// 修复后的 run_wash_task.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
    "time"

	solanaGo "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/programs/system"
	// token "github.com/gagliardetto/solana-go/programs/token"
	// ata "github.com/gagliardetto/solana-go/programs/associated-token-account"

	"marketcontrol/pkg/config"
	"marketcontrol/internal/models"
	"marketcontrol/pkg/solana"
)

func waitForConfirmation(client *rpc.Client, sig solanaGo.Signature, timeout time.Duration) (bool, error) {
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

        time.Sleep(2 * time.Second) // 每 2 秒轮询一次
    }

    return false, fmt.Errorf("超时未确认")
}

func main() {
	manageID := flag.Uint("manage-id", 0, "wash task manage ID")
	flag.Parse()

	if *manageID == 0 {
		log.Fatal("请提供有效的 wash task manage ID")
	}

	config.InitDB()

	var tasks []models.WashTask
	if err := config.DB.Where("wash_task_manage_id = ?", *manageID).Order("sort_id asc").Find(&tasks).Error; err != nil {
		log.Fatalf("获取任务失败: %v", err)
	}
	if len(tasks) == 0 {
		log.Fatal("未找到相关任务")
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		log.Fatal("Solana RPC endpoint not configured")
	}

	// Create client
	client := rpc.New(solanaRPC)

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].SortID < tasks[j].SortID
	})

	km := solana.NewKeyManager()

	for _, task := range tasks {
		if task.IsSuccess {
			log.Printf("任务 %d 已完成，跳过", task.ID)
			continue
		}

		var fromNode models.AddressNode
		if err := config.DB.Where("id = ?", task.FromNodeID).First(&fromNode).Error; err != nil {
			log.Printf("获取发送方节点信息失败: %v", err)
			continue
		}

		if fromNode.NodeType == models.NodeTypeLeaf {
			log.Printf("> 叶节点 跳过")
			continue
		} 

		var sendAddressManage models.AddressManage
		if err := config.DB.Where("address = ?", task.FromAddress).First(&sendAddressManage).Error; err != nil {
			log.Printf("获取地址 %s 的私钥失败: %v", task.FromAddress, err)
			continue
		}

		fromPubkey, err := solanaGo.PublicKeyFromBase58(task.FromAddress)
		if err != nil {
			log.Printf("无效的发送地址 %s: %v", task.FromAddress, err)
			continue
		}

		encryptPassword := os.Getenv("ENCRYPTPASSWORD")
		if encryptPassword == "" {
			log.Printf("未设置 ENCRYPTPASSWORD 环境变量")
			continue
		}

		decryptedPrivateKey, err := km.DecryptPrivateKey(sendAddressManage.PrivateKey, encryptPassword)
		if err != nil {
			log.Printf("解密私钥失败: %v", err)
			continue
		}

		privateKey := solanaGo.PrivateKey(decryptedPrivateKey)

		toPubkey, err := solanaGo.PublicKeyFromBase58(task.ToAddress)
		if err != nil {
			log.Printf("无效的接收地址 %s: %v", task.ToAddress, err)
			continue
		}

		// 计算实际可发送的金额
		minBalance := uint64(8000) // 最小转账 gas
		// minBalance := uint64(1500000) // 最小转账 gas
		var transferInstruction solanaGo.Instruction
		var toBalance uint64 // Declare toBalance at a higher scope
		if task.SendToken == "sol" {
			// 查询发送方账户的 SOL 余额
			balanceResult, err := client.GetBalance(
				context.Background(),
				fromPubkey,
				rpc.CommitmentFinalized,
			)
			if err != nil {
				log.Printf("获取账户余额失败: %v", err)
				continue
			}

			balance := balanceResult.Value

			// 获取接收方账户的初始 SOL 余额
			toBalanceResult, err := client.GetBalance(
				context.Background(),
				toPubkey,
				rpc.CommitmentFinalized,
			)
			if err != nil {
				log.Printf("获取接收方余额失败: %v", err)
				continue
			}
			toBalance = toBalanceResult.Value // Remove the := and use = instead
			log.Printf("> 接收方初始余额: %d lamports", toBalance)

			// 获取发送方节点信息
			// var fromNode models.AddressNode
			// if err := config.DB.Where("id = ?", task.FromNodeID).First(&fromNode).Error; err != nil {
			// 	log.Printf("获取发送方节点信息失败: %v", err)
			// 	continue
			// }

			if balance < minBalance {
				log.Printf("> 账户余额不足，当前余额: %d lamports, 帐户：%s", balance, fromPubkey.String())
				continue
			}

			// 如果是中间节点，发送全部余额（减去最小保留额）
			if fromNode.NodeType == models.NodeTypeIntermediate {
				task.SendAmount = balance - minBalance
			// 	log.Printf("> 中间节点，调整发送金额为全部余额: %d lamports", task.SendAmount)
			} else if balance < task.SendAmount + minBalance {
				// 其他节点类型，保持原有的余额检查逻辑
				if balance <= minBalance {
					log.Printf("账户余额不足，当前余额: %d lamports", balance)
					continue
				}
				task.SendAmount = balance - minBalance
			}

			// fmt.Println("> SOL:")
			// fmt.Println("> fromPubkey:", fromPubkey.String())
			// fmt.Println("> toPubkey:", toPubkey.String())
			// fmt.Println("> balance:", balance)
			// fmt.Println("> task.SendAmount:", task.SendAmount)

			transferInstruction = system.NewTransferInstruction(
				task.SendAmount,
				fromPubkey,
				toPubkey,
			).Build()
		} else {
			// mint := solanaGo.MustPublicKeyFromBase58(task.SendToken)
			// fromTokenAccount, err := ata.GetAssociatedTokenAddress(fromPubkey, mint)
			// if err != nil {
			// 	log.Printf("获取发送方代币账户失败: %v", err)
			// 	continue
			// }
			// toTokenAccount, err := ata.GetAssociatedTokenAddress(toPubkey, mint)
			// if err != nil {
			// 	log.Printf("获取接收方代币账户失败: %v", err)
			// 	continue
			// }
			// transferInstruction = token.NewTransferInstruction(
			// 	task.SendAmount,
			// 	fromTokenAccount,
			// 	toTokenAccount,
			// 	fromPubkey,
			// 	[]solanaGo.PublicKey{},
			// ).Build()
		}

		recent, err := client.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
		if err != nil {
			log.Printf("> 获取最新区块哈希失败: %v", err)
			continue
		}

		tx, err := solanaGo.NewTransaction(
			[]solanaGo.Instruction{transferInstruction},
			recent.Value.Blockhash,
			solanaGo.TransactionPayer(fromPubkey),
		)
		if err != nil {
			log.Printf("> 创建交易失败: %v", err)
			continue
		}

		_, err = tx.Sign(func(key solanaGo.PublicKey) *solanaGo.PrivateKey {
			if key.Equals(fromPubkey) {
				return &privateKey
			}
			return nil
		})
		if err != nil {
			log.Printf("> 交易签名失败: %v", err)
			continue
		}

		sig, err := client.SendTransaction(context.Background(), tx)
		if err != nil {
			log.Printf("> 发送交易失败: %v", err)
			continue
		}
		fmt.Println("> sig:", sig.String())

		ok, err := waitForConfirmation(client, sig, 30 * time.Second)
		if err != nil || !ok {
			log.Printf("交易确认失败: %v", err)
			continue
		}

		// 检查接收方余额变化
		toBalanceAfterResult, err := client.GetBalance(
			context.Background(),
			toPubkey,
			rpc.CommitmentFinalized,
		)
		if err != nil {
			log.Printf("获取接收方最终余额失败: %v", err)
			break
		}
		toBalanceAfter := toBalanceAfterResult.Value
		log.Printf("> 接收方最终余额: %d lamports", toBalanceAfter)

		// 只有当余额确实发生变化时，才更新任务状态为成功
		if toBalanceAfter > toBalance {
			if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
				"signature":   sig.String(),
				"is_success": true,
				"send_amount": task.SendAmount,
			}).Error; err != nil {
				log.Printf("> 更新任务状态失败: %v", err)
				break
			}
			log.Printf("> 任务 %d 完成，交易签名: %s，接收方余额增加: %d lamports", 
				task.ID, sig.String(), toBalanceAfter-toBalance)
		} else {
			log.Printf("> 交易可能失败，接收方余额未增加")
			break
		}

		if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"signature":   sig.String(),
			"is_success": true,
			"send_amount": task.SendAmount, // 更新实际发送的金额
		}).Error; err != nil {
			log.Printf("> 更新任务状态失败: %v", err)
			break
		}

		log.Printf("> 任务 %d 完成，交易签名: %s", task.ID, sig.String())
		// break
	}
}