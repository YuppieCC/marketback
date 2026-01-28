package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// TransactionsMonitorConfigRequest represents the request body for creating/updating a transactions monitor config
type TransactionsMonitorConfigRequest struct {
	Address        string `json:"address" binding:"required"`
	Enabled        bool   `json:"enabled"`
	LastSlot       uint   `json:"last_slot"`
	StartSlot      uint   `json:"start_slot"`
	LastTimestamp  uint   `json:"last_timestamp"`
	StartTimestamp uint   `json:"start_timestamp"`
	LastSignature  string `json:"last_signature"`
	StartSignature string `json:"start_signature"`
	TxCount        uint   `json:"tx_count"`
	LastExecution  uint   `json:"last_execution"`
	Retry          bool   `json:"retry"`
}

// AddressTransactionRequest represents the request body for creating/updating an address transaction
type AddressTransactionRequest struct {
	Address   string  `json:"address" binding:"required"`
	Signature string  `json:"signature" binding:"required"`
	FeePayer  string  `json:"fee_payer"`
	Fee       float64 `json:"fee"`
	Slot      uint    `json:"slot"`
	Timestamp uint    `json:"timestamp"`
	Type      string  `json:"type"`
	Source    string  `json:"source"`
	Data      []byte  `json:"data"`
}

// AddressBalanceChangeRequest represents the request body for creating/updating an address balance change
type AddressBalanceChangeRequest struct {
	Slot         uint    `json:"slot" binding:"required"`
	Timestamp    uint    `json:"timestamp" binding:"required"`
	Signature    string  `json:"signature" binding:"required"`
	Address      string  `json:"address" binding:"required"`
	Mint         string  `json:"mint" binding:"required"`
	AmountChange float64 `json:"amount_change" binding:"required"`
}

// FilterAddressBalanceChangeRequest represents the request body for filtering address balance changes
type FilterAddressBalanceChangeRequest struct {
	Signature string `json:"signature"`
	Address   string `json:"address"`
	Mint      string `json:"mint"`
}

// PumpfuninternalSwapRequest represents the request body for creating/updating a swap record
type PumpfuninternalSwapRequest struct {
	Slot                  uint    `json:"slot" binding:"required"`
	Timestamp             uint    `json:"timestamp" binding:"required"`
	Signature             string  `json:"signature" binding:"required"`
	Address               string  `json:"address" binding:"required"`
	Mint                  string  `json:"mint" binding:"required"`
	BondingCurvePda       string  `json:"bonding_curve_pda" binding:"required"`
	TraderMintChange      float64 `json:"trader_mint_change"`
	TraderSolChange       float64 `json:"trader_sol_change"`
	PoolMintChange        float64 `json:"pool_mint_change"`
	PoolSolChange         float64 `json:"pool_sol_change"`
	FeeRecipientSolChange float64 `json:"fee_recipient_sol_change"`
	CreatorSolChange      float64 `json:"creator_sol_change"`
}

// PumpfuninternalHolderRequest represents the request body for creating/updating a holder record
type PumpfuninternalHolderRequest struct {
	Address         string  `json:"address" binding:"required"`
	HolderType      string  `json:"holder_type"`
	BondingCurvePda string  `json:"bonding_curve_pda" binding:"required"`
	Mint            string  `json:"mint" binding:"required"`
	LastSlot        uint    `json:"last_slot"`
	StartSlot       uint    `json:"start_slot"`
	LastTimestamp   uint    `json:"last_timestamp"`
	StartTimestamp  uint    `json:"start_timestamp"`
	EndSignature    string  `json:"end_signature"`
	StartSignature  string  `json:"start_signature"`
	MintChange      float64 `json:"mint_change"`
	SolChange       float64 `json:"sol_change"`
	MintVolume      float64 `json:"mint_volume"`
	SolVolume       float64 `json:"sol_volume"`
	TxCount         uint    `json:"tx_count"`
}

// DeleteTransactionsMonitorConfigWithDataRequest represents the request body for deleting a config with related data
type DeleteTransactionsMonitorConfigWithDataRequest struct {
	PoolPlatform string `json:"pool_platform" binding:"required"`
	Address      string `json:"address" binding:"required"`
}

// PumpfunAmmPoolSwapRequest represents the request body for creating/updating a swap record
type PumpfunAmmPoolSwapRequest struct {
	Slot                      uint    `json:"slot" binding:"required"`
	Timestamp                 uint    `json:"timestamp" binding:"required"`
	PoolAddress               string  `json:"pool_address" binding:"required"`
	Signature                 string  `json:"signature" binding:"required"`
	Fee                       float64 `json:"fee"`
	Address                   string  `json:"address" binding:"required"`
	BaseMint                  string  `json:"base_mint" binding:"required"`
	QuoteMint                 string  `json:"quote_mint" binding:"required"`
	TraderBaseChange          float64 `json:"trader_base_change"`
	TraderQuoteChange         float64 `json:"trader_quote_change"`
	TraderSolChange           float64 `json:"trader_sol_change"`
	PoolBaseChange            float64 `json:"pool_base_change"`
	PoolQuoteChange           float64 `json:"pool_quote_change"`
	PoolBaseAccountSolChange  float64 `json:"pool_base_account_sol_change"`
	PoolQuoteAccountSolChange float64 `json:"pool_quote_account_sol_change"`
}

// PumpfunAmmpoolHolderRequest represents the request body for creating/updating a holder record
type PumpfunAmmpoolHolderRequest struct {
	Address           string  `json:"address" binding:"required"`
	HolderType        string  `json:"holder_type"`
	PoolAddress       string  `json:"pool_address" binding:"required"`
	BaseMint          string  `json:"base_mint" binding:"required"`
	QuoteMint         string  `json:"quote_mint" binding:"required"`
	LastSlot          uint    `json:"last_slot"`
	StartSlot         uint    `json:"start_slot"`
	LastTimestamp     uint    `json:"last_timestamp"`
	StartTimestamp    uint    `json:"start_timestamp"`
	EndSignature      string  `json:"end_signature"`
	StartSignature    string  `json:"start_signature"`
	BaseChange        float64 `json:"base_change"`
	QuoteChange       float64 `json:"quote_change"`
	SolChange         float64 `json:"sol_change"`
	TraderBaseVolume  float64 `json:"trader_base_volume"`
	TraderQuoteVolume float64 `json:"trader_quote_volume"`
	TraderSolVolume   float64 `json:"trader_sol_volume"`
	TxCount           uint    `json:"tx_count"`
}

// HolderByProjectIDRequest represents the request body for filtering holders by role type
type HolderByProjectIDRequest struct {
	RoleType   string `json:"role_type" binding:"required,oneof=pool project retail_investors"`
	OrderField string `json:"order_field" binding:"omitempty,oneof=mint_change sol_change base_change quote_change mint_volume sol_volume trader_base_volume trader_quote_volume trader_sol_volume last_timestamp start_timestamp"`
	OrderType  string `json:"order_type" binding:"omitempty,oneof=asc desc"`
}

// RaydiumPoolHolderRequest represents the request body for creating/updating a Raydium pool holder record
type RaydiumPoolHolderRequest struct {
	Address        string  `json:"address" binding:"required"`
	HolderType     string  `json:"holder_type"`
	PoolAddress    string  `json:"pool_address" binding:"required"`
	BaseMint       string  `json:"base_mint" binding:"required"`
	QuoteMint      string  `json:"quote_mint" binding:"required"`
	LastSlot       uint    `json:"last_slot"`
	StartSlot      uint    `json:"start_slot"`
	LastTimestamp  uint    `json:"last_timestamp"`
	StartTimestamp uint    `json:"start_timestamp"`
	EndSignature   string  `json:"end_signature"`
	StartSignature string  `json:"start_signature"`
	BaseChange     float64 `json:"base_change"`
	QuoteChange    float64 `json:"quote_change"`
	SolChange      float64 `json:"sol_change"`
	TxCount        uint    `json:"tx_count"`
}

// RaydiumPoolSwapRequest represents the request body for creating/updating a Raydium pool swap record
type RaydiumPoolSwapRequest struct {
	Slot              uint    `json:"slot" binding:"required"`
	Timestamp         uint    `json:"timestamp" binding:"required"`
	PoolAddress       string  `json:"pool_address" binding:"required"`
	Signature         string  `json:"signature" binding:"required"`
	Fee               float64 `json:"fee"`
	Address           string  `json:"address" binding:"required"`
	BaseMint          string  `json:"base_mint" binding:"required"`
	QuoteMint         string  `json:"quote_mint" binding:"required"`
	TraderBaseChange  float64 `json:"trader_base_change"`
	TraderQuoteChange float64 `json:"trader_quote_change"`
	TraderSolChange   float64 `json:"trader_sol_change"`
	PoolBaseChange    float64 `json:"pool_base_change"`
	PoolQuoteChange   float64 `json:"pool_quote_change"`
}

// MeteoradbcHolderRequest represents the request body for creating/updating a Meteoradbc holder record
type MeteoradbcHolderRequest struct {
	Address        string  `json:"address" binding:"required"`
	HolderType     string  `json:"holder_type"`
	PoolAddress    string  `json:"pool_address" binding:"required"`
	BaseMint       string  `json:"base_mint" binding:"required"`
	QuoteMint      string  `json:"quote_mint" binding:"required"`
	LastSlot       uint    `json:"last_slot"`
	StartSlot      uint    `json:"start_slot"`
	LastTimestamp  uint    `json:"last_timestamp"`
	StartTimestamp uint    `json:"start_timestamp"`
	EndSignature   string  `json:"end_signature"`
	StartSignature string  `json:"start_signature"`
	BaseChange     float64 `json:"base_change"`
	QuoteChange    float64 `json:"quote_change"`
	SolChange      float64 `json:"sol_change"`
	TxCount        uint    `json:"tx_count"`
}

// MeteoradbcSwapRequest represents the request body for creating/updating a Meteoradbc swap record
type MeteoradbcSwapRequest struct {
	Slot              uint    `json:"slot" binding:"required"`
	Timestamp         uint    `json:"timestamp" binding:"required"`
	PoolAddress       string  `json:"pool_address" binding:"required"`
	Signature         string  `json:"signature" binding:"required"`
	Fee               float64 `json:"fee"`
	Address           string  `json:"address" binding:"required"`
	BaseMint          string  `json:"base_mint" binding:"required"`
	QuoteMint         string  `json:"quote_mint" binding:"required"`
	TraderBaseChange  float64 `json:"trader_base_change"`
	TraderQuoteChange float64 `json:"trader_quote_change"`
	TraderSolChange   float64 `json:"trader_sol_change"`
	PoolBaseChange    float64 `json:"pool_base_change"`
	PoolQuoteChange   float64 `json:"pool_quote_change"`
}

// MeteoracpmmHolderRequest represents the request body for creating/updating a Meteoracpmm holder record
type MeteoracpmmHolderRequest struct {
	Address        string  `json:"address" binding:"required"`
	HolderType     string  `json:"holder_type"`
	PoolAddress    string  `json:"pool_address" binding:"required"`
	BaseMint       string  `json:"base_mint" binding:"required"`
	QuoteMint      string  `json:"quote_mint" binding:"required"`
	LastSlot       uint    `json:"last_slot"`
	StartSlot      uint    `json:"start_slot"`
	LastTimestamp  uint    `json:"last_timestamp"`
	StartTimestamp uint    `json:"start_timestamp"`
	EndSignature   string  `json:"end_signature"`
	StartSignature string  `json:"start_signature"`
	BaseChange     float64 `json:"base_change"`
	QuoteChange    float64 `json:"quote_change"`
	SolChange      float64 `json:"sol_change"`
	TxCount        uint    `json:"tx_count"`
}

// MeteoracpmmSwapRequest represents the request body for creating/updating a Meteoracpmm swap record
type MeteoracpmmSwapRequest struct {
	Slot              uint    `json:"slot" binding:"required"`
	Timestamp         uint    `json:"timestamp" binding:"required"`
	PoolAddress       string  `json:"pool_address" binding:"required"`
	Signature         string  `json:"signature" binding:"required"`
	Fee               float64 `json:"fee"`
	Address           string  `json:"address" binding:"required"`
	BaseMint          string  `json:"base_mint" binding:"required"`
	QuoteMint         string  `json:"quote_mint" binding:"required"`
	TraderBaseChange  float64 `json:"trader_base_change"`
	TraderQuoteChange float64 `json:"trader_quote_change"`
	TraderSolChange   float64 `json:"trader_sol_change"`
	PoolBaseChange    float64 `json:"pool_base_change"`
	PoolQuoteChange   float64 `json:"pool_quote_change"`
}

// SwapTransactionRequest represents the request body for creating/updating a swap transaction record
type SwapTransactionRequest struct {
	Signature   string  `json:"signature" binding:"required"`
	Slot        uint    `json:"slot" binding:"required"`
	Timestamp   uint    `json:"timestamp" binding:"required"`
	PayerType   string  `json:"payer_type"`
	Payer       string  `json:"payer"`
	PoolAddress string  `json:"pool_address" binding:"required"`
	BaseMint    string  `json:"base_mint" binding:"required"`
	QuoteMint   string  `json:"quote_mint" binding:"required"`
	BaseChange  float64 `json:"base_change"`
	QuoteChange float64 `json:"quote_change"`
	IsSuccess   bool    `json:"is_success"`
	TxMeta      string  `json:"tx_meta"`
	TxError     string  `json:"tx_error"`
}

// ListTransactionsMonitorConfigs returns a list of all transactions monitor configs
func ListTransactionsMonitorConfigs(c *gin.Context) {
	var configs []models.TransactionsMonitorConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetTransactionsMonitorConfig returns a specific transactions monitor config by ID
func GetTransactionsMonitorConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.TransactionsMonitorConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateTransactionsMonitorConfig creates a new transactions monitor config
func CreateTransactionsMonitorConfig(c *gin.Context) {
	var request TransactionsMonitorConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := models.TransactionsMonitorConfig{
		Address:        request.Address,
		Enabled:        request.Enabled,
		LastSlot:       request.LastSlot,
		StartSlot:      request.StartSlot,
		LastTimestamp:  request.LastTimestamp,
		StartTimestamp: request.StartTimestamp,
		LastSignature:  request.LastSignature,
		StartSignature: request.StartSignature,
		TxCount:        request.TxCount,
		LastExecution:  request.LastExecution,
		Retry:          request.Retry,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, config)
}

// UpdateTransactionsMonitorConfig updates an existing transactions monitor config
func UpdateTransactionsMonitorConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request TransactionsMonitorConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var config models.TransactionsMonitorConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	config.Address = request.Address
	config.Enabled = request.Enabled
	config.LastSlot = request.LastSlot
	config.StartSlot = request.StartSlot
	config.LastTimestamp = request.LastTimestamp
	config.StartTimestamp = request.StartTimestamp
	config.LastSignature = request.LastSignature
	config.StartSignature = request.StartSignature
	config.TxCount = request.TxCount
	config.LastExecution = request.LastExecution
	config.Retry = request.Retry

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// DeleteTransactionsMonitorConfig deletes a transactions monitor config
func DeleteTransactionsMonitorConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.TransactionsMonitorConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// ListAddressTransactions returns a list of all address transactions
func ListAddressTransactions(c *gin.Context) {
	var transactions []models.AddressTransaction
	if err := dbconfig.DB.Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// GetAddressTransaction returns a specific address transaction by ID
func GetAddressTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var transaction models.AddressTransaction
	if err := dbconfig.DB.First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

// CreateAddressTransaction creates a new address transaction
func CreateAddressTransaction(c *gin.Context) {
	var request AddressTransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction := models.AddressTransaction{
		Address:   request.Address,
		Signature: request.Signature,
		FeePayer:  request.FeePayer,
		Fee:       request.Fee,
		Slot:      request.Slot,
		Timestamp: request.Timestamp,
		Type:      request.Type,
		Source:    request.Source,
		Data:      request.Data,
	}

	if err := dbconfig.DB.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, transaction)
}

// UpdateAddressTransaction updates an existing address transaction
func UpdateAddressTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request AddressTransactionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var transaction models.AddressTransaction
	if err := dbconfig.DB.First(&transaction, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	transaction.Address = request.Address
	transaction.Signature = request.Signature
	transaction.FeePayer = request.FeePayer
	transaction.Fee = request.Fee
	transaction.Slot = request.Slot
	transaction.Timestamp = request.Timestamp
	transaction.Type = request.Type
	transaction.Source = request.Source
	transaction.Data = request.Data

	if err := dbconfig.DB.Save(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

// DeleteAddressTransaction deletes an address transaction
func DeleteAddressTransaction(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.AddressTransaction{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// ListAddressBalanceChanges returns a list of all address balance changes
func ListAddressBalanceChanges(c *gin.Context) {
	var changes []models.AddressBalanceChange
	if err := dbconfig.DB.Find(&changes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, changes)
}

// GetAddressBalanceChange returns a specific address balance change by ID
func GetAddressBalanceChange(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var change models.AddressBalanceChange
	if err := dbconfig.DB.First(&change, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, change)
}

// CreateAddressBalanceChange creates a new address balance change
func CreateAddressBalanceChange(c *gin.Context) {
	var request AddressBalanceChangeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	change := models.AddressBalanceChange{
		Slot:         request.Slot,
		Timestamp:    request.Timestamp,
		Signature:    request.Signature,
		Address:      request.Address,
		Mint:         request.Mint,
		AmountChange: request.AmountChange,
	}

	if err := dbconfig.DB.Create(&change).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, change)
}

// UpdateAddressBalanceChange updates an existing address balance change
func UpdateAddressBalanceChange(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request AddressBalanceChangeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var change models.AddressBalanceChange
	if err := dbconfig.DB.First(&change, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	change.Slot = request.Slot
	change.Timestamp = request.Timestamp
	change.Signature = request.Signature
	change.Address = request.Address
	change.Mint = request.Mint
	change.AmountChange = request.AmountChange

	if err := dbconfig.DB.Save(&change).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, change)
}

// DeleteAddressBalanceChange deletes an address balance change
func DeleteAddressBalanceChange(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.AddressBalanceChange{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterListAddressBalanceChanges returns a filtered list of address balance changes
func FilterListAddressBalanceChanges(c *gin.Context) {
	var request FilterAddressBalanceChangeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证至少有一个过滤参数
	if request.Signature == "" && request.Address == "" && request.Mint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one filter parameter (signature, address, or mint) is required"})
		return
	}

	// 构建查询
	query := dbconfig.DB.Model(&models.AddressBalanceChange{})

	if request.Signature != "" {
		query = query.Where("signature = ?", request.Signature)
	}
	if request.Address != "" {
		query = query.Where("address = ?", request.Address)
	}
	if request.Mint != "" {
		query = query.Where("mint = ?", request.Mint)
	}

	var changes []models.AddressBalanceChange
	if err := query.Find(&changes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, changes)
}

// ListPumpfuninternalSwaps returns a list of all swap records
func ListPumpfuninternalSwaps(c *gin.Context) {
	var swaps []models.PumpfuninternalSwap
	if err := dbconfig.DB.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, swaps)
}

// GetPumpfuninternalSwap returns a specific swap record by ID
func GetPumpfuninternalSwap(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var swap models.PumpfuninternalSwap
	if err := dbconfig.DB.First(&swap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, swap)
}

// CreatePumpfuninternalSwap creates a new swap record
func CreatePumpfuninternalSwap(c *gin.Context) {
	var request PumpfuninternalSwapRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap := models.PumpfuninternalSwap{
		Slot:                  request.Slot,
		Timestamp:             request.Timestamp,
		Signature:             request.Signature,
		Address:               request.Address,
		Mint:                  request.Mint,
		BondingCurvePda:       request.BondingCurvePda,
		TraderMintChange:      request.TraderMintChange,
		TraderSolChange:       request.TraderSolChange,
		PoolMintChange:        request.PoolMintChange,
		PoolSolChange:         request.PoolSolChange,
		FeeRecipientSolChange: request.FeeRecipientSolChange,
		CreatorSolChange:      request.CreatorSolChange,
	}

	if err := dbconfig.DB.Create(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, swap)
}

// UpdatePumpfuninternalSwap updates an existing swap record
func UpdatePumpfuninternalSwap(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request PumpfuninternalSwapRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var swap models.PumpfuninternalSwap
	if err := dbconfig.DB.First(&swap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	swap.Slot = request.Slot
	swap.Timestamp = request.Timestamp
	swap.Signature = request.Signature
	swap.Address = request.Address
	swap.Mint = request.Mint
	swap.BondingCurvePda = request.BondingCurvePda
	swap.TraderMintChange = request.TraderMintChange
	swap.TraderSolChange = request.TraderSolChange
	swap.PoolMintChange = request.PoolMintChange
	swap.PoolSolChange = request.PoolSolChange
	swap.FeeRecipientSolChange = request.FeeRecipientSolChange
	swap.CreatorSolChange = request.CreatorSolChange

	if err := dbconfig.DB.Save(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, swap)
}

// DeletePumpfuninternalSwap deletes a swap record
func DeletePumpfuninternalSwap(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.PumpfuninternalSwap{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterPumpfuninternalSwaps returns a filtered list of swap records
func FilterPumpfuninternalSwaps(c *gin.Context) {
	var request struct {
		Signature       string `json:"signature"`
		Address         string `json:"address"`
		Mint            string `json:"mint"`
		BondingCurvePda string `json:"bonding_curve_pda"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.Signature == "" && request.Address == "" && request.Mint == "" && request.BondingCurvePda == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one filter parameter (signature, address, mint, or bonding_curve_pda) is required"})
		return
	}

	query := dbconfig.DB.Model(&models.PumpfuninternalSwap{})

	if request.Signature != "" {
		query = query.Where("signature = ?", request.Signature)
	}
	if request.Address != "" {
		query = query.Where("address = ?", request.Address)
	}
	if request.Mint != "" {
		query = query.Where("mint = ?", request.Mint)
	}
	if request.BondingCurvePda != "" {
		query = query.Where("bonding_curve_pda = ?", request.BondingCurvePda)
	}

	var swaps []models.PumpfuninternalSwap
	if err := query.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swaps)
}

// ListPumpfuninternalHolders returns a list of all holder records
func ListPumpfuninternalHolders(c *gin.Context) {
	var holders []models.PumpfuninternalHolder
	if err := dbconfig.DB.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, holders)
}

// GetPumpfuninternalHolder returns a specific holder record by ID
func GetPumpfuninternalHolder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var holder models.PumpfuninternalHolder
	if err := dbconfig.DB.First(&holder, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, holder)
}

// CreatePumpfuninternalHolder creates a new holder record
func CreatePumpfuninternalHolder(c *gin.Context) {
	var request PumpfuninternalHolderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder := models.PumpfuninternalHolder{
		Address:         request.Address,
		HolderType:      request.HolderType,
		BondingCurvePda: request.BondingCurvePda,
		Mint:            request.Mint,
		LastSlot:        request.LastSlot,
		StartSlot:       request.StartSlot,
		LastTimestamp:   request.LastTimestamp,
		StartTimestamp:  request.StartTimestamp,
		EndSignature:    request.EndSignature,
		StartSignature:  request.StartSignature,
		MintChange:      request.MintChange,
		SolChange:       request.SolChange,
		MintVolume:      request.MintVolume,
		SolVolume:       request.SolVolume,
		TxCount:         request.TxCount,
	}

	if err := dbconfig.DB.Create(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, holder)
}

// UpdatePumpfuninternalHolder updates an existing holder record
func UpdatePumpfuninternalHolder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request PumpfuninternalHolderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var holder models.PumpfuninternalHolder
	if err := dbconfig.DB.First(&holder, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	holder.Address = request.Address
	holder.HolderType = request.HolderType
	holder.BondingCurvePda = request.BondingCurvePda
	holder.Mint = request.Mint
	holder.LastSlot = request.LastSlot
	holder.StartSlot = request.StartSlot
	holder.LastTimestamp = request.LastTimestamp
	holder.StartTimestamp = request.StartTimestamp
	holder.EndSignature = request.EndSignature
	holder.StartSignature = request.StartSignature
	holder.MintChange = request.MintChange
	holder.SolChange = request.SolChange
	holder.MintVolume = request.MintVolume
	holder.SolVolume = request.SolVolume
	holder.TxCount = request.TxCount

	if err := dbconfig.DB.Save(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, holder)
}

// DeletePumpfuninternalHolder deletes a holder record
func DeletePumpfuninternalHolder(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.PumpfuninternalHolder{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterPumpfuninternalHolders returns a filtered list of holder records
func FilterPumpfuninternalHolders(c *gin.Context) {
	var request struct {
		Address         string `json:"address"`
		HolderType      string `json:"holder_type"`
		BondingCurvePda string `json:"bonding_curve_pda"`
		Mint            string `json:"mint"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := dbconfig.DB.Model(&models.PumpfuninternalHolder{})

	if request.Address != "" {
		query = query.Where("address = ?", request.Address)
	}
	if request.HolderType != "" {
		query = query.Where("holder_type = ?", request.HolderType)
	}
	if request.BondingCurvePda != "" {
		query = query.Where("bonding_curve_pda = ?", request.BondingCurvePda)
	}
	if request.Mint != "" {
		query = query.Where("mint = ?", request.Mint)
	}

	var holders []models.PumpfuninternalHolder
	if err := query.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holders)
}

// ListPumpfuninternalSwapsByPoolID 根据池子ID获取交换记录
func ListPumpfuninternalSwapsByPoolID(c *gin.Context) {
	// 获取 pool_id 参数
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	// 获取 PumpfuninternalConfig
	var pumpConfig models.PumpfuninternalConfig
	if err := dbconfig.DB.First(&pumpConfig, poolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
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
	if err := dbconfig.DB.Model(&models.PumpfuninternalSwap{}).
		Where("bonding_curve_pda = ?", pumpConfig.BondingCurvePda).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取交换记录
	var swaps []models.PumpfuninternalSwap
	if err := dbconfig.DB.Where("bonding_curve_pda = ?", pumpConfig.BondingCurvePda).
		Order("slot DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      swaps,
	})
}

// GetPumpfuninternalHolderByProjectID 根据项目ID获取持有者信息
func GetPumpfuninternalHolderByProjectID(c *gin.Context) {
	// 获取 project_id 参数
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

	// 解析请求体获取 role_type
	var request HolderByProjectIDRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取 ProjectConfig
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.First(&projectConfig, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 检查 PoolPlatform 是否为 pumpfun_internal
	if projectConfig.PoolPlatform != "pumpfun_internal" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project is not using pumpfun_internal platform"})
		return
	}

	// 获取 TokenConfig 来计算 mint_proportion
	var tokenConfig models.TokenConfig
	if err := dbconfig.DB.First(&tokenConfig, projectConfig.TokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 获取 PumpfuninternalConfig
	var pumpConfig models.PumpfuninternalConfig
	if err := dbconfig.DB.Where("id = ?", projectConfig.PoolID).First(&pumpConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pumpfuninternal config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 构建排序字符串
	orderClause := ""
	if request.OrderField != "" && request.OrderType != "" {
		orderClause = request.OrderField + " " + request.OrderType
	}

	// 辅助函数：为持有者数据添加 mint_proportion
	addMintProportion := func(holders []models.PumpfuninternalHolder) []map[string]interface{} {
		result := make([]map[string]interface{}, len(holders))
		for i, holder := range holders {
			// 将结构体转换为 map
			holderMap := map[string]interface{}{
				"id":                holder.ID,
				"address":           holder.Address,
				"holder_type":       holder.HolderType,
				"bonding_curve_pda": holder.BondingCurvePda,
				"mint":              holder.Mint,
				"last_slot":         holder.LastSlot,
				"start_slot":        holder.StartSlot,
				"last_timestamp":    holder.LastTimestamp,
				"start_timestamp":   holder.StartTimestamp,
				"end_signature":     holder.EndSignature,
				"start_signature":   holder.StartSignature,
				"mint_change":       holder.MintChange,
				"sol_change":        holder.SolChange,
				"mint_volume":       holder.MintVolume,
				"sol_volume":        holder.SolVolume,
				"tx_count":          holder.TxCount,
				"created_at":        holder.CreatedAt,
				"updated_at":        holder.UpdatedAt,
			}

			// 计算 mint_proportion
			mintProportion := 0.0
			if tokenConfig.TotalSupply > 0 {
				mintProportion = holder.MintChange / tokenConfig.TotalSupply
			}
			holderMap["mint_proportion"] = mintProportion

			result[i] = holderMap
		}
		return result
	}

	// 根据 role_type 返回对应的数据
	switch request.RoleType {
	case "pool":
		// 获取池子持有者数据
		query := dbconfig.DB.Model(&models.PumpfuninternalHolder{}).Where("bonding_curve_pda = ? AND holder_type = ?",
			pumpConfig.BondingCurvePda, "pool")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var poolHolders []models.PumpfuninternalHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&poolHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(poolHolders),
		})

	case "project":
		// 获取项目地址持有者数据
		query := dbconfig.DB.Model(&models.PumpfuninternalHolder{}).Where("bonding_curve_pda = ? AND holder_type = ?",
			pumpConfig.BondingCurvePda, "project")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var projectHolders []models.PumpfuninternalHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&projectHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(projectHolders),
		})

	case "retail_investors":
		// 获取散户持有者数据
		query := dbconfig.DB.Model(&models.PumpfuninternalHolder{}).Where("bonding_curve_pda = ? AND holder_type = ?",
			pumpConfig.BondingCurvePda, "retail_investors")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var retailHolders []models.PumpfuninternalHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&retailHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(retailHolders),
		})
	}
}

// DeleteTransactionsMonitorConfigWithData deletes a transactions monitor config and its related data
func DeleteTransactionsMonitorConfigWithData(c *gin.Context) {
	var request DeleteTransactionsMonitorConfigWithDataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 查找 TransactionsMonitorConfig
	var config models.TransactionsMonitorConfig
	if err := dbconfig.DB.Where("address = ?", request.Address).First(&config).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "TransactionsMonitorConfig not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. 查找相关的 AddressTransaction
	var transactions []models.AddressTransaction
	if err := dbconfig.DB.Where("address = ? AND slot BETWEEN ? AND ?",
		config.Address, config.StartSlot, config.LastSlot).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 收集所有交易签名
	signatures := make([]string, len(transactions))
	for i, tx := range transactions {
		signatures[i] = tx.Signature
	}

	// 3. 如果是 pumpfun_internal 平台，处理相关数据
	if request.PoolPlatform == "pumpfun_internal" {
		// 查找相关的 PumpfuninternalConfig
		var pumpConfig models.PumpfuninternalConfig
		if err := dbconfig.DB.Where("associated_bonding_curve = ?", config.Address).First(&pumpConfig).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 如果找不到配置，继续执行但记录日志
			logrus.Printf("PumpfuninternalConfig not found for address: %s", config.Address)
		} else {
			// 删除相关的 PumpfuninternalHolder 数据
			if err := dbconfig.DB.Where("bonding_curve_pda = ?", pumpConfig.BondingCurvePda).Delete(&models.PumpfuninternalHolder{}).Error; err != nil {
				logrus.Printf("Error deleting PumpfuninternalHolder records: %v", err)
			}
		}

		// 删除相关的 PumpfuninternalSwap 数据
		if len(signatures) > 0 {
			if err := dbconfig.DB.Where("signature IN ?", signatures).Delete(&models.PumpfuninternalSwap{}).Error; err != nil {
				logrus.Printf("Error deleting PumpfuninternalSwap records: %v", err)
			}
		}
	} else if request.PoolPlatform == "pumpfun_amm" {
		// 查找相关的 PumpfunAmmPoolConfig
		var pumpConfig models.PumpfunAmmPoolConfig
		if err := dbconfig.DB.Where("pool_address = ?", config.Address).First(&pumpConfig).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 如果找不到配置，继续执行但记录日志
			logrus.Printf("PumpfunAmmPoolConfig not found for address: %s", config.Address)
		} else {
			// 删除相关的 PumpfunAmmpoolHolder 数据
			if err := dbconfig.DB.Where("pool_address = ?", pumpConfig.PoolAddress).Delete(&models.PumpfunAmmpoolHolder{}).Error; err != nil {
				logrus.Printf("Error deleting PumpfunAmmpoolHolder records: %v", err)
			}
		}

		// 删除相关的 PumpfunAmmPoolSwap 数据
		if len(signatures) > 0 {
			if err := dbconfig.DB.Where("signature IN ?", signatures).Delete(&models.PumpfunAmmPoolSwap{}).Error; err != nil {
				logrus.Printf("Error deleting PumpfunAmmPoolSwap records: %v", err)
			}
		}
	} else if request.PoolPlatform == "raydium_launchpad" {
		// 查找相关的 RaydiumLaunchpadPoolConfig
		var raydiumConfig models.RaydiumLaunchpadPoolConfig
		if err := dbconfig.DB.Where("pool_address = ?", config.Address).First(&raydiumConfig).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 如果找不到配置，继续执行但记录日志
			logrus.Printf("RaydiumLaunchpadPoolConfig not found for address: %s", config.Address)
		} else {
			// 删除相关的 RaydiumPoolHolder 数据
			if err := dbconfig.DB.Where("pool_address = ? AND base_mint = ? AND quote_mint = ?",
				raydiumConfig.PoolAddress, raydiumConfig.BaseMint, raydiumConfig.QuoteMint).Delete(&models.RaydiumPoolHolder{}).Error; err != nil {
				logrus.Printf("Error deleting RaydiumPoolHolder records: %v", err)
			}
		}

		// 删除相关的 RaydiumPoolSwap 数据
		if len(signatures) > 0 {
			if err := dbconfig.DB.Where("signature IN ?", signatures).Delete(&models.RaydiumPoolSwap{}).Error; err != nil {
				logrus.Printf("Error deleting RaydiumPoolSwap records: %v", err)
			}
		}
	} else if request.PoolPlatform == "raydium_cpmm" {
		// 查找相关的 RaydiumCpmmPoolConfig
		var raydiumConfig models.RaydiumCpmmPoolConfig
		if err := dbconfig.DB.Where("pool_address = ?", config.Address).First(&raydiumConfig).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 如果找不到配置，继续执行但记录日志
			logrus.Printf("RaydiumCpmmPoolConfig not found for address: %s", config.Address)
		} else {
			// 删除相关的 RaydiumPoolHolder 数据
			if err := dbconfig.DB.Where("pool_address = ? AND base_mint = ? AND quote_mint = ?",
				raydiumConfig.PoolAddress, raydiumConfig.BaseMint, raydiumConfig.QuoteMint).Delete(&models.RaydiumPoolHolder{}).Error; err != nil {
				logrus.Printf("Error deleting RaydiumPoolHolder records: %v", err)
			}
		}

		// 删除相关的 RaydiumPoolSwap 数据
		if len(signatures) > 0 {
			if err := dbconfig.DB.Where("signature IN ?", signatures).Delete(&models.RaydiumPoolSwap{}).Error; err != nil {
				logrus.Printf("Error deleting RaydiumPoolSwap records: %v", err)
			}
		}
	}

	// 4. 删除相关的 AddressBalanceChange 数据
	if len(signatures) > 0 {
		if err := dbconfig.DB.Where("signature IN ?", signatures).Delete(&models.AddressBalanceChange{}).Error; err != nil {
			logrus.Printf("Error deleting AddressBalanceChange records: %v", err)
		}
	}

	// 5. 删除 AddressTransaction 数据
	if len(signatures) > 0 {
		if err := dbconfig.DB.Where("signature IN ?", signatures).Delete(&models.AddressTransaction{}).Error; err != nil {
			logrus.Printf("Error deleting AddressTransaction records: %v", err)
		}
	}

	// 6. 最后删除 TransactionsMonitorConfig
	if err := dbconfig.DB.Delete(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":              "Successfully deleted config and related data",
		"deleted_transactions": len(signatures),
	})
}

// ListPumpfunAmmPoolSwaps returns a list of all swap records
func ListPumpfunAmmPoolSwaps(c *gin.Context) {
	var swaps []models.PumpfunAmmPoolSwap
	if err := dbconfig.DB.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, swaps)
}

// GetPumpfunAmmPoolSwap returns a specific swap record by ID
func GetPumpfunAmmPoolSwap(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var swap models.PumpfunAmmPoolSwap
	if err := dbconfig.DB.First(&swap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, swap)
}

// CreatePumpfunAmmPoolSwap creates a new swap record
func CreatePumpfunAmmPoolSwap(c *gin.Context) {
	var request PumpfunAmmPoolSwapRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap := models.PumpfunAmmPoolSwap{
		Slot:                      request.Slot,
		Timestamp:                 request.Timestamp,
		PoolAddress:               request.PoolAddress,
		Signature:                 request.Signature,
		Fee:                       request.Fee,
		Address:                   request.Address,
		BaseMint:                  request.BaseMint,
		QuoteMint:                 request.QuoteMint,
		TraderBaseChange:          request.TraderBaseChange,
		TraderQuoteChange:         request.TraderQuoteChange,
		TraderSolChange:           request.TraderSolChange,
		PoolBaseChange:            request.PoolBaseChange,
		PoolQuoteChange:           request.PoolQuoteChange,
		PoolBaseAccountSolChange:  request.PoolBaseAccountSolChange,
		PoolQuoteAccountSolChange: request.PoolQuoteAccountSolChange,
	}

	if err := dbconfig.DB.Create(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, swap)
}

// UpdatePumpfunAmmPoolSwap updates an existing swap record
func UpdatePumpfunAmmPoolSwap(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request PumpfunAmmPoolSwapRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var swap models.PumpfunAmmPoolSwap
	if err := dbconfig.DB.First(&swap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	swap.Slot = request.Slot
	swap.Timestamp = request.Timestamp
	swap.PoolAddress = request.PoolAddress
	swap.Signature = request.Signature
	swap.Fee = request.Fee
	swap.Address = request.Address
	swap.BaseMint = request.BaseMint
	swap.QuoteMint = request.QuoteMint
	swap.TraderBaseChange = request.TraderBaseChange
	swap.TraderQuoteChange = request.TraderQuoteChange
	swap.TraderSolChange = request.TraderSolChange
	swap.PoolBaseChange = request.PoolBaseChange
	swap.PoolQuoteChange = request.PoolQuoteChange
	swap.PoolBaseAccountSolChange = request.PoolBaseAccountSolChange
	swap.PoolQuoteAccountSolChange = request.PoolQuoteAccountSolChange

	if err := dbconfig.DB.Save(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, swap)
}

// DeletePumpfunAmmPoolSwap deletes a swap record
func DeletePumpfunAmmPoolSwap(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.PumpfunAmmPoolSwap{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterPumpfunAmmPoolSwaps returns a filtered list of swap records
func FilterPumpfunAmmPoolSwaps(c *gin.Context) {
	var request struct {
		PoolAddress string `json:"pool_address"`
		Signature   string `json:"signature"`
		Address     string `json:"address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if request.PoolAddress == "" && request.Signature == "" && request.Address == "" && request.BaseMint == "" && request.QuoteMint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one filter parameter is required"})
		return
	}

	query := dbconfig.DB.Model(&models.PumpfunAmmPoolSwap{})

	if request.PoolAddress != "" {
		query = query.Where("pool_address = ?", request.PoolAddress)
	}
	if request.Signature != "" {
		query = query.Where("signature = ?", request.Signature)
	}
	if request.Address != "" {
		query = query.Where("address = ?", request.Address)
	}
	if request.BaseMint != "" {
		query = query.Where("base_mint = ?", request.BaseMint)
	}
	if request.QuoteMint != "" {
		query = query.Where("quote_mint = ?", request.QuoteMint)
	}

	var swaps []models.PumpfunAmmPoolSwap
	if err := query.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swaps)
}

// ListPumpfunAmmpoolHolders lists all holders
func ListPumpfunAmmpoolHolders(c *gin.Context) {
	var holders []models.PumpfunAmmpoolHolder
	if err := dbconfig.DB.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, holders)
}

// GetPumpfunAmmpoolHolder gets a specific holder by ID
func GetPumpfunAmmpoolHolder(c *gin.Context) {
	var holder models.PumpfunAmmpoolHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, holder)
}

// CreatePumpfunAmmpoolHolder creates a new holder
func CreatePumpfunAmmpoolHolder(c *gin.Context) {
	var req PumpfunAmmpoolHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder := models.PumpfunAmmpoolHolder{
		Address:           req.Address,
		HolderType:        req.HolderType,
		PoolAddress:       req.PoolAddress,
		BaseMint:          req.BaseMint,
		QuoteMint:         req.QuoteMint,
		LastSlot:          req.LastSlot,
		StartSlot:         req.StartSlot,
		LastTimestamp:     req.LastTimestamp,
		StartTimestamp:    req.StartTimestamp,
		EndSignature:      req.EndSignature,
		StartSignature:    req.StartSignature,
		BaseChange:        req.BaseChange,
		QuoteChange:       req.QuoteChange,
		SolChange:         req.SolChange,
		TraderBaseVolume:  req.TraderBaseVolume,
		TraderQuoteVolume: req.TraderQuoteVolume,
		TraderSolVolume:   req.TraderSolVolume,
		TxCount:           req.TxCount,
	}

	if err := dbconfig.DB.Create(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, holder)
}

// UpdatePumpfunAmmpoolHolder updates an existing holder
func UpdatePumpfunAmmpoolHolder(c *gin.Context) {
	var holder models.PumpfunAmmpoolHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req PumpfunAmmpoolHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder.Address = req.Address
	holder.HolderType = req.HolderType
	holder.PoolAddress = req.PoolAddress
	holder.BaseMint = req.BaseMint
	holder.QuoteMint = req.QuoteMint
	holder.LastSlot = req.LastSlot
	holder.StartSlot = req.StartSlot
	holder.LastTimestamp = req.LastTimestamp
	holder.StartTimestamp = req.StartTimestamp
	holder.EndSignature = req.EndSignature
	holder.StartSignature = req.StartSignature
	holder.BaseChange = req.BaseChange
	holder.QuoteChange = req.QuoteChange
	holder.SolChange = req.SolChange
	holder.TraderBaseVolume = req.TraderBaseVolume
	holder.TraderQuoteVolume = req.TraderQuoteVolume
	holder.TraderSolVolume = req.TraderSolVolume
	holder.TxCount = req.TxCount

	if err := dbconfig.DB.Save(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holder)
}

// DeletePumpfunAmmpoolHolder deletes a holder
func DeletePumpfunAmmpoolHolder(c *gin.Context) {
	var holder models.PumpfunAmmpoolHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterPumpfunAmmpoolHolders filters holders based on criteria
func FilterPumpfunAmmpoolHolders(c *gin.Context) {
	var req struct {
		Address     string `json:"address"`
		HolderType  string `json:"holder_type"`
		PoolAddress string `json:"pool_address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := dbconfig.DB.Model(&models.PumpfunAmmpoolHolder{})

	if req.Address != "" {
		query = query.Where("address = ?", req.Address)
	}
	if req.HolderType != "" {
		query = query.Where("holder_type = ?", req.HolderType)
	}
	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}

	var holders []models.PumpfunAmmpoolHolder
	if err := query.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holders)
}

// ListPumpfunAmmPoolSwapsByPoolID 根据池子ID获取交换记录
func ListPumpfunAmmPoolSwapsByPoolID(c *gin.Context) {
	// 获取 pool_id 参数
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	// 获取 PumpfunAmmPoolConfig
	var pumpConfig models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.First(&pumpConfig, poolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
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
	if err := dbconfig.DB.Model(&models.PumpfunAmmPoolSwap{}).
		Where("pool_address = ?", pumpConfig.PoolAddress).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取交换记录
	var swaps []models.PumpfunAmmPoolSwap
	if err := dbconfig.DB.Where("pool_address = ?", pumpConfig.PoolAddress).
		Order("slot DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      swaps,
	})
}

// ListMeteoradbcSwapsByPoolID returns Meteoradbc swaps by pool ID
func ListMeteoradbcSwapsByPoolID(c *gin.Context) {
	// 获取 pool_id 参数
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	// 获取 MeteoradbcConfig
	var meteoradbcConfig models.MeteoradbcConfig
	if err := dbconfig.DB.First(&meteoradbcConfig, poolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
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
	if err := dbconfig.DB.Model(&models.MeteoradbcSwap{}).
		Where("pool_address = ?", meteoradbcConfig.PoolAddress).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取交换记录
	var swaps []models.MeteoradbcSwap
	if err := dbconfig.DB.Where("pool_address = ?", meteoradbcConfig.PoolAddress).
		Order("slot DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      swaps,
	})
}

// GetPumpfunAmmpoolHolderByProjectID returns holders data for a project's AMM pool
func GetPumpfunAmmpoolHolderByProjectID(c *gin.Context) {
	// 获取 project_id 参数
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

	// 解析请求体获取 role_type
	var request HolderByProjectIDRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取 ProjectConfig
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.First(&projectConfig, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 检查 PoolPlatform 是否为 pumpfun_amm
	if projectConfig.PoolPlatform != "pumpfun_amm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project is not using pumpfun_amm platform"})
		return
	}

	// 获取 TokenConfig 来计算 mint_proportion
	var tokenConfig models.TokenConfig
	if err := dbconfig.DB.First(&tokenConfig, projectConfig.TokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 获取 PumpfunAmmPoolConfig
	var pumpConfig models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.Where("id = ?", projectConfig.PoolID).First(&pumpConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "PumpfunAmmPool config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 构建排序字符串
	orderClause := ""
	if request.OrderField != "" && request.OrderType != "" {
		orderClause = request.OrderField + " " + request.OrderType
	}

	// 辅助函数：为持有者数据添加 mint_proportion
	addMintProportion := func(holders []models.PumpfunAmmpoolHolder) []map[string]interface{} {
		result := make([]map[string]interface{}, len(holders))
		for i, holder := range holders {
			// 将结构体转换为 map
			holderMap := map[string]interface{}{
				"id":                  holder.ID,
				"address":             holder.Address,
				"holder_type":         holder.HolderType,
				"pool_address":        holder.PoolAddress,
				"base_mint":           holder.BaseMint,
				"quote_mint":          holder.QuoteMint,
				"last_slot":           holder.LastSlot,
				"start_slot":          holder.StartSlot,
				"last_timestamp":      holder.LastTimestamp,
				"start_timestamp":     holder.StartTimestamp,
				"end_signature":       holder.EndSignature,
				"start_signature":     holder.StartSignature,
				"base_change":         holder.BaseChange,
				"quote_change":        holder.QuoteChange,
				"sol_change":          holder.SolChange,
				"trader_base_volume":  holder.TraderBaseVolume,
				"trader_quote_volume": holder.TraderQuoteVolume,
				"trader_sol_volume":   holder.TraderSolVolume,
				"tx_count":            holder.TxCount,
				"created_at":          holder.CreatedAt,
				"updated_at":          holder.UpdatedAt,
			}

			// 计算 mint_proportion
			mintProportion := 0.0
			if tokenConfig.TotalSupply > 0 {
				mintProportion = holder.BaseChange / tokenConfig.TotalSupply
			}
			holderMap["mint_proportion"] = mintProportion

			result[i] = holderMap
		}
		return result
	}

	// 根据 role_type 返回对应的数据
	switch request.RoleType {
	case "pool":
		// 获取池子持有者数据
		query := dbconfig.DB.Model(&models.PumpfunAmmpoolHolder{}).Where("pool_address = ? AND holder_type = ?",
			pumpConfig.PoolAddress, "pool")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var poolHolders []models.PumpfunAmmpoolHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&poolHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(poolHolders),
		})

	case "project":
		// 获取项目地址持有者数据
		query := dbconfig.DB.Model(&models.PumpfunAmmpoolHolder{}).Where("pool_address = ? AND holder_type = ?",
			pumpConfig.PoolAddress, "project")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var projectHolders []models.PumpfunAmmpoolHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&projectHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(projectHolders),
		})

	case "retail_investors":
		// 获取散户持有者数据
		query := dbconfig.DB.Model(&models.PumpfunAmmpoolHolder{}).Where("pool_address = ? AND holder_type = ?",
			pumpConfig.PoolAddress, "retail_investors")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取散户持有者数据
		var retailHolders []models.PumpfunAmmpoolHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&retailHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(retailHolders),
		})
	}
}

// RaydiumPoolHolder CRUD handlers

// ListRaydiumPoolHolders lists all Raydium pool holders
func ListRaydiumPoolHolders(c *gin.Context) {
	var holders []models.RaydiumPoolHolder
	if err := dbconfig.DB.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, holders)
}

// GetRaydiumPoolHolder gets a specific Raydium pool holder by ID
func GetRaydiumPoolHolder(c *gin.Context) {
	var holder models.RaydiumPoolHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, holder)
}

// CreateRaydiumPoolHolder creates a new Raydium pool holder
func CreateRaydiumPoolHolder(c *gin.Context) {
	var req RaydiumPoolHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder := models.RaydiumPoolHolder{
		Address:        req.Address,
		HolderType:     req.HolderType,
		PoolAddress:    req.PoolAddress,
		BaseMint:       req.BaseMint,
		QuoteMint:      req.QuoteMint,
		LastSlot:       req.LastSlot,
		StartSlot:      req.StartSlot,
		LastTimestamp:  req.LastTimestamp,
		StartTimestamp: req.StartTimestamp,
		EndSignature:   req.EndSignature,
		StartSignature: req.StartSignature,
		BaseChange:     req.BaseChange,
		QuoteChange:    req.QuoteChange,
		SolChange:      req.SolChange,
		TxCount:        req.TxCount,
	}

	if err := dbconfig.DB.Create(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, holder)
}

// UpdateRaydiumPoolHolder updates an existing Raydium pool holder
func UpdateRaydiumPoolHolder(c *gin.Context) {
	var holder models.RaydiumPoolHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req RaydiumPoolHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder.Address = req.Address
	holder.HolderType = req.HolderType
	holder.PoolAddress = req.PoolAddress
	holder.BaseMint = req.BaseMint
	holder.QuoteMint = req.QuoteMint
	holder.LastSlot = req.LastSlot
	holder.StartSlot = req.StartSlot
	holder.LastTimestamp = req.LastTimestamp
	holder.StartTimestamp = req.StartTimestamp
	holder.EndSignature = req.EndSignature
	holder.StartSignature = req.StartSignature
	holder.BaseChange = req.BaseChange
	holder.QuoteChange = req.QuoteChange
	holder.SolChange = req.SolChange
	holder.TxCount = req.TxCount

	if err := dbconfig.DB.Save(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holder)
}

// DeleteRaydiumPoolHolder deletes a Raydium pool holder
func DeleteRaydiumPoolHolder(c *gin.Context) {
	var holder models.RaydiumPoolHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterRaydiumPoolHolders filters Raydium pool holders based on criteria
func FilterRaydiumPoolHolders(c *gin.Context) {
	var req struct {
		Address     string `json:"address"`
		HolderType  string `json:"holder_type"`
		PoolAddress string `json:"pool_address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := dbconfig.DB.Model(&models.RaydiumPoolHolder{})

	if req.Address != "" {
		query = query.Where("address = ?", req.Address)
	}
	if req.HolderType != "" {
		query = query.Where("holder_type = ?", req.HolderType)
	}
	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}

	var holders []models.RaydiumPoolHolder
	if err := query.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holders)
}

// RaydiumPoolSwap CRUD handlers

// ListRaydiumPoolSwaps lists all Raydium pool swaps
func ListRaydiumPoolSwaps(c *gin.Context) {
	var swaps []models.RaydiumPoolSwap
	if err := dbconfig.DB.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, swaps)
}

// GetRaydiumPoolSwap gets a specific Raydium pool swap by ID
func GetRaydiumPoolSwap(c *gin.Context) {
	var swap models.RaydiumPoolSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, swap)
}

// CreateRaydiumPoolSwap creates a new Raydium pool swap
func CreateRaydiumPoolSwap(c *gin.Context) {
	var req RaydiumPoolSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap := models.RaydiumPoolSwap{
		Slot:              req.Slot,
		Timestamp:         req.Timestamp,
		PoolAddress:       req.PoolAddress,
		Signature:         req.Signature,
		Fee:               req.Fee,
		Address:           req.Address,
		BaseMint:          req.BaseMint,
		QuoteMint:         req.QuoteMint,
		TraderBaseChange:  req.TraderBaseChange,
		TraderQuoteChange: req.TraderQuoteChange,
		TraderSolChange:   req.TraderSolChange,
		PoolBaseChange:    req.PoolBaseChange,
		PoolQuoteChange:   req.PoolQuoteChange,
	}

	if err := dbconfig.DB.Create(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, swap)
}

// UpdateRaydiumPoolSwap updates an existing Raydium pool swap
func UpdateRaydiumPoolSwap(c *gin.Context) {
	var swap models.RaydiumPoolSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req RaydiumPoolSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap.Slot = req.Slot
	swap.Timestamp = req.Timestamp
	swap.PoolAddress = req.PoolAddress
	swap.Signature = req.Signature
	swap.Fee = req.Fee
	swap.Address = req.Address
	swap.BaseMint = req.BaseMint
	swap.QuoteMint = req.QuoteMint
	swap.TraderBaseChange = req.TraderBaseChange
	swap.TraderQuoteChange = req.TraderQuoteChange
	swap.TraderSolChange = req.TraderSolChange
	swap.PoolBaseChange = req.PoolBaseChange
	swap.PoolQuoteChange = req.PoolQuoteChange

	if err := dbconfig.DB.Save(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swap)
}

// DeleteRaydiumPoolSwap deletes a Raydium pool swap
func DeleteRaydiumPoolSwap(c *gin.Context) {
	var swap models.RaydiumPoolSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterRaydiumPoolSwaps filters Raydium pool swaps based on criteria
func FilterRaydiumPoolSwaps(c *gin.Context) {
	var req struct {
		PoolAddress string `json:"pool_address"`
		Signature   string `json:"signature"`
		Address     string `json:"address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PoolAddress == "" && req.Signature == "" && req.Address == "" && req.BaseMint == "" && req.QuoteMint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one filter parameter is required"})
		return
	}

	query := dbconfig.DB.Model(&models.RaydiumPoolSwap{})

	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.Signature != "" {
		query = query.Where("signature = ?", req.Signature)
	}
	if req.Address != "" {
		query = query.Where("address = ?", req.Address)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}

	var swaps []models.RaydiumPoolSwap
	if err := query.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swaps)
}

// MeteoradbcHolder CRUD handlers

// ListMeteoradbcHolders lists all Meteoradbc holders
func ListMeteoradbcHolders(c *gin.Context) {
	var holders []models.MeteoradbcHolder
	if err := dbconfig.DB.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, holders)
}

// GetMeteoradbcHolder gets a specific Meteoradbc holder by ID
func GetMeteoradbcHolder(c *gin.Context) {
	var holder models.MeteoradbcHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, holder)
}

// CreateMeteoradbcHolder creates a new Meteoradbc holder
func CreateMeteoradbcHolder(c *gin.Context) {
	var req MeteoradbcHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder := models.MeteoradbcHolder{
		Address:        req.Address,
		HolderType:     req.HolderType,
		PoolAddress:    req.PoolAddress,
		BaseMint:       req.BaseMint,
		QuoteMint:      req.QuoteMint,
		LastSlot:       req.LastSlot,
		StartSlot:      req.StartSlot,
		LastTimestamp:  req.LastTimestamp,
		StartTimestamp: req.StartTimestamp,
		EndSignature:   req.EndSignature,
		StartSignature: req.StartSignature,
		BaseChange:     req.BaseChange,
		QuoteChange:    req.QuoteChange,
		SolChange:      req.SolChange,
		TxCount:        req.TxCount,
	}

	if err := dbconfig.DB.Create(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, holder)
}

// UpdateMeteoradbcHolder updates an existing Meteoradbc holder
func UpdateMeteoradbcHolder(c *gin.Context) {
	var holder models.MeteoradbcHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req MeteoradbcHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder.Address = req.Address
	holder.HolderType = req.HolderType
	holder.PoolAddress = req.PoolAddress
	holder.BaseMint = req.BaseMint
	holder.QuoteMint = req.QuoteMint
	holder.LastSlot = req.LastSlot
	holder.StartSlot = req.StartSlot
	holder.LastTimestamp = req.LastTimestamp
	holder.StartTimestamp = req.StartTimestamp
	holder.EndSignature = req.EndSignature
	holder.StartSignature = req.StartSignature
	holder.BaseChange = req.BaseChange
	holder.QuoteChange = req.QuoteChange
	holder.SolChange = req.SolChange
	holder.TxCount = req.TxCount

	if err := dbconfig.DB.Save(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holder)
}

// DeleteMeteoradbcHolder deletes a Meteoradbc holder
func DeleteMeteoradbcHolder(c *gin.Context) {
	var holder models.MeteoradbcHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterMeteoradbcHolders filters Meteoradbc holders based on criteria
func FilterMeteoradbcHolders(c *gin.Context) {
	var req struct {
		Address     string `json:"address"`
		HolderType  string `json:"holder_type"`
		PoolAddress string `json:"pool_address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := dbconfig.DB.Model(&models.MeteoradbcHolder{})

	if req.Address != "" {
		query = query.Where("address = ?", req.Address)
	}
	if req.HolderType != "" {
		query = query.Where("holder_type = ?", req.HolderType)
	}
	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}

	var holders []models.MeteoradbcHolder
	if err := query.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holders)
}

// GetMeteoradbcHolderByProjectID returns holders data for a project's Meteora DBC pool
func GetMeteoradbcHolderByProjectID(c *gin.Context) {
	// 获取 project_id 参数
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

	// 解析请求体获取 role_type
	var request HolderByProjectIDRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取 ProjectConfig
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.First(&projectConfig, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 检查 PoolPlatform 是否为 meteoradbc
	if projectConfig.PoolPlatform != "meteora_dbc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project is not using meteoradbc platform"})
		return
	}

	// 获取 TokenConfig 来计算 mint_proportion
	var tokenConfig models.TokenConfig
	if err := dbconfig.DB.First(&tokenConfig, projectConfig.TokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 获取 MeteoradbcConfig
	var meteoradbcConfig models.MeteoradbcConfig
	if err := dbconfig.DB.Where("id = ?", projectConfig.PoolID).First(&meteoradbcConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoradbc config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 构建排序字符串
	orderClause := ""
	if request.OrderField != "" && request.OrderType != "" {
		orderClause = request.OrderField + " " + request.OrderType
	}

	// 辅助函数：为持有者数据添加 mint_proportion
	addMintProportion := func(holders []models.MeteoradbcHolder) []map[string]interface{} {
		result := make([]map[string]interface{}, len(holders))
		for i, holder := range holders {
			// 将结构体转换为 map
			holderMap := map[string]interface{}{
				"id":              holder.ID,
				"address":         holder.Address,
				"holder_type":     holder.HolderType,
				"pool_address":    holder.PoolAddress,
				"base_mint":       holder.BaseMint,
				"quote_mint":      holder.QuoteMint,
				"last_slot":       holder.LastSlot,
				"start_slot":      holder.StartSlot,
				"last_timestamp":  holder.LastTimestamp,
				"start_timestamp": holder.StartTimestamp,
				"end_signature":   holder.EndSignature,
				"start_signature": holder.StartSignature,
				"base_change":     holder.BaseChange,
				"quote_change":    holder.QuoteChange,
				"sol_change":      holder.SolChange,
				"tx_count":        holder.TxCount,
				"created_at":      holder.CreatedAt,
				"updated_at":      holder.UpdatedAt,
			}

			// 计算 mint_proportion
			mintProportion := 0.0
			if tokenConfig.TotalSupply > 0 {
				mintProportion = holder.BaseChange / tokenConfig.TotalSupply
			}
			holderMap["mint_proportion"] = mintProportion

			result[i] = holderMap
		}
		return result
	}

	// 根据 role_type 返回对应的数据
	switch request.RoleType {
	case "pool":
		// 获取池子持有者数据
		query := dbconfig.DB.Model(&models.MeteoradbcHolder{}).Where("pool_address = ? AND holder_type = ?",
			meteoradbcConfig.PoolAddress, "pool")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var poolHolders []models.MeteoradbcHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&poolHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(poolHolders),
		})

	case "project":
		// 获取项目地址持有者数据
		query := dbconfig.DB.Model(&models.MeteoradbcHolder{}).Where("pool_address = ? AND holder_type = ?",
			meteoradbcConfig.PoolAddress, "project")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var projectHolders []models.MeteoradbcHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&projectHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(projectHolders),
		})

	case "retail_investors":
		// 获取散户持有者数据
		query := dbconfig.DB.Model(&models.MeteoradbcHolder{}).Where("pool_address = ? AND holder_type = ?",
			meteoradbcConfig.PoolAddress, "retail_investors")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取散户持有者数据
		var retailHolders []models.MeteoradbcHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&retailHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(retailHolders),
		})
	}
}

// MeteoradbcSwap CRUD handlers

// ListMeteoradbcSwaps lists all Meteoradbc swaps
func ListMeteoradbcSwaps(c *gin.Context) {
	var swaps []models.MeteoradbcSwap
	if err := dbconfig.DB.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, swaps)
}

// GetMeteoradbcSwap gets a specific Meteoradbc swap by ID
func GetMeteoradbcSwap(c *gin.Context) {
	var swap models.MeteoradbcSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, swap)
}

// CreateMeteoradbcSwap creates a new Meteoradbc swap
func CreateMeteoradbcSwap(c *gin.Context) {
	var req MeteoradbcSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap := models.MeteoradbcSwap{
		Slot:              req.Slot,
		Timestamp:         req.Timestamp,
		PoolAddress:       req.PoolAddress,
		Signature:         req.Signature,
		Fee:               req.Fee,
		Address:           req.Address,
		BaseMint:          req.BaseMint,
		QuoteMint:         req.QuoteMint,
		TraderBaseChange:  req.TraderBaseChange,
		TraderQuoteChange: req.TraderQuoteChange,
		TraderSolChange:   req.TraderSolChange,
		PoolBaseChange:    req.PoolBaseChange,
		PoolQuoteChange:   req.PoolQuoteChange,
	}

	if err := dbconfig.DB.Create(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, swap)
}

// UpdateMeteoradbcSwap updates an existing Meteoradbc swap
func UpdateMeteoradbcSwap(c *gin.Context) {
	var swap models.MeteoradbcSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req MeteoradbcSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap.Slot = req.Slot
	swap.Timestamp = req.Timestamp
	swap.PoolAddress = req.PoolAddress
	swap.Signature = req.Signature
	swap.Fee = req.Fee
	swap.Address = req.Address
	swap.BaseMint = req.BaseMint
	swap.QuoteMint = req.QuoteMint
	swap.TraderBaseChange = req.TraderBaseChange
	swap.TraderQuoteChange = req.TraderQuoteChange
	swap.TraderSolChange = req.TraderSolChange
	swap.PoolBaseChange = req.PoolBaseChange
	swap.PoolQuoteChange = req.PoolQuoteChange

	if err := dbconfig.DB.Save(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swap)
}

// DeleteMeteoradbcSwap deletes a Meteoradbc swap
func DeleteMeteoradbcSwap(c *gin.Context) {
	var swap models.MeteoradbcSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterMeteoradbcSwaps filters Meteoradbc swaps based on criteria
func FilterMeteoradbcSwaps(c *gin.Context) {
	var req struct {
		PoolAddress string `json:"pool_address"`
		Signature   string `json:"signature"`
		Address     string `json:"address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PoolAddress == "" && req.Signature == "" && req.Address == "" && req.BaseMint == "" && req.QuoteMint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one filter parameter is required"})
		return
	}

	query := dbconfig.DB.Model(&models.MeteoradbcSwap{})

	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.Signature != "" {
		query = query.Where("signature = ?", req.Signature)
	}
	if req.Address != "" {
		query = query.Where("address = ?", req.Address)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}

	var swaps []models.MeteoradbcSwap
	if err := query.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swaps)
}

// ListMeteoracpmmHolders lists all Meteoracpmm holders
func ListMeteoracpmmHolders(c *gin.Context) {
	var holders []models.MeteoracpmmHolder
	if err := dbconfig.DB.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, holders)
}

// GetMeteoracpmmHolder gets a specific Meteoracpmm holder by ID
func GetMeteoracpmmHolder(c *gin.Context) {
	var holder models.MeteoracpmmHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, holder)
}

// CreateMeteoracpmmHolder creates a new Meteoracpmm holder
func CreateMeteoracpmmHolder(c *gin.Context) {
	var req MeteoracpmmHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder := models.MeteoracpmmHolder{
		Address:        req.Address,
		HolderType:     req.HolderType,
		PoolAddress:    req.PoolAddress,
		BaseMint:       req.BaseMint,
		QuoteMint:      req.QuoteMint,
		LastSlot:       req.LastSlot,
		StartSlot:      req.StartSlot,
		LastTimestamp:  req.LastTimestamp,
		StartTimestamp: req.StartTimestamp,
		EndSignature:   req.EndSignature,
		StartSignature: req.StartSignature,
		BaseChange:     req.BaseChange,
		QuoteChange:    req.QuoteChange,
		SolChange:      req.SolChange,
		TxCount:        req.TxCount,
	}

	if err := dbconfig.DB.Create(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, holder)
}

// UpdateMeteoracpmmHolder updates an existing Meteoracpmm holder
func UpdateMeteoracpmmHolder(c *gin.Context) {
	var holder models.MeteoracpmmHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req MeteoracpmmHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holder.Address = req.Address
	holder.HolderType = req.HolderType
	holder.PoolAddress = req.PoolAddress
	holder.BaseMint = req.BaseMint
	holder.QuoteMint = req.QuoteMint
	holder.LastSlot = req.LastSlot
	holder.StartSlot = req.StartSlot
	holder.LastTimestamp = req.LastTimestamp
	holder.StartTimestamp = req.StartTimestamp
	holder.EndSignature = req.EndSignature
	holder.StartSignature = req.StartSignature
	holder.BaseChange = req.BaseChange
	holder.QuoteChange = req.QuoteChange
	holder.SolChange = req.SolChange
	holder.TxCount = req.TxCount

	if err := dbconfig.DB.Save(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holder)
}

// DeleteMeteoracpmmHolder deletes a Meteoracpmm holder
func DeleteMeteoracpmmHolder(c *gin.Context) {
	var holder models.MeteoracpmmHolder
	if err := dbconfig.DB.First(&holder, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&holder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterMeteoracpmmHolders filters Meteoracpmm holders based on criteria
func FilterMeteoracpmmHolders(c *gin.Context) {
	var req struct {
		Address     string `json:"address"`
		HolderType  string `json:"holder_type"`
		PoolAddress string `json:"pool_address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := dbconfig.DB.Model(&models.MeteoracpmmHolder{})

	if req.Address != "" {
		query = query.Where("address = ?", req.Address)
	}
	if req.HolderType != "" {
		query = query.Where("holder_type = ?", req.HolderType)
	}
	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}

	var holders []models.MeteoracpmmHolder
	if err := query.Find(&holders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holders)
}

// GetMeteoracpmmHolderByProjectID returns holders data for a project's Meteora CPMM pool
func GetMeteoracpmmHolderByProjectID(c *gin.Context) {
	// 获取 project_id 参数
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

	// 解析请求体获取 role_type
	var request HolderByProjectIDRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取 ProjectConfig
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.First(&projectConfig, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 检查 PoolPlatform 是否为 meteoracpmm
	if projectConfig.PoolPlatform != "meteora_cpmm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project is not using meteoracpmm platform"})
		return
	}

	// 获取 TokenConfig 来计算 mint_proportion
	var tokenConfig models.TokenConfig
	if err := dbconfig.DB.First(&tokenConfig, projectConfig.TokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 获取 MeteoracpmmConfig
	var meteoracpmmConfig models.MeteoracpmmConfig
	if err := dbconfig.DB.Where("id = ?", projectConfig.PoolID).First(&meteoracpmmConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoracpmm config not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 构建排序字符串
	orderClause := ""
	if request.OrderField != "" && request.OrderType != "" {
		orderClause = request.OrderField + " " + request.OrderType
	}

	// 辅助函数：为持有者数据添加 mint_proportion
	addMintProportion := func(holders []models.MeteoracpmmHolder) []map[string]interface{} {
		result := make([]map[string]interface{}, len(holders))
		for i, holder := range holders {
			// 将结构体转换为 map
			holderMap := map[string]interface{}{
				"id":              holder.ID,
				"address":         holder.Address,
				"holder_type":     holder.HolderType,
				"pool_address":    holder.PoolAddress,
				"base_mint":       holder.BaseMint,
				"quote_mint":      holder.QuoteMint,
				"last_slot":       holder.LastSlot,
				"start_slot":      holder.StartSlot,
				"last_timestamp":  holder.LastTimestamp,
				"start_timestamp": holder.StartTimestamp,
				"end_signature":   holder.EndSignature,
				"start_signature": holder.StartSignature,
				"base_change":     holder.BaseChange,
				"quote_change":    holder.QuoteChange,
				"sol_change":      holder.SolChange,
				"tx_count":        holder.TxCount,
				"created_at":      holder.CreatedAt,
				"updated_at":      holder.UpdatedAt,
			}

			// 计算 mint_proportion
			mintProportion := 0.0
			if tokenConfig.TotalSupply > 0 {
				mintProportion = holder.BaseChange / tokenConfig.TotalSupply
			}
			holderMap["mint_proportion"] = mintProportion

			result[i] = holderMap
		}
		return result
	}

	// 根据 role_type 返回对应的数据
	switch request.RoleType {
	case "pool":
		// 获取池子持有者数据
		query := dbconfig.DB.Model(&models.MeteoracpmmHolder{}).Where("pool_address = ? AND holder_type = ?",
			meteoracpmmConfig.PoolAddress, "pool")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var poolHolders []models.MeteoracpmmHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&poolHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(poolHolders),
		})

	case "project":
		// 获取项目地址持有者数据
		query := dbconfig.DB.Model(&models.MeteoracpmmHolder{}).Where("pool_address = ? AND holder_type = ?",
			meteoracpmmConfig.PoolAddress, "project")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取分页数据
		var projectHolders []models.MeteoracpmmHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&projectHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(projectHolders),
		})

	case "retail_investors":
		// 获取散户持有者数据
		query := dbconfig.DB.Model(&models.MeteoracpmmHolder{}).Where("pool_address = ? AND holder_type = ?",
			meteoracpmmConfig.PoolAddress, "retail_investors")

		if orderClause != "" {
			query = query.Order(orderClause)
		}

		// 查询总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 获取散户持有者数据
		var retailHolders []models.MeteoracpmmHolder
		if err := query.Offset((page - 1) * pageSize).Limit(pageSize).Find(&retailHolders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      addMintProportion(retailHolders),
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_type. Must be one of: pool, project, retail_investors"})
	}
}

// ListMeteoracpmmSwaps lists all Meteoracpmm swaps
func ListMeteoracpmmSwaps(c *gin.Context) {
	var swaps []models.MeteoracpmmSwap
	if err := dbconfig.DB.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, swaps)
}

// GetMeteoracpmmSwap gets a specific Meteoracpmm swap by ID
func GetMeteoracpmmSwap(c *gin.Context) {
	var swap models.MeteoracpmmSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, swap)
}

// CreateMeteoracpmmSwap creates a new Meteoracpmm swap
func CreateMeteoracpmmSwap(c *gin.Context) {
	var req MeteoracpmmSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap := models.MeteoracpmmSwap{
		Slot:              req.Slot,
		Timestamp:         req.Timestamp,
		PoolAddress:       req.PoolAddress,
		Signature:         req.Signature,
		Fee:               req.Fee,
		Address:           req.Address,
		BaseMint:          req.BaseMint,
		QuoteMint:         req.QuoteMint,
		TraderBaseChange:  req.TraderBaseChange,
		TraderQuoteChange: req.TraderQuoteChange,
		TraderSolChange:   req.TraderSolChange,
		PoolBaseChange:    req.PoolBaseChange,
		PoolQuoteChange:   req.PoolQuoteChange,
	}

	if err := dbconfig.DB.Create(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, swap)
}

// UpdateMeteoracpmmSwap updates an existing Meteoracpmm swap
func UpdateMeteoracpmmSwap(c *gin.Context) {
	var swap models.MeteoracpmmSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req MeteoracpmmSwapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	swap.Slot = req.Slot
	swap.Timestamp = req.Timestamp
	swap.PoolAddress = req.PoolAddress
	swap.Signature = req.Signature
	swap.Fee = req.Fee
	swap.Address = req.Address
	swap.BaseMint = req.BaseMint
	swap.QuoteMint = req.QuoteMint
	swap.TraderBaseChange = req.TraderBaseChange
	swap.TraderQuoteChange = req.TraderQuoteChange
	swap.TraderSolChange = req.TraderSolChange
	swap.PoolBaseChange = req.PoolBaseChange
	swap.PoolQuoteChange = req.PoolQuoteChange

	if err := dbconfig.DB.Save(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swap)
}

// DeleteMeteoracpmmSwap deletes a Meteoracpmm swap
func DeleteMeteoracpmmSwap(c *gin.Context) {
	var swap models.MeteoracpmmSwap
	if err := dbconfig.DB.First(&swap, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&swap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterMeteoracpmmSwaps filters Meteoracpmm swaps based on criteria
func FilterMeteoracpmmSwaps(c *gin.Context) {
	var req struct {
		PoolAddress string `json:"pool_address"`
		Signature   string `json:"signature"`
		Address     string `json:"address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PoolAddress == "" && req.Signature == "" && req.Address == "" && req.BaseMint == "" && req.QuoteMint == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one filter parameter is required"})
		return
	}

	query := dbconfig.DB.Model(&models.MeteoracpmmSwap{})

	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.Signature != "" {
		query = query.Where("signature = ?", req.Signature)
	}
	if req.Address != "" {
		query = query.Where("address = ?", req.Address)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}

	var swaps []models.MeteoracpmmSwap
	if err := query.Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, swaps)
}

// ListMeteoracpmmSwapsByPoolID returns Meteoracpmm swaps by pool ID
func ListMeteoracpmmSwapsByPoolID(c *gin.Context) {
	// 获取 pool_id 参数
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	// 获取 MeteoracpmmConfig
	var meteoracpmmConfig models.MeteoracpmmConfig
	if err := dbconfig.DB.First(&meteoracpmmConfig, poolID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
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
	if err := dbconfig.DB.Model(&models.MeteoracpmmSwap{}).
		Where("pool_address = ?", meteoracpmmConfig.PoolAddress).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取交换记录
	var swaps []models.MeteoracpmmSwap
	if err := dbconfig.DB.Where("pool_address = ?", meteoracpmmConfig.PoolAddress).
		Order("slot DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&swaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      swaps,
	})
}

// MigrateHolderByPoolAddress 根据 poolAddress 迁移 MeteoradbcHolder 到 MeteoracpmmHolder
func MigrateHolderByPoolAddress(c *gin.Context) {
	poolAddress := c.Param("poolAddress")
	if poolAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "poolAddress is required"})
		return
	}

	// 查询 MeteoradbcConfig
	var meteoradbcConfig models.MeteoradbcConfig
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).First(&meteoradbcConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "MeteoradbcConfig not found for pool address: " + poolAddress})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 检查 DammV2PoolAddress 是否存在
	if meteoradbcConfig.DammV2PoolAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DammV2PoolAddress is empty, cannot migrate"})
		return
	}

	// 查询所有 MeteoradbcHolder，排除 HolderType 为 "pool" 的数据
	var meteoradbcHolders []models.MeteoradbcHolder
	if err := dbconfig.DB.Where("pool_address = ? AND holder_type != ?", poolAddress, "pool").Find(&meteoradbcHolders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query MeteoradbcHolder: " + err.Error()})
		return
	}

	// 统计迁移结果
	migratedCount := 0
	skippedCount := 0
	errorCount := 0

	// 批量复制数据到 MeteoracpmmHolder
	for _, dbcHolder := range meteoradbcHolders {
		// 检查是否已存在相同的 MeteoracpmmHolder 记录
		var existingCpmmHolder models.MeteoracpmmHolder
		result := dbconfig.DB.Where("address = ? AND pool_address = ? AND base_mint = ? AND quote_mint = ?",
			dbcHolder.Address, meteoradbcConfig.DammV2PoolAddress, dbcHolder.BaseMint, dbcHolder.QuoteMint).First(&existingCpmmHolder)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// 创建新的 MeteoracpmmHolder 记录
				cpmmHolder := models.MeteoracpmmHolder{
					Address:        dbcHolder.Address,
					HolderType:     dbcHolder.HolderType,
					PoolAddress:    meteoradbcConfig.DammV2PoolAddress, // 使用 DammV2PoolAddress
					BaseMint:       dbcHolder.BaseMint,
					QuoteMint:      dbcHolder.QuoteMint,
					LastSlot:       dbcHolder.LastSlot,
					StartSlot:      dbcHolder.StartSlot,
					LastTimestamp:  dbcHolder.LastTimestamp,
					StartTimestamp: dbcHolder.StartTimestamp,
					EndSignature:   dbcHolder.EndSignature,
					StartSignature: dbcHolder.StartSignature,
					BaseChange:     dbcHolder.BaseChange,
					QuoteChange:    dbcHolder.QuoteChange,
					SolChange:      dbcHolder.SolChange,
					TxCount:        dbcHolder.TxCount,
				}
				if err := dbconfig.DB.Create(&cpmmHolder).Error; err != nil {
					logrus.Errorf("Failed to create MeteoracpmmHolder for address %s: %v", dbcHolder.Address, err)
					errorCount++
					continue
				}
				migratedCount++
				logrus.Infof("Migrated MeteoradbcHolder to MeteoracpmmHolder: address=%s, pool_address=%s -> %s",
					dbcHolder.Address, dbcHolder.PoolAddress, meteoradbcConfig.DammV2PoolAddress)
			} else {
				logrus.Errorf("Failed to check existing MeteoracpmmHolder for address %s: %v", dbcHolder.Address, result.Error)
				errorCount++
				continue
			}
		} else {
			// 记录已存在，跳过
			skippedCount++
			logrus.Infof("MeteoracpmmHolder already exists for address %s, pool_address %s, skipping migration",
				dbcHolder.Address, meteoradbcConfig.DammV2PoolAddress)
		}
	}

	// 返回迁移结果
	c.JSON(http.StatusOK, gin.H{
		"message":        "Migration completed",
		"pool_address":   poolAddress,
		"damm_v2_pool":   meteoradbcConfig.DammV2PoolAddress,
		"total_found":    len(meteoradbcHolders),
		"migrated_count": migratedCount,
		"skipped_count":  skippedCount,
		"error_count":    errorCount,
	})
}

// ListSwapTransactions lists all swap transactions
func ListSwapTransactions(c *gin.Context) {
	var transactions []models.SwapTransaction
	if err := dbconfig.DB.Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transactions)
}

// GetSwapTransaction gets a specific swap transaction by ID
func GetSwapTransaction(c *gin.Context) {
	var transaction models.SwapTransaction
	if err := dbconfig.DB.First(&transaction, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, transaction)
}

// CreateSwapTransaction creates a new swap transaction
func CreateSwapTransaction(c *gin.Context) {
	var req SwapTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction := models.SwapTransaction{
		Signature:   req.Signature,
		Slot:        req.Slot,
		Timestamp:   req.Timestamp,
		PayerType:   req.PayerType,
		Payer:       req.Payer,
		PoolAddress: req.PoolAddress,
		BaseMint:    req.BaseMint,
		QuoteMint:   req.QuoteMint,
		BaseChange:  req.BaseChange,
		QuoteChange: req.QuoteChange,
		IsSuccess:   req.IsSuccess,
		TxMeta:      req.TxMeta,
		TxError:     req.TxError,
	}

	if err := dbconfig.DB.Create(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, transaction)
}

// UpdateSwapTransaction updates an existing swap transaction
func UpdateSwapTransaction(c *gin.Context) {
	var transaction models.SwapTransaction
	if err := dbconfig.DB.First(&transaction, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var req SwapTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	transaction.Signature = req.Signature
	transaction.Slot = req.Slot
	transaction.Timestamp = req.Timestamp
	transaction.PayerType = req.PayerType
	transaction.Payer = req.Payer
	transaction.PoolAddress = req.PoolAddress
	transaction.BaseMint = req.BaseMint
	transaction.QuoteMint = req.QuoteMint
	transaction.BaseChange = req.BaseChange
	transaction.QuoteChange = req.QuoteChange
	transaction.IsSuccess = req.IsSuccess
	transaction.TxMeta = req.TxMeta
	transaction.TxError = req.TxError

	if err := dbconfig.DB.Save(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transaction)
}

// DeleteSwapTransaction deletes a swap transaction
func DeleteSwapTransaction(c *gin.Context) {
	var transaction models.SwapTransaction
	if err := dbconfig.DB.First(&transaction, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	if err := dbconfig.DB.Delete(&transaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// FilterSwapTransactions filters swap transactions based on criteria
func FilterSwapTransactions(c *gin.Context) {
	var req struct {
		Signature   string `json:"signature"`
		PoolAddress string `json:"pool_address"`
		BaseMint    string `json:"base_mint"`
		QuoteMint   string `json:"quote_mint"`
		PayerType   string `json:"payer_type"`
		Payer       string `json:"payer"`
		IsSuccess   *bool  `json:"is_success"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Signature == "" && req.PoolAddress == "" && req.BaseMint == "" && req.QuoteMint == "" && req.PayerType == "" && req.Payer == "" && req.IsSuccess == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one filter parameter is required"})
		return
	}

	query := dbconfig.DB.Model(&models.SwapTransaction{})

	if req.Signature != "" {
		query = query.Where("signature = ?", req.Signature)
	}
	if req.PoolAddress != "" {
		query = query.Where("pool_address = ?", req.PoolAddress)
	}
	if req.BaseMint != "" {
		query = query.Where("base_mint = ?", req.BaseMint)
	}
	if req.QuoteMint != "" {
		query = query.Where("quote_mint = ?", req.QuoteMint)
	}
	if req.PayerType != "" {
		query = query.Where("payer_type = ?", req.PayerType)
	}
	if req.Payer != "" {
		query = query.Where("payer = ?", req.Payer)
	}
	if req.IsSuccess != nil {
		query = query.Where("is_success = ?", *req.IsSuccess)
	}

	var transactions []models.SwapTransaction
	if err := query.Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// ListSwapTransactionsByPoolID returns swap transactions by pool address
func ListSwapTransactionsByPoolID(c *gin.Context) {
	poolAddress := c.Param("pool_id")
	if poolAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pool_id is required"})
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
	if err := dbconfig.DB.Model(&models.SwapTransaction{}).
		Where("pool_address = ?", poolAddress).
		Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取交易记录
	var transactions []models.SwapTransaction
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).
		Order("slot DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      transactions,
	})
}

// GetSwapTransactionsByProject returns swap transactions by project ID and calculates RetailSolAmount
func GetSwapTransactionsByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// 1. Get ProjectConfig by project_id
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.First(&projectConfig, projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 2. Get TokenConfig by TokenID to get mint
	var tokenConfig models.TokenConfig
	if err := dbconfig.DB.First(&tokenConfig, projectConfig.TokenID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Token not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 3. Query SwapTransaction: BaseMint = mint AND IsSuccess = true, ordered by Slot DESC
	var transactions []models.SwapTransaction
	if err := dbconfig.DB.Where("base_mint = ? AND is_success = ?", tokenConfig.Mint, true).
		Order("slot DESC").
		Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 4. Calculate RetailSolAmount: SUM(-QuoteChange)
	var retailSolAmount float64
	for _, tx := range transactions {
		retailSolAmount += -tx.QuoteChange
	}

	// 5. Save RetailSolAmount to ProjectConfig
	projectConfig.RetailSolAmount = retailSolAmount
	if err := dbconfig.DB.Save(&projectConfig).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save RetailSolAmount: " + err.Error()})
		return
	}

	// 6. If RetailSolAmount < 0, return 0
	// if retailSolAmount < 0 {
	// 	retailSolAmount = 0
	// }

	// 7. Convert transactions to response format (excluding tx_meta and tx_error)
	type SwapTransactionResponse struct {
		ID              uint      `json:"id"`
		Signature       string    `json:"signature"`
		Slot            uint      `json:"slot"`
		Timestamp       uint      `json:"timestamp"`
		Datetime        string    `json:"datetime"`
		PayerType       string    `json:"payer_type"`
		Payer           string    `json:"payer"`
		PoolAddress     string    `json:"pool_address"`
		BaseMint        string    `json:"base_mint"`
		QuoteMint       string    `json:"quote_mint"`
		BaseChange      float64   `json:"base_change"`
		QuoteChange     float64   `json:"quote_change"`
		PoolBaseChange  float64   `json:"pool_base_change"`
		PoolQuoteChange float64   `json:"pool_quote_change"`
		IsSuccess       bool      `json:"is_success"`
		CreatedAt       time.Time `json:"created_at"`
	}

	transactionResponses := make([]SwapTransactionResponse, len(transactions))
	for i, tx := range transactions {
		// Convert Timestamp (uint, seconds) to datetime string format "2006-01-02 15:04:05"
		datetime := ""
		if tx.Timestamp > 0 {
			t := time.Unix(int64(tx.Timestamp), 0)
			datetime = t.Format("2006-01-02 15:04:05")
		}

		transactionResponses[i] = SwapTransactionResponse{
			ID:              tx.ID,
			Signature:       tx.Signature,
			Slot:            tx.Slot,
			Timestamp:       tx.Timestamp,
			Datetime:        datetime,
			PayerType:       tx.PayerType,
			Payer:           tx.Payer,
			PoolAddress:     tx.PoolAddress,
			BaseMint:        tx.BaseMint,
			QuoteMint:       tx.QuoteMint,
			BaseChange:      tx.BaseChange,
			QuoteChange:     tx.QuoteChange,
			PoolBaseChange:  tx.BaseChange * -1,
			PoolQuoteChange: tx.QuoteChange * -1,
			IsSuccess:       tx.IsSuccess,
			CreatedAt:       tx.CreatedAt,
		}
	}

	// Return result
	c.JSON(http.StatusOK, gin.H{
		"project_id":        projectID,
		"token_mint":        tokenConfig.Mint,
		"retail_sol_amount": retailSolAmount,
		"transaction_count": len(transactions),
		"transactions":      transactionResponses,
	})
}
