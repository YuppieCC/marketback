package models

import "time"

type TemplateRoleConfig struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	RoleName       string    `gorm:"size:64;not null" json:"role_name"`
	UpdateInterval float64   `json:"update_interval"`
	UpdateEnabled  bool      `json:"update_enabled" gorm:"default:true"`
	LastUpdateAt   *time.Time `json:"last_update_at"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	Addresses      []TemplateRoleAddress `gorm:"foreignKey:TemplateRoleID" json:"addresses"`
}

type TemplateRoleAddress struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	TemplateRoleID  uint      `gorm:"not null" json:"template_role_id"`
	Address         string    `gorm:"size:100;not null" json:"address"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	Role            *TemplateRoleConfig `gorm:"foreignKey:TemplateRoleID" json:"role"`
}

func (TemplateRoleConfig) TableName() string {
	return "template_role_config"
}

func (TemplateRoleAddress) TableName() string {
	return "template_role_address"
} 