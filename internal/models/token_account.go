package models

import "time"

type TokenAccount struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	OwnerAddress   string    `gorm:"size:100;not null" json:"owner_address"`
	Mint           string    `gorm:"size:100;not null" json:"mint"`
	AccountAddress string    `gorm:"size:100;uniqueIndex;not null" json:"account_address"`
	IsClose        bool      `gorm:"default:false" json:"is_close"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (TokenAccount) TableName() string {
	return "token_account"
} 