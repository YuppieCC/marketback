package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NodeType 表示节点类型
type NodeType string

const (
	NodeTypeRoot         NodeType = "root"
	NodeTypeIntermediate NodeType = "intermediate"
	NodeTypeLeaf        NodeType = "leaf"
)

// AddressNode 表示链路图中的节点
type AddressNode struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	MapID        uint      `gorm:"column:map_id;not null" json:"map_id"`
	NodeLabel    string    `gorm:"column:node_label;size:20;not null" json:"node_label"`
	NodeValue    string    `gorm:"column:node_value;size:100;not null" json:"node_value"`
	NodeType     NodeType  `gorm:"column:node_type;size:20" json:"node_type"`
	NodeChainID  int       `gorm:"column:node_chain_id" json:"node_chain_id"`
	NodeDepthID  int       `gorm:"column:node_depth_id" json:"node_depth_id"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	WashMap      WashMap   `gorm:"foreignKey:MapID" json:"-"`
}

// TableName 指定表名
func (AddressNode) TableName() string {
	return "address_nodes"
}

// WashMap 表示洗币图谱模型
type WashMap struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	ProjectID     uint      `gorm:"column:project_id;not null" json:"project_id"`
	ProjectLabel  string    `gorm:"column:project_label;size:50;not null" json:"project_label"`
	MapType       string    `gorm:"column:map_type;size:20;not null" json:"map_type"`
	MapParams     JSONMap   `gorm:"column:map_params;type:jsonb;not null" json:"map_params"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// JSONMap 是一个自定义类型，用于处理 JSONB 数据
type JSONMap map[string]interface{}

// Value 实现 driver.Valuer 接口
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言失败：无法将数据转换为字节切片")
	}
	
	return json.Unmarshal(bytes, &j)
}

// TableName 指定表名
func (WashMap) TableName() string {
	return "address_map"
}

// AddressEdge 表示节点之间的边关系
type AddressEdge struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	FromNodeID  uint      `gorm:"column:from_node_id;not null" json:"from_node_id"`
	ToNodeID    uint      `gorm:"column:to_node_id;not null" json:"to_node_id"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	FromNode    AddressNode `gorm:"foreignKey:FromNodeID" json:"from_node"`
	ToNode      AddressNode `gorm:"foreignKey:ToNodeID" json:"to_node"`
}

// TableName 指定表名
func (AddressEdge) TableName() string {
	return "address_edges"
}

// StatusType 表示任务状态类型
type StatusType string

const (
    StatusUnprocessed StatusType = "unprocessed"
    StatusProcessed   StatusType = "processed"
    StatusProcessing  StatusType = "processing"
    StatusFailed     StatusType = "failed"
)

// WashTask 表示洗币任务模型
type WashTask struct {
    ID              uint      `gorm:"primarykey" json:"id"`
    MapID           uint      `gorm:"column:map_id;not null" json:"map_id"`
    WashTaskManageID uint     `gorm:"column:wash_task_manage_id;not null" json:"wash_task_manage_id"`
    SortID          uint      `gorm:"column:sort_id;not null" json:"sort_id"`
    FromNodeID      uint      `gorm:"column:from_node_id;not null" json:"from_node_id"`
    ToNodeID        uint      `gorm:"column:to_node_id;not null" json:"to_node_id"`
    FromAddress   string    `gorm:"column:from_address;size:100;not null" json:"from_address"`
    ToAddress     string    `gorm:"column:to_address;size:100;not null" json:"to_address"`
    SendToken     string    `gorm:"column:send_token;size:100;not null" json:"send_token"`
    TokenDecimals uint      `gorm:"column:token_decimals" json:"token_decimals"`
    SendAmount    uint64    `gorm:"column:send_amount" json:"send_amount"`
    Gas          float64   `gorm:"column:gas" json:"gas"`
    Reverse      bool      `gorm:"column:reverse" json:"reverse"`
    Signature    string    `gorm:"column:signature;size:100" json:"signature"`
    IsSuccess    bool      `gorm:"column:is_success;default:false" json:"is_success"`
    CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    WashMap      WashMap   `gorm:"foreignKey:MapID" json:"-"`
    FromNode     AddressNode `gorm:"foreignKey:FromNodeID" json:"from_node"`
    ToNode       AddressNode `gorm:"foreignKey:ToNodeID" json:"to_node"`
    WashTaskManage WashTaskManage `gorm:"foreignKey:WashTaskManageID" json:"-"`
    Status        StatusType `gorm:"column:status;type:string;default:'unprocessed'" json:"status"`
}

// TableName 指定表名
func (WashTask) TableName() string {
    return "wash_task"
}

// WashTaskManage 表示洗币任务管理模型
type WashTaskManage struct {
    ID                uint      `gorm:"primarykey" json:"id"`
    MapID             uint      `gorm:"column:map_id;not null" json:"map_id"`
    TaskCount         uint      `gorm:"column:task_count" json:"task_count"`
    TaskGas           float64   `gorm:"column:task_gas" json:"task_gas"`
    SendToken         string    `gorm:"column:send_token;size:100;not null" json:"send_token"`
    TokenDecimals     uint      `gorm:"column:token_decimals" json:"token_decimals"`
    TaskAmount        float64   `gorm:"column:task_amount;not null" json:"task_amount"`
    LeafCount         uint      `gorm:"column:leaf_count" json:"leaf_count"`
    Enabled           bool      `gorm:"column:enabled;default:false" json:"enabled"`
    RootHasEnoughToken bool     `gorm:"column:root_has_enough_token;default:false" json:"root_has_enough_token"`
    Endpoint          string    `gorm:"column:endpoint;default:''" json:"endpoint"`
    CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
    WashMap           WashMap   `gorm:"foreignKey:MapID" json:"-"`
    Status            StatusType `gorm:"column:status;type:string;default:'unprocessed'" json:"status"`
}

// TableName 指定表名
func (WashTaskManage) TableName() string {
    return "wash_task_manage"
}