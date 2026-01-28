package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

type TokenAccountRequest struct {
	OwnerAddress   string `json:"owner_address" binding:"required"`
	Mint           string `json:"mint" binding:"required"`
	AccountAddress string `json:"account_address" binding:"required"`
	IsClose        bool   `json:"is_close"`
}

// ListTokenAccounts returns a list of all token accounts
func ListTokenAccounts(c *gin.Context) {
	var accounts []models.TokenAccount
	if err := dbconfig.DB.Find(&accounts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

// GetTokenAccount returns a specific token account by ID
func GetTokenAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var account models.TokenAccount
	if err := dbconfig.DB.First(&account, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, account)
}

// CreateTokenAccount creates a new token account
func CreateTokenAccount(c *gin.Context) {
	var request TokenAccountRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account := models.TokenAccount{
		OwnerAddress:   request.OwnerAddress,
		Mint:           request.Mint,
		AccountAddress: request.AccountAddress,
		IsClose:        request.IsClose,
	}
	if err := dbconfig.DB.Create(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, account)
}

// UpdateTokenAccount updates an existing token account
func UpdateTokenAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var request TokenAccountRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var account models.TokenAccount
	if err := dbconfig.DB.First(&account, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	account.OwnerAddress = request.OwnerAddress
	account.Mint = request.Mint
	account.AccountAddress = request.AccountAddress
	account.IsClose = request.IsClose
	if err := dbconfig.DB.Save(&account).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, account)
}

// DeleteTokenAccount deletes a token account
func DeleteTokenAccount(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	if err := dbconfig.DB.Delete(&models.TokenAccount{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
} 