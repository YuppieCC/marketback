package handlers

import (
	"net/http"
	"strconv"
	"os"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	pumpsolana "marketcontrol/pkg/solana"
	"gorm.io/gorm"
)

// PumpfuninternalConfigRequest represents the request body for creating/updating a pumpfuninternal config
type PumpfuninternalConfigRequest struct {
	Platform               string   `json:"platform" binding:"required"`
	Mint                   string   `json:"mint" binding:"required"`
	FeeRecipient           string   `json:"fee_recipient" binding:"required"`
	FeeRate                *float64 `json:"fee_rate"`
	Status                 string   `json:"status" binding:"required"`
}

// UpdateStatusRequest represents the request body for updating config status
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ListPumpfuninternalConfigs returns a list of all pumpfuninternal configs
func ListPumpfuninternalConfigs(c *gin.Context) {
	var configs []models.PumpfuninternalConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// ListPumpfuninternalConfigsBySlice returns a paginated list of pumpfuninternal configs
func ListPumpfuninternalConfigsBySlice(c *gin.Context) {
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
		validFields := []string{"id", "platform", "mint", "fee_rate", "status", "created_at", "updated_at"}
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
	if err := dbconfig.DB.Model(&models.PumpfuninternalConfig{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated results
	var configs []models.PumpfuninternalConfig
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

// GetPumpfuninternalConfig returns a specific pumpfuninternal config by ID
func GetPumpfuninternalConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.PumpfuninternalConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetPumpfuninternalConfigByMint returns a specific pumpfuninternal config by mint
func GetPumpfuninternalConfigByMint(c *gin.Context) {
	mint := c.Param("mint")
	var config models.PumpfuninternalConfig
	if err := dbconfig.DB.Where("mint = ?", mint).First(&config).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreatePumpfuninternalConfig creates a new pumpfuninternal config with on-chain data
func CreatePumpfuninternalConfig(c *gin.Context) {
	var request PumpfuninternalConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create client
	client := rpc.New(solanaRPC)

	// Parse mint and fee recipient addresses
	mintPubkey, err := solana.PublicKeyFromBase58(request.Mint)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mint address"})
		return
	}

	feeRecipientPubkey, err := solana.PublicKeyFromBase58(request.FeeRecipient)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fee recipient address"})
		return
	}

	// Use default fee rate if not provided
	feeRate := 0.01
	if request.FeeRate != nil && *request.FeeRate != 0 {
		feeRate = *request.FeeRate
	}

	// Get on-chain data
	poolStat, err := pumpsolana.GetPumpFunInternalPoolStat(client, mintPubkey, feeRate, feeRecipientPubkey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get on-chain data: " + err.Error()})
		return
	}

	// Create config with on-chain data
	config := models.PumpfuninternalConfig{
		Platform:               request.Platform,
		Mint:                  poolStat.Mint,
		BondingCurvePda:       poolStat.BondingCurvePDA,
		AssociatedBondingCurve: poolStat.AssociatedBondingCurve,
		CreatorVaultPda:       poolStat.CreatorVaultPDA,
		FeeRecipient:          poolStat.FeeRecipient,
		FeeRate:               poolStat.FeeRate,
		Status:                request.Status,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// UpdatePumpfuninternalConfig updates an existing pumpfuninternal config
// func UpdatePumpfuninternalConfig(c *gin.Context) {
// 	id, err := strconv.Atoi(c.Param("id"))
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
// 		return
// 	}

// 	var request PumpfuninternalConfigRequest
// 	if err := c.ShouldBindJSON(&request); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	var config models.PumpfuninternalConfig
// 	if err := dbconfig.DB.First(&config, id).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
// 		return
// 	}

// 	config.Platform = request.Platform
// 	config.Mint = request.Mint
// 	config.BondingCurvePda = request.BondingCurvePda
// 	config.AssociatedBondingCurve = request.AssociatedBondingCurve
// 	config.CreatorVaultPda = request.CreatorVaultPda
// 	config.FeeRecipient = request.FeeRecipient
// 	if request.FeeRate != nil {
// 		config.FeeRate = *request.FeeRate
// 	}
// 	config.Status = request.Status

// 	if err := dbconfig.DB.Save(&config).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, config)
// }

// UpdatePumpfuninternalConfigStatus updates the status of a Pumpfuninternal config
func UpdatePumpfuninternalConfigStatus(c *gin.Context) {
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

	var config models.PumpfuninternalConfig
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

// DeletePumpfuninternalConfig deletes a pumpfuninternal config
func DeletePumpfuninternalConfig(c *gin.Context) {
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
		Where("pool_platform = ? AND pool_id = ?", "pumpfun_internal", id).
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
	result := tx.Where("pumpfuninternal_id = ?", id).Delete(&models.PumpfuninternalStat{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete associated stats"})
		return
	}
	deletedStatsCount := result.RowsAffected

	// Delete the pool config
	if err := tx.Delete(&models.PumpfuninternalConfig{}, id).Error; err != nil {
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