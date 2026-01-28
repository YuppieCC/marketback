package models

import (
	"time"
)

// BlockchainConfig represents the blockchain configuration model
type BlockchainConfig struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ChainID   uint      `json:"chain_id" gorm:"uniqueIndex"`
	Name      string    `json:"name" gorm:"not null"`
	Network   string    `json:"network" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name
func (BlockchainConfig) TableName() string {
	return "blockchain_configs"
}