package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// ListRpcConfigs returns a list of all RPC configurations
func ListRpcConfigs(c *gin.Context) {
	var configs []models.RpcConfig
	if err := dbconfig.DB.Preload("BlockchainConfig").Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetRpcConfig returns a specific RPC configuration by ID
func GetRpcConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.RpcConfig
	if err := dbconfig.DB.Preload("BlockchainConfig").First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// RpcConfigRequest represents the request body for creating/updating an RPC configuration
type RpcConfigRequest struct {
	Endpoint          string `json:"endpoint" binding:"required"`
	IsActive          bool   `json:"is_active"`
	BlockchainConfigID uint  `json:"blockchain_config_id" binding:"required"`
}

// CreateRpcConfig creates a new RPC configuration
func CreateRpcConfig(c *gin.Context) {
	var request RpcConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify that the blockchain config exists
	var blockchainConfig models.BlockchainConfig
	if err := dbconfig.DB.First(&blockchainConfig, request.BlockchainConfigID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blockchain_config_id"})
		return
	}

	config := models.RpcConfig{
		Endpoint:          request.Endpoint,
		IsActive:          request.IsActive,
		BlockchainConfigID: request.BlockchainConfigID,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload the config with the associated blockchain config
	if err := dbconfig.DB.Preload("BlockchainConfig").First(&config, config.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load created config"})
		return
	}

	c.JSON(http.StatusCreated, config)
}

// UpdateRpcConfig updates an existing RPC configuration
func UpdateRpcConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request RpcConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify that the blockchain config exists
	var blockchainConfig models.BlockchainConfig
	if err := dbconfig.DB.First(&blockchainConfig, request.BlockchainConfigID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blockchain_config_id"})
		return
	}

	var config models.RpcConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	config.Endpoint = request.Endpoint
	config.IsActive = request.IsActive
	config.BlockchainConfigID = request.BlockchainConfigID

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload the config with the associated blockchain config
	if err := dbconfig.DB.Preload("BlockchainConfig").First(&config, config.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load updated config"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// DeleteRpcConfig deletes an RPC configuration
func DeleteRpcConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.RpcConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
} 