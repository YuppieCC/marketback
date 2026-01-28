package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/solana"
	"marketcontrol/pkg/utils"
	"os"
	"path/filepath"
)

// WashTaskWithBalance 包含余额信息的洗币任务响应结构体
type WashTaskWithBalance struct {
	models.WashTask
	FromAddressBalance *float64 `json:"from_address_balance,omitempty"`
	ToAddressBalance   *float64 `json:"to_address_balance,omitempty"`
}

type WashMapParams struct {
	RootLabel string `json:"rootLabel"`
	Count     int    `json:"count"`
	Depth     int    `json:"depth"`
}

// CreateWashMap 创建洗币图谱
func CreateWashMap(c *gin.Context) {
	var req struct {
		ProjectID    uint          `json:"project_id" binding:"required"`
		ProjectLabel string        `json:"project_label" binding:"required"`
		MapType      string        `json:"map_type" binding:"required"`
		MapParams    WashMapParams `json:"map_params" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 将 MapParams 转换为 JSONMap
	paramsBytes, err := json.Marshal(req.MapParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "参数序列化失败"})
		return
	}

	var jsonMap models.JSONMap
	if err := json.Unmarshal(paramsBytes, &jsonMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "参数反序列化失败"})
		return
	}

	// 创建洗币图谱记录
	washMap := &models.WashMap{
		ProjectID:    req.ProjectID,
		ProjectLabel: req.ProjectLabel,
		MapType:      req.MapType,
		MapParams:    jsonMap,
	}

	if err := dbconfig.DB.Create(washMap).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 根据图谱类型生成节点和边
	var utilNodes []*utils.AddressNode
	switch req.MapType {
	case "linear":
		utilNodes = utils.BuildLinearChains(req.MapParams.RootLabel, req.MapParams.Count, req.MapParams.Depth)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图谱类型"})
		return
	}

	// 使用 ExportAllNodes 获取所有节点
	allNodes := utils.ExportAllNodes(utilNodes)

	km := solana.NewKeyManager()

	// 首先创建所有节点，并保存节点ID映射
	nodeIDMap := make(map[string]uint)
	for _, node := range allNodes {
		newAddress, err := GenerateSingleAddress(km)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("生成地址失败: %v", err)})
			return
		}
		// 在创建节点时使用新生成的地址
		addressNode := &models.AddressNode{
			MapID:       washMap.ID,
			NodeLabel:   node.ID,
			NodeValue:   newAddress.Address,
			NodeType:    models.NodeType(node.Type),
			NodeChainID: node.CountID, // 新增：设置链路ID
			NodeDepthID: node.DepthID, // 新增：设置深度ID
		}

		if err := dbconfig.DB.Create(addressNode).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		nodeIDMap[node.ID] = addressNode.ID
	}

	// 使用 GenerateEdges 生成所有边的关系
	utilEdges := utils.GenerateEdges(utilNodes)
	for _, edge := range utilEdges {
		fromNodeID, fromExists := nodeIDMap[edge.From]
		toNodeID, toExists := nodeIDMap[edge.To]

		if !fromExists || !toExists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "节点ID映射失败"})
			return
		}

		addressEdge := &models.AddressEdge{
			FromNodeID: fromNodeID,
			ToNodeID:   toNodeID,
		}

		if err := dbconfig.DB.Create(addressEdge).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 重新加载完整的 washMap 数据
	if err := dbconfig.DB.First(&washMap, washMap.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载创建的图谱失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "创建成功",
		"data":    washMap,
	})
}

// GetWashMap 返回指定ID的洗币图谱
func GetWashMap(c *gin.Context) {
	id := c.Param("id")

	var washMap models.WashMap
	if err := dbconfig.DB.First(&washMap, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录未找到"})
		return
	}
	c.JSON(http.StatusOK, washMap)
}

// ListWashMapNodes 返回指定图谱的所有节点
func ListWashMapNodes(c *gin.Context) {
	mapID := c.Param("id")

	var nodes []models.AddressNode
	if err := dbconfig.DB.Where("map_id = ?", mapID).Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

// ListWashMapEdges 返回指定图谱的所有边
func ListWashMapEdges(c *gin.Context) {
	mapID := c.Param("id")

	var edges []models.AddressEdge
	if err := dbconfig.DB.Where(
		"from_node_id IN (SELECT id FROM address_nodes WHERE map_id = ?) AND to_node_id IN (SELECT id FROM address_nodes WHERE map_id = ?)",
		mapID, mapID).Preload("FromNode").Preload("ToNode").Find(&edges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, edges)
}

// DeleteWashMap 删除洗币图谱及其关联的节点和边
func DeleteWashMap(c *gin.Context) {
	id := c.Param("id")

	// 开启事务
	tx := dbconfig.DB.Begin()

	// 删除关联的边
	if err := tx.Where("from_node_id IN (SELECT id FROM address_nodes WHERE map_id = ?) OR to_node_id IN (SELECT id FROM address_nodes WHERE map_id = ?)", id, id).
		Delete(&models.AddressEdge{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 删除节点
	if err := tx.Where("map_id = ?", id).Delete(&models.AddressNode{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 删除图谱
	if err := tx.Delete(&models.WashMap{}, id).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 提交事务
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "图谱及关联数据删除成功"})
}

// CreateWashTaskRequest 表示创建洗币任务的请求
type CreateWashTaskRequest struct {
	MapID    uint    `json:"map_id" binding:"required"`
	Gas      float64 `json:"gas" binding:"required"`
	Token    string  `json:"token" binding:"required"`
	Decimals uint    `json:"decimals" binding:"required"`
	Amount   float64 `json:"amount" binding:"required"`
	Endpoint string  `json:"endpoint" binding:"required"`
}

// CreateWashTask 创建洗币任务
func CreateWashTask(c *gin.Context) {
	var req CreateWashTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 开启事务
	tx := dbconfig.DB.Begin()

	// 获取所有叶子节点
	var leafNodes []models.AddressNode
	if err := tx.Where("map_id = ? AND node_type = ?", req.MapID, models.NodeTypeLeaf).Find(&leafNodes).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取叶子节点失败"})
		return
	}

	if len(leafNodes) == 0 {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到叶子节点"})
		return
	}

	// 计算每个叶子节点应该收到的金额（扣除 gas 后平均分配）
	totalGas := req.Gas * float64(len(leafNodes))
	if req.Amount <= totalGas {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "金额不足以支付 gas"})
		return
	}

	// 考虑代币小数位数进行计算
	decimalsMultiplier := math.Pow10(int(req.Decimals))
	// Get the wash map data first
	var washMap models.WashMap
	if err := tx.First(&washMap, req.MapID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Wash map not found"})
		return
	}

	// 直接使用 MapParams，因为它已经是 JSONMap 类型
	mapParams := washMap.MapParams
	mapChainCount, ok := mapParams["count"].(float64)
	if !ok {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取图谱链路数量"})
		return
	}

	// 计算每条链路理论分配的金额
	amountPerLeaf := float64(req.Amount) / float64(mapChainCount)

	// 获取所有边关系
	var edges []models.AddressEdge
	if err := tx.Where("from_node_id IN (SELECT id FROM address_nodes WHERE map_id = ?) OR to_node_id IN (SELECT id FROM address_nodes WHERE map_id = ?)",
		req.MapID, req.MapID).Preload("FromNode").Preload("ToNode").Find(&edges).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取边关系失败"})
		return
	}

	// 创建任务管理记录
	taskManage := models.WashTaskManage{
		MapID:         req.MapID,
		TaskCount:     uint(len(edges)),
		TaskGas:       req.Gas,
		SendToken:     req.Token,
		TokenDecimals: req.Decimals,
		TaskAmount:    req.Amount,
		LeafCount:     uint(len(leafNodes)),
		Endpoint:      req.Endpoint,
	}

	if err := tx.Create(&taskManage).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建任务管理记录失败: %v", err)})
		return
	}

	// 创建任务
	var tasks []models.WashTask
	decimalsMultiplier = math.Pow10(int(req.Decimals))

	for i, edge := range edges {
		// 验证节点是否存在
		var fromNode, toNode models.AddressNode
		if err := tx.First(&fromNode, edge.FromNodeID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("来源节点不存在: %v", err)})
			return
		}
		if err := tx.First(&toNode, edge.ToNodeID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("目标节点不存在: %v", err)})
			return
		}

		var sendAmount uint64

		switch edge.FromNode.NodeType {
		case models.NodeTypeRoot:
			// 根节点发送计算
			sendAmount = uint64((amountPerLeaf - req.Gas) * decimalsMultiplier)
		case models.NodeTypeIntermediate:
			// 中间节点发送计算
			gasEachDepth := float64(edge.FromNode.NodeDepthID+1) * req.Gas
			sendAmount = uint64((amountPerLeaf - gasEachDepth) * decimalsMultiplier)
		case models.NodeTypeLeaf:
			// 叶子节点不允许发送
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "leaf 节点不能再发送，请检查 map"})
			return
		}

		task := models.WashTask{
			MapID:            req.MapID,
			WashTaskManageID: taskManage.ID,
			SortID:           uint(i + 1),
			FromNodeID:       edge.FromNodeID,
			ToNodeID:         edge.ToNodeID,
			FromAddress:      edge.FromNode.NodeValue,
			ToAddress:        edge.ToNode.NodeValue,
			SendToken:        req.Token,
			TokenDecimals:    req.Decimals,
			SendAmount:       sendAmount,
			Gas:              req.Gas,
			Reverse:          false,
			FromNode:         edge.FromNode,
			ToNode:           edge.ToNode,
		}

		if err := tx.Create(&task).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建任务失败: %v", err)})
			return
		}
		tasks = append(tasks, task)
	}

	// 重新加载任务数据以确保关联数据完整
	for i := range tasks {
		if err := tx.Preload("FromNode").Preload("ToNode").First(&tasks[i], tasks[i].ID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载任务数据失败"})
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "创建任务成功",
		"data": gin.H{
			"task_manage_id": taskManage.ID, // 添加
			"map_id":         taskManage.MapID,
			"task_count":     taskManage.TaskCount,
			"task_gas":       taskManage.TaskGas,
			"send_token":     taskManage.SendToken,
			"token_decimals": taskManage.TokenDecimals,
			"task_amount":    taskManage.TaskAmount,
			"leaf_count":     taskManage.LeafCount,
			"tasks_len":      len(tasks),
		},
	})
}

// GetWashTaskByManageID 返回指定任务管理ID的所有任务
func GetWashTaskByManageID(c *gin.Context) {
	manageID := c.Param("WashTaskManageID")

	// 获取分页参数
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "10")

	// 获取排序参数
	orderField := c.DefaultQuery("order_field", "sort_id")
	orderType := c.DefaultQuery("order_type", "asc")

	// 解析分页参数
	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	pageSizeNum, err := strconv.Atoi(pageSize)
	if err != nil || pageSizeNum < 1 || pageSizeNum > 100 {
		pageSizeNum = 10
	}

	// 验证排序字段
	allowedOrderFields := map[string]string{
		"id":                  "id",
		"map_id":              "map_id",
		"wash_task_manage_id": "wash_task_manage_id",
		"sort_id":             "sort_id",
		"from_node_id":        "from_node_id",
		"to_node_id":          "to_node_id",
		"from_address":        "from_address",
		"to_address":          "to_address",
		"send_token":          "send_token",
		"token_decimals":      "token_decimals",
		"send_amount":         "send_amount",
		"gas":                 "gas",
		"reverse":             "reverse",
		"signature":           "signature",
		"is_success":          "is_success",
		"created_at":          "created_at",
		"updated_at":          "updated_at",
		"status":              "status",
	}

	dbOrderField, exists := allowedOrderFields[orderField]
	if !exists {
		dbOrderField = "sort_id" // 默认排序字段
	}

	// 验证排序类型
	orderType = strings.ToLower(orderType)
	if orderType != "asc" && orderType != "desc" {
		orderType = "asc" // 默认升序
	}

	// 构建排序字符串
	orderClause := fmt.Sprintf("%s %s", dbOrderField, orderType)

	// 计算偏移量
	offset := (pageNum - 1) * pageSizeNum

	// 获取总记录数
	var total int64
	if err := dbconfig.DB.Model(&models.WashTask{}).Where("wash_task_manage_id = ?", manageID).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 查询分页数据
	var tasks []models.WashTask
	if err := dbconfig.DB.Where("wash_task_manage_id = ?", manageID).
		Preload("FromNode").Preload("ToNode").Preload("WashTaskManage").
		Order(orderClause).
		Offset(offset).Limit(pageSizeNum).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 创建包含余额信息的响应切片
	var tasksWithBalance []WashTaskWithBalance

	for _, task := range tasks {
		taskWithBalance := WashTaskWithBalance{
			WashTask: task,
		}

		// 查询 from_address 的余额
		var fromBalance models.WalletTokenStat
		if err := dbconfig.DB.Where("owner_address = ? AND mint = ?",
			task.FromAddress, task.WashTaskManage.SendToken).First(&fromBalance).Error; err == nil {
			taskWithBalance.FromAddressBalance = &fromBalance.BalanceReadable
		} else {
			// 如果数据不存在，设置为 0
			zeroBalance := 0.0
			taskWithBalance.FromAddressBalance = &zeroBalance
		}

		// 查询 to_address 的余额
		var toBalance models.WalletTokenStat
		if err := dbconfig.DB.Where("owner_address = ? AND mint = ?",
			task.ToAddress, task.WashTaskManage.SendToken).First(&toBalance).Error; err == nil {
			taskWithBalance.ToAddressBalance = &toBalance.BalanceReadable
		} else {
			// 如果数据不存在，设置为 0
			zeroBalance := 0.0
			taskWithBalance.ToAddressBalance = &zeroBalance
		}

		tasksWithBalance = append(tasksWithBalance, taskWithBalance)
	}

	// 计算分页信息
	totalPages := int(math.Ceil(float64(total) / float64(pageSizeNum)))
	hasNext := pageNum < totalPages
	hasPrev := pageNum > 1

	c.JSON(http.StatusOK, gin.H{
		"data": tasksWithBalance,
		"pagination": gin.H{
			"page":        pageNum,
			"page_size":   pageSizeNum,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
		"sorting": gin.H{
			"order_field": orderField,
			"order_type":  orderType,
		},
	})
}

// GetWashTaskManage 返回指定ID的洗币任务管理记录
func GetWashTaskManage(c *gin.Context) {
	manageID := c.Param("WashTaskManageID")

	var taskManage models.WashTaskManage
	if err := dbconfig.DB.Preload("WashMap").First(&taskManage, manageID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录未找到"})
		return
	}
	c.JSON(http.StatusOK, taskManage)
}

// ExportWashMapRequest 表示导出洗币图谱地址的请求
type ExportWashMapRequest struct {
	MapID    uint   `json:"map_id" binding:"required"`
	NodeType string `json:"node_type" binding:"required"`
}

// ExportWashMap 导出指定图谱的地址
func ExportWashMap(c *gin.Context) {
	var req ExportWashMapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查询符合条件的节点
	var nodes []models.AddressNode
	if err := dbconfig.DB.Where("map_id = ? AND node_type = ?", req.MapID, req.NodeType).Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询节点失败: %v", err)})
		return
	}

	if len(nodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到符合条件的节点"})
		return
	}

	// 创建结果映射
	result := make(map[string]string)

	// 遍历节点，查询对应的地址管理信息
	for _, node := range nodes {
		var addressManage models.AddressManage
		if err := dbconfig.DB.Where("address = ?", node.NodeValue).First(&addressManage).Error; err != nil {
			continue
		}
		result[addressManage.Address] = addressManage.PrivateKey
	}

	if len(result) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到任何匹配的地址管理信息"})
		return
	}

	// 确保输出目录存在
	outputDir := "output/address_keystore"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建输出目录失败: %v", err)})
		return
	}

	// 构建输出文件路径
	timestamp := time.Now().Format("20060102150405")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("%d_%s_%s.json", req.MapID, req.NodeType, timestamp))

	// 将结果写入文件
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("JSON编码失败: %v", err)})
		return
	}

	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("写入文件失败: %v", err)})
		return
	}

	// 设置响应头，使文件可下载
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%d_%s_%s.json", req.MapID, req.NodeType, timestamp))
	c.Header("Content-Type", "application/json")
	c.File(outputPath)
}

// UpdateWashTaskManageRequest 表示更新洗币任务的请求
type UpdateWashTaskManageRequest struct {
	TaskGas  *float64 `json:"task_gas"`
	Enabled  *bool    `json:"enabled"`
	Endpoint *string  `json:"endpoint"`
	Retry    *bool    `json:"retry"`
}

// UpdateWashTaskManagByID 更新洗币任务
func UpdateWashTaskManagByID(c *gin.Context) {
	manageID := c.Param("WashTaskManageID")

	var req UpdateWashTaskManageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证至少有一个参数
	if req.TaskGas == nil && req.Enabled == nil && req.Endpoint == nil && req.Retry == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要提供一个更新参数"})
		return
	}

	// 开启事务
	tx := dbconfig.DB.Begin()

	// 获取任务管理记录
	var taskManage models.WashTaskManage
	if err := tx.First(&taskManage, manageID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "任务管理记录未找到"})
		return
	}

	// 如果是重试操作
	if req.Retry != nil && *req.Retry {
		// 重置所有相关任务的成功状态
		if err := tx.Model(&models.WashTask{}).Where("wash_task_manage_id = ?", manageID).
			Update("is_success", false).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "重置任务状态失败"})
			return
		}
	}

	// 更新任务管理记录
	if req.TaskGas != nil {
		taskManage.TaskGas = *req.TaskGas
		// 更新所有相关任务的 Gas
		if err := tx.Model(&models.WashTask{}).Where("wash_task_manage_id = ?", manageID).
			Update("gas", *req.TaskGas).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务 Gas 失败"})
			return
		}
	}

	if req.Enabled != nil {
		taskManage.Enabled = *req.Enabled
		// 新增逻辑：如果禁用，则重置状态
		if !*req.Enabled {
			taskManage.Status = models.StatusUnprocessed
			// 更新所有 status 为 processing 的 WashTask 为 unprocessed
			if err := tx.Model(&models.WashTask{}).
				Where("wash_task_manage_id = ? AND status = ?", manageID, models.StatusProcessing).
				Update("status", models.StatusUnprocessed).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "重置任务状态失败"})
				return
			}
		}
	}

	if req.Endpoint != nil {
		taskManage.Endpoint = *req.Endpoint
	}

	// 保存任务管理记录的更改
	if err := tx.Save(&taskManage).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务管理记录失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"data":    taskManage,
	})
}

// UpdateTaskRequest 表示更新单个任务的请求
type UpdateTaskRequest struct {
	SendAmount *uint64  `json:"send_amount"`
	Gas        *float64 `json:"gas"`
}

// UpdateTaskByID 更新单个任务
func UpdateTaskByID(c *gin.Context) {
	taskID := c.Param("WashTaskID")

	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证至少有一个参数
	if req.SendAmount == nil && req.Gas == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要提供一个更新参数"})
		return
	}

	// 开启事务
	tx := dbconfig.DB.Begin()

	// 获取任务记录
	var task models.WashTask
	if err := tx.First(&task, taskID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "任务记录未找到"})
		return
	}

	// 更新任务记录
	if req.SendAmount != nil {
		task.SendAmount = *req.SendAmount
	}
	if req.Gas != nil {
		task.Gas = *req.Gas
	}

	// 保存任务记录的更改
	if err := tx.Save(&task).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务记录失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
		"data":    task,
	})
}

// GetWashTask 返回指定ID的洗币任务
func GetWashTask(c *gin.Context) {
	taskID := c.Param("WashTaskID")

	var task models.WashTask
	if err := dbconfig.DB.Preload("FromNode").Preload("ToNode").First(&task, taskID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务记录未找到"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// ListWashMapsByProjectIDRequest 表示按项目ID查询洗币图谱的请求
type ListWashMapsByProjectIDRequest struct {
	ProjectID uint `json:"project_id"`
}

// ListWashMapsByProjectID 返回指定项目ID的所有洗币图谱列表
func ListWashMapsByProjectID(c *gin.Context) {
	var req ListWashMapsByProjectIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var washMaps []models.WashMap
	if err := dbconfig.DB.Where("project_id = ?", req.ProjectID).Find(&washMaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, washMaps)
}

// ListWashTaskManageByProjectIDRequest 表示获取洗币任务管理列表的请求
type ListWashTaskManageByProjectIDRequest struct {
	ProjectID uint `json:"project_id"`
}

// ListWashTaskManageByProjectID 根据项目ID获取所有相关的洗币任务管理记录
func ListWashTaskManageByProjectID(c *gin.Context) {
	var req ListWashTaskManageByProjectIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 首先查找所有符合 ProjectID 的 WashMap
	var washMaps []models.WashMap
	if err := dbconfig.DB.Where("project_id = ?", req.ProjectID).Find(&washMaps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(washMaps) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到相关的洗币图谱"})
		return
	}

	// 获取所有 WashMap 的 ID
	var mapIDs []uint
	for _, wm := range washMaps {
		mapIDs = append(mapIDs, wm.ID)
	}

	// 查询所有相关的洗币任务管理记录
	var taskManages []models.WashTaskManage
	if err := dbconfig.DB.Where("map_id IN ?", mapIDs).
		Preload("WashMap").
		Find(&taskManages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, taskManages)

}
