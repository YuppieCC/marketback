package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// BlockchainConfigRequest represents the request body for creating/updating a blockchain configuration
type BlockchainConfigRequest struct {
	ChainID uint   `json:"chain_id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Network string `json:"network" binding:"required"`
}

// ListBlockchainConfigs returns a list of all blockchain configurations
func ListBlockchainConfigs(c *gin.Context) {
	var configs []models.BlockchainConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetBlockchainConfig returns a specific blockchain configuration by ID
func GetBlockchainConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.BlockchainConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateBlockchainConfig creates a new blockchain configuration
func CreateBlockchainConfig(c *gin.Context) {
	var request BlockchainConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": gin.H{
				"chain_id": "Required field, must be a number",
				"name": "Required field, must be a string",
				"network": "Required field, must be a string",
			},
		})
		return
	}

	// Validate chain_id is not zero
	if request.ChainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id cannot be zero"})
		return
	}

	config := models.BlockchainConfig{
		ChainID: request.ChainID,
		Name:    request.Name,
		Network: request.Network,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, config)
}

// UpdateBlockchainConfig updates an existing blockchain configuration
func UpdateBlockchainConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request BlockchainConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
			"details": gin.H{
				"chain_id": "Required field, must be a number",
				"name": "Required field, must be a string",
				"network": "Required field, must be a string",
			},
		})
		return
	}

	// Validate chain_id is not zero
	if request.ChainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chain_id cannot be zero"})
		return
	}

	var config models.BlockchainConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	config.ChainID = request.ChainID
	config.Name = request.Name
	config.Network = request.Network

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// DeleteBlockchainConfig deletes a blockchain configuration
func DeleteBlockchainConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.BlockchainConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
} 