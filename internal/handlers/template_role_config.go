package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// CreateTemplateRoleConfig 创建模板角色配置
func CreateTemplateRoleConfig(c *gin.Context) {
	var req models.TemplateRoleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := dbconfig.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

// GetTemplateRoleConfig 获取模板角色配置
func GetTemplateRoleConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var config models.TemplateRoleConfig
	if err := dbconfig.DB.Preload("Addresses").First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// ListTemplateRoleConfigs 列出所有模板角色配置
func ListTemplateRoleConfigs(c *gin.Context) {
	var configs []models.TemplateRoleConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// UpdateTemplateRoleConfig 更新模板角色配置
func UpdateTemplateRoleConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var req models.TemplateRoleConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var config models.TemplateRoleConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	config.RoleName = req.RoleName
	config.UpdateInterval = req.UpdateInterval
	config.UpdateEnabled = req.UpdateEnabled
	config.LastUpdateAt = req.LastUpdateAt
	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// DeleteTemplateRoleConfig 删除模板角色配置
func DeleteTemplateRoleConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	if err := dbconfig.DB.Delete(&models.TemplateRoleConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// CreateTemplateRoleConfigByCopy 通过 role_id 复制 RoleConfig 及其 RoleAddress 到模板表
func CreateTemplateRoleConfigByCopy(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}
	// 查找 RoleConfig
	var roleConfig models.RoleConfig
	if err := dbconfig.DB.Preload("Addresses").First(&roleConfig, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "RoleConfig not found"})
		return
	}
	// 复制 RoleConfig 到 TemplateRoleConfig
	templateConfig := models.TemplateRoleConfig{
		RoleName:       roleConfig.RoleName,
		UpdateInterval: roleConfig.UpdateInterval,
		UpdateEnabled:  roleConfig.UpdateEnabled,
		LastUpdateAt:   roleConfig.LastUpdateAt,
	}
	if err := dbconfig.DB.Create(&templateConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 复制 RoleAddress 到 TemplateRoleAddress
	for _, addr := range roleConfig.Addresses {
		templateAddr := models.TemplateRoleAddress{
			TemplateRoleID: templateConfig.ID,
			Address:        addr.Address,
		}
		if err := dbconfig.DB.Create(&templateAddr).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusCreated, templateConfig)
}

// CreateTemplateRoleConfigByWithAddress 支持通过参数和文件创建模板角色及其地址
func CreateTemplateRoleConfigByWithAddress(c *gin.Context) {
	roleName := c.PostForm("role_name")
	updateIntervalStr := c.PostForm("update_interval")
	updateEnabledStr := c.PostForm("update_enabled")

	if roleName == "" || updateIntervalStr == "" || updateEnabledStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_name, update_interval, update_enabled are required"})
		return
	}

	updateInterval, err := strconv.ParseFloat(updateIntervalStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "update_interval must be a number"})
		return
	}
	updateEnabled, err := strconv.ParseBool(updateEnabledStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "update_enabled must be a boolean"})
		return
	}

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

	// 创建 TemplateRoleConfig
	templateConfig := models.TemplateRoleConfig{
		RoleName:       roleName,
		UpdateInterval: updateInterval,
		UpdateEnabled:  updateEnabled,
	}
	if err := dbconfig.DB.Create(&templateConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建 TemplateRoleAddress
	for address := range addressMap {
		templateAddr := models.TemplateRoleAddress{
			TemplateRoleID: templateConfig.ID,
			Address:        address,
		}
		if err := dbconfig.DB.Create(&templateAddr).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusCreated, templateConfig)
} 