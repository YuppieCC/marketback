package models

import (
	"encoding/json"
	"time"
)

type StrategyConfig struct {
	ID             uint            `gorm:"primarykey" json:"id"`
	ProjectID      uint            `gorm:"not null" json:"project_id"`
	RoleID         uint            `gorm:"not null" json:"role_id"`
	StrategyName   string          `gorm:"size:20;not null" json:"strategy_name"`
	StrategyType   string          `gorm:"size:20;not null" json:"strategy_type"`
	StrategyParams json.RawMessage `gorm:"type:jsonb" json:"strategy_params"`
	StrategyStat   json.RawMessage `gorm:"type:jsonb" json:"strategy_stat"`
	Enabled        bool            `gorm:"default:false" json:"enabled"`
	CreatedAt      time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
}

func (StrategyConfig) TableName() string {
	return "strategy_config"
}
