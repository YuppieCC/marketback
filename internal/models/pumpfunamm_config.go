package models

import (
	"time"
)

// PumpfunAmmPoolConfig represents the configuration for a Pumpfun AMM pool
type PumpfunAmmPoolConfig struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	PoolAddress          string    `json:"pool_address" gorm:"size:44;uniqueIndex"`
	PoolBump             uint8     `json:"pool_bump"`
	Index                uint16    `json:"index"`
	Creator              string    `json:"creator" gorm:"size:44"`
	BaseMint             string    `json:"base_mint" gorm:"size:44"`
	QuoteMint            string    `json:"quote_mint" gorm:"size:44"`
	LpMint               string    `json:"lp_mint" gorm:"size:44"`
	PoolBaseTokenAccount string    `json:"pool_base_token_account" gorm:"size:44"`
	PoolQuoteTokenAccount string   `json:"pool_quote_token_account" gorm:"size:44"`
	LpSupply             uint64    `json:"lp_supply"`
	CoinCreator          string    `json:"coin_creator" gorm:"size:44"`
	Status               string    `json:"status" gorm:"size:20;default:'active'"`
	CreatedAt            time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for PumpfunAmmPoolConfig
func (PumpfunAmmPoolConfig) TableName() string {
	return "pumpfunammpool_config"
}
