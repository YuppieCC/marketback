package handlers

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	solanaUtils "marketcontrol/pkg/solana"

	log "github.com/sirupsen/logrus"
)

const (
	WSOl_MINT         = "So11111111111111111111111111111111111111112"
	MAX_WORKERS       = 5  // 最大并发工作协程数
	BATCH_MAX_WORKERS = 11 // 批量更新最大并发数
)

// 更新任务结构
type UpdateTask struct {
	Address string
	Mint    string
	Delay   float64
}

// 更新结果结构
type UpdateResult struct {
	Success bool
	Message string
	Updates []map[string]interface{}
	Error   error
}

// AddressUpdateTask represents a task for updating a single address
type AddressUpdateTask struct {
	Address string
	Tokens  []string
}

// AddressUpdateResult represents the result of updating a single address
type AddressUpdateResult struct {
	Address string
	Success bool
	Error   error
}

// 全局工作池
var (
	taskQueue   = make(chan UpdateTask, 100) // 任务队列
	workerPool  sync.WaitGroup               // 工作协程同步
	initialized = false                      // 初始化标志
	initOnce    sync.Once                    // 确保只初始化一次
)

// 初始化工作池
func initializeWorkerPool() {
	initOnce.Do(func() {
		for i := 0; i < MAX_WORKERS; i++ {
			workerPool.Add(1)
			go worker()
		}
		initialized = true
	})
}

// 工作协程
func worker() {
	defer workerPool.Done()

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		log.Errorf("Solana RPC endpoint not configured")
		return
	}

	// Create client
	client := rpc.New(solanaRPC)

	for task := range taskQueue {
		// 处理延迟
		if task.Delay > 0 {
			time.Sleep(time.Duration(task.Delay * float64(time.Second)))
		}

		// 执行更新
		pubkey, err := solana.PublicKeyFromBase58(task.Address)
		if err != nil {
			log.Errorf("Invalid address format: %v", err)
			continue
		}

		// Get SOL balance
		solBalance, solUpdateTime, err := solanaUtils.GetSolBalance(client, pubkey)
		if err != nil {
			log.Errorf("Failed to get SOL balance: %v", err)
			continue
		}
		updateWalletTokenStat(dbconfig.DB, task.Address, "sol", 1e9, solBalance, solUpdateTime)

		// Get WSOL balance
		// wsolBalance, wsolUpdateTime, err := solanaUtils.GetTokenBalance(dbconfig.DB, client, pubkey, WSOl_MINT)
		// if err != nil {
		// 	log.Errorf("Failed to get WSOL balance: %v", err)
		// 	continue
		// }
		// updateWalletTokenStat(dbconfig.DB, task.Address, WSOl_MINT, 1e9, wsolBalance, wsolUpdateTime)

		// Get specified token balance
		tokenBalance, tokenUpdateTime, err := solanaUtils.GetTokenBalance(dbconfig.DB, client, pubkey, task.Mint)
		if err != nil {
			log.Errorf("Failed to get token balance: %v", err)
			continue
		}

		// Get token decimals from TokenConfig
		var token models.TokenConfig
		if err := dbconfig.DB.Where("mint = ?", task.Mint).First(&token).Error; err != nil {
			log.Errorf("Failed to get token config: %v", err)
			continue
		}
		decimalsWithPow := math.Pow(10, float64(token.Decimals))
		updateWalletTokenStat(dbconfig.DB, task.Address, task.Mint, decimalsWithPow, tokenBalance, tokenUpdateTime)

		log.Infof("Successfully updated wallet token stats for address: %s, mint: %s", task.Address, task.Mint)
	}
}

type UpdateWalletTokenStatRequest struct {
	Mint  string  `json:"mint" binding:"required"`
	Delay float64 `json:"delay"`
}

// BatchUpdateWalletTokenStatsByAddressListRequest represents the request body for batch updating wallet token stats
type BatchUpdateWalletTokenStatsByAddressListRequest struct {
	Tokens      []string `json:"tokens" binding:"required"`
	AddressList []string `json:"address_list" binding:"required"`
}

// UpdateAddressByFilterRequest represents the request body for updating addresses by filter
type UpdateAddressByFilterRequest struct {
	FilterType     string  `json:"filter_type"`
	FilterValue    string  `json:"filter_value"`
	Token          string  `json:"token"`
	UpdateInterval float64 `json:"update_interval"`
	PageSize       int     `json:"page_size"`
	Page           int     `json:"page"`
}

// ReviewAddressByFilterRequest represents the request body for reviewing addresses by filter
type ReviewAddressByFilterRequest struct {
	FilterType     string  `json:"filter_type"`
	FilterValue    string  `json:"filter_value"`
	UpdateInterval float64 `json:"update_interval"`
	PageSize       int     `json:"page_size"`
}

// UpdateWalletTokenStatsByRoleRequest represents the request body for updating wallet token stats by role
type UpdateWalletTokenStatsByRoleRequest struct {
	RoleID   uint   `json:"role_id" binding:"required"`
	Mint     string `json:"mint" binding:"required"`
	Decimals uint8  `json:"decimals" binding:"required"`
}

// UpdateWalletTokenStatsByRole updates token stats for all addresses in a role
func UpdateWalletTokenStatsByRole(c *gin.Context) {
	// Get role_id from URL parameter
	roleIDStr := c.Param("role_id")
	if roleIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_id is required in URL"})
		return
	}

	// Parse role_id from URL
	roleIDUint64, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}
	roleID := uint(roleIDUint64)

	// Parse request body
	var request UpdateWalletTokenStatsByRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Use role_id from URL parameter (override body if different)
	request.RoleID = roleID

	// Validate mint address format
	if _, err := solana.PublicKeyFromBase58(request.Mint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mint address format"})
		return
	}

	// Get all addresses for this role_id
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", request.RoleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get addresses for role_id %d: %v", request.RoleID, err)})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   fmt.Sprintf("No addresses found for role_id %d", request.RoleID),
			"role_id": request.RoleID,
		})
		return
	}

	// Extract addresses
	addresses := make([]string, 0, len(roleAddresses))
	for _, roleAddr := range roleAddresses {
		// Validate address format
		if _, err := solana.PublicKeyFromBase58(roleAddr.Address); err != nil {
			log.Warnf("Invalid address format in role_address table: %s, skipping", roleAddr.Address)
			continue
		}
		addresses = append(addresses, roleAddr.Address)
	}

	if len(addresses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No valid addresses found for this role_id",
			"role_id": request.RoleID,
		})
		return
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create RPC client
	client := rpc.New(solanaRPC)

	// Get multiple accounts info using GetMultiAccountsInfo
	balances, err := solanaUtils.GetMultiAccountsInfo(client, addresses, request.Mint, request.Decimals)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get multi accounts info: %v", err)})
		return
	}

	// Return response
	response := gin.H{
		"mint":     request.Mint,
		"balances": balances,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateWalletTokenStatsByAddress updates token stats for a specific address
func UpdateWalletTokenStatsByAddress(c *gin.Context) {
	// 确保工作池已初始化
	if !initialized {
		initializeWorkerPool()
	}

	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address is required"})
		return
	}

	var request UpdateWalletTokenStatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate address format
	if _, err := solana.PublicKeyFromBase58(address); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address format"})
		return
	}

	// 创建更新任务
	task := UpdateTask{
		Address: address,
		Mint:    request.Mint,
		Delay:   request.Delay,
	}

	// 尝试将任务加入队列
	select {
	case taskQueue <- task:
		c.JSON(http.StatusAccepted, gin.H{
			"message": "Update task queued successfully",
			"address": address,
			"mint":    request.Mint,
			"delay":   request.Delay,
		})
	default:
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Update queue is full, please try again later",
		})
	}
}

// BatchUpdateWalletTokenStatsByAddressList batch updates token stats for multiple addresses and tokens
func BatchUpdateWalletTokenStatsByAddressList(c *gin.Context) {
	var request BatchUpdateWalletTokenStatsByAddressListRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate inputs
	if len(request.Tokens) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one token is required"})
		return
	}

	if len(request.AddressList) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one address is required"})
		return
	}

	// Validate all addresses format
	for _, address := range request.AddressList {
		if _, err := solana.PublicKeyFromBase58(address); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid address format",
				"address": address,
			})
			return
		}
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// 创建任务通道和结果通道
	taskChan := make(chan AddressUpdateTask, len(request.AddressList))
	resultChan := make(chan AddressUpdateResult, len(request.AddressList))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < BATCH_MAX_WORKERS; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// 每个工作协程创建自己的 RPC 客户端
			client := rpc.New(solanaRPC)

			for task := range taskChan {
				// 为每个工作协程添加不同的延迟，避免同时请求
				time.Sleep(time.Duration(workerID*20) * time.Millisecond)

				result := processAddressUpdate(client, task.Address, task.Tokens)
				resultChan <- result
			}
		}(i)
	}

	// 发送任务到任务通道
	for _, address := range request.AddressList {
		taskChan <- AddressUpdateTask{
			Address: address,
			Tokens:  request.Tokens,
		}
	}
	close(taskChan)

	// 等待所有工作协程完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var successCount int
	var failedAddresses []string

	for result := range resultChan {
		if result.Success {
			successCount++
			log.Infof("Successfully updated wallet token stats for address: %s", result.Address)
		} else {
			failedAddresses = append(failedAddresses, result.Address)
			if result.Error != nil {
				log.Errorf("Failed to update wallet token stats for address %s: %v", result.Address, result.Error)
			}
		}
	}

	// 返回结果
	response := gin.H{
		"message":         "Batch update completed",
		"total_addresses": len(request.AddressList),
		"success_count":   successCount,
		"failed_count":    len(failedAddresses),
	}

	if len(failedAddresses) > 0 {
		response["failed_addresses"] = failedAddresses
	}

	c.JSON(http.StatusOK, response)
}

// processAddressUpdate processes token updates for a single address
func processAddressUpdate(client *rpc.Client, address string, tokens []string) AddressUpdateResult {
	pubkey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return AddressUpdateResult{
			Address: address,
			Success: false,
			Error:   err,
		}
	}

	// 迭代所有代币
	for _, token := range tokens {
		if token == WSOl_MINT {
			continue
		}

		if token == "sol" {
			// 处理 SOL 余额
			solBalance, solUpdateTime, err := solanaUtils.GetSolBalance(client, pubkey)
			if err != nil {
				return AddressUpdateResult{
					Address: address,
					Success: false,
					Error:   err,
				}
			}
			updateWalletTokenStat(dbconfig.DB, address, "sol", 1e9, solBalance, solUpdateTime)
		} else {
			// 处理其他代币余额
			tokenBalance, tokenUpdateTime, err := solanaUtils.GetTokenBalance(dbconfig.DB, client, pubkey, token)
			if err != nil {
				return AddressUpdateResult{
					Address: address,
					Success: false,
					Error:   err,
				}
			}

			// Get token decimals from TokenConfig
			var tokenConfig models.TokenConfig
			decimalsWithPow := 1e6 // default decimals for unknown tokens
			if err := dbconfig.DB.Where("mint = ?", token).First(&tokenConfig).Error; err != nil {
				log.Warnf("Token config not found for mint %s, using default decimals", token)
			} else {
				decimalsWithPow = math.Pow(10, float64(tokenConfig.Decimals))
			}

			updateWalletTokenStat(dbconfig.DB, address, token, decimalsWithPow, tokenBalance, tokenUpdateTime)
		}
	}

	return AddressUpdateResult{
		Address: address,
		Success: true,
		Error:   nil,
	}
}

// UpdateAddressByFilter updates token stats for addresses filtered by criteria
func UpdateAddressByFilter(c *gin.Context) {
	var request UpdateAddressByFilterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if request.FilterType == "" {
		request.FilterType = "node_type"
	}
	if request.FilterValue == "" {
		request.FilterValue = "intermediate"
	}
	if request.Token == "" {
		request.Token = "sol"
	}
	if request.UpdateInterval == 0 {
		request.UpdateInterval = 0.1
	}
	if request.PageSize == 0 {
		request.PageSize = 100
	}
	if request.Page <= 0 {
		request.Page = 1
	}

	// 根据过滤条件获取地址列表
	var addresses []string
	var err error

	switch request.FilterType {
	case "node_type":
		addresses, err = getAddressesByNodeType(request.FilterValue)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported filter_type. Currently only 'node_type' is supported"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addresses: " + err.Error()})
		return
	}

	if len(addresses) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":      "No addresses found matching the filter criteria",
			"filter_type":  request.FilterType,
			"filter_value": request.FilterValue,
		})
		return
	}

	// 计算分页
	totalAddresses := len(addresses)
	totalPages := totalAddresses / request.PageSize
	if totalAddresses%request.PageSize != 0 {
		totalPages++
	}

	// 验证页码
	if request.Page > totalPages {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Page number exceeds total pages",
			"total_pages":    totalPages,
			"requested_page": request.Page,
		})
		return
	}

	// 计算分页索引
	startIndex := (request.Page - 1) * request.PageSize
	endIndex := startIndex + request.PageSize
	if endIndex > totalAddresses {
		endIndex = totalAddresses
	}

	// 分页切片地址
	pagedAddresses := addresses[startIndex:endIndex]

	// 调用批量更新逻辑
	result := performBatchUpdateWithInterval(pagedAddresses, []string{request.Token}, request.UpdateInterval)

	c.JSON(http.StatusOK, gin.H{
		"message":             "Filter-based update completed",
		"filter_type":         request.FilterType,
		"filter_value":        request.FilterValue,
		"token":               request.Token,
		"update_interval":     request.UpdateInterval,
		"total_addresses":     totalAddresses,
		"page":                request.Page,
		"page_size":           request.PageSize,
		"total_pages":         totalPages,
		"processed_addresses": len(pagedAddresses),
		"success_count":       result.SuccessCount,
		"failed_count":        result.FailedCount,
		"failed_addresses":    result.FailedAddresses,
	})
}

// getAddressesByNodeType retrieves addresses from AddressNode based on NodeType
func getAddressesByNodeType(nodeType string) ([]string, error) {
	var nodes []models.AddressNode
	if err := dbconfig.DB.Where("node_type = ?", nodeType).Find(&nodes).Error; err != nil {
		return nil, err
	}

	addresses := make([]string, len(nodes))
	for i, node := range nodes {
		addresses[i] = node.NodeValue
	}

	return addresses, nil
}

// ReviewAddressByFilter reviews addresses filtered by criteria and calculates update time estimates
func ReviewAddressByFilter(c *gin.Context) {
	var request ReviewAddressByFilterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if request.FilterType == "" {
		request.FilterType = "node_type"
	}
	if request.FilterValue == "" {
		request.FilterValue = "intermediate"
	}
	if request.UpdateInterval == 0 {
		request.UpdateInterval = 0.1
	}
	if request.PageSize == 0 {
		request.PageSize = 100
	}

	// 根据过滤条件获取地址列表
	var addresses []string
	var err error

	switch request.FilterType {
	case "node_type":
		addresses, err = getAddressesByNodeType(request.FilterValue)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported filter_type. Currently only 'node_type' is supported"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addresses: " + err.Error()})
		return
	}

	addressCount := len(addresses)

	// 计算页数
	pages := addressCount / request.PageSize
	if addressCount%request.PageSize != 0 {
		pages++
	}

	// 计算更新时间（以秒为单位）
	// 每个地址需要 update_interval 秒，总时间 = 地址数量 * 更新间隔
	updateTime := float64(addressCount) * request.UpdateInterval

	c.JSON(http.StatusOK, gin.H{
		"address_count": addressCount,
		"page_size":     request.PageSize,
		"pages":         pages,
		"update_time":   updateTime,
	})
}

// BatchUpdateResult represents the result of batch update operation
type BatchUpdateResult struct {
	SuccessCount    int      `json:"success_count"`
	FailedCount     int      `json:"failed_count"`
	FailedAddresses []string `json:"failed_addresses"`
}

// performBatchUpdateWithInterval executes batch update with specified interval between tasks
func performBatchUpdateWithInterval(addresses []string, tokens []string, updateInterval float64) BatchUpdateResult {
	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		log.Errorf("Solana RPC endpoint not configured")
		return BatchUpdateResult{
			SuccessCount:    0,
			FailedCount:     len(addresses),
			FailedAddresses: addresses,
		}
	}

	// 创建任务通道和结果通道
	taskChan := make(chan AddressUpdateTask, len(addresses))
	resultChan := make(chan AddressUpdateResult, len(addresses))

	// 启动工作协程
	var wg sync.WaitGroup
	for i := 0; i < BATCH_MAX_WORKERS; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// 每个工作协程创建自己的 RPC 客户端
			client := rpc.New(solanaRPC)

			for task := range taskChan {
				// 应用更新间隔
				if updateInterval > 0 {
					time.Sleep(time.Duration(updateInterval * float64(time.Second)))
				}

				// 为每个工作协程添加不同的延迟，避免同时请求
				time.Sleep(time.Duration(workerID*20) * time.Millisecond)

				result := processAddressUpdate(client, task.Address, task.Tokens)
				resultChan <- result
			}
		}(i)
	}

	// 发送任务到任务通道
	for _, address := range addresses {
		taskChan <- AddressUpdateTask{
			Address: address,
			Tokens:  tokens,
		}
	}
	close(taskChan)

	// 等待所有工作协程完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var successCount int
	var failedAddresses []string

	for result := range resultChan {
		if result.Success {
			successCount++
			log.Infof("Successfully updated wallet token stats for address: %s", result.Address)
		} else {
			failedAddresses = append(failedAddresses, result.Address)
			if result.Error != nil {
				log.Errorf("Failed to update wallet token stats for address %s: %v", result.Address, result.Error)
			}
		}
	}

	return BatchUpdateResult{
		SuccessCount:    successCount,
		FailedCount:     len(failedAddresses),
		FailedAddresses: failedAddresses,
	}
}

// UpdateFromSourceRequest represents the request body for updating wallet token stats from external source
type UpdateFromSourceRequest struct {
	OwnerAddress    string  `json:"owner_address" binding:"required"`
	Mint            string  `json:"mint" binding:"required"`
	Balance         uint64  `json:"balance"`
	BalanceReadable float64 `json:"balance_readable"`
}

// UpdateWalletTokenStatFromSource updates wallet token stat with data from external source
func UpdateWalletTokenStatFromSource(c *gin.Context) {
	var request UpdateFromSourceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate owner address format
	if _, err := solana.PublicKeyFromBase58(request.OwnerAddress); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid owner_address format"})
		return
	}

	// Update or create wallet token stat
	var stat models.WalletTokenStat
	if err := dbconfig.DB.Where("owner_address = ? AND mint = ?", request.OwnerAddress, request.Mint).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new record
			stat = models.WalletTokenStat{
				OwnerAddress:    request.OwnerAddress,
				Mint:            request.Mint,
				Balance:         request.Balance,
				BalanceReadable: request.BalanceReadable,
				Slot:            0,
				BlockTime:       time.Now(),
			}
			if err := dbconfig.DB.Create(&stat).Error; err != nil {
				log.Errorf("Failed to create wallet token stat for address: %s, mint: %s, error: %v", request.OwnerAddress, request.Mint, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet token stat"})
				return
			}
			log.Infof("Created new wallet token stat for address: %s, mint: %s", request.OwnerAddress, request.Mint)
		} else {
			log.Errorf("Failed to query wallet token stat for address: %s, mint: %s, error: %v", request.OwnerAddress, request.Mint, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query wallet token stat"})
			return
		}
	} else {
		// Update existing record
		stat.Balance = request.Balance
		stat.BalanceReadable = request.BalanceReadable
		stat.BlockTime = time.Now()
		if err := dbconfig.DB.Save(&stat).Error; err != nil {
			log.Errorf("Failed to update wallet token stat for address: %s, mint: %s, error: %v", request.OwnerAddress, request.Mint, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update wallet token stat"})
			return
		}
		log.Infof("Updated wallet token stat for address: %s, mint: %s", request.OwnerAddress, request.Mint)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Wallet token stat updated successfully",
		"data":    stat,
	})
}

func updateWalletTokenStat(db *gorm.DB, address, mint string, decimalsWithPow float64, balance uint64, updateTime time.Time) {
	var stat models.WalletTokenStat
	if err := db.Where("owner_address = ? AND mint = ?", address, mint).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			stat = models.WalletTokenStat{
				OwnerAddress:    address,
				Mint:            mint,
				Balance:         balance,
				BalanceReadable: float64(balance) / decimalsWithPow,
				Slot:            0,
				BlockTime:       updateTime,
			}
			if err := db.Create(&stat).Error; err != nil {
				log.Errorf("> 创建 地址: %s, 代币: %s, WalletTokenStat 失败: %v", address, mint, err)
			}
		} else {
			log.Errorf("> 查询 地址: %s, 代币: %s, WalletTokenStat 失败: %v", address, mint, err)
		}
	} else {
		stat.Balance = balance
		stat.BalanceReadable = float64(balance) / decimalsWithPow
		stat.Slot = 0
		stat.BlockTime = updateTime
		if err := db.Save(&stat).Error; err != nil {
			log.Errorf("> 更新 地址: %s, 代币: %s, WalletTokenStat 失败: %v", address, mint, err)
		}
	}
}

// batchUpdateWalletTokenStats performs batch UPSERT operation for wallet token stats
// Returns success count, failed count, and list of failed addresses
func batchUpdateWalletTokenStats(db *gorm.DB, stats []models.WalletTokenStat) (successCount int, failedCount int, failedAddresses []string) {
	if len(stats) == 0 {
		return 0, 0, nil
	}

	// Use PostgreSQL's ON CONFLICT for UPSERT operation
	// This requires a unique constraint on (owner_address, mint)
	// If the constraint doesn't exist, we'll fall back to individual updates
	err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "owner_address"}, {Name: "mint"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"balance",
			"balance_readable",
			"slot",
			"block_time",
			"updated_at",
		}),
	}).CreateInBatches(stats, 100).Error

	if err != nil {
		// If UPSERT fails (e.g., no unique constraint), fall back to individual updates
		log.Warnf("Batch UPSERT failed, falling back to individual updates: %v", err)
		return batchUpdateWalletTokenStatsFallback(db, stats)
	}

	// Track unique addresses for failed count
	addressSet := make(map[string]bool)
	for _, stat := range stats {
		addressSet[stat.OwnerAddress] = true
	}

	return len(stats), 0, nil
}

// batchUpdateWalletTokenStatsFallback performs batch update using a different strategy
// when UPSERT is not available (e.g., no unique constraint)
func batchUpdateWalletTokenStatsFallback(db *gorm.DB, stats []models.WalletTokenStat) (successCount int, failedCount int, failedAddresses []string) {
	if len(stats) == 0 {
		return 0, 0, nil
	}

	// Query existing records in batches to avoid SQL statement size limits
	const queryChunkSize = 50
	existingMap := make(map[string]*models.WalletTokenStat)

	// Query in chunks using a simpler approach
	for i := 0; i < len(stats); i += queryChunkSize {
		end := i + queryChunkSize
		if end > len(stats) {
			end = len(stats)
		}
		chunk := stats[i:end]

		// Build query for this chunk using raw SQL with tuple matching
		// PostgreSQL supports: WHERE (owner_address, mint) IN ((addr1, mint1), (addr2, mint2), ...)
		var placeholders []string
		var args []interface{}
		for _, stat := range chunk {
			placeholders = append(placeholders, "(?, ?)")
			args = append(args, stat.OwnerAddress, stat.Mint)
		}

		// Join placeholders with comma
		placeholderStr := ""
		for i, p := range placeholders {
			if i > 0 {
				placeholderStr += ", "
			}
			placeholderStr += p
		}
		query := fmt.Sprintf("(owner_address, mint) IN (%s)", placeholderStr)
		var chunkResults []models.WalletTokenStat
		if err := db.Where(query, args...).Find(&chunkResults).Error; err != nil {
			log.Errorf("Failed to query existing stats chunk: %v", err)
			// If tuple IN doesn't work, fall back to individual queries for this chunk
			for _, stat := range chunk {
				var existing models.WalletTokenStat
				if err := db.Where("owner_address = ? AND mint = ?", stat.OwnerAddress, stat.Mint).First(&existing).Error; err == nil {
					key := existing.OwnerAddress + "|" + existing.Mint
					existingMap[key] = &existing
				}
			}
			continue
		}

		// Add to existing map
		for k := range chunkResults {
			key := chunkResults[k].OwnerAddress + "|" + chunkResults[k].Mint
			existingMap[key] = &chunkResults[k]
		}
	}

	// Separate into creates and updates
	var toCreate []models.WalletTokenStat
	var toUpdate []models.WalletTokenStat

	for _, stat := range stats {
		key := stat.OwnerAddress + "|" + stat.Mint
		if existing, exists := existingMap[key]; exists {
			// Update existing record
			existing.Balance = stat.Balance
			existing.BalanceReadable = stat.BalanceReadable
			existing.Slot = stat.Slot
			existing.BlockTime = stat.BlockTime
			toUpdate = append(toUpdate, *existing)
		} else {
			// Create new record
			toCreate = append(toCreate, stat)
		}
	}

	// Batch create
	if len(toCreate) > 0 {
		if err := db.CreateInBatches(toCreate, 100).Error; err != nil {
			log.Errorf("Failed to batch create stats: %v", err)
			// Track failed addresses
			for _, stat := range toCreate {
				failedAddresses = append(failedAddresses, stat.OwnerAddress)
			}
			failedCount += len(toCreate)
		} else {
			successCount += len(toCreate)
		}
	}

	// Batch update in transactions
	if len(toUpdate) > 0 {
		const updateChunkSize = 100
		for i := 0; i < len(toUpdate); i += updateChunkSize {
			end := i + updateChunkSize
			if end > len(toUpdate) {
				end = len(toUpdate)
			}
			chunk := toUpdate[i:end]

			// Use transaction for batch update
			tx := db.Begin()
			if tx.Error != nil {
				log.Errorf("Failed to start transaction: %v", tx.Error)
				// Fall back to individual updates
				for _, stat := range chunk {
					if err := db.Save(&stat).Error; err != nil {
						log.Errorf("Failed to update stat: %v", err)
						failedAddresses = append(failedAddresses, stat.OwnerAddress)
						failedCount++
					} else {
						successCount++
					}
				}
				continue
			}

			chunkSuccess := true
			for j := range chunk {
				if err := tx.Save(&chunk[j]).Error; err != nil {
					log.Errorf("Failed to update stat for address %s, mint %s: %v", chunk[j].OwnerAddress, chunk[j].Mint, err)
					failedAddresses = append(failedAddresses, chunk[j].OwnerAddress)
					failedCount++
					chunkSuccess = false
				} else {
					successCount++
				}
			}

			if chunkSuccess {
				if err := tx.Commit().Error; err != nil {
					log.Errorf("Failed to commit transaction: %v", err)
					tx.Rollback()
				}
			} else {
				tx.Rollback()
			}
		}
	}

	return successCount, failedCount, failedAddresses
}

// batchUpdateWalletTokenStatsIndividual is the final fallback - updates one by one
func batchUpdateWalletTokenStatsIndividual(db *gorm.DB, stats []models.WalletTokenStat) (successCount int, failedCount int, failedAddresses []string) {
	for _, stat := range stats {
		decimalsWithPow := 1e9 // default for SOL
		if stat.Mint != "sol" {
			// Try to get decimals from token config
			var tokenConfig models.TokenConfig
			if err := db.Where("mint = ?", stat.Mint).First(&tokenConfig).Error; err == nil {
				decimalsWithPow = math.Pow(10, float64(tokenConfig.Decimals))
			}
		}

		updateWalletTokenStat(db, stat.OwnerAddress, stat.Mint, decimalsWithPow, stat.Balance, stat.BlockTime)
		successCount++
	}
	return successCount, 0, nil
}

// BatchUpdateWalletTokenStatsByAddressListV2Request represents the request body for batch updating wallet token stats v2
type BatchUpdateWalletTokenStatsByAddressListV2Request struct {
	RoleID   uint   `json:"role_id" binding:"required"`
	Mint     string `json:"mint" binding:"required"`
	Decimals uint8  `json:"decimals" binding:"required"`
}

// BatchUpdateWalletTokenStatsByAddressListV2 batch updates token stats for all addresses in a role using GetMultiAccountsInfo
func BatchUpdateWalletTokenStatsByAddressListV2(c *gin.Context) {
	var request BatchUpdateWalletTokenStatsByAddressListV2Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate mint address format
	if _, err := solana.PublicKeyFromBase58(request.Mint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mint address format"})
		return
	}

	// Get all addresses for this role_id
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", request.RoleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get addresses for role_id %d: %v", request.RoleID, err)})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   fmt.Sprintf("No addresses found for role_id %d", request.RoleID),
			"role_id": request.RoleID,
		})
		return
	}

	// Extract addresses
	addresses := make([]string, 0, len(roleAddresses))
	for _, roleAddr := range roleAddresses {
		// Validate address format
		if _, err := solana.PublicKeyFromBase58(roleAddr.Address); err != nil {
			log.Warnf("Invalid address format in role_address table: %s, skipping", roleAddr.Address)
			continue
		}
		addresses = append(addresses, roleAddr.Address)
	}

	if len(addresses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No valid addresses found for this role_id",
			"role_id": request.RoleID,
		})
		return
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create RPC client
	client := rpc.New(solanaRPC)

	// Get multiple accounts info (SOL and mint balances)
	balances, err := solanaUtils.GetMultiAccountsInfo(client, addresses, request.Mint, request.Decimals)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get multi accounts info: %v", err)})
		return
	}

	// Calculate decimalsWithPow for updateWalletTokenStat
	decimalsWithPow := math.Pow(10, float64(request.Decimals))
	updateTime := time.Now()

	// Prepare batch update data
	statsToUpdate := make([]models.WalletTokenStat, 0, len(balances)*2) // *2 for mint and sol balances

	for _, balance := range balances {
		// Prepare mint balance stat
		mintStat := models.WalletTokenStat{
			OwnerAddress:    balance.AccountAddress,
			Mint:            request.Mint,
			Balance:         balance.Balance,
			BalanceReadable: float64(balance.Balance) / decimalsWithPow,
			Slot:            0,
			BlockTime:       updateTime,
		}
		statsToUpdate = append(statsToUpdate, mintStat)

		// Prepare SOL balance stat
		solStat := models.WalletTokenStat{
			OwnerAddress:    balance.AccountAddress,
			Mint:            "sol",
			Balance:         balance.Lamports,
			BalanceReadable: float64(balance.Lamports) / 1e9,
			Slot:            0,
			BlockTime:       updateTime,
		}
		statsToUpdate = append(statsToUpdate, solStat)
	}

	// Batch update using UPSERT (ON CONFLICT)
	successCount, failedCount, failedAddresses := batchUpdateWalletTokenStats(dbconfig.DB, statsToUpdate)

	if successCount > 0 {
		log.Infof("Batch updated %d wallet token stats successfully", successCount)
	}
	if failedCount > 0 {
		log.Warnf("Failed to update %d wallet token stats", failedCount)
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"role_id":           request.RoleID,
		"mint":              request.Mint,
		"decimals":          request.Decimals,
		"success_count":     successCount,
		"failed_count":      failedCount,
		"failed_addresses":  failedAddresses,
		"total_addresses":   len(addresses),
		"updated_addresses": len(balances),
	})
}
