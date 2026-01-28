package models

import (
	"time"
)

// RaydiumLaunchpadPoolConfig represents the raydium_launchpad_pool_config table
type RaydiumLaunchpadPoolConfig struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PoolAddress         string    `json:"pool_address" gorm:"type:varchar(128);uniqueIndex;not null"`
	Platform            string    `json:"platform" gorm:"type:varchar(20);not null"`
	PoolConfigEpoch     uint64    `json:"pool_config_epoch" gorm:"not null"`
	CurveType           uint64    `json:"curve_type" gorm:"default:0"`
	Index               uint64    `json:"index" gorm:"default:0"`
	MigrateFee          float64   `json:"migrate_fee" gorm:"default:0"`
	TradeFeeRate        float64   `json:"trade_fee_rate" gorm:"not null"`
	MaxShareFeeRate     float64   `json:"max_share_fee_rate" gorm:"not null"`
	MinSupplyA          float64   `json:"min_supply_a" gorm:"not null"`
	MaxLockRate         float64   `json:"max_lock_rate" gorm:"not null"`
	MinSellRateA        float64   `json:"min_sell_rate_a" gorm:"not null"`
	MinMigrateRateA     float64   `json:"min_migrate_rate_a" gorm:"not null"`
	MinFundRaisingB     float64   `json:"min_fund_raising_b" gorm:"not null"`
	MintB               string    `json:"mint_b" gorm:"type:varchar(128);not null"`
	ProtocolFeeOwner    string    `json:"protocol_fee_owner" gorm:"type:varchar(128);not null"`
	MigrateFeeOwner     string    `json:"migrate_fee_owner" gorm:"type:varchar(128);not null"`
	MigrateToAmmWallet  string    `json:"migrate_to_amm_wallet" gorm:"type:varchar(128);not null"`
	MigrateToCpmmWallet string    `json:"migrate_to_cpmm_wallet" gorm:"type:varchar(128);not null"`
	ProgramID           string    `json:"program_id" gorm:"type:varchar(128);not null"`
	BaseIsWsol          bool      `json:"base_is_wsol" gorm:"default:true"`
	BaseMint            string    `json:"base_mint" gorm:"type:varchar(128);not null"`
	QuoteMint           string    `json:"quote_mint" gorm:"type:varchar(128);not null"`
	BaseVault           string    `json:"base_vault" gorm:"type:varchar(128);not null"`
	QuoteVault          string    `json:"quote_vault" gorm:"type:varchar(128);not null"`
	ConfigID            string    `json:"config_id" gorm:"type:varchar(128);not null"`
	Creator             string    `json:"creator" gorm:"type:varchar(128);not null"`
	Status              string    `json:"status" gorm:"type:varchar(20);not null"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for RaydiumLaunchpadPoolConfig
func (RaydiumLaunchpadPoolConfig) TableName() string {
	return "raydium_launchpad_pool_config"
}

// RaydiumCpmmPoolConfig represents the raydium_cpmm_pool_config table
type RaydiumCpmmPoolConfig struct {
	ID              uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	Platform        string    `json:"platform" gorm:"type:varchar(20);not null"`
	ProgramID       string    `json:"program_id" gorm:"type:varchar(128);not null"`
	PoolAddress     string    `json:"pool_address" gorm:"type:varchar(128);uniqueIndex;not null"`
	BaseIsWsol      bool      `json:"base_is_wsol" gorm:"default:false"`
	BaseMint        string    `json:"base_mint" gorm:"type:varchar(128);not null"`
	QuoteMint       string    `json:"quote_mint" gorm:"type:varchar(128);not null"`
	BaseVault       string    `json:"base_vault" gorm:"type:varchar(128);not null"`
	QuoteVault      string    `json:"quote_vault" gorm:"type:varchar(128);not null"`
	FeeRate         float64   `json:"fee_rate" gorm:"not null"`
	ConfigID        string    `json:"config_id" gorm:"type:varchar(128);not null"`
	ConfigIndex     uint64    `json:"config_index" gorm:"not null"`
	ProtocolFeeRate float64   `json:"protocol_fee_rate" gorm:"not null"`
	TradeFeeRate    float64   `json:"trade_fee_rate" gorm:"not null"`
	FundFeeRate     float64   `json:"fund_fee_rate" gorm:"not null"`
	CreatePoolFee   float64   `json:"create_pool_fee" gorm:"not null"`
	LpMint          string    `json:"lp_mint" gorm:"type:varchar(128);not null"`
	BurnPercent     float64   `json:"burn_percent" gorm:"not null"`
	Status          string    `json:"status" gorm:"type:varchar(20);not null"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for RaydiumCpmmPoolConfig
func (RaydiumCpmmPoolConfig) TableName() string {
	return "raydium_cpmm_pool_config"
}

// RaydiumPoolRelation represents the relationship between tokens and pools in Raydium
type RaydiumPoolRelation struct {
	ID                        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	MintA                     string    `json:"mint_a" gorm:"type:varchar(128)"`
	MintB                     string    `json:"mint_b" gorm:"type:varchar(128)"`
	LaunchpadPoolID           string    `json:"launchpad_pool_id" gorm:"type:varchar(128)"`
	CpmmPoolID                string    `json:"cpmm_pool_id" gorm:"type:varchar(128)"`
	LaunchpadPoolBaseVault    string    `json:"launchpad_pool_base_vault" gorm:"type:varchar(128);default:''"`
	LaunchpadPoolQuoteVault   string    `json:"launchpad_pool_quote_vault" gorm:"type:varchar(128);default:''"`
	CpmmPoolBaseVault         string    `json:"cpmm_pool_base_vault" gorm:"type:varchar(128);default:''"`
	CpmmPoolQuoteVault        string    `json:"cpmm_pool_quote_vault" gorm:"type:varchar(128);default:''"`
	LaunchpadPoolBaseIsWsol   bool      `json:"launchpad_pool_base_is_wsol" gorm:"default:false"`
	CpmmPoolBaseIsWsol        bool      `json:"cpmm_pool_base_is_wsol" gorm:"default:true"`
	Completed                 bool      `json:"completed" gorm:"default:false"`
	TokenMintSig              string    `json:"token_mint_sig" gorm:"type:varchar(128)"`
	MigrateSig                string    `json:"migrate_sig" gorm:"type:varchar(128)"`
	CreatedAt                 time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for RaydiumPoolRelation
func (RaydiumPoolRelation) TableName() string {
	return "raydiumpool_relation"
} 