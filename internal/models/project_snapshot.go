package models

import "time"

type WalletTokenSnapshot struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	ProjectID       uint      `json:"project_id"`
	SnapshotID      uint      `json:"snapshot_id"`
	RoleID          uint      `json:"role_id"`
	OwnerAddress    string    `gorm:"size:100" json:"owner_address"`
	Mint            string    `gorm:"size:100" json:"mint"`
	BalanceReadable float64   `json:"balance_readable"`
	Slot            uint64    `json:"slot"`
	BlockTime       time.Time `json:"block_time"`
	SourceUpdatedAt time.Time `json:"source_updated_at"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type PoolSnapshot struct {
	ID                 uint      `gorm:"primarykey" json:"id"`
	ProjectID          uint      `json:"project_id"`
	SnapshotID         uint      `json:"snapshot_id"`
	PoolAddress        string    `gorm:"size:100" json:"pool_address"`
	BaseAmountReadable float64   `json:"base_amount_readable"`
	QuoteAmountReadable float64  `json:"quote_amount_readable"`
	MarketValue        float64   `json:"market_value"`
	LpSupply           uint64    `json:"lp_supply"`
	Price              float64   `json:"price"`
	SourceUpdatedAt    time.Time `json:"source_updated_at"`
	CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type PumpfuninternalSnapshot struct {
	ID                  uint      `gorm:"primarykey" json:"id"`
	ProjectID          uint      `json:"project_id"`
	SnapshotID         uint      `json:"snapshot_id"`
	PumpfuninternalID  uint      `json:"pumpfuninternal_id"`
	Mint               string    `gorm:"size:100" json:"mint"`
	UnknownData        uint64    `json:"unknown_data"`
	VirtualTokenReserves uint64  `json:"virtual_token_reserves"`
	VirtualSolReserves uint64   `json:"virtual_sol_reserves"`
	RealTokenReserves  uint64   `json:"real_token_reserves"`
	RealSolReserves    uint64   `json:"real_sol_reserves"`
	TokenTotalSupply   uint64   `json:"token_total_supply"`
	Complete           bool      `json:"complete"`
	Creator           string    `gorm:"size:100" json:"creator"`
	Price             float64   `json:"price"`
	FeeRecipient      string    `gorm:"size:100" json:"fee_recipient"`
	SolBalance        float64   `json:"sol_balance"`
	TokenBalance      float64   `json:"token_balance"`
	Slot             uint64    `json:"slot"`
	SourceUpdatedAt   time.Time `json:"source_updated_at"`
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (WalletTokenSnapshot) TableName() string {
	return "wallet_token_snapshots"
}

func (PoolSnapshot) TableName() string {
	return "pool_snapshots"
}

func (PumpfuninternalSnapshot) TableName() string {
	return "pumpfuninternal_snapshots"
}
