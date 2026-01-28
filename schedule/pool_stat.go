package main

import (
	"context"
	"math"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"gorm.io/gorm"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	solanautil "marketcontrol/pkg/solana"

	"marketcontrol/pkg/utils"

	log "github.com/sirupsen/logrus"
)

const (
	POOL_MAX_CONCURRENT               = 2
	POOL_UPDATE_INTERVAL              = 1
	PUMPFUN_MAX_CONCURRENT            = 2
	PUMPFUN_UPDATE_INTERVAL           = 5
	PUMPFUN_AMM_MAX_CONCURRENT        = 2
	PUMPFUN_AMM_UPDATE_INTERVAL       = 5
	RAYDIUM_LAUNCHPAD_MAX_CONCURRENT  = 2
	RAYDIUM_LAUNCHPAD_UPDATE_INTERVAL = 1
	RAYDIUM_CPMM_MAX_CONCURRENT       = 2
	RAYDIUM_CPMM_UPDATE_INTERVAL      = 1
	METEORADBC_MAX_CONCURRENT         = 2
	METEORADBC_UPDATE_INTERVAL        = 1
	METEORACPMM_MAX_CONCURRENT        = 2
	METEORACPMM_UPDATE_INTERVAL       = 1
)

func main() {
	// 日志输出到文件
	os.MkdirAll("logs", 0755)
	file, err := os.OpenFile("logs/update_pool_stat_schedule.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(log.ErrorLevel)
	// log.SetLevel(log.InfoLevel)
	log.Infof("> 初始化程序完成")

	if err == nil {
		log.SetOutput(file)
	} else {
		log.Warn("无法打开日志文件，日志将输出到标准输出")
	}

	config.InitDB()

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		log.Fatal("Solana RPC endpoint not configured")
	}

	// Create client
	client := rpc.New(solanaRPC)
	log.Infof("> 初始化程序完成")

	for {
		time.Sleep(2 * time.Second)

		// 并行处理 PoolConfig、MeteoradbcConfig、MeteoracpmmConfig
		var wg sync.WaitGroup
		wg.Add(2)

		// // 处理 PoolConfig
		// go func() {
		// 	defer wg.Done()
		// 	UpdatePoolStats(client)
		// }()

		// // 处理 PumpfuninternalConfig
		// go func() {
		// 	defer wg.Done()
		// 	UpdatePumpfunInternalStats(client)
		// }()

		// // 处理 PumpfunAmmPoolConfig
		// go func() {
		// 	defer wg.Done()
		// 	UpdatePumpfunAmmPoolStats(client)
		// }()

		// // 处理 RaydiumLaunchpadPoolConfig
		// go func() {
		// 	defer wg.Done()
		// 	UpdateRaydiumLaunchpadPoolStats(client)
		// }()

		// // 处理 RaydiumCpmmPoolConfig
		// go func() {
		// 	defer wg.Done()
		// 	UpdateRaydiumCpmmPoolStats(client)
		// }()

		// 处理 MeteoradbcConfig
		go func() {
			defer wg.Done()
			UpdateMeteoradbcPoolStats(client)
		}()

		// 处理 MeteoracpmmConfig
		go func() {
			defer wg.Done()
			UpdateMeteoracpmmPoolStats(client)
		}()

		wg.Wait()
	}
}

func UpdatePoolStats(client *rpc.Client) {
	var pools []models.PoolConfig
	if err := config.DB.Preload("BaseMint").Preload("QuoteMint").Where("status = ?", "active").Find(&pools).Error; err != nil {
		log.Fatalf("> 查询池子失败: %v", err)
	}

	poolSem := make(chan struct{}, POOL_MAX_CONCURRENT)
	var poolWg sync.WaitGroup
	for _, pool := range pools {
		poolWg.Add(1)
		poolSem <- struct{}{}
		go func(pool models.PoolConfig) {
			defer poolWg.Done()
			defer func() { <-poolSem }()
			if pool.BaseMint == nil || pool.QuoteMint == nil {
				log.Warnf("池子 %d 未配置 BaseMint 或 QuoteMint，跳过", pool.ID)
				return
			}
			baseMint := pool.BaseMint.Mint
			quoteMint := pool.QuoteMint.Mint
			baseIsWsol := pool.BaseIsWSOL
			baseDecimalsPow := math.Pow(10, float64(pool.BaseMint.Decimals))
			quoteDecimalsPow := math.Pow(10, float64(pool.QuoteMint.Decimals))

			log.Infof("> 开始更新池子 %d 的余额", pool.ID)
			log.Infof("> 池子 %d 的 BaseMint: %s, QuoteMint: %s", pool.ID, baseMint, quoteMint)

			// 查询 BaseVault 余额
			baseVaultPubkey, err := solana.PublicKeyFromBase58(pool.BaseVault)
			if err != nil {
				log.Errorf("> 解析 BaseVault 地址失败: %s", pool.BaseVault)
				return
			}

			// 查询 BaseVault 余额
			baseBalResp, err := client.GetTokenAccountBalance(context.Background(), baseVaultPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", baseVaultPubkey, err)
				return
			}
			if baseBalResp == nil || baseBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", baseVaultPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", baseVaultPubkey, baseBalResp.Value.Amount)
			baseAmt, err := strconv.ParseUint(baseBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			// 查询 QuoteVault 余额
			quoteVaultPubkey, err := solana.PublicKeyFromBase58(pool.QuoteVault)
			if err != nil {
				log.Errorf("> 解析 QuoteVault 地址失败: %s", pool.QuoteVault)
				return
			}

			quoteBalResp, err := client.GetTokenAccountBalance(context.Background(), quoteVaultPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", quoteVaultPubkey, err)
				return
			}
			if quoteBalResp == nil || quoteBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", quoteVaultPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", quoteVaultPubkey, quoteBalResp.Value.Amount)
			quoteAmt, err := strconv.ParseUint(quoteBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			updateTime := time.Now()
			log.Infof("> 池子 %d 的 BaseMint 余额: %d, QuoteMint 余额: %d, 更新时间: %s", pool.ID, baseAmt, quoteAmt, updateTime)
			UpdatePoolStat(config.DB, pool.ID, baseAmt, quoteAmt, baseDecimalsPow, quoteDecimalsPow, baseIsWsol, updateTime)
		}(pool)
		time.Sleep(time.Duration(POOL_UPDATE_INTERVAL * float64(time.Second)))
	}
	poolWg.Wait()
}

func UpdatePumpfunInternalStats(client *rpc.Client) {
	var configs []models.PumpfuninternalConfig
	if err := config.DB.Where("status = ?", "active").Find(&configs).Error; err != nil {
		log.Fatalf("> 查询 Pumpfun Internal 配置失败: %v", err)
	}

	pumpfunSem := make(chan struct{}, PUMPFUN_MAX_CONCURRENT)
	var pumpfunWg sync.WaitGroup
	for _, cfg := range configs {
		pumpfunWg.Add(1)
		pumpfunSem <- struct{}{}
		go func(cfg models.PumpfuninternalConfig) {
			defer pumpfunWg.Done()
			defer func() { <-pumpfunSem }()

			log.Infof("> 开始更新 Pumpfun Internal 池子 %d 的状态", cfg.ID)

			// 转换地址
			mint, err := solana.PublicKeyFromBase58(cfg.Mint)
			if err != nil {
				log.Errorf("> 解析 Mint 地址失败: %s", cfg.Mint)
				return
			}

			feeRecipient, err := solana.PublicKeyFromBase58(cfg.FeeRecipient)
			if err != nil {
				log.Errorf("> 解析 FeeRecipient 地址失败: %s", cfg.FeeRecipient)
				return
			}

			// 获取池子状态
			stat, err := solanautil.GetPumpFunInternalPoolStat(client, mint, cfg.FeeRate, feeRecipient)
			if err != nil {
				log.Errorf("> 获取 Pumpfun Internal 池子状态失败: %v", err)
				return
			}

			// 获取池子代币余额
			associatedBondingCurve, err := solana.PublicKeyFromBase58(cfg.AssociatedBondingCurve)
			if err != nil {
				log.Errorf("> 解析 AssociatedBondingCurve 地址失败: %s", cfg.AssociatedBondingCurve)
				return
			}
			tokenBalResp, err := client.GetTokenAccountBalance(context.Background(), associatedBondingCurve, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 获取 Pumpfun Internal 池子代币余额失败: %v", err)
				return
			}
			if tokenBalResp == nil || tokenBalResp.Value == nil {
				log.Errorf("> 获取 Pumpfun Internal 池子代币余额返回空值")
				return
			}
			tokenAmt, err := strconv.ParseUint(tokenBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析代币余额失败: %v", err)
				return
			}
			tokenAmtFloat := float64(tokenAmt) / 1e6

			// 获取池子 sol 余额
			bondingCurvePda, err := solana.PublicKeyFromBase58(cfg.BondingCurvePda)
			if err != nil {
				log.Errorf("> 解析 BondingCurvePda 地址失败: %s", cfg.BondingCurvePda)
				return
			}
			solBalResp, err := client.GetBalance(context.Background(), bondingCurvePda, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 获取 Pumpfun Internal 池子 sol 余额失败: %v", err)
				return
			}
			solAmt := float64(solBalResp.Value) / 1e9 // Convert lamports to SOL

			updateTime := time.Now()
			log.Infof("> Pumpfun Internal 池子 %d 状态更新时间: %s", cfg.ID, updateTime)
			UpdatePumpfunInternalStat(config.DB, cfg.ID, stat, tokenAmtFloat, solAmt)
		}(cfg)
		time.Sleep(time.Duration(PUMPFUN_UPDATE_INTERVAL * float64(time.Second)))
	}
	pumpfunWg.Wait()
}

func UpdatePumpfunAmmPoolStats(client *rpc.Client) {
	var pools []models.PumpfunAmmPoolConfig
	if err := config.DB.Where("status = ?", "active").Find(&pools).Error; err != nil {
		log.Fatalf("> 查询 PumpfunAmm 池子失败: %v", err)
	}

	poolSem := make(chan struct{}, PUMPFUN_AMM_MAX_CONCURRENT)
	var poolWg sync.WaitGroup
	for _, pool := range pools {
		poolWg.Add(1)
		poolSem <- struct{}{}
		go func(pool models.PumpfunAmmPoolConfig) {
			defer poolWg.Done()
			defer func() { <-poolSem }()

			log.Infof("> 开始更新 PumpfunAmm 池子 %d 的余额", pool.ID)
			log.Infof("> 池子 %d 的 BaseMint: %s, QuoteMint: %s", pool.ID, pool.BaseMint, pool.QuoteMint)

			// 查询 BaseTokenAccount 余额
			baseTokenAccountPubkey, err := solana.PublicKeyFromBase58(pool.PoolBaseTokenAccount)
			if err != nil {
				log.Errorf("> 解析 PoolBaseTokenAccount 地址失败: %s", pool.PoolBaseTokenAccount)
				return
			}

			baseBalResp, err := client.GetTokenAccountBalance(context.Background(), baseTokenAccountPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", baseTokenAccountPubkey, err)
				return
			}
			if baseBalResp == nil || baseBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", baseTokenAccountPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", baseTokenAccountPubkey, baseBalResp.Value.Amount)
			baseAmt, err := strconv.ParseUint(baseBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			// 查询 QuoteTokenAccount 余额
			quoteTokenAccountPubkey, err := solana.PublicKeyFromBase58(pool.PoolQuoteTokenAccount)
			if err != nil {
				log.Errorf("> 解析 PoolQuoteTokenAccount 地址失败: %s", pool.PoolQuoteTokenAccount)
				return
			}

			quoteBalResp, err := client.GetTokenAccountBalance(context.Background(), quoteTokenAccountPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", quoteTokenAccountPubkey, err)
				return
			}
			if quoteBalResp == nil || quoteBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", quoteTokenAccountPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", quoteTokenAccountPubkey, quoteBalResp.Value.Amount)
			quoteAmt, err := strconv.ParseUint(quoteBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			// 获取当前时间作为更新时间
			timestamp := time.Now().UTC().Unix()
			updateTime := time.Unix(timestamp, 0)
			log.Infof("> 池子 %d 的 BaseMint 余额: %d, QuoteMint 余额: %d, 更新时间: %s", pool.ID, baseAmt, quoteAmt, updateTime)

			// 更新统计信息
			UpdatePumpfunAmmPoolStat(config.DB, pool.ID, baseAmt, quoteAmt, pool.LpSupply, updateTime)
		}(pool)
		time.Sleep(time.Duration(PUMPFUN_AMM_UPDATE_INTERVAL * float64(time.Second)))
	}
	poolWg.Wait()
}

func UpdateRaydiumLaunchpadPoolStats(client *rpc.Client) {
	var configs []models.RaydiumLaunchpadPoolConfig
	if err := config.DB.Where("status = ?", "active").Find(&configs).Error; err != nil {
		log.Fatalf("> 查询 Raydium Launchpad 配置失败: %v", err)
	}

	raydiumSem := make(chan struct{}, RAYDIUM_LAUNCHPAD_MAX_CONCURRENT)
	var raydiumWg sync.WaitGroup
	for _, cfg := range configs {
		raydiumWg.Add(1)
		raydiumSem <- struct{}{}
		go func(cfg models.RaydiumLaunchpadPoolConfig) {
			defer raydiumWg.Done()
			defer func() { <-raydiumSem }()

			log.Infof("> 开始更新 Raydium Launchpad 池子 %d 的状态", cfg.ID)

			// 转换池地址
			poolAddress, err := solana.PublicKeyFromBase58(cfg.PoolAddress)
			if err != nil {
				log.Errorf("> 解析 PoolAddress 地址失败: %s", cfg.PoolAddress)
				return
			}

			// 获取 RPC 端点
			solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
			if solanaRPC == "" {
				log.Errorf("> Solana RPC endpoint not configured")
				return
			}

			// 获取池子状态
			poolInfo, err := solanautil.GetLaunchpadPoolInfo(solanaRPC, poolAddress)
			if err != nil {
				log.Errorf("> 获取 Raydium Launchpad 池子状态失败: %v", err)
				return
			}

			// 获取池子代币余额
			baseVaultPubkey, err := solana.PublicKeyFromBase58(cfg.BaseVault)
			if err != nil {
				log.Errorf("> 解析 BaseVault 地址失败: %s", cfg.BaseVault)
				return
			}
			baseBalResp, err := client.GetTokenAccountBalance(context.Background(), baseVaultPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 获取 Raydium Launchpad 池子基础代币余额失败: %v", err)
				return
			}
			if baseBalResp == nil || baseBalResp.Value == nil {
				log.Errorf("> 获取 Raydium Launchpad 池子基础代币余额返回空值")
				return
			}
			baseTokenAmt, err := strconv.ParseUint(baseBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析基础代币余额失败: %v", err)
				return
			}
			baseTokenAmtFloat := float64(baseTokenAmt) / 1e6 // 6 decimals

			// 获取池子引用代币余额
			quoteVaultPubkey, err := solana.PublicKeyFromBase58(cfg.QuoteVault)
			if err != nil {
				log.Errorf("> 解析 QuoteVault 地址失败: %s", cfg.QuoteVault)
				return
			}
			quoteBalResp, err := client.GetTokenAccountBalance(context.Background(), quoteVaultPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 获取 Raydium Launchpad 池子引用代币余额失败: %v", err)
				return
			}
			if quoteBalResp == nil || quoteBalResp.Value == nil {
				log.Errorf("> 获取 Raydium Launchpad 池子引用代币余额返回空值")
				return
			}
			quoteTokenAmt, err := strconv.ParseUint(quoteBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析引用代币余额失败: %v", err)
				return
			}
			quoteTokenAmtFloat := float64(quoteTokenAmt) / 1e9 // 9 decimals for SOL

			updateTime := time.Now()
			log.Infof("> Raydium Launchpad 池子 %d 状态更新时间: %s", cfg.ID, updateTime)
			UpdateRaydiumLaunchpadPoolStat(config.DB, cfg.PoolAddress, poolInfo, baseTokenAmtFloat, quoteTokenAmtFloat)
		}(cfg)
		time.Sleep(time.Duration(RAYDIUM_LAUNCHPAD_UPDATE_INTERVAL * float64(time.Second)))
	}
	raydiumWg.Wait()
}

func UpdateRaydiumCpmmPoolStats(client *rpc.Client) {
	var pools []models.RaydiumCpmmPoolConfig
	if err := config.DB.Where("status = ?", "active").Find(&pools).Error; err != nil {
		log.Fatalf("> 查询 Raydium CPMM 池子失败: %v", err)
	}

	poolSem := make(chan struct{}, RAYDIUM_CPMM_MAX_CONCURRENT)
	var poolWg sync.WaitGroup
	for _, pool := range pools {
		poolWg.Add(1)
		poolSem <- struct{}{}
		go func(pool models.RaydiumCpmmPoolConfig) {
			defer poolWg.Done()
			defer func() { <-poolSem }()

			log.Infof("> 开始更新 Raydium CPMM 池子 %d 的余额", pool.ID)
			log.Infof("> 池子 %d 的 BaseMint: %s, QuoteMint: %s", pool.ID, pool.BaseMint, pool.QuoteMint)

			// 查询 BaseVault 余额
			baseVaultPubkey, err := solana.PublicKeyFromBase58(pool.BaseVault)
			if err != nil {
				log.Errorf("> 解析 BaseVault 地址失败: %s", pool.BaseVault)
				return
			}

			baseBalResp, err := client.GetTokenAccountBalance(context.Background(), baseVaultPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", baseVaultPubkey, err)
				return
			}
			if baseBalResp == nil || baseBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", baseVaultPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", baseVaultPubkey, baseBalResp.Value.Amount)
			baseAmt, err := strconv.ParseUint(baseBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			// 查询 QuoteVault 余额
			quoteVaultPubkey, err := solana.PublicKeyFromBase58(pool.QuoteVault)
			if err != nil {
				log.Errorf("> 解析 QuoteVault 地址失败: %s", pool.QuoteVault)
				return
			}

			quoteBalResp, err := client.GetTokenAccountBalance(context.Background(), quoteVaultPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", quoteVaultPubkey, err)
				return
			}
			if quoteBalResp == nil || quoteBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", quoteVaultPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", quoteVaultPubkey, quoteBalResp.Value.Amount)
			quoteAmt, err := strconv.ParseUint(quoteBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			// 获取当前时间作为更新时间
			timestamp := time.Now().UTC().Unix()
			updateTime := time.Unix(timestamp, 0)
			log.Infof("> 池子 %d 的 BaseMint 余额: %d, QuoteMint 余额: %d, 更新时间: %s", pool.ID, baseAmt, quoteAmt, updateTime)

			// 更新统计信息
			UpdateRaydiumCpmmPoolStat(config.DB, pool.ID, baseAmt, quoteAmt, updateTime)
		}(pool)
		time.Sleep(time.Duration(RAYDIUM_CPMM_UPDATE_INTERVAL * float64(time.Second)))
	}
	poolWg.Wait()
}

// UpdatePoolStat 更新或插入 PoolStat
func UpdatePoolStat(db *gorm.DB, poolID uint, baseAmt, quoteAmt uint64, baseDecimalsPow, quoteDecimalsPow float64, baseIsWsol bool, updateTime time.Time) {
	// 使用 UTC 时间
	updateTime = updateTime.UTC()

	var baseAmountReadable = float64(baseAmt) / baseDecimalsPow
	var quoteAmountReadable = float64(quoteAmt) / quoteDecimalsPow
	var price float64
	var marketValue float64
	if baseIsWsol {
		price = baseAmountReadable / quoteAmountReadable
		marketValue = baseAmountReadable * 2
	} else {
		price = quoteAmountReadable / baseAmountReadable
		marketValue = quoteAmountReadable * 2
	}

	var stat models.PoolStat
	if err := db.Where("pool_id = ?", poolID).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			stat = models.PoolStat{
				PoolID:              poolID,
				BaseAmount:          baseAmt,
				QuoteAmount:         quoteAmt,
				BaseAmountReadable:  baseAmountReadable,
				QuoteAmountReadable: quoteAmountReadable,
				MarketValue:         marketValue,
				LpSupply:            0,
				Price:               price,
				Slot:                0,
				BlockTime:           updateTime,
			}
			if err := db.Create(&stat).Error; err != nil {
				log.Errorf("> 创建 PoolStat 失败: %v", err)
			}
		} else {
			log.Errorf("> 查询 PoolStat 失败: %v", err)
		}
	} else {
		stat.BaseAmount = baseAmt
		stat.QuoteAmount = quoteAmt
		stat.BaseAmountReadable = baseAmountReadable
		stat.QuoteAmountReadable = quoteAmountReadable
		stat.MarketValue = marketValue
		stat.LpSupply = 0
		stat.Price = price
		stat.Slot = 0
		stat.BlockTime = updateTime
		if err := db.Save(&stat).Error; err != nil {
			log.Errorf("> 更新 PoolStat 失败: %v", err)
		}
	}
}

// UpdatePumpfunInternalStat 更新或插入 PumpfuninternalStat
func UpdatePumpfunInternalStat(db *gorm.DB, pumpfunInternalID uint, stat *solanautil.PumpFunInternalPoolStat, tokenAmt float64, solAmt float64) {
	timestamp := time.Now().UTC().Unix()
	updateTime := time.Unix(timestamp, 0)
	var pumpfunStat models.PumpfuninternalStat
	if err := db.Where("pumpfuninternal_id = ?", pumpfunInternalID).First(&pumpfunStat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			pumpfunStat = models.PumpfuninternalStat{
				PumpfuninternalID:    pumpfunInternalID,
				Mint:                 stat.Mint,
				UnknownData:          stat.UnknownData,
				VirtualTokenReserves: stat.VirtualTokenReserves,
				VirtualSolReserves:   stat.VirtualSolReserves,
				RealTokenReserves:    stat.RealTokenReserves,
				RealSolReserves:      stat.RealSolReserves,
				TokenTotalSupply:     stat.TokenTotalSupply,
				Complete:             stat.Complete,
				Creator:              stat.Creator,
				Price:                stat.Price,
				FeeRecipient:         stat.FeeRecipient,
				SolBalance:           solAmt,
				TokenBalance:         tokenAmt,
				BlockTime:            updateTime,
			}
			if err := db.Create(&pumpfunStat).Error; err != nil {
				log.Errorf("> 创建 PumpfuninternalStat 失败: %v", err)
			}
		} else {
			log.Errorf("> 查询 PumpfuninternalStat 失败: %v", err)
		}
	} else {
		pumpfunStat.Mint = stat.Mint
		pumpfunStat.UnknownData = stat.UnknownData
		pumpfunStat.VirtualTokenReserves = stat.VirtualTokenReserves
		pumpfunStat.VirtualSolReserves = stat.VirtualSolReserves
		pumpfunStat.RealTokenReserves = stat.RealTokenReserves
		pumpfunStat.RealSolReserves = stat.RealSolReserves
		pumpfunStat.TokenTotalSupply = stat.TokenTotalSupply
		pumpfunStat.Complete = stat.Complete
		pumpfunStat.Creator = stat.Creator
		pumpfunStat.Price = stat.Price
		pumpfunStat.FeeRecipient = stat.FeeRecipient
		pumpfunStat.SolBalance = solAmt
		pumpfunStat.TokenBalance = tokenAmt
		pumpfunStat.BlockTime = updateTime
		if err := db.Save(&pumpfunStat).Error; err != nil {
			log.Errorf("> 更新 PumpfuninternalStat 失败: %v", err)
		}
	}
}

// UpdatePumpfunAmmPoolStat 更新或插入 PumpfunAmmPoolStat
func UpdatePumpfunAmmPoolStat(db *gorm.DB, poolID uint, baseAmt, quoteAmt, lpSupply uint64, updateTime time.Time) {
	// 假设 base 和 quote 都是 6 位小数
	baseDecimalsPow := math.Pow(10, 6)
	quoteDecimalsPow := math.Pow(10, 9)

	baseAmountReadable := float64(baseAmt) / baseDecimalsPow
	quoteAmountReadable := float64(quoteAmt) / quoteDecimalsPow

	// 计算价格和市值
	price := quoteAmountReadable / baseAmountReadable
	marketValue := quoteAmountReadable * 2

	var stat models.PumpfunAmmPoolStat
	if err := db.Where("pool_id = ?", poolID).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			stat = models.PumpfunAmmPoolStat{
				PoolID:              poolID,
				BaseAmount:          baseAmt,
				QuoteAmount:         quoteAmt,
				BaseAmountReadable:  baseAmountReadable,
				QuoteAmountReadable: quoteAmountReadable,
				MarketValue:         marketValue,
				LpSupply:            lpSupply,
				Price:               price,
				Slot:                0,
				BlockTime:           updateTime,
			}
			if err := db.Create(&stat).Error; err != nil {
				log.Errorf("> 创建 PumpfunAmmPoolStat 失败: %v", err)
			}
		} else {
			log.Errorf("> 查询 PumpfunAmmPoolStat 失败: %v", err)
		}
	} else {
		stat.BaseAmount = baseAmt
		stat.QuoteAmount = quoteAmt
		stat.BaseAmountReadable = baseAmountReadable
		stat.QuoteAmountReadable = quoteAmountReadable
		stat.MarketValue = marketValue
		stat.LpSupply = lpSupply
		stat.Price = price
		stat.Slot = 0
		stat.BlockTime = updateTime
		if err := db.Save(&stat).Error; err != nil {
			log.Errorf("> 更新 PumpfunAmmPoolStat 失败: %v", err)
		}
	}
}

// UpdateRaydiumLaunchpadPoolStat 更新或插入 RaydiumLaunchpadPoolStat
func UpdateRaydiumLaunchpadPoolStat(db *gorm.DB, poolAddress string, poolInfo *solanautil.LaunchpadPoolInfo, baseBalance float64, quoteBalance float64) {
	timestamp := time.Now().UTC().Unix()
	updateTime := time.Unix(timestamp, 0)

	// 解析字符串数据为数字
	epoch, _ := strconv.ParseUint(poolInfo.Epoch, 10, 64)
	supply, _ := strconv.ParseFloat(poolInfo.Supply, 64)
	totalSellA, _ := strconv.ParseFloat(poolInfo.TotalSellA, 64)
	virtualA, _ := strconv.ParseFloat(poolInfo.VirtualA, 64)
	virtualB, _ := strconv.ParseFloat(poolInfo.VirtualB, 64)
	realA, _ := strconv.ParseFloat(poolInfo.RealA, 64)
	realB, _ := strconv.ParseFloat(poolInfo.RealB, 64)
	totalFundRaisingB, _ := strconv.ParseFloat(poolInfo.TotalFundRaisingB, 64)
	protocolFee, _ := strconv.ParseFloat(poolInfo.ProtocolFee, 64)
	platformFee, _ := strconv.ParseFloat(poolInfo.PlatformFee, 64)
	migrateFee, _ := strconv.ParseFloat(poolInfo.MigrateFee, 64)

	// 解析 VestingSchedule 数据
	vestingTotalLockedAmount, _ := strconv.ParseFloat(poolInfo.VestingSchedule.TotalLockedAmount, 64)
	vestingCliffPeriod, _ := strconv.ParseFloat(poolInfo.VestingSchedule.CliffPeriod, 64)
	vestingUnlockPeriod, _ := strconv.ParseFloat(poolInfo.VestingSchedule.UnlockPeriod, 64)
	vestingStartTime, _ := strconv.ParseFloat(poolInfo.VestingSchedule.StartTime, 64)
	vestingTotalAllocatedShare, _ := strconv.ParseFloat(poolInfo.VestingSchedule.TotalAllocatedShare, 64)
	mintProgramFlag := float64(poolInfo.MintProgramFlag)

	var raydiumStat models.RaydiumLaunchpadPoolStat
	if err := db.Where("pool_address = ?", poolAddress).First(&raydiumStat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			raydiumStat = models.RaydiumLaunchpadPoolStat{
				PoolAddress:                        poolAddress,
				Epoch:                              epoch,
				Bump:                               uint64(poolInfo.Bump),
				PoolStatus:                         uint64(poolInfo.Status),
				Mint:                               poolInfo.MintA, // 使用 MintA 作为主要代币
				MigrateType:                        uint64(poolInfo.MigrateType),
				Supply:                             supply,
				TotalSellA:                         totalSellA,
				VirtualA:                           virtualA,
				VirtualB:                           virtualB,
				RealA:                              realA,
				RealB:                              realB,
				TotalFundRaisingB:                  totalFundRaisingB,
				ProtocolFee:                        protocolFee,
				PlatformFee:                        platformFee,
				MigrateFee:                         migrateFee,
				VestingScheduleTotalLockedAmount:   vestingTotalLockedAmount,
				VestingScheduleCliffPeriod:         vestingCliffPeriod,
				VestingScheduleUnlockPeriod:        vestingUnlockPeriod,
				VestingScheduleStartTime:           vestingStartTime,
				VestingScheduleTotalAllocatedShare: vestingTotalAllocatedShare,
				MintProgramFlag:                    mintProgramFlag,
				BaseBalance:                        baseBalance,
				QuoteBalance:                       quoteBalance,
				Slot:                               0, // 可以根据需要设置
				BlockTime:                          updateTime,
			}
			if err := db.Create(&raydiumStat).Error; err != nil {
				log.Errorf("> 创建 RaydiumLaunchpadPoolStat 失败: %v", err)
			}
		} else {
			log.Errorf("> 查询 RaydiumLaunchpadPoolStat 失败: %v", err)
		}
	} else {
		// 更新现有记录
		raydiumStat.Epoch = epoch
		raydiumStat.Bump = uint64(poolInfo.Bump)
		raydiumStat.PoolStatus = uint64(poolInfo.Status)
		raydiumStat.Mint = poolInfo.MintA
		raydiumStat.MigrateType = uint64(poolInfo.MigrateType)
		raydiumStat.Supply = supply
		raydiumStat.TotalSellA = totalSellA
		raydiumStat.VirtualA = virtualA
		raydiumStat.VirtualB = virtualB
		raydiumStat.RealA = realA
		raydiumStat.RealB = realB
		raydiumStat.TotalFundRaisingB = totalFundRaisingB
		raydiumStat.ProtocolFee = protocolFee
		raydiumStat.PlatformFee = platformFee
		raydiumStat.MigrateFee = migrateFee
		raydiumStat.VestingScheduleTotalLockedAmount = vestingTotalLockedAmount
		raydiumStat.VestingScheduleCliffPeriod = vestingCliffPeriod
		raydiumStat.VestingScheduleUnlockPeriod = vestingUnlockPeriod
		raydiumStat.VestingScheduleStartTime = vestingStartTime
		raydiumStat.VestingScheduleTotalAllocatedShare = vestingTotalAllocatedShare
		raydiumStat.MintProgramFlag = mintProgramFlag
		raydiumStat.BaseBalance = baseBalance
		raydiumStat.QuoteBalance = quoteBalance
		raydiumStat.BlockTime = updateTime
		if err := db.Save(&raydiumStat).Error; err != nil {
			log.Errorf("> 更新 RaydiumLaunchpadPoolStat 失败: %v", err)
		}
	}
}

// UpdateRaydiumCpmmPoolStat 更新或插入 RaydiumCpmmPoolStat
func UpdateRaydiumCpmmPoolStat(db *gorm.DB, poolID uint, baseAmt, quoteAmt uint64, updateTime time.Time) {
	// 假设 base 和 quote 都是 6 位小数（可以根据实际情况调整）
	baseDecimalsPow := math.Pow(10, 6)
	quoteDecimalsPow := math.Pow(10, 9)

	baseAmountReadable := float64(baseAmt) / baseDecimalsPow
	quoteAmountReadable := float64(quoteAmt) / quoteDecimalsPow

	// 计算价格和市值
	price := quoteAmountReadable / baseAmountReadable
	marketValue := quoteAmountReadable * 2

	var stat models.RaydiumCpmmPoolStat
	if err := db.Where("pool_id = ?", poolID).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			stat = models.RaydiumCpmmPoolStat{
				PoolID:              poolID,
				BaseAmount:          baseAmt,
				QuoteAmount:         quoteAmt,
				BaseAmountReadable:  baseAmountReadable,
				QuoteAmountReadable: quoteAmountReadable,
				MarketValue:         marketValue,
				LpSupply:            0, // 默认值，可以根据需要设置
				BurnPercent:         0, // 默认值，可以根据需要设置
				Price:               price,
				Slot:                0,
				BlockTime:           updateTime,
			}
			if err := db.Create(&stat).Error; err != nil {
				log.Errorf("> 创建 RaydiumCpmmPoolStat 失败: %v", err)
			}
		} else {
			log.Errorf("> 查询 RaydiumCpmmPoolStat 失败: %v", err)
		}
	} else {
		stat.BaseAmount = baseAmt
		stat.QuoteAmount = quoteAmt
		stat.BaseAmountReadable = baseAmountReadable
		stat.QuoteAmountReadable = quoteAmountReadable
		stat.MarketValue = marketValue
		stat.Price = price
		stat.Slot = 0
		stat.BlockTime = updateTime
		if err := db.Save(&stat).Error; err != nil {
			log.Errorf("> 更新 RaydiumCpmmPoolStat 失败: %v", err)
		}
	}
}

func UpdateMeteoradbcPoolStats(client *rpc.Client) {
	var pools []models.MeteoradbcConfig
	if err := config.DB.Where("status = ?", "active").Find(&pools).Error; err != nil {
		log.Fatalf("> 查询 Meteoradbc 池子失败: %v", err)
	}

	poolSem := make(chan struct{}, METEORADBC_MAX_CONCURRENT)
	var poolWg sync.WaitGroup
	for _, pool := range pools {
		poolWg.Add(1)
		poolSem <- struct{}{}
		go func(pool models.MeteoradbcConfig) {
			defer poolWg.Done()
			defer func() { <-poolSem }()

			log.Infof("> 开始更新 Meteoradbc 池子 %d 的余额", pool.ID)
			log.Infof("> 池子 %d 的 BaseMint: %s, QuoteMint: %s", pool.ID, pool.BaseMint, pool.QuoteMint)

			// 查询 BaseTokenAccount 余额
			baseTokenAccountPubkey, err := solana.PublicKeyFromBase58(pool.PoolBaseTokenAccount)
			if err != nil {
				log.Errorf("> 解析 PoolBaseTokenAccount 地址失败: %s", pool.PoolBaseTokenAccount)
				return
			}

			baseBalResp, err := client.GetTokenAccountBalance(context.Background(), baseTokenAccountPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", baseTokenAccountPubkey, err)
				return
			}
			if baseBalResp == nil || baseBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", baseTokenAccountPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", baseTokenAccountPubkey, baseBalResp.Value.Amount)
			baseAmt, err := strconv.ParseUint(baseBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			// 查询 QuoteTokenAccount 余额
			quoteTokenAccountPubkey, err := solana.PublicKeyFromBase58(pool.PoolQuoteTokenAccount)
			if err != nil {
				log.Errorf("> 解析 PoolQuoteTokenAccount 地址失败: %s", pool.PoolQuoteTokenAccount)
				return
			}

			quoteBalResp, err := client.GetTokenAccountBalance(context.Background(), quoteTokenAccountPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询 account %s 的余额失败: %v", quoteTokenAccountPubkey, err)
				return
			}
			if quoteBalResp == nil || quoteBalResp.Value == nil {
				log.Errorf("> 查询 account %s 的余额返回空值", quoteTokenAccountPubkey)
				return
			}
			log.Infof("> 查询 account %s 的余额成功: %s", quoteTokenAccountPubkey, quoteBalResp.Value.Amount)
			quoteAmt, err := strconv.ParseUint(quoteBalResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析余额失败: %v", err)
				return
			}

			// 获取当前时间作为更新时间
			timestamp := time.Now().UTC().Unix()
			updateTime := time.Unix(timestamp, 0)
			log.Infof("> 池子 %d 的 BaseMint 余额: %d, QuoteMint 余额: %d, 更新时间: %s", pool.ID, baseAmt, quoteAmt, updateTime)

			// 使用 utils.GetTokenPrice 计算价格，并更新统计信息
			price := 1.0 // 默认价格
			if tokenPrice, _, err := utils.GetTokenPrice(pool.BaseMint); err != nil {
				log.Errorf("> 获取代币价格失败 base=%s: %v", pool.BaseMint, err)
			} else {
				price = tokenPrice
			}
			log.Infof("> 池子 %d 计算得到价格: %f", pool.ID, price)
			UpdateMeteoradbcPoolStat(config.DB, pool.PoolAddress, baseAmt, quoteAmt, price, updateTime)

			// 如果未迁移，则尝试根据 DammV2PoolAddress 同步更新对应的 CPMM 池统计
			if !pool.IsMigrated && pool.DammV2PoolAddress != "" {
				var cpmm models.MeteoracpmmConfig
				if err := config.DB.Where("pool_address = ?", pool.DammV2PoolAddress).First(&cpmm).Error; err != nil {
					if err == gorm.ErrRecordNotFound {
						log.Errorf("> 未找到对应的 MeteoracpmmConfig，pool_address=%s", pool.DammV2PoolAddress)
					} else {
						log.Errorf("> 查询 MeteoracpmmConfig 失败: %v", err)
					}
				} else {
					migrated := UpdateMeteoracpmmPoolStatFlow(client, cpmm)
					if migrated {
						if err := config.DB.Model(&models.ProjectConfig{}).
							Where("pool_platform = ? AND pool_id = ?", "meteora_dbc", pool.ID).
							Update("is_migrated", true).Error; err != nil {
							log.Errorf("> 更新 ProjectConfig 迁移状态失败: %v", err)
						} else {
							log.Infof("> 已更新 ProjectConfig 迁移状态为 true [pool_platform=meteora_dbc, pool_id=%d]", pool.ID)
						}
					}
				}
			}
		}(pool)
		time.Sleep(time.Duration(METEORADBC_UPDATE_INTERVAL * float64(time.Second)))
	}
	poolWg.Wait()
}

// UpdateMeteoradbcPoolStat 更新或插入 MeteoradbcPoolStat
func UpdateMeteoradbcPoolStat(db *gorm.DB, poolAddress string, baseAmt, quoteAmt uint64, price float64, updateTime time.Time) {
	// 假设 base 和 quote 都是 6 位小数
	baseDecimalsPow := math.Pow(10, 6)
	quoteDecimalsPow := math.Pow(10, 9)

	baseAmountReadable := float64(baseAmt) / baseDecimalsPow
	quoteAmountReadable := float64(quoteAmt) / quoteDecimalsPow

	// 计算市值（遵循原有逻辑）
	marketValue := quoteAmountReadable * 2

	var stat models.MeteoradbcPoolStat
	if err := db.Where("pool_address = ?", poolAddress).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			stat = models.MeteoradbcPoolStat{
				PoolAddress:         poolAddress,
				BaseAmount:          baseAmt,
				QuoteAmount:         quoteAmt,
				BaseAmountReadable:  baseAmountReadable,
				QuoteAmountReadable: quoteAmountReadable,
				MarketValue:         marketValue,
				Price:               price,
				Slot:                0,
				BlockTime:           updateTime,
			}
			if err := db.Create(&stat).Error; err != nil {
				log.Errorf("> 创建 MeteoradbcPoolStat 失败: %v", err)
			}
		} else {
			log.Errorf("> 查询 MeteoradbcPoolStat 失败: %v", err)
		}
	} else {
		stat.BaseAmount = baseAmt
		stat.QuoteAmount = quoteAmt
		stat.BaseAmountReadable = baseAmountReadable
		stat.QuoteAmountReadable = quoteAmountReadable
		stat.MarketValue = marketValue
		stat.Price = price
		stat.Slot = 0
		stat.BlockTime = updateTime
		if err := db.Save(&stat).Error; err != nil {
			log.Errorf("> 更新 MeteoradbcPoolStat 失败: %v", err)
		}
	}
}

func UpdateMeteoracpmmPoolStats(client *rpc.Client) {
	var pools []models.MeteoracpmmConfig
	if err := config.DB.Where("status = ?", "active").Find(&pools).Error; err != nil {
		log.Fatalf("> 查询 Meteoracpmm 池子失败: %v", err)
	}

	poolSem := make(chan struct{}, METEORACPMM_MAX_CONCURRENT)
	var poolWg sync.WaitGroup
	for _, pool := range pools {
		poolWg.Add(1)
		poolSem <- struct{}{}
		go func(pool models.MeteoracpmmConfig) {
			defer poolWg.Done()
			defer func() { <-poolSem }()

			UpdateMeteoracpmmPoolStatFlow(client, pool)
		}(pool)
		time.Sleep(time.Duration(METEORACPMM_UPDATE_INTERVAL * float64(time.Second)))
	}
	poolWg.Wait()
}

// UpdateMeteoracpmmPoolStatFlow 查询 Meteoracpmm 池子的代币余额并更新统计信息
func UpdateMeteoracpmmPoolStatFlow(client *rpc.Client, pool models.MeteoracpmmConfig) bool {
	log.Infof("> 开始更新 Meteoracpmm 池子 %d 的余额", pool.ID)
	log.Infof("> 池子 %d 的 BaseMint: %s, QuoteMint: %s", pool.ID, pool.BaseMint, pool.QuoteMint)

	// 查询 BaseTokenAccount 余额
	baseTokenAccountPubkey, err := solana.PublicKeyFromBase58(pool.PoolBaseTokenAccount)
	if err != nil {
		log.Errorf("> 解析 PoolBaseTokenAccount 地址失败: %s", pool.PoolBaseTokenAccount)
		return false
	}

	baseBalResp, err := client.GetTokenAccountBalance(context.Background(), baseTokenAccountPubkey, rpc.CommitmentFinalized)
	if err != nil {
		log.Errorf("> 查询 account %s 的余额失败: %v", baseTokenAccountPubkey, err)
		return false
	}
	if baseBalResp == nil || baseBalResp.Value == nil {
		log.Errorf("> 查询 account %s 的余额返回空值", baseTokenAccountPubkey)
		return false
	}
	log.Infof("> 查询 account %s 的余额成功: %s", baseTokenAccountPubkey, baseBalResp.Value.Amount)
	baseAmt, err := strconv.ParseUint(baseBalResp.Value.Amount, 10, 64)
	if err != nil {
		log.Errorf("> 解析余额失败: %v", err)
		return false
	}

	// 查询 QuoteTokenAccount 余额
	quoteTokenAccountPubkey, err := solana.PublicKeyFromBase58(pool.PoolQuoteTokenAccount)
	if err != nil {
		log.Errorf("> 解析 PoolQuoteTokenAccount 地址失败: %s", pool.PoolQuoteTokenAccount)
		return false
	}

	quoteBalResp, err := client.GetTokenAccountBalance(context.Background(), quoteTokenAccountPubkey, rpc.CommitmentFinalized)
	if err != nil {
		log.Errorf("> 查询 account %s 的余额失败: %v", quoteTokenAccountPubkey, err)
		return false
	}
	if quoteBalResp == nil || quoteBalResp.Value == nil {
		log.Errorf("> 查询 account %s 的余额返回空值", quoteTokenAccountPubkey)
		return false
	}
	log.Infof("> 查询 account %s 的余额成功: %s", quoteTokenAccountPubkey, quoteBalResp.Value.Amount)
	quoteAmt, err := strconv.ParseUint(quoteBalResp.Value.Amount, 10, 64)
	if err != nil {
		log.Errorf("> 解析余额失败: %v", err)
		return false
	}

	// 迁移状态：任一余额大于 0 视为已迁移
	isMigrated := (baseAmt > 0) || (quoteAmt > 0)

	// 获取当前时间作为更新时间
	timestamp := time.Now().UTC().Unix()
	updateTime := time.Unix(timestamp, 0)
	log.Infof("> 池子 %d 的 BaseMint 余额: %d, QuoteMint 余额: %d, 更新时间: %s", pool.ID, baseAmt, quoteAmt, updateTime)

	// 使用 utils.GetTokenPrice 计算价格，并更新统计信息
	price := 1.0 // 默认价格
	if tokenPrice, _, err := utils.GetTokenPrice(pool.BaseMint); err != nil {
		log.Errorf("> 获取代币价格失败 base=%s: %v", pool.BaseMint, err)
	} else {
		price = tokenPrice
	}
	log.Infof("> 池子 %d 计算得到价格: %f", pool.ID, price)
	UpdateMeteoracpmmPoolStat(config.DB, pool.PoolAddress, baseAmt, quoteAmt, price, updateTime)
	return isMigrated
}

// UpdateMeteoracpmmPoolStat 更新或插入 MeteoracpmmPoolStat
func UpdateMeteoracpmmPoolStat(db *gorm.DB, poolAddress string, baseAmt, quoteAmt uint64, price float64, updateTime time.Time) {
	// 假设 base 和 quote 都是 6 位小数
	baseDecimalsPow := math.Pow(10, 6)
	quoteDecimalsPow := math.Pow(10, 9)

	baseAmountReadable := float64(baseAmt) / baseDecimalsPow
	quoteAmountReadable := float64(quoteAmt) / quoteDecimalsPow

	// 计算市值（遵循原有逻辑）
	marketValue := quoteAmountReadable * 2

	var stat models.MeteoracpmmPoolStat
	if err := db.Where("pool_address = ?", poolAddress).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			stat = models.MeteoracpmmPoolStat{
				PoolAddress:         poolAddress,
				BaseAmount:          baseAmt,
				QuoteAmount:         quoteAmt,
				BaseAmountReadable:  baseAmountReadable,
				QuoteAmountReadable: quoteAmountReadable,
				MarketValue:         marketValue,
				Price:               price,
				Slot:                0,
				BlockTime:           updateTime,
			}
			if err := db.Create(&stat).Error; err != nil {
				log.Errorf("> 创建 MeteoracpmmPoolStat 失败: %v", err)
			}
		} else {
			log.Errorf("> 查询 MeteoracpmmPoolStat 失败: %v", err)
		}
	} else {
		stat.BaseAmount = baseAmt
		stat.QuoteAmount = quoteAmt
		stat.BaseAmountReadable = baseAmountReadable
		stat.QuoteAmountReadable = quoteAmountReadable
		stat.MarketValue = marketValue
		stat.Price = price
		stat.Slot = 0
		stat.BlockTime = updateTime
		if err := db.Save(&stat).Error; err != nil {
			log.Errorf("> 更新 MeteoracpmmPoolStat 失败: %v", err)
		}
	}
}
