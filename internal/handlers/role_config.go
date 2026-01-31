package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"

	"github.com/blocto/solana-go-sdk/types"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gin-gonic/gin"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	keyManager "marketcontrol/pkg/solana"
	solanaUtils "marketcontrol/pkg/solana"

	log "github.com/sirupsen/logrus"
)

// RoleConfigRequest represents the request body for creating/updating a role config
type RoleConfigRequest struct {
	RoleName       string  `json:"role_name" binding:"required"`
	UpdateInterval float64 `json:"update_interval"`
	UpdateEnabled  bool    `json:"update_enabled"`
	Hidden         bool    `json:"hidden"`
}

// CreateRoleConfigWithProjectRequest represents the request body for creating a role config with project relation
type CreateRoleConfigWithProjectRequest struct {
	RoleName       string  `json:"role_name" binding:"required"`
	UpdateInterval float64 `json:"update_interval"`
	UpdateEnabled  bool    `json:"update_enabled"`
	Hidden         bool    `json:"hidden"`
	ProjectID      uint    `json:"project_id" binding:"required"`
}

// RoleAddressRequest represents the request body for creating/updating a role address
type RoleAddressRequest struct {
	RoleID  uint   `json:"role_id" binding:"required"`
	Address string `json:"address" binding:"required"`
}

// RoleConfigRelationRequest represents the request body for creating a role config relation
type RoleConfigRelationRequest struct {
	ProjectID uint `json:"project_id" binding:"required"`
	RoleID    uint `json:"role_id" binding:"required"`
}

// ListRoleConfigs returns a list of all role configs
func ListRoleConfigs(c *gin.Context) {
	var roles []models.RoleConfig
	if err := dbconfig.DB.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, roles)
}

// GetRoleConfig returns a specific role config by ID
func GetRoleConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var role models.RoleConfig
	if err := dbconfig.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, role)
}

// CreateRoleConfig creates a new role config
func CreateRoleConfig(c *gin.Context) {
	var request RoleConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := models.RoleConfig{
		RoleName:       request.RoleName,
		UpdateInterval: request.UpdateInterval,
		UpdateEnabled:  request.UpdateEnabled,
		Hidden:         request.Hidden,
	}

	if err := dbconfig.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, role)
}

// UpdateRoleConfig updates an existing role config
func UpdateRoleConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// Read the raw JSON to check if any field is provided
	var rawJSON map[string]interface{}
	if err := c.ShouldBindJSON(&rawJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if at least one valid field is provided
	if _, hasRoleName := rawJSON["role_name"]; !hasRoleName {
		if _, hasUpdateInterval := rawJSON["update_interval"]; !hasUpdateInterval {
			if _, hasUpdateEnabled := rawJSON["update_enabled"]; !hasUpdateEnabled {
				if _, hasHidden := rawJSON["hidden"]; !hasHidden {
					c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field (role_name, update_interval, update_enabled, or hidden) must be provided"})
					return
				}
			}
		}
	}

	// var request RoleConfigRequest
	// if err := json.NewDecoder(c.Request.Body).Decode(&request); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	var role models.RoleConfig
	if err := dbconfig.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Update only provided fields
	if roleName, exists := rawJSON["role_name"]; exists {
		role.RoleName = roleName.(string)
	}
	if updateInterval, exists := rawJSON["update_interval"]; exists {
		role.UpdateInterval = updateInterval.(float64)
	}
	if updateEnabled, exists := rawJSON["update_enabled"]; exists {
		role.UpdateEnabled = updateEnabled.(bool)
	}
	if hidden, exists := rawJSON["hidden"]; exists {
		role.Hidden = hidden.(bool)
	}

	if err := dbconfig.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, role)
}

// DeleteRoleConfig deletes a role config
func DeleteRoleConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.RoleConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// AddressWithRoleInfo represents the response structure for address with role information
type AddressWithRoleInfo struct {
	Address   string               `json:"address"`
	RoleCount int                  `json:"role_count"`
	RoleLists []*models.RoleConfig `json:"role_lists"`
}

// ListRoleAddresses returns a list of all role addresses with role information
func ListRoleAddresses(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "150"))
	order := c.DefaultQuery("order", "desc") // 默认降序

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

	// 验证排序参数
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	var addresses []models.RoleAddress
	if err := dbconfig.DB.Preload("Role").Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 使用 map 来统计和去重
	addressMap := make(map[string]*AddressWithRoleInfo)

	// 遍历所有地址，进行统计和角色信息收集
	for _, addr := range addresses {
		if info, exists := addressMap[addr.Address]; exists {
			// 地址已存在，增加计数并添加角色信息
			info.RoleCount++
			// 检查角色是否已添加
			roleExists := false
			for _, role := range info.RoleLists {
				if role.ID == addr.Role.ID {
					roleExists = true
					break
				}
			}
			if !roleExists {
				info.RoleLists = append(info.RoleLists, addr.Role)
			}
		} else {
			// 新地址，创建记录
			addressMap[addr.Address] = &AddressWithRoleInfo{
				Address:   addr.Address,
				RoleCount: 1,
				RoleLists: []*models.RoleConfig{addr.Role},
			}
		}
	}

	// 转换 map 为 slice 并排序
	result := make([]AddressWithRoleInfo, 0, len(addressMap))
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
	var pageData []AddressWithRoleInfo
	if start < total {
		pageData = result[start:end]
	} else {
		pageData = []AddressWithRoleInfo{}
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

// GetRoleAddress returns a specific role address by ID
func GetRoleAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var address models.RoleAddress
	if err := dbconfig.DB.First(&address, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, address)
}

// CreateRoleAddress creates a new role address
func CreateRoleAddress(c *gin.Context) {
	var request RoleAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	address := models.RoleAddress{
		RoleID:  request.RoleID,
		Address: request.Address,
	}

	if err := dbconfig.DB.Create(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, address)
}

// DeleteRoleAddress deletes a role address
func DeleteRoleAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.RoleAddress{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// GetRoleConfigByProjectID returns all role configs for a given project_id
func GetRoleConfigByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var relations []models.RoleConfigRelation
	if err := dbconfig.DB.Where("project_id = ?", projectID).Find(&relations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var roleIDs []uint
	for _, relation := range relations {
		roleIDs = append(roleIDs, relation.RoleID)
	}

	var roles []models.RoleConfig
	if err := dbconfig.DB.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// GetRoleAddressByRoleID returns all role addresses for a given role_id with pagination
func GetRoleAddressByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	// 查询总记录数
	var total int64
	if err := dbconfig.DB.Model(&models.RoleAddress{}).
		Where("role_id = ?", roleID).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取角色地址数据
	var addresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      addresses,
	})
}

// BatchCreateRoleAddressRequest represents the request body for batch creating role addresses
type BatchCreateRoleAddressRequest struct {
	RoleID uint `json:"role_id" binding:"required"`
}

// BatchCreateRoleAddress handles batch creation of role addresses from a JSON file
func BatchCreateRoleAddress(c *gin.Context) {
	// Get role_id from query parameter
	roleID, err := strconv.Atoi(c.Query("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id"})
		return
	}

	// Read the uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	// Read file content
	content, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Parse JSON content
	var addressMap map[string]string
	if err := json.Unmarshal(content, &addressMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Start a transaction
	tx := dbconfig.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Process each address
	for address, privateKey := range addressMap {
		// Check if address exists in AddressManage
		var addressManage models.AddressManage
		result := tx.Where("address = ?", address).First(&addressManage)

		// If address doesn't exist, create it
		if result.Error != nil {
			addressManage = models.AddressManage{
				Address:    address,
				PrivateKey: privateKey,
			}
			if err := tx.Create(&addressManage).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create address manage entry"})
				return
			}
		}

		// Create RoleAddress entry
		roleAddress := models.RoleAddress{
			RoleID:  uint(roleID),
			Address: address,
		}
		if err := tx.Create(&roleAddress).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role address entry"})
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Successfully created role addresses"})
}

// DeleteRoleAddressByRoleID deletes all role addresses for a given role_id
func DeleteRoleAddressByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	result := dbconfig.DB.Where("role_id = ?", roleID).Delete(&models.RoleAddress{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Role addresses deleted successfully",
		"deleted_count": result.RowsAffected,
	})
}

// DeleteRoleConfigWithAddressByRoleID deletes a role config and all its addresses after checking strategy dependencies
func DeleteRoleConfigWithAddressByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	// 1. 检查是否存在依赖的策略
	var strategyCount int64
	if err := dbconfig.DB.Model(&models.StrategyConfig{}).Where("role_id = ?", roleID).Count(&strategyCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check strategy dependencies"})
		return
	}

	if strategyCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Cannot delete role: there are strategies depending on this role",
			"strategy_count": strategyCount,
		})
		return
	}

	// 2. 检查是否存在 RoleConfigRelation
	var relationCount int64
	if err := dbconfig.DB.Model(&models.RoleConfigRelation{}).Where("role_id = ?", roleID).Count(&relationCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check role config relations"})
		return
	}

	if relationCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Cannot delete role: there are project relations depending on this role",
			"relation_count": relationCount,
		})
		return
	}

	// 3. 开启事务进行删除
	tx := dbconfig.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// 4. 删除关联的地址
	addressResult := tx.Where("role_id = ?", roleID).Delete(&models.RoleAddress{})
	if addressResult.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role addresses"})
		return
	}

	// 5. 删除角色配置
	roleResult := tx.Delete(&models.RoleConfig{}, roleID)
	if roleResult.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role config"})
		return
	}

	if roleResult.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Role config not found"})
		return
	}

	// 6. 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                 "Role config and addresses deleted successfully",
		"deleted_addresses_count": addressResult.RowsAffected,
	})
}

// CreateRoleConfigByTemplateID 根据模板创建 RoleConfig 和 RoleAddress
func CreateRoleConfigByTemplateID(c *gin.Context) {
	type Req struct {
		ProjectID  uint `json:"project_id" binding:"required"`
		TemplateID uint `json:"template_id" binding:"required"`
	}
	var req Req
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查 ProjectConfig 是否存在
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, req.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id: ProjectConfig not found"})
		return
	}

	// 查询模板
	var template models.TemplateRoleConfig
	if err := dbconfig.DB.Preload("Addresses").First(&template, req.TemplateID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "TemplateRoleConfig not found"})
		return
	}

	// 开始事务
	tx := dbconfig.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// 创建 RoleConfig
	roleConfig := models.RoleConfig{
		RoleName:       template.RoleName,
		UpdateInterval: template.UpdateInterval,
		UpdateEnabled:  template.UpdateEnabled,
		Hidden:         false, // default when creating from template
		LastUpdateAt:   template.LastUpdateAt,
	}

	if err := tx.Create(&roleConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建关联关系
	relation := models.RoleConfigRelation{
		RoleID:    roleConfig.ID,
		ProjectID: req.ProjectID,
	}

	if err := tx.Create(&relation).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建 RoleAddress
	for _, addr := range template.Addresses {
		roleAddr := models.RoleAddress{
			RoleID:  roleConfig.ID,
			Address: addr.Address,
		}
		if err := tx.Create(&roleAddr).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, roleConfig)
}

// GetRoleAddressCountByRoleID returns the count of addresses for a specific role
func GetRoleAddressCountByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var count int64
	if err := dbconfig.DB.Model(&models.RoleAddress{}).Where("role_id = ?", roleID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"role_id": roleID,
		"count":   count,
	})
}

// GetTotalRoleAddressByRoleID returns all role addresses for a given role_id without pagination
func GetTotalRoleAddressByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	// 检查 RoleConfig 是否存在
	var roleConfig models.RoleConfig
	if err := dbconfig.DB.First(&roleConfig, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role config not found"})
		return
	}

	// 查询总记录数
	var total int64
	if err := dbconfig.DB.Model(&models.RoleAddress{}).
		Where("role_id = ?", roleID).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取所有角色地址数据
	var addresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).
		Order("id DESC").
		Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"role_id":   roleID,
		"role_name": roleConfig.RoleName,
		"total":     total,
		"data":      addresses,
	})
}

// CreateRoleConfigRelation creates a new role config relation
func CreateRoleConfigRelation(c *gin.Context) {
	var request RoleConfigRelationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查 role_id 是否存在
	var role models.RoleConfig
	if err := dbconfig.DB.First(&role, request.RoleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id: RoleConfig not found"})
		return
	}

	// 检查 project_id 是否存在
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, request.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id: ProjectConfig not found"})
		return
	}

	relation := models.RoleConfigRelation{
		RoleID:    request.RoleID,
		ProjectID: request.ProjectID,
	}

	if err := dbconfig.DB.Create(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, relation)
}

// DeleteRoleConfigRelation deletes a role config relation by ID
func DeleteRoleConfigRelation(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	result := dbconfig.DB.Delete(&models.RoleConfigRelation{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// MigrateAllRoleConfig migrates all existing role configs to role config relations
func MigrateAllRoleConfig(c *gin.Context) {
	// var roles []models.RoleConfig
	// if err := dbconfig.DB.Find(&roles).Error; err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }

	// successCount := 0
	// errorCount := 0
	// errors := make([]string, 0)

	// for _, role := range roles {
	// 	relation := models.RoleConfigRelation{
	// 		RoleID:    role.ID,
	// 		ProjectID: role.ProjectID,
	// 	}

	// 	if err := dbconfig.DB.Create(&relation).Error; err != nil {
	// 		errorCount++
	// 		errors = append(errors, fmt.Sprintf("Failed to create relation for role_id %d: %v", role.ID, err))
	// 		continue
	// 	}

	// 	successCount++
	// }

	// c.JSON(http.StatusOK, gin.H{
	// 	"message": "Migration completed",
	// 	"success_count": successCount,
	// 	"error_count": errorCount,
	// 	"errors": errors,
	// })
}

// DeleteRoleConfigRelationByFilterRequest represents the request body for filtering role config relations to delete
type DeleteRoleConfigRelationByFilterRequest struct {
	ProjectID *uint `json:"project_id"`
	RoleID    *uint `json:"role_id"`
}

// DeleteRoleConfigRelationByFilter deletes role config relations based on project_id and role_id filters
func DeleteRoleConfigRelationByFilter(c *gin.Context) {
	var request DeleteRoleConfigRelationByFilterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 如果没有提供任何筛选条件，返回错误
	if request.ProjectID == nil && request.RoleID == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least one filter (project_id or role_id) must be provided",
		})
		return
	}

	// 构建策略查询条件
	strategyQuery := dbconfig.DB.Model(&models.StrategyConfig{})
	if request.ProjectID != nil {
		strategyQuery = strategyQuery.Where("project_id = ?", *request.ProjectID)
	}
	if request.RoleID != nil {
		strategyQuery = strategyQuery.Where("role_id = ?", *request.RoleID)
	}

	// 检查是否存在依赖的策略
	var strategyCount int64
	if err := strategyQuery.Count(&strategyCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check strategy dependencies"})
		return
	}

	if strategyCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Cannot delete role config relations: there are strategies depending on these relations",
			"strategy_count": strategyCount,
		})
		return
	}

	// 构建删除查询条件
	query := dbconfig.DB.Model(&models.RoleConfigRelation{})
	if request.ProjectID != nil {
		query = query.Where("project_id = ?", *request.ProjectID)
	}
	if request.RoleID != nil {
		query = query.Where("role_id = ?", *request.RoleID)
	}

	// 执行删除操作
	result := query.Delete(&models.RoleConfigRelation{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No records found matching the filter criteria"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Records deleted successfully",
		"deleted_count": result.RowsAffected,
	})
}

// GetRoleConfigRelationByRoleID returns all role config relations for a given role_id
func GetRoleConfigRelationByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var relations []models.RoleConfigRelation
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&relations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(relations) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No relations found for this role_id"})
		return
	}

	c.JSON(http.StatusOK, relations)
}

// GetRoleConfigRelationByProjectID returns all role config relations for a given project_id
func GetRoleConfigRelationByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var relations []models.RoleConfigRelation
	if err := dbconfig.DB.Where("project_id = ?", projectID).Find(&relations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(relations) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No relations found for this project_id"})
		return
	}

	c.JSON(http.StatusOK, relations)
}

// ListRoleConfigRelations returns a list of all role config relations
func ListRoleConfigRelations(c *gin.Context) {
	var relations []models.RoleConfigRelation
	if err := dbconfig.DB.Find(&relations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, relations)
}

// BatchAddressRequest represents the request body for batch adding/deleting role addresses
type BatchAddressRequest struct {
	RoleID       uint     `json:"role_id" binding:"required"`
	AddressLists []string `json:"address_lists" binding:"required"`
}

// BatchAddRoleAddress handles batch addition of role addresses
func BatchAddRoleAddress(c *gin.Context) {
	var request BatchAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查 RoleConfig 是否存在
	var roleConfig models.RoleConfig
	if err := dbconfig.DB.First(&roleConfig, request.RoleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role config not found"})
		return
	}

	// 开始事务
	tx := dbconfig.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	successCount := 0
	skippedCount := 0
	failedAddresses := make([]string, 0)

	// 遍历地址列表
	for _, address := range request.AddressLists {
		// 检查地址是否存在于 AddressManage
		var addressManage models.AddressManage
		if err := tx.Where("address = ?", address).First(&addressManage).Error; err != nil {
			failedAddresses = append(failedAddresses, address)
			continue
		}

		// 检查地址是否已存在于 RoleAddress
		var existingRoleAddress models.RoleAddress
		if err := tx.Where("role_id = ? AND address = ?", request.RoleID, address).First(&existingRoleAddress).Error; err == nil {
			skippedCount++
			continue
		}

		// 创建新的 RoleAddress
		roleAddress := models.RoleAddress{
			RoleID:  request.RoleID,
			Address: address,
		}
		if err := tx.Create(&roleAddress).Error; err != nil {
			failedAddresses = append(failedAddresses, address)
			continue
		}
		successCount++
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Batch add role addresses completed",
		"success_count":    successCount,
		"skipped_count":    skippedCount,
		"failed_addresses": failedAddresses,
	})
}

// BatchDeleteRoleAddress handles batch deletion of role addresses
func BatchDeleteRoleAddress(c *gin.Context) {
	var request BatchAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查 RoleConfig 是否存在
	var roleConfig models.RoleConfig
	if err := dbconfig.DB.First(&roleConfig, request.RoleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Role config not found"})
		return
	}

	// 开始事务
	tx := dbconfig.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	deletedCount := 0
	notFoundCount := 0
	failedAddresses := make([]string, 0)

	// 遍历地址列表
	for _, address := range request.AddressLists {
		result := tx.Where("role_id = ? AND address = ?", request.RoleID, address).Delete(&models.RoleAddress{})
		if result.Error != nil {
			failedAddresses = append(failedAddresses, address)
			continue
		}
		if result.RowsAffected > 0 {
			deletedCount++
		} else {
			notFoundCount++
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Batch delete role addresses completed",
		"deleted_count":    deletedCount,
		"not_found_count":  notFoundCount,
		"failed_addresses": failedAddresses,
	})
}

// ExportAddressByRoleID exports all addresses for a given role_id in the specified format
func ExportAddressByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	// 检查 RoleConfig 是否存在
	var roleConfig models.RoleConfig
	if err := dbconfig.DB.First(&roleConfig, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role config not found"})
		return
	}

	// 获取所有角色地址数据
	var addresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).
		Order("id DESC").
		Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 提取地址列表
	addressLists := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		addressLists = append(addressLists, addr.Address)
	}

	// 返回指定格式
	c.JSON(http.StatusOK, gin.H{
		"role_id":       roleID,
		"address_lists": addressLists,
	})
}

// TransferMintToTargetByRoleRequest represents the request body for transferring mint to target by role
type TransferMintToTargetByRoleRequest struct {
	RoleID uint   `json:"role_id" binding:"required"`
	Target string `json:"target" binding:"required"`
	Mint   string `json:"mint" binding:"required"`
	Rps    int    `json:"rps" binding:"required"`
}

// TransferMintToTargetByRole transfers mint tokens from all addresses in a role to target address
func TransferMintToTargetByRole(c *gin.Context) {
	var request TransferMintToTargetByRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate addresses
	if _, err := solana.PublicKeyFromBase58(request.Target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target address format"})
		return
	}

	if _, err := solana.PublicKeyFromBase58(request.Mint); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mint address format"})
		return
	}

	// Validate rps
	if request.Rps <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "rps must be greater than 0"})
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
	accounts := make([]string, 0, len(roleAddresses))
	for _, roleAddr := range roleAddresses {
		accounts = append(accounts, roleAddr.Address)
	}

	// Get private keys for all addresses from AddressManage table
	encryptPassword := os.Getenv("ENCRYPTPASSWORD")
	if encryptPassword == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "ENCRYPTPASSWORD environment variable not set"})
		return
	}

	km := keyManager.NewKeyManager()
	accountToPrivateKey := make(map[string]*solana.PrivateKey)

	for _, accountStr := range accounts {
		// Get encrypted private key from database
		var addressManage models.AddressManage
		if err := dbconfig.DB.Where("address = ?", accountStr).First(&addressManage).Error; err != nil {
			log.Warnf("No private key found for address %s, skipping", accountStr)
			continue
		}

		// Decrypt private key
		decryptedKey, err := km.DecryptPrivateKey(addressManage.PrivateKey, encryptPassword)
		if err != nil {
			log.Warnf("Failed to decrypt private key for address %s: %v", accountStr, err)
			continue
		}

		// Convert to blocto Account
		account, err := types.AccountFromBytes(decryptedKey)
		if err != nil {
			log.Warnf("Failed to create account from bytes for address %s: %v", accountStr, err)
			continue
		}

		// Convert blocto PrivateKey to gagliardetto PrivateKey
		// blocto Account.PrivateKey is [64]byte, we need to convert it to []byte first
		privateKeyBytes := account.PrivateKey[:]
		privateKey := solana.PrivateKey(privateKeyBytes)

		accountToPrivateKey[accountStr] = &privateKey
	}

	if len(accountToPrivateKey) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No valid private keys found for addresses in this role",
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

	// Call MultiTransferMintToTarget
	results, err := solanaUtils.MultiTransferMintToTarget(
		client,
		accounts,
		request.Mint,
		request.Target,
		request.Rps,
		accountToPrivateKey,
		6,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to transfer mint: %v", err)})
		return
	}

	// Count success and failed transactions
	successCount := 0
	failedCount := 0
	totalCount := 0
	if results != nil {
		totalCount = len(results)
		for _, res := range results {
			if res.Success {
				successCount++
			} else {
				failedCount++
			}
		}
	}

	// Return results
	c.JSON(http.StatusOK, gin.H{
		"role_id":       request.RoleID,
		"target":        request.Target,
		"mint":          request.Mint,
		"rps":           request.Rps,
		"results":       results,
		"success_count": successCount,
		"failed_count":  failedCount,
		"total_count":   totalCount,
		"rpc":           solanaRPC,
	})
}

// GetRoleAddressSolBalances returns SOL balances for all addresses in a role
func GetRoleAddressSolBalances(c *gin.Context) {
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

	// Get all addresses for this role_id
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get addresses for role_id %d: %v", roleID, err)})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"role_id":  roleID,
			"count":    0,
			"balances": []interface{}{},
		})
		return
	}

	// Extract addresses and validate format
	addresses := make([]string, 0, len(roleAddresses))
	accountPubkeys := make([]solana.PublicKey, 0, len(roleAddresses))

	for _, roleAddr := range roleAddresses {
		// Validate address format
		pubkey, err := solana.PublicKeyFromBase58(roleAddr.Address)
		if err != nil {
			log.Warnf("Invalid address format in role_address table: %s, skipping", roleAddr.Address)
			continue
		}
		addresses = append(addresses, roleAddr.Address)
		accountPubkeys = append(accountPubkeys, pubkey)
	}

	if len(accountPubkeys) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No valid addresses found for this role_id",
			"role_id": roleID,
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

	// Get SOL balances using GetMultiAccountsSol
	lamportsMap, err := solanaUtils.GetMultiAccountsSol(client, accountPubkeys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get SOL balances: %v", err)})
		return
	}

	// Format response data
	type BalanceInfo struct {
		Address            string  `json:"address"`
		Lamports           uint64  `json:"lamports"`
		SolBalance         float64 `json:"sol_balance"`
		SolBalanceReadable string  `json:"sol_balance_readable"`
	}

	balances := make([]BalanceInfo, 0, len(addresses))
	for _, address := range addresses {
		lamports := lamportsMap[address]
		solBalance := float64(lamports) / 1e9

		balances = append(balances, BalanceInfo{
			Address:            address,
			Lamports:           lamports,
			SolBalance:         solBalance,
			SolBalanceReadable: fmt.Sprintf("%.9f", solBalance),
		})
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"role_id":  roleID,
		"count":    len(balances),
		"balances": balances,
	})
}

// CheckRoleAddressExistRequest represents the request body for checking if addresses exist for a role
type CheckRoleAddressExistRequest struct {
	RoleID       uint     `json:"role_id" binding:"required"`
	AddressLists []string `json:"address_lists" binding:"required"`
}

// CheckRoleAddressExist checks if addresses exist for a specific role
func CheckRoleAddressExist(c *gin.Context) {
	var request CheckRoleAddressExistRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(request.AddressLists) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address_lists cannot be empty"})
		return
	}

	// Get existing addresses from database for the specific role
	var existingAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ? AND address IN ?", request.RoleID, request.AddressLists).Find(&existingAddresses).Error; err != nil {
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
