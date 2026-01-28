package models

import (
	"encoding/json"
	"time"
)

type StrategySignal struct {
	ID                  uint            `gorm:"primarykey" json:"id"`
	ProjectID           uint            `gorm:"not null" json:"project_id"`
	StrategyID          uint            `gorm:"not null" json:"strategy_id"`
	RoleID              uint            `gorm:"not null" json:"role_id"`
	StrategyParams      json.RawMessage `gorm:"type:jsonb" json:"strategy_params"`
	StrategyStat        json.RawMessage `gorm:"type:jsonb" json:"strategy_stat"`
	TransactionParams   json.RawMessage `gorm:"type:jsonb;column:transaction_params" json:"transaction_params"`
	TransactionDetail   json.RawMessage `gorm:"type:jsonb;column:transaction_detail" json:"transaction_detail"`
	UseBundle           bool            `gorm:"default:false" json:"use_bundle"`
	SimulateResult      string          `gorm:"type:varchar(128);default:''" json:"simulate_result"`
	TransactionLog      string          `gorm:"type:text;default:''" json:"transaction_log"`
	CreatedAt           time.Time       `json:"created_at" gorm:"autoCreateTime"`
}

func (StrategySignal) TableName() string {
	return "strategy_signal"
} 