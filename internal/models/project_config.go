package models

import (
	"encoding/json"
	// "fmt"
	"time"
	// "gorm.io/gorm"
)

type ProjectConfig struct {
	ID                uint            `gorm:"primarykey" json:"id"`
	Name              string          `gorm:"size:64;not null" json:"name"`
	PoolPlatform      string          `gorm:"size:20;not null;default:'raydium'" json:"pool_platform"` // 'raydium' or 'pumpfun_internal'
	PoolID            uint            `gorm:"not null" json:"pool_id"`
	TokenID           uint            `gorm:"not null" json:"token_id"`
	TokenMetadataID   uint            `gorm:"default:0" json:"token_metadata_id"`
	SnapshotEnabled   bool            `json:"snapshot_enabled"`
	SnapshotCount     int             `json:"snapshot_count"`
	IsActive          bool            `gorm:"default:true" json:"is_active"`
	UpdateStatEnabled bool            `gorm:"default:true" json:"update_stat_enabled"`
	IsMigrated        bool            `gorm:"default:false" json:"is_migrated"`
	IsLocked          bool            `gorm:"default:false" json:"is_locked"`
	AssetsBalance     float64         `gorm:"default:0" json:"assets_balance"`
	RetailSolAmount   float64         `gorm:"default:0" json:"retail_sol_amount"`
	PoolConfig        string          `json:"pool_config" gorm:"size:44"`
	Event             json.RawMessage `json:"event" gorm:"type:jsonb"`
	Vesting           json.RawMessage `json:"vesting" gorm:"type:jsonb"`
	CreatedAt         time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	// Pool            *PoolConfig  `gorm:"foreignKey:PoolID;references:ID" json:"pool,omitempty"`
	// PumpfunPool     *PumpfuninternalConfig `gorm:"foreignKey:PoolID;references:ID" json:"pumpfun_pool,omitempty"`
	Token *TokenConfig `gorm:"foreignKey:TokenID" json:"token"`
}

func (ProjectConfig) TableName() string {
	return "project_config"
}

// AfterFind 在查询后处理不同平台的池子数据
// func (p *ProjectConfig) AfterFind(tx *gorm.DB) error {
// 	if p.PoolPlatform == "pumpfun_internal" {
// 		p.Pool = nil
// 	} else {
// 		p.PumpfunPool = nil
// 	}
// 	return nil
// }

// ProjectFundTransferRecord represents the project fund transfer record table
type ProjectFundTransferRecord struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	ProjectID  uint      `gorm:"not null" json:"project_id"`
	Mint       string    `gorm:"size:100;not null" json:"mint"`
	Direction  string    `gorm:"size:20;not null" json:"direction"` // "in" or "out"
	Amount     float64   `gorm:"not null" json:"amount"`
	TargetName string    `gorm:"size:32;not null;default:'project'" json:"target_name"` // 可选: "project", "pool", "retail_investors"，默认为 "project"
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (ProjectFundTransferRecord) TableName() string {
	return "project_fund_transfer_record"
}

// ProjectExtraAddress 项目额外地址表
type ProjectExtraAddress struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	ProjectID       uint      `gorm:"not null" json:"project_id"`
	Address         string    `gorm:"size:100;not null" json:"address"`
	Enabled         bool      `gorm:"default:true" json:"enabled"`
	PrivateKeyVaild bool      `gorm:"column:private_key_vaild;default:false" json:"private_key_vaild"`
	PrivateKey      string    `gorm:"size:244" json:"private_key"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (ProjectExtraAddress) TableName() string {
	return "project_extra_address"
}
