package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type TokenConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Mint      string    `gorm:"size:100;uniqueIndex;not null" json:"mint"`
	Symbol    string    `gorm:"size:16;not null" json:"symbol"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Decimals  int       `gorm:"not null" json:"decimals"`
	LogoURI   string    `gorm:"type:text" json:"logo_uri"`
	TotalSupply float64      `gorm:"not null" json:"total_supply"`
	Creator   string    `gorm:"size:128;default:''" json:"creator"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (TokenConfig) TableName() string {
	return "token_info"
}

// TokenMetadata represents the token_metadata table
type TokenMetadata struct {
	ID          uint            `gorm:"primarykey;autoIncrement" json:"id"`
	Name        string          `gorm:"size:128" json:"name"`
	Symbol      string          `gorm:"size:64" json:"symbol"`
	Description string          `gorm:"size:512" json:"description"`
	Image       string          `gorm:"size:255" json:"image"`
	Twitter     string          `gorm:"size:128" json:"twitter"`
	Telegram    string          `gorm:"size:128" json:"telegram"`
	Website     string          `gorm:"size:255" json:"website"`
	SourceURL   string          `gorm:"type:text;default:''" json:"source_url"`
	SourceData  JSONB           `gorm:"type:jsonb" json:"source_data"`
	IsFavorite  bool            `gorm:"default:false" json:"is_favorite"`
	CreatedAt   time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// JSONB is a custom type to handle JSONB data
type JSONB map[string]interface{}

// Value implements the driver.Valuer interface
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	
	return json.Unmarshal(bytes, j)
}

func (TokenMetadata) TableName() string {
	return "token_metadata"
}
