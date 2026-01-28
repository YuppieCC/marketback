package handlers

import (
	"errors"
	"math"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"marketcontrol/internal/handlers/business"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// ListWalletTokenStats 获取所有钱包代币统计信息
func ListWalletTokenStats(c *gin.Context) {
	var stats []models.WalletTokenStat
	if err := dbconfig.DB.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetWalletTokenStat 获取指定ID的钱包代币统计信息
func GetWalletTokenStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.WalletTokenStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetWalletTokenStatsByAddress 获取指定地址的所有代币统计信息
func GetWalletTokenStatsByAddress(c *gin.Context) {
	address := c.Param("address")
	var stats []models.WalletTokenStat
	if err := dbconfig.DB.Where("owner_address = ?", address).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetWalletTokenStatsByMint 获取指定代币的所有钱包统计信息
func GetWalletTokenStatsByMint(c *gin.Context) {
	mint := c.Param("mint")
	var stats []models.WalletTokenStat
	if err := dbconfig.DB.Where("mint = ?", mint).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ListPoolStats 获取所有池子统计信息
func ListPoolStats(c *gin.Context) {
	var stats []models.PoolStat
	if err := dbconfig.DB.Preload("Pool").Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetPoolStat 获取指定ID的池子统计信息
func GetPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.PoolStat
	if err := dbconfig.DB.Preload("Pool").First(&stat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetPoolStatsByPoolID 获取指定池子ID的所有统计信息
func GetPoolStatsByPoolID(c *gin.Context) {
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	var stats []models.PoolStat
	if err := dbconfig.DB.Preload("Pool").Where("pool_id = ?", poolID).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetPoolStatByProjectID 根据 project_id 获取对应的池子统计信息
func GetPoolStatByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var project models.ProjectConfig
	if err := dbconfig.DB.Where("id = ?", projectID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	switch project.PoolPlatform {
	case "raydium":
		var poolStat models.PoolStat
		if err := dbconfig.DB.Preload("Pool").Where("pool_id = ?", project.PoolID).First(&poolStat).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "PoolStat not found"})
			return
		}
		resp := BuildPoolStatResp(&poolStat)
		if resp != nil && resp.Pool != nil {
			// 将 platform 移到上层
			resp.Platform = resp.Pool.Platform
			resp.Pool.Platform = ""
		}
		c.JSON(http.StatusOK, resp)

	case "pumpfun_internal":
		var pumpfunStat models.PumpfuninternalStat
		if err := dbconfig.DB.Preload("PumpfunPool").
			Where("pumpfuninternal_id = ?", project.PoolID).
			Order("block_time DESC").
			First(&pumpfunStat).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "PumpfuninternalStat not found"})
			return
		}
		resp := BuildPumpfuninternalStatRespSimple(&pumpfunStat)
		resp.Platform = "pumpfun_internal"
		c.JSON(http.StatusOK, resp)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
	}
}

// GetWalletTokenStatsByRole 根据 role_id 获取该角色下所有地址的钱包代币统计信息，支持分页
func GetWalletTokenStatsByRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var req struct {
		Tokens     []string `json:"tokens"`
		OrderField string   `json:"order_field"`
		OrderType  string   `json:"order_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 验证 order_type 参数
	if req.OrderType != "" && req.OrderType != "asc" && req.OrderType != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_type must be 'asc' or 'desc'"})
		return
	}

	// 获取角色下的所有地址
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total":     0,
			"page":      page,
			"page_size": pageSize,
			"data":      []interface{}{},
		})
		return
	}

	// 提取所有地址
	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}

	// 查询所有地址的代币统计信息
	var stats []models.WalletTokenStat
	query := dbconfig.DB.Where("owner_address IN ?", addresses)
	if len(req.Tokens) > 0 {
		query = query.Where("mint IN ?", req.Tokens)
	}
	if err := query.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 分组：按地址分组代币统计
	statMap := make(map[string]map[string]models.WalletTokenStat)
	for _, stat := range stats {
		if statMap[stat.OwnerAddress] == nil {
			statMap[stat.OwnerAddress] = make(map[string]models.WalletTokenStat)
		}
		statMap[stat.OwnerAddress][stat.Mint] = stat
	}

	var result []TokenGroup
	for _, address := range addresses {
		var tokenList []WalletTokenStatResp

		// 确保返回所有请求的代币，即使余额为0
		for _, mint := range req.Tokens {
			if stat, exists := statMap[address][mint]; exists {
				// 存在该代币的统计数据
				tokenList = append(tokenList, WalletTokenStatResp{
					Mint:            stat.Mint,
					Decimals:        stat.Decimals,
					Balance:         stat.Balance,
					BalanceReadable: stat.BalanceReadable,
					Slot:            stat.Slot,
					BlockTime:       stat.BlockTime.UnixMilli(),
					CreatedAt:       stat.CreatedAt.UnixMilli(),
					UpdatedAt:       stat.UpdatedAt.UnixMilli(),
				})
			} else {
				// 不存在该代币的统计数据，返回0余额
				tokenList = append(tokenList, WalletTokenStatResp{
					Mint:            mint,
					Decimals:        0,
					Balance:         0,
					BalanceReadable: 0,
					Slot:            0,
					BlockTime:       0,
					CreatedAt:       0,
					UpdatedAt:       0,
				})
			}
		}

		result = append(result, TokenGroup{
			OwnerAddress: address,
			Tokens:       tokenList,
		})
	}

	// 排序逻辑
	if req.OrderField != "" {
		sort.Slice(result, func(i, j int) bool {
			// 获取指定 mint 的余额进行排序
			balanceI := getTokenBalanceByMint(result[i].Tokens, req.OrderField)
			balanceJ := getTokenBalanceByMint(result[j].Tokens, req.OrderField)

			if req.OrderType == "desc" {
				return balanceI > balanceJ
			}
			return balanceI < balanceJ // 默认 asc
		})
	}

	// 计算分页
	total := int64(len(result))
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}
	if start >= len(result) {
		c.JSON(http.StatusOK, gin.H{
			"total":     total,
			"page":      page,
			"page_size": pageSize,
			"data":      []interface{}{},
		})
		return
	}

	// 获取当前页的数据
	pagedResult := result[start:end]

	// 返回分页结果
	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      pagedResult,
	})
}

// getTokenBalanceByMint 根据 mint 获取代币余额，用于排序
func getTokenBalanceByMint(tokens []WalletTokenStatResp, mint string) float64 {
	for _, token := range tokens {
		if token.Mint == mint {
			return token.BalanceReadable
		}
	}
	return 0 // 如果没找到该代币，返回 0
}

// GetTotalWalletTokenStatsByRole 根据 role_id 获取该角色所有的 WalletTokenStat 数据，返回全部数据不分页
func GetTotalWalletTokenStatsByRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var req struct {
		Tokens []string `json:"tokens"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 获取角色下的所有地址
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"total": 0,
			"data":  []interface{}{},
		})
		return
	}

	// 提取所有地址列表
	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}

	// 查询这些地址的代币统计信息
	var stats []models.WalletTokenStat
	query := dbconfig.DB.Where("owner_address IN ?", addresses)
	if len(req.Tokens) > 0 {
		query = query.Where("mint IN ?", req.Tokens)
	}
	if err := query.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 分组
	resultMap := make(map[string][]models.WalletTokenStat)
	for _, stat := range stats {
		resultMap[stat.OwnerAddress] = append(resultMap[stat.OwnerAddress], stat)
	}

	var result []TokenGroup
	for owner, tokens := range resultMap {
		var tokenList []WalletTokenStatResp
		for _, t := range tokens {
			tokenList = append(tokenList, WalletTokenStatResp{
				Mint:            t.Mint,
				Decimals:        t.Decimals,
				Balance:         t.Balance,
				BalanceReadable: t.BalanceReadable,
				Slot:            t.Slot,
				BlockTime:       t.BlockTime.UnixMilli(),
				CreatedAt:       t.CreatedAt.UnixMilli(),
				UpdatedAt:       t.UpdatedAt.UnixMilli(),
			})
		}
		result = append(result, TokenGroup{
			OwnerAddress: owner,
			Tokens:       tokenList,
		})
	}

	// 返回全部数据
	c.JSON(http.StatusOK, gin.H{
		"total": len(addresses),
		"data":  result,
	})
}

// GetAggregateWalletTokenStatsByRole 根据 role_id 聚合所有地址的代币余额
func GetAggregateWalletTokenStatsByRole(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var req struct {
		Tokens []string `json:"tokens"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", roleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}

	var stats []models.WalletTokenStat
	query := dbconfig.DB.Where("owner_address IN ?", addresses)
	if len(req.Tokens) > 0 {
		query = query.Where("mint IN ?", req.Tokens)
	}
	if err := query.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tokenMap := AggregateTokenStats(stats)

	var result []AggregateTokenStat
	for _, v := range tokenMap {
		result = append(result, *v)
	}

	c.JSON(http.StatusOK, result)
}

// GetAggregateWalletTokenStatsByProject 根据 project_id 聚合所有角色所有地址的代币余额
func GetAggregateWalletTokenStatsByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var req struct {
		Tokens []string `json:"tokens"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 查找所有角色下的地址，去重
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Table("role_address").
		Select("DISTINCT address").
		Joins("JOIN role_config ON role_address.role_id = role_config.id").
		Where("role_config.project_id = ?", projectID).
		Scan(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(roleAddresses) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}

	// 查询这些地址下的 tokens
	var stats []models.WalletTokenStat
	query := dbconfig.DB.Where("owner_address IN ?", addresses)
	if len(req.Tokens) > 0 {
		query = query.Where("mint IN ?", req.Tokens)
	}
	if err := query.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tokenMap := AggregateTokenStats(stats)

	var result []AggregateTokenStat
	for _, mint := range req.Tokens {
		if agg, ok := tokenMap[mint]; ok {
			result = append(result, *agg)
		} else {
			result = append(result, AggregateTokenStat{
				Mint:            mint,
				Decimals:        0,
				Balance:         0,
				BalanceReadable: 0,
			})
		}
	}

	c.JSON(http.StatusOK, result)
}

// GetSettleStatsByProject 根据项目配置获取结算统计
func GetSettleStatsByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// 1. 获取项目、token信息
	var project models.ProjectConfig
	if err := dbconfig.DB.Preload("Token").Where("id = ?", projectID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}
	if project.Token == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project token not found"})
		return
	}

	// 根据不同的池子平台调用不同的结算逻辑
	switch project.PoolPlatform {
	// case "raydium":
	// 	getSettleStatsByRaydiumPool(c, project)
	case "pumpfun_internal":
		getSettleStatsByPumpfunPool(c, project)
	case "pumpfun_amm":
		getSettleStatsByPumpfunAmmPool(c, project)
	case "meteora_dbc":
		getSettleStatsByMeteoradbcPool(c, project)
	case "meteora_cpmm":
		getSettleStatsByMeteoracpmmPool(c, project)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
	}
}

// GetRetailSolAmountByProject 计算并返回指定项目的散户 SOL 金额
func GetRetailSolAmountByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var project models.ProjectConfig
	if err := dbconfig.DB.Where("id = ?", projectID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	retailSol := 0.0

	switch project.PoolPlatform {
	case "meteora_cpmm":
		var pool models.MeteoracpmmConfig
		if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "MeteoracpmmConfig not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query MeteoracpmmConfig"})
			return
		}

		var holders []models.MeteoracpmmHolder
		if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?", "retail_investors", pool.PoolAddress).
			Find(&holders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query holders"})
			return
		}

		for _, h := range holders {
			// 忽略名单
			ignore := false
			for _, ig := range business.IGNORE_METEORA_RETAIL_ADDRESS {
				if h.Address == ig {
					ignore = true
					break
				}
			}
			if ignore {
				continue
			}

			baseChange := h.BaseChange
			quoteChange := h.QuoteChange
			if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
				baseChange = 0.0
			}
			if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
				quoteChange = 0.0
			}
			if baseChange > 0 {
				retailSol += math.Abs(quoteChange)
			}
		}

	case "meteora_dbc":
		var pool models.MeteoradbcConfig
		if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "MeteoradbcConfig not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query MeteoradbcConfig"})
			return
		}

		var holders []models.MeteoradbcHolder
		if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?", "retail_investors", pool.PoolAddress).
			Find(&holders).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query holders"})
			return
		}

		for _, h := range holders {
			// 忽略名单
			ignore := false
			for _, ig := range business.IGNORE_METEORA_RETAIL_ADDRESS {
				if h.Address == ig {
					ignore = true
					break
				}
			}
			if ignore {
				continue
			}

			baseChange := h.BaseChange
			quoteChange := h.QuoteChange
			if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
				baseChange = 0.0
			}
			if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
				quoteChange = 0.0
			}
			if baseChange > 0 {
				retailSol += math.Abs(quoteChange)
			}
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project_id":        projectID,
		"pool_platform":     project.PoolPlatform,
		"pool_id":           project.PoolID,
		"retail_sol_amount": retailSol,
	})
}

// getSettleStatsByPumpfunPool 处理 Pumpfun Internal 池子的结算统计
func getSettleStatsByPumpfunPool(c *gin.Context, project models.ProjectConfig) {
	// 获取池子统计
	var pumpfunStat models.PumpfuninternalStat
	if err := dbconfig.DB.Preload("PumpfunPool").
		Where("pumpfuninternal_id = ?", project.PoolID).
		First(&pumpfunStat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PumpfuninternalStat not found"})
		return
	}

	// 调用业务逻辑计算结算数据
	settleStats, err := business.CalculatePumpfunPoolSettle(&project, &pumpfunStat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建响应
	c.JSON(http.StatusOK, gin.H{
		"pool_stat": BuildPumpfuninternalStatRespSimple(&pumpfunStat),
		"settle":    settleStats,
	})
}

// getSettleStatsByPumpfunAmmPool 处理 Pumpfun AMM 池子的结算统计
func getSettleStatsByPumpfunAmmPool(c *gin.Context, project models.ProjectConfig) {
	// 获取池子统计
	var pumpfunStat models.PumpfunAmmPoolStat
	if err := dbconfig.DB.Where("pool_id = ?", project.PoolID).First(&pumpfunStat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PumpfunAmmPoolStat not found"})
		return
	}

	// 获取 PumpfunAmmPoolConfig 来获取 PoolAddress
	var poolConfig models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.Where("id = ?", pumpfunStat.PoolID).First(&poolConfig).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PumpfunAmmPoolConfig not found"})
		return
	}

	// 获取池子的 holder 信息来获取 base_change 和 quote_change
	var poolHolder models.PumpfunAmmpoolHolder
	baseChange := 0.0
	quoteChange := 0.0

	// 查询 PumpfunAmmpoolHolder，Address 为 PoolAddress
	if err := dbconfig.DB.Where("address = ? AND pool_address = ?", poolConfig.PoolAddress, poolConfig.PoolAddress).First(&poolHolder).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pool holder: " + err.Error()})
			return
		}
		// 如果没有找到记录，使用默认值 0.0
	} else {
		baseChange = poolHolder.BaseChange
		quoteChange = poolHolder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}
	}

	// 调用业务逻辑计算结算数据
	// settleStats, err := business.CalculatePumpfunAmmPoolSettle(&project, &pumpfunStat)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }

	// 使用结合了 PumpfunAmmPoolHolder 和 PumpfuninternalHolder 的计算函数
	settleStats, err := business.CalculatePumpfunCombinedPoolSettle(&project, &pumpfunStat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建响应
	c.JSON(http.StatusOK, gin.H{
		"pool_stat": gin.H{
			"id":                    pumpfunStat.ID,
			"pool_id":               pumpfunStat.PoolID,
			"base_amount":           pumpfunStat.BaseAmount,
			"quote_amount":          pumpfunStat.QuoteAmount,
			"base_amount_readable":  pumpfunStat.BaseAmountReadable,
			"quote_amount_readable": pumpfunStat.QuoteAmountReadable,
			"market_value":          pumpfunStat.MarketValue,
			"lp_supply":             pumpfunStat.LpSupply,
			"price":                 pumpfunStat.Price,
			"slot":                  pumpfunStat.Slot,
			"block_time":            pumpfunStat.BlockTime.UnixMilli(),
			"created_at":            pumpfunStat.CreatedAt.UnixMilli(),
			"updated_at":            pumpfunStat.UpdatedAt.UnixMilli(),
			"base_change":           baseChange,
			"quote_change":          quoteChange,
		},
		"settle": settleStats,
	})
}

// getSettleStatsByMeteoradbcPool 处理 Meteoradbc Pool 池子的结算统计
func getSettleStatsByMeteoradbcPool(c *gin.Context, project models.ProjectConfig) {
	// 获取 MeteoradbcConfig 来获取 PoolAddress
	var poolConfig models.MeteoradbcConfig
	if err := dbconfig.DB.Where("id = ?", project.PoolID).First(&poolConfig).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MeteoradbcConfig not found"})
		return
	}

	// 获取池子统计
	var meteoradbcStat models.MeteoradbcPoolStat
	if err := dbconfig.DB.Where("pool_address = ?", poolConfig.PoolAddress).First(&meteoradbcStat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MeteoradbcPoolStat not found"})
		return
	}

	// 获取池子的 holder 信息来获取 base_change 和 quote_change
	var poolHolder models.MeteoradbcHolder
	baseChange := 0.0
	quoteChange := 0.0

	// 查询 MeteoradbcHolder，Address 为 PoolAddress
	if err := dbconfig.DB.Where("address = ? AND pool_address = ?", poolConfig.PoolAddress, poolConfig.PoolAddress).First(&poolHolder).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pool holder: " + err.Error()})
			return
		}
		// 如果没有找到记录，使用默认值 0.0
	} else {
		baseChange = poolHolder.BaseChange
		quoteChange = poolHolder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}
	}

	// 调用业务逻辑计算结算数据
	settleStats, err := business.CalculateMeteoradbcPoolSettle(&project, &meteoradbcStat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建响应
	c.JSON(http.StatusOK, gin.H{
		"pool_stat": gin.H{
			"id":                    meteoradbcStat.ID,
			"pool_address":          meteoradbcStat.PoolAddress,
			"base_amount":           meteoradbcStat.BaseAmount,
			"quote_amount":          meteoradbcStat.QuoteAmount,
			"base_amount_readable":  meteoradbcStat.BaseAmountReadable,
			"quote_amount_readable": meteoradbcStat.QuoteAmountReadable,
			"market_value":          meteoradbcStat.MarketValue,
			"price":                 meteoradbcStat.Price,
			"slot":                  meteoradbcStat.Slot,
			"block_time":            meteoradbcStat.BlockTime.UnixMilli(),
			"created_at":            meteoradbcStat.CreatedAt.UnixMilli(),
			"updated_at":            meteoradbcStat.UpdatedAt.UnixMilli(),
			"base_change":           baseChange,
			"quote_change":          quoteChange,
		},
		"settle": settleStats,
	})
}

// getSettleStatsByMeteoracpmmPool 处理 Meteoracpmm 池子的结算统计
func getSettleStatsByMeteoracpmmPool(c *gin.Context, project models.ProjectConfig) {
	// 获取 MeteoracpmmConfig 来获取 PoolAddress
	var poolConfig models.MeteoracpmmConfig
	if err := dbconfig.DB.Where("id = ?", project.PoolID).First(&poolConfig).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MeteoracpmmConfig not found"})
		return
	}

	// 获取池子统计
	var meteoracpmmStat models.MeteoracpmmPoolStat
	if err := dbconfig.DB.Where("pool_address = ?", poolConfig.PoolAddress).First(&meteoracpmmStat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MeteoracpmmPoolStat not found"})
		return
	}

	// 获取池子的 holder 信息来获取 base_change 和 quote_change
	var poolHolder models.MeteoracpmmHolder
	baseChange := 0.0
	quoteChange := 0.0

	// 查询 MeteoracpmmHolder，Address 为 PoolAddress
	if err := dbconfig.DB.Where("address = ? AND pool_address = ?", poolConfig.PoolAddress, poolConfig.PoolAddress).First(&poolHolder).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query pool holder: " + err.Error()})
			return
		}
		// 如果没有找到记录，使用默认值 0.0
	} else {
		baseChange = poolHolder.BaseChange
		quoteChange = poolHolder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}
	}

	// 调用业务逻辑计算结算数据
	settleStats, err := business.CalculateMeteoracpmmPoolSettle(&project, &meteoracpmmStat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建响应
	c.JSON(http.StatusOK, gin.H{
		"pool_stat": gin.H{
			"id":                    meteoracpmmStat.ID,
			"pool_address":          meteoracpmmStat.PoolAddress,
			"base_amount":           meteoracpmmStat.BaseAmount,
			"quote_amount":          meteoracpmmStat.QuoteAmount,
			"base_amount_readable":  meteoracpmmStat.BaseAmountReadable,
			"quote_amount_readable": meteoracpmmStat.QuoteAmountReadable,
			"market_value":          meteoracpmmStat.MarketValue,
			"price":                 meteoracpmmStat.Price,
			"slot":                  meteoracpmmStat.Slot,
			"block_time":            meteoracpmmStat.BlockTime.UnixMilli(),
			"created_at":            meteoracpmmStat.CreatedAt.UnixMilli(),
			"updated_at":            meteoracpmmStat.UpdatedAt.UnixMilli(),
			"base_change":           baseChange,
			"quote_change":          quoteChange,
		},
		"settle": settleStats,
	})
}

// ListPumpfuninternalStats returns a list of all pumpfun internal stats
func ListPumpfuninternalStats(c *gin.Context) {
	var stats []models.PumpfuninternalStat
	if err := dbconfig.DB.Preload("PumpfunPool").Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetPumpfuninternalStat returns a specific pumpfun internal stat by ID
func GetPumpfuninternalStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.PumpfuninternalStat
	if err := dbconfig.DB.Preload("PumpfunPool").First(&stat, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetPumpfuninternalStatsByPoolID returns all stats for a specific pool
func GetPumpfuninternalStatsByPoolID(c *gin.Context) {
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	var stats []models.PumpfuninternalStat
	if err := dbconfig.DB.Preload("PumpfunPool").Where("pumpfuninternal_id = ?", poolID).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetPumpfuninternalStatByProjectID returns the latest stat for a project's pool
func GetPumpfuninternalStatByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var project models.ProjectConfig
	if err := dbconfig.DB.Where("id = ? AND pool_platform = ?", projectID, "pumpfun_internal").First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not a pumpfun internal pool"})
		return
	}

	var stat models.PumpfuninternalStat
	if err := dbconfig.DB.Preload("PumpfunPool").
		Where("pumpfuninternal_id = ?", project.PoolID).
		Order("block_time DESC").
		First(&stat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PumpfuninternalStat not found"})
		return
	}

	resp := BuildPumpfuninternalStatRespSimple(&stat)
	resp.Platform = "pumpfun_internal"
	c.JSON(http.StatusOK, gin.H{
		"id":                     resp.ID,
		"pumpfuninternal_id":     resp.PumpfuninternalID,
		"platform":               resp.Platform,
		"mint":                   resp.Mint,
		"unknown_data":           resp.UnknownData,
		"virtual_token_reserves": resp.VirtualTokenReserves,
		"virtual_sol_reserves":   resp.VirtualSolReserves,
		"real_token_reserves":    resp.RealTokenReserves,
		"real_sol_reserves":      resp.RealSolReserves,
		"token_total_supply":     resp.TokenTotalSupply,
		"complete":               resp.Complete,
		"creator":                resp.Creator,
		"price":                  resp.Price,
		"fee_recipient":          resp.FeeRecipient,
		"sol_balance":            resp.SolBalance,
		"token_balance":          resp.TokenBalance,
		"slot":                   resp.Slot,
		"block_time":             resp.BlockTime,
		"created_at":             resp.CreatedAt,
		"updated_at":             resp.UpdatedAt,
		"pumpfun_pool": gin.H{
			"id":                       stat.PumpfunPool.ID,
			"bonding_curve_pda":        stat.PumpfunPool.BondingCurvePda,
			"associated_bonding_curve": stat.PumpfunPool.AssociatedBondingCurve,
			"creator_vault_pda":        stat.PumpfunPool.CreatorVaultPda,
			"fee_recipient":            stat.PumpfunPool.FeeRecipient,
			"mint":                     stat.PumpfunPool.Mint,
			"fee_rate":                 stat.PumpfunPool.FeeRate,
			"status":                   stat.PumpfunPool.Status,
			"created_at":               stat.PumpfunPool.CreatedAt.UnixMilli(),
			"updated_at":               stat.PumpfunPool.UpdatedAt.UnixMilli(),
		},
	})
}

// GetPumpfuninternalStatsByMint returns all stats for a specific mint
func GetPumpfuninternalStatsByMint(c *gin.Context) {
	mint := c.Param("mint")
	var stats []models.PumpfuninternalStat
	if err := dbconfig.DB.Preload("PumpfunPool").Where("mint = ?", mint).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ListPumpfunAmmPoolStats returns a list of all pool stats
func ListPumpfunAmmPoolStats(c *gin.Context) {
	var stats []models.PumpfunAmmPoolStat
	if err := dbconfig.DB.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetPumpfunAmmPoolStat returns a specific pool stat by ID
func GetPumpfunAmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.PumpfunAmmPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetPumpfunAmmPoolStatByPoolID returns a pool stat by pool ID
func GetPumpfunAmmPoolStatByPoolID(c *gin.Context) {
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool ID format"})
		return
	}

	var stat models.PumpfunAmmPoolStat
	if err := dbconfig.DB.Where("pool_id = ?", poolID).First(&stat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetPumpfunAmmPoolStatByProjectID returns the latest stat for a project's AMM pool
func GetPumpfunAmmPoolStatByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var project models.ProjectConfig
	if err := dbconfig.DB.Where("id = ? AND pool_platform = ?", projectID, "pumpfun_amm").First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not a pumpfun AMM pool"})
		return
	}

	var stat models.PumpfunAmmPoolStat
	if err := dbconfig.DB.Where("pool_id = ?", project.PoolID).
		Order("block_time DESC").
		First(&stat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "PumpfunAmmPoolStat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                    stat.ID,
		"pool_id":               stat.PoolID,
		"base_amount":           stat.BaseAmount,
		"quote_amount":          stat.QuoteAmount,
		"base_amount_readable":  stat.BaseAmountReadable,
		"quote_amount_readable": stat.QuoteAmountReadable,
		"market_value":          stat.MarketValue,
		"lp_supply":             stat.LpSupply,
		"price":                 stat.Price,
		"slot":                  stat.Slot,
		"block_time":            stat.BlockTime.UnixMilli(),
		"created_at":            stat.CreatedAt.UnixMilli(),
		"updated_at":            stat.UpdatedAt.UnixMilli(),
	})
}

// DeletePumpfunAmmPoolStat deletes a pumpfun amm pool stat by ID
func DeletePumpfunAmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.PumpfunAmmPoolStat{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "PumpfunAmmPoolStat deleted successfully"})
}

// RaydiumLaunchpadPoolStat CRUD handlers

// ListRaydiumLaunchpadPoolStats returns all Raydium Launchpad pool stats
func ListRaydiumLaunchpadPoolStats(c *gin.Context) {
	var stats []models.RaydiumLaunchpadPoolStat
	if err := dbconfig.DB.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetRaydiumLaunchpadPoolStat returns a specific Raydium Launchpad pool stat by ID
func GetRaydiumLaunchpadPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.RaydiumLaunchpadPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Raydium Launchpad pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetRaydiumLaunchpadPoolStatByPoolID returns Raydium Launchpad pool stats by pumpfuninternal_id
func GetRaydiumLaunchpadPoolStatByPoolID(c *gin.Context) {
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	var stats []models.RaydiumLaunchpadPoolStat
	if err := dbconfig.DB.Where("pumpfuninternal_id = ?", poolID).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetRaydiumLaunchpadPoolStatByProjectID returns Raydium Launchpad pool stat by project ID
func GetRaydiumLaunchpadPoolStatByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Find the project config
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.Where("id = ?", projectID).First(&projectConfig).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the latest stat for this project's pool
	var stat models.RaydiumLaunchpadPoolStat
	if err := dbconfig.DB.Where("pumpfuninternal_id = ?", projectConfig.PoolID).
		Order("created_at desc").First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Raydium Launchpad pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetRaydiumLaunchpadPoolStatsByMint returns Raydium Launchpad pool stats by mint address
func GetRaydiumLaunchpadPoolStatsByMint(c *gin.Context) {
	mint := c.Param("mint")

	var stats []models.RaydiumLaunchpadPoolStat
	if err := dbconfig.DB.Where("mint = ?", mint).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// CreateRaydiumLaunchpadPoolStat creates a new Raydium Launchpad pool stat
func CreateRaydiumLaunchpadPoolStat(c *gin.Context) {
	var stat models.RaydiumLaunchpadPoolStat
	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Create(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stat)
}

// UpdateRaydiumLaunchpadPoolStat updates an existing Raydium Launchpad pool stat
func UpdateRaydiumLaunchpadPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.RaydiumLaunchpadPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Raydium Launchpad pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Save(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// DeleteRaydiumLaunchpadPoolStat deletes a Raydium Launchpad pool stat by ID
func DeleteRaydiumLaunchpadPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.RaydiumLaunchpadPoolStat{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Raydium Launchpad pool stat deleted successfully"})
}

// RaydiumCpmmPoolStat CRUD handlers

// ListRaydiumCpmmPoolStats returns all Raydium CPMM pool stats
func ListRaydiumCpmmPoolStats(c *gin.Context) {
	var stats []models.RaydiumCpmmPoolStat
	if err := dbconfig.DB.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetRaydiumCpmmPoolStat returns a specific Raydium CPMM pool stat by ID
func GetRaydiumCpmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.RaydiumCpmmPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Raydium CPMM pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetRaydiumCpmmPoolStatByPoolID returns Raydium CPMM pool stats by pool_id
func GetRaydiumCpmmPoolStatByPoolID(c *gin.Context) {
	poolID, err := strconv.Atoi(c.Param("pool_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id format"})
		return
	}

	var stats []models.RaydiumCpmmPoolStat
	if err := dbconfig.DB.Where("pool_id = ?", poolID).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetRaydiumCpmmPoolStatByProjectID returns Raydium CPMM pool stat by project ID
func GetRaydiumCpmmPoolStatByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Find the project config
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.Where("id = ?", projectID).First(&projectConfig).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the latest stat for this project's pool
	var stat models.RaydiumCpmmPoolStat
	if err := dbconfig.DB.Where("pool_id = ?", projectConfig.PoolID).
		Order("created_at desc").First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Raydium CPMM pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// CreateRaydiumCpmmPoolStat creates a new Raydium CPMM pool stat
func CreateRaydiumCpmmPoolStat(c *gin.Context) {
	var stat models.RaydiumCpmmPoolStat
	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Create(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stat)
}

// UpdateRaydiumCpmmPoolStat updates an existing Raydium CPMM pool stat
func UpdateRaydiumCpmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.RaydiumCpmmPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Raydium CPMM pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Save(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// DeleteRaydiumCpmmPoolStat deletes a Raydium CPMM pool stat by ID
func DeleteRaydiumCpmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.RaydiumCpmmPoolStat{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Raydium CPMM pool stat deleted successfully"})
}

// LocateDuplicateWalletTokenStat 查找重复的钱包代币统计数据，不删除数据
func LocateDuplicateWalletTokenStat(c *gin.Context) {
	// 获取请求参数
	var req struct {
		RoleID uint   `json:"role_id" binding:"required"`
		Mint   string `json:"mint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// 获取角色下的所有地址
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", req.RoleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role addresses"})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No addresses found for this role"})
		return
	}

	// 提取所有地址
	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}

	// 存储有重复记录的地址
	var duplicateAddresses []string

	// 对每个地址进行处理
	for _, address := range addresses {
		// 查找该地址下指定代币的所有统计数据
		var count int64
		if err := dbconfig.DB.Model(&models.WalletTokenStat{}).
			Where("owner_address = ? AND mint = ?", address, req.Mint).
			Count(&count).Error; err != nil {
			continue // 如果查询出错，跳过当前地址
		}

		// 如果有多条记录，说明有重复
		if count > 1 {
			duplicateAddresses = append(duplicateAddresses, address)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Duplicate records located successfully",
		"duplicate_addresses": duplicateAddresses,
		"total_addresses":     len(duplicateAddresses),
	})
}

// RemoveDuplicateWalletTokenStat 删除重复的钱包代币统计数据，保留最新的数据
func RemoveDuplicateWalletTokenStat(c *gin.Context) {
	// 获取请求参数
	var req struct {
		RoleID uint   `json:"role_id" binding:"required"`
		Mint   string `json:"mint" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// 获取角色下的所有地址
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Where("role_id = ?", req.RoleID).Find(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get role addresses"})
		return
	}

	if len(roleAddresses) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No addresses found for this role"})
		return
	}

	// 提取所有地址
	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}

	// 统计删除的记录数和地址
	var totalDeleted int64 = 0
	var affectedAddresses []string

	// 对每个地址进行处理
	for _, address := range addresses {
		// 查找该地址下指定代币的所有统计数据
		var stats []models.WalletTokenStat
		if err := dbconfig.DB.Where("owner_address = ? AND mint = ?", address, req.Mint).
			Order("block_time DESC").Find(&stats).Error; err != nil {
			continue // 如果查询出错，跳过当前地址
		}

		// 如果有多条记录，保留最新的，删除其他的
		if len(stats) > 1 {
			// 第一条是最新的（因为我们按 block_time DESC 排序）
			latestStat := stats[0]

			// 删除其他记录
			if err := dbconfig.DB.Where("owner_address = ? AND mint = ? AND id != ?",
				address, req.Mint, latestStat.ID).Delete(&models.WalletTokenStat{}).Error; err != nil {
				continue // 如果删除出错，跳过当前地址
			}

			totalDeleted += int64(len(stats) - 1)
			affectedAddresses = append(affectedAddresses, address)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Duplicate records removed successfully",
		"total_deleted":      totalDeleted,
		"affected_addresses": affectedAddresses,
	})
}

// MeteoradbcPoolStat CRUD handlers

// ListMeteoradbcPoolStats returns all Meteoradbc pool stats
func ListMeteoradbcPoolStats(c *gin.Context) {
	var stats []models.MeteoradbcPoolStat
	if err := dbconfig.DB.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetMeteoradbcPoolStat returns a specific Meteoradbc pool stat by ID
func GetMeteoradbcPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.MeteoradbcPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoradbc pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// CreateMeteoradbcPoolStat creates a new Meteoradbc pool stat
func CreateMeteoradbcPoolStat(c *gin.Context) {
	var stat models.MeteoradbcPoolStat
	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Create(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stat)
}

// UpdateMeteoradbcPoolStat updates an existing Meteoradbc pool stat
func UpdateMeteoradbcPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.MeteoradbcPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoradbc pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Save(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// DeleteMeteoradbcPoolStat deletes a Meteoradbc pool stat by ID
func DeleteMeteoradbcPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.MeteoradbcPoolStat{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Meteoradbc pool stat deleted successfully"})
}

// GetMeteoradbcPoolStatByPoolAddress returns Meteoradbc pool stats by pool address
func GetMeteoradbcPoolStatByPoolAddress(c *gin.Context) {
	poolAddress := c.Param("pool_address")

	var stats []models.MeteoradbcPoolStat
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetMeteoradbcPoolStatByProjectID returns Meteoradbc pool stat by project ID
func GetMeteoradbcPoolStatByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Find the project config
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.Where("id = ? AND pool_platform = ?", projectID, "meteora_dbc").First(&projectConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not a Meteoradbc pool"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the MeteoradbcConfig to find the pool address
	var meteoradbcConfig models.MeteoradbcConfig
	if err := dbconfig.DB.Where("id = ?", projectConfig.PoolID).First(&meteoradbcConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoradbc config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the latest stat for this project's pool
	var stat models.MeteoradbcPoolStat
	if err := dbconfig.DB.Where("pool_address = ?", meteoradbcConfig.PoolAddress).
		Order("created_at desc").First(&stat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoradbc pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert time fields to timestamps
	response := gin.H{
		"id":                    stat.ID,
		"pool_address":          stat.PoolAddress,
		"base_amount":           stat.BaseAmount,
		"quote_amount":          stat.QuoteAmount,
		"base_amount_readable":  stat.BaseAmountReadable,
		"quote_amount_readable": stat.QuoteAmountReadable,
		"market_value":          stat.MarketValue,
		"price":                 stat.Price,
		"slot":                  stat.Slot,
		"block_time":            stat.BlockTime.UnixMilli(),
		"created_at":            stat.CreatedAt.UnixMilli(),
		"updated_at":            stat.UpdatedAt.UnixMilli(),
	}

	c.JSON(http.StatusOK, response)
}

// MeteoracpmmPoolStat CRUD handlers

// ListMeteoracpmmPoolStats returns all Meteoracpmm pool stats
func ListMeteoracpmmPoolStats(c *gin.Context) {
	var stats []models.MeteoracpmmPoolStat
	if err := dbconfig.DB.Find(&stats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// GetMeteoracpmmPoolStat returns a specific Meteoracpmm pool stat by ID
func GetMeteoracpmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.MeteoracpmmPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoracpmm pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// CreateMeteoracpmmPoolStat creates a new Meteoracpmm pool stat
func CreateMeteoracpmmPoolStat(c *gin.Context) {
	var stat models.MeteoracpmmPoolStat
	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Create(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stat)
}

// UpdateMeteoracpmmPoolStat updates an existing Meteoracpmm pool stat
func UpdateMeteoracpmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var stat models.MeteoracpmmPoolStat
	if err := dbconfig.DB.First(&stat, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoracpmm pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := c.ShouldBindJSON(&stat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := dbconfig.DB.Save(&stat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// DeleteMeteoracpmmPoolStat deletes a Meteoracpmm pool stat by ID
func DeleteMeteoracpmmPoolStat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.MeteoracpmmPoolStat{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Meteoracpmm pool stat deleted successfully"})
}

// GetMeteoracpmmPoolStatByPoolAddress returns Meteoracpmm pool stats by pool address
func GetMeteoracpmmPoolStatByPoolAddress(c *gin.Context) {
	poolAddress := c.Param("pool_address")

	var stat models.MeteoracpmmPoolStat
	if err := dbconfig.DB.Where("pool_address = ?", poolAddress).
		Order("created_at desc").First(&stat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoracpmm pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stat)
}

// GetMeteoracpmmPoolStatByProjectID returns Meteoracpmm pool stat by project ID
func GetMeteoracpmmPoolStatByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Find the project config
	var projectConfig models.ProjectConfig
	if err := dbconfig.DB.Where("id = ? AND pool_platform = ?", projectID, "meteora_cpmm").First(&projectConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or not a Meteoracpmm pool"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the MeteoracpmmConfig to find the pool address
	var meteoracpmmConfig models.MeteoracpmmConfig
	if err := dbconfig.DB.Where("id = ?", projectConfig.PoolID).First(&meteoracpmmConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoracpmm config not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get the latest stat for this project's pool
	var stat models.MeteoracpmmPoolStat
	if err := dbconfig.DB.Where("pool_address = ?", meteoracpmmConfig.PoolAddress).
		Order("created_at desc").First(&stat).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Meteoracpmm pool stat not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert time fields to timestamps
	response := gin.H{
		"id":                    stat.ID,
		"pool_address":          stat.PoolAddress,
		"base_amount":           stat.BaseAmount,
		"quote_amount":          stat.QuoteAmount,
		"base_amount_readable":  stat.BaseAmountReadable,
		"quote_amount_readable": stat.QuoteAmountReadable,
		"market_value":          stat.MarketValue,
		"price":                 stat.Price,
		"slot":                  stat.Slot,
		"block_time":            stat.BlockTime.UnixMilli(),
		"created_at":            stat.CreatedAt.UnixMilli(),
		"updated_at":            stat.UpdatedAt.UnixMilli(),
	}

	c.JSON(http.StatusOK, response)
}
