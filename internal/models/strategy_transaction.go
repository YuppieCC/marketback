package models

import (
	"time"
)

type StrategyTransaction struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	ProjectID       uint      `gorm:"not null" json:"project_id"`
	StrategyID      uint      `gorm:"not null" json:"strategy_id"`
	SignalID        uint      `gorm:"not null" json:"signal_id"`
	Timestamp       time.Time `gorm:"not null" json:"timestamp"`
	Wallet          string    `gorm:"type:varchar(100);not null" json:"wallet"`
	Direction       string    `gorm:"type:varchar(20);not null" json:"direction"`
	SendAmount      float64   `gorm:"not null" json:"send_amount"`
	BalanceReadable float64   `gorm:"not null" json:"balance_readable"`
	GetAmount       float64   `gorm:"not null" json:"get_amount"`
	Status          string    `gorm:"type:varchar(20);not null" json:"status"`
	Signature       string    `gorm:"type:text;not null" json:"signature"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (StrategyTransaction) TableName() string {
	return "strategy_transaction"
} 