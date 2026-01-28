package models

import (
	"time"
)

// SystemLog represents a record in system_logs table
type SystemLog struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	ProjectID  uint      `gorm:"column:project_id;default:0" json:"project_id"`
	Level      string    `gorm:"column:level;size:10;not null" json:"level"` // DEBUG, INFO, WARN, ERROR, FATAL
	Message    string    `gorm:"column:message;type:text;not null" json:"message"`
	Module     string    `gorm:"column:module;size:100" json:"module"`
	ErrorStack string    `gorm:"column:error_stack;type:text" json:"error_stack"`
	Meta       JSONMap   `gorm:"column:meta;type:jsonb" json:"meta"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (SystemLog) TableName() string {
	return "system_logs"
}

// SystemParams represents a record in system_params table
type SystemParams struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	Name         string    `gorm:"column:name;size:128;not null" json:"name"`
	IsActive     bool      `gorm:"column:is_active;default:true" json:"is_active"`
	PresetID     uint      `gorm:"column:preset_id" json:"preset_id"`
	PresetName   string    `gorm:"column:preset_name;default:''" json:"preset_name"`
	ParamsConfig JSONMap   `gorm:"column:params_config;type:jsonb" json:"params_config"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SystemParams) TableName() string {
	return "system_params"
}

// SystemCommand represents a record in system_command table
type SystemCommand struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	IsActive      bool      `gorm:"column:is_active;default:true" json:"is_active"`
	ProjectID     uint      `gorm:"column:project_id;default:0" json:"project_id"`
	CommandName   string    `gorm:"column:command_name;default:''" json:"command_name"`
	CommandParams JSONMap   `gorm:"column:command_params;type:jsonb" json:"command_params"`
	VerifyParams  JSONMap   `gorm:"column:verify_params;type:jsonb" json:"verify_params"`
	IsSuccess     bool      `gorm:"column:is_success;default:false" json:"is_success"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SystemCommand) TableName() string {
	return "system_command"
}
