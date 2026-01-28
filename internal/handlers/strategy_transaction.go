package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// StrategyTransactionRequest represents the request body for creating/updating a strategy transaction
type StrategyTransactionRequest struct {
	ProjectID       uint      `json:"project_id" binding:"required"`
	StrategyID      uint      `json:"strategy_id" binding:"required"`
	SignalID        uint      `json:"signal_id" binding:"required"`
	Timestamp       time.Time `json:"timestamp" binding:"required"`
	Wallet          string    `json:"wallet" binding:"required"`
	Direction       string    `json:"direction" binding:"required"`
	SendAmount      float64   `json:"send_amount"`
	BalanceReadable float64   `json:"balance_readable"`
	GetAmount       float64   `json:"get_amount"`
	Status          string    `json:"status"`
	Signature       string    `json:"signature"`
}

// StrategyTransactionUpdateRequest represents the request body for updating a strategy transaction (partial update)
type StrategyTransactionUpdateRequest struct {
	ProjectID       *uint      `json:"project_id,omitempty"`
	StrategyID      *uint      `json:"strategy_id,omitempty"`
	SignalID        *uint      `json:"signal_id,omitempty"`
	Timestamp       *time.Time `json:"timestamp,omitempty"`
	Wallet          *string    `json:"wallet,omitempty"`
	Direction       *string    `json:"direction,omitempty"`
	SendAmount      *float64   `json:"send_amount,omitempty"`
	BalanceReadable *float64   `json:"balance_readable,omitempty"`
	GetAmount       *float64   `json:"get_amount,omitempty"`
	Status          *string    `json:"status,omitempty"`
	Signature       *string    `json:"signature,omitempty"`
}

// ListStrategyTransactions returns a list of all strategy transactions
func ListStrategyTransactions(c *gin.Context) {
	// Parse limit parameter, default to 100
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var transactions []models.StrategyTransaction
	if err := dbconfig.DB.Order("created_at DESC").Limit(limit).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// GetStrategyTransaction returns a specific strategy transaction by ID
func GetStrategyTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var transaction models.StrategyTransaction
	if err := dbconfig.DB.First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

// GetStrategyTransactionsByStrategyID returns strategy transactions filtered by strategy_id
func GetStrategyTransactionsByStrategyID(c *gin.Context) {
	strategyID, err := strconv.Atoi(c.Param("strategy_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy_id format"})
		return
	}

	// Parse limit parameter, default to 100
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var transactions []models.StrategyTransaction
	if err := dbconfig.DB.Where("strategy_id = ?", strategyID).Order("created_at DESC").Limit(limit).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// GetStrategyTransactionsBySignalID returns strategy transactions filtered by signal_id
func GetStrategyTransactionsBySignalID(c *gin.Context) {
	signalID, err := strconv.Atoi(c.Param("signal_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid signal_id format"})
		return
	}

	// Parse limit parameter, default to 100
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var transactions []models.StrategyTransaction
	if err := dbconfig.DB.Where("signal_id = ?", signalID).Order("created_at DESC").Limit(limit).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// GetStrategyTransactionsByWallet returns strategy transactions filtered by wallet address
func GetStrategyTransactionsByWallet(c *gin.Context) {
	wallet := c.Param("wallet")
	if wallet == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Wallet address is required"})
		return
	}

	// Parse limit parameter, default to 100
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var transactions []models.StrategyTransaction
	if err := dbconfig.DB.Where("wallet = ?", wallet).Order("created_at DESC").Limit(limit).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// GetStrategyTransactionsByProjectID returns strategy transactions filtered by project_id
func GetStrategyTransactionsByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Parse limit parameter, default to 100
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var transactions []models.StrategyTransaction
	if err := dbconfig.DB.Where("project_id = ?", projectID).Order("created_at DESC").Limit(limit).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// CreateStrategyTransaction creates a new strategy transaction
func CreateStrategyTransaction(c *gin.Context) {
	var request StrategyTransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction := models.StrategyTransaction{
		ProjectID:       request.ProjectID,
		StrategyID:      request.StrategyID,
		SignalID:        request.SignalID,
		Timestamp:       request.Timestamp,
		Wallet:          request.Wallet,
		Direction:       request.Direction,
		SendAmount:      request.SendAmount,
		BalanceReadable: request.BalanceReadable,
		GetAmount:       request.GetAmount,
		Status:          request.Status,
		Signature:       request.Signature,
	}

	if err := dbconfig.DB.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, transaction)
}

// UpdateStrategyTransaction updates an existing strategy transaction
func UpdateStrategyTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request StrategyTransactionUpdateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if at least one field is provided for update
	if request.ProjectID == nil && request.StrategyID == nil && request.SignalID == nil && 
		request.Timestamp == nil && request.Wallet == nil && request.Direction == nil &&
		request.SendAmount == nil && request.BalanceReadable == nil && request.GetAmount == nil &&
		request.Status == nil && request.Signature == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one field must be provided for update"})
		return
	}

	var transaction models.StrategyTransaction
	if err := dbconfig.DB.First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if request.ProjectID != nil {
		transaction.ProjectID = *request.ProjectID
	}
	if request.StrategyID != nil {
		transaction.StrategyID = *request.StrategyID
	}
	if request.SignalID != nil {
		transaction.SignalID = *request.SignalID
	}
	if request.Timestamp != nil {
		transaction.Timestamp = *request.Timestamp
	}
	if request.Wallet != nil {
		transaction.Wallet = *request.Wallet
	}
	if request.Direction != nil {
		transaction.Direction = *request.Direction
	}
	if request.SendAmount != nil {
		transaction.SendAmount = *request.SendAmount
	}
	if request.BalanceReadable != nil {
		transaction.BalanceReadable = *request.BalanceReadable
	}
	if request.GetAmount != nil {
		transaction.GetAmount = *request.GetAmount
	}
	if request.Status != nil {
		transaction.Status = *request.Status
	}
	if request.Signature != nil {
		transaction.Signature = *request.Signature
	}

	if err := dbconfig.DB.Save(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

// DeleteStrategyTransaction deletes a strategy transaction
func DeleteStrategyTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.StrategyTransaction{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
} 