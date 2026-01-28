package models

import (
	"time"
)

// MeteoradbcConfig represents the configuration for a Meteoradbc pool
type MeteoradbcConfig struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	PoolAddress           string    `json:"pool_address" gorm:"size:44;uniqueIndex"`
	Creator               string    `json:"creator" gorm:"size:44"`
	PoolConfig            string    `json:"pool_config" gorm:"size:44"`
	BaseMint              string    `json:"base_mint" gorm:"size:44"`
	QuoteMint             string    `json:"quote_mint" gorm:"size:44"`
	PoolBaseTokenAccount  string    `json:"pool_base_token_account" gorm:"size:44"`
	PoolQuoteTokenAccount string    `json:"pool_quote_token_account" gorm:"size:44"`
	FirstBuyer            string    `json:"first_buyer" gorm:"size:44"`
	DammV2PoolAddress     string    `json:"damm_v2_pool_address" gorm:"size:44"`
	IsMigrated            bool      `json:"is_migrated" gorm:"default:false"`
	Status                string    `json:"status" gorm:"size:20;default:'active'"`
	CreatedAt             time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for MeteoradbcConfig
func (MeteoradbcConfig) TableName() string {
	return "meteoradbc_config"
}
