package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/solana"
	"marketcontrol/pkg/utils"
)

// AutoCreateWashMapRequest 表示自动创建洗币图谱和任务的请求
type AutoCreateWashMapRequest struct {
	ProjectID    uint   `json:"projectId" binding:"required"`
	ProjectLabel string `json:"projectLabel" binding:"required"`
	MapType      string `json:"mapType" binding:"required"`
	TargetRole   uint   `json:"targetRole" binding:"required"` // 新增：目标角色ID
	MapParams    struct {
		RootLabel   string `json:"rootLabel"`
		RootAddress string `json:"rootAddress"`
		Count       int    `json:"count"`
		Depth       int    `json:"depth"`
	} `json:"mapParams" binding:"required"`
	TaskParams struct {
		Gas         float64   `json:"gas"`
		Token       string    `json:"token"`
		Decimals    uint      `json:"decimals"`
		AmountArray []float64 `json:"amountArrary"`
		Endpoint    string    `json:"endpoint"`
	} `json:"taskParams" binding:"required"`
}

// AutoCreateWashMapV2Request 表示自动创建洗币图谱和任务的请求V2版本
type AutoCreateWashMapV2Request struct {
	ProjectID    uint   `json:"projectId"`
	ProjectLabel string `json:"projectLabel"`
	MapType      string `json:"mapType" binding:"required"`
	TargetRole   uint   `json:"targetRole" binding:"required"` // 目标角色ID
	MapParams    struct {
		RootLabel   string `json:"rootLabel"`
		RootAddress string `json:"rootAddress"`
		Count       int    `json:"count"`
		Depth       int    `json:"depth"`
	} `json:"mapParams" binding:"required"`
	TaskParams struct {
		Gas          float64   `json:"gas"`
		Token        string    `json:"token"`
		Decimals     uint      `json:"decimals"`
		AmountArray  []float64 `json:"amountArrary"`
		AddressArray []string  `json:"addressArray"` // 新增：指定的叶子节点地址数组
		Endpoint     string    `json:"endpoint"`
	} `json:"taskParams" binding:"required"`
}

// AutoCreateWashMapWithTask 自动创建洗币图谱和任务
func AutoCreateWashMapWithTask(c *gin.Context) {
	var req AutoCreateWashMapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 AmountArray 长度是否匹配 Count
	if len(req.TaskParams.AmountArray) != req.MapParams.Count {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AmountArray 长度必须与 Count 相同"})
		return
	}

	// 检查最近5分钟内是否有使用相同根节点地址的任务
	timeWindow := time.Now().Add(-2 * time.Minute)
	var recentTaskManages []models.WashTaskManage
	if err := dbconfig.DB.Where("created_at > ?", timeWindow).Find(&recentTaskManages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询最近任务失败: %v", err)})
		return
	}

	// 收集所有最近任务的 MapID
	var mapIDs []uint
	for _, tm := range recentTaskManages {
		mapIDs = append(mapIDs, tm.MapID)
	}

	// 如果有最近的任务，检查根节点地址
	if len(mapIDs) > 0 {
		var rootNodes []models.AddressNode
		if err := dbconfig.DB.Where("map_id IN ? AND node_type = ?", mapIDs, models.NodeTypeRoot).
			Find(&rootNodes).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询根节点失败: %v", err)})
			return
		}

		// 检查是否有重复的根节点地址
		for _, node := range rootNodes {
			if node.NodeValue == req.MapParams.RootAddress {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("根节点地址 %s 在最近30分钟内已被使用", req.MapParams.RootAddress),
				})
				return
			}
		}
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

	// 开启事务
	tx := dbconfig.DB.Begin()

	// 创建洗币图谱记录
	washMap := &models.WashMap{
		ProjectID:    req.ProjectID,
		ProjectLabel: req.ProjectLabel,
		MapType:      req.MapType,
		MapParams:    jsonMap,
	}

	if err := tx.Create(washMap).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 根据图谱类型生成节点和边
	var utilNodes []*utils.AddressNode
	switch req.MapType {
	case "linear":
		utilNodes = utils.BuildLinearChains(req.MapParams.RootLabel, req.MapParams.Count, req.MapParams.Depth)
	default:
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图谱类型"})
		return
	}

	// 使用 ExportAllNodes 获取所有节点
	allNodes := utils.ExportAllNodes(utilNodes)

	km := solana.NewKeyManager()

	// 首先创建所有节点，并保存节点ID映射
	nodeIDMap := make(map[string]uint)
	for _, node := range allNodes {
		var nodeValue string
		if node.Type == "root" {
			// 对于根节点，使用指定的地址
			nodeValue = req.MapParams.RootAddress
		} else {
			// 对于其他节点，生成新地址
			newAddress, err := GenerateSingleAddress(km)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("生成地址失败: %v", err)})
				return
			}
			nodeValue = newAddress.Address

			// 如果是叶子节点，保存到 RoleAddress
			if node.Type == "leaf" {
				roleAddress := &models.RoleAddress{
					RoleID:  req.TargetRole,
					Address: nodeValue,
				}
				if err := tx.Create(roleAddress).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存角色地址失败: %v", err)})
					return
				}
			}
		}

		addressNode := &models.AddressNode{
			MapID:       washMap.ID,
			NodeLabel:   node.ID,
			NodeValue:   nodeValue,
			NodeType:    models.NodeType(node.Type),
			NodeChainID: node.CountID,
			NodeDepthID: node.DepthID,
		}

		if err := tx.Create(addressNode).Error; err != nil {
			tx.Rollback()
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
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "节点ID映射失败"})
			return
		}

		addressEdge := &models.AddressEdge{
			FromNodeID: fromNodeID,
			ToNodeID:   toNodeID,
		}

		if err := tx.Create(addressEdge).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 创建任务管理记录
	taskManage := models.WashTaskManage{
		MapID:         washMap.ID,
		TaskCount:     uint(len(utilEdges)),
		TaskGas:       req.TaskParams.Gas,
		SendToken:     req.TaskParams.Token,
		TokenDecimals: req.TaskParams.Decimals,
		TaskAmount:    0, // 将在后面计算总金额
		LeafCount:     uint(req.MapParams.Count),
		Endpoint:      req.TaskParams.Endpoint,
		Enabled:       true, // 设置为启用状态
	}

	if err := tx.Create(&taskManage).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建任务管理记录失败: %v", err)})
		return
	}

	// 创建任务
	decimalsMultiplier := math.Pow10(int(req.TaskParams.Decimals))
	var totalAmount float64

	for i, edge := range utilEdges {
		// 验证节点是否存在
		var fromNode, toNode models.AddressNode
		if err := tx.First(&fromNode, nodeIDMap[edge.From]).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("来源节点不存在: %v", err)})
			return
		}
		if err := tx.First(&toNode, nodeIDMap[edge.To]).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("目标节点不存在: %v", err)})
			return
		}

		var sendAmount uint64
		var amountPerChain float64

		switch fromNode.NodeType {
		case models.NodeTypeRoot:
			// 根节点发送计算
			// 使用目标节点的 NodeChainID 来获取对应的金额
			if toNode.NodeChainID > 0 && toNode.NodeChainID <= len(req.TaskParams.AmountArray) {
				amountPerChain = req.TaskParams.AmountArray[toNode.NodeChainID-1]
			} else {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("目标节点的 NodeChainID %d 无效（数组长度为 %d）", toNode.NodeChainID, len(req.TaskParams.AmountArray))})
				return
			}
			sendAmount = uint64((amountPerChain - req.TaskParams.Gas) * decimalsMultiplier)
			totalAmount += amountPerChain
		case models.NodeTypeIntermediate:
			// 中间节点发送计算
			// 使用目标节点的 NodeChainID 来获取对应的金额
			if toNode.NodeChainID > 0 && toNode.NodeChainID <= len(req.TaskParams.AmountArray) {
				amountPerChain = req.TaskParams.AmountArray[toNode.NodeChainID-1]
			} else {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("目标节点的 NodeChainID %d 无效（数组长度为 %d）", toNode.NodeChainID, len(req.TaskParams.AmountArray))})
				return
			}
			gasEachDepth := float64(fromNode.NodeDepthID+1) * req.TaskParams.Gas
			sendAmount = uint64((amountPerChain - gasEachDepth) * decimalsMultiplier)
		case models.NodeTypeLeaf:
			// 叶子节点不允许发送
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "leaf 节点不能再发送，请检查 map"})
			return
		}

		task := models.WashTask{
			MapID:            washMap.ID,
			WashTaskManageID: taskManage.ID,
			SortID:           uint(i + 1), // 添加 SortID
			FromNodeID:       fromNode.ID,
			ToNodeID:         toNode.ID,
			FromAddress:      fromNode.NodeValue,
			ToAddress:        toNode.NodeValue,
			SendToken:        req.TaskParams.Token,
			TokenDecimals:    req.TaskParams.Decimals,
			SendAmount:       sendAmount,
			Gas:              req.TaskParams.Gas,
			Reverse:          false,
			FromNode:         fromNode,
			ToNode:           toNode,
		}

		if err := tx.Create(&task).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建任务失败: %v", err)})
			return
		}
	}

	// 更新任务管理记录的总金额
	if err := tx.Model(&taskManage).Update("task_amount", totalAmount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务管理总金额失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "创建成功",
		"data": gin.H{
			"task_manage_id": taskManage.ID,
			"map_id":         taskManage.MapID,
			"task_count":     taskManage.TaskCount,
			"task_gas":       taskManage.TaskGas,
			"send_token":     taskManage.SendToken,
			"token_decimals": taskManage.TokenDecimals,
			"task_amount":    totalAmount,
			"leaf_count":     taskManage.LeafCount,
		},
	})
}

// AutoCreateWashMapWithTaskV2 自动创建洗币图谱和任务V2版本（使用指定的叶子节点地址）
func AutoCreateWashMapWithTaskV2(c *gin.Context) {
	var req AutoCreateWashMapV2Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 AmountArray 和 AddressArray 长度是否一致
	if len(req.TaskParams.AmountArray) != len(req.TaskParams.AddressArray) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AmountArray 和 AddressArray 长度必须相同"})
		return
	}

	// 验证 AmountArray 长度是否匹配 Count
	if len(req.TaskParams.AmountArray) != req.MapParams.Count {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AmountArray 长度必须与 Count 相同"})
		return
	}

	// 检查最近 2 分钟内是否有使用相同根节点地址的任务
	timeWindow := time.Now().Add(-1 * time.Minute)
	var recentTaskManages []models.WashTaskManage
	if err := dbconfig.DB.Where("created_at > ?", timeWindow).Find(&recentTaskManages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询最近任务失败: %v", err)})
		return
	}

	// 收集所有最近任务的 MapID
	var mapIDs []uint
	for _, tm := range recentTaskManages {
		mapIDs = append(mapIDs, tm.MapID)
	}

	// 如果有最近的任务，检查根节点地址
	if len(mapIDs) > 0 {
		var rootNodes []models.AddressNode
		if err := dbconfig.DB.Where("map_id IN ? AND node_type = ?", mapIDs, models.NodeTypeRoot).
			Find(&rootNodes).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询根节点失败: %v", err)})
			return
		}

		// 检查是否有重复的根节点地址
		for _, node := range rootNodes {
			if node.NodeValue == req.MapParams.RootAddress {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("根节点地址 %s 在最近30分钟内已被使用", req.MapParams.RootAddress),
				})
				return
			}
		}
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

	// 开启事务
	tx := dbconfig.DB.Begin()

	// 创建洗币图谱记录
	washMap := &models.WashMap{
		ProjectID:    req.ProjectID,
		ProjectLabel: req.ProjectLabel,
		MapType:      req.MapType,
		MapParams:    jsonMap,
	}

	if err := tx.Create(washMap).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 根据图谱类型生成节点和边
	var utilNodes []*utils.AddressNode
	switch req.MapType {
	case "linear":
		utilNodes = utils.BuildLinearChains(req.MapParams.RootLabel, req.MapParams.Count, req.MapParams.Depth)
	default:
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的图谱类型"})
		return
	}

	// 使用 ExportAllNodes 获取所有节点
	allNodes := utils.ExportAllNodes(utilNodes)

	km := solana.NewKeyManager()

	// 首先创建所有节点，并保存节点ID映射
	nodeIDMap := make(map[string]uint)
	for _, node := range allNodes {
		var nodeValue string
		switch node.Type {
		case "root":
			// 对于根节点，使用指定的地址
			nodeValue = req.MapParams.RootAddress
		case "leaf":
			// 对于叶子节点，使用 AddressArray 中指定的地址
			if node.CountID > 0 && node.CountID <= len(req.TaskParams.AddressArray) {
				nodeValue = req.TaskParams.AddressArray[node.CountID-1]

				// 保存到 RoleAddress（如果不存在的话）
				var existingRoleAddress models.RoleAddress
				result := tx.Where("role_id = ? AND address = ?", req.TargetRole, nodeValue).First(&existingRoleAddress)
				if result.Error != nil {
					// 如果不存在，创建新的 RoleAddress
					roleAddress := &models.RoleAddress{
						RoleID:  req.TargetRole,
						Address: nodeValue,
					}
					if err := tx.Create(roleAddress).Error; err != nil {
						tx.Rollback()
						c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存角色地址失败: %v", err)})
						return
					}
				}
			} else {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("叶子节点的 NodeChainID %d 超出 AddressArray 范围", node.CountID)})
				return
			}
		default: // "intermediate"
			// 对于中间节点，生成新地址
			newAddress, err := GenerateSingleAddress(km)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("生成地址失败: %v", err)})
				return
			}
			nodeValue = newAddress.Address
		}

		addressNode := &models.AddressNode{
			MapID:       washMap.ID,
			NodeLabel:   node.ID,
			NodeValue:   nodeValue,
			NodeType:    models.NodeType(node.Type),
			NodeChainID: node.CountID,
			NodeDepthID: node.DepthID,
		}

		if err := tx.Create(addressNode).Error; err != nil {
			tx.Rollback()
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
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "节点ID映射失败"})
			return
		}

		addressEdge := &models.AddressEdge{
			FromNodeID: fromNodeID,
			ToNodeID:   toNodeID,
		}

		if err := tx.Create(addressEdge).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// 创建任务管理记录
	taskManage := models.WashTaskManage{
		MapID:         washMap.ID,
		TaskCount:     uint(len(utilEdges)),
		TaskGas:       req.TaskParams.Gas,
		SendToken:     req.TaskParams.Token,
		TokenDecimals: req.TaskParams.Decimals,
		TaskAmount:    0, // 将在后面计算总金额
		LeafCount:     uint(req.MapParams.Count),
		Endpoint:      req.TaskParams.Endpoint,
		Enabled:       true, // 设置为启用状态
	}

	if err := tx.Create(&taskManage).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建任务管理记录失败: %v", err)})
		return
	}

	// 创建任务
	decimalsMultiplier := math.Pow10(int(req.TaskParams.Decimals))
	var totalAmount float64

	for i, edge := range utilEdges {
		// 验证节点是否存在
		var fromNode, toNode models.AddressNode
		if err := tx.First(&fromNode, nodeIDMap[edge.From]).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("来源节点不存在: %v", err)})
			return
		}
		if err := tx.First(&toNode, nodeIDMap[edge.To]).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("目标节点不存在: %v", err)})
			return
		}

		var sendAmount uint64
		var amountPerChain float64

		switch fromNode.NodeType {
		case models.NodeTypeRoot:
			// 根节点发送计算
			// 使用目标节点的 NodeChainID 来获取对应的金额
			if toNode.NodeChainID > 0 && toNode.NodeChainID <= len(req.TaskParams.AmountArray) {
				amountPerChain = req.TaskParams.AmountArray[toNode.NodeChainID-1]
			} else {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("目标节点的 NodeChainID %d 无效（数组长度为 %d）", toNode.NodeChainID, len(req.TaskParams.AmountArray))})
				return
			}
			sendAmount = uint64((amountPerChain - req.TaskParams.Gas) * decimalsMultiplier)
			totalAmount += amountPerChain
		case models.NodeTypeIntermediate:
			// 中间节点发送计算
			// 使用目标节点的 NodeChainID 来获取对应的金额
			if toNode.NodeChainID > 0 && toNode.NodeChainID <= len(req.TaskParams.AmountArray) {
				amountPerChain = req.TaskParams.AmountArray[toNode.NodeChainID-1]
			} else {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("目标节点的 NodeChainID %d 无效（数组长度为 %d）", toNode.NodeChainID, len(req.TaskParams.AmountArray))})
				return
			}
			gasEachDepth := float64(fromNode.NodeDepthID+1) * req.TaskParams.Gas
			sendAmount = uint64((amountPerChain - gasEachDepth) * decimalsMultiplier)
		case models.NodeTypeLeaf:
			// 叶子节点不允许发送
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "leaf 节点不能再发送，请检查 map"})
			return
		}

		task := models.WashTask{
			MapID:            washMap.ID,
			WashTaskManageID: taskManage.ID,
			SortID:           uint(i + 1), // 添加 SortID
			FromNodeID:       fromNode.ID,
			ToNodeID:         toNode.ID,
			FromAddress:      fromNode.NodeValue,
			ToAddress:        toNode.NodeValue,
			SendToken:        req.TaskParams.Token,
			TokenDecimals:    req.TaskParams.Decimals,
			SendAmount:       sendAmount,
			Gas:              req.TaskParams.Gas,
			Reverse:          false,
			FromNode:         fromNode,
			ToNode:           toNode,
		}

		if err := tx.Create(&task).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建任务失败: %v", err)})
			return
		}
	}

	// 更新任务管理记录的总金额
	if err := tx.Model(&taskManage).Update("task_amount", totalAmount).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务管理总金额失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "提交事务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "创建成功",
		"data": gin.H{
			"task_manage_id": taskManage.ID,
			"map_id":         taskManage.MapID,
			"task_count":     taskManage.TaskCount,
			"task_gas":       taskManage.TaskGas,
			"send_token":     taskManage.SendToken,
			"token_decimals": taskManage.TokenDecimals,
			"task_amount":    totalAmount,
			"leaf_count":     taskManage.LeafCount,
		},
	})
}
