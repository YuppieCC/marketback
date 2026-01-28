package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"

	"github.com/gin-gonic/gin"
)

// StrategyConfigRequest represents the request body for creating/updating a strategy config
type StrategyConfigRequest struct {
	ProjectID      uint            `json:"project_id" binding:"required"`
	RoleID         uint            `json:"role_id" binding:"required"`
	StrategyName   string          `json:"strategy_name" binding:"required"`
	StrategyType   string          `json:"strategy_type" binding:"required"`
	StrategyParams json.RawMessage `json:"strategy_params"`
	StrategyStat   json.RawMessage `json:"strategy_stat"`
	Enabled        *bool           `json:"enabled"`
}

// UpdateStrategyParamsRequest represents the request body for updating strategy params
type UpdateStrategyParamsRequest struct {
	StrategyParams json.RawMessage `json:"strategy_params" binding:"required"`
	StrategyName   string          `json:"strategy_name"`
}

// UpdateStrategyStatRequest represents the request body for updating strategy stat
type UpdateStrategyStatRequest struct {
	StrategyStat json.RawMessage `json:"strategy_stat" binding:"required"`
}

// ListStrategyConfigs returns a list of all strategy configs
func ListStrategyConfigs(c *gin.Context) {
	var strategies []models.StrategyConfig
	if err := dbconfig.DB.Find(&strategies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, strategies)
}

// ListStrategyConfigsByProjectId returns strategy configs filtered by project_id
func ListStrategyConfigsByProjectId(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var strategies []models.StrategyConfig
	if err := dbconfig.DB.Where("project_id = ?", projectID).Find(&strategies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, strategies)
}

// GetStrategyConfig returns a specific strategy config by ID
func GetStrategyConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var strategy models.StrategyConfig
	if err := dbconfig.DB.First(&strategy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, strategy)
}

// CreateStrategyConfig creates a new strategy config
func CreateStrategyConfig(c *gin.Context) {
	var request StrategyConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	strategy := models.StrategyConfig{
		ProjectID:      request.ProjectID,
		RoleID:         request.RoleID,
		StrategyName:   request.StrategyName,
		StrategyType:   request.StrategyType,
		StrategyParams: request.StrategyParams,
		StrategyStat:   request.StrategyStat,
	}

	// Set Enabled field - if not provided, use the model's default value (false)
	if request.Enabled != nil {
		strategy.Enabled = *request.Enabled
	}

	if err := dbconfig.DB.Create(&strategy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, strategy)
}

// UpdateStrategyConfig updates an existing strategy config
func UpdateStrategyConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request StrategyConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var strategy models.StrategyConfig
	if err := dbconfig.DB.First(&strategy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	strategy.ProjectID = request.ProjectID
	strategy.RoleID = request.RoleID
	strategy.StrategyName = request.StrategyName
	strategy.StrategyType = request.StrategyType
	strategy.StrategyParams = request.StrategyParams
	strategy.StrategyStat = request.StrategyStat

	// Update Enabled field if provided
	if request.Enabled != nil {
		strategy.Enabled = *request.Enabled
	}

	if err := dbconfig.DB.Save(&strategy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, strategy)
}

// UpdateStrategyParams updates only the strategy_params field
func UpdateStrategyParams(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request UpdateStrategyParamsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var strategy models.StrategyConfig
	if err := dbconfig.DB.First(&strategy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	strategy.StrategyParams = request.StrategyParams
	if request.StrategyName != "" {
		strategy.StrategyName = request.StrategyName
	}

	if err := dbconfig.DB.Save(&strategy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, strategy)
}

// UpdateStrategyStat updates only the strategy_stat field
func UpdateStrategyStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request UpdateStrategyStatRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var strategy models.StrategyConfig
	if err := dbconfig.DB.First(&strategy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	strategy.StrategyStat = request.StrategyStat

	if err := dbconfig.DB.Save(&strategy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, strategy)
}

// DeleteStrategyConfig deletes a strategy config
func DeleteStrategyConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.StrategyConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// ToggleStrategyConfig 切换策略的启用状态
func ToggleStrategyConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var strategy models.StrategyConfig
	if err := dbconfig.DB.First(&strategy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	// 切换启用状态
	strategy.Enabled = !strategy.Enabled

	// 保存更新
	if err := dbconfig.DB.Save(&strategy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update strategy"})
		return
	}

	message := "Strategy disabled successfully"
	if strategy.Enabled {
		message = "Strategy enabled successfully"
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      strategy.ID,
		"enabled": strategy.Enabled,
		"message": message,
	})
}

// CloseStrategyConfigsByProjectId closes all strategy configs for a specific project by setting Enabled to false
func CloseStrategyConfigsByProjectId(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// 批量更新所有匹配的 strategy configs，将 Enabled 设置为 false
	result := dbconfig.DB.Model(&models.StrategyConfig{}).Where("project_id = ?", projectID).Update("enabled", false)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "All strategies for project closed successfully",
		"project_id":   projectID,
		"rows_updated": result.RowsAffected,
	})
}

// CloseStrategyTypeRequest represents the request body for closing a specific strategy type by project
type CloseStrategyTypeRequest struct {
	ProjectID    uint   `json:"project_id" binding:"required"`
	StrategyType string `json:"strategy_type" binding:"required"`
}

// CloseStrategyTypeByProjectId closes all strategies of a specific type for a project
func CloseStrategyTypeByProjectId(c *gin.Context) {
	var request CloseStrategyTypeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := dbconfig.DB.Model(&models.StrategyConfig{}).
		Where("project_id = ? AND strategy_type = ?", request.ProjectID, request.StrategyType).
		Update("enabled", false)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Strategies closed successfully",
		"project_id":    request.ProjectID,
		"strategy_type": request.StrategyType,
		"rows_updated":  result.RowsAffected,
	})
}

type CheckStrategyCloseRequest struct {
	ProjectID    uint   `json:"project_id" binding:"required"`
	StrategyType string `json:"strategy_type" binding:"required"`
}

func CheckStrategyCloseByProjectId(c *gin.Context) {
	var request CheckStrategyCloseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 正确方式：Count 到变量中
	var count int64
	err := dbconfig.DB.Model(&models.StrategyConfig{}).
		Where("project_id = ? AND strategy_type = ? AND enabled = true", request.ProjectID, request.StrategyType).
		Count(&count).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	isClosed := count == 0

	c.JSON(http.StatusOK, gin.H{
		"message":       "Strategy check successful",
		"project_id":    request.ProjectID,
		"strategy_type": request.StrategyType,
		"count":         count,
		"is_closed":     isClosed,
	})
}
