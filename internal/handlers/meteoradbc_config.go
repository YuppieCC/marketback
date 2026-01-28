package handlers

import (
	"net/http"
	"strconv"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"

	"github.com/gin-gonic/gin"
)

// MeteoradbcConfigRequest represents the request body for creating/updating a meteoradbc configuration
type MeteoradbcConfigRequest struct {
	PoolAddress           string `json:"pool_address" binding:"required"`
	Creator               string `json:"creator" binding:"required"`
	PoolConfig            string `json:"pool_config" binding:"required"`
	BaseMint              string `json:"base_mint" binding:"required"`
	QuoteMint             string `json:"quote_mint" binding:"required"`
	PoolBaseTokenAccount  string `json:"pool_base_token_account" binding:"required"`
	PoolQuoteTokenAccount string `json:"pool_quote_token_account" binding:"required"`
	FirstBuyer            string `json:"first_buyer"`
	DammV2PoolAddress     string `json:"damm_v2_pool_address"`
	IsMigrated            bool   `json:"is_migrated"`
	Status                string `json:"status"`
}

// ListMeteoradbcConfigs returns a list of all meteoradbc configurations
func ListMeteoradbcConfigs(c *gin.Context) {
	var configs []models.MeteoradbcConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetMeteoradbcConfig returns a specific meteoradbc configuration by ID
func GetMeteoradbcConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.MeteoradbcConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetMeteoradbcConfigByPoolAddress returns a specific meteoradbc configuration by pool address
func GetMeteoradbcConfigByPoolAddress(c *gin.Context) {
	poolAddress := c.Param("pool_address")
	if poolAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pool address is required"})
		return
	}

	var config models.MeteoradbcConfig
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetMeteoradbcConfigByCreator returns a meteoradbc configuration by creator
func GetMeteoradbcConfigByCreator(c *gin.Context) {
	creator := c.Param("creator")
	if creator == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Creator is required"})
		return
	}

	var config models.MeteoradbcConfig
	if err := dbconfig.DB.Where("creator = ?", creator).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetMeteoradbcConfigByMint returns a specific meteoradbc configuration by base_mint
func GetMeteoradbcConfigByMint(c *gin.Context) {
	mintAddress := c.Param("mint_address")
	if mintAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mint address is required"})
		return
	}

	var config models.MeteoradbcConfig
	if err := dbconfig.DB.Where("base_mint = ?", mintAddress).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateMeteoradbcConfig creates a new meteoradbc configuration
func CreateMeteoradbcConfig(c *gin.Context) {
	var request MeteoradbcConfigRequest
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

	config := models.MeteoradbcConfig{
		PoolAddress:           request.PoolAddress,
		Creator:               request.Creator,
		PoolConfig:            request.PoolConfig,
		BaseMint:              request.BaseMint,
		QuoteMint:             request.QuoteMint,
		PoolBaseTokenAccount:  request.PoolBaseTokenAccount,
		PoolQuoteTokenAccount: request.PoolQuoteTokenAccount,
		FirstBuyer:            request.FirstBuyer,
		DammV2PoolAddress:     request.DammV2PoolAddress,
		IsMigrated:            request.IsMigrated,
		Status:                request.Status,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// UpdateMeteoradbcConfig updates an existing meteoradbc configuration
func UpdateMeteoradbcConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request MeteoradbcConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	var config models.MeteoradbcConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Update fields
	config.PoolAddress = request.PoolAddress
	config.Creator = request.Creator
	config.PoolConfig = request.PoolConfig
	config.BaseMint = request.BaseMint
	config.QuoteMint = request.QuoteMint
	config.PoolBaseTokenAccount = request.PoolBaseTokenAccount
	config.PoolQuoteTokenAccount = request.PoolQuoteTokenAccount
	config.FirstBuyer = request.FirstBuyer
	config.DammV2PoolAddress = request.DammV2PoolAddress
	config.IsMigrated = request.IsMigrated
	if request.Status != "" {
		config.Status = request.Status
	}

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteMeteoradbcConfig deletes a meteoradbc configuration
func DeleteMeteoradbcConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.MeteoradbcConfig
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

// UpdateMeteoradbcConfigStatus updates the status of a meteoradbc configuration
func UpdateMeteoradbcConfigStatus(c *gin.Context) {
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

	var config models.MeteoradbcConfig
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

// ListMeteoradbcConfigsBySlice returns a paginated list of meteoradbc configs
func ListMeteoradbcConfigsBySlice(c *gin.Context) {
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
		validFields := []string{"id", "pool_address", "creator", "pool_config", "base_mint", "quote_mint", "pool_base_token_account", "pool_quote_token_account", "first_buyer", "damm_v2_pool_address", "is_migrated", "status", "created_at", "updated_at"}
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
	if err := dbconfig.DB.Model(&models.MeteoradbcConfig{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated results
	var configs []models.MeteoradbcConfig
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
