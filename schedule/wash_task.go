package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"gorm.io/gorm"

	solanaGo "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	"marketcontrol/pkg/solana"
)

type MapParamsType struct {
	Count     int    `json:"count"`
	Depth     int    `json:"depth"`
	RootLabel string `json:"rootLabel"`
}

// ExecuteWashTasks 并发执行转账任务
func ExecuteWashTasks(tasks []models.WashTask, client *rpc.Client, km *solana.KeyManager, minGas uint64) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 限制并发数为3

	for _, task := range tasks {
		wg.Add(1)
		go func(t models.WashTask) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 限制执行速度，0.05秒
			time.Sleep(10 * time.Second)

			if err := ProcessTask(client, &t, km, minGas); err != nil {
				log.Printf("处理任务失败 [ID:%d]: %v", t.ID, err)
			}
		}(task)
	}

	wg.Wait()
}

// CreateFirstDepthWashTasks 创建第一层深度的转账任务
func CreateFirstDepthWashTasks(addressNodesFromMap []models.AddressNode, rootNode models.AddressNode, taskManageID uint) []models.WashTask {
	var firstDepthWashTasks []models.WashTask

	// 筛选出所有 NodeDepthID 为 1 的数据
	var addressNodeFromFirstDepth []models.AddressNode
	for _, node := range addressNodesFromMap {
		if node.NodeDepthID == 1 {
			addressNodeFromFirstDepth = append(addressNodeFromFirstDepth, node)
		}
	}

	// 获取对应的 WashTask
	for _, node := range addressNodeFromFirstDepth {
		var task models.WashTask
		if err := config.DB.Where("wash_task_manage_id = ? AND from_node_id = ? AND to_node_id = ? AND is_success = ? AND status = ?",
			taskManageID, rootNode.ID, node.ID, false, "unprocessed").First(&task).Error; err == nil {
			firstDepthWashTasks = append(firstDepthWashTasks, task)
		}
	}

	return firstDepthWashTasks
}

// CreateOtherDepthWashTasks 创建其他深度的转账任务
func CreateOtherDepthWashTasks(addressNodesFromMap []models.AddressNode, nodeChain, nodeDepth int, taskManageID uint, client *rpc.Client) []models.WashTask {
	var otherDepthWashTasks []models.WashTask

	if nodeDepth <= 1 {
		return otherDepthWashTasks
	}

	// 遍历每条链路
	for chainID := 1; chainID <= nodeChain; chainID++ {
		// 遍历每个深度
		for depthID := 1; depthID < nodeDepth; depthID++ {
			// 获取当前深度的FromNode
			var fromNode models.AddressNode
			for _, node := range addressNodesFromMap {
				if node.NodeChainID == chainID && node.NodeDepthID == depthID {
					fromNode = node
					break
				}
			}

			// 获取下一深度的ToNode
			var toNode models.AddressNode
			for _, node := range addressNodesFromMap {
				if node.NodeChainID == chainID && node.NodeDepthID == depthID+1 {
					toNode = node
					break
				}
			}

			// 如果找到了对应的节点
			if fromNode.ID > 0 && toNode.ID > 0 {
				// 检查 fromNode 的余额
				fromPubkey, err := solanaGo.PublicKeyFromBase58(fromNode.NodeValue)
				if err != nil {
					log.Printf("无效的发送地址 %s: %v", fromNode.NodeValue, err)
					continue
				}

				balance, err := client.GetBalance(
					context.Background(),
					fromPubkey,
					rpc.CommitmentFinalized,
				)
				if err != nil {
					log.Printf("获取账户余额失败 [地址:%s]: %v", fromNode.NodeValue, err)
					continue
				}

				// 更新钱包代币统计
				if err := UpdateWalletTokenStatLocal(fromNode.NodeValue, "sol", 9, balance.Value, balance.Context.Slot, time.Now()); err != nil {
					log.Printf("更新钱包代币统计失败 [地址:%s]: %v", fromNode.NodeValue, err)
				}

				// 只有当余额大于0时才添加任务
				if balance.Value > 0 {
					var task models.WashTask
					if err := config.DB.Where("wash_task_manage_id = ? AND from_node_id = ? AND to_node_id = ? AND is_success = ? AND status = ?",
						taskManageID, fromNode.ID, toNode.ID, false, "unprocessed").First(&task).Error; err == nil {
						otherDepthWashTasks = append(otherDepthWashTasks, task)
						log.Printf("> 添加任务 [FromNode:%s, Balance:%d]", fromNode.NodeValue, balance.Value)
						break // 找到任务就跳出当前深度循环
					}
				} else {
					log.Printf("> 跳过任务 [FromNode:%s] - 余额为0", fromNode.NodeValue)
				}
			}
		}
	}

	return otherDepthWashTasks
}

// 全局变量，用于记录任务的初始余额
var taskInitialBalances = make(map[uint]uint64)

// 全局变量，用于记录任务的下次验证时间
var taskVerifyTime = make(map[uint]time.Time)

// 互斥锁，用于保护全局 map
var (
	balancesMutex   sync.RWMutex
	verifyTimeMutex sync.RWMutex
)

// 全局参数：过期时间，默认为 20 分钟
var ExpiredTime = 15 * time.Minute

// CloseWashTaskAfterExpired 检查并关闭过期的洗币任务
func CloseWashTaskAfterExpired(taskManage *models.WashTaskManage, client *rpc.Client) error {
	// 检查任务是否已过期
	if time.Since(taskManage.CreatedAt) > ExpiredTime {
		// 更新任务状态为超时
		if err := config.DB.Model(taskManage).Updates(map[string]interface{}{
			"enabled": false,
			"status":  "timeout",
		}).Error; err != nil {
			return fmt.Errorf("更新过期任务状态失败 [TaskManageID:%d]: %v", taskManage.ID, err)
		}

		log.Printf("> 任务已过期并关闭 [TaskManageID:%d, CreatedAt:%s, ExpiredTime:%s]",
			taskManage.ID, taskManage.CreatedAt.Format("2006-01-02 15:04:05"), ExpiredTime.String())

		// 更新地图余额，记录过期时的资金分布状态
		if err := UpdateMapBalance(client, taskManage.MapID); err != nil {
			log.Printf("更新过期任务地图余额失败 [TaskManageID:%d]: %v", taskManage.ID, err)
		} else {
			log.Printf("> 已更新过期任务的地图余额 [TaskManageID:%d, MapID:%d]", taskManage.ID, taskManage.MapID)
		}

		// 恢复根节点购买权限
		// var rootNode models.AddressNode
		// if err := config.DB.Where("map_id = ? AND node_type = ?", taskManage.MapID, models.NodeTypeRoot).First(&rootNode).Error; err == nil {
		// 	if err := ModifyRootAddressBuyPermission(rootNode.NodeValue, true); err != nil {
		// 		log.Printf("恢复过期任务根节点购买权限失败 [TaskManageID:%d]: %v", taskManage.ID, err)
		// 	} else {
		// 		log.Printf("> 已恢复过期任务根节点购买权限 [TaskManageID:%d, RootAddress:%s]", taskManage.ID, rootNode.NodeValue)
		// 	}
		// }

		return nil
	}

	return nil
}

func ProcessTask(client *rpc.Client, task *models.WashTask, km *solana.KeyManager, minGas uint64) error {
	// check task verify time
	// verifyTimeMutex.RLock()
	// nextVerifyTime, exists := taskVerifyTime[task.ID]
	// verifyTimeMutex.RUnlock()

	// if exists && !nextVerifyTime.Before(time.Now()) {
	// 	log.Printf("任务 %d 验证时间未到，跳过", task.ID)
	// 	return nil
	// }

	if task.IsSuccess {
		log.Printf("任务 %d 已完成，跳过", task.ID)
		return nil
	}

	if task.Status != "unprocessed" {
		log.Printf("任务 %d 状态不是 unprocessed，跳过", task.ID)
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

		// 更新发送方钱包代币统计
		if err := UpdateWalletTokenStatLocal(task.FromAddress, "sol", 9, balance, balanceResult.Context.Slot, time.Now()); err != nil {
			log.Printf("更新发送方钱包代币统计失败 [地址:%s]: %v", task.FromAddress, err)
		}

		// 获取接收方初始余额并保存
		toBalanceResult, err := client.GetBalance(
			context.Background(),
			toPubkey,
			rpc.CommitmentFinalized,
		)
		if err != nil {
			return fmt.Errorf("获取接收方余额失败: %v", err)
		}

		// 更新接收方钱包代币统计
		if err := UpdateWalletTokenStatLocal(task.ToAddress, "sol", 9, toBalanceResult.Value, toBalanceResult.Context.Slot, time.Now()); err != nil {
			log.Printf("更新接收方钱包代币统计失败 [地址:%s]: %v", task.ToAddress, err)
		}

		balancesMutex.Lock()
		taskInitialBalances[task.ID] = toBalanceResult.Value
		balancesMutex.Unlock()

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

		transferInstruction := system.NewTransferInstruction(
			task.SendAmount,
			fromPubkey,
			toPubkey,
		).Build()

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

		// 计算下次验证时间
		nextVerifyTime := time.Now().Add(time.Second * 15)
		verifyTimeMutex.Lock()
		taskVerifyTime[task.ID] = nextVerifyTime
		verifyTimeMutex.Unlock()

		sig, err := client.SendTransaction(context.Background(), tx)
		if err != nil {
			return fmt.Errorf("发送交易失败: %v", err)
		}
		fmt.Println("> sig:", sig.String())

		// 更新任务状态，记录签名
		if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"signature":   sig.String(),
			"is_success":  false,
			"send_amount": task.SendAmount,
			"status":      "processing",
		}).Error; err != nil {
			return fmt.Errorf("更新任务状态失败: %v", err)
		}
	}

	return nil
}

func WaitForConfirmationAndVerify(client *rpc.Client, taskManageID uint) error {
	var tasks []models.WashTask
	if err := config.DB.Where("wash_task_manage_id = ? AND signature != '' AND is_success = ? AND status = ?",
		taskManageID, false, "processing").Find(&tasks).Error; err != nil {
		return fmt.Errorf("获取待确认任务失败: %v", err)
	}

	for _, task := range tasks {
		// 检查是否需要验证
		verifyTimeMutex.RLock()
		nextVerifyTime, exists := taskVerifyTime[task.ID]
		verifyTimeMutex.RUnlock()

		if !exists || nextVerifyTime.After(time.Now()) {
			log.Printf("> 任务 %d 验证时间未到，跳过", task.ID)
			continue
		}

		sig := solanaGo.MustSignatureFromBase58(task.Signature)

		// 检查交易确认状态
		ok, err := WaitForConfirmation(client, sig, 30*time.Second)
		if err != nil || !ok {
			log.Printf("任务 %d 交易确认失败: %v", task.ID, err)
			continue
		}

		if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
			"signature":   task.Signature,
			"is_success":  true,
			"send_amount": task.SendAmount,
			"status":      "processed",
		}).Error; err != nil {
			log.Printf("更新任务状态失败: %v", err)
			continue
		}

		// // 获取接收方当前余额
		// toPubkey, err := solanaGo.PublicKeyFromBase58(task.ToAddress)
		// if err != nil {
		// 	log.Printf("无效的接收地址 %s: %v", task.ToAddress, err)
		// 	continue
		// }

		// // 获取接收方最终余额
		// toBalanceAfterResult, err := client.GetBalance(
		// 	context.Background(),
		// 	toPubkey,
		// 	rpc.CommitmentFinalized,
		// )
		// if err != nil {
		// 	log.Printf("获取接收方最终余额失败: %v", err)
		// 	continue
		// }

		// // 获取初始余额
		// balancesMutex.RLock()
		// initialBalance, exists := taskInitialBalances[task.ID]
		// balancesMutex.RUnlock()

		// if !exists {
		// 	log.Printf("未找到任务 %d 的初始余额记录", task.ID)
		// 	continue
		// }

		// // 计算余额变化
		// diffBalance := toBalanceAfterResult.Value - initialBalance
		// if diffBalance > 0 {
		// 	if err := config.DB.Model(&models.WashTask{}).Where("id = ?", task.ID).Updates(map[string]interface{}{
		// 		"signature":   task.Signature,
		// 		"is_success": true,
		// 		"send_amount": diffBalance,
		// 		"status":     "processed",
		// 	}).Error; err != nil {
		// 		log.Printf("更新任务状态失败: %v", err)
		// 		continue
		// 	}
		// 	log.Printf("> 任务 %d 完成，交易签名: %s，接收方余额增加: %d lamports",
		// 		task.ID, task.Signature, diffBalance)

		// 	// 清理初始余额记录和验证时间记录
		// 	balancesMutex.Lock()
		// 	delete(taskInitialBalances, task.ID)
		// 	balancesMutex.Unlock()

		// 	verifyTimeMutex.Lock()
		// 	delete(taskVerifyTime, task.ID)
		// 	verifyTimeMutex.Unlock()
		// }
	}

	// 重置未发送但状态为 processing 的任务为 unprocessed
	if err := HandleUnsendTask(taskManageID); err != nil {
		log.Printf("重置未发送任务状态失败: %v", err)
	}

	return nil
}

// HandleUnsendTask 重置未发送但状态为 processing 的任务为 unprocessed
func HandleUnsendTask(taskManageID uint) error {
	result := config.DB.Model(&models.WashTask{}).
		Where("wash_task_manage_id = ? AND signature = '' AND is_success = ? AND status = ?", taskManageID, false, "processing").
		Update("status", "unprocessed")

	if result.Error != nil {
		return fmt.Errorf("重置未发送任务状态失败: %v", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("> 已重置未发送任务状态为 unprocessed，共 %d 条 [TaskManageID:%d]", result.RowsAffected, taskManageID)
	}

	return nil
}

func UpdateAllLeafBalance(client *rpc.Client, mapID uint) error {
	// 获取所有叶子节点
	var leafNodes []models.AddressNode
	if err := config.DB.Where("map_id = ? AND node_type = ?", mapID, models.NodeTypeLeaf).Find(&leafNodes).Error; err != nil {
		return fmt.Errorf("获取叶子节点失败: %v", err)
	}

	if len(leafNodes) == 0 {
		return fmt.Errorf("未找到叶子节点")
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 限制并发数为5

	for _, node := range leafNodes {
		wg.Add(1)
		go func(node models.AddressNode) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 限制执行速度
			time.Sleep(50 * time.Millisecond)

			// 获取节点余额
			pubkey, err := solanaGo.PublicKeyFromBase58(node.NodeValue)
			if err != nil {
				log.Printf("无效的地址 %s: %v", node.NodeValue, err)
				return
			}

			balance, err := client.GetBalance(
				context.Background(),
				pubkey,
				rpc.CommitmentFinalized,
			)
			if err != nil {
				log.Printf("获取账户余额失败 [地址:%s]: %v", node.NodeValue, err)
				return
			}

			// 更新钱包代币统计
			if err := UpdateWalletTokenStatLocal(node.NodeValue, "sol", 9, balance.Value, balance.Context.Slot, time.Now()); err != nil {
				log.Printf("更新钱包代币统计失败 [地址:%s]: %v", node.NodeValue, err)
				return
			}

			log.Printf("> 更新叶子节点余额成功 [地址:%s, 余额:%d]", node.NodeValue, balance.Value)
		}(node)
	}

	wg.Wait()
	return nil
}

// UpdateMapBalance 更新指定 map_id 的所有节点余额
func UpdateMapBalance(client *rpc.Client, mapID uint) error {
	// 获取指定 map_id 的所有节点
	var allNodes []models.AddressNode
	if err := config.DB.Where("map_id = ?", mapID).Find(&allNodes).Error; err != nil {
		return fmt.Errorf("获取地图节点失败: %v", err)
	}

	if len(allNodes) == 0 {
		return fmt.Errorf("未找到地图节点 [MapID:%d]", mapID)
	}

	log.Printf("> 开始更新地图余额 [MapID:%d, 节点数:%d]", mapID, len(allNodes))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 限制并发数为5

	// 统计不同类型节点的数量
	var rootCount, intermediateCount, leafCount int
	for _, node := range allNodes {
		switch node.NodeType {
		case models.NodeTypeRoot:
			rootCount++
		case models.NodeTypeIntermediate:
			intermediateCount++
		case models.NodeTypeLeaf:
			leafCount++
		}
	}

	log.Printf("> 节点统计 [Root:%d, Intermediate:%d, Leaf:%d]", rootCount, intermediateCount, leafCount)

	for _, node := range allNodes {
		wg.Add(1)
		go func(node models.AddressNode) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 限制执行速度
			time.Sleep(10 * time.Second)

			// 获取节点余额
			pubkey, err := solanaGo.PublicKeyFromBase58(node.NodeValue)
			if err != nil {
				log.Printf("无效的地址 %s [NodeType:%s]: %v", node.NodeValue, node.NodeType, err)
				return
			}

			balance, err := client.GetBalance(
				context.Background(),
				pubkey,
				rpc.CommitmentFinalized,
			)
			if err != nil {
				log.Printf("获取账户余额失败 [地址:%s, NodeType:%s]: %v", node.NodeValue, node.NodeType, err)
				return
			}

			// 更新钱包代币统计
			if err := UpdateWalletTokenStatLocal(node.NodeValue, "sol", 9, balance.Value, balance.Context.Slot, time.Now()); err != nil {
				log.Printf("更新钱包代币统计失败 [地址:%s, NodeType:%s]: %v", node.NodeValue, node.NodeType, err)
				return
			}

			balanceReadable := float64(balance.Value) / math.Pow10(9)
			log.Printf("> 更新节点余额成功 [地址:%s, NodeType:%s, ChainID:%d, DepthID:%d, 余额:%d lamports / %.6f SOL]",
				node.NodeValue, node.NodeType, node.NodeChainID, node.NodeDepthID, balance.Value, balanceReadable)
		}(node)
	}

	wg.Wait()
	log.Printf("> 地图余额更新完成 [MapID:%d]", mapID)
	return nil
}

// UpdateWalletTokenStatLocal 创建或更新指定地址和代币的余额统计
func UpdateWalletTokenStatLocal(address, mint string, decimals uint, balance uint64, slot uint64, blockTime time.Time) error {
	// 查找现有的 WalletTokenStat
	var stat models.WalletTokenStat
	result := config.DB.Where("owner_address = ? AND mint = ?", address, mint).First(&stat)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询 WalletTokenStat 失败 [地址:%s]: %v", address, result.Error)
	}

	// 更新或创建 WalletTokenStat
	stat.OwnerAddress = address
	stat.Mint = mint
	stat.Decimals = decimals
	stat.Balance = balance
	stat.BalanceReadable = float64(balance) / math.Pow10(int(decimals))
	stat.Slot = slot
	stat.BlockTime = blockTime

	if result.Error == nil {
		// 记录存在，执行更新
		if err := config.DB.Save(&stat).Error; err != nil {
			return fmt.Errorf("更新 WalletTokenStat 失败 [地址:%s]: %v", address, err)
		}
	} else {
		// 记录不存在，执行创建
		if err := config.DB.Create(&stat).Error; err != nil {
			return fmt.Errorf("创建 WalletTokenStat 失败 [地址:%s]: %v", address, err)
		}
	}

	return nil
}

func ModifyRootAddressBuyPermission(rootAddress string, isBuyAllowed bool) error {
	// 获取所有活跃的项目配置
	var activeProjects []models.ProjectConfig
	if err := config.DB.Where("is_active = ?", true).Find(&activeProjects).Error; err != nil {
		return fmt.Errorf("获取活跃项目失败: %v", err)
	}

	// 遍历每个活跃项目
	for _, project := range activeProjects {
		// 获取项目对应的代币配置
		var tokenConfig models.TokenConfig
		if err := config.DB.Where("id = ?", project.TokenID).First(&tokenConfig).Error; err != nil {
			log.Printf("获取代币配置失败 [TokenID:%d]: %v", project.TokenID, err)
			continue
		}

		// 查找或创建地址配置
		var addressConfig models.AddressConfig
		result := config.DB.Where("address = ? AND mint = ?", rootAddress, tokenConfig.Mint).First(&addressConfig)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// 创建新的地址配置
				addressConfig = models.AddressConfig{
					Address:           rootAddress,
					Mint:              tokenConfig.Mint,
					IsBuyAllowed:      isBuyAllowed,
					IsSellAllowed:     true,
					IsTradeAllowed:    true,
					TradePriorityRate: 1,
				}
				if err := config.DB.Create(&addressConfig).Error; err != nil {
					log.Printf("创建地址配置失败 [Address:%s, Mint:%s]: %v", rootAddress, tokenConfig.Mint, err)
					continue
				}
			} else {
				log.Printf("查询地址配置失败: %v", result.Error)
				continue
			}
		} else {
			// 更新现有配置
			addressConfig.IsBuyAllowed = isBuyAllowed
			if err := config.DB.Save(&addressConfig).Error; err != nil {
				log.Printf("更新地址配置失败 [Address:%s, Mint:%s]: %v", rootAddress, tokenConfig.Mint, err)
				continue
			}
		}

		log.Printf("> 已更新地址配置 [Address:%s, Mint:%s]", rootAddress, tokenConfig.Mint)
	}

	return nil
}

func CheckRootHasEnoughToken(client *rpc.Client, taskManage *models.WashTaskManage) (bool, error) {
	// 如果已经有足够的代币，直接返回
	if taskManage.RootHasEnoughToken {
		return true, nil
	}

	// 获取根节点地址
	var rootNode models.AddressNode
	if err := config.DB.Where("map_id = ? AND node_type = ?", taskManage.MapID, models.NodeTypeRoot).First(&rootNode).Error; err != nil {
		return false, fmt.Errorf("获取根节点失败: %v", err)
	}

	// 如果是 SOL 代币，检查余额
	if taskManage.SendToken == "sol" {
		rootPubkey, err := solanaGo.PublicKeyFromBase58(rootNode.NodeValue)
		if err != nil {
			return false, fmt.Errorf("无效的根节点地址 %s: %v", rootNode.NodeValue, err)
		}

		balance, err := client.GetBalance(
			context.Background(),
			rootPubkey,
			rpc.CommitmentFinalized,
		)
		if err != nil {
			return false, fmt.Errorf("获取根节点余额失败: %v", err)
		}

		// 更新根节点钱包代币统计
		if err := UpdateWalletTokenStatLocal(rootNode.NodeValue, "sol", 9, balance.Value, balance.Context.Slot, time.Now()); err != nil {
			log.Printf("更新根节点钱包代币统计失败 [地址:%s]: %v", rootNode.NodeValue, err)
		}

		// 计算所需的总金额（包括所有转账金额）
		requiredAmount := taskManage.TaskAmount * math.Pow10(int(taskManage.TokenDecimals))

		// 如果余额足够，更新状态并阻止购买权限
		if float64(balance.Value) >= requiredAmount {
			if err := config.DB.Model(taskManage).Update("root_has_enough_token", true).Error; err != nil {
				return false, fmt.Errorf("更新根节点余额状态失败: %v", err)
			}
			taskManage.RootHasEnoughToken = true

			// 阻止根节点地址的购买权限
			if err := ModifyRootAddressBuyPermission(rootNode.NodeValue, false); err != nil {
				log.Printf("阻止根节点购买权限失败: %v", err)
			}

			log.Printf("> 根节点 %s 余额充足，已更新状态", rootNode.NodeValue)
			return true, nil
		} else {
			log.Printf("> 根节点 %s 余额不足 (当前: %d, 需要: %f)", rootNode.NodeValue, balance.Value, requiredAmount)
			return false, nil
		}
	}

	return false, nil
}

func UpdateRootBalance(client *rpc.Client, mapID uint) error {
	// 获取根节点
	var rootNode models.AddressNode
	if err := config.DB.Where("map_id = ? AND node_type = ?", mapID, models.NodeTypeRoot).First(&rootNode).Error; err != nil {
		return fmt.Errorf("获取根节点失败: %v", err)
	}

	// 获取节点余额
	pubkey, err := solanaGo.PublicKeyFromBase58(rootNode.NodeValue)
	if err != nil {
		return fmt.Errorf("无效的地址 %s: %v", rootNode.NodeValue, err)
	}

	balance, err := client.GetBalance(
		context.Background(),
		pubkey,
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return fmt.Errorf("获取账户余额失败 [地址:%s]: %v", rootNode.NodeValue, err)
	}

	// 更新钱包代币统计
	if err := UpdateWalletTokenStatLocal(rootNode.NodeValue, "sol", 9, balance.Value, balance.Context.Slot, time.Now()); err != nil {
		return fmt.Errorf("更新钱包代币统计失败 [地址:%s]: %v", rootNode.NodeValue, err)
	}

	log.Printf("> 更新根节点余额成功 [地址:%s, 余额:%d]", rootNode.NodeValue, balance.Value)

	// 恢复根节点地址的购买权限
	// if err := ModifyRootAddressBuyPermission(rootNode.NodeValue, true); err != nil {
	// 	log.Printf("恢复根节点购买权限失败: %v", err)
	// }

	return nil
}

func main() {
	config.InitDB()
	for {
		var taskManages []models.WashTaskManage
		if err := config.DB.Where("enabled = ?", true).Find(&taskManages).Error; err != nil {
			log.Printf("获取任务管理列表失败: %v", err)
			time.Sleep(2 * time.Minute)
			continue
		}

		for _, taskManage := range taskManages {
			// 初始化 client (用于过期检查和后续处理)
			client := rpc.New(taskManage.Endpoint)
			km := solana.NewKeyManager()
			minGas := uint64(taskManage.TaskGas * math.Pow10(int(taskManage.TokenDecimals)))

			// 检查并关闭过期的任务
			if err := CloseWashTaskAfterExpired(&taskManage, client); err != nil {
				log.Printf("检查过期任务失败: %v", err)
				continue
			}

			// 如果任务已被标记为过期，跳过处理
			if !taskManage.Enabled {
				continue
			}

			// 检查根节点余额状态
			hasEnoughToken, err := CheckRootHasEnoughToken(client, &taskManage)
			if err != nil {
				log.Printf("检查根节点余额失败: %v", err)
				continue
			}

			// 初始化 client
			if err := config.DB.Model(&models.WashTaskManage{}).
				Where("id = ?", taskManage.ID).
				Update("status", "processing").Error; err != nil {
				log.Printf("更新任务管理状态失败: %v", err)
				continue
			}

			// 如果余额不足，跳过后续操作
			if !hasEnoughToken {
				log.Printf("> 任务 %d 根节点余额不足，跳过执行", taskManage.ID)
				continue
			}

			// 验证待确认的交易
			if err := WaitForConfirmationAndVerify(client, taskManage.ID); err != nil {
				log.Printf("验证交易失败: %v", err)
			}

			// 获取对应的 WashMap
			var washMap models.WashMap
			if err := config.DB.Where("id = ?", taskManage.MapID).First(&washMap).Error; err != nil {
				log.Printf("获取 WashMap 失败 [ID:%d]: %v", taskManage.MapID, err)
				continue
			}

			// 解析 MapParams
			var mapParams MapParamsType
			mapParamsBytes, err := json.Marshal(washMap.MapParams)
			if err != nil {
				log.Printf("序列化 MapParams 失败: %v", err)
				continue
			}

			if err := json.Unmarshal(mapParamsBytes, &mapParams); err != nil {
				log.Printf("解析 MapParams 失败: %v", err)
				continue
			}

			// 获取所有相关的 AddressNode
			var addressNodesFromMap []models.AddressNode
			if err := config.DB.Where("map_id = ?", taskManage.MapID).Find(&addressNodesFromMap).Error; err != nil {
				log.Printf("获取 AddressNode 失败: %v", err)
				continue
			}

			// 获取 root 节点
			var rootNode models.AddressNode
			for _, node := range addressNodesFromMap {
				if node.NodeType == models.NodeTypeRoot {
					rootNode = node
					break
				}
			}

			if rootNode.ID == 0 {
				log.Printf("未找到 root 节点 [MapID:%d]", taskManage.MapID)
				continue
			}

			// 执行第一深度任务
			firstDepthTasks := CreateFirstDepthWashTasks(addressNodesFromMap, rootNode, taskManage.ID)
			if len(firstDepthTasks) > 0 {
				log.Printf("> 执行第一深度任务，共 %d 个任务", len(firstDepthTasks))
				ExecuteWashTasks(firstDepthTasks, client, km, minGas)
			}

			// 执行其他深度任务
			otherDepthTasks := CreateOtherDepthWashTasks(addressNodesFromMap, mapParams.Count, mapParams.Depth, taskManage.ID, client)
			if len(otherDepthTasks) > 0 {
				log.Printf("> 执行其他深度任务，共 %d 个任务", len(otherDepthTasks))
				ExecuteWashTasks(otherDepthTasks, client, km, minGas)
			}

			// 每个 WashTaskManage 结束后休息 2 分钟
			log.Printf("> WashTaskManage %d 处理完成，休息 1 秒", taskManage.ID)
			time.Sleep(15 * time.Second)

			// 检查所有任务是否完成
			if verify_all_task_done(taskManage.ID) {
				// 更新整个地图的所有节点余额
				if err := UpdateMapBalance(client, taskManage.MapID); err != nil {
					log.Printf("更新地图余额失败: %v", err)
				}

				// 恢复根节点购买权限
				// var rootNode models.AddressNode
				// if err := config.DB.Where("map_id = ? AND node_type = ?", taskManage.MapID, models.NodeTypeRoot).First(&rootNode).Error; err == nil {
				// 	if err := ModifyRootAddressBuyPermission(rootNode.NodeValue, true); err != nil {
				// 		log.Printf("恢复根节点购买权限失败: %v", err)
				// 	}
				// }

				log.Printf("> 所有任务完成，ID：%d", taskManage.ID)
			}
		}

		// log.Println("> 进行下一轮任务，sleep...")
		time.Sleep(10 * time.Second)
	}
}

func WaitForConfirmation(client *rpc.Client, sig solanaGo.Signature, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)

	// fmt.Printf("> 正在确认: %s\n", sig.String())
	for time.Now().Before(deadline) {
		status, err := client.GetSignatureStatuses(context.Background(), true, sig)
		if err != nil {
			return false, err
		}

		if status.Value[0] != nil && status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
			fmt.Printf("> 确认完成: %s\n", sig.String())
			return true, nil
		}

		time.Sleep(10 * time.Second)
	}

	return false, fmt.Errorf("超时未确认")
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
			"status":  "processed",
		}).Error; err != nil {
		log.Printf("更新任务管理状态失败: %v", err)
	}

	return true
}
