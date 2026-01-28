package handlers

import (
	"marketcontrol/internal/models"
)

// AggregateTokenStat 聚合代币统计结构体
type AggregateTokenStat struct {
	Mint            string  `json:"mint"`
	Decimals        uint    `json:"decimals"`
	Balance         uint64  `json:"balance"`
	BalanceReadable float64 `json:"balance_readable"`
}

// WalletTokenStatResp 钱包代币统计响应结构体
type WalletTokenStatResp struct {
	Mint            string  `json:"mint"`
	Decimals        uint    `json:"decimals"`
	Balance         uint64  `json:"balance"`
	BalanceReadable float64 `json:"balance_readable"`
	Slot            uint64  `json:"slot"`
	BlockTime       int64   `json:"block_time"`
	CreatedAt       int64   `json:"created_at"`
	UpdatedAt       int64   `json:"updated_at"`
}

// TokenGroup 代币分组结构体
type TokenGroup struct {
	OwnerAddress string                `json:"owner_address"`
	Tokens       []WalletTokenStatResp `json:"tokens"`
}

// TokenConfigResp 代币配置响应结构体
type TokenConfigResp struct {
	ID          uint    `json:"id"`
	Mint        string  `json:"mint"`
	Symbol      string  `json:"symbol"`
	Name        string  `json:"name"`
	Decimals    int     `json:"decimals"`
	LogoURI     string  `json:"logo_uri"`
	CreatedAt   int64   `json:"created_at"`
	UpdatedAt   int64   `json:"updated_at"`
	TotalSupply float64 `json:"total_supply"`
}

// PoolConfigResp 池子配置响应结构体
type PoolConfigResp struct {
	ID          uint             `json:"id"`
	Platform    string           `json:"platform"`
	PoolAddress string           `json:"pool_address"`
	BaseIsWSOL  bool             `json:"base_is_wsol"`
	BaseMintID  uint             `json:"base_mint_id"`
	QuoteMintID uint             `json:"quote_mint_id"`
	BaseVault   string           `json:"base_vault"`
	QuoteVault  string           `json:"quote_vault"`
	LpMintID    uint             `json:"lp_mint_id"`
	FeeRate     float64          `json:"fee_rate"`
	Status      string           `json:"status"`
	CreatedAt   int64            `json:"created_at"`
	UpdatedAt   int64            `json:"updated_at"`
	BaseMint    *TokenConfigResp `json:"base_mint"`
	QuoteMint   *TokenConfigResp `json:"quote_mint"`
}

// PoolStatResp 池子统计响应结构体
type PoolStatResp struct {
	ID                  uint            `json:"id"`
	PoolID              uint            `json:"pool_id"`
	Platform            string          `json:"platform"`
	BaseAmount          uint64          `json:"base_amount"`
	QuoteAmount         uint64          `json:"quote_amount"`
	BaseAmountReadable  float64         `json:"base_amount_readable"`
	QuoteAmountReadable float64         `json:"quote_amount_readable"`
	MarketValue         float64         `json:"market_value"`
	LpSupply            uint64          `json:"lp_supply"`
	Price               float64         `json:"price"`
	Slot                uint64          `json:"slot"`
	BlockTime           int64           `json:"block_time"`
	CreatedAt           int64           `json:"created_at"`
	UpdatedAt           int64           `json:"updated_at"`
	Pool                *PoolConfigResp `json:"pool"`
}

// PoolStatRespSimple 池子统计响应结构体（不包含pool字段）
type PoolStatRespSimple struct {
	ID                  uint    `json:"id"`
	PoolID              uint    `json:"pool_id"`
	Platform            string  `json:"platform"`
	BaseAmount          uint64  `json:"base_amount"`
	QuoteAmount         uint64  `json:"quote_amount"`
	BaseAmountReadable  float64 `json:"base_amount_readable"`
	QuoteAmountReadable float64 `json:"quote_amount_readable"`
	MarketValue         float64 `json:"market_value"`
	LpSupply            uint64  `json:"lp_supply"`
	Price               float64 `json:"price"`
	Slot                uint64  `json:"slot"`
	BlockTime           int64   `json:"block_time"`
	CreatedAt           int64   `json:"created_at"`
	UpdatedAt           int64   `json:"updated_at"`
}

// PumpfuninternalStatResp 响应结构体
type PumpfuninternalStatResp struct {
	ID                  uint    `json:"id"`
	PumpfuninternalID  uint    `json:"pumpfuninternal_id"`
	Platform           string  `json:"platform"`
	Mint               string  `json:"mint"`
	UnknownData        uint64  `json:"unknown_data"`
	VirtualTokenReserves uint64 `json:"virtual_token_reserves"`
	VirtualSolReserves  uint64 `json:"virtual_sol_reserves"`
	RealTokenReserves   uint64 `json:"real_token_reserves"`
	RealSolReserves     uint64 `json:"real_sol_reserves"`
	TokenTotalSupply    uint64 `json:"token_total_supply"`
	Complete           bool    `json:"complete"`
	Creator           string  `json:"creator"`
	Price             float64 `json:"price"`
	FeeRecipient      string  `json:"fee_recipient"`
	SolBalance        float64 `json:"sol_balance"`
	TokenBalance      float64 `json:"token_balance"`
	Slot             uint64  `json:"slot"`
	BlockTime         int64   `json:"block_time"`
	CreatedAt         int64   `json:"created_at"`
	UpdatedAt         int64   `json:"updated_at"`
}

// AggregateTokenStats 聚合代币统计数据
func AggregateTokenStats(stats []models.WalletTokenStat) map[string]*AggregateTokenStat {
	tokenMap := make(map[string]*AggregateTokenStat)
	for _, stat := range stats {
		if agg, ok := tokenMap[stat.Mint]; ok {
			agg.Balance += stat.Balance
			agg.BalanceReadable += stat.BalanceReadable
		} else {
			tokenMap[stat.Mint] = &AggregateTokenStat{
				Mint:            stat.Mint,
				Decimals:        stat.Decimals,
				Balance:         stat.Balance,
				BalanceReadable: stat.BalanceReadable,
			}
		}
	}
	return tokenMap
}

// BuildTokenConfigResp 构建代币配置响应
func BuildTokenConfigResp(token *models.TokenConfig) *TokenConfigResp {
	if token == nil {
		return nil
	}
	return &TokenConfigResp{
		ID:          token.ID,
		Mint:        token.Mint,
		Symbol:      token.Symbol,
		Name:        token.Name,
		Decimals:    token.Decimals,
		LogoURI:     token.LogoURI,
		CreatedAt:   token.CreatedAt.UnixMilli(),
		UpdatedAt:   token.UpdatedAt.UnixMilli(),
		TotalSupply: token.TotalSupply,
	}
}

// BuildPoolConfigResp 构建池子配置响应
func BuildPoolConfigResp(pool *models.PoolConfig) *PoolConfigResp {
	if pool == nil {
		return nil
	}
	return &PoolConfigResp{
		ID:          pool.ID,
		Platform:    pool.Platform,
		PoolAddress: pool.PoolAddress,
		BaseIsWSOL:  pool.BaseIsWSOL,
		BaseMintID:  pool.BaseMintID,
		QuoteMintID: pool.QuoteMintID,
		BaseVault:   pool.BaseVault,
		QuoteVault:  pool.QuoteVault,
		LpMintID:    pool.LpMintID,
		FeeRate:     pool.FeeRate,
		Status:      pool.Status,
		CreatedAt:   pool.CreatedAt.UnixMilli(),
		UpdatedAt:   pool.UpdatedAt.UnixMilli(),
		BaseMint:    BuildTokenConfigResp(pool.BaseMint),
		QuoteMint:   BuildTokenConfigResp(pool.QuoteMint),
	}
}

// BuildPoolStatResp 构建池子统计响应
func BuildPoolStatResp(poolStat *models.PoolStat) *PoolStatResp {
	if poolStat == nil {
		return nil
	}
	return &PoolStatResp{
		ID:                  poolStat.ID,
		PoolID:              poolStat.PoolID,
		Platform:            "raydium",
		BaseAmount:          poolStat.BaseAmount,
		QuoteAmount:         poolStat.QuoteAmount,
		BaseAmountReadable:  poolStat.BaseAmountReadable,
		QuoteAmountReadable: poolStat.QuoteAmountReadable,
		MarketValue:         poolStat.MarketValue,
		LpSupply:            poolStat.LpSupply,
		Price:               poolStat.Price,
		Slot:                poolStat.Slot,
		BlockTime:           poolStat.BlockTime.UnixMilli(),
		CreatedAt:           poolStat.CreatedAt.UnixMilli(),
		UpdatedAt:           poolStat.UpdatedAt.UnixMilli(),
		Pool:                BuildPoolConfigResp(poolStat.Pool),
	}
}

// BuildPoolStatRespSimple 构建池子统计响应（不包含pool字段）
func BuildPoolStatRespSimple(poolStat *models.PoolStat) *PoolStatRespSimple {
	if poolStat == nil {
		return nil
	}
	return &PoolStatRespSimple{
		ID:                  poolStat.ID,
		PoolID:              poolStat.PoolID,
		Platform:            "raydium",
		BaseAmount:          poolStat.BaseAmount,
		QuoteAmount:         poolStat.QuoteAmount,
		BaseAmountReadable:  poolStat.BaseAmountReadable,
		QuoteAmountReadable: poolStat.QuoteAmountReadable,
		MarketValue:         poolStat.MarketValue,
		LpSupply:            poolStat.LpSupply,
		Price:               poolStat.Price,
		Slot:                poolStat.Slot,
		BlockTime:           poolStat.BlockTime.UnixMilli(),
		CreatedAt:           poolStat.CreatedAt.UnixMilli(),
		UpdatedAt:           poolStat.UpdatedAt.UnixMilli(),
	}
}

// BuildPumpfuninternalStatRespSimple 构建简化的 PumpfuninternalStat 响应
func BuildPumpfuninternalStatRespSimple(stat *models.PumpfuninternalStat) *PumpfuninternalStatResp {
	if stat == nil {
		return nil
	}
	return &PumpfuninternalStatResp{
		ID:                  stat.ID,
		PumpfuninternalID:  stat.PumpfuninternalID,
		Platform:           "pumpfun_internal",
		Mint:               stat.Mint,
		UnknownData:        stat.UnknownData,
		VirtualTokenReserves: stat.VirtualTokenReserves,
		VirtualSolReserves:  stat.VirtualSolReserves,
		RealTokenReserves:   stat.RealTokenReserves,
		RealSolReserves:     stat.RealSolReserves,
		TokenTotalSupply:    stat.TokenTotalSupply,
		Complete:            stat.Complete,
		Creator:             stat.Creator,
		Price:               stat.Price,
		FeeRecipient:        stat.FeeRecipient,
		SolBalance:          stat.SolBalance,
		TokenBalance:        stat.TokenBalance,
		Slot:               stat.Slot,
		BlockTime:          stat.BlockTime.UnixMilli(),
		CreatedAt:          stat.CreatedAt.UnixMilli(),
		UpdatedAt:          stat.UpdatedAt.UnixMilli(),
	}
} 