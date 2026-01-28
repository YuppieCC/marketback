package models

import (
	"time"
)

// PumpfuninternalConfig represents a pumpfuninternal configuration
type PumpfuninternalConfig struct {
	ID                     uint      `json:"id" gorm:"primaryKey"`
	Platform               string    `json:"platform"`
	Mint                   string    `json:"mint" gorm:"uniqueIndex:uni_pumpfuninternal_config_mint"`
	BondingCurvePda        string    `json:"bonding_curve_pda"`
	AssociatedBondingCurve string    `json:"associated_bonding_curve"`
	CreatorVaultPda        string    `json:"creator_vault_pda"`
	FeeRecipient           string    `json:"fee_recipient"`
	FeeRate                float64   `json:"fee_rate"`
	Status                 string    `json:"status"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (PumpfuninternalConfig) TableName() string {
	return "pumpfuninternal_config"
} 