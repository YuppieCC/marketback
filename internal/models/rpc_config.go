package models

import (
	"time"

	"gorm.io/gorm"
)

// RpcConfig represents an RPC configuration for a blockchain
type RpcConfig struct {
	ID                uint           `json:"id" gorm:"primaryKey"`
	Endpoint          string         `json:"endpoint" gorm:"not null"`
	IsActive          bool           `json:"is_active" gorm:"default:true"`
	BlockchainConfigID uint          `json:"blockchain_config_id" gorm:"not null"`
	BlockchainConfig   BlockchainConfig `json:"blockchain_config" gorm:"foreignKey:BlockchainConfigID"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// TableName specifies the table name for RpcConfig
func (RpcConfig) TableName() string {
	return "rpc_configs"
} 