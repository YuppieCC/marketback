package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gagliardetto/solana-go"
	"gorm.io/gorm"

	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/internal/models"
	mcsolana "marketcontrol/pkg/solana"
)

// CreateLaunchpadPoolConfigByMintRequest represents the request body for creating a launchpad pool config by mint addresses
type CreateLaunchpadPoolConfigByMintRequest struct {
	RpcEndpoint string `json:"rpc_endpoint" binding:"required"`
	MintA       string `json:"mint_a" binding:"required"`
	MintB       string `json:"mint_b" binding:"required"`
}

// RaydiumLaunchpadPoolConfigRequest represents the request body for creating/updating pool config
type RaydiumLaunchpadPoolConfigRequest struct {
	PoolAddress         string  `json:"pool_address" binding:"required"`
	Platform            string  `json:"platform" binding:"required"`
	PoolConfigEpoch     uint64  `json:"pool_config_epoch"`
	CurveType           uint64  `json:"curve_type"`
	Index               uint64  `json:"index"`
	MigrateFee          float64 `json:"migrate_fee"`
	TradeFeeRate        float64 `json:"trade_fee_rate"`
	MaxShareFeeRate     float64 `json:"max_share_fee_rate"`
	MinSupplyA          float64 `json:"min_supply_a"`
	MaxLockRate         float64 `json:"max_lock_rate"`
	MinSellRateA        float64 `json:"min_sell_rate_a"`
	MinMigrateRateA     float64 `json:"min_migrate_rate_a"`
	MinFundRaisingB     float64 `json:"min_fund_raising_b"`
	MintB               string  `json:"mint_b" binding:"required"`
	ProtocolFeeOwner    string  `json:"protocol_fee_owner" binding:"required"`
	MigrateFeeOwner     string  `json:"migrate_fee_owner" binding:"required"`
	MigrateToAmmWallet  string  `json:"migrate_to_amm_wallet" binding:"required"`
	MigrateToCpmmWallet string  `json:"migrate_to_cpmm_wallet" binding:"required"`
	ProgramID           string  `json:"program_id" binding:"required"`
	BaseIsWsol          bool    `json:"base_is_wsol"`
	BaseMint            string  `json:"base_mint" binding:"required"`
	QuoteMint           string  `json:"quote_mint" binding:"required"`
	BaseVault           string  `json:"base_vault" binding:"required"`
	QuoteVault          string  `json:"quote_vault" binding:"required"`
	ConfigID            string  `json:"config_id" binding:"required"`
	Creator             string  `json:"creator" binding:"required"`
	Status              string  `json:"status" binding:"required"`
}

// ListRaydiumLaunchpadPoolConfigs returns all Raydium launchpad pool configs
func ListRaydiumLaunchpadPoolConfigs(c *gin.Context) {
	var configs []models.RaydiumLaunchpadPoolConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetRaydiumLaunchpadPoolConfig returns a specific pool config by ID
func GetRaydiumLaunchpadPoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.RaydiumLaunchpadPoolConfig
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

// GetRaydiumLaunchpadPoolConfigByPoolAddress returns a pool config by pool address
func GetRaydiumLaunchpadPoolConfigByPoolAddress(c *gin.Context) {
	poolAddress := c.Param("pool_address")

	var config models.RaydiumLaunchpadPoolConfig
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateRaydiumLaunchpadPoolConfig creates a new pool config
func CreateRaydiumLaunchpadPoolConfig(c *gin.Context) {
	var req RaydiumLaunchpadPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := models.RaydiumLaunchpadPoolConfig{
		PoolAddress:         req.PoolAddress,
		Platform:            req.Platform,
		PoolConfigEpoch:     req.PoolConfigEpoch,
		CurveType:           req.CurveType,
		Index:               req.Index,
		MigrateFee:          req.MigrateFee,
		TradeFeeRate:        req.TradeFeeRate,
		MaxShareFeeRate:     req.MaxShareFeeRate,
		MinSupplyA:          req.MinSupplyA,
		MaxLockRate:         req.MaxLockRate,
		MinSellRateA:        req.MinSellRateA,
		MinMigrateRateA:     req.MinMigrateRateA,
		MinFundRaisingB:     req.MinFundRaisingB,
		MintB:               req.MintB,
		ProtocolFeeOwner:    req.ProtocolFeeOwner,
		MigrateFeeOwner:     req.MigrateFeeOwner,
		MigrateToAmmWallet:  req.MigrateToAmmWallet,
		MigrateToCpmmWallet: req.MigrateToCpmmWallet,
		ProgramID:           req.ProgramID,
		BaseIsWsol:          req.BaseIsWsol,
		BaseMint:            req.BaseMint,
		QuoteMint:           req.QuoteMint,
		BaseVault:           req.BaseVault,
		QuoteVault:          req.QuoteVault,
		ConfigID:            req.ConfigID,
		Creator:             req.Creator,
		Status:              req.Status,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, config)
}

// UpdateRaydiumLaunchpadPoolConfig updates an existing pool config
func UpdateRaydiumLaunchpadPoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.RaydiumLaunchpadPoolConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req RaydiumLaunchpadPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	config.PoolAddress = req.PoolAddress
	config.Platform = req.Platform
	config.PoolConfigEpoch = req.PoolConfigEpoch
	config.CurveType = req.CurveType
	config.Index = req.Index
	config.MigrateFee = req.MigrateFee
	config.TradeFeeRate = req.TradeFeeRate
	config.MaxShareFeeRate = req.MaxShareFeeRate
	config.MinSupplyA = req.MinSupplyA
	config.MaxLockRate = req.MaxLockRate
	config.MinSellRateA = req.MinSellRateA
	config.MinMigrateRateA = req.MinMigrateRateA
	config.MinFundRaisingB = req.MinFundRaisingB
	config.MintB = req.MintB
	config.ProtocolFeeOwner = req.ProtocolFeeOwner
	config.MigrateFeeOwner = req.MigrateFeeOwner
	config.MigrateToAmmWallet = req.MigrateToAmmWallet
	config.MigrateToCpmmWallet = req.MigrateToCpmmWallet
	config.ProgramID = req.ProgramID
	config.BaseIsWsol = req.BaseIsWsol
	config.BaseMint = req.BaseMint
	config.QuoteMint = req.QuoteMint
	config.BaseVault = req.BaseVault
	config.QuoteVault = req.QuoteVault
	config.ConfigID = req.ConfigID
	config.Creator = req.Creator
	config.Status = req.Status

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// DeleteRaydiumLaunchpadPoolConfig deletes a pool config
func DeleteRaydiumLaunchpadPoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.RaydiumLaunchpadPoolConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pool config deleted successfully"})
}

// CreateRaydiumLaunchpadPoolConfigByMint creates a launchpad pool config and pool relation by mint addresses
func CreateRaydiumLaunchpadPoolConfigByMint(c *gin.Context) {
	var req CreateLaunchpadPoolConfigByMintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse mint addresses
	mintA, err := solana.PublicKeyFromBase58(req.MintA)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mintA address: " + err.Error()})
		return
	}

	mintB, err := solana.PublicKeyFromBase58(req.MintB)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mintB address: " + err.Error()})
		return
	}

	// Step 1: Get pool IDs using the same logic as GetLaunchpadAndCpmmId
	poolIds, err := mcsolana.GetLaunchpadAndCpmmId(mintA, mintB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pool IDs: " + err.Error()})
		return
	}

	// Get CPMM pool vault addresses
	vaultResult, err := mcsolana.GetCpmmPoolVault(poolIds.CpmmPoolId, mintB, mintA)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get CPMM pool vaults: " + err.Error()})
		return
	}

	// Get CPMM LP mint address
	lpMintResult, err := mcsolana.GetPdaLpMint(mcsolana.CREATE_CPMM_POOL_PROGRAM, poolIds.CpmmPoolId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get CPMM LP mint: " + err.Error()})
		return
	}

	// Get CPMM PDA AMM Config ID
	ammConfigResult, err := mcsolana.GetCpmmPdaAmmConfigId(mcsolana.CREATE_CPMM_POOL_PROGRAM, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get CPMM AMM config ID: " + err.Error()})
		return
	}

	// Step 2: Get launchpad pool info
	poolData, err := mcsolana.GetLaunchpadPoolInfo(req.RpcEndpoint, poolIds.LaunchpadPoolId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get launchpad pool info: " + err.Error()})
		return
	}

	// Step 3: Get launchpad pool config
	configId, err := solana.PublicKeyFromBase58(poolData.ConfigId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config ID from pool data: " + err.Error()})
		return
	}

	configData, err := mcsolana.GetLaunchpadPoolConfig(req.RpcEndpoint, configId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get launchpad pool config: " + err.Error()})
		return
	}

	// Step 4: Create RaydiumLaunchpadPoolConfig based on poolData and configData
	launchpadConfig := models.RaydiumLaunchpadPoolConfig{
		PoolAddress:         poolIds.LaunchpadPoolId.String(),
		Platform:            "raydium_launchpad",
		PoolConfigEpoch:     uint64(configData.Epoch),
		CurveType:           uint64(configData.CurveType),
		Index:               uint64(configData.Index),
		MigrateFee:          float64(configData.MigrateFee),
		TradeFeeRate:        configData.TradeFeeRate,
		MaxShareFeeRate:     configData.MaxShareFeeRate,
		MinSupplyA:          configData.MinSupplyA,
		MaxLockRate:         configData.MaxLockRate,
		MinSellRateA:        configData.MinSellRateA,
		MinMigrateRateA:     configData.MinMigrateRateA,
		MinFundRaisingB:     configData.MinFundRaisingB,
		MintB:               configData.MintB,
		ProtocolFeeOwner:    configData.ProtocolFeeOwner,
		MigrateFeeOwner:     configData.MigrateFeeOwner,
		MigrateToAmmWallet:  configData.MigrateToAmmWallet,
		MigrateToCpmmWallet: configData.MigrateToCpmmWallet,
		ProgramID:           poolData.PlatformId,
		BaseIsWsol:          req.MintA == "So11111111111111111111111111111111111111112", // Check if mintA is WSOL
		BaseMint:            poolData.MintA,
		QuoteMint:           poolData.MintB,
		BaseVault:           poolData.VaultA,
		QuoteVault:          poolData.VaultB,
		ConfigID:            poolData.ConfigId,
		Creator:             poolData.Creator,
		Status:              "active", // Default status
	}

	// Create RaydiumPoolRelation
	poolRelation := models.RaydiumPoolRelation{
		MintA:                   req.MintA,
		MintB:                   req.MintB,
		LaunchpadPoolID:         poolIds.LaunchpadPoolId.String(),
		CpmmPoolID:              poolIds.CpmmPoolId.String(),
		LaunchpadPoolBaseVault:  launchpadConfig.BaseVault,
		LaunchpadPoolQuoteVault: launchpadConfig.QuoteVault,
		CpmmPoolBaseVault:       vaultResult.BaseVault.String(),
		CpmmPoolQuoteVault:      vaultResult.QuoteVault.String(),
	}

	// Start database transaction
	tx := dbconfig.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction: " + tx.Error.Error()})
		return
	}

	// Create RaydiumPoolRelation
	if err := tx.Create(&poolRelation).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pool relation: " + err.Error()})
		return
	}

	// Create RaydiumLaunchpadPoolConfig
	if err := tx.Create(&launchpadConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create launchpad config: " + err.Error()})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	// Return success response with created data
	c.JSON(http.StatusCreated, gin.H{
		"message":          "Launchpad pool config created successfully",
		"pool_relation":    poolRelation,
		"launchpad_config": launchpadConfig,
		"pool_ids": gin.H{
			"cpmmPoolId":         poolIds.CpmmPoolId.String(),
			"launchpadPoolId":    poolIds.LaunchpadPoolId.String(),
			"cpmmBaseVault":      vaultResult.BaseVault.String(),
			"cpmmQuoteVault":     vaultResult.QuoteVault.String(),
			"cpmmLpMint":         lpMintResult.PublicKey.String(),
			"cpmmPdaAmmConfigId": ammConfigResult.PublicKey.String(),
		},
	})
}

// RaydiumCpmmPoolConfigRequest represents the request body for creating/updating a CPMM pool config
type RaydiumCpmmPoolConfigRequest struct {
	Platform        string  `json:"platform" binding:"required"`
	ProgramID       string  `json:"program_id" binding:"required"`
	PoolAddress     string  `json:"pool_address" binding:"required"`
	BaseIsWsol      bool    `json:"base_is_wsol"`
	BaseMint        string  `json:"base_mint" binding:"required"`
	QuoteMint       string  `json:"quote_mint" binding:"required"`
	BaseVault       string  `json:"base_vault" binding:"required"`
	QuoteVault      string  `json:"quote_vault" binding:"required"`
	FeeRate         float64 `json:"fee_rate"`
	ConfigID        string  `json:"config_id" binding:"required"`
	ConfigIndex     uint64  `json:"config_index"`
	ProtocolFeeRate float64 `json:"protocol_fee_rate"`
	TradeFeeRate    float64 `json:"trade_fee_rate"`
	FundFeeRate     float64 `json:"fund_fee_rate"`
	CreatePoolFee   float64  `json:"create_pool_fee"`
	LpMint          string  `json:"lp_mint" binding:"required"`
	BurnPercent     float64 `json:"burn_percent"`
	Status          string  `json:"status" binding:"required"`
}

// RaydiumPoolRelationRequest represents the request body for creating/updating a pool relation
type RaydiumPoolRelationRequest struct {
	MintA                     string `json:"mint_a" binding:"required"`
	MintB                     string `json:"mint_b" binding:"required"`
	LaunchpadPoolID           string `json:"launchpad_pool_id" binding:"required"`
	CpmmPoolID                string `json:"cpmm_pool_id" binding:"required"`
	LaunchpadPoolBaseVault    string `json:"launchpad_pool_base_vault"`
	LaunchpadPoolQuoteVault   string `json:"launchpad_pool_quote_vault"`
	CpmmPoolBaseVault         string `json:"cpmm_pool_base_vault"`
	CpmmPoolQuoteVault        string `json:"cpmm_pool_quote_vault"`
	LaunchpadPoolBaseIsWsol   bool   `json:"launchpad_pool_base_is_wsol"`
	CpmmPoolBaseIsWsol        bool   `json:"cpmm_pool_base_is_wsol"`
	Completed                 bool   `json:"completed"`
	TokenMintSig              string `json:"token_mint_sig"`
	MigrateSig                string `json:"migrate_sig"`
}

// ListRaydiumCpmmPoolConfigs returns all Raydium CPMM pool configs
func ListRaydiumCpmmPoolConfigs(c *gin.Context) {
	var configs []models.RaydiumCpmmPoolConfig
	if err := dbconfig.DB.Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

// GetRaydiumCpmmPoolConfig returns a specific CPMM pool config by ID
func GetRaydiumCpmmPoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.RaydiumCpmmPoolConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "CPMM pool config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// GetRaydiumCpmmPoolConfigByPoolAddress returns a CPMM pool config by pool address
func GetRaydiumCpmmPoolConfigByPoolAddress(c *gin.Context) {
	poolAddress := c.Param("pool_address")

	var config models.RaydiumCpmmPoolConfig
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "CPMM pool config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// CreateRaydiumCpmmPoolConfig creates a new CPMM pool config
func CreateRaydiumCpmmPoolConfig(c *gin.Context) {
	var req RaydiumCpmmPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config := models.RaydiumCpmmPoolConfig{
		Platform:        req.Platform,
		ProgramID:       req.ProgramID,
		PoolAddress:     req.PoolAddress,
		BaseIsWsol:      req.BaseIsWsol,
		BaseMint:        req.BaseMint,
		QuoteMint:       req.QuoteMint,
		BaseVault:       req.BaseVault,
		QuoteVault:      req.QuoteVault,
		FeeRate:         req.FeeRate,
		ConfigID:        req.ConfigID,
		ConfigIndex:     req.ConfigIndex,
		ProtocolFeeRate: req.ProtocolFeeRate,
		TradeFeeRate:    req.TradeFeeRate,
		FundFeeRate:     req.FundFeeRate,
		CreatePoolFee:   req.CreatePoolFee,
		LpMint:          req.LpMint,
		BurnPercent:     req.BurnPercent,
		Status:          req.Status,
	}

	if err := dbconfig.DB.Create(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, config)
}

// UpdateRaydiumCpmmPoolConfig updates an existing CPMM pool config
func UpdateRaydiumCpmmPoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var config models.RaydiumCpmmPoolConfig
	if err := dbconfig.DB.First(&config, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "CPMM pool config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req RaydiumCpmmPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	config.Platform = req.Platform
	config.ProgramID = req.ProgramID
	config.PoolAddress = req.PoolAddress
	config.BaseIsWsol = req.BaseIsWsol
	config.BaseMint = req.BaseMint
	config.QuoteMint = req.QuoteMint
	config.BaseVault = req.BaseVault
	config.QuoteVault = req.QuoteVault
	config.FeeRate = req.FeeRate
	config.ConfigID = req.ConfigID
	config.ConfigIndex = req.ConfigIndex
	config.ProtocolFeeRate = req.ProtocolFeeRate
	config.TradeFeeRate = req.TradeFeeRate
	config.FundFeeRate = req.FundFeeRate
	config.CreatePoolFee = req.CreatePoolFee
	config.LpMint = req.LpMint
	config.BurnPercent = req.BurnPercent
	config.Status = req.Status

	if err := dbconfig.DB.Save(&config).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, config)
}

// DeleteRaydiumCpmmPoolConfig deletes a CPMM pool config
func DeleteRaydiumCpmmPoolConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.RaydiumCpmmPoolConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "CPMM pool config deleted successfully"})
}

// RaydiumPoolRelation handlers

// ListRaydiumPoolRelations returns all Raydium pool relations
func ListRaydiumPoolRelations(c *gin.Context) {
	var relations []models.RaydiumPoolRelation
	if err := dbconfig.DB.Find(&relations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, relations)
}

// GetRaydiumPoolRelation gets a specific Raydium pool relation by ID
func GetRaydiumPoolRelation(c *gin.Context) {
	var relation models.RaydiumPoolRelation
	if err := dbconfig.DB.First(&relation, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool relation not found"})
		return
	}
	c.JSON(http.StatusOK, relation)
}

// GetRaydiumPoolRelationByMints gets a specific Raydium pool relation by mint addresses
func GetRaydiumPoolRelationByMints(c *gin.Context) {
	mintA := c.Param("mint_a")
	mintB := c.Param("mint_b")
	
	var relation models.RaydiumPoolRelation
	if err := dbconfig.DB.Where("mint_a = ? AND mint_b = ?", mintA, mintB).First(&relation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool relation not found"})
		return
	}
	c.JSON(http.StatusOK, relation)
}

// CreateRaydiumPoolRelation creates a new Raydium pool relation
func CreateRaydiumPoolRelation(c *gin.Context) {
	var req RaydiumPoolRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	relation := models.RaydiumPoolRelation{
		MintA:                     req.MintA,
		MintB:                     req.MintB,
		LaunchpadPoolID:           req.LaunchpadPoolID,
		CpmmPoolID:                req.CpmmPoolID,
		LaunchpadPoolBaseVault:    req.LaunchpadPoolBaseVault,
		LaunchpadPoolQuoteVault:   req.LaunchpadPoolQuoteVault,
		CpmmPoolBaseVault:         req.CpmmPoolBaseVault,
		CpmmPoolQuoteVault:        req.CpmmPoolQuoteVault,
		LaunchpadPoolBaseIsWsol:   req.LaunchpadPoolBaseIsWsol,
		CpmmPoolBaseIsWsol:        req.CpmmPoolBaseIsWsol,
		Completed:                 req.Completed,
		TokenMintSig:              req.TokenMintSig,
		MigrateSig:                req.MigrateSig,
	}

	if err := dbconfig.DB.Create(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, relation)
}

// UpdateRaydiumPoolRelation updates an existing Raydium pool relation
func UpdateRaydiumPoolRelation(c *gin.Context) {
	var relation models.RaydiumPoolRelation
	if err := dbconfig.DB.First(&relation, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool relation not found"})
		return
	}

	var req RaydiumPoolRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	relation.MintA = req.MintA
	relation.MintB = req.MintB
	relation.LaunchpadPoolID = req.LaunchpadPoolID
	relation.CpmmPoolID = req.CpmmPoolID
	relation.LaunchpadPoolBaseVault = req.LaunchpadPoolBaseVault
	relation.LaunchpadPoolQuoteVault = req.LaunchpadPoolQuoteVault
	relation.CpmmPoolBaseVault = req.CpmmPoolBaseVault
	relation.CpmmPoolQuoteVault = req.CpmmPoolQuoteVault
	relation.LaunchpadPoolBaseIsWsol = req.LaunchpadPoolBaseIsWsol
	relation.CpmmPoolBaseIsWsol = req.CpmmPoolBaseIsWsol
	relation.Completed = req.Completed
	relation.TokenMintSig = req.TokenMintSig
	relation.MigrateSig = req.MigrateSig

	if err := dbconfig.DB.Save(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, relation)
}

// DeleteRaydiumPoolRelation deletes a Raydium pool relation
func DeleteRaydiumPoolRelation(c *gin.Context) {
	var relation models.RaydiumPoolRelation
	if err := dbconfig.DB.First(&relation, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool relation not found"})
		return
	}

	if err := dbconfig.DB.Delete(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pool relation deleted successfully"})
} 