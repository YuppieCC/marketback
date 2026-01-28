package models

import (
	"time"
)

// ProjectSettleRecord 项目结算记录
type ProjectSettleRecord struct {
	ID                             uint      `json:"id" gorm:"primaryKey"`
	ProjectID                      uint      `json:"project_id" gorm:"not null;index"`
	PoolPlatform                   string    `json:"pool_platform" gorm:"size:50;not null"`
	PoolPrice                      float64   `json:"pool_price"`
	PoolSlot                       uint64    `json:"pool_slot"`
	PoolUpdatedAt                  time.Time `json:"pool_updated_at"`
	TokenByProject                 float64   `json:"token_by_project"`
	TokenByPool                    float64   `json:"token_by_pool"`
	TokenByRetailInvestors         float64   `json:"token_by_retail_investors"`
	TokenAllocationByRetailInvestors float64 `json:"token_allocation_by_retail_investors"`
	TokenAllocationByProject       float64   `json:"token_allocation_by_project"`
	TokenAllocationByPool          float64   `json:"token_allocation_by_pool"`
	TvlByProjectToken             float64   `json:"tvl_by_project_token"`
	TvlByProject                  float64   `json:"tvl_by_project"`
	TvlByRetailInvestors          float64   `json:"tvl_by_retail_investors"`
	ProjectPnl                     float64   `json:"project_pnl"`
	ProjectMinPnl                  float64   `json:"project_min_pnl"`
	CreatedAt                      time.Time `json:"created_at" gorm:"autoCreateTime"`
	CreatedAtByZeroSec            time.Time `json:"created_at_by_zero_sec" gorm:"index"` // 零秒时间戳，用于按分钟聚合
}
