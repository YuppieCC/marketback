package handlers

import (
	"net/http"
	"strconv"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	solana_go "github.com/gagliardetto/solana-go"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	solana_pkg "marketcontrol/pkg/solana"
)

// CreatePumpfunAmmPoolConfigRequest represents the request body for creating a pool config
type CreatePumpfunAmmPoolConfigRequest struct {
	PoolAddress          string `json:"pool_address" binding:"required"`
	PoolBump             uint8  `json:"pool_bump" binding:"required"`
	Index                uint16 `json:"index"`
	Creator              string `json:"creator" binding:"required"`
	BaseMint             string `json:"base_mint" binding:"required"`
	QuoteMint            string `json:"quote_mint" binding:"required"`
	LpMint               string `json:"lp_mint" binding:"required"`
	PoolBaseTokenAccount string `json:"pool_base_token_account" binding:"required"`
	PoolQuoteTokenAccount string `json:"pool_quote_token_account" binding:"required"`
	LpSupply             uint64 `json:"lp_supply" binding:"required"`
	CoinCreator          string `json:"coin_creator" binding:"required"`
	Status               string `json:"status"`
}

// AutoCreatePumpfunAmmPoolConfigRequest represents the request body for auto-creating a pool config
type AutoCreatePumpfunAmmPoolConfigRequest struct {
	MintPubkey string `json:"mint_pubkey" binding:"required"`
}

// CreatePumpfunAmmPoolConfigHandler handles the creation of a new pool config
func CreatePumpfunAmmPoolConfig(c *gin.Context) {
	var req CreatePumpfunAmmPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := models.PumpfunAmmPoolConfig{
		PoolAddress:          req.PoolAddress,
		PoolBump:             req.PoolBump,
		Index:                req.Index,
		Creator:              req.Creator,
		BaseMint:             req.BaseMint,
		QuoteMint:            req.QuoteMint,
		LpMint:               req.LpMint,
		PoolBaseTokenAccount: req.PoolBaseTokenAccount,
		PoolQuoteTokenAccount: req.PoolQuoteTokenAccount,
		LpSupply:             req.LpSupply,
		CoinCreator:          req.CoinCreator,
		Status:               req.Status,
	}

	if config.Status == "" {
		config.Status = "active" // Set default status if not provided
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// GetPumpfunAmmPoolConfigHandler handles retrieving a pool config by ID
func GetPumpfunAmmPoolConfig(c *gin.Context) {
	id := c.Param("id")
	var config models.PumpfunAmmPoolConfig

	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// ListPumpfunAmmPoolConfigsHandler handles retrieving all pool configs
func ListPumpfunAmmPoolConfigs(c *gin.Context) {
	var configs []models.PumpfunAmmPoolConfig

	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// ListPumpfunAmmPoolConfigsBySlice returns a paginated list of pumpfun amm pool configs
func ListPumpfunAmmPoolConfigsBySlice(c *gin.Context) {
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
		validFields := []string{"id", "pool_address", "creator", "base_mint", "quote_mint", "lp_mint", "coin_creator", "status", "created_at", "updated_at"}
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
	if err := dbconfig.DB.Model(&models.PumpfunAmmPoolConfig{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated results
	var configs []models.PumpfunAmmPoolConfig
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

// UpdatePumpfunAmmPoolConfigHandler handles updating a pool config
func UpdatePumpfunAmmPoolConfig(c *gin.Context) {
	id := c.Param("id")
	var req CreatePumpfunAmmPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var config models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	config.PoolBump = req.PoolBump
	config.Index = req.Index
	config.Creator = req.Creator
	config.BaseMint = req.BaseMint
	config.QuoteMint = req.QuoteMint
	config.LpMint = req.LpMint
	config.PoolBaseTokenAccount = req.PoolBaseTokenAccount
	config.PoolQuoteTokenAccount = req.PoolQuoteTokenAccount
	config.LpSupply = req.LpSupply
	config.CoinCreator = req.CoinCreator
	if req.Status != "" {
		config.Status = req.Status
	}

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeletePumpfunAmmPoolConfig handles deleting a pool config
func DeletePumpfunAmmPoolConfig(c *gin.Context) {
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// Start a transaction
	tx := dbconfig.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Check if any project is using this pool
	var projectCount int64
	if err := tx.Model(&models.ProjectConfig{}).
		Where("pool_platform = ? AND pool_id = ?", "pumpfun_amm", idInt).
		Count(&projectCount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project dependencies"})
		return
	}

	if projectCount > 0 {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot delete pool: there are projects using this pool",
			"project_count": projectCount,
		})
		return
	}

	// Delete associated stats first
	result := tx.Where("pool_id = ?", idInt).Delete(&models.PumpfunAmmPoolStat{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete associated stats"})
		return
	}
	deletedStatsCount := result.RowsAffected

	// Delete the pool config
	if err := tx.Delete(&models.PumpfunAmmPoolConfig{}, idInt).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pool config and associated stats deleted successfully",
		"deleted_stats_count": deletedStatsCount,
	})
}

// UpdatePumpfunAmmPoolConfigStatus updates the status of a PumpfunAmm pool config
func UpdatePumpfunAmmPoolConfigStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request UpdateStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate status
	if request.Status != "active" && request.Status != "inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be either 'active' or 'inactive'"})
		return
	}

	var config models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update status
	if err := dbconfig.DB.Model(&config).Update("status", request.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// AutoCreatePumpfunAmmPoolConfig automatically creates a PumpfunAmmPoolConfig based on BaseMint
func AutoCreatePumpfunAmmPoolConfig(c *gin.Context) {
	var req AutoCreatePumpfunAmmPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse the BaseMint public key to validate it
	_, err := solana_go.PublicKeyFromBase58(req.MintPubkey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mint public key"})
		return
	}

	// Query TokenConfig to get the Creator (CoinCreator)
	var tokenConfig models.TokenConfig
	if err := dbconfig.DB.Where("mint = ?", req.MintPubkey).First(&tokenConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token configuration not found for the provided mint"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query token configuration"})
		return
	}

	// Validate that Creator field is not empty
	if tokenConfig.Creator == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token configuration has no creator specified"})
		return
	}

	// Generate pool configuration using the solana package function
	creator := "4SA8GUevgpg2EG7X9cLigXZpeafS124pp5gRnGRava4s"
	poolConfigData, err := solana_pkg.GetPumpfunAmmPoolConfigByMint(req.MintPubkey, creator, tokenConfig.Creator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate pool configuration: " + err.Error()})
		return
	}

	// Create the pool config from the generated data
	config := models.PumpfunAmmPoolConfig{
		PoolAddress:           poolConfigData.PoolAddress,
		PoolBump:              poolConfigData.PoolBump,
		Index:                 poolConfigData.Index,
		Creator:               poolConfigData.Creator,
		BaseMint:              poolConfigData.BaseMint,
		QuoteMint:             poolConfigData.QuoteMint,
		LpMint:                poolConfigData.LpMint,
		PoolBaseTokenAccount:  poolConfigData.PoolBaseTokenAccount,
		PoolQuoteTokenAccount: poolConfigData.PoolQuoteTokenAccount,
		LpSupply:              poolConfigData.LpSupply,
		CoinCreator:           poolConfigData.CoinCreator,
		Status:                poolConfigData.Status,
	}

	// Check if a pool with the same PoolAddress already exists
	var existingConfig models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.Where("pool_address = ?", config.PoolAddress).First(&existingConfig).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Pool configuration already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking existing pool"})
		return
	}

	// Save to database
	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
} 