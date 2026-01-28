package models

import "time"

// TransactionsMonitorConfig represents the configuration for monitoring pool transactions
type TransactionsMonitorConfig struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Address        string    `json:"address" gorm:"type:varchar(100)"`
	Enabled        bool      `json:"enabled"`
	LastSlot       uint      `json:"last_slot"`
	StartSlot      uint      `json:"start_slot"`
	LastTimestamp  uint      `json:"last_timestamp"`
	StartTimestamp uint      `json:"start_timestamp"`
	LastSignature  string    `json:"last_signature" gorm:"type:varchar(100)"`
	StartSignature string    `json:"start_signature" gorm:"type:varchar(100)"`
	TxCount        uint      `json:"tx_count"`
	LastExecution  uint      `json:"last_execution"`
	Retry          bool      `json:"retry" gorm:"default:false"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for TransactionsMonitorConfig
func (TransactionsMonitorConfig) TableName() string {
	return "transactions_monitor_config"
}

// AddressTransaction represents a transaction record for a specific address
type AddressTransaction struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Address   string    `json:"address" gorm:"type:varchar(100)"`
	Signature string    `json:"signature" gorm:"type:varchar(100);uniqueIndex"`
	FeePayer  string    `json:"fee_payer" gorm:"type:varchar(100)"`
	Fee       float64   `json:"fee"`
	Slot      uint      `json:"slot"`
	Timestamp uint      `json:"timestamp"`
	Type      string    `json:"type" gorm:"type:varchar(50)"`
	Source    string    `json:"source" gorm:"type:varchar(50)"`
	Data      []byte    `json:"data" gorm:"type:jsonb"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for AddressTransaction
func (AddressTransaction) TableName() string {
	return "address_transaction"
}

// AddressBalanceChange 地址余额变化记录
type AddressBalanceChange struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Slot         uint      `json:"slot"`
	Timestamp    uint      `json:"timestamp"`
	Signature    string    `json:"signature" gorm:"type:varchar(100)"`
	Address      string    `json:"address" gorm:"type:varchar(100)"`
	Mint         string    `json:"mint" gorm:"type:varchar(100)"`
	AmountChange float64   `json:"amount_change"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName 指定表名
func (AddressBalanceChange) TableName() string {
	return "address_balance_change"
}

// PumpfuninternalSwap represents a swap record in the pumpfuninternal system
type PumpfuninternalSwap struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	Slot                  uint      `json:"slot"`
	Timestamp             uint      `json:"timestamp"`
	Signature             string    `json:"signature" gorm:"type:varchar(100)"`
	Address               string    `json:"address" gorm:"type:varchar(100)"`
	Mint                  string    `json:"mint" gorm:"type:varchar(100)"`
	BondingCurvePda       string    `json:"bonding_curve_pda" gorm:"type:varchar(100)"`
	TraderMintChange      float64   `json:"trader_mint_change"`
	TraderSolChange       float64   `json:"trader_sol_change"`
	PoolMintChange        float64   `json:"pool_mint_change"`
	PoolSolChange         float64   `json:"pool_sol_change"`
	FeeRecipientSolChange float64   `json:"fee_recipient_sol_change"`
	CreatorSolChange      float64   `json:"creator_sol_change"`
	CreatedAt             time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for PumpfuninternalSwap
func (PumpfuninternalSwap) TableName() string {
	return "pumpfuninternal_swap"
}

// PumpfuninternalHolder represents a holder record in the pumpfuninternal system
type PumpfuninternalHolder struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Address         string    `json:"address" gorm:"type:varchar(100)"`
	HolderType      string    `json:"holder_type" gorm:"type:varchar(64)"`
	BondingCurvePda string    `json:"bonding_curve_pda" gorm:"type:varchar(100)"`
	Mint            string    `json:"mint" gorm:"type:varchar(100)"`
	LastSlot        uint      `json:"last_slot"`
	StartSlot       uint      `json:"start_slot"`
	LastTimestamp   uint      `json:"last_timestamp"`
	StartTimestamp  uint      `json:"start_timestamp"`
	EndSignature    string    `json:"end_signature" gorm:"type:varchar(100)"`
	StartSignature  string    `json:"start_signature" gorm:"type:varchar(100)"`
	MintChange      float64   `json:"mint_change"`
	SolChange       float64   `json:"sol_change"`
	MintVolume      float64   `json:"mint_volume" gorm:"default:0"`
	SolVolume       float64   `json:"sol_volume" gorm:"default:0"`
	TxCount         uint      `json:"tx_count" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for PumpfuninternalHolder
func (PumpfuninternalHolder) TableName() string {
	return "pumpfuninternal_holder"
}

// PumpfunAmmPoolSwap represents a swap record in the pumpfunammpool system
type PumpfunAmmPoolSwap struct {
	ID                        uint      `json:"id" gorm:"primaryKey"`
	Slot                      uint      `json:"slot"`
	Timestamp                 uint      `json:"timestamp"`
	PoolAddress               string    `json:"pool_address" gorm:"type:varchar(100)"`
	Signature                 string    `json:"signature" gorm:"type:varchar(100)"`
	Fee                       float64   `json:"fee"`
	Address                   string    `json:"address" gorm:"type:varchar(100)"`
	BaseMint                  string    `json:"base_mint" gorm:"type:varchar(100)"`
	QuoteMint                 string    `json:"quote_mint" gorm:"type:varchar(100)"`
	TraderBaseChange          float64   `json:"trader_base_change"`
	TraderQuoteChange         float64   `json:"trader_quote_change"`
	TraderSolChange           float64   `json:"trader_sol_change"`
	PoolBaseChange            float64   `json:"pool_base_change"`
	PoolQuoteChange           float64   `json:"pool_quote_change"`
	PoolBaseAccountSolChange  float64   `json:"pool_base_account_sol_change"`
	PoolQuoteAccountSolChange float64   `json:"pool_quote_account_sol_change"`
	CreatedAt                 time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for PumpfunAmmPoolSwap
func (PumpfunAmmPoolSwap) TableName() string {
	return "pumpfunammpool_swap"
}

// PumpfunAmmpoolHolder represents a holder record in the pumpfunammpool system
type PumpfunAmmpoolHolder struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Address           string    `json:"address" gorm:"type:varchar(100)"`
	HolderType        string    `json:"holder_type" gorm:"type:varchar(64)"`
	PoolAddress       string    `json:"pool_address" gorm:"type:varchar(100)"`
	BaseMint          string    `json:"base_mint" gorm:"type:varchar(100)"`
	QuoteMint         string    `json:"quote_mint" gorm:"type:varchar(100)"`
	LastSlot          uint      `json:"last_slot"`
	StartSlot         uint      `json:"start_slot"`
	LastTimestamp     uint      `json:"last_timestamp"`
	StartTimestamp    uint      `json:"start_timestamp"`
	EndSignature      string    `json:"end_signature" gorm:"type:varchar(100)"`
	StartSignature    string    `json:"start_signature" gorm:"type:varchar(100)"`
	BaseChange        float64   `json:"base_change"`
	QuoteChange       float64   `json:"quote_change"`
	SolChange         float64   `json:"sol_change"`
	TraderBaseVolume  float64   `json:"trader_base_volume" gorm:"default:0"`
	TraderQuoteVolume float64   `json:"trader_quote_volume" gorm:"default:0"`
	TraderSolVolume   float64   `json:"trader_sol_volume" gorm:"default:0"`
	TxCount           uint      `json:"tx_count" gorm:"default:0"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for PumpfunAmmpoolHolder
func (PumpfunAmmpoolHolder) TableName() string {
	return "pumpfunammpool_holder"
}

// RaydiumPoolHolder represents a holder in a Raydium pool
type RaydiumPoolHolder struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Address        string    `json:"address" gorm:"type:varchar(128)"`
	HolderType     string    `json:"holder_type" gorm:"type:varchar(64)"`
	PoolAddress    string    `json:"pool_address" gorm:"type:varchar(128)"`
	BaseMint       string    `json:"base_mint" gorm:"type:varchar(128)"`
	QuoteMint      string    `json:"quote_mint" gorm:"type:varchar(128)"`
	LastSlot       uint      `json:"last_slot"`
	StartSlot      uint      `json:"start_slot"`
	LastTimestamp  uint      `json:"last_timestamp"`
	StartTimestamp uint      `json:"start_timestamp"`
	EndSignature   string    `json:"end_signature" gorm:"type:varchar(128)"`
	StartSignature string    `json:"start_signature" gorm:"type:varchar(128)"`
	BaseChange     float64   `json:"base_change"`
	QuoteChange    float64   `json:"quote_change"`
	SolChange      float64   `json:"sol_change"`
	TxCount        uint      `json:"tx_count" gorm:"default:0"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for RaydiumPoolHolder
func (RaydiumPoolHolder) TableName() string {
	return "raydiumpool_holder"
}

// RaydiumPoolSwap represents a swap record in a Raydium pool
type RaydiumPoolSwap struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Slot              uint      `json:"slot"`
	Timestamp         uint      `json:"timestamp"`
	PoolAddress       string    `json:"pool_address" gorm:"type:varchar(128)"`
	Signature         string    `json:"signature" gorm:"type:varchar(128)"`
	Fee               float64   `json:"fee"`
	Address           string    `json:"address" gorm:"type:varchar(128)"`
	BaseMint          string    `json:"base_mint" gorm:"type:varchar(128)"`
	QuoteMint         string    `json:"quote_mint" gorm:"type:varchar(128)"`
	TraderBaseChange  float64   `json:"trader_base_change"`
	TraderQuoteChange float64   `json:"trader_quote_change"`
	TraderSolChange   float64   `json:"trader_sol_change"`
	PoolBaseChange    float64   `json:"pool_base_change"`
	PoolQuoteChange   float64   `json:"pool_quote_change"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for RaydiumPoolSwap
func (RaydiumPoolSwap) TableName() string {
	return "raydiumpool_swap"
}

// MeteoradbcHolder represents a holder in a Meteoradbc pool
type MeteoradbcHolder struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Address        string    `json:"address" gorm:"type:varchar(128)"`
	HolderType     string    `json:"holder_type" gorm:"type:varchar(64)"`
	PoolAddress    string    `json:"pool_address" gorm:"type:varchar(128)"`
	BaseMint       string    `json:"base_mint" gorm:"type:varchar(128)"`
	QuoteMint      string    `json:"quote_mint" gorm:"type:varchar(128)"`
	LastSlot       uint      `json:"last_slot"`
	StartSlot      uint      `json:"start_slot"`
	LastTimestamp  uint      `json:"last_timestamp"`
	StartTimestamp uint      `json:"start_timestamp"`
	EndSignature   string    `json:"end_signature" gorm:"type:varchar(128)"`
	StartSignature string    `json:"start_signature" gorm:"type:varchar(128)"`
	BaseChange     float64   `json:"base_change"`
	QuoteChange    float64   `json:"quote_change"`
	SolChange      float64   `json:"sol_change"`
	TxCount        uint      `json:"tx_count" gorm:"default:0"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for MeteoradbcHolder
func (MeteoradbcHolder) TableName() string {
	return "meteoradbc_holder"
}

// MeteoradbcSwap represents a swap record in a Meteoradbc pool
type MeteoradbcSwap struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Slot              uint      `json:"slot"`
	Timestamp         uint      `json:"timestamp"`
	PoolAddress       string    `json:"pool_address" gorm:"type:varchar(128)"`
	Signature         string    `json:"signature" gorm:"type:varchar(128)"`
	Fee               float64   `json:"fee"`
	Address           string    `json:"address" gorm:"type:varchar(128)"`
	BaseMint          string    `json:"base_mint" gorm:"type:varchar(128)"`
	QuoteMint         string    `json:"quote_mint" gorm:"type:varchar(128)"`
	TraderBaseChange  float64   `json:"trader_base_change"`
	TraderQuoteChange float64   `json:"trader_quote_change"`
	TraderSolChange   float64   `json:"trader_sol_change"`
	PoolBaseChange    float64   `json:"pool_base_change"`
	PoolQuoteChange   float64   `json:"pool_quote_change"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for MeteoradbcSwap
func (MeteoradbcSwap) TableName() string {
	return "meteoradbc_swap"
}

// MeteoradbcHolder represents a holder in a Meteoradbc pool
type MeteoracpmmHolder struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	Address        string    `json:"address" gorm:"type:varchar(128)"`
	HolderType     string    `json:"holder_type" gorm:"type:varchar(64)"`
	PoolAddress    string    `json:"pool_address" gorm:"type:varchar(128)"`
	BaseMint       string    `json:"base_mint" gorm:"type:varchar(128)"`
	QuoteMint      string    `json:"quote_mint" gorm:"type:varchar(128)"`
	LastSlot       uint      `json:"last_slot"`
	StartSlot      uint      `json:"start_slot"`
	LastTimestamp  uint      `json:"last_timestamp"`
	StartTimestamp uint      `json:"start_timestamp"`
	EndSignature   string    `json:"end_signature" gorm:"type:varchar(128)"`
	StartSignature string    `json:"start_signature" gorm:"type:varchar(128)"`
	BaseChange     float64   `json:"base_change"`
	QuoteChange    float64   `json:"quote_change"`
	SolChange      float64   `json:"sol_change"`
	TxCount        uint      `json:"tx_count" gorm:"default:0"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for MeteoradbcHolder
func (MeteoracpmmHolder) TableName() string {
	return "meteoracpmm_holder"
}

// MeteoradbcSwap represents a swap record in a Meteoradbc pool
type MeteoracpmmSwap struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	Slot              uint      `json:"slot"`
	Timestamp         uint      `json:"timestamp"`
	PoolAddress       string    `json:"pool_address" gorm:"type:varchar(128)"`
	Signature         string    `json:"signature" gorm:"type:varchar(128)"`
	Fee               float64   `json:"fee"`
	Address           string    `json:"address" gorm:"type:varchar(128)"`
	BaseMint          string    `json:"base_mint" gorm:"type:varchar(128)"`
	QuoteMint         string    `json:"quote_mint" gorm:"type:varchar(128)"`
	TraderBaseChange  float64   `json:"trader_base_change"`
	TraderQuoteChange float64   `json:"trader_quote_change"`
	TraderSolChange   float64   `json:"trader_sol_change"`
	PoolBaseChange    float64   `json:"pool_base_change"`
	PoolQuoteChange   float64   `json:"pool_quote_change"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for MeteoradbcSwap
func (MeteoracpmmSwap) TableName() string {
	return "meteoracpmm_swap"
}

// SwapTransaction represents a swap transaction record
type SwapTransaction struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Signature   string    `json:"signature" gorm:"type:varchar(128);uniqueIndex"`
	Slot        uint      `json:"slot"`
	Timestamp   uint      `json:"timestamp"`
	PayerType   string    `json:"payer_type" gorm:"type:varchar(64)"`
	Payer       string    `json:"payer" gorm:"type:varchar(128)"`
	PoolAddress string    `json:"pool_address" gorm:"type:varchar(128)"`
	BaseMint    string    `json:"base_mint" gorm:"type:varchar(128)"`
	QuoteMint   string    `json:"quote_mint" gorm:"type:varchar(128)"`
	BaseChange  float64   `json:"base_change"`
	QuoteChange float64   `json:"quote_change"`
	IsSuccess   bool      `json:"is_success"`
	TxMeta      string    `json:"tx_meta" gorm:"type:text;default:''"`
	TxError     string    `json:"tx_error" gorm:"type:text;default:''"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for SwapTransaction
func (SwapTransaction) TableName() string {
	return "swap_transaction"
}
