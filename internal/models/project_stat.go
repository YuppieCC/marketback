package models

import "time"

// WalletTokenStat 代表钱包代币统计信息
type WalletTokenStat struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	OwnerAddress    string    `gorm:"size:100;not null" json:"owner_address"`
	Mint            string    `gorm:"size:100;not null" json:"mint"`
	Decimals        uint      `json:"decimals"`
	Balance         uint64    `json:"balance"`
	BalanceReadable float64   `json:"balance_readable"`
	Slot            uint64    `json:"slot"`
	BlockTime       time.Time `json:"block_time"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// PoolStat 代表池子统计信息
type PoolStat struct {
	ID                  uint        `gorm:"primarykey" json:"id"`
	PoolID              uint        `gorm:"not null" json:"pool_id"`
	BaseAmount          uint64      `json:"base_amount"`
	QuoteAmount         uint64      `json:"quote_amount"`
	BaseAmountReadable  float64     `json:"base_amount_readable"`
	QuoteAmountReadable float64     `json:"quote_amount_readable"`
	MarketValue         float64     `json:"market_value"`
	LpSupply            uint64      `json:"lp_supply"`
	Price               float64     `json:"price"`
	Slot                uint64      `json:"slot"`
	BlockTime           time.Time   `json:"block_time"`
	CreatedAt           time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
	Pool                *PoolConfig `gorm:"foreignKey:PoolID" json:"pool"`
}

// PumpfuninternalStat represents the statistics of a pumpfun internal pool
type PumpfuninternalStat struct {
	ID                   uint                   `gorm:"primarykey" json:"id"`
	PumpfuninternalID    uint                   `gorm:"not null" json:"pumpfuninternal_id"`
	Mint                 string                 `gorm:"size:100;uniqueIndex" json:"mint"`
	UnknownData          uint64                 `json:"unknown_data"`
	VirtualTokenReserves uint64                 `json:"virtual_token_reserves"`
	VirtualSolReserves   uint64                 `json:"virtual_sol_reserves"`
	RealTokenReserves    uint64                 `json:"real_token_reserves"`
	RealSolReserves      uint64                 `json:"real_sol_reserves"`
	TokenTotalSupply     uint64                 `json:"token_total_supply"`
	Complete             bool                   `json:"complete"`
	Creator              string                 `gorm:"size:100" json:"creator"`
	Price                float64                `json:"price"`
	FeeRecipient         string                 `gorm:"size:100" json:"fee_recipient"`
	SolBalance           float64                `json:"sol_balance"`
	TokenBalance         float64                `json:"token_balance"`
	Slot                 uint64                 `json:"slot"`
	BlockTime            time.Time              `json:"block_time"`
	CreatedAt            time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
	PumpfunPool          *PumpfuninternalConfig `gorm:"foreignKey:PumpfuninternalID" json:"pumpfun_pool"`
}

// PumpfunAmmPoolStat represents statistics for a PumpfunAmm pool
type PumpfunAmmPoolStat struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	PoolID              uint      `json:"pool_id" gorm:"not null;index"`
	BaseAmount          uint64    `json:"base_amount"`
	QuoteAmount         uint64    `json:"quote_amount"`
	BaseAmountReadable  float64   `json:"base_amount_readable"`
	QuoteAmountReadable float64   `json:"quote_amount_readable"`
	MarketValue         float64   `json:"market_value"`
	LpSupply            uint64    `json:"lp_supply"`
	Price               float64   `json:"price"`
	Slot                uint64    `json:"slot"`
	BlockTime           time.Time `json:"block_time"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (WalletTokenStat) TableName() string {
	return "wallet_token_stat"
}

func (PoolStat) TableName() string {
	return "pool_stat"
}

func (PumpfuninternalStat) TableName() string {
	return "pumpfuninternal_stat"
}

func (PumpfunAmmPoolStat) TableName() string {
	return "pumpfunammpool_stat"
}

// RaydiumLaunchpadPoolStat represents statistics for a Raydium Launchpad pool
type RaydiumLaunchpadPoolStat struct {
	ID                                 uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PoolAddress                        string    `json:"pool_address" gorm:"type:varchar(128);uniqueIndex"`
	Epoch                              uint64    `json:"epoch"`
	Bump                               uint64    `json:"bump"`
	PoolStatus                         uint64    `json:"pool_status" gorm:"default:0"`
	Mint                               string    `json:"mint" gorm:"type:varchar(128)"`
	MigrateType                        uint64    `json:"migrate_type"`
	Supply                             float64   `json:"supply"`
	TotalSellA                         float64   `json:"total_sell_a"`
	VirtualA                           float64   `json:"virtual_a"`
	VirtualB                           float64   `json:"virtual_b"`
	RealA                              float64   `json:"real_a"`
	RealB                              float64   `json:"real_b"`
	TotalFundRaisingB                  float64   `json:"total_fund_raising_b"`
	ProtocolFee                        float64   `json:"protocol_fee"`
	PlatformFee                        float64   `json:"platform_fee"`
	MigrateFee                         float64   `json:"migrate_fee"`
	VestingScheduleTotalLockedAmount   float64   `json:"vesting_schedule_total_locked_amount"`
	VestingScheduleCliffPeriod         float64   `json:"vesting_schedule_cliff_period"`
	VestingScheduleUnlockPeriod        float64   `json:"vesting_schedule_unlock_period"`
	VestingScheduleStartTime           float64   `json:"vesting_schedule_start_time"`
	VestingScheduleTotalAllocatedShare float64   `json:"vesting_schedule_total_allocated_share"`
	MintProgramFlag                    float64   `json:"mint_program_flag"`
	BaseBalance                        float64   `json:"base_balance"`
	QuoteBalance                       float64   `json:"quote_balance"`
	Slot                               uint64    `json:"slot"`
	BlockTime                          time.Time `json:"block_time" gorm:"type:timestamp"`
	CreatedAt                          time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt                          time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (RaydiumLaunchpadPoolStat) TableName() string {
	return "raydium_launchpad_pool_stat"
}

// RaydiumCpmmPoolStat represents statistics for a Raydium CPMM pool
type RaydiumCpmmPoolStat struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PoolID              uint      `json:"pool_id" gorm:"not null"`
	BaseAmount          uint64    `json:"base_amount"`
	QuoteAmount         uint64    `json:"quote_amount"`
	BaseAmountReadable  float64   `json:"base_amount_readable"`
	QuoteAmountReadable float64   `json:"quote_amount_readable"`
	MarketValue         float64   `json:"market_value"`
	LpSupply            uint64    `json:"lp_supply"`
	BurnPercent         float64   `json:"burn_percent"`
	Price               float64   `json:"price"`
	Slot                uint64    `json:"slot"`
	BlockTime           time.Time `json:"block_time" gorm:"type:timestamp"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (RaydiumCpmmPoolStat) TableName() string {
	return "raydium_cpmm_pool_stat"
}

// MeteoradbcPoolStat represents statistics for a Meteoradbc pool
type MeteoradbcPoolStat struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PoolAddress         string    `json:"pool_address" gorm:"type:varchar(128);uniqueIndex"`
	BaseAmount          uint64    `json:"base_amount"`
	QuoteAmount         uint64    `json:"quote_amount"`
	BaseAmountReadable  float64   `json:"base_amount_readable"`
	QuoteAmountReadable float64   `json:"quote_amount_readable"`
	MarketValue         float64   `json:"market_value"`
	Price               float64   `json:"price"`
	Slot                uint64    `json:"slot"`
	BlockTime           time.Time `json:"block_time" gorm:"type:timestamp"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (MeteoradbcPoolStat) TableName() string {
	return "meteoradbc_pool_stat"
}

// MeteoracpmmPoolStat represents statistics for a Meteoracpmm pool
type MeteoracpmmPoolStat struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	PoolAddress         string    `json:"pool_address" gorm:"type:varchar(128);uniqueIndex"`
	BaseAmount          uint64    `json:"base_amount"`
	QuoteAmount         uint64    `json:"quote_amount"`
	BaseAmountReadable  float64   `json:"base_amount_readable"`
	QuoteAmountReadable float64   `json:"quote_amount_readable"`
	MarketValue         float64   `json:"market_value"`
	Price               float64   `json:"price"`
	Slot                uint64    `json:"slot"`
	BlockTime           time.Time `json:"block_time" gorm:"type:timestamp"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (MeteoracpmmPoolStat) TableName() string {
	return "meteoracpmm_pool_stat"
}
