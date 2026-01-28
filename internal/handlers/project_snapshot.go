package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/utils"
)

// AggregateTokenSnapshot 聚合代币快照结构体
type AggregateTokenSnapshot struct {
	Mint            string  `json:"mint"`
	BalanceReadable float64 `json:"balance_readable"`
}

// ListWalletTokenSnapshots 获取所有钱包快照
func ListWalletTokenSnapshots(c *gin.Context) {
	var snapshots []models.WalletTokenSnapshot
	if err := dbconfig.DB.Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshots)
}

// GetWalletTokenSnapshot 获取指定ID的钱包快照
func GetWalletTokenSnapshot(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var snapshot models.WalletTokenSnapshot
	if err := dbconfig.DB.First(&snapshot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

// ListWalletTokenSnapshotsByProject 获取指定项目的所有钱包快照
func ListWalletTokenSnapshotsByProject(c *gin.Context) {
	projectID := c.Param("project_id")
	var snapshots []models.WalletTokenSnapshot
	if err := dbconfig.DB.Where("project_id = ?", projectID).Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshots)
}

// ListWalletTokenSnapshotsBySnapshotID 获取指定快照ID的所有钱包快照
func ListWalletTokenSnapshotsBySnapshotID(c *gin.Context) {
	snapshotID := c.Param("snapshot_id")
	var snapshots []models.WalletTokenSnapshot
	if err := dbconfig.DB.Where("snapshot_id = ?", snapshotID).Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, snapshots)
}

// ListPoolSnapshots 获取所有池子快照
func ListPoolSnapshots(c *gin.Context) {
	// 获取查询参数
	platform := c.Query("platform")
	if platform == "" {
		platform = "raydium" // 默认为 raydium
	}

	switch platform {
	case "raydium":
		var snapshots []models.PoolSnapshot
		if err := dbconfig.DB.Find(&snapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var result []*PoolSnapshotResp
		for _, snapshot := range snapshots {
			snap := snapshot // Create a new variable to avoid pointer issues
			result = append(result, BuildPoolSnapshotResp(&snap))
		}
		c.JSON(http.StatusOK, result)

	case "pumpfun_internal":
		var snapshots []models.PumpfuninternalSnapshot
		if err := dbconfig.DB.Find(&snapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var result []*PumpfuninternalSnapshotResp
		for _, snapshot := range snapshots {
			snap := snapshot // Create a new variable to avoid pointer issues
			result = append(result, BuildPumpfuninternalSnapshotResp(&snap))
		}
		c.JSON(http.StatusOK, result)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
	}
}

// GetPoolSnapshot 获取指定ID的池子快照
func GetPoolSnapshot(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// 获取查询参数
	platform := c.Query("platform")
	if platform == "" {
		platform = "raydium" // 默认为 raydium
	}

	switch platform {
	case "raydium":
		var snapshot models.PoolSnapshot
		if err := dbconfig.DB.First(&snapshot, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
			return
		}
		c.JSON(http.StatusOK, BuildPoolSnapshotResp(&snapshot))

	case "pumpfun_internal":
		var snapshot models.PumpfuninternalSnapshot
		if err := dbconfig.DB.First(&snapshot, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
			return
		}
		c.JSON(http.StatusOK, BuildPumpfuninternalSnapshotResp(&snapshot))

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
	}
}

// GetPoolSnapshotByProject 获取指定项目的池子快照
func GetPoolSnapshotByProject(c *gin.Context) {
	projectID := c.Param("project_id")

	var req struct {
		SnapshotID uint `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 1. 获取项目信息
	var project models.ProjectConfig
	if err := dbconfig.DB.Where("id = ?", projectID).First(&project).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// 2. 根据不同的池子平台返回对应的快照数据
	switch project.PoolPlatform {
	case "raydium":
		var snapshot models.PoolSnapshot
		query := dbconfig.DB.Where("project_id = ?", projectID)
		if req.SnapshotID > 0 {
			query = query.Where("snapshot_id = ?", req.SnapshotID)
		}
		if err := query.First(&snapshot).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found"})
			return
		}
		c.JSON(http.StatusOK, BuildPoolSnapshotResp(&snapshot))

	case "pumpfun_internal":
		var snapshot models.PumpfuninternalSnapshot
		query := dbconfig.DB.Where("project_id = ?", projectID)
		if req.SnapshotID > 0 {
			query = query.Where("snapshot_id = ?", req.SnapshotID)
		}
		if err := query.First(&snapshot).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found"})
			return
		}
		c.JSON(http.StatusOK, BuildPumpfuninternalSnapshotResp(&snapshot))

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
	}
}

// ListPoolSnapshotsBySnapshotID 获取指定快照ID的所有池子快照
func ListPoolSnapshotsBySnapshotID(c *gin.Context) {
	snapshotID := c.Param("snapshot_id")

	// 获取查询参数
	platform := c.Query("platform")
	if platform == "" {
		platform = "raydium" // 默认为 raydium
	}

	switch platform {
	case "raydium":
		var snapshots []models.PoolSnapshot
		if err := dbconfig.DB.Where("snapshot_id = ?", snapshotID).Find(&snapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var result []*PoolSnapshotResp
		for _, snapshot := range snapshots {
			snap := snapshot // Create a new variable to avoid pointer issues
			result = append(result, BuildPoolSnapshotResp(&snap))
		}
		c.JSON(http.StatusOK, result)

	case "pumpfun_internal":
		var snapshots []models.PumpfuninternalSnapshot
		if err := dbconfig.DB.Where("snapshot_id = ?", snapshotID).Find(&snapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var result []*PumpfuninternalSnapshotResp
		for _, snapshot := range snapshots {
			snap := snapshot // Create a new variable to avoid pointer issues
			result = append(result, BuildPumpfuninternalSnapshotResp(&snap))
		}
		c.JSON(http.StatusOK, result)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
	}
}

// ListWalletTokenSnapshotsByRoleID 根据 role_id 和 tokens 查询所有钱包快照
func ListWalletTokenSnapshotsByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var req struct {
		Tokens     []string `json:"tokens"`
		SnapshotID uint     `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var snapshots []models.WalletTokenSnapshot
	query := dbconfig.DB.Where("role_id = ?", roleID)
	if len(req.Tokens) > 0 {
		query = query.Where("mint IN ?", req.Tokens)
	}
	if req.SnapshotID > 0 {
		query = query.Where("snapshot_id = ?", req.SnapshotID)
	}
	if err := query.Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 分组
	resultMap := make(map[string][]models.WalletTokenSnapshot)
	for _, snap := range snapshots {
		resultMap[snap.OwnerAddress] = append(resultMap[snap.OwnerAddress], snap)
	}

	type WalletTokenSnapshotResp struct {
		Mint            string  `json:"mint"`
		BalanceReadable float64 `json:"balance_readable"`
		Slot            uint64  `json:"slot"`
		BlockTime       int64   `json:"block_time"`
		SourceUpdatedAt int64   `json:"source_updated_at"`
		CreatedAt       int64   `json:"created_at"`
	}

	type TokenGroup struct {
		OwnerAddress string                   `json:"owner_address"`
		Tokens       []WalletTokenSnapshotResp `json:"tokens"`
	}

	var result []TokenGroup
	for owner, tokens := range resultMap {
		var tokenList []WalletTokenSnapshotResp
		for _, t := range tokens {
			tokenList = append(tokenList, WalletTokenSnapshotResp{
				Mint:            t.Mint,
				BalanceReadable: t.BalanceReadable,
				Slot:            t.Slot,
				BlockTime:       t.BlockTime.UnixMilli(),
				SourceUpdatedAt: t.SourceUpdatedAt.UnixMilli(),
				CreatedAt:       t.CreatedAt.UnixMilli(),
			})
		}
		result = append(result, TokenGroup{
			OwnerAddress: owner,
			Tokens:       tokenList,
		})
	}

	c.JSON(http.StatusOK, result)
}

// ListAggregateWalletTokenSnapshotsByRoleID 聚合所有 owner_address 下同一 mint 的钱包快照余额
func ListAggregateWalletTokenSnapshotsByRoleID(c *gin.Context) {
	roleID, err := strconv.Atoi(c.Param("role_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role_id format"})
		return
	}

	var req struct {
		Tokens     []string `json:"tokens"`
		SnapshotID uint     `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var snapshots []models.WalletTokenSnapshot
	query := dbconfig.DB.Where("role_id = ?", roleID)
	if len(req.Tokens) > 0 {
		query = query.Where("mint IN ?", req.Tokens)
	}
	if req.SnapshotID > 0 {
		query = query.Where("snapshot_id = ?", req.SnapshotID)
	}
	if err := query.Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tokenMap := make(map[string]*AggregateTokenSnapshot)
	for _, snap := range snapshots {
		if agg, ok := tokenMap[snap.Mint]; ok {
			agg.BalanceReadable += snap.BalanceReadable
		} else {
			tokenMap[snap.Mint] = &AggregateTokenSnapshot{
				Mint:            snap.Mint,
				BalanceReadable: snap.BalanceReadable,
			}
		}
	}

	var result []AggregateTokenSnapshot
	for _, v := range tokenMap {
		result = append(result, *v)
	}

	c.JSON(http.StatusOK, result)
}

// GetSettleSnapshotByProject 根据快照数据计算结算统计
func GetSettleSnapshotByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var req struct {
		SnapshotID uint `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.SnapshotID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot_id is required"})
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
	case "raydium":
		getSettleSnapshotsByRaydiumPool(c, project, req.SnapshotID)
	case "pumpfun_internal":
		getSettleSnapshotsByPumpfunPool(c, project, req.SnapshotID)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
	}
}

// getSettleSnapshotsByRaydiumPool 处理 Raydium 池子的结算统计
func getSettleSnapshotsByRaydiumPool(c *gin.Context, project models.ProjectConfig, snapshotID uint) {
	// 2. 获取池子快照数据
	var poolSnapshot models.PoolSnapshot
	if err := dbconfig.DB.Where("project_id = ? AND snapshot_id = ?", project.ID, snapshotID).First(&poolSnapshot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool snapshot not found"})
		return
	}

	// 3. 获取池子配置
	var pool models.PoolConfig
	if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		return
	}

	// 获取项目聚合的代币余额（基于快照数据）
	tokenMint := project.Token.Mint
	totalSupply := project.Token.TotalSupply
	
	// 查询项目角色下的地址
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Table("role_address").
		Select("DISTINCT address").
		Joins("JOIN role_config ON role_address.role_id = role_config.id").
		Where("role_config.project_id = ?", project.ID).
		Scan(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}
	
	var walletSnapshots []models.WalletTokenSnapshot
	var y float64 = 0
	if len(addresses) > 0 {
		if err := dbconfig.DB.Where("project_id = ? AND snapshot_id = ? AND mint = ? AND owner_address IN ?", project.ID, snapshotID, tokenMint, addresses).Find(&walletSnapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, snapshot := range walletSnapshots {
			y += snapshot.BalanceReadable
		}
	}

	// 4. 计算 project_token_market_value（使用快照中的池子数据）
	fee := pool.FeeRate
	base := poolSnapshot.BaseAmountReadable
	quote := poolSnapshot.QuoteAmountReadable
	tvlByProject := utils.SimulateConstantProductAmountOut(y, "y", base, quote, fee)
	
	// 5. 计算 retail_investors_token 及其市值
	retailInvestorsToken := totalSupply - quote - y
	tvlByRetailInvestors := utils.SimulateConstantProductAmountOut(retailInvestorsToken, "y", base, quote, fee)

	// 6. 计算总市值相关变量
	tokenByExpectPool := y + retailInvestorsToken
	tvlByExpectPool := utils.SimulateConstantProductAmountOut(tokenByExpectPool, "y", base, quote, fee)
	tvlByProjectLastSoldOut := tvlByExpectPool - tvlByRetailInvestors

	// 7. 查询项目代币和SOL代币的快照统计数据
	solMint := "So11111111111111111111111111111111111111112"
	nativeSolMint := "sol"
	var allSnapshots []models.WalletTokenSnapshot
	if len(addresses) > 0 {
		if err := dbconfig.DB.Where("project_id = ? AND snapshot_id = ? AND mint IN ? AND owner_address IN ?", project.ID, snapshotID, []string{tokenMint, solMint, nativeSolMint}, addresses).Find(&allSnapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	
	tokenMap := AggregateSnapshotTokens(allSnapshots)
	
	var tokens []AggregateTokenSnapshot
	for _, v := range tokenMap {
		tokens = append(tokens, *v)
	}

	poolSnapshotResp := BuildPoolSnapshotResp(&poolSnapshot)

	c.JSON(http.StatusOK, gin.H{
		"pool_snapshot": poolSnapshotResp,
		"tokens": tokens,
		"settle": gin.H{
			"token_by_project": y,
			"token_by_pool": quote,
			"token_by_retail_investors": retailInvestorsToken,
			"token_by_expect_pool": tokenByExpectPool,
			"tvl_by_expect_pool": tvlByExpectPool,
			"tvl_by_project_last_sold_out": tvlByProjectLastSoldOut,
			"tvl_by_project": tvlByProject,
			"tvl_by_retail_investors": tvlByRetailInvestors,
		},
	})
}

// getSettleSnapshotsByPumpfunPool 处理 Pumpfun Internal 池子的结算统计
func getSettleSnapshotsByPumpfunPool(c *gin.Context, project models.ProjectConfig, snapshotID uint) {
	// 2. 获取池子快照数据
	var pumpfunSnapshot models.PumpfuninternalSnapshot
	if err := dbconfig.DB.Where("project_id = ? AND snapshot_id = ?", project.ID, snapshotID).First(&pumpfunSnapshot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pumpfuninternal snapshot not found"})
		return
	}

	// 3. 获取池子配置
	var pumpfunPool models.PumpfuninternalConfig
	if err := dbconfig.DB.First(&pumpfunPool, project.PoolID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pumpfun pool not found"})
		return
	}

	// 获取项目聚合的代币余额（基于快照数据）
	tokenMint := project.Token.Mint
	totalSupply := project.Token.TotalSupply
	
	// 查询项目角色下的地址
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Table("role_address").
		Select("DISTINCT address").
		Joins("JOIN role_config ON role_address.role_id = role_config.id").
		Where("role_config.project_id = ?", project.ID).
		Scan(&roleAddresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var addresses []string
	for _, addr := range roleAddresses {
		addresses = append(addresses, addr.Address)
	}
	
	var walletSnapshots []models.WalletTokenSnapshot
	var y float64 = 0
	if len(addresses) > 0 {
		if err := dbconfig.DB.Where("project_id = ? AND snapshot_id = ? AND mint = ? AND owner_address IN ?", project.ID, snapshotID, tokenMint, addresses).Find(&walletSnapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		for _, snapshot := range walletSnapshots {
			y += snapshot.BalanceReadable
		}
	}

	// 4. 计算 project_token_market_value
	fee := pumpfunPool.FeeRate
	virtualTokenReserves := float64(pumpfunSnapshot.VirtualTokenReserves) / 1e6
	virtualSolReserves := float64(pumpfunSnapshot.VirtualSolReserves) / 1e9
	
	result, err := utils.SimulateBondingCurveAmountOut(y, "token", virtualSolReserves, virtualTokenReserves, fee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to simulate project token market value: " + err.Error()})
		return
	}
	tvlByProject := result.GetAmount
	
	// 5. 计算 retail_investors_token 及其市值
	retailInvestorsToken := totalSupply - pumpfunSnapshot.TokenBalance - y
	result, err = utils.SimulateBondingCurveAmountOut(retailInvestorsToken, "token", virtualSolReserves, virtualTokenReserves, fee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to simulate retail investors market value: " + err.Error()})
		return
	}
	tvlByRetailInvestors := result.GetAmount

	// 6. 计算总市值相关变量
	tokenByExpectPool := y + retailInvestorsToken
	result, err = utils.SimulateBondingCurveAmountOut(tokenByExpectPool, "token", virtualSolReserves, virtualTokenReserves, fee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to simulate expected pool market value: " + err.Error()})
		return
	}
	tvlByExpectPool := result.GetAmount
	tvlByProjectLastSoldOut := tvlByExpectPool - tvlByRetailInvestors

	// 7. 查询项目代币和SOL代币的快照统计数据
	solMint := "So11111111111111111111111111111111111111112"
	nativeSolMint := "sol"
	var allSnapshots []models.WalletTokenSnapshot
	if len(addresses) > 0 {
		if err := dbconfig.DB.Where("project_id = ? AND snapshot_id = ? AND mint IN ? AND owner_address IN ?", project.ID, snapshotID, []string{tokenMint, solMint, nativeSolMint}, addresses).Find(&allSnapshots).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	
	tokenMap := AggregateSnapshotTokens(allSnapshots)
	
	var tokens []AggregateTokenSnapshot
	for _, v := range tokenMap {
		tokens = append(tokens, *v)
	}

	pumpfunSnapshotResp := BuildPumpfuninternalSnapshotResp(&pumpfunSnapshot)

	c.JSON(http.StatusOK, gin.H{
		"pool_snapshot": pumpfunSnapshotResp,
		"tokens": tokens,
		"settle": gin.H{
			"token_by_project": y,
			"token_by_pool": pumpfunSnapshot.TokenBalance,
			"token_by_retail_investors": retailInvestorsToken,
			"token_by_expect_pool": tokenByExpectPool,
			"tvl_by_expect_pool": tvlByExpectPool,
			"tvl_by_project_last_sold_out": tvlByProjectLastSoldOut,
			"tvl_by_project": tvlByProject,
			"tvl_by_retail_investors": tvlByRetailInvestors,
		},
	})
}

// AggregateSnapshotTokens 聚合快照代币数据
func AggregateSnapshotTokens(snapshots []models.WalletTokenSnapshot) map[string]*AggregateTokenSnapshot {
	tokenMap := make(map[string]*AggregateTokenSnapshot)
	for _, snapshot := range snapshots {
		if agg, ok := tokenMap[snapshot.Mint]; ok {
			agg.BalanceReadable += snapshot.BalanceReadable
		} else {
			tokenMap[snapshot.Mint] = &AggregateTokenSnapshot{
				Mint:            snapshot.Mint,
				BalanceReadable: snapshot.BalanceReadable,
			}
		}
	}
	return tokenMap
}

// PoolSnapshotResp 池子快照响应结构体
type PoolSnapshotResp struct {
	ID                  uint    `json:"id"`
	ProjectID           uint    `json:"project_id"`
	SnapshotID          uint    `json:"snapshot_id"`
	PoolAddress         string  `json:"pool_address"`
	BaseAmountReadable  float64 `json:"base_amount_readable"`
	QuoteAmountReadable float64 `json:"quote_amount_readable"`
	MarketValue         float64 `json:"market_value"`
	LpSupply            uint64  `json:"lp_supply"`
	Price               float64 `json:"price"`
	SourceUpdatedAt     int64   `json:"source_updated_at"`
	CreatedAt           int64   `json:"created_at"`
}

// BuildPoolSnapshotResp 构建池子快照响应
func BuildPoolSnapshotResp(poolSnapshot *models.PoolSnapshot) *PoolSnapshotResp {
	if poolSnapshot == nil {
		return nil
	}
	return &PoolSnapshotResp{
		ID:                  poolSnapshot.ID,
		ProjectID:           poolSnapshot.ProjectID,
		SnapshotID:          poolSnapshot.SnapshotID,
		PoolAddress:         poolSnapshot.PoolAddress,
		BaseAmountReadable:  poolSnapshot.BaseAmountReadable,
		QuoteAmountReadable: poolSnapshot.QuoteAmountReadable,
		MarketValue:         poolSnapshot.MarketValue,
		LpSupply:            poolSnapshot.LpSupply,
		Price:               poolSnapshot.Price,
		SourceUpdatedAt:     poolSnapshot.SourceUpdatedAt.UnixMilli(),
		CreatedAt:           poolSnapshot.CreatedAt.UnixMilli(),
	}
}

// PumpfuninternalSnapshotResp 响应结构体
type PumpfuninternalSnapshotResp struct {
	ID                  uint    `json:"id"`
	ProjectID          uint    `json:"project_id"`
	SnapshotID         uint    `json:"snapshot_id"`
	PumpfuninternalID  uint    `json:"pumpfuninternal_id"`
	Mint               string  `json:"mint"`
	UnknownData        uint64  `json:"unknown_data"`
	VirtualTokenReserves uint64 `json:"virtual_token_reserves"`
	VirtualSolReserves uint64  `json:"virtual_sol_reserves"`
	RealTokenReserves  uint64  `json:"real_token_reserves"`
	RealSolReserves    uint64  `json:"real_sol_reserves"`
	TokenTotalSupply   uint64  `json:"token_total_supply"`
	Complete           bool    `json:"complete"`
	Creator           string  `json:"creator"`
	Price             float64 `json:"price"`
	FeeRecipient      string  `json:"fee_recipient"`
	SolBalance        float64 `json:"sol_balance"`
	TokenBalance      float64 `json:"token_balance"`
	Slot             uint64  `json:"slot"`
	SourceUpdatedAt   int64   `json:"source_updated_at"`
	CreatedAt         int64   `json:"created_at"`
}

// BuildPumpfuninternalSnapshotResp 构建响应
func BuildPumpfuninternalSnapshotResp(snapshot *models.PumpfuninternalSnapshot) *PumpfuninternalSnapshotResp {
	if snapshot == nil {
		return nil
	}
	return &PumpfuninternalSnapshotResp{
		ID:                  snapshot.ID,
		ProjectID:          snapshot.ProjectID,
		SnapshotID:         snapshot.SnapshotID,
		PumpfuninternalID:  snapshot.PumpfuninternalID,
		Mint:               snapshot.Mint,
		UnknownData:        snapshot.UnknownData,
		VirtualTokenReserves: snapshot.VirtualTokenReserves,
		VirtualSolReserves:   snapshot.VirtualSolReserves,
		RealTokenReserves:    snapshot.RealTokenReserves,
		RealSolReserves:      snapshot.RealSolReserves,
		TokenTotalSupply:     snapshot.TokenTotalSupply,
		Complete:             snapshot.Complete,
		Creator:             snapshot.Creator,
		Price:               snapshot.Price,
		FeeRecipient:        snapshot.FeeRecipient,
		SolBalance:          snapshot.SolBalance,
		TokenBalance:        snapshot.TokenBalance,
		Slot:               snapshot.Slot,
		SourceUpdatedAt:     snapshot.SourceUpdatedAt.UnixMilli(),
		CreatedAt:           snapshot.CreatedAt.UnixMilli(),
	}
}

// ListPumpfuninternalSnapshots 获取所有 Pumpfuninternal 快照
func ListPumpfuninternalSnapshots(c *gin.Context) {
	var snapshots []models.PumpfuninternalSnapshot
	if err := dbconfig.DB.Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []*PumpfuninternalSnapshotResp
	for _, snapshot := range snapshots {
		snap := snapshot // Create a new variable to avoid pointer issues
		result = append(result, BuildPumpfuninternalSnapshotResp(&snap))
	}
	c.JSON(http.StatusOK, result)
}

// GetPumpfuninternalSnapshot 获取指定ID的 Pumpfuninternal 快照
func GetPumpfuninternalSnapshot(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var snapshot models.PumpfuninternalSnapshot
	if err := dbconfig.DB.First(&snapshot, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, BuildPumpfuninternalSnapshotResp(&snapshot))
}

// GetPumpfuninternalSnapshotByProject 获取指定项目的 Pumpfuninternal 快照
func GetPumpfuninternalSnapshotByProject(c *gin.Context) {
	projectID := c.Param("project_id")

	var req struct {
		SnapshotID uint `json:"snapshot_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	var snapshot models.PumpfuninternalSnapshot
	query := dbconfig.DB.Where("project_id = ?", projectID)
	if req.SnapshotID > 0 {
		query = query.Where("snapshot_id = ?", req.SnapshotID)
	}
	if err := query.First(&snapshot).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found"})
		return
	}
	c.JSON(http.StatusOK, BuildPumpfuninternalSnapshotResp(&snapshot))
}

// ListPumpfuninternalSnapshotsBySnapshotID 获取指定快照ID的所有 Pumpfuninternal 快照
func ListPumpfuninternalSnapshotsBySnapshotID(c *gin.Context) {
	snapshotID := c.Param("snapshot_id")
	var snapshots []models.PumpfuninternalSnapshot
	if err := dbconfig.DB.Where("snapshot_id = ?", snapshotID).Find(&snapshots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var result []*PumpfuninternalSnapshotResp
	for _, snapshot := range snapshots {
		snap := snapshot // Create a new variable to avoid pointer issues
		result = append(result, BuildPumpfuninternalSnapshotResp(&snap))
	}
	c.JSON(http.StatusOK, result)
}
