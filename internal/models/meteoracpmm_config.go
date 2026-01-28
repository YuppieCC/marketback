package models

import (
	"time"
)

// MeteoracpmmConfig represents the configuration for a Meteoracpmm pool
type MeteoracpmmConfig struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	PoolAddress           string    `json:"pool_address" gorm:"size:44;uniqueIndex"`
	DbcPoolAddress        string    `json:"dbc_pool_address" gorm:"size:44"`
	Creator               string    `json:"creator" gorm:"size:44"`
	BaseMint              string    `json:"base_mint" gorm:"size:44"`
	QuoteMint             string    `json:"quote_mint" gorm:"size:44"`
	PoolBaseTokenAccount  string    `json:"pool_base_token_account" gorm:"size:44"`
	PoolQuoteTokenAccount string    `json:"pool_quote_token_account" gorm:"size:44"`
	Status                string    `json:"status" gorm:"size:20;default:'active'"`
	CreatedAt             time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for MeteoracpmmConfig
func (MeteoracpmmConfig) TableName() string {
	return "meteoracpmm_config"
}
