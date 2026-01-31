package models

import (
	"time"
)

type RoleConfig struct {
	ID             uint          `gorm:"primarykey" json:"id"`
	RoleName       string        `gorm:"size:64;not null" json:"role_name"`
	UpdateInterval float64       `json:"update_interval"`
	UpdateEnabled  bool          `json:"update_enabled" gorm:"default:true"`
	Hidden         bool          `json:"hidden" gorm:"column:hidden;default:false"`
	LastUpdateAt   *time.Time    `json:"last_update_at"`
	CreatedAt      time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	Addresses      []RoleAddress `gorm:"foreignKey:RoleID" json:"addresses"`
}

type RoleAddress struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	RoleID    uint      `gorm:"not null" json:"role_id"`
	Address   string    `gorm:"size:100;not null" json:"address"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	Role      *RoleConfig `gorm:"foreignKey:RoleID" json:"role"`
}

// RoleConfigRelation represents the relationship between RoleConfig and ProjectConfig
type RoleConfigRelation struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	RoleID    uint      `gorm:"not null" json:"role_id"`
	ProjectID uint      `gorm:"not null" json:"project_id"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (RoleConfig) TableName() string {
	return "role_config"
}

func (RoleAddress) TableName() string {
	return "role_address"
}

func (RoleConfigRelation) TableName() string {
	return "role_config_relation"
}
