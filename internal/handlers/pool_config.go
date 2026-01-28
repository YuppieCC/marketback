package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// PoolConfigRequest represents the request body for creating/updating a pool config
type PoolConfigRequest struct {
	Platform     string  `json:"platform" binding:"required"`
	PoolAddress  string  `json:"pool_address" binding:"required"`
	BaseIsWSOL   *bool   `json:"base_is_wsol"`
	BaseMintID   uint    `json:"base_mint_id" binding:"required"`
	QuoteMintID  uint    `json:"quote_mint_id" binding:"required"`
	BaseVault    string  `json:"base_vault"`
	QuoteVault   string  `json:"quote_vault"`
	LpMintID     uint    `json:"lp_mint_id"`
	FeeRate      *float64 `json:"fee_rate"`
	Status       string  `json:"status"`
}

// ListPoolConfigs returns a list of all pool configs
func ListPoolConfigs(c *gin.Context) {
	var pools []models.PoolConfig
	if err := dbconfig.DB.Preload("BaseMint").Preload("QuoteMint").Find(&pools).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pools)
}

// GetPoolConfig returns a specific pool config by ID
func GetPoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var pool models.PoolConfig
	if err := dbconfig.DB.Preload("BaseMint").Preload("QuoteMint").First(&pool, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, pool)
}

// CreatePoolConfig creates a new pool config
func CreatePoolConfig(c *gin.Context) {
	var request PoolConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	feeRate := 0.0
	if request.FeeRate != nil {
		feeRate = *request.FeeRate
	}

	pool := models.PoolConfig{
		Platform:    request.Platform,
		PoolAddress: request.PoolAddress,
		BaseMintID:  request.BaseMintID,
		QuoteMintID: request.QuoteMintID,
		BaseVault:   request.BaseVault,
		QuoteVault:  request.QuoteVault,
		LpMintID:    request.LpMintID,
		FeeRate:     feeRate,
		Status:      request.Status,
	}
	if request.BaseIsWSOL != nil {
		pool.BaseIsWSOL = *request.BaseIsWSOL
	}

	if err := dbconfig.DB.Create(&pool).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pool)
}

// UpdatePoolConfig updates an existing pool config
func UpdatePoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request PoolConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pool models.PoolConfig
	if err := dbconfig.DB.First(&pool, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	pool.Platform = request.Platform
	pool.PoolAddress = request.PoolAddress
	pool.BaseMintID = request.BaseMintID
	pool.QuoteMintID = request.QuoteMintID
	pool.BaseVault = request.BaseVault
	pool.QuoteVault = request.QuoteVault
	pool.LpMintID = request.LpMintID
	if request.FeeRate != nil {
		pool.FeeRate = *request.FeeRate
	}
	pool.Status = request.Status
	if request.BaseIsWSOL != nil {
		pool.BaseIsWSOL = *request.BaseIsWSOL
	}

	if err := dbconfig.DB.Save(&pool).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pool)
}

// DeletePoolConfig deletes a pool config
func DeletePoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
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
		Where("pool_platform = ? AND pool_id = ?", "raydium", id).
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
	result := tx.Where("pool_id = ?", id).Delete(&models.PoolStat{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete associated stats"})
		return
	}
	deletedStatsCount := result.RowsAffected

	// Delete the pool config
	if err := tx.Delete(&models.PoolConfig{}, id).Error; err != nil {
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
