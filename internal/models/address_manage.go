package models

import (
	"time"

	"gorm.io/gorm"
)

// AddressTag represents the type of address tag
type AddressTag string

// 删除 const 声明，因为不再需要 Tag 相关的常量
const (
	TagMarketMaker    AddressTag = "marketmaker"
	TagBrush          AddressTag = "brush"
	TagGasDistributor AddressTag = "gas-distributor"
)

// AddressManage represents a managed blockchain address
type AddressManage struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	Address    string         `gorm:"size:100;not null;uniqueIndex:idx_address_manages_address" json:"address"`
	PrivateKey string         `gorm:"size:255;not null" json:"private_key"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (AddressManage) TableName() string {
	return "address_manages"
}

// AddressConfig represents the address configuration for trading
type AddressConfig struct {
	ID                   uint           `gorm:"primarykey" json:"id"`
	Address              string         `gorm:"size:128;not null" json:"address"`
	Mint                 string         `gorm:"size:128;not null" json:"mint"`
	IsBuyAllowed         bool           `json:"is_buy_allowed"`
	IsSellAllowed        bool           `json:"is_sell_allowed"`
	IsTradeAllowed       bool           `json:"is_trade_allowed"`
	TradePriorityRate    uint           `gorm:"default:1" json:"trade_priority_rate"`
	IsClosed             bool           `gorm:"default:false" json:"is_closed"`
	IsTokenAccountClosed bool           `gorm:"default:false" json:"is_token_account_closed"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (AddressConfig) TableName() string {
	return "address_config"
}

// DisposableAddressManage represents a disposable managed blockchain address
type DisposableAddressManage struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	Address      string         `gorm:"size:100;not null;uniqueIndex:idx_disposable_address_manages_address" json:"address"`
	PrivateKey   string         `gorm:"size:255;not null" json:"private_key"`
	IsDeprecated bool           `gorm:"default:false" json:"is_deprecated"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName specifies the table name
func (DisposableAddressManage) TableName() string {
	return "disposable_address_manages"
}
