package models

import (
	"time"
)

type PoolConfig struct {
	ID           uint        `gorm:"primarykey" json:"id"`
	Platform     string      `gorm:"size:20;not null" json:"platform"`
	PoolAddress  string      `gorm:"size:100;uniqueIndex;not null" json:"pool_address"`
	BaseIsWSOL   bool        `gorm:"default:true" json:"base_is_wsol"`
	BaseMintID   uint        `gorm:"not null" json:"base_mint_id"`
	QuoteMintID  uint        `gorm:"not null" json:"quote_mint_id"`
	BaseVault    string      `gorm:"size:100" json:"base_vault"`
	QuoteVault   string      `gorm:"size:100" json:"quote_vault"`
	LpMintID     uint        `json:"lp_mint_id"`
	FeeRate      float64     `gorm:"default:0" json:"fee_rate"`
	Status       string      `gorm:"size:20" json:"status"`
	CreatedAt    time.Time   `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time   `json:"updated_at" gorm:"autoUpdateTime"`
	BaseMint     *TokenConfig `gorm:"foreignKey:BaseMintID" json:"base_mint"`
	QuoteMint    *TokenConfig `gorm:"foreignKey:QuoteMintID" json:"quote_mint"`
}

func (PoolConfig) TableName() string {
	return "pool_config"
}
