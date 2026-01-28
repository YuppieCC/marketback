package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// StrategySignalRequest represents the request body for creating/updating a strategy signal
type StrategySignalRequest struct {
	ProjectID         uint            `json:"project_id" binding:"required"`
	StrategyID        uint            `json:"strategy_id" binding:"required"`
	RoleID            uint            `json:"role_id" binding:"required"`
	StrategyParams    json.RawMessage `json:"strategy_params"`
	StrategyStat      json.RawMessage `json:"strategy_stat"`
	TransactionParams json.RawMessage `json:"transaction_params"`
	TransactionDetail json.RawMessage `json:"transaction_detail"`
	UseBundle         *bool           `json:"use_bundle"`
	SimulateResult    *string         `json:"simulate_result"`
	TransactionLog    *string         `json:"transaction_log"`
}

// StrategySignalUpdateRequest represents the request body for updating a strategy signal (all fields optional)
type StrategySignalUpdateRequest struct {
	ProjectID         *uint           `json:"project_id"`
	StrategyID        *uint           `json:"strategy_id"`
	RoleID            *uint           `json:"role_id"`
	StrategyParams    json.RawMessage `json:"strategy_params"`
	StrategyStat      json.RawMessage `json:"strategy_stat"`
	TransactionParams json.RawMessage `json:"transaction_params"`
	TransactionDetail json.RawMessage `json:"transaction_detail"`
	UseBundle         *bool           `json:"use_bundle"`
	SimulateResult    *string         `json:"simulate_result"`
	TransactionLog    *string         `json:"transaction_log"`
}

// ListStrategySignals returns a list of all strategy signals
func ListStrategySignals(c *gin.Context) {
	var signals []models.StrategySignal
	if err := dbconfig.DB.Find(&signals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, signals)
}

// GetStrategySignal returns a specific strategy signal by ID
func GetStrategySignal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var signal models.StrategySignal
	if err := dbconfig.DB.First(&signal, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, signal)
}

// GetStrategySignalsByProjectID returns strategy signals filtered by project_id with pagination
func GetStrategySignalsByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
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
	if err := dbconfig.DB.Model(&models.StrategySignal{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取策略信号数据
	var signals []models.StrategySignal
	if err := dbconfig.DB.Where("project_id = ?", projectID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&signals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"page": page,
		"page_size": pageSize,
		"data": signals,
	})
}

// GetStrategySignalsByStrategyID returns strategy signals filtered by strategy_id
func GetStrategySignalsByStrategyID(c *gin.Context) {
	strategyID, err := strconv.Atoi(c.Param("strategy_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy_id format"})
		return
	}

	var signals []models.StrategySignal
	if err := dbconfig.DB.Where("strategy_id = ?", strategyID).Order("created_at DESC").Find(&signals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, signals)
}

// GetStrategySignalsByRoleID returns strategy signals filtered by role_id
func GetStrategySignalsByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var signals []models.StrategySignal
	if err := dbconfig.DB.Where("role_id = ?", roleID).Order("created_at DESC").Find(&signals).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, signals)
}

// CreateStrategySignal creates a new strategy signal
func CreateStrategySignal(c *gin.Context) {
	var request StrategySignalRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	signal := models.StrategySignal{
		ProjectID:         request.ProjectID,
		StrategyID:        request.StrategyID,
		RoleID:            request.RoleID,
		StrategyParams:    request.StrategyParams,
		StrategyStat:      request.StrategyStat,
		TransactionParams: request.TransactionParams,
		TransactionDetail: request.TransactionDetail,
	}

	// Handle optional fields with default values
	if request.UseBundle != nil {
		signal.UseBundle = *request.UseBundle
	}
	if request.SimulateResult != nil {
		signal.SimulateResult = *request.SimulateResult
	}
	if request.TransactionLog != nil {
		signal.TransactionLog = *request.TransactionLog
	}

	if err := dbconfig.DB.Create(&signal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, signal)
}

// UpdateStrategySignal updates an existing strategy signal
func UpdateStrategySignal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request StrategySignalUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var signal models.StrategySignal
	if err := dbconfig.DB.First(&signal, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Only update fields that are provided in the request
	if request.ProjectID != nil {
		signal.ProjectID = *request.ProjectID
	}
	if request.StrategyID != nil {
		signal.StrategyID = *request.StrategyID
	}
	if request.RoleID != nil {
		signal.RoleID = *request.RoleID
	}
	if request.StrategyParams != nil {
		signal.StrategyParams = request.StrategyParams
	}
	if request.StrategyStat != nil {
		signal.StrategyStat = request.StrategyStat
	}
	if request.TransactionParams != nil {
		signal.TransactionParams = request.TransactionParams
	}
	if request.TransactionDetail != nil {
		signal.TransactionDetail = request.TransactionDetail
	}
	if request.UseBundle != nil {
		signal.UseBundle = *request.UseBundle
	}
	if request.SimulateResult != nil {
		signal.SimulateResult = *request.SimulateResult
	}
	if request.TransactionLog != nil {
		signal.TransactionLog = *request.TransactionLog
	}

	if err := dbconfig.DB.Save(&signal).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, signal)
}

// DeleteStrategySignal deletes a strategy signal
func DeleteStrategySignal(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.StrategySignal{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
} 