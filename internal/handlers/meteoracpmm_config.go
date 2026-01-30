package handlers

import (
	"net/http"
	"strconv"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"

	"github.com/gin-gonic/gin"
)

// MeteoracpmmConfigRequest represents the request body for creating/updating a meteoracpmm configuration
type MeteoracpmmConfigRequest struct {
	PoolAddress           string `json:"pool_address" binding:"required"`
	DbcPoolAddress        string `json:"dbc_pool_address" binding:"required"`
	Creator               string `json:"creator" binding:"required"`
	BaseMint              string `json:"base_mint" binding:"required"`
	QuoteMint             string `json:"quote_mint" binding:"required"`
	PoolBaseTokenAccount  string `json:"pool_base_token_account" binding:"required"`
	PoolQuoteTokenAccount string `json:"pool_quote_token_account" binding:"required"`
	Status                string `json:"status"`
}

// ListMeteoracpmmConfigs returns a list of all meteoracpmm configurations
func ListMeteoracpmmConfigs(c *gin.Context) {
	var configs []models.MeteoracpmmConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetMeteoracpmmConfig returns a specific meteoracpmm configuration by ID
func GetMeteoracpmmConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetMeteoracpmmConfigByPoolAddress returns a specific meteoracpmm configuration by pool address
func GetMeteoracpmmConfigByPoolAddress(c *gin.Context) {
	poolAddress := c.Param("pool_address")
	if poolAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pool address is required"})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetLatestMeteoracpmmConfigByCreator returns the meteoracpmm configuration with the latest ID for the given creator
func GetLatestMeteoracpmmConfigByCreator(c *gin.Context) {
	creator := c.Param("creator")
	if creator == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Creator is required"})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.Where("creator = ?", creator).Order("id DESC").First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetMeteoracpmmConfigByCreator returns a meteoracpmm configuration by creator
func GetMeteoracpmmConfigByCreator(c *gin.Context) {
	creator := c.Param("creator")
	if creator == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Creator is required"})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.Where("creator = ?", creator).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetMeteoracpmmConfigByMint returns a specific meteoracpmm configuration by base_mint
func GetMeteoracpmmConfigByMint(c *gin.Context) {
	mintAddress := c.Param("mint_address")
	if mintAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mint address is required"})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.Where("base_mint = ?", mintAddress).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateMeteoracpmmConfig creates a new meteoracpmm configuration
func CreateMeteoracpmmConfig(c *gin.Context) {
	var request MeteoracpmmConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Set default status if not provided
	if request.Status == "" {
		request.Status = "active"
	}

	config := models.MeteoracpmmConfig{
		PoolAddress:           request.PoolAddress,
		DbcPoolAddress:        request.DbcPoolAddress,
		Creator:               request.Creator,
		BaseMint:              request.BaseMint,
		QuoteMint:             request.QuoteMint,
		PoolBaseTokenAccount:  request.PoolBaseTokenAccount,
		PoolQuoteTokenAccount: request.PoolQuoteTokenAccount,
		Status:                request.Status,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// UpdateMeteoracpmmConfig updates an existing meteoracpmm configuration
func UpdateMeteoracpmmConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request MeteoracpmmConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Update fields
	config.PoolAddress = request.PoolAddress
	config.DbcPoolAddress = request.DbcPoolAddress
	config.Creator = request.Creator
	config.BaseMint = request.BaseMint
	config.QuoteMint = request.QuoteMint
	config.PoolBaseTokenAccount = request.PoolBaseTokenAccount
	config.PoolQuoteTokenAccount = request.PoolQuoteTokenAccount
	if request.Status != "" {
		config.Status = request.Status
	}

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteMeteoracpmmConfig deletes a meteoracpmm configuration
func DeleteMeteoracpmmConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration deleted successfully"})
}

// UpdateMeteoracpmmConfigStatus updates the status of a meteoracpmm configuration
func UpdateMeteoracpmmConfigStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	var config models.MeteoracpmmConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	config.Status = request.Status
	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ListMeteoracpmmConfigsBySlice returns a paginated list of meteoracpmm configs
func ListMeteoracpmmConfigsBySlice(c *gin.Context) {
	// Parse query parameters with defaults
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize := 10
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	orderField := "id"
	if of := c.Query("order_field"); of != "" {
		// Validate order field to prevent SQL injection
		validFields := []string{"id", "pool_address", "dbc_pool_address", "creator", "base_mint", "quote_mint", "pool_base_token_account", "pool_quote_token_account", "status", "created_at", "updated_at"}
		for _, field := range validFields {
			if of == field {
				orderField = of
				break
			}
		}
	}

	orderType := "desc"
	if ot := c.Query("order_type"); ot == "asc" || ot == "desc" {
		orderType = ot
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count
	var total int64
	if err := dbconfig.DB.Model(&models.MeteoracpmmConfig{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated results
	var configs []models.MeteoracpmmConfig
	if err := dbconfig.DB.Order(orderField + " " + orderType).
		Offset(offset).
		Limit(pageSize).
		Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate pagination info
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)

	response := gin.H{
		"data": configs,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	}

	c.JSON(http.StatusOK, response)
}
