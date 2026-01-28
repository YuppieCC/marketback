package business

import (
	"errors"
	"math"
	"strconv"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/utils"

	"github.com/sirupsen/logrus"
)

const RENT_FUND = 0.00203928

// IGNORE_METEORA_RETAIL_ADDRESS 需要忽略的 Meteora 散户地址列表
var IGNORE_METEORA_RETAIL_ADDRESS = []string{
	// 在这里添加需要忽略的地址
	"FhVo3mqL8PW5pH5U2CN4XE33DokiyZnUwuGpH2hmHLuM",
	"HLnpSz9h2S4hiLQ43rnSD9XkcUThA7B8hQMKmDaiTLcC",
}

// SettleStats 结构体用于统一返回结算数据
type SettleStats struct {
	PoolInitialToken                 float64 `json:"pool_initial_token"`
	RetailInvestorsInitialToken      float64 `json:"retail_investors_initial_token"`
	ProjectInitialSol                float64 `json:"project_initial_sol"`
	ProjectInitialToken              float64 `json:"project_initial_token"`
	TokenByProject                   float64 `json:"token_by_project"`
	TokenByPool                      float64 `json:"token_by_pool"`
	TokenByRetailInvestors           float64 `json:"token_by_retail_investors"`
	TokenByExpectPool                float64 `json:"token_by_expect_pool"`
	TokenAllocationByRetailInvestors float64 `json:"token_allocation_by_retail_investors"`
	TokenAllocationByProject         float64 `json:"token_allocation_by_project"`
	TokenAllocationByPool            float64 `json:"token_allocation_by_pool"`
	TokenAllocationTotal             float64 `json:"token_allocation_total"`
	TvlByExpectPool                  float64 `json:"tvl_by_expect_pool"`
	TvlByProjectToken                float64 `json:"tvl_by_project_token"`
	TvlByProject                     float64 `json:"tvl_by_project"`
	TvlByRetailInvestors             float64 `json:"tvl_by_retail_investors"`
	ProjectPnl                       float64 `json:"project_pnl"`
	ProjectMinPnl                    float64 `json:"project_min_pnl"`
	ProjectPnlWithRent               float64 `json:"project_pnl_with_rent"`
	ProjectControlDifficulty         float64 `json:"project_control_difficulty"`
	PoolMonitorLastExecution         int64   `json:"pool_monitor_last_execution"`
	TvlByProjectInitialToken         float64 `json:"tvl_by_project_initial_token"`
	TvlByProjectRent                 float64 `json:"tvl_by_project_rent"`
	SolByRetailInvestors             float64 `json:"sol_by_retail_investors"`
}

// getProjectInitialFunds 获取项目初始资金
func getProjectInitialFunds(projectID uint, mint string, targetName string) (float64, error) {
	var transferRecords []models.ProjectFundTransferRecord
	var initialAmount float64 = 0

	if err := dbconfig.DB.Where("project_id = ? AND mint = ? AND target_name = ?",
		projectID, mint, targetName).Find(&transferRecords).Error; err != nil {
		return 0, err
	}

	for _, record := range transferRecords {
		if record.Direction == "in" {
			initialAmount += record.Amount
		} else if record.Direction == "out" {
			initialAmount -= record.Amount
		}
	}

	return initialAmount, nil
}

// shouldIgnoreRetailAddress 检查地址是否在忽略列表中
func shouldIgnoreRetailAddress(address string) bool {
	for _, ignoreAddr := range IGNORE_METEORA_RETAIL_ADDRESS {
		if address == ignoreAddr {
			return true
		}
	}
	return false
}

// CalculatePumpfunPoolSettle 计算 Pumpfun Internal Pool 的结算数据
func CalculatePumpfunPoolSettle(project *models.ProjectConfig, pumpfunStat *models.PumpfuninternalStat) (*SettleStats, error) {
	if project == nil || project.Token == nil {
		return nil, errors.New("invalid project or token configuration")
	}

	// 1. 获取监控配置
	var monitorConfig models.TransactionsMonitorConfig
	if err := dbconfig.DB.Where("address = ?", pumpfunStat.PumpfunPool.AssociatedBondingCurve).
		First(&monitorConfig).Error; err != nil {
		return nil, err
	}

	// 2. 计算项目代币变化 (HolderType == "project")
	var projectHolders []models.PumpfuninternalHolder
	var tokenChangeByProject float64 = 0
	var solChangeByProject float64 = 0
	var projectHoldersCount float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND bonding_curve_pda = ? AND last_slot <= ? AND start_slot >= ?",
		"project",
		pumpfunStat.PumpfunPool.BondingCurvePda,
		monitorConfig.LastSlot,
		monitorConfig.StartSlot).Find(&projectHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range projectHolders {
		if holder.MintChange > 0 {
			projectHoldersCount++
		}
		tokenChangeByProject += holder.MintChange
		solChangeByProject += (holder.SolChange / 1e9)
	}

	// 3. 获取初始资金数据
	projectInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "project")
	if err != nil {
		return nil, err
	}

	if projectInitialToken == 0 {
		projectInitialToken = 0.0000001
	}

	poolInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "pool")
	if err != nil {
		return nil, err
	}

	retailInvestorsInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "retail_investors")
	if err != nil {
		return nil, err
	}

	projectInitialSol, err := getProjectInitialFunds(project.ID, "sol", "project")
	if err != nil {
		return nil, err
	}

	// 4. 计算池子代币变化 (HolderType == "pool")
	var poolHolders []models.PumpfuninternalHolder
	var tokenChangeByPool float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND bonding_curve_pda = ? AND last_slot <= ? AND start_slot >= ?",
		"pool",
		pumpfunStat.PumpfunPool.BondingCurvePda,
		monitorConfig.LastSlot,
		monitorConfig.StartSlot).Find(&poolHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range poolHolders {
		tokenChangeByPool += holder.MintChange
	}

	// 5. 计算散户代币变化 (HolderType == "retail_investors")
	var retailHolders []models.PumpfuninternalHolder
	var tokenChangeByRetailInvestors float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND bonding_curve_pda = ? AND last_slot <= ? AND start_slot >= ?",
		"retail_investors",
		pumpfunStat.PumpfunPool.BondingCurvePda,
		monitorConfig.LastSlot,
		monitorConfig.StartSlot).Find(&retailHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range retailHolders {
		tokenChangeByRetailInvestors += holder.MintChange
	}

	// 6. 计算最终数值
	tokenByProject := projectInitialToken + tokenChangeByProject
	// 如果 tokenByProject 为 0，则设置为 1
	if tokenByProject == 0 {
		tokenByProject = 1
	}

	tokenByPool := poolInitialToken + tokenChangeByPool
	tokenByRetailInvestors := retailInvestorsInitialToken + tokenChangeByRetailInvestors
	tokenByExpectPool := tokenByProject + tokenByRetailInvestors

	// 7. 计算虚拟储备
	virtualSolReserves, virtualTokenReserves := utils.GetVirtualReserves(tokenByPool)
	if virtualSolReserves == 0 || virtualTokenReserves == 0 {
		if virtualSolReserves == 0 {
			virtualSolReserves = 0.000001
		}
		if virtualTokenReserves == 0 {
			virtualTokenReserves = 0.000001
		}
	}

	// 8. 计算市值
	result, err := utils.SimulateBondingCurveAmountOut(tokenByProject, "token", virtualSolReserves, virtualTokenReserves, pumpfunStat.PumpfunPool.FeeRate)
	if err != nil {
		return nil, err
	}
	tvlByProjectToken := result.GetAmount

	result, err = utils.SimulateBondingCurveAmountOut(tokenByRetailInvestors, "token", virtualSolReserves, virtualTokenReserves, pumpfunStat.PumpfunPool.FeeRate)
	if err != nil {
		return nil, err
	}
	tvlByRetailInvestors := result.GetAmount

	result, err = utils.SimulateBondingCurveAmountOut(tokenByExpectPool, "token", virtualSolReserves, virtualTokenReserves, pumpfunStat.PumpfunPool.FeeRate)
	if err != nil {
		return nil, err
	}
	tvlByExpectPool := result.GetAmount

	// 9. 计算 PNL 相关数据
	projectSolBalance := solChangeByProject + projectInitialSol
	tvlByProjectTokenLastSoldOut := tvlByExpectPool - tvlByRetailInvestors
	projectPnl := tvlByProjectToken + solChangeByProject
	projectMinPnl := tvlByProjectTokenLastSoldOut + solChangeByProject
	projectControlDifficulty := projectPnl - projectMinPnl

	// 10. 构建返回数据
	return &SettleStats{
		PoolInitialToken:                 poolInitialToken,
		RetailInvestorsInitialToken:      retailInvestorsInitialToken,
		ProjectInitialSol:                projectInitialSol,
		ProjectInitialToken:              projectInitialToken,
		TokenByProject:                   tokenByProject,
		TokenByPool:                      tokenByPool,
		TokenByRetailInvestors:           tokenByRetailInvestors,
		TokenByExpectPool:                tokenByExpectPool,
		TokenAllocationByRetailInvestors: tokenByRetailInvestors / project.Token.TotalSupply,
		TokenAllocationByProject:         tokenByProject / project.Token.TotalSupply,
		TokenAllocationByPool:            tokenByPool / project.Token.TotalSupply,
		TokenAllocationTotal:             (tokenByProject + tokenByRetailInvestors + tokenByPool) / project.Token.TotalSupply,
		TvlByExpectPool:                  tvlByExpectPool,
		TvlByProjectToken:                tvlByProjectToken,
		TvlByProject:                     tvlByProjectToken + projectSolBalance,
		TvlByRetailInvestors:             tvlByRetailInvestors,
		ProjectPnl:                       projectPnl,
		ProjectMinPnl:                    projectMinPnl,
		ProjectPnlWithRent:               projectPnl + projectHoldersCount*RENT_FUND,
		ProjectControlDifficulty:         projectControlDifficulty,
		PoolMonitorLastExecution:         int64(monitorConfig.LastExecution),
		TvlByProjectInitialToken:         0,
		TvlByProjectRent:                 projectHoldersCount * RENT_FUND,
		SolByRetailInvestors:             0,
	}, nil
}

// CalculatePumpfunAmmPoolSettle 计算 Pumpfun AMM Pool 的结算数据
func CalculatePumpfunAmmPoolSettle(project *models.ProjectConfig, pumpfunStat *models.PumpfunAmmPoolStat) (*SettleStats, error) {
	if project == nil || project.Token == nil {
		return nil, errors.New("invalid project or token configuration")
	}

	// 1. 获取池子配置
	var pumpfunPool models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.First(&pumpfunPool, project.PoolID).Error; err != nil {
		return nil, err
	}

	// 2. 获取监控配置
	var monitorConfig models.TransactionsMonitorConfig
	if err := dbconfig.DB.Where("address = ?", pumpfunPool.PoolAddress).First(&monitorConfig).Error; err != nil {
		return nil, err
	}

	// 3. 计算项目代币变化 (HolderType == "project")
	var projectHolders []models.PumpfunAmmpoolHolder
	var tokenChangeByProject float64 = 0
	var solChangeByProject float64 = 0
	var projectHoldersCount float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ? AND last_slot <= ? AND start_slot >= ?",
		"project",
		pumpfunPool.PoolAddress,
		monitorConfig.LastSlot,
		monitorConfig.StartSlot).Find(&projectHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range projectHolders {
		baseChange := holder.BaseChange
		solChange := holder.SolChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(solChange) || math.IsInf(solChange, 0) {
			solChange = 0.0
		}

		if baseChange > 0 {
			projectHoldersCount++
		}
		tokenChangeByProject += baseChange
		solChangeByProject += (solChange / 1e9)
	}

	// 4. 获取初始资金数据
	projectInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "project")
	if err != nil {
		return nil, err
	}

	poolInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "pool")
	if err != nil {
		return nil, err
	}

	poolInitialSol, err := getProjectInitialFunds(project.ID, "sol", "pool")
	if err != nil {
		return nil, err
	}

	retailInvestorsInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "retail_investors")
	if err != nil {
		return nil, err
	}

	projectInitialSol, err := getProjectInitialFunds(project.ID, "sol", "project")
	if err != nil {
		return nil, err
	}

	// 5. 计算池子代币变化 (HolderType == "pool")
	var poolHolders []models.PumpfunAmmpoolHolder
	var tokenChangeByPool float64 = 0
	var solChangeByPool float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"pool",
		pumpfunPool.PoolAddress).Find(&poolHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range poolHolders {
		baseChange := holder.BaseChange
		quoteChange := holder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}

		tokenChangeByPool += baseChange
		solChangeByPool += quoteChange
	}

	// 6. 计算散户代币变化 (HolderType == "retail_investors")
	var retailHolders []models.PumpfunAmmpoolHolder
	var tokenChangeByRetailInvestors float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"retail_investors",
		pumpfunPool.PoolAddress).Find(&retailHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range retailHolders {
		baseChange := holder.BaseChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}

		tokenChangeByRetailInvestors += baseChange
	}

	// 7. 计算最终数值
	tokenByProject := projectInitialToken + tokenChangeByProject

	// 如果 tokenByProject 为 0，则设置为 1
	if tokenByProject == 0 {
		tokenByProject = 1
	}

	tokenByPool := poolInitialToken + tokenChangeByPool
	solByPool := poolInitialSol + solChangeByPool
	tokenByRetailInvestors := retailInvestorsInitialToken + tokenChangeByRetailInvestors
	tokenByExpectPool := tokenByProject + tokenByRetailInvestors

	// 8. 计算市值
	fee := 0.0025 // AMM pool 固定费率
	tvlByProjectToken := utils.SimulateConstantProductAmountOut(tokenByProject, "x", tokenByPool, solByPool, fee)
	tvlByRetailInvestors := utils.SimulateConstantProductAmountOut(tokenByRetailInvestors, "x", tokenByPool, solByPool, fee)
	tvlByExpectPool := utils.SimulateConstantProductAmountOut(tokenByExpectPool, "x", tokenByPool, solByPool, fee)
	tvlByProjectInitialToken := utils.SimulateConstantProductAmountOut(projectInitialToken, "x", poolInitialToken, poolInitialSol, fee)

	// 9. 计算 PNL 相关数据
	projectSolBalance := solChangeByProject + projectInitialSol
	tvlByProjectTokenLastSoldOut := tvlByExpectPool - tvlByRetailInvestors
	projectPnl := tvlByProjectToken + solChangeByProject - tvlByProjectInitialToken
	projectMinPnl := tvlByProjectTokenLastSoldOut + solChangeByProject - tvlByProjectInitialToken
	projectControlDifficulty := projectPnl - projectMinPnl

	// 10. 构建返回数据
	return &SettleStats{
		PoolInitialToken:                 poolInitialToken,
		RetailInvestorsInitialToken:      retailInvestorsInitialToken,
		ProjectInitialSol:                projectInitialSol,
		ProjectInitialToken:              projectInitialToken,
		TokenByProject:                   tokenByProject,
		TokenByPool:                      tokenByPool,
		TokenByRetailInvestors:           tokenByRetailInvestors,
		TokenByExpectPool:                tokenByExpectPool,
		TokenAllocationByRetailInvestors: tokenByRetailInvestors / project.Token.TotalSupply,
		TokenAllocationByProject:         tokenByProject / project.Token.TotalSupply,
		TokenAllocationByPool:            tokenByPool / project.Token.TotalSupply,
		TokenAllocationTotal:             (tokenByProject + tokenByRetailInvestors + tokenByPool) / project.Token.TotalSupply,
		TvlByExpectPool:                  tvlByExpectPool,
		TvlByProjectToken:                tvlByProjectToken,
		TvlByProject:                     tvlByProjectToken + projectSolBalance,
		TvlByRetailInvestors:             tvlByRetailInvestors,
		ProjectPnl:                       projectPnl,
		ProjectMinPnl:                    projectMinPnl,
		ProjectPnlWithRent:               projectPnl + projectHoldersCount*RENT_FUND,
		ProjectControlDifficulty:         projectControlDifficulty,
		PoolMonitorLastExecution:         int64(monitorConfig.LastExecution),
		TvlByProjectInitialToken:         tvlByProjectInitialToken,
		TvlByProjectRent:                 projectHoldersCount * RENT_FUND,
		SolByRetailInvestors:             0,
	}, nil
}

// CalculateRaydiumCpmmSettle 计算 Raydium CPMM Pool 的结算数据
func CalculateRaydiumCpmmSettle(project *models.ProjectConfig, raydiumStat *models.RaydiumCpmmPoolStat) (*SettleStats, error) {
	if project == nil || project.Token == nil {
		return nil, errors.New("invalid project or token configuration")
	}

	// 1. 获取池子配置
	var raydiumPool models.RaydiumCpmmPoolConfig
	if err := dbconfig.DB.First(&raydiumPool, project.PoolID).Error; err != nil {
		return nil, err
	}

	// 2. 获取监控配置
	var monitorConfig models.TransactionsMonitorConfig
	if err := dbconfig.DB.Where("address = ?", raydiumPool.PoolAddress).First(&monitorConfig).Error; err != nil {
		return nil, err
	}

	// 定义筛选条件: base_mint 为 project.token.mint, quote_mint 为 WSOL
	quoteMint := "So11111111111111111111111111111111111111112" // WSOL address

	// 3. 计算项目代币变化 (HolderType == "project")
	var projectHolders []models.RaydiumPoolHolder
	var tokenChangeByProject float64 = 0
	var solChangeByProject float64 = 0
	var projectHoldersCount float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND base_mint = ? AND quote_mint = ?",
		"project",
		project.Token.Mint,
		quoteMint).Find(&projectHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range projectHolders {
		baseChange := holder.BaseChange
		solChange := holder.SolChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(solChange) || math.IsInf(solChange, 0) {
			solChange = 0.0
		}

		if baseChange > 0 {
			projectHoldersCount++
		}
		tokenChangeByProject += baseChange
		solChangeByProject += (solChange / 1e9)
	}

	// 4. 获取初始资金数据
	projectInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "project")
	if err != nil {
		return nil, err
	}

	poolInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "pool")
	if err != nil {
		return nil, err
	}

	poolInitialSol, err := getProjectInitialFunds(project.ID, "sol", "pool")
	if err != nil {
		return nil, err
	}

	retailInvestorsInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "retail_investors")
	if err != nil {
		return nil, err
	}

	projectInitialSol, err := getProjectInitialFunds(project.ID, "sol", "project")
	if err != nil {
		return nil, err
	}

	// 5. 计算池子代币变化 (HolderType == "pool")
	var poolHolders []models.RaydiumPoolHolder
	var tokenChangeByPool float64 = 0
	var solChangeByPool float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND base_mint = ? AND quote_mint = ?",
		"pool",
		project.Token.Mint,
		quoteMint).Find(&poolHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range poolHolders {
		baseChange := holder.BaseChange
		quoteChange := holder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}

		tokenChangeByPool += baseChange
		solChangeByPool += quoteChange
	}

	// 6. 计算散户代币变化 (HolderType == "retail_investors")
	var retailHolders []models.RaydiumPoolHolder
	var tokenChangeByRetailInvestors float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND base_mint = ? AND quote_mint = ?",
		"retail_investors",
		project.Token.Mint,
		quoteMint).Find(&retailHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range retailHolders {
		baseChange := holder.BaseChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}

		tokenChangeByRetailInvestors += baseChange
	}

	// 7. 计算最终数值
	tokenByProject := projectInitialToken + tokenChangeByProject
	tokenByPool := poolInitialToken + tokenChangeByPool
	solByPool := poolInitialSol + solChangeByPool
	tokenByRetailInvestors := retailInvestorsInitialToken + tokenChangeByRetailInvestors
	tokenByExpectPool := tokenByProject + tokenByRetailInvestors

	// 8. 计算市值
	fee := raydiumPool.FeeRate // 使用配置中的费率
	tvlByProjectToken := utils.SimulateConstantProductAmountOut(tokenByProject, "x", tokenByPool, solByPool, fee)
	tvlByRetailInvestors := utils.SimulateConstantProductAmountOut(tokenByRetailInvestors, "x", tokenByPool, solByPool, fee)
	tvlByExpectPool := utils.SimulateConstantProductAmountOut(tokenByExpectPool, "x", tokenByPool, solByPool, fee)
	tvlByProjectInitialToken := utils.SimulateConstantProductAmountOut(projectInitialToken, "x", poolInitialToken, poolInitialSol, fee)

	// 9. 计算 PNL 相关数据
	projectSolBalance := solChangeByProject + projectInitialSol
	tvlByProjectTokenLastSoldOut := tvlByExpectPool - tvlByRetailInvestors
	projectPnl := tvlByProjectToken + solChangeByProject - tvlByProjectInitialToken
	projectMinPnl := tvlByProjectTokenLastSoldOut + solChangeByProject - tvlByProjectInitialToken
	projectControlDifficulty := projectPnl - projectMinPnl

	// 10. 构建返回数据
	return &SettleStats{
		PoolInitialToken:                 poolInitialToken,
		RetailInvestorsInitialToken:      retailInvestorsInitialToken,
		ProjectInitialSol:                projectInitialSol,
		ProjectInitialToken:              projectInitialToken,
		TokenByProject:                   tokenByProject,
		TokenByPool:                      tokenByPool,
		TokenByRetailInvestors:           tokenByRetailInvestors,
		TokenByExpectPool:                tokenByExpectPool,
		TokenAllocationByRetailInvestors: tokenByRetailInvestors / project.Token.TotalSupply,
		TokenAllocationByProject:         tokenByProject / project.Token.TotalSupply,
		TokenAllocationByPool:            tokenByPool / project.Token.TotalSupply,
		TokenAllocationTotal:             (tokenByProject + tokenByRetailInvestors + tokenByPool) / project.Token.TotalSupply,
		TvlByExpectPool:                  tvlByExpectPool,
		TvlByProjectToken:                tvlByProjectToken,
		TvlByProject:                     tvlByProjectToken + projectSolBalance,
		TvlByRetailInvestors:             tvlByRetailInvestors,
		ProjectPnl:                       projectPnl,
		ProjectMinPnl:                    projectMinPnl,
		ProjectPnlWithRent:               projectPnl + projectHoldersCount*RENT_FUND,
		ProjectControlDifficulty:         projectControlDifficulty,
		PoolMonitorLastExecution:         int64(monitorConfig.LastExecution),
		TvlByProjectInitialToken:         tvlByProjectInitialToken,
		TvlByProjectRent:                 projectHoldersCount * RENT_FUND,
		SolByRetailInvestors:             0,
	}, nil
}

// CalculatePumpfunCombinedPoolSettle 计算结合了 PumpfunAmmPoolHolder 和 PumpfuninternalHolder 的结算数据
func CalculatePumpfunCombinedPoolSettle(project *models.ProjectConfig, pumpfunStat *models.PumpfunAmmPoolStat) (*SettleStats, error) {
	if project == nil || project.Token == nil {
		return nil, errors.New("invalid project or token configuration")
	}

	// 1. 获取池子配置
	var pumpfunPool models.PumpfunAmmPoolConfig
	if err := dbconfig.DB.First(&pumpfunPool, project.PoolID).Error; err != nil {
		return nil, err
	}

	// 2. 获取监控配置
	var monitorConfig models.TransactionsMonitorConfig
	if err := dbconfig.DB.Where("address = ?", pumpfunPool.PoolAddress).First(&monitorConfig).Error; err != nil {
		return nil, err
	}

	// 3. 获取 PumpfunAmmpoolHolder 数据 (HolderType == "project")
	var projectHolders []models.PumpfunAmmpoolHolder
	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"project",
		pumpfunPool.PoolAddress).Find(&projectHolders).Error; err != nil {
		return nil, err
	}

	// 4. 获取相关的 PumpfuninternalHolder 数据 (Mint 为 BaseMint 且 holder_type 为 "project" 或 "retail_investors")
	var internalHolders []models.PumpfuninternalHolder
	if err := dbconfig.DB.Where("holder_type IN (?, ?) AND mint = ?",
		"project", "retail_investors",
		pumpfunPool.BaseMint).Find(&internalHolders).Error; err != nil {
		return nil, err
	}

	// 5. 创建 address 到 PumpfuninternalHolder 的映射
	internalHolderMap := make(map[string]*models.PumpfuninternalHolder)
	for i := range internalHolders {
		internalHolderMap[internalHolders[i].Address] = &internalHolders[i]
	}

	// 6. 计算项目代币变化，结合两个数据源
	var tokenChangeByProject float64 = 0
	var solChangeByProject float64 = 0
	var projectHoldersCount float64 = 0

	// 创建一个映射来合并所有项目持有者地址
	projectAddressMap := make(map[string]bool)

	// 收集所有项目持有者地址
	for _, holder := range projectHolders {
		projectAddressMap[holder.Address] = true
	}

	// 从内部持有者中添加项目地址
	for _, holder := range internalHolders {
		if holder.HolderType == "project" {
			projectAddressMap[holder.Address] = true
		}
	}

	// 遍历所有项目持有者地址进行合并计算
	for address := range projectAddressMap {
		var baseChange, solChange float64

		// 查找 AMM 池数据
		for _, holder := range projectHolders {
			if holder.Address == address {
				baseChange += holder.BaseChange
				solChange += holder.SolChange
				break
			}
		}

		// 查找内部持有者数据
		if internalHolder, exists := internalHolderMap[address]; exists && internalHolder.HolderType == "project" {
			baseChange += internalHolder.MintChange
			solChange += internalHolder.SolChange
		}

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(solChange) || math.IsInf(solChange, 0) {
			solChange = 0.0
		}

		if baseChange > 0 {
			projectHoldersCount++
		}
		tokenChangeByProject += baseChange
		solChangeByProject += solChange / 1e9
	}

	// 7. 获取初始资金数据
	projectInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "project")
	if err != nil {
		return nil, err
	}

	if projectInitialToken == 0 {
		projectInitialToken = 0.0000001
	}

	poolInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "pool")
	if err != nil {
		return nil, err
	}

	poolInitialSol, err := getProjectInitialFunds(project.ID, "sol", "pool")
	if err != nil {
		return nil, err
	}

	retailInvestorsInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "retail_investors")
	if err != nil {
		return nil, err
	}

	projectInitialSol, err := getProjectInitialFunds(project.ID, "sol", "project")
	if err != nil {
		return nil, err
	}

	// 8. 计算池子代币变化 (HolderType == "pool")
	var poolHolders []models.PumpfunAmmpoolHolder
	var tokenChangeByPool float64 = 0
	var solChangeByPool float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ? AND last_slot <= ? AND start_slot >= ?",
		"pool",
		pumpfunPool.PoolAddress,
		monitorConfig.LastSlot,
		monitorConfig.StartSlot).Find(&poolHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range poolHolders {
		baseChange := holder.BaseChange
		quoteChange := holder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}

		tokenChangeByPool += baseChange
		solChangeByPool += quoteChange
	}

	// 9. 计算散户代币变化 (HolderType == "retail_investors")，结合两个数据源
	var retailHolders []models.PumpfunAmmpoolHolder
	var tokenChangeByRetailInvestors float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"retail_investors",
		pumpfunPool.PoolAddress).Find(&retailHolders).Error; err != nil {
		return nil, err
	}

	// 创建一个映射来合并所有散户持有者地址
	retailAddressMap := make(map[string]bool)

	// 收集所有散户持有者地址
	for _, holder := range retailHolders {
		retailAddressMap[holder.Address] = true
	}

	// 从内部持有者中添加散户地址
	for _, holder := range internalHolders {
		if holder.HolderType == "retail_investors" {
			retailAddressMap[holder.Address] = true
		}
	}

	// 遍历所有散户持有者地址进行合并计算
	for address := range retailAddressMap {
		var baseChange, solChange float64

		// 查找 AMM 池数据
		for _, holder := range retailHolders {
			if holder.Address == address {
				baseChange += holder.BaseChange
				solChange += holder.SolChange
				break
			}
		}

		// 查找内部持有者数据
		if internalHolder, exists := internalHolderMap[address]; exists && internalHolder.HolderType == "retail_investors" {
			baseChange += internalHolder.MintChange
			solChange += internalHolder.SolChange
		}

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(solChange) || math.IsInf(solChange, 0) {
			solChange = 0.0
		}

		tokenChangeByRetailInvestors += baseChange
		// 注意：散户的 SOL 变化通常不影响项目计算，但为了一致性也进行处理
	}

	// 10. 计算最终数值
	tokenByProject := projectInitialToken + tokenChangeByProject

	// 如果 tokenByProject 为 0，则设置为 1
	if tokenByProject == 0 {
		tokenByProject = 1
	}

	tokenByPool := poolInitialToken + tokenChangeByPool
	solByPool := poolInitialSol + solChangeByPool
	tokenByRetailInvestors := retailInvestorsInitialToken + tokenChangeByRetailInvestors
	tokenByExpectPool := tokenByProject + tokenByRetailInvestors

	// 11. 计算市值
	fee := 0.0025 // AMM pool 固定费率
	tvlByProjectToken := utils.SimulateConstantProductAmountOut(tokenByProject, "x", tokenByPool, solByPool, fee)
	tvlByRetailInvestors := utils.SimulateConstantProductAmountOut(tokenByRetailInvestors, "x", tokenByPool, solByPool, fee)
	tvlByExpectPool := utils.SimulateConstantProductAmountOut(tokenByExpectPool, "x", tokenByPool, solByPool, fee)
	tvlByProjectInitialToken := utils.SimulateConstantProductAmountOut(projectInitialToken, "x", poolInitialToken, poolInitialSol, fee)

	// 12. 计算 PNL 相关数据
	projectSolBalance := solChangeByProject + projectInitialSol
	tvlByProjectTokenLastSoldOut := tvlByExpectPool - tvlByRetailInvestors
	projectPnl := tvlByProjectToken + solChangeByProject - tvlByProjectInitialToken
	projectMinPnl := tvlByProjectTokenLastSoldOut + solChangeByProject - tvlByProjectInitialToken
	projectControlDifficulty := projectPnl - projectMinPnl

	// 13. 构建返回数据
	return &SettleStats{
		PoolInitialToken:                 poolInitialToken,
		RetailInvestorsInitialToken:      retailInvestorsInitialToken,
		ProjectInitialSol:                projectInitialSol,
		ProjectInitialToken:              projectInitialToken,
		TokenByProject:                   tokenByProject,
		TokenByPool:                      tokenByPool,
		TokenByRetailInvestors:           tokenByRetailInvestors,
		TokenByExpectPool:                tokenByExpectPool,
		TokenAllocationByRetailInvestors: tokenByRetailInvestors / project.Token.TotalSupply,
		TokenAllocationByProject:         tokenByProject / project.Token.TotalSupply,
		TokenAllocationByPool:            tokenByPool / project.Token.TotalSupply,
		TokenAllocationTotal:             (tokenByProject + tokenByRetailInvestors + tokenByPool) / project.Token.TotalSupply,
		TvlByExpectPool:                  tvlByExpectPool,
		TvlByProjectToken:                tvlByProjectToken,
		TvlByProject:                     tvlByProjectToken + projectSolBalance,
		TvlByRetailInvestors:             tvlByRetailInvestors,
		ProjectPnl:                       projectPnl,
		ProjectMinPnl:                    projectMinPnl,
		ProjectPnlWithRent:               projectPnl + projectHoldersCount*RENT_FUND,
		ProjectControlDifficulty:         projectControlDifficulty,
		PoolMonitorLastExecution:         int64(monitorConfig.LastExecution),
		TvlByProjectInitialToken:         tvlByProjectInitialToken,
		TvlByProjectRent:                 projectHoldersCount * RENT_FUND,
		SolByRetailInvestors:             0,
	}, nil
}

// CalculateMeteoracpmmPoolSettle 计算 Meteoracpmm Pool 的结算数据
func CalculateMeteoracpmmPoolSettle(project *models.ProjectConfig, meteoracpmmStat *models.MeteoracpmmPoolStat) (*SettleStats, error) {
	if project == nil || project.Token == nil {
		return nil, errors.New("invalid project or token configuration")
	}

	// 1. 获取池子配置
	var meteoracpmmPool models.MeteoracpmmConfig
	if err := dbconfig.DB.First(&meteoracpmmPool, project.PoolID).Error; err != nil {
		return nil, err
	}

	// 2. 获取监控配置
	var monitorConfig models.TransactionsMonitorConfig
	if err := dbconfig.DB.Where("address = ?", meteoracpmmPool.PoolAddress).First(&monitorConfig).Error; err != nil {
		return nil, err
	}

	// 3. 计算项目代币变化 (HolderType == "project")
	var projectHolders []models.MeteoracpmmHolder
	var tokenChangeByProject float64 = 0
	var solChangeByProject float64 = 0
	var projectHoldersCount float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"project",
		meteoracpmmPool.PoolAddress).Find(&projectHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range projectHolders {
		baseChange := holder.BaseChange
		solChange := holder.SolChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(solChange) || math.IsInf(solChange, 0) {
			solChange = 0.0
		}

		if baseChange > 0.0000001 {
			projectHoldersCount++
		}
		tokenChangeByProject += baseChange
		solChangeByProject += (solChange / 1e9)
	}

	// 4. 获取初始资金数据
	projectInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "project")
	if err != nil {
		return nil, err
	}

	poolInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "pool")
	if err != nil {
		return nil, err
	}

	retailInvestorsInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "retail_investors")
	if err != nil {
		return nil, err
	}

	projectInitialSol, err := getProjectInitialFunds(project.ID, "sol", "project")
	if err != nil {
		return nil, err
	}

	// 5. 计算池子代币变化 (HolderType == "pool")
	// 辅助函数：处理 holder 变化并累加
	processHolderChanges := func(baseChange, quoteChange float64) (float64, float64) {
		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}
		return baseChange, quoteChange
	}

	var tokenChangeByPool float64 = 0
	var solChangeByPool float64 = 0

	// 查询 MeteoracpmmHolder
	var poolHolders []models.MeteoracpmmHolder
	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"pool",
		meteoracpmmPool.PoolAddress).Find(&poolHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range poolHolders {
		baseChange, quoteChange := processHolderChanges(holder.BaseChange, holder.QuoteChange)
		tokenChangeByPool += baseChange
		solChangeByPool += quoteChange
	}

	// 6. 计算散户代币变化 (HolderType == "retail_investors")
	var retailHolders []models.MeteoracpmmHolder
	var tokenChangeByRetailInvestors float64 = 0
	var solByRetailInvestors float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"retail_investors",
		meteoracpmmPool.PoolAddress).Find(&retailHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range retailHolders {
		// 忽略列表中的地址
		if shouldIgnoreRetailAddress(holder.Address) {
			continue
		}

		baseChange := holder.BaseChange
		quoteChange := holder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}

		tokenChangeByRetailInvestors += baseChange

		// 如果 baseChange > 0，累加 quoteChange 的绝对值到 SolByRetailInvestors
		if baseChange > 0 {
			solByRetailInvestors += math.Abs(quoteChange)
		}
	}

	// 7. 计算最终数值
	tokenByProject := projectInitialToken + tokenChangeByProject

	// 如果 tokenByProject 为 0，则设置为 1
	if tokenByProject == 0 {
		tokenByProject = 1
	}

	tokenByPool := poolInitialToken + tokenChangeByPool
	tokenByRetailInvestors := retailInvestorsInitialToken + tokenChangeByRetailInvestors
	tokenByExpectPool := tokenByProject + tokenByRetailInvestors

	// 8. 使用 Jupiter API 计算市值
	// 将代币数量转换为最小单位 (假设代币有6位小数)
	tokenByProjectAmount := strconv.FormatInt(int64(tokenByProject*1e6), 10)
	tokenByRetailInvestorsAmount := strconv.FormatInt(int64(tokenByRetailInvestors*1e6), 10)
	tokenByExpectPoolAmount := strconv.FormatInt(int64(tokenByExpectPool*1e6), 10)
	projectInitialTokenAmount := strconv.FormatInt(int64(projectInitialToken*1e6), 10)

	// 使用 Jupiter API 获取价格
	wsolMint := "So11111111111111111111111111111111111111112"

	// 计算项目代币市值
	var tvlByProjectToken float64 = 0
	projectTokenQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, tokenByProjectAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for project token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(projectTokenQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse project token out amount: %v, setting to 0", err)
		} else {
			tvlByProjectToken = outAmount / 1e9
		}
	}

	// 计算散户代币市值
	var tvlByRetailInvestors float64 = 0
	retailTokenQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, tokenByRetailInvestorsAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for retail token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(retailTokenQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse retail token out amount: %v, setting to 0", err)
		} else {
			tvlByRetailInvestors = outAmount / 1e9
		}
	}

	// 计算期望池子代币市值
	var tvlByExpectPool float64 = 0
	expectPoolQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, tokenByExpectPoolAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for expect pool token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(expectPoolQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse expect pool token out amount: %v, setting to 0", err)
		} else {
			tvlByExpectPool = outAmount / 1e9
		}
	}

	// 计算项目初始代币市值
	var tvlByProjectInitialToken float64 = 0
	initialTokenQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, projectInitialTokenAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for initial token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(initialTokenQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse initial token out amount: %v, setting to 0", err)
		} else {
			tvlByProjectInitialToken = outAmount / 1e9
		}
	}

	// 9. 计算 PNL 相关数据
	projectSolBalance := solChangeByProject + projectInitialSol
	tvlByProjectTokenLastSoldOut := tvlByExpectPool - tvlByRetailInvestors
	projectPnl := tvlByProjectToken + solChangeByProject - tvlByProjectInitialToken
	projectMinPnl := tvlByProjectTokenLastSoldOut + solChangeByProject - tvlByProjectInitialToken
	projectControlDifficulty := projectPnl - projectMinPnl

	// 10. 构建返回数据
	return &SettleStats{
		PoolInitialToken:                 poolInitialToken,
		RetailInvestorsInitialToken:      retailInvestorsInitialToken,
		ProjectInitialSol:                projectInitialSol,
		ProjectInitialToken:              projectInitialToken,
		TokenByProject:                   tokenByProject,
		TokenByPool:                      tokenByPool,
		TokenByRetailInvestors:           tokenByRetailInvestors,
		TokenByExpectPool:                tokenByExpectPool,
		TokenAllocationByRetailInvestors: tokenByRetailInvestors / project.Token.TotalSupply,
		TokenAllocationByProject:         tokenByProject / project.Token.TotalSupply,
		TokenAllocationByPool:            tokenByPool / project.Token.TotalSupply,
		TokenAllocationTotal:             (tokenByProject + tokenByRetailInvestors + tokenByPool) / project.Token.TotalSupply,
		TvlByExpectPool:                  tvlByExpectPool,
		TvlByProjectToken:                tvlByProjectToken,
		TvlByProject:                     tvlByProjectToken + projectSolBalance,
		TvlByRetailInvestors:             tvlByRetailInvestors,
		ProjectPnl:                       projectPnl,
		ProjectMinPnl:                    projectMinPnl,
		ProjectPnlWithRent:               projectPnl + projectHoldersCount*RENT_FUND,
		ProjectControlDifficulty:         projectControlDifficulty,
		PoolMonitorLastExecution:         int64(monitorConfig.LastExecution),
		TvlByProjectInitialToken:         tvlByProjectInitialToken,
		TvlByProjectRent:                 projectHoldersCount * RENT_FUND,
		SolByRetailInvestors:             solByRetailInvestors,
	}, nil
}

// CalculateMeteoradbcPoolSettle 计算 Meteoradbc Pool 的结算数据
func CalculateMeteoradbcPoolSettle(project *models.ProjectConfig, meteoradbcStat *models.MeteoradbcPoolStat) (*SettleStats, error) {
	if project == nil || project.Token == nil {
		return nil, errors.New("invalid project or token configuration")
	}

	// 1. 获取池子配置
	var meteoradbcPool models.MeteoradbcConfig
	if err := dbconfig.DB.First(&meteoradbcPool, project.PoolID).Error; err != nil {
		return nil, err
	}

	// 2. 获取监控配置
	var monitorConfig models.TransactionsMonitorConfig
	if err := dbconfig.DB.Where("address = ?", meteoradbcPool.PoolAddress).First(&monitorConfig).Error; err != nil {
		return nil, err
	}

	// 3. 计算项目代币变化 (HolderType == "project")
	var projectHolders []models.MeteoradbcHolder
	var tokenChangeByProject float64 = 0
	var solChangeByProject float64 = 0
	var projectHoldersCount float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ? AND last_slot <= ? AND start_slot >= ?",
		"project",
		meteoradbcPool.PoolAddress,
		monitorConfig.LastSlot,
		monitorConfig.StartSlot).Find(&projectHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range projectHolders {
		baseChange := holder.BaseChange
		solChange := holder.SolChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(solChange) || math.IsInf(solChange, 0) {
			solChange = 0.0
		}

		if baseChange > 0.0000001 {
			projectHoldersCount++
		}
		tokenChangeByProject += baseChange
		solChangeByProject += (solChange / 1e9)
	}

	// 4. 获取初始资金数据
	projectInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "project")
	if err != nil {
		return nil, err
	}

	poolInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "pool")
	if err != nil {
		return nil, err
	}

	retailInvestorsInitialToken, err := getProjectInitialFunds(project.ID, project.Token.Mint, "retail_investors")
	if err != nil {
		return nil, err
	}

	projectInitialSol, err := getProjectInitialFunds(project.ID, "sol", "project")
	if err != nil {
		return nil, err
	}

	// 5. 计算池子代币变化 (HolderType == "pool")
	var poolHolders []models.MeteoradbcHolder
	var tokenChangeByPool float64 = 0
	var solChangeByPool float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"pool",
		meteoradbcPool.PoolAddress).Find(&poolHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range poolHolders {
		baseChange := holder.BaseChange
		quoteChange := holder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}

		tokenChangeByPool += baseChange
		solChangeByPool += quoteChange
	}

	// 6. 计算散户代币变化 (HolderType == "retail_investors")
	var retailHolders []models.MeteoradbcHolder
	var tokenChangeByRetailInvestors float64 = 0
	var solByRetailInvestors float64 = 0

	if err := dbconfig.DB.Where("holder_type = ? AND pool_address = ?",
		"retail_investors",
		meteoradbcPool.PoolAddress).Find(&retailHolders).Error; err != nil {
		return nil, err
	}

	for _, holder := range retailHolders {
		// 忽略列表中的地址
		if shouldIgnoreRetailAddress(holder.Address) {
			continue
		}

		baseChange := holder.BaseChange
		quoteChange := holder.QuoteChange

		// 检查并处理 NaN 值
		if math.IsNaN(baseChange) || math.IsInf(baseChange, 0) {
			baseChange = 0.0
		}
		if math.IsNaN(quoteChange) || math.IsInf(quoteChange, 0) {
			quoteChange = 0.0
		}

		tokenChangeByRetailInvestors += baseChange

		// 如果 baseChange > 0，累加 quoteChange 到 SolByRetailInvestors
		if baseChange > 0 {
			solByRetailInvestors += math.Abs(quoteChange)
		}
	}

	// 7. 计算最终数值
	tokenByProject := projectInitialToken + tokenChangeByProject

	// 如果 tokenByProject 为 0，则设置为 1
	if tokenByProject == 0 {
		tokenByProject = 1
	}

	tokenByPool := poolInitialToken + tokenChangeByPool
	tokenByRetailInvestors := retailInvestorsInitialToken + tokenChangeByRetailInvestors
	tokenByExpectPool := tokenByProject + tokenByRetailInvestors

	// 8. 使用 Jupiter API 计算市值
	// 将代币数量转换为最小单位 (假设代币有6位小数)
	tokenByProjectAmount := strconv.FormatInt(int64(tokenByProject*1e6), 10)
	tokenByRetailInvestorsAmount := strconv.FormatInt(int64(tokenByRetailInvestors*1e6), 10)
	tokenByExpectPoolAmount := strconv.FormatInt(int64(tokenByExpectPool*1e6), 10)
	projectInitialTokenAmount := strconv.FormatInt(int64(projectInitialToken*1e6), 10)

	// 使用 Jupiter API 获取价格
	wsolMint := "So11111111111111111111111111111111111111112"

	// 计算项目代币市值
	var tvlByProjectToken float64 = 0
	projectTokenQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, tokenByProjectAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for project token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(projectTokenQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse project token out amount: %v, setting to 0", err)
		} else {
			tvlByProjectToken = outAmount / 1e9
		}
	}

	// 计算散户代币市值
	var tvlByRetailInvestors float64 = 0
	retailTokenQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, tokenByRetailInvestorsAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for retail token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(retailTokenQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse retail token out amount: %v, setting to 0", err)
		} else {
			tvlByRetailInvestors = outAmount / 1e9
		}
	}

	// 计算期望池子代币市值
	var tvlByExpectPool float64 = 0
	expectPoolQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, tokenByExpectPoolAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for expect pool token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(expectPoolQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse expect pool token out amount: %v, setting to 0", err)
		} else {
			tvlByExpectPool = outAmount / 1e9
		}
	}

	// 计算项目初始代币市值
	var tvlByProjectInitialToken float64 = 0
	initialTokenQuote, err := utils.GetSwapResult(project.Token.Mint, wsolMint, projectInitialTokenAmount, 20)
	if err != nil {
		logrus.Warnf("Failed to get Jupiter quote for initial token: %v, setting to 0", err)
	} else {
		outAmount, err := strconv.ParseFloat(initialTokenQuote.OutAmount, 64)
		if err != nil {
			logrus.Warnf("Failed to parse initial token out amount: %v, setting to 0", err)
		} else {
			tvlByProjectInitialToken = outAmount / 1e9
		}
	}

	// 9. 计算 PNL 相关数据
	projectSolBalance := solChangeByProject + projectInitialSol
	tvlByProjectTokenLastSoldOut := tvlByExpectPool - tvlByRetailInvestors
	projectPnl := tvlByProjectToken + solChangeByProject - tvlByProjectInitialToken
	projectMinPnl := tvlByProjectTokenLastSoldOut + solChangeByProject - tvlByProjectInitialToken
	projectControlDifficulty := projectPnl - projectMinPnl

	// 10. 构建返回数据
	return &SettleStats{
		PoolInitialToken:                 poolInitialToken,
		RetailInvestorsInitialToken:      retailInvestorsInitialToken,
		ProjectInitialSol:                projectInitialSol,
		ProjectInitialToken:              projectInitialToken,
		TokenByProject:                   tokenByProject,
		TokenByPool:                      tokenByPool,
		TokenByRetailInvestors:           tokenByRetailInvestors,
		TokenByExpectPool:                tokenByExpectPool,
		TokenAllocationByRetailInvestors: tokenByRetailInvestors / project.Token.TotalSupply,
		TokenAllocationByProject:         tokenByProject / project.Token.TotalSupply,
		TokenAllocationByPool:            tokenByPool / project.Token.TotalSupply,
		TokenAllocationTotal:             (tokenByProject + tokenByRetailInvestors + tokenByPool) / project.Token.TotalSupply,
		TvlByExpectPool:                  tvlByExpectPool,
		TvlByProjectToken:                tvlByProjectToken,
		TvlByProject:                     tvlByProjectToken + projectSolBalance,
		TvlByRetailInvestors:             tvlByRetailInvestors,
		ProjectPnl:                       projectPnl,
		ProjectMinPnl:                    projectMinPnl,
		ProjectPnlWithRent:               projectPnl + projectHoldersCount*RENT_FUND,
		ProjectControlDifficulty:         projectControlDifficulty,
		PoolMonitorLastExecution:         int64(monitorConfig.LastExecution),
		TvlByProjectInitialToken:         tvlByProjectInitialToken,
		TvlByProjectRent:                 projectHoldersCount * RENT_FUND,
		SolByRetailInvestors:             solByRetailInvestors,
	}, nil
}
