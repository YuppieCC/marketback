package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"sync"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/solana"
	"os"

	"github.com/blocto/solana-go-sdk/types"
	solanaGo "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

// ListAddresses returns a list of all managed addresses
func ListAddresses(c *gin.Context) {
	var addresses []models.AddressManage
	if err := dbconfig.DB.Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, addresses)
}

// GetAddress returns a specific managed address by address string
func GetAddress(c *gin.Context) {
	addressStr := c.Param("address")
	if addressStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address parameter is required"})
		return
	}

	var address models.AddressManage
	if err := dbconfig.DB.Where("address = ?", addressStr).First(&address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, address)
}

// AddressRequest represents the request body for creating/updating an address
type AddressRequest struct {
	Address    string `json:"address" binding:"required"`
	PrivateKey string `json:"private_key" binding:"required"`
}

// GenerateAddressRequest represents the request body for generating addresses
type GenerateAddressRequest struct {
	Count int `json:"count" binding:"required,min=1,max=1000"`
}

// DeleteAddress deletes a managed address
func DeleteAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.AddressManage{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// GenerateAddresses generates multiple Solana addresses
func GenerateAddresses(c *gin.Context) {
	var request GenerateAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建一个新的 key manager
	km := solana.NewKeyManager()

	addresses := make([]models.AddressManage, 0, request.Count)
	for i := 0; i < request.Count; i++ {
		address, err := GenerateSingleAddress(km)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":                fmt.Sprintf("生成地址 %d 失败: %v", i+1, err),
				"successful_addresses": len(addresses),
			})
			return
		}
		addresses = append(addresses, *address)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   fmt.Sprintf("成功生成 %d 个 Solana 地址", len(addresses)),
		"addresses": addresses,
	})
}

// GenerateSingleAddress 生成单个 Solana 地址并保存到数据库
func GenerateSingleAddress(km *solana.KeyManager) (*models.AddressManage, error) {
	// 生成新的 Solana 密钥对
	account, err := km.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("生成 Solana 密钥对失败: %v", err)
	}

	// 获取 Solana 地址
	solanaAddress := account.PublicKey.ToBase58()

	// 从环境变量获取加密密码
	encryptPassword := os.Getenv("ENCRYPTPASSWORD")
	if encryptPassword == "" {
		return nil, fmt.Errorf("未设置 ENCRYPTPASSWORD 环境变量")
	}

	// 加密私钥
	encryptedKey, err := km.EncryptPrivateKey(account.PrivateKey, encryptPassword)
	if err != nil {
		return nil, fmt.Errorf("加密私钥失败: %v", err)
	}

	// 保存加密的密钥到文件
	fileName := fmt.Sprintf("%s.json", solanaAddress)
	if err := km.SaveEncryptedKeyToFile(encryptedKey, fileName); err != nil {
		return nil, fmt.Errorf("保存密钥到文件失败: %v", err)
	}

	// 创建新的地址记录
	address := &models.AddressManage{
		Address:    solanaAddress,
		PrivateKey: encryptedKey,
	}

	// 保存到数据库
	if err := dbconfig.DB.Create(address).Error; err != nil {
		return nil, fmt.Errorf("创建地址记录失败: %v", err)
	}

	return address, nil
}

// DecryptPrivateKeyRequest represents the request body for decrypting a private key
type DecryptPrivateKeyRequest struct {
	EncryptedKey string `json:"encrypted_key" binding:"required"`
	Password     string `json:"password" binding:"required"`
}

// DecryptPrivateKey decrypts a private key using the provided password
func DecryptPrivateKey(c *gin.Context) {
	var request DecryptPrivateKeyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create a new key manager
	km := solana.NewKeyManager()

	// Decrypt the private key
	decryptedKey, err := km.DecryptPrivateKey(request.EncryptedKey, request.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decrypt private key: " + err.Error()})
		return
	}

	// Convert decrypted key to account
	account, err := types.AccountFromBytes(decryptedKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account from private key: " + err.Error()})
		return
	}

	// Convert private key bytes to array of integers
	privateKeyArray := make([]int, len(account.PrivateKey))
	for i, b := range account.PrivateKey {
		privateKeyArray[i] = int(b)
	}

	c.JSON(http.StatusOK, gin.H{
		"address":            account.PublicKey.ToBase58(),
		"private_key":        privateKeyArray,
		"private_key_source": account.PrivateKey,
	})
}

// ListAddressesByRole returns managed addresses associated with a specific role
func ListAddressesByRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	// First, get all addresses associated with the role
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, []models.AddressManage{})
		return
	}

	// Extract addresses from role_addresses
	addresses := make([]string, len(roleAddresses))
	for i, ra := range roleAddresses {
		addresses[i] = ra.Address
	}

	// Query AddressManage records for these addresses
	var addressManages []models.AddressManage
	if err := dbconfig.DB.Where("address IN ?", addresses).Find(&addressManages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, addressManages)
}

// ExportPasswordRequest represents the request body for exporting addresses with a new password
type ExportPasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// ExportAddress represents an address entry in the export file
type ExportAddress struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
}

// ExportWithNewPassword exports all addresses with re-encrypted private keys using a new password
func ExportWithNewPassword(c *gin.Context) {
	var request ExportPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get all addresses from the database
	var addresses []models.AddressManage
	if err := dbconfig.DB.Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addresses: " + err.Error()})
		return
	}

	// Create a new key manager
	km := solana.NewKeyManager()

	// Process each address
	exportAddresses := make([]ExportAddress, 0)
	for _, addr := range addresses {
		// Decrypt with old password
		decryptedKey, err := km.DecryptPrivateKey(addr.PrivateKey, request.OldPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to decrypt address %s: %v", addr.Address, err)})
			return
		}

		// Convert decrypted key to account
		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create account for address %s: %v", addr.Address, err)})
			return
		}

		// Verify address matches
		if account.PublicKey.ToBase58() != addr.Address {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Address mismatch for %s", addr.Address)})
			return
		}

		// Re-encrypt with new password
		newEncryptedKey, err := km.EncryptPrivateKey(account.PrivateKey, request.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to re-encrypt address %s: %v", addr.Address, err)})
			return
		}

		// Add to export list
		exportAddresses = append(exportAddresses, ExportAddress{
			Address:    addr.Address,
			PrivateKey: newEncryptedKey,
		})
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename=addresses_export.json")
	c.Header("Content-Type", "application/json")

	// Send the JSON response
	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Successfully exported %d addresses", len(exportAddresses)),
		"addresses": exportAddresses,
	})
}

// ImportRequest represents the request body for importing addresses
type ImportRequest struct {
	Password  string          `json:"password" binding:"required"`
	Addresses []ExportAddress `json:"addresses" binding:"required"`
}

// ImportAndVerifyPassword handles the import and verification of addresses from a JSON file
func ImportAndVerifyPassword(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form: " + err.Error()})
		return
	}

	// Get password from form
	password := c.Request.FormValue("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
		return
	}

	// Get uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file: " + err.Error()})
		return
	}
	defer file.Close()

	// Read and parse the JSON file
	var importData struct {
		Addresses []ExportAddress `json:"addresses"`
	}
	if err := json.NewDecoder(file).Decode(&importData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON file: " + err.Error()})
		return
	}

	// Create key manager
	km := solana.NewKeyManager()

	// First verify existing addresses in database
	var existingAddresses []models.AddressManage
	if err := dbconfig.DB.Find(&existingAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch existing addresses: " + err.Error()})
		return
	}

	// Verify existing addresses
	for _, addr := range existingAddresses {
		decryptedKey, err := km.DecryptPrivateKey(addr.PrivateKey, password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to decrypt existing address %s: %v", addr.Address, err)})
			return
		}

		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create account for existing address %s: %v", addr.Address, err)})
			return
		}

		if account.PublicKey.ToBase58() != addr.Address {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Address mismatch for existing address %s", addr.Address)})
			return
		}
	}

	// (moved) ExportWithNewPasswordFromRole will be defined after ImportAndVerifyPassword

	// Verify imported addresses
	existingAddressMap := make(map[string]bool)
	for _, addr := range existingAddresses {
		existingAddressMap[addr.Address] = true
	}

	// Track new addresses to import
	var newAddresses []models.AddressManage

	// Verify and prepare imported addresses
	for _, importAddr := range importData.Addresses {
		// Skip if address already exists
		if existingAddressMap[importAddr.Address] {
			continue
		}

		// Verify the imported address
		decryptedKey, err := km.DecryptPrivateKey(importAddr.PrivateKey, password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to decrypt imported address %s: %v", importAddr.Address, err)})
			return
		}

		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create account for imported address %s: %v", importAddr.Address, err)})
			return
		}

		if account.PublicKey.ToBase58() != importAddr.Address {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Address mismatch for imported address %s", importAddr.Address)})
			return
		}

		// Add to new addresses list
		newAddresses = append(newAddresses, models.AddressManage{
			Address:    importAddr.Address,
			PrivateKey: importAddr.PrivateKey,
		})
	}

	// Import new addresses
	if len(newAddresses) > 0 {
		if err := dbconfig.DB.Create(&newAddresses).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import new addresses: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        fmt.Sprintf("Successfully imported %d new addresses", len(newAddresses)),
		"imported_count": len(newAddresses),
		"skipped_count":  len(importData.Addresses) - len(newAddresses),
	})
}

// ExportWithNewPasswordFromRole exports addresses for a specific role with re-encrypted private keys using a new password
func ExportWithNewPasswordFromRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("rold_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rold_id format"})
		return
	}

	var request ExportPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch role addresses by role_id
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role addresses: " + err.Error()})
		return
	}

	if len(roleAddresses) == 0 {
		// No addresses for this role; return an empty export
		c.Header("Content-Disposition", "attachment; filename=addresses_export.json")
		c.Header("Content-Type", "application/json")
		c.JSON(http.StatusOK, gin.H{
			"message":   "Successfully exported 0 addresses",
			"addresses": []ExportAddress{},
		})
		return
	}

	// Extract address strings
	addressStrs := make([]string, len(roleAddresses))
	for i, ra := range roleAddresses {
		addressStrs[i] = ra.Address
	}

	// Fetch managed addresses matching these addresses
	var addresses []models.AddressManage
	if err := dbconfig.DB.Where("address IN ?", addressStrs).Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addresses: " + err.Error()})
		return
	}

	// Create a new key manager
	km := solana.NewKeyManager()

	// Process each address
	exportAddresses := make([]ExportAddress, 0, len(addresses))
	for _, addr := range addresses {
		// Decrypt with old password
		decryptedKey, err := km.DecryptPrivateKey(addr.PrivateKey, request.OldPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to decrypt address %s: %v", addr.Address, err)})
			return
		}

		// Convert decrypted key to account
		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create account for address %s: %v", addr.Address, err)})
			return
		}

		// Verify address matches
		if account.PublicKey.ToBase58() != addr.Address {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Address mismatch for %s", addr.Address)})
			return
		}

		// Re-encrypt with new password
		newEncryptedKey, err := km.EncryptPrivateKey(account.PrivateKey, request.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to re-encrypt address %s: %v", addr.Address, err)})
			return
		}

		// Add to export list
		exportAddresses = append(exportAddresses, ExportAddress{
			Address:    addr.Address,
			PrivateKey: newEncryptedKey,
		})
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename=addresses_export.json")
	c.Header("Content-Type", "application/json")

	// Send the JSON response
	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Successfully exported %d addresses", len(exportAddresses)),
		"addresses": exportAddresses,
	})
}

// AddressRoleInfo represents the response structure for address with role information
type AddressRoleInfo struct {
	Address   string               `json:"address"`
	RoleCount int                  `json:"role_count"`
	RoleLists []*models.RoleConfig `json:"role_lists"`
}

// ReviewAddressesByRoleCount returns a list of addresses with their role counts and role information
func ReviewAddressesByRoleCount(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "150"))
	order := c.DefaultQuery("order", "desc")

	// 验证分页参数
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 150
	}
	if pageSize > 500 {
		pageSize = 500 // 设置最大限制
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	// 获取所有地址
	var addresses []models.AddressManage
	if err := dbconfig.DB.Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 使用 map 来统计和去重
	addressMap := make(map[string]*AddressRoleInfo)

	// 初始化所有地址的记录
	for _, addr := range addresses {
		addressMap[addr.Address] = &AddressRoleInfo{
			Address:   addr.Address,
			RoleCount: 0,
			RoleLists: []*models.RoleConfig{},
		}
	}

	// 获取所有角色地址关联及其角色信息
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Preload("Role").Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 统计每个地址的角色信息
	for _, roleAddr := range roleAddresses {
		if info, exists := addressMap[roleAddr.Address]; exists {
			info.RoleCount++
			// 检查角色是否已添加
			roleExists := false
			for _, role := range info.RoleLists {
				if role.ID == roleAddr.Role.ID {
					roleExists = true
					break
				}
			}
			if !roleExists {
				info.RoleLists = append(info.RoleLists, roleAddr.Role)
			}
		}
	}

	// 转换 map 为 slice 并排序
	result := make([]AddressRoleInfo, 0, len(addressMap))
	for _, info := range addressMap {
		result = append(result, *info)
	}

	// 根据 role_count 排序
	if order == "asc" {
		sort.Slice(result, func(i, j int) bool {
			return result[i].RoleCount < result[j].RoleCount
		})
	} else {
		sort.Slice(result, func(i, j int) bool {
			return result[i].RoleCount > result[j].RoleCount
		})
	}

	// 计算总记录数和总页数
	total := len(result)
	totalPages := (total + pageSize - 1) / pageSize

	// 确保页码不超过总页数
	if page > totalPages {
		page = totalPages
	}

	// 计算分页的起始和结束索引
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	// 获取当前页的数据
	var pageData []AddressRoleInfo
	if start < total {
		pageData = result[start:end]
	} else {
		pageData = []AddressRoleInfo{}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":        total,
		"total_pages":  totalPages,
		"current_page": page,
		"page_size":    pageSize,
		"order":        order,
		"data":         pageData,
	})
}

// TokenStatRequest represents the request body for reviewing addresses by token stats
type TokenStatRequest struct {
	Page      int      `json:"page" form:"page"`
	PageSize  int      `json:"page_size" form:"page_size"`
	OrderType string   `json:"order_type" form:"order_type"`
	OrderBy   string   `json:"order_by" form:"order_by"`
	Tokens    []string `json:"tokens" form:"tokens"`
}

// TokenStatInfo represents token statistics for an address
type TokenStatInfo struct {
	Mint            string  `json:"mint"`
	Decimals        int     `json:"decimals"`
	Balance         int64   `json:"balance"`
	BalanceReadable float64 `json:"balance_readable"`
	Slot            int64   `json:"slot"`
	BlockTime       int64   `json:"block_time"`
	CreatedAt       int64   `json:"created_at"`
	UpdatedAt       int64   `json:"updated_at"`
}

// AddressTokenStats represents an address with its token statistics
type AddressTokenStats struct {
	OwnerAddress string          `json:"owner_address"`
	Tokens       []TokenStatInfo `json:"tokens"`
}

// ReviewAddressesByTokenStat returns a list of addresses with their token statistics
func ReviewAddressesByTokenStat(c *gin.Context) {
	var request TokenStatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values
	if request.Page < 1 {
		request.Page = 1
	}
	if request.PageSize < 1 {
		request.PageSize = 150
	}
	if request.OrderType == "" {
		request.OrderType = "desc"
	}
	if request.OrderBy == "" {
		request.OrderBy = "sol"
	}
	if len(request.Tokens) == 0 {
		request.Tokens = []string{"sol", "So11111111111111111111111111111111111111112"}
	}

	// Remove duplicates from tokens array
	uniqueTokensMap := make(map[string]bool)
	uniqueTokens := make([]string, 0)
	for _, token := range request.Tokens {
		if !uniqueTokensMap[token] {
			uniqueTokensMap[token] = true
			uniqueTokens = append(uniqueTokens, token)
		}
	}
	request.Tokens = uniqueTokens

	// Validate order_type
	if request.OrderType != "asc" && request.OrderType != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_type must be either 'asc' or 'desc'"})
		return
	}

	// Validate order_by
	validOrderBy := false
	for _, token := range request.Tokens {
		if token == request.OrderBy {
			validOrderBy = true
			break
		}
	}
	if !validOrderBy {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_by must be one of the tokens"})
		return
	}

	// Get all addresses
	var addresses []models.AddressManage
	if err := dbconfig.DB.Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(addresses) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total":          0,
			"total_pages":    0,
			"current_page":   request.Page,
			"page_size":      request.PageSize,
			"order_type":     request.OrderType,
			"order_by":       request.OrderBy,
			"data":           []interface{}{},
			"aggregate_data": []interface{}{},
		})
		return
	}

	// Extract addresses for query
	addressList := make([]string, len(addresses))
	for i, addr := range addresses {
		addressList[i] = addr.Address
	}

	// Get token stats for all addresses and requested tokens
	var stats []models.WalletTokenStat
	if err := dbconfig.DB.Where("owner_address IN ? AND mint IN ?", addressList, request.Tokens).
		Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate aggregate data
	aggregateTokenMap := AggregateTokenStats(stats)
	var aggregateData []AggregateTokenStat
	for _, mint := range request.Tokens {
		if agg, ok := aggregateTokenMap[mint]; ok {
			aggregateData = append(aggregateData, *agg)
		} else {
			// Add default values for tokens without data
			aggregateData = append(aggregateData, AggregateTokenStat{
				Mint:            mint,
				Decimals:        0,
				Balance:         0,
				BalanceReadable: 0,
			})
		}
	}

	// Group stats by address
	resultMap := make(map[string][]models.WalletTokenStat)
	for _, stat := range stats {
		resultMap[stat.OwnerAddress] = append(resultMap[stat.OwnerAddress], stat)
	}

	// Process results
	var result []TokenGroup
	for _, addr := range addresses {
		tokenStats := resultMap[addr.Address]
		tokenList := make([]WalletTokenStatResp, 0, len(request.Tokens))

		// Process each requested token
		for _, mint := range request.Tokens {
			found := false
			for _, stat := range tokenStats {
				if stat.Mint == mint {
					tokenList = append(tokenList, WalletTokenStatResp{
						Mint:            stat.Mint,
						Decimals:        stat.Decimals,
						Balance:         stat.Balance,
						BalanceReadable: stat.BalanceReadable,
						Slot:            stat.Slot,
						BlockTime:       stat.BlockTime.UnixMilli(),
						CreatedAt:       stat.CreatedAt.UnixMilli(),
						UpdatedAt:       stat.UpdatedAt.UnixMilli(),
					})
					found = true
					break
				}
			}
			if !found {
				// Add default values for missing tokens
				tokenList = append(tokenList, WalletTokenStatResp{
					Mint:            mint,
					Decimals:        0,
					Balance:         0,
					BalanceReadable: 0,
					Slot:            0,
					BlockTime:       0,
					CreatedAt:       0,
					UpdatedAt:       0,
				})
			}
		}

		result = append(result, TokenGroup{
			OwnerAddress: addr.Address,
			Tokens:       tokenList,
		})
	}

	// Sort results based on order_by and order_type
	sort.Slice(result, func(i, j int) bool {
		var balanceI, balanceJ float64
		for _, token := range result[i].Tokens {
			if token.Mint == request.OrderBy {
				balanceI = token.BalanceReadable
				break
			}
		}
		for _, token := range result[j].Tokens {
			if token.Mint == request.OrderBy {
				balanceJ = token.BalanceReadable
				break
			}
		}

		if request.OrderType == "asc" {
			return balanceI < balanceJ
		}
		return balanceI > balanceJ
	})

	// Calculate pagination
	total := len(result)
	totalPages := (total + request.PageSize - 1) / request.PageSize

	// Ensure page number is valid
	if request.Page > totalPages {
		request.Page = totalPages
	}

	// Calculate slice bounds
	start := (request.Page - 1) * request.PageSize
	end := start + request.PageSize
	if end > total {
		end = total
	}

	// Get the current page's data
	var pageData []TokenGroup
	if start < total {
		pageData = result[start:end]
	} else {
		pageData = []TokenGroup{}
	}

	c.JSON(http.StatusOK, gin.H{
		"total":          total,
		"total_pages":    totalPages,
		"current_page":   request.Page,
		"page_size":      request.PageSize,
		"order_type":     request.OrderType,
		"order_by":       request.OrderBy,
		"data":           pageData,
		"aggregate_data": aggregateData,
	})
}

// AddressConfigRequest represents the request body for creating/updating an address config
type AddressConfigRequest struct {
	Address              string `json:"address" binding:"required"`
	Mint                 string `json:"mint" binding:"required"`
	IsBuyAllowed         *bool  `json:"is_buy_allowed"`
	IsSellAllowed        *bool  `json:"is_sell_allowed"`
	IsTradeAllowed       *bool  `json:"is_trade_allowed"`
	TradePriorityRate    *uint  `json:"trade_priority_rate"`
	IsClosed             *bool  `json:"is_closed"`
	IsTokenAccountClosed *bool  `json:"is_token_account_closed"`
}

// CreateAddressConfig creates a new address configuration
func CreateAddressConfig(c *gin.Context) {
	var request AddressConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default values if not provided
	isBuyAllowed := true
	isSellAllowed := true
	isTradeAllowed := true
	tradePriorityRate := uint(1)
	isClosed := false
	isTokenAccountClosed := false

	if request.IsBuyAllowed != nil {
		isBuyAllowed = *request.IsBuyAllowed
	}
	if request.IsSellAllowed != nil {
		isSellAllowed = *request.IsSellAllowed
	}
	if request.IsTradeAllowed != nil {
		isTradeAllowed = *request.IsTradeAllowed
	}
	if request.TradePriorityRate != nil {
		tradePriorityRate = *request.TradePriorityRate
	}
	if request.IsClosed != nil {
		isClosed = *request.IsClosed
	}
	if request.IsTokenAccountClosed != nil {
		isTokenAccountClosed = *request.IsTokenAccountClosed
	}

	addressConfig := models.AddressConfig{
		Address:              request.Address,
		Mint:                 request.Mint,
		IsBuyAllowed:         isBuyAllowed,
		IsSellAllowed:        isSellAllowed,
		IsTradeAllowed:       isTradeAllowed,
		TradePriorityRate:    tradePriorityRate,
		IsClosed:             isClosed,
		IsTokenAccountClosed: isTokenAccountClosed,
	}

	if err := dbconfig.DB.Create(&addressConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, addressConfig)
}

// GetAddressConfig returns a specific address configuration by ID
func GetAddressConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var addressConfig models.AddressConfig
	if err := dbconfig.DB.First(&addressConfig, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, addressConfig)
}

// CreateOrUpdateAddressConfig creates or updates an address configuration based on address and mint
func CreateOrUpdateAddressConfig(c *gin.Context) {
	var request AddressConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Try to find existing record by address and mint
	var addressConfig models.AddressConfig
	err := dbconfig.DB.Where("address = ? AND mint = ?", request.Address, request.Mint).First(&addressConfig).Error

	// Set default values if not provided
	isBuyAllowed := true
	isSellAllowed := true
	isTradeAllowed := true
	tradePriorityRate := uint(1)
	isClosed := false
	isTokenAccountClosed := false

	if request.IsBuyAllowed != nil {
		isBuyAllowed = *request.IsBuyAllowed
	}
	if request.IsSellAllowed != nil {
		isSellAllowed = *request.IsSellAllowed
	}
	if request.IsTradeAllowed != nil {
		isTradeAllowed = *request.IsTradeAllowed
	}
	if request.TradePriorityRate != nil {
		tradePriorityRate = *request.TradePriorityRate
	}
	if request.IsClosed != nil {
		isClosed = *request.IsClosed
	}
	if request.IsTokenAccountClosed != nil {
		isTokenAccountClosed = *request.IsTokenAccountClosed
	}

	if err != nil {
		// Record not found, create new one
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newAddressConfig := models.AddressConfig{
				Address:              request.Address,
				Mint:                 request.Mint,
				IsBuyAllowed:         isBuyAllowed,
				IsSellAllowed:        isSellAllowed,
				IsTradeAllowed:       isTradeAllowed,
				TradePriorityRate:    tradePriorityRate,
				IsClosed:             isClosed,
				IsTokenAccountClosed: isTokenAccountClosed,
			}

			if err := dbconfig.DB.Create(&newAddressConfig).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, newAddressConfig)
			return
		}
		// Other database error
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Record found, update it
	addressConfig.Address = request.Address
	addressConfig.Mint = request.Mint
	addressConfig.IsBuyAllowed = isBuyAllowed
	addressConfig.IsSellAllowed = isSellAllowed
	addressConfig.IsTradeAllowed = isTradeAllowed
	addressConfig.TradePriorityRate = tradePriorityRate
	addressConfig.IsClosed = isClosed
	addressConfig.IsTokenAccountClosed = isTokenAccountClosed

	if err := dbconfig.DB.Save(&addressConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, addressConfig)
}

// ListAddressConfigs returns a list of all address configurations with optional filtering
func ListAddressConfigs(c *gin.Context) {
	address := c.Query("address")
	mint := c.Query("mint")

	query := dbconfig.DB.Model(&models.AddressConfig{})

	if address != "" {
		query = query.Where("address = ?", address)
	}
	if mint != "" {
		query = query.Where("mint = ?", mint)
	}

	var addressConfigs []models.AddressConfig
	if err := query.Find(&addressConfigs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, addressConfigs)
}

// UpdateAddressConfig updates an existing address configuration
func UpdateAddressConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var addressConfig models.AddressConfig
	if err := dbconfig.DB.First(&addressConfig, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var request AddressConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	addressConfig.Address = request.Address
	addressConfig.Mint = request.Mint

	if request.IsBuyAllowed != nil {
		addressConfig.IsBuyAllowed = *request.IsBuyAllowed
	}
	if request.IsSellAllowed != nil {
		addressConfig.IsSellAllowed = *request.IsSellAllowed
	}
	if request.IsTradeAllowed != nil {
		addressConfig.IsTradeAllowed = *request.IsTradeAllowed
	}
	if request.TradePriorityRate != nil {
		addressConfig.TradePriorityRate = *request.TradePriorityRate
	}
	if request.IsClosed != nil {
		addressConfig.IsClosed = *request.IsClosed
	}
	if request.IsTokenAccountClosed != nil {
		addressConfig.IsTokenAccountClosed = *request.IsTokenAccountClosed
	}

	if err := dbconfig.DB.Save(&addressConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, addressConfig)
}

// DeleteAddressConfig deletes an address configuration
func DeleteAddressConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.AddressConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address configuration deleted successfully"})
}

// GmgnTrackFormatRequest represents the request body for GMGN track export
type GmgnTrackFormatRequest struct {
	Name      string `json:"name" binding:"required"`
	Emoji     string `json:"emoji" binding:"required"`
	GroupName string `json:"group_name" binding:"required"`
}

// GmgnTrackItem represents a single GMGN track entry
type GmgnTrackItem struct {
	Address string   `json:"address"`
	Name    string   `json:"name"`
	Emoji   string   `json:"emoji"`
	Groups  []string `json:"groups"`
}

// ExportWithGmgnTrackFormatFromRole exports addresses for a role in GMGN track format
func ExportWithGmgnTrackFormatFromRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var request GmgnTrackFormatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch role addresses by role_id
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch role addresses: " + err.Error()})
		return
	}

	// Build GMGN track format entries
	result := make([]GmgnTrackItem, 0, len(roleAddresses))
	for _, ra := range roleAddresses {
		item := GmgnTrackItem{
			Address: ra.Address,
			Name:    request.Name,
			Emoji:   request.Emoji,
			Groups:  []string{request.GroupName},
		}
		result = append(result, item)
	}

	c.JSON(http.StatusOK, result)
}

// GetAddressConfigByAddressAndMint returns address configuration by address and mint
func GetAddressConfigByAddressAndMint(c *gin.Context) {
	address := c.Param("address")
	mint := c.Param("mint")

	if address == "" || mint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address and mint parameters are required"})
		return
	}

	var addressConfig models.AddressConfig
	if err := dbconfig.DB.Where("address = ? AND mint = ?", address, mint).First(&addressConfig).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, addressConfig)
}

// ListAddressConfigByRole returns address configurations associated with a specific role
func ListAddressConfigByRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	// First, get all addresses associated with the role
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, []models.AddressConfig{})
		return
	}

	// Extract addresses from role_addresses
	addresses := make([]string, len(roleAddresses))
	for i, ra := range roleAddresses {
		addresses[i] = ra.Address
	}

	// Query AddressConfig records for these addresses
	var addressConfigs []models.AddressConfig
	if err := dbconfig.DB.Where("address IN ?", addresses).Find(&addressConfigs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, addressConfigs)
}

// FilterRequest represents the request body for filtering address configs
type FilterRequest struct {
	RoleID       int    `json:"role_id" binding:"required"`
	Mint         string `json:"mint" binding:"required"`
	PoolPlatform string `json:"pool_platform" binding:"omitempty"`
}

// GetAddressConfigByFilter filters address configurations by role_id and mint
func GetAddressConfigByFilter(c *gin.Context) {
	var request FilterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// First, get all addresses associated with the role
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", request.RoleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	// Extract addresses from role_addresses
	addresses := make([]string, len(roleAddresses))
	for i, ra := range roleAddresses {
		addresses[i] = ra.Address
	}

	// Query AddressConfig records for these addresses and specific mint
	var addressConfigs []models.AddressConfig
	if err := dbconfig.DB.Where("address IN ? AND mint = ?", addresses, request.Mint).Find(&addressConfigs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get transaction counts based on pool platform
	txCounts := make(map[string]uint)

	switch request.PoolPlatform {
	case "pumpfun_amm":
		var holders []models.PumpfunAmmpoolHolder
		if err := dbconfig.DB.Where("address IN ? AND (base_mint = ? OR quote_mint = ?)",
			addresses, request.Mint, request.Mint).
			Select("address, SUM(tx_count) as tx_count").
			Group("address").
			Find(&holders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, holder := range holders {
			txCounts[holder.Address] = holder.TxCount
		}
	case "pumpfun_internal":
		var holders []models.PumpfuninternalHolder
		if err := dbconfig.DB.Where("address IN ? AND mint = ?",
			addresses, request.Mint).
			Select("address, SUM(tx_count) as tx_count").
			Group("address").
			Find(&holders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, holder := range holders {
			txCounts[holder.Address] = holder.TxCount
		}
	default:
		// For any other platform or empty platform, tx_count will default to 0
	}

	// Build a map of existing configs for quick lookup
	configMap := make(map[string]models.AddressConfig)
	for _, config := range addressConfigs {
		configMap[config.Address] = config
	}

	// Build the response in the required format, including all role addresses
	result := make(map[string]map[string]interface{})
	for _, roleAddr := range roleAddresses {
		config, exists := configMap[roleAddr.Address]

		// Use default values if config doesn't exist
		isBuyAllowed := true
		isSellAllowed := true
		isTradeAllowed := true
		tradePriorityRate := uint(1)
		isClosed := false
		isTokenAccountClosed := false

		if exists {
			isBuyAllowed = config.IsBuyAllowed
			isSellAllowed = config.IsSellAllowed
			isTradeAllowed = config.IsTradeAllowed
			tradePriorityRate = config.TradePriorityRate
			isClosed = config.IsClosed
			isTokenAccountClosed = config.IsTokenAccountClosed
		}

		result[roleAddr.Address] = map[string]interface{}{
			"is_buy_allowed":          isBuyAllowed,
			"is_sell_allowed":         isSellAllowed,
			"is_trade_allowed":        isTradeAllowed,
			"trade_priority_rate":     tradePriorityRate,
			"is_closed":               isClosed,
			"is_token_account_closed": isTokenAccountClosed,
			"tx_count":                txCounts[roleAddr.Address],
		}
	}

	c.JSON(http.StatusOK, result)
}

// CheckAddressExistsRequest represents the request body for checking address existence
type CheckAddressExistsRequest struct {
	AddressLists []string `json:"address_lists" binding:"required"`
}

// CheckAddressExists checks if addresses exist in the AddressManage table
func CheckAddressExists(c *gin.Context) {
	var request CheckAddressExistsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.AddressLists) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address_lists cannot be empty"})
		return
	}

	// Get existing addresses from database
	var existingAddresses []models.AddressManage
	if err := dbconfig.DB.Where("address IN ?", request.AddressLists).Find(&existingAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create a map for fast lookup
	existingMap := make(map[string]bool)
	for _, addr := range existingAddresses {
		existingMap[addr.Address] = true
	}

	// Build result map
	result := make(map[string]bool)
	for _, address := range request.AddressLists {
		result[address] = existingMap[address]
	}

	c.JSON(http.StatusOK, result)
}

// DisposableAddressRequest represents the request body for creating/updating a disposable address
type DisposableAddressRequest struct {
	Address      string `json:"address" binding:"required"`
	PrivateKey   string `json:"private_key" binding:"required"`
	IsDeprecated *bool  `json:"is_deprecated"`
}

// BatchUpdateDisposableAddressRequest is the body for POST /disposable-address-manage/batch-update
type BatchUpdateDisposableAddressRequest struct {
	StartID      uint `json:"start_id" binding:"required"`
	EndID        uint `json:"end_id" binding:"required"`
	IsDeprecated bool `json:"is_deprecated"`
}

// ListDisposableAddresses returns a list of all disposable managed addresses
func ListDisposableAddresses(c *gin.Context) {
	var addresses []models.DisposableAddressManage
	if err := dbconfig.DB.Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, addresses)
}

// GetDisposableAddress returns a specific disposable managed address by ID
func GetDisposableAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var address models.DisposableAddressManage
	if err := dbconfig.DB.First(&address, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, address)
}

// CreateDisposableAddress creates a new disposable managed address
func CreateDisposableAddress(c *gin.Context) {
	var request DisposableAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default value for IsDeprecated if not provided
	isDeprecated := false
	if request.IsDeprecated != nil {
		isDeprecated = *request.IsDeprecated
	}

	address := models.DisposableAddressManage{
		Address:      request.Address,
		PrivateKey:   request.PrivateKey,
		IsDeprecated: isDeprecated,
	}

	if err := dbconfig.DB.Create(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, address)
}

// UpdateDisposableAddress updates an existing disposable managed address
func UpdateDisposableAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var address models.DisposableAddressManage
	if err := dbconfig.DB.First(&address, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var request DisposableAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	address.Address = request.Address
	address.PrivateKey = request.PrivateKey
	if request.IsDeprecated != nil {
		address.IsDeprecated = *request.IsDeprecated
	}

	if err := dbconfig.DB.Save(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, address)
}

// DeleteDisposableAddress deletes a disposable managed address
func DeleteDisposableAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.DisposableAddressManage{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// BatchUpdateDisposableAddress updates IsDeprecated for all DisposableAddressManage records with id in [start_id, end_id].
func BatchUpdateDisposableAddress(c *gin.Context) {
	var request BatchUpdateDisposableAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.StartID > request.EndID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_id must be less than or equal to end_id"})
		return
	}

	result := dbconfig.DB.Model(&models.DisposableAddressManage{}).
		Where("id >= ? AND id <= ?", request.StartID, request.EndID).
		Update("is_deprecated", request.IsDeprecated)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Batch update completed",
		"rows_affected":   result.RowsAffected,
		"start_id":        request.StartID,
		"end_id":          request.EndID,
		"is_deprecated":   request.IsDeprecated,
	})
}

// GenerateDisposableAddresses generates multiple Solana addresses for disposable addresses
func GenerateDisposableAddresses(c *gin.Context) {
	var request GenerateAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建一个新的 key manager
	km := solana.NewKeyManager()

	addresses := make([]models.DisposableAddressManage, 0, request.Count)
	for i := 0; i < request.Count; i++ {
		address, err := GenerateSingleDisposableAddress(km)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":                fmt.Sprintf("生成地址 %d 失败: %v", i+1, err),
				"successful_addresses": len(addresses),
			})
			return
		}
		addresses = append(addresses, *address)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   fmt.Sprintf("成功生成 %d 个 Solana 地址", len(addresses)),
		"addresses": addresses,
	})
}

// GenerateSingleDisposableAddress 生成单个 Solana 地址并保存到数据库
func GenerateSingleDisposableAddress(km *solana.KeyManager) (*models.DisposableAddressManage, error) {
	// 生成新的 Solana 密钥对
	account, err := km.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("生成 Solana 密钥对失败: %v", err)
	}

	// 获取 Solana 地址
	solanaAddress := account.PublicKey.ToBase58()

	// 从环境变量获取加密密码
	encryptPassword := os.Getenv("ENCRYPTPASSWORD")
	if encryptPassword == "" {
		return nil, fmt.Errorf("未设置 ENCRYPTPASSWORD 环境变量")
	}

	// 加密私钥
	encryptedKey, err := km.EncryptPrivateKey(account.PrivateKey, encryptPassword)
	if err != nil {
		return nil, fmt.Errorf("加密私钥失败: %v", err)
	}

	// 保存加密的密钥到文件
	fileName := fmt.Sprintf("%s.json", solanaAddress)
	if err := km.SaveEncryptedKeyToFile(encryptedKey, fileName); err != nil {
		return nil, fmt.Errorf("保存密钥到文件失败: %v", err)
	}

	// 创建新的一次性地址记录
	address := &models.DisposableAddressManage{
		Address:      solanaAddress,
		PrivateKey:   encryptedKey,
		IsDeprecated: false,
	}

	// 保存到数据库
	if err := dbconfig.DB.Create(address).Error; err != nil {
		return nil, fmt.Errorf("创建地址记录失败: %v", err)
	}

	return address, nil
}

// GetAndReplaceDisposableAddressRequest represents the request body for get-and-replace
type GetAndReplaceDisposableAddressRequest struct {
	DeprecatedAddress string `json:"deprecated-address" binding:"required"`
	RoleId            uint   `json:"role_id"`
}

// GetAndReplaceDisposableAddress gets a new address and marks the deprecated address as deprecated
func GetAndReplaceDisposableAddress(c *gin.Context) {
	var request GetAndReplaceDisposableAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 找到 IsDeprecated 不为 true 且不是 DeprecatedAddress 的地址，选择第一个作为新地址
	var newAddress models.DisposableAddressManage
	if err := dbconfig.DB.Where("is_deprecated = ? AND address != ?", false, request.DeprecatedAddress).First(&newAddress).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "没有可用的未废弃地址"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 找到 Address 为 DeprecatedAddress 的数据，将 IsDeprecated 设置为 true
	var deprecatedAddress models.DisposableAddressManage
	if err := dbconfig.DB.Where("address = ?", request.DeprecatedAddress).First(&deprecatedAddress).Error; err == nil {
		// 如果找到了废弃地址，则更新 IsDeprecated 为 true
		deprecatedAddress.IsDeprecated = true
		if err := dbconfig.DB.Save(&deprecatedAddress).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// 如果找不到废弃地址，继续执行并返回新地址

	// 将新地址复制到 AddressManage 表
	addressManage := models.AddressManage{
		Address:    newAddress.Address,
		PrivateKey: newAddress.PrivateKey,
	}

	// 检查 AddressManage 中是否已存在相同地址
	var existingAddress models.AddressManage
	err := dbconfig.DB.Where("address = ?", newAddress.Address).First(&existingAddress).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 地址不存在，创建新记录
			if err := dbconfig.DB.Create(&addressManage).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("复制地址到 AddressManage 失败: %v", err)})
				return
			}
		} else {
			// 其他数据库错误
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("检查地址是否存在时出错: %v", err)})
			return
		}
	}
	// 如果地址已存在，不做任何操作，直接继续

	// 若提供了 RoleId，则更新对应 RoleConfig 的 MainAddress 为 newAddress
	if request.RoleId > 0 {
		var roleConfig models.RoleConfig
		if err := dbconfig.DB.First(&roleConfig, request.RoleId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "RoleConfig not found for role_id"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		roleConfig.MainAddress = newAddress.Address
		if err := dbconfig.DB.Save(&roleConfig).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新 RoleConfig MainAddress 失败: %v", err)})
			return
		}
	}

	// 返回新地址
	c.JSON(http.StatusOK, newAddress)
}

// ExportWithNewPasswordInDisposableAddressManage exports all disposable addresses with re-encrypted private keys using a new password
func ExportWithNewPasswordInDisposableAddressManage(c *gin.Context) {
	var request ExportPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get all disposable addresses from the database
	var addresses []models.DisposableAddressManage
	if err := dbconfig.DB.Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch disposable addresses: " + err.Error()})
		return
	}

	// Create a new key manager
	km := solana.NewKeyManager()

	// Process each address
	exportAddresses := make([]ExportAddress, 0)
	for _, addr := range addresses {
		// Decrypt with old password
		decryptedKey, err := km.DecryptPrivateKey(addr.PrivateKey, request.OldPassword)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to decrypt disposable address %s: %v", addr.Address, err)})
			return
		}

		// Convert decrypted key to account
		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create account for disposable address %s: %v", addr.Address, err)})
			return
		}

		// Verify address matches
		if account.PublicKey.ToBase58() != addr.Address {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Address mismatch for disposable address %s", addr.Address)})
			return
		}

		// Re-encrypt with new password
		newEncryptedKey, err := km.EncryptPrivateKey(account.PrivateKey, request.NewPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to re-encrypt disposable address %s: %v", addr.Address, err)})
			return
		}

		// Add to export list
		exportAddresses = append(exportAddresses, ExportAddress{
			Address:    addr.Address,
			PrivateKey: newEncryptedKey,
		})
	}

	// Set headers for file download
	c.Header("Content-Disposition", "attachment; filename=disposable_addresses_export.json")
	c.Header("Content-Type", "application/json")

	// Send the JSON response
	c.JSON(http.StatusOK, gin.H{
		"message":   fmt.Sprintf("Successfully exported %d disposable addresses", len(exportAddresses)),
		"addresses": exportAddresses,
	})
}

// ImportAndVerifyPasswordInDisposableAddressManage handles the import and verification of disposable addresses from a JSON file
func ImportAndVerifyPasswordInDisposableAddressManage(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form: " + err.Error()})
		return
	}

	// Get password from form
	password := c.Request.FormValue("password")
	if password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
		return
	}

	// Get uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file: " + err.Error()})
		return
	}
	defer file.Close()

	// Read and parse the JSON file
	var importData struct {
		Addresses []ExportAddress `json:"addresses"`
	}
	if err := json.NewDecoder(file).Decode(&importData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON file: " + err.Error()})
		return
	}

	// Create key manager
	km := solana.NewKeyManager()

	// First verify existing disposable addresses in database
	var existingAddresses []models.DisposableAddressManage
	if err := dbconfig.DB.Find(&existingAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch existing disposable addresses: " + err.Error()})
		return
	}

	// Verify existing addresses
	for _, addr := range existingAddresses {
		decryptedKey, err := km.DecryptPrivateKey(addr.PrivateKey, password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to decrypt existing disposable address %s: %v", addr.Address, err)})
			return
		}

		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create account for existing disposable address %s: %v", addr.Address, err)})
			return
		}

		if account.PublicKey.ToBase58() != addr.Address {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Address mismatch for existing disposable address %s", addr.Address)})
			return
		}
	}

	// Verify imported addresses
	existingAddressMap := make(map[string]bool)
	for _, addr := range existingAddresses {
		existingAddressMap[addr.Address] = true
	}

	// Track new addresses to import
	var newAddresses []models.DisposableAddressManage

	// Verify and prepare imported addresses
	for _, importAddr := range importData.Addresses {
		// Skip if address already exists
		if existingAddressMap[importAddr.Address] {
			continue
		}

		// Verify the imported address
		decryptedKey, err := km.DecryptPrivateKey(importAddr.PrivateKey, password)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to decrypt imported disposable address %s: %v", importAddr.Address, err)})
			return
		}

		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create account for imported disposable address %s: %v", importAddr.Address, err)})
			return
		}

		if account.PublicKey.ToBase58() != importAddr.Address {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Address mismatch for imported disposable address %s", importAddr.Address)})
			return
		}

		// Add to new addresses list
		newAddresses = append(newAddresses, models.DisposableAddressManage{
			Address:    importAddr.Address,
			PrivateKey: importAddr.PrivateKey,
		})
	}

	// Import new addresses
	if len(newAddresses) > 0 {
		if err := dbconfig.DB.Create(&newAddresses).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import new disposable addresses: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        fmt.Sprintf("Successfully imported %d new disposable addresses", len(newAddresses)),
		"imported_count": len(newAddresses),
		"skipped_count":  len(importData.Addresses) - len(newAddresses),
	})
}

// MultiTransferSolRequest represents the request body for multi-transfer SOL
type MultiTransferSolRequest struct {
	Task                  []TransferSolTask `json:"task" binding:"required"`
	CheckTargetExistsInDB bool              `json:"check_target_exists_in_db"`
	Rps                   int               `json:"rps" binding:"required,min=1"`
}

// TransferSolTask represents a single SOL transfer task
type TransferSolTask struct {
	From     string `json:"from" binding:"required"`
	To       string `json:"to" binding:"required"`
	Lamports string `json:"lamports" binding:"required"`
}

// TransferSolResult represents the result of a SOL transfer
type TransferSolResult struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Lamports  uint64 `json:"lamports"`
	Success   bool   `json:"success"`
	Signature string `json:"signature,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SolTransferTask represents a single SOL transfer task for internal processing
type SolTransferTask struct {
	From       string
	To         string
	FromPubkey solanaGo.PublicKey
	ToPubkey   solanaGo.PublicKey
	PrivateKey *solanaGo.PrivateKey
	Lamports   uint64
}

// MultiTransferSol handles concurrent SOL transfers
func MultiTransferSol(c *gin.Context) {
	var request MultiTransferSolRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate RPS
	if request.Rps <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rps must be greater than 0"})
		return
	}

	// Get encryption password
	encryptPassword := os.Getenv("ENCRYPTPASSWORD")
	if encryptPassword == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ENCRYPTPASSWORD environment variable not set"})
		return
	}

	// Get Solana RPC endpoint
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create RPC client
	client := rpc.New(solanaRPC)

	// Create key manager
	km := solana.NewKeyManager()

	// Prepare transfer tasks
	var tasks []SolTransferTask
	var skippedTasks []TransferSolTask

	// Check target addresses in DB if required
	targetAddressSet := make(map[string]bool)
	if request.CheckTargetExistsInDB {
		var targetAddresses []string
		for _, task := range request.Task {
			targetAddresses = append(targetAddresses, task.To)
		}

		if len(targetAddresses) > 0 {
			var addressManages []models.AddressManage
			if err := dbconfig.DB.Where("address IN ?", targetAddresses).Find(&addressManages).Error; err == nil {
				for _, am := range addressManages {
					targetAddressSet[am.Address] = true
				}
			}
		}
	}

	// Process each task
	for _, task := range request.Task {
		// Validate addresses
		fromPubkey, err := solanaGo.PublicKeyFromBase58(task.From)
		if err != nil {
			log.Warnf("Invalid from address: %s, skipping", task.From)
			skippedTasks = append(skippedTasks, task)
			continue
		}

		toPubkey, err := solanaGo.PublicKeyFromBase58(task.To)
		if err != nil {
			log.Warnf("Invalid to address: %s, skipping", task.To)
			skippedTasks = append(skippedTasks, task)
			continue
		}

		// Check target exists in DB if required
		if request.CheckTargetExistsInDB {
			if !targetAddressSet[task.To] {
				log.Warnf("Target address %s not found in AddressManage, skipping", task.To)
				skippedTasks = append(skippedTasks, task)
				continue
			}
		}

		// Parse lamports
		lamports, err := strconv.ParseUint(task.Lamports, 10, 64)
		if err != nil {
			log.Warnf("Invalid lamports: %s, skipping", task.Lamports)
			skippedTasks = append(skippedTasks, task)
			continue
		}

		if lamports == 0 {
			log.Warnf("Lamports is 0, skipping")
			skippedTasks = append(skippedTasks, task)
			continue
		}

		// Get private key for from address
		var addressManage models.AddressManage
		if err := dbconfig.DB.Where("address = ?", task.From).First(&addressManage).Error; err != nil {
			log.Warnf("No private key found for address %s, skipping", task.From)
			skippedTasks = append(skippedTasks, task)
			continue
		}

		// Decrypt private key
		decryptedKey, err := km.DecryptPrivateKey(addressManage.PrivateKey, encryptPassword)
		if err != nil {
			log.Warnf("Failed to decrypt private key for address %s: %v", task.From, err)
			skippedTasks = append(skippedTasks, task)
			continue
		}

		// Convert to blocto Account
		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			log.Warnf("Failed to create account from bytes for address %s: %v", task.From, err)
			skippedTasks = append(skippedTasks, task)
			continue
		}

		// Convert blocto PrivateKey to gagliardetto PrivateKey
		privateKeyBytes := account.PrivateKey[:]
		privateKey := solanaGo.PrivateKey(privateKeyBytes)

		tasks = append(tasks, SolTransferTask{
			From:       task.From,
			To:         task.To,
			FromPubkey: fromPubkey,
			ToPubkey:   toPubkey,
			PrivateKey: &privateKey,
			Lamports:   lamports,
		})
	}

	if len(tasks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "No valid transfer tasks found",
			"skipped_count": len(skippedTasks),
			"skipped_tasks": skippedTasks,
		})
		return
	}

	// Execute transfers concurrently with rate limiting
	limiter := rate.NewLimiter(rate.Limit(request.Rps), request.Rps)
	resultCh := make(chan TransferSolResult, len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t SolTransferTask) {
			defer wg.Done()

			// Rate limiting
			if err := limiter.Wait(context.Background()); err != nil {
				resultCh <- TransferSolResult{
					From:     t.From,
					To:       t.To,
					Lamports: t.Lamports,
					Success:  false,
					Error:    fmt.Sprintf("rate limiter wait failed: %v", err),
				}
				return
			}

			// Execute transfer
			res := executeSolTransfer(client, t)
			resultCh <- res
		}(task)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var results []TransferSolResult
	for res := range resultCh {
		results = append(results, res)
	}

	// Count success and failures
	successCount := 0
	failureCount := 0
	for _, res := range results {
		if res.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"total_tasks":   len(request.Task),
		"valid_tasks":   len(tasks),
		"skipped_tasks": len(skippedTasks),
		"success_count": successCount,
		"failure_count": failureCount,
		"results":       results,
		"skipped":       skippedTasks,
	})
}

// executeSolTransfer executes a single SOL transfer
func executeSolTransfer(client *rpc.Client, task SolTransferTask) TransferSolResult {
	ctx := context.Background()

	// Get latest blockhash
	bh, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return TransferSolResult{
			From:     task.From,
			To:       task.To,
			Lamports: task.Lamports,
			Success:  false,
			Error:    fmt.Sprintf("failed to get latest blockhash: %v", err),
		}
	}

	// Create transfer instruction
	ix := system.NewTransferInstruction(
		task.Lamports,
		task.FromPubkey,
		task.ToPubkey,
	).Build()

	// Create transaction
	tx, err := solanaGo.NewTransaction(
		[]solanaGo.Instruction{ix},
		bh.Value.Blockhash,
		solanaGo.TransactionPayer(task.FromPubkey),
	)
	if err != nil {
		return TransferSolResult{
			From:     task.From,
			To:       task.To,
			Lamports: task.Lamports,
			Success:  false,
			Error:    fmt.Sprintf("failed to create transaction: %v", err),
		}
	}

	// Sign transaction
	if _, err := tx.Sign(func(key solanaGo.PublicKey) *solanaGo.PrivateKey {
		if key.Equals(task.FromPubkey) {
			return task.PrivateKey
		}
		return nil
	}); err != nil {
		return TransferSolResult{
			From:     task.From,
			To:       task.To,
			Lamports: task.Lamports,
			Success:  false,
			Error:    fmt.Sprintf("failed to sign transaction: %v", err),
		}
	}

	// Send transaction
	sig, err := client.SendTransaction(ctx, tx)
	if err != nil {
		return TransferSolResult{
			From:     task.From,
			To:       task.To,
			Lamports: task.Lamports,
			Success:  false,
			Error:    fmt.Sprintf("failed to send transaction: %v", err),
		}
	}

	return TransferSolResult{
		From:      task.From,
		To:        task.To,
		Lamports:  task.Lamports,
		Success:   true,
		Signature: sig.String(),
	}
}

// ImportCsv handles the import of addresses from a CSV file
func ImportCsv(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form: " + err.Error()})
		return
	}

	// Get uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file: " + err.Error()})
		return
	}
	defer file.Close()

	// Read and parse the CSV file
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header row
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV header: " + err.Error()})
		return
	}

	// Find column indices
	addressIdx := -1
	privateKeyIdx := -1
	for i, col := range header {
		switch col {
		case "address":
			addressIdx = i
		case "private_key":
			privateKeyIdx = i
		}
	}

	if addressIdx == -1 || privateKeyIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV must contain 'address' and 'private_key' columns"})
		return
	}

	// Get existing addresses to avoid duplicates
	var existingAddresses []models.AddressManage
	if err := dbconfig.DB.Find(&existingAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch existing addresses: " + err.Error()})
		return
	}

	existingAddressMap := make(map[string]bool)
	for _, addr := range existingAddresses {
		existingAddressMap[addr.Address] = true
	}

	// Track new addresses to import
	var newAddresses []models.AddressManage
	var skippedCount int
	var errorCount int
	var errorMessages []string

	// Read data rows
	lineNum := 1 // Start from 1 (header is line 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: Failed to read CSV row: %v", lineNum+1, err))
			continue
		}

		lineNum++

		// Check if we have enough columns
		if len(record) <= addressIdx || len(record) <= privateKeyIdx {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: Insufficient columns", lineNum))
			continue
		}

		address := record[addressIdx]
		privateKey := record[privateKeyIdx]

		// Skip empty rows
		if address == "" || privateKey == "" {
			continue
		}

		// Skip if address already exists
		if existingAddressMap[address] {
			skippedCount++
			continue
		}

		// Validate address format (basic check - should be base58)
		if len(address) < 32 || len(address) > 44 {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: Invalid address format: %s", lineNum, address))
			continue
		}

		// Add to new addresses list
		newAddresses = append(newAddresses, models.AddressManage{
			Address:    address,
			PrivateKey: privateKey,
		})

		// Mark as existing to avoid duplicates in the same import
		existingAddressMap[address] = true
	}

	// Import new addresses
	var importedCount int
	if len(newAddresses) > 0 {
		if err := dbconfig.DB.Create(&newAddresses).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import new addresses: " + err.Error()})
			return
		}
		importedCount = len(newAddresses)
	}

	response := gin.H{
		"message":         fmt.Sprintf("Successfully imported %d new addresses", importedCount),
		"imported_count":  importedCount,
		"skipped_count":   skippedCount,
		"error_count":     errorCount,
		"total_processed": lineNum - 1,
	}

	if len(errorMessages) > 0 {
		response["errors"] = errorMessages
	}

	c.JSON(http.StatusOK, response)
}

// ImportCsvInDisposableAddressManage handles the import of disposable addresses from a CSV file
func ImportCsvInDisposableAddressManage(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form: " + err.Error()})
		return
	}

	// Get uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file: " + err.Error()})
		return
	}
	defer file.Close()

	// Read and parse the CSV file
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header row
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV header: " + err.Error()})
		return
	}

	// Find column indices
	addressIdx := -1
	privateKeyIdx := -1
	for i, col := range header {
		switch col {
		case "address":
			addressIdx = i
		case "private_key":
			privateKeyIdx = i
		}
	}

	if addressIdx == -1 || privateKeyIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV must contain 'address' and 'private_key' columns"})
		return
	}

	// Get existing disposable addresses to avoid duplicates
	var existingAddresses []models.DisposableAddressManage
	if err := dbconfig.DB.Find(&existingAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch existing disposable addresses: " + err.Error()})
		return
	}

	existingAddressMap := make(map[string]bool)
	for _, addr := range existingAddresses {
		existingAddressMap[addr.Address] = true
	}

	// Track new addresses to import
	var newAddresses []models.DisposableAddressManage
	var skippedCount int
	var errorCount int
	var errorMessages []string

	// Read data rows
	lineNum := 1 // Start from 1 (header is line 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: Failed to read CSV row: %v", lineNum+1, err))
			continue
		}

		lineNum++

		// Check if we have enough columns
		if len(record) <= addressIdx || len(record) <= privateKeyIdx {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: Insufficient columns", lineNum))
			continue
		}

		address := record[addressIdx]
		privateKey := record[privateKeyIdx]

		// Skip empty rows
		if address == "" || privateKey == "" {
			continue
		}

		// Skip if address already exists
		if existingAddressMap[address] {
			skippedCount++
			continue
		}

		// Validate address format (basic check - should be base58)
		if len(address) < 32 || len(address) > 44 {
			errorCount++
			errorMessages = append(errorMessages, fmt.Sprintf("Line %d: Invalid address format: %s", lineNum, address))
			continue
		}

		// Add to new addresses list
		newAddresses = append(newAddresses, models.DisposableAddressManage{
			Address:      address,
			PrivateKey:   privateKey,
			IsDeprecated: false,
		})

		// Mark as existing to avoid duplicates in the same import
		existingAddressMap[address] = true
	}

	// Import new addresses
	var importedCount int
	if len(newAddresses) > 0 {
		if err := dbconfig.DB.Create(&newAddresses).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import new disposable addresses: " + err.Error()})
			return
		}
		importedCount = len(newAddresses)
	}

	response := gin.H{
		"message":         fmt.Sprintf("Successfully imported %d new disposable addresses", importedCount),
		"imported_count":  importedCount,
		"skipped_count":   skippedCount,
		"error_count":     errorCount,
		"total_processed": lineNum - 1,
	}

	if len(errorMessages) > 0 {
		response["errors"] = errorMessages
	}

	c.JSON(http.StatusOK, response)
}
