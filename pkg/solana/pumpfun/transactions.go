package pumpfun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	"marketcontrol/pkg/helius"
	pumpsolana "marketcontrol/pkg/solana"

	"github.com/gagliardetto/solana-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	maxWorkers      = 3     // 最大并发工作协程数
	apiRequestDelay = 20000 // API 请求间隔（毫秒）
	MAX_PAGE        = 2     // 最大翻页次数
)

var (
	updateMutex  sync.Mutex // 添加互斥锁
	addressLocks sync.Map   // 地址级别的锁映射
)

// UpdateWalletTokenStat 更新钱包代币统计信息
func UpdateWalletTokenStat(db *gorm.DB, address, mint string, amountChange float64, decimals uint) error {
	var walletStat models.WalletTokenStat

	logrus.Infof("UpdateWalletTokenStat: address: %s, mint: %s, amountChange: %f, decimals: %d", address, mint, amountChange, decimals)
	// 查找现有的 WalletTokenStat 记录
	result := db.Where("owner_address = ? AND mint = ?", address, mint).First(&walletStat)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 没有找到记录，创建新记录
			balance := uint64(0)
			balanceReadable := 0.0

			if amountChange > 0 {
				balance = uint64(amountChange * math.Pow10(int(decimals)))
				balanceReadable = amountChange
			}

			walletStat = models.WalletTokenStat{
				OwnerAddress:    address,
				Mint:            mint,
				Decimals:        decimals,
				Balance:         balance,
				BalanceReadable: balanceReadable,
				Slot:            0, // 默认值，可以根据需要设置
				BlockTime:       time.Now(),
			}

			if err := db.Create(&walletStat).Error; err != nil {
				return fmt.Errorf("error creating wallet token stat: %v", err)
			}
		} else {
			return fmt.Errorf("error finding wallet token stat: %v", result.Error)
		}
	} else {
		// 已有 WalletTokenStat 记录，更新现有记录
		newBalanceReadable := walletStat.BalanceReadable + amountChange

		if newBalanceReadable >= 0 {
			walletStat.BalanceReadable = newBalanceReadable
			walletStat.Balance = uint64(newBalanceReadable * math.Pow10(int(decimals)))
		}

		// 更新时间戳
		walletStat.UpdatedAt = time.Now()

		if err := db.Save(&walletStat).Error; err != nil {
			return fmt.Errorf("error updating wallet token stat: %v", err)
		}
	}

	return nil
}

// GetOrCreateTransactionsMonitorConfig 获取或创建监控配置
func GetOrCreateTransactionsMonitorConfig(db *gorm.DB, address string) (*models.TransactionsMonitorConfig, error) {
	var config models.TransactionsMonitorConfig

	// 查找现有配置
	result := db.Where("address = ?", address).First(&config)
	if result.Error == nil {
		return &config, nil
	}

	// 明确检查是否是"记录未找到"的错误
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error querying config: %v", result.Error)
	}

	// 如果确实是记录不存在，创建新配置
	config = models.TransactionsMonitorConfig{
		Address:        address,
		Enabled:        true,
		LastSlot:       0,
		StartSlot:      0,
		LastTimestamp:  0,
		StartTimestamp: 0,
		LastSignature:  "",
		StartSignature: "",
		LastExecution:  0,
		Retry:          false,
	}

	// 使用事务来确保创建操作的原子性
	err := db.Transaction(func(tx *gorm.DB) error {
		// 再次检查是否存在，避免并发情况下的重复创建
		var count int64
		if err := tx.Model(&models.TransactionsMonitorConfig{}).
			Where("address = ?", address).Count(&count).Error; err != nil {
			return fmt.Errorf("error checking existing config in transaction: %v", err)
		}
		if count > 0 {
			return fmt.Errorf("config for address %s already exists", address)
		}

		// 确认不存在后再创建
		if err := tx.Create(&config).Error; err != nil {
			return fmt.Errorf("error creating config: %v", err)
		}

		return nil
	})

	if err != nil {
		logrus.Errorf("Failed to create transactions monitor config: %v", err)
		return nil, err
	}

	logrus.Infof("Successfully created transactions monitor config for address: %s", address)
	return &config, nil
}

// CreateAddressBalanceChange 创建地址余额变化记录，返回内存中的变化数据而不写入数据库
func CreateAddressBalanceChange(tx helius.EnhancedTransaction, targetExchange string) []models.AddressBalanceChange {
	var changes []models.AddressBalanceChange

	// 获取基本交易信息
	slot := uint(tx.Slot)
	timestamp := uint(tx.Timestamp)
	signature := tx.Signature

	// 处理代币转账 - 只处理涉及目标交易所的转账
	for _, tokenTransfer := range tx.TokenTransfers {
		// 检查是否涉及目标交易所
		isTradeWithExchange := tokenTransfer.FromUserAccount == targetExchange || tokenTransfer.ToUserAccount == targetExchange
		if !isTradeWithExchange {
			// 与交易所有关
			continue
		}
		// 创建发送方余额变化记录
		fromChange := models.AddressBalanceChange{
			Slot:         slot,
			Timestamp:    timestamp,
			Signature:    signature,
			Address:      tokenTransfer.FromUserAccount,
			Mint:         tokenTransfer.Mint,
			AmountChange: tokenTransfer.TokenAmount * -1, // 负值表示转出
		}
		changes = append(changes, fromChange)

		// 创建接收方余额变化记录
		toChange := models.AddressBalanceChange{
			Slot:         slot,
			Timestamp:    timestamp,
			Signature:    signature,
			Address:      tokenTransfer.ToUserAccount,
			Mint:         tokenTransfer.Mint,
			AmountChange: tokenTransfer.TokenAmount, // 正值表示转入
		}
		changes = append(changes, toChange)
	}

	// 处理原生代币(SOL)余额变化
	for _, accountData := range tx.AccountData {
		if accountData.NativeBalanceChange != 0 {
			// 创建 SOL 余额变化记录
			nativeChange := models.AddressBalanceChange{
				Slot:         slot,
				Timestamp:    timestamp,
				Signature:    signature,
				Address:      accountData.Account,
				Mint:         "sol", // 使用 "sol" 表示原生代币
				AmountChange: float64(accountData.NativeBalanceChange),
			}
			changes = append(changes, nativeChange)
		}
	}

	return changes
}

// GetAggregateBalanceChanges 获取指定条件的余额变化总和
func GetAggregateBalanceChanges(changes []models.AddressBalanceChange, address, mint string) float64 {
	var totalChange float64

	for _, change := range changes {
		if change.Address == address && change.Mint == mint {
			totalChange += change.AmountChange
		}
	}

	return totalChange
}

// CreatePumpfuninternalSwap 创建 Pumpfun 内部交换记录，返回 swap 数据数组
func CreatePumpfuninternalSwap(db *gorm.DB, tx helius.EnhancedTransaction, cfg models.PumpfuninternalConfig, balanceChanges []models.AddressBalanceChange) ([]*models.PumpfuninternalSwap, error) {
	var swaps []*models.PumpfuninternalSwap

	// 检查是否已存在相同的 Signature 和 Address 记录（针对 tx.FeePayer）
	var existingSwap models.PumpfuninternalSwap
	result := db.Where("signature = ? AND address = ?", tx.Signature, tx.FeePayer).First(&existingSwap)
	if result.Error == nil {
		// 记录已存在，返回错误
		return nil, fmt.Errorf("swap record already exists for signature %s and address %s", tx.Signature, tx.FeePayer)
	}

	// 检查是否是"记录未找到"的错误，如果是则继续创建
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing swap record: %v", result.Error)
	}

	// 创建主要的交换记录（针对 tx.FeePayer），使用传入的余额变化数据
	mainSwap := models.PumpfuninternalSwap{
		Slot:                  uint(tx.Slot),
		Timestamp:             uint(tx.Timestamp),
		Signature:             tx.Signature,
		Address:               tx.FeePayer,
		Mint:                  cfg.Mint,
		BondingCurvePda:       cfg.BondingCurvePda,
		TraderMintChange:      GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, cfg.Mint),
		TraderSolChange:       GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, "sol"),
		PoolMintChange:        GetAggregateBalanceChanges(balanceChanges, cfg.BondingCurvePda, cfg.Mint),
		PoolSolChange:         GetAggregateBalanceChanges(balanceChanges, cfg.BondingCurvePda, "sol"),
		FeeRecipientSolChange: GetAggregateBalanceChanges(balanceChanges, cfg.FeeRecipient, "sol"),
		CreatorSolChange:      GetAggregateBalanceChanges(balanceChanges, cfg.CreatorVaultPda, "sol"),
	}

	// 保存主要交换记录到数据库
	if err := db.Create(&mainSwap).Error; err != nil {
		return nil, fmt.Errorf("error creating main swap record: %v", err)
	}
	swaps = append(swaps, &mainSwap)

	logrus.Printf("Created main swap record for signature %s with trader mint change: %f, trader sol change: %f",
		tx.Signature, mainSwap.TraderMintChange, mainSwap.TraderSolChange)

	// 更新主要交换记录的钱包代币统计 - mint
	if mainSwap.TraderMintChange != 0 {
		if err := UpdateWalletTokenStat(db, tx.FeePayer, cfg.Mint, mainSwap.TraderMintChange, 6); err != nil {
			logrus.Warnf("Failed to update wallet token stat for mint %s, address %s: %v", cfg.Mint, tx.FeePayer, err)
		}
	}

	// 更新主要交换记录的钱包代币统计 - sol
	if mainSwap.TraderSolChange != 0 {
		if err := UpdateWalletTokenStat(db, tx.FeePayer, "sol", mainSwap.TraderSolChange/math.Pow10(9), 9); err != nil {
			logrus.Warnf("Failed to update wallet token stat for sol, address %s: %v", tx.FeePayer, err)
		}
	}

	// 迭代 balanceChanges，为每个地址（除了 tx.FeePayer）创建额外的交换记录
	processedAddresses := make(map[string]bool)
	processedAddresses[tx.FeePayer] = true         // 标记 tx.FeePayer 已处理
	processedAddresses[cfg.BondingCurvePda] = true // 标记池子已处理

	for _, change := range balanceChanges {
		// 跳过 tx.FeePayer 和已处理的地址
		if change.Mint != cfg.Mint {
			continue
		}
		if processedAddresses[change.Address] {
			continue
		}
		// 如果 change.Address 为空字符串，则跳过
		if change.Address == "" {
			continue
		}
		processedAddresses[change.Address] = true

		// 检查是否已存在相同的 Signature 和 Address 记录
		var existingAdditionalSwap models.PumpfuninternalSwap
		result := db.Where("signature = ? AND address = ?", tx.Signature, change.Address).First(&existingAdditionalSwap)
		if result.Error == nil {
			// 记录已存在，跳过
			logrus.Warnf("Additional swap record already exists for signature %s and address %s, skipping", tx.Signature, change.Address)
			continue
		}

		// 检查是否是"记录未找到"的错误，如果是则继续创建
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logrus.Errorf("error checking existing additional swap record: %v", result.Error)
			continue
		}

		// 创建额外的交换记录
		additionalSwap := models.PumpfuninternalSwap{
			Slot:                  uint(tx.Slot),
			Timestamp:             uint(tx.Timestamp),
			Signature:             tx.Signature,
			Address:               change.Address,
			Mint:                  cfg.Mint,
			BondingCurvePda:       cfg.BondingCurvePda,
			TraderMintChange:      GetAggregateBalanceChanges(balanceChanges, change.Address, cfg.Mint),
			TraderSolChange:       GetAggregateBalanceChanges(balanceChanges, change.Address, "sol"),
			PoolMintChange:        0,
			PoolSolChange:         0,
			FeeRecipientSolChange: 0,
			CreatorSolChange:      0,
		}

		// 保存额外的交换记录到数据库
		if err := db.Create(&additionalSwap).Error; err != nil {
			logrus.Errorf("error creating additional swap record for address %s: %v", change.Address, err)
			continue
		}
		swaps = append(swaps, &additionalSwap)

		logrus.Printf("Created additional swap record for signature %s, address %s with mint change: %f, sol change: %f",
			tx.Signature, change.Address, additionalSwap.TraderMintChange, additionalSwap.TraderSolChange)

		// 更新钱包代币统计 - mint
		if additionalSwap.TraderMintChange != 0 {
			if err := UpdateWalletTokenStat(db, change.Address, cfg.Mint, additionalSwap.TraderMintChange, 6); err != nil {
				logrus.Warnf("Failed to update wallet token stat for mint %s, address %s: %v", cfg.Mint, change.Address, err)
			}
		}

		// 更新钱包代币统计 - sol
		if additionalSwap.TraderSolChange != 0 {
			if err := UpdateWalletTokenStat(db, change.Address, "sol", additionalSwap.TraderSolChange/math.Pow10(9), 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for sol, address %s: %v", change.Address, err)
			}
		}
	}

	return swaps, nil
}

// UpdatePumpfunAmmPoolHolder updates holder information for AMM pool (T+1 logic with individual swap)
func UpdatePumpfunAmmPoolHolder(db *gorm.DB, swap *models.PumpfunAmmPoolSwap, cfg models.PumpfunAmmPoolConfig) error {
	// 确定 HolderType
	holderType := "retail_investors" // 默认值

	// 检查是否为项目地址 - AddressManage 表
	var addressManage models.AddressManage
	if err := db.Where("address = ?", swap.Address).First(&addressManage).Error; err == nil {
		holderType = "project"
	} else {
		// 检查是否为项目地址 - ProjectExtraAddress 表
		var projectExtraAddress models.ProjectExtraAddress
		if err := db.Where("address = ?", swap.Address).First(&projectExtraAddress).Error; err == nil {
			holderType = "project"
		} else if swap.Address == swap.PoolAddress {
			// 检查是否为池地址
			holderType = "pool"
		}
	}

	// 更新交易者的持有者记录
	var traderHolder models.PumpfunAmmpoolHolder
	result := db.Where("address = ? AND pool_address = ? AND base_mint = ? AND quote_mint = ?",
		swap.Address, swap.PoolAddress, cfg.BaseMint, cfg.QuoteMint).First(&traderHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new record
			traderHolder = models.PumpfunAmmpoolHolder{
				Address:           swap.Address,
				HolderType:        holderType,
				PoolAddress:       swap.PoolAddress,
				BaseMint:          cfg.BaseMint,
				QuoteMint:         cfg.QuoteMint,
				LastSlot:          swap.Slot,
				StartSlot:         swap.Slot,
				LastTimestamp:     swap.Timestamp,
				StartTimestamp:    swap.Timestamp,
				EndSignature:      swap.Signature,
				StartSignature:    swap.Signature,
				BaseChange:        swap.TraderBaseChange,
				QuoteChange:       swap.TraderQuoteChange,
				SolChange:         swap.TraderSolChange,
				TraderBaseVolume:  math.Abs(swap.TraderBaseChange),
				TraderQuoteVolume: math.Abs(swap.TraderQuoteChange),
				TraderSolVolume:   math.Abs(swap.TraderSolChange),
				TxCount:           1,
			}
			if err := db.Create(&traderHolder).Error; err != nil {
				return fmt.Errorf("error creating trader holder record: %v", err)
			}
		} else {
			return fmt.Errorf("error finding trader holder record: %v", result.Error)
		}
	} else {
		// Update existing record - T+1 aggregation logic
		if swap.Slot > traderHolder.LastSlot {
			traderHolder.LastSlot = swap.Slot
			traderHolder.LastTimestamp = swap.Timestamp
			traderHolder.EndSignature = swap.Signature
		}
		if traderHolder.StartSlot == 0 || swap.Slot < traderHolder.StartSlot {
			traderHolder.StartSlot = swap.Slot
			traderHolder.StartTimestamp = swap.Timestamp
			traderHolder.StartSignature = swap.Signature
		}

		// Aggregate changes
		traderHolder.BaseChange += swap.TraderBaseChange
		traderHolder.QuoteChange += swap.TraderQuoteChange
		traderHolder.SolChange += swap.TraderSolChange

		// Aggregate volumes (absolute values)
		traderHolder.TraderBaseVolume += math.Abs(swap.TraderBaseChange)
		traderHolder.TraderQuoteVolume += math.Abs(swap.TraderQuoteChange)
		traderHolder.TraderSolVolume += math.Abs(swap.TraderSolChange)
		traderHolder.TxCount++

		// 更新 holder type (可能发生变化)
		traderHolder.HolderType = holderType

		if err := db.Save(&traderHolder).Error; err != nil {
			return fmt.Errorf("error updating trader holder record: %v", err)
		}
	}

	// Update pool holder record
	var poolHolder models.PumpfunAmmpoolHolder
	result = db.Where("address = ? AND pool_address = ? AND base_mint = ? AND quote_mint = ?",
		cfg.PoolAddress, cfg.PoolAddress, cfg.BaseMint, cfg.QuoteMint).First(&poolHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new pool holder record
			poolSolChange := swap.PoolBaseAccountSolChange + swap.PoolQuoteAccountSolChange
			poolHolder = models.PumpfunAmmpoolHolder{
				Address:           cfg.PoolAddress,
				HolderType:        "pool",
				PoolAddress:       cfg.PoolAddress,
				BaseMint:          cfg.BaseMint,
				QuoteMint:         cfg.QuoteMint,
				LastSlot:          swap.Slot,
				StartSlot:         swap.Slot,
				LastTimestamp:     swap.Timestamp,
				StartTimestamp:    swap.Timestamp,
				EndSignature:      swap.Signature,
				StartSignature:    swap.Signature,
				BaseChange:        swap.PoolBaseChange,
				QuoteChange:       swap.PoolQuoteChange,
				SolChange:         poolSolChange,
				TraderBaseVolume:  math.Abs(swap.PoolBaseChange),
				TraderQuoteVolume: math.Abs(swap.PoolQuoteChange),
				TraderSolVolume:   math.Abs(poolSolChange),
				TxCount:           1,
			}
			if err := db.Create(&poolHolder).Error; err != nil {
				return fmt.Errorf("error creating pool holder record: %v", err)
			}
		} else {
			return fmt.Errorf("error finding pool holder record: %v", result.Error)
		}
	} else {
		// Update existing pool holder record
		if swap.Slot > poolHolder.LastSlot {
			poolHolder.LastSlot = swap.Slot
			poolHolder.LastTimestamp = swap.Timestamp
			poolHolder.EndSignature = swap.Signature
		}
		if poolHolder.StartSlot == 0 || swap.Slot < poolHolder.StartSlot {
			poolHolder.StartSlot = swap.Slot
			poolHolder.StartTimestamp = swap.Timestamp
			poolHolder.StartSignature = swap.Signature
		}

		poolSolChange := swap.PoolBaseAccountSolChange + swap.PoolQuoteAccountSolChange
		poolHolder.BaseChange += swap.PoolBaseChange
		poolHolder.QuoteChange += swap.PoolQuoteChange
		poolHolder.SolChange += poolSolChange
		poolHolder.TraderBaseVolume += math.Abs(swap.PoolBaseChange)
		poolHolder.TraderQuoteVolume += math.Abs(swap.PoolQuoteChange)
		poolHolder.TraderSolVolume += math.Abs(poolSolChange)
		poolHolder.TxCount++

		if err := db.Save(&poolHolder).Error; err != nil {
			return fmt.Errorf("error updating pool holder record: %v", err)
		}
	}

	return nil
}

// CheckTransactionsExistence 批量检查交易是否存在，返回存在的签名集合
func CheckTransactionsExistence(db *gorm.DB, signatures []string) (map[string]bool, error) {
	if len(signatures) == 0 {
		return make(map[string]bool), nil
	}

	var existingSignatures []string
	if err := db.Model(&models.AddressTransaction{}).
		Where("signature IN ?", signatures).
		Pluck("signature", &existingSignatures).Error; err != nil {
		return nil, fmt.Errorf("error checking transaction existence: %v", err)
	}

	existingMap := make(map[string]bool)
	for _, sig := range existingSignatures {
		existingMap[sig] = true
	}

	return existingMap, nil
}

// CreatePumpfunAmmPoolSwap creates a PumpfunAmmPool swap record，返回 swap 数据数组
func CreatePumpfunAmmPoolSwap(db *gorm.DB, tx helius.EnhancedTransaction, cfg models.PumpfunAmmPoolConfig, balanceChanges []models.AddressBalanceChange) ([]*models.PumpfunAmmPoolSwap, error) {
	var swaps []*models.PumpfunAmmPoolSwap

	// 检查是否已存在相同的 Signature 和 Address 记录（针对 tx.FeePayer）
	var existingSwap models.PumpfunAmmPoolSwap
	result := db.Where("signature = ? AND address = ?", tx.Signature, tx.FeePayer).First(&existingSwap)
	if result.Error == nil {
		// 记录已存在，返回错误
		return nil, fmt.Errorf("swap record already exists for signature %s and address %s", tx.Signature, tx.FeePayer)
	}

	// 检查是否是"记录未找到"的错误，如果是则继续创建
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing swap record: %v", result.Error)
	}

	// 创建主要的交换记录（针对 tx.FeePayer），使用传入的余额变化数据
	mainSwap := models.PumpfunAmmPoolSwap{
		Slot:                      uint(tx.Slot),
		Timestamp:                 uint(tx.Timestamp),
		PoolAddress:               cfg.PoolAddress,
		Signature:                 tx.Signature,
		Fee:                       float64(tx.Fee),
		Address:                   tx.FeePayer,
		BaseMint:                  cfg.BaseMint,
		QuoteMint:                 cfg.QuoteMint,
		TraderBaseChange:          GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, cfg.BaseMint),
		TraderQuoteChange:         GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, cfg.QuoteMint),
		TraderSolChange:           GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, "sol"),
		PoolBaseChange:            GetAggregateBalanceChanges(balanceChanges, cfg.PoolAddress, cfg.BaseMint),
		PoolQuoteChange:           GetAggregateBalanceChanges(balanceChanges, cfg.PoolAddress, cfg.QuoteMint),
		PoolBaseAccountSolChange:  GetAggregateBalanceChanges(balanceChanges, cfg.PoolBaseTokenAccount, "sol"),
		PoolQuoteAccountSolChange: GetAggregateBalanceChanges(balanceChanges, cfg.PoolQuoteTokenAccount, "sol"),
	}

	// 保存主要交换记录到数据库
	if err := db.Create(&mainSwap).Error; err != nil {
		return nil, fmt.Errorf("error creating main swap record: %v", err)
	}
	swaps = append(swaps, &mainSwap)

	logrus.Printf("Created main swap record for signature %s , address %s with trader base change: %f, trader quote change: %f",
		tx.Signature, mainSwap.Address, mainSwap.TraderBaseChange, mainSwap.TraderQuoteChange)

	// 更新主要交换记录的钱包代币统计 - base mint
	if mainSwap.TraderBaseChange != 0 {
		if err := UpdateWalletTokenStat(db, tx.FeePayer, cfg.BaseMint, mainSwap.TraderBaseChange, 6); err != nil {
			logrus.Warnf("Failed to update wallet token stat for base mint %s, address %s: %v", cfg.BaseMint, tx.FeePayer, err)
		}
	}

	// 更新主要交换记录的钱包代币统计 - quote mint
	if mainSwap.TraderQuoteChange != 0 {
		if err := UpdateWalletTokenStat(db, tx.FeePayer, cfg.QuoteMint, mainSwap.TraderQuoteChange, 9); err != nil {
			logrus.Warnf("Failed to update wallet token stat for quote mint %s, address %s: %v", cfg.QuoteMint, tx.FeePayer, err)
		}
	}

	// 更新主要交换记录的钱包代币统计 - sol
	if mainSwap.TraderSolChange != 0 {
		if err := UpdateWalletTokenStat(db, tx.FeePayer, "sol", mainSwap.TraderSolChange/math.Pow10(9), 9); err != nil {
			logrus.Warnf("Failed to update wallet token stat for sol, address %s: %v", tx.FeePayer, err)
		}
	}

	// 迭代 balanceChanges，为每个地址（除了 tx.FeePayer）创建额外的交换记录
	processedAddresses := make(map[string]bool)
	processedAddresses[tx.FeePayer] = true     // 标记 tx.FeePayer 已处理
	processedAddresses[cfg.PoolAddress] = true // 标记池子已处理

	for _, change := range balanceChanges {
		if change.Mint != cfg.BaseMint {
			continue
		}

		// 跳过 tx.FeePayer 和已处理的地址
		if processedAddresses[change.Address] {
			continue
		}
		processedAddresses[change.Address] = true

		// 检查是否已存在相同的 Signature 和 Address 记录
		var existingAdditionalSwap models.PumpfunAmmPoolSwap
		result := db.Where("signature = ? AND address = ?", tx.Signature, change.Address).First(&existingAdditionalSwap)
		if result.Error == nil {
			// 记录已存在，跳过
			logrus.Warnf("Additional swap record already exists for signature %s and address %s, skipping", tx.Signature, change.Address)
			continue
		}

		// 检查是否是"记录未找到"的错误，如果是则继续创建
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logrus.Errorf("error checking existing additional swap record: %v", result.Error)
			continue
		}

		// 创建额外的交换记录
		additionalSwap := models.PumpfunAmmPoolSwap{
			Slot:                      uint(tx.Slot),
			Timestamp:                 uint(tx.Timestamp),
			PoolAddress:               cfg.PoolAddress,
			Signature:                 tx.Signature,
			Fee:                       float64(tx.Fee),
			Address:                   change.Address,
			BaseMint:                  cfg.BaseMint,
			QuoteMint:                 cfg.QuoteMint,
			TraderBaseChange:          GetAggregateBalanceChanges(balanceChanges, change.Address, cfg.BaseMint),
			TraderQuoteChange:         GetAggregateBalanceChanges(balanceChanges, change.Address, cfg.QuoteMint),
			TraderSolChange:           GetAggregateBalanceChanges(balanceChanges, change.Address, "sol"),
			PoolBaseChange:            0,
			PoolQuoteChange:           0,
			PoolBaseAccountSolChange:  0,
			PoolQuoteAccountSolChange: 0,
		}

		// 保存额外的交换记录到数据库
		if err := db.Create(&additionalSwap).Error; err != nil {
			logrus.Errorf("error creating additional swap record for address %s: %v", change.Address, err)
			continue
		}
		swaps = append(swaps, &additionalSwap)

		logrus.Printf("Created additional swap record for signature %s, address %s with base change: %f, quote change: %f",
			tx.Signature, change.Address, additionalSwap.TraderBaseChange, additionalSwap.TraderQuoteChange)

		// 更新钱包代币统计 - base mint
		if additionalSwap.TraderBaseChange != 0 {
			if err := UpdateWalletTokenStat(db, change.Address, cfg.BaseMint, additionalSwap.TraderBaseChange, 6); err != nil {
				logrus.Warnf("Failed to update wallet token stat for base mint %s, address %s: %v", cfg.BaseMint, change.Address, err)
			}
		}

		// 更新钱包代币统计 - quote mint
		if additionalSwap.TraderQuoteChange != 0 {
			if err := UpdateWalletTokenStat(db, change.Address, cfg.QuoteMint, additionalSwap.TraderQuoteChange, 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for quote mint %s, address %s: %v", cfg.QuoteMint, change.Address, err)
			}
		}

		// 更新钱包代币统计 - sol
		if additionalSwap.TraderSolChange != 0 {
			if err := UpdateWalletTokenStat(db, change.Address, "sol", additionalSwap.TraderSolChange/math.Pow10(9), 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for sol, address %s: %v", change.Address, err)
			}
		}
	}

	return swaps, nil
}

// UpdatePumpfuninternalHolder 更新持有者信息 (T+1 logic with individual swap)
func UpdatePumpfuninternalHolder(db *gorm.DB, swap *models.PumpfuninternalSwap, cfg models.PumpfuninternalConfig) error {
	// 确定 HolderType
	holderType := "retail_investors" // 默认值

	// 检查是否为项目地址 - AddressManage 表
	var addressManage models.AddressManage
	if err := db.Where("address = ?", swap.Address).First(&addressManage).Error; err == nil {
		holderType = "project"
	} else {
		// 检查是否为项目地址 - ProjectExtraAddress 表
		var projectExtraAddress models.ProjectExtraAddress
		if err := db.Where("address = ?", swap.Address).First(&projectExtraAddress).Error; err == nil {
			holderType = "project"
		} else if swap.Address == cfg.BondingCurvePda || swap.Address == cfg.FeeRecipient || swap.Address == cfg.CreatorVaultPda {
			// 检查是否为池地址或手续费接收者或创建者
			holderType = "pool"
		}
	}

	// 更新交易者的持有者记录
	var traderHolder models.PumpfuninternalHolder
	result := db.Where("address = ? AND bonding_curve_pda = ? AND mint = ?",
		swap.Address, swap.BondingCurvePda, swap.Mint).First(&traderHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 创建新记录
			traderHolder = models.PumpfuninternalHolder{
				Address:         swap.Address,
				HolderType:      holderType,
				BondingCurvePda: swap.BondingCurvePda,
				Mint:            swap.Mint,
				LastSlot:        swap.Slot,
				StartSlot:       swap.Slot,
				LastTimestamp:   swap.Timestamp,
				StartTimestamp:  swap.Timestamp,
				EndSignature:    swap.Signature,
				StartSignature:  swap.Signature,
				MintChange:      swap.TraderMintChange,
				SolChange:       swap.TraderSolChange,
				MintVolume:      math.Abs(swap.TraderMintChange),
				SolVolume:       math.Abs(swap.TraderSolChange),
				TxCount:         1,
			}
			if err := db.Create(&traderHolder).Error; err != nil {
				return fmt.Errorf("error creating trader holder record: %v", err)
			}
		} else {
			return fmt.Errorf("error finding trader holder record: %v", result.Error)
		}
	} else {
		// 更新现有记录 - T+1 聚合逻辑
		if swap.Slot > traderHolder.LastSlot {
			traderHolder.LastSlot = swap.Slot
			traderHolder.LastTimestamp = swap.Timestamp
			traderHolder.EndSignature = swap.Signature
		}
		if traderHolder.StartSlot == 0 || swap.Slot < traderHolder.StartSlot {
			traderHolder.StartSlot = swap.Slot
			traderHolder.StartTimestamp = swap.Timestamp
			traderHolder.StartSignature = swap.Signature
		}

		// 聚合变化量
		traderHolder.MintChange += swap.TraderMintChange
		traderHolder.SolChange += swap.TraderSolChange

		// 聚合交易量 (绝对值)
		traderHolder.MintVolume += math.Abs(swap.TraderMintChange)
		traderHolder.SolVolume += math.Abs(swap.TraderSolChange)
		traderHolder.TxCount++

		// 更新 holder type (可能发生变化)
		traderHolder.HolderType = holderType

		if err := db.Save(&traderHolder).Error; err != nil {
			return fmt.Errorf("error updating trader holder record: %v", err)
		}
	}

	// 更新池子的持有者记录
	var poolHolder models.PumpfuninternalHolder
	result = db.Where("address = ? AND bonding_curve_pda = ? AND mint = ?",
		cfg.BondingCurvePda, cfg.BondingCurvePda, cfg.Mint).First(&poolHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			poolHolder = models.PumpfuninternalHolder{
				Address:         cfg.BondingCurvePda,
				HolderType:      "pool",
				BondingCurvePda: cfg.BondingCurvePda,
				Mint:            cfg.Mint,
				LastSlot:        swap.Slot,
				StartSlot:       swap.Slot,
				LastTimestamp:   swap.Timestamp,
				StartTimestamp:  swap.Timestamp,
				EndSignature:    swap.Signature,
				StartSignature:  swap.Signature,
				MintChange:      swap.PoolMintChange,
				SolChange:       swap.PoolSolChange,
				MintVolume:      math.Abs(swap.PoolMintChange),
				SolVolume:       math.Abs(swap.PoolSolChange),
				TxCount:         1,
			}
			if err := db.Create(&poolHolder).Error; err != nil {
				return fmt.Errorf("error creating pool holder record: %v", err)
			}
		} else {
			return fmt.Errorf("error finding pool holder record: %v", result.Error)
		}
	} else {
		// 更新现有记录
		if swap.Slot > poolHolder.LastSlot {
			poolHolder.LastSlot = swap.Slot
			poolHolder.LastTimestamp = swap.Timestamp
			poolHolder.EndSignature = swap.Signature
		}
		if poolHolder.StartSlot == 0 || swap.Slot < poolHolder.StartSlot {
			poolHolder.StartSlot = swap.Slot
			poolHolder.StartTimestamp = swap.Timestamp
			poolHolder.StartSignature = swap.Signature
		}

		poolHolder.MintChange += swap.PoolMintChange
		poolHolder.SolChange += swap.PoolSolChange

		// 聚合交易量 (绝对值)
		poolHolder.MintVolume += math.Abs(swap.PoolMintChange)
		poolHolder.SolVolume += math.Abs(swap.PoolSolChange)
		poolHolder.TxCount++

		if err := db.Save(&poolHolder).Error; err != nil {
			return fmt.Errorf("error updating pool holder record: %v", err)
		}
	}

	// 更新手续费接收者的持有者记录
	var feeRecipientHolder models.PumpfuninternalHolder
	result = db.Where("address = ? AND bonding_curve_pda = ? AND mint = ?",
		cfg.FeeRecipient, cfg.BondingCurvePda, cfg.Mint).First(&feeRecipientHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			feeRecipientHolder = models.PumpfuninternalHolder{
				Address:         cfg.FeeRecipient,
				HolderType:      "pool",
				BondingCurvePda: cfg.BondingCurvePda,
				Mint:            cfg.Mint,
				LastSlot:        swap.Slot,
				StartSlot:       swap.Slot,
				LastTimestamp:   swap.Timestamp,
				StartTimestamp:  swap.Timestamp,
				EndSignature:    swap.Signature,
				StartSignature:  swap.Signature,
				MintChange:      0,
				SolChange:       swap.FeeRecipientSolChange,
				MintVolume:      0,
				SolVolume:       math.Abs(swap.FeeRecipientSolChange),
				TxCount:         1,
			}
			if err := db.Create(&feeRecipientHolder).Error; err != nil {
				return fmt.Errorf("error creating fee recipient holder record: %v", err)
			}
		}
	} else {
		// 更新现有记录
		if swap.Slot > feeRecipientHolder.LastSlot {
			feeRecipientHolder.LastSlot = swap.Slot
			feeRecipientHolder.LastTimestamp = swap.Timestamp
			feeRecipientHolder.EndSignature = swap.Signature
		}
		if feeRecipientHolder.StartSlot == 0 || swap.Slot < feeRecipientHolder.StartSlot {
			feeRecipientHolder.StartSlot = swap.Slot
			feeRecipientHolder.StartTimestamp = swap.Timestamp
			feeRecipientHolder.StartSignature = swap.Signature
		}

		feeRecipientHolder.SolChange += swap.FeeRecipientSolChange

		// 聚合交易量 (绝对值)
		feeRecipientHolder.SolVolume += math.Abs(swap.FeeRecipientSolChange)
		feeRecipientHolder.TxCount++

		if err := db.Save(&feeRecipientHolder).Error; err != nil {
			return fmt.Errorf("error updating fee recipient holder record: %v", err)
		}
	}

	// 更新创建者的持有者记录
	var creatorHolder models.PumpfuninternalHolder
	result = db.Where("address = ? AND bonding_curve_pda = ? AND mint = ?",
		cfg.CreatorVaultPda, cfg.BondingCurvePda, cfg.Mint).First(&creatorHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			creatorHolder = models.PumpfuninternalHolder{
				Address:         cfg.CreatorVaultPda,
				HolderType:      "pool",
				BondingCurvePda: cfg.BondingCurvePda,
				Mint:            cfg.Mint,
				LastSlot:        swap.Slot,
				StartSlot:       swap.Slot,
				LastTimestamp:   swap.Timestamp,
				StartTimestamp:  swap.Timestamp,
				EndSignature:    swap.Signature,
				StartSignature:  swap.Signature,
				MintChange:      0,
				SolChange:       swap.CreatorSolChange,
				MintVolume:      0,
				SolVolume:       math.Abs(swap.CreatorSolChange),
				TxCount:         1,
			}
			if err := db.Create(&creatorHolder).Error; err != nil {
				return fmt.Errorf("error creating creator holder record: %v", err)
			}
		} else {
			return fmt.Errorf("error finding creator holder record: %v", result.Error)
		}
	} else {
		// 更新现有记录
		if swap.Slot > creatorHolder.LastSlot {
			creatorHolder.LastSlot = swap.Slot
			creatorHolder.LastTimestamp = swap.Timestamp
			creatorHolder.EndSignature = swap.Signature
		}
		if creatorHolder.StartSlot == 0 || swap.Slot < creatorHolder.StartSlot {
			creatorHolder.StartSlot = swap.Slot
			creatorHolder.StartTimestamp = swap.Timestamp
			creatorHolder.StartSignature = swap.Signature
		}

		creatorHolder.SolChange += swap.CreatorSolChange

		// 聚合交易量 (绝对值)
		creatorHolder.SolVolume += math.Abs(swap.CreatorSolChange)
		creatorHolder.TxCount++

		if err := db.Save(&creatorHolder).Error; err != nil {
			return fmt.Errorf("error updating creator holder record: %v", err)
		}
	}

	return nil
}

// ProcessPumpfuninternalConfig processes transactions for a single Pumpfun internal configuration
func ProcessPumpfuninternalConfig(ctx context.Context, db *gorm.DB, heliusClient *helius.Client, cfg models.PumpfuninternalConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	// 获取或创建监控配置
	monitorConfig, err := GetOrCreateTransactionsMonitorConfig(db, cfg.AssociatedBondingCurve)
	if err != nil {
		logrus.Errorf("获取监控配置失败 %s: %v", cfg.AssociatedBondingCurve, err)
		return
	}

	// 检查是否启用监控
	if !monitorConfig.Enabled {
		return
	}

	var oldOne *helius.EnhancedTransaction
	hasCalledAPI := false // 新增标志，用于跟踪是否调用过API
	currentPage := 0      // 当前页数

	// 保存并记录配置更新
	saveConfig := func() {
		// 检查 context 是否已取消
		if ctx.Err() != nil {
			logrus.Warnf("Context cancelled during saveConfig for address: %s", cfg.AssociatedBondingCurve)
			return
		}

		// 如果调用过API，更新LastExecution
		if hasCalledAPI {
			monitorConfig.LastExecution = uint(time.Now().Unix())
		}

		// 保存配置
		if err := db.Save(monitorConfig).Error; err != nil {
			logrus.Errorf("更新监控配置失败 %s: %v", cfg.AssociatedBondingCurve, err)
		}
	}

mainLoop:
	for {
		select {
		case <-ctx.Done():
			logrus.Warnf("Context cancelled for address: %s, reason: %v", cfg.AssociatedBondingCurve, ctx.Err())
			break mainLoop
		default:
			// 检查是否达到最大页数
			if currentPage >= MAX_PAGE {
				break mainLoop
			}
			currentPage++

			// 设置查询选项
			limit := 100
			opts := helius.TransactionOptions{
				Limit: &limit,
			}

			if oldOne != nil {
				opts.Before = helius.StringPtr(oldOne.Signature)
			}

			// 使用带超时的 sleep
			timer := time.NewTimer(time.Duration(apiRequestDelay) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				break mainLoop
			case <-timer.C:
			}

			// 获取交易数据
			logrus.Printf("GetEnhancedTransactionsByAddress for address: %s, page: %d", cfg.AssociatedBondingCurve, currentPage)

			// 创建带超时的子 context
			apiCtx, apiCancel := context.WithTimeout(ctx, 30*time.Second)
			defer apiCancel()

			// 检查 context 是否已取消
			if apiCtx.Err() != nil {
				break mainLoop
			}

			transactions, err := heliusClient.GetEnhancedTransactionsByAddress(cfg.AssociatedBondingCurve, &opts)

			hasCalledAPI = true // 标记已调用API

			if err != nil {
				if ctx.Err() != nil {
					// context 已取消，退出循环
					break mainLoop
				}
				logrus.Errorf("获取交易数据失败 %s: %v", cfg.AssociatedBondingCurve, err)

				// 使用带超时的 sleep
				timer := time.NewTimer(5 * time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					break mainLoop
				case <-timer.C:
				}
				continue
			}

			if len(transactions) == 0 {
				logrus.Printf("Completed transactions processing, transactions is empty.")
				break mainLoop
			}

			oldOne = &transactions[len(transactions)-1]

			// 批量检查交易是否存在
			signatures := make([]string, len(transactions))
			for i, tx := range transactions {
				signatures[i] = tx.Signature
			}

			existingTxMap, err := CheckTransactionsExistence(db, signatures)
			if err != nil {
				logrus.Errorf("批量检查交易存在性失败: %v", err)
				continue
			}

			// 遍历交易数据并保存
			for _, tx := range transactions {
				// 检查 context 是否已取消
				if ctx.Err() != nil {
					break mainLoop
				}

				// 使用批量查询结果检查交易是否已存在
				if existingTxMap[tx.Signature] {
					continue
				}

				// 准备交易数据
				data, err := json.Marshal(tx)
				if err != nil {
					logrus.Errorf("序列化交易数据失败: %v", err)
					continue
				}

				// 创建新的地址交易记录
				addressTx := models.AddressTransaction{
					Address:   cfg.AssociatedBondingCurve,
					Signature: tx.Signature,
					FeePayer:  tx.FeePayer,
					Fee:       float64(tx.Fee),
					Slot:      uint(tx.Slot),
					Timestamp: uint(tx.Timestamp),
					Type:      tx.Type,
					Source:    tx.Source,
					Data:      data,
				}

				// 保存到数据库
				if err := db.Create(&addressTx).Error; err != nil {
					logrus.Errorf("保存交易记录失败 %s: %v", tx.Signature, err)
					continue
				}

				// Handle CREATE_POOL transactions
				var pumpfunAmmPoolConfig *models.PumpfunAmmPoolConfig
				if tx.Type == "CREATE_POOL" {
					// 1. CreatePumpfunAmmPoolConfigWithTransaction
					var monitorConfigNew *models.TransactionsMonitorConfig
					var err error
					pumpfunAmmPoolConfig, monitorConfigNew, err = CreatePumpfunAmmPoolConfigWithTransaction(db, tx, cfg)
					if err != nil {
						logrus.Errorf("Failed to create PumpfunAmmPoolConfig %s: %v", tx.Signature, err)
						// continue
					} else {
						// 1.5. UpdateProjectConfig
						if err := UpdateProjectConfig(cfg, pumpfunAmmPoolConfig); err != nil {
							logrus.Errorf("Failed to update ProjectConfig %s: %v", tx.Signature, err)
						}
					}

					// 2. CreatePumpfunAmmPoolMigrateTxData
					if err := CreatePumpfunAmmPoolMigrateTxData(db, tx, pumpfunAmmPoolConfig, monitorConfigNew); err != nil {
						logrus.Errorf("Failed to migrate transaction data %s: %v", tx.Signature, err)
						// continue
					}

					logrus.Printf("Successfully processed CREATE_POOL transaction %s with pool address %s", tx.Signature, pumpfunAmmPoolConfig.PoolAddress)
					// continue // Skip normal processing for CREATE_POOL transactions
				}

				// 获取余额变化数据（不写入数据库）
				balanceChanges := CreateAddressBalanceChange(tx, cfg.BondingCurvePda)

				// 如果是 CREATE_POOL 类型且有 pumpfunAmmPoolConfig，则去除 Creator 的数据
				if tx.Type == "CREATE_POOL" && pumpfunAmmPoolConfig != nil {
					var filteredBalanceChanges []models.AddressBalanceChange
					for _, change := range balanceChanges {
						if change.Address != pumpfunAmmPoolConfig.Creator {
							filteredBalanceChanges = append(filteredBalanceChanges, change)
						}
					}
					balanceChanges = filteredBalanceChanges
				}

				// 创建 Pumpfun 内部交换记录，使用内存中的余额变化数据
				swaps, err := CreatePumpfuninternalSwap(db, tx, cfg, balanceChanges)
				if err != nil {
					logrus.Errorf("创建交换记录失败 %s: %v", tx.Signature, err)
					continue
				}

				// 实时更新持有者信息 (T+1 logic)
				for _, swap := range swaps {
					if err := UpdatePumpfuninternalHolder(db, swap, cfg); err != nil {
						logrus.Errorf("更新持有者信息失败 %s: %v", tx.Signature, err)
					}
				}

				// 更新监控配置
				txSlot := uint(tx.Slot)
				txTimestamp := uint(tx.Timestamp)

				if txSlot > monitorConfig.LastSlot {
					monitorConfig.LastSlot = txSlot
					monitorConfig.LastTimestamp = txTimestamp
					monitorConfig.LastSignature = tx.Signature
				}

				if monitorConfig.StartSlot == 0 || txSlot < monitorConfig.StartSlot {
					monitorConfig.StartSlot = txSlot
					monitorConfig.StartTimestamp = txTimestamp
					monitorConfig.StartSignature = tx.Signature
				}

				monitorConfig.TxCount++
			}

			// 保存配置
			saveConfig()
		}
	}

	// 最后一次保存配置，确保更新 LastExecution 和重置 Retry
	saveConfig()
	logrus.Printf("Completed ProcessPumpfuninternalConfig for address: %s", cfg.AssociatedBondingCurve)
}

// ProcessPumpfunAmmPoolConfig processes transactions for a single AMM pool configuration
func ProcessPumpfunAmmPoolConfig(ctx context.Context, db *gorm.DB, heliusClient *helius.Client, cfg models.PumpfunAmmPoolConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get or create monitor config
	monitorConfig, err := GetOrCreateTransactionsMonitorConfig(db, cfg.PoolAddress)
	if err != nil {
		logrus.Errorf("Failed to get monitor config for %s: %v", cfg.PoolAddress, err)
		return
	}

	// Check if monitoring is enabled
	if !monitorConfig.Enabled {
		return
	}

	var oldOne *helius.EnhancedTransaction
	hasCalledAPI := false
	currentPage := 0

	// Save and record config updates
	saveConfig := func() {
		// Check if context is cancelled
		if ctx.Err() != nil {
			logrus.Warnf("Context cancelled during saveConfig for address: %s", cfg.PoolAddress)
			return
		}

		// Update LastExecution if API was called
		if hasCalledAPI {
			monitorConfig.LastExecution = uint(time.Now().Unix())
		}

		// Save config
		if err := db.Save(monitorConfig).Error; err != nil {
			logrus.Errorf("Failed to update monitor config for %s: %v", cfg.PoolAddress, err)
		}
	}

mainLoop:
	for {
		select {
		case <-ctx.Done():
			logrus.Warnf("Context cancelled for address: %s, reason: %v", cfg.PoolAddress, ctx.Err())
			break mainLoop
		default:
			// Check if max page count reached
			if currentPage >= MAX_PAGE {
				break mainLoop
			}
			currentPage++

			// Set query options
			limit := 100
			opts := helius.TransactionOptions{
				Limit: &limit,
			}

			if oldOne != nil {
				opts.Before = helius.StringPtr(oldOne.Signature)
			}

			// Sleep with timeout
			timer := time.NewTimer(time.Duration(apiRequestDelay) * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				break mainLoop
			case <-timer.C:
			}

			// Get transaction data
			logrus.Printf("GetEnhancedTransactionsByAddress for address: %s, page: %d", cfg.PoolAddress, currentPage)

			// Create timeout context
			apiCtx, apiCancel := context.WithTimeout(ctx, 30*time.Second)
			defer apiCancel()

			// Check if context is cancelled
			if apiCtx.Err() != nil {
				break mainLoop
			}

			transactions, err := heliusClient.GetEnhancedTransactionsByAddress(cfg.PoolAddress, &opts)

			hasCalledAPI = true

			if err != nil {
				if ctx.Err() != nil {
					break mainLoop
				}
				logrus.Errorf("Failed to get transactions for %s: %v", cfg.PoolAddress, err)

				timer := time.NewTimer(5 * time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					break mainLoop
				case <-timer.C:
				}
				continue
			}

			if len(transactions) == 0 {
				logrus.Printf("Completed transactions processing, transactions is empty.")
				break mainLoop
			}

			oldOne = &transactions[len(transactions)-1]

			// 批量检查交易是否存在
			signatures := make([]string, len(transactions))
			for i, tx := range transactions {
				signatures[i] = tx.Signature
			}

			existingTxMap, err := CheckTransactionsExistence(db, signatures)
			if err != nil {
				logrus.Errorf("批量检查交易存在性失败: %v", err)
				continue
			}

			// Process transactions
			for _, tx := range transactions {
				if ctx.Err() != nil {
					break mainLoop
				}

				// 使用批量查询结果检查交易是否已存在
				if existingTxMap[tx.Signature] {
					continue
				}

				// Prepare transaction data
				data, err := json.Marshal(tx)
				if err != nil {
					logrus.Errorf("Failed to serialize transaction data: %v", err)
					continue
				}

				// Create new address transaction
				addressTx := models.AddressTransaction{
					Address:   cfg.PoolAddress,
					Signature: tx.Signature,
					FeePayer:  tx.FeePayer,
					Fee:       float64(tx.Fee),
					Slot:      uint(tx.Slot),
					Timestamp: uint(tx.Timestamp),
					Type:      tx.Type,
					Source:    tx.Source,
					Data:      data,
				}

				// Save to database
				if err := db.Create(&addressTx).Error; err != nil {
					logrus.Errorf("Failed to save transaction record %s: %v", tx.Signature, err)
					continue
				}

				// 获取余额变化数据（不写入数据库）
				balanceChanges := CreateAddressBalanceChange(tx, cfg.PoolAddress)

				// Create AMM pool swap record using the balance changes from memory
				swaps, err := CreatePumpfunAmmPoolSwap(db, tx, cfg, balanceChanges)
				if err != nil {
					logrus.Errorf("Failed to create swap record %s: %v", tx.Signature, err)
					continue
				}

				// 实时更新持有者信息 (T+1 logic)
				for _, swap := range swaps {
					if err := UpdatePumpfunAmmPoolHolder(db, swap, cfg); err != nil {
						logrus.Errorf("Failed to update holder information %s: %v", tx.Signature, err)
					}
				}

				// Update monitor config
				txSlot := uint(tx.Slot)
				txTimestamp := uint(tx.Timestamp)

				if txSlot > monitorConfig.LastSlot {
					monitorConfig.LastSlot = txSlot
					monitorConfig.LastTimestamp = txTimestamp
					monitorConfig.LastSignature = tx.Signature
				}

				if monitorConfig.StartSlot == 0 || txSlot < monitorConfig.StartSlot {
					monitorConfig.StartSlot = txSlot
					monitorConfig.StartTimestamp = txTimestamp
					monitorConfig.StartSignature = tx.Signature
				}

				monitorConfig.TxCount++
			}

			// Save config
			saveConfig()
		}
	}

	// Final config save
	saveConfig()
	logrus.Printf("Completed ProcessPumpfunAmmPoolConfig for address: %s", cfg.PoolAddress)
}

// UpdatePumpfunInternalTransactions 更新 Pumpfun 内部交易数据
func UpdatePumpfunInternalTransactions(ctx context.Context) error {
	// 尝试获取锁，如果已经在执行则直接返回
	if !updateMutex.TryLock() {
		logrus.Info("Previous update is still running, skipping this round")
		return nil
	}
	defer updateMutex.Unlock()

	// 初始化数据库连接
	config.InitDB()
	db := config.DB
	if db == nil {
		return fmt.Errorf("failed to initialize database")
	}

	// 初始化 Helius API 客户端
	heliusAPIKey := os.Getenv("HELIUS_API_KEY")
	if heliusAPIKey == "" {
		return fmt.Errorf("HELIUS_API_KEY environment variable is not set")
	}
	heliusClient := helius.NewClient(heliusAPIKey)

	// 获取所有状态为 active 的 PumpfuninternalConfig
	var configs []models.PumpfuninternalConfig
	if err := db.Where("status = ?", "active").Find(&configs).Error; err != nil {
		return err
	}

	// 使用 worker pool 处理配置
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for _, config := range configs {
		// 尝试获取地址的锁
		if _, loaded := addressLocks.LoadOrStore(config.AssociatedBondingCurve, true); loaded {
			logrus.Infof("Address %s is already being processed, skipping", config.AssociatedBondingCurve)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(cfg models.PumpfuninternalConfig) {
			defer func() {
				<-semaphore                                     // 释放信号量
				addressLocks.Delete(cfg.AssociatedBondingCurve) // 释放地址锁
			}()
			ProcessPumpfuninternalConfig(ctx, db, heliusClient, cfg, &wg)
		}(config)
	}

	wg.Wait()
	return nil
}

// UpdatePumpfunAmmPoolTransactions updates AMM pool transaction data
func UpdatePumpfunAmmPoolTransactions(ctx context.Context) error {
	// Try to acquire lock
	if !updateMutex.TryLock() {
		logrus.Info("Previous update is still running, skipping this round")
		return nil
	}
	defer updateMutex.Unlock()

	// Initialize database connection
	config.InitDB()
	db := config.DB
	if db == nil {
		return fmt.Errorf("failed to initialize database")
	}

	// Initialize Helius API client
	heliusAPIKey := os.Getenv("HELIUS_API_KEY")
	if heliusAPIKey == "" {
		return fmt.Errorf("HELIUS_API_KEY environment variable is not set")
	}
	heliusClient := helius.NewClient(heliusAPIKey)

	// Get all active PumpfunAmmPoolConfig
	var configs []models.PumpfunAmmPoolConfig
	if err := db.Where("status = ?", "active").Find(&configs).Error; err != nil {
		return err
	}

	// Use worker pool to process configurations
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	for _, config := range configs {
		// Try to acquire address lock
		if _, loaded := addressLocks.LoadOrStore(config.PoolAddress, true); loaded {
			logrus.Infof("Address %s is already being processed, skipping", config.PoolAddress)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(cfg models.PumpfunAmmPoolConfig) {
			defer func() {
				<-semaphore                          // Release semaphore
				addressLocks.Delete(cfg.PoolAddress) // Release address lock
			}()
			ProcessPumpfunAmmPoolConfig(ctx, db, heliusClient, cfg, &wg)
		}(config)
	}

	wg.Wait()
	return nil
}

// CreatePumpfunAmmPoolConfigWithTransaction creates a PumpfunAmmPoolConfig based on transaction data
func CreatePumpfunAmmPoolConfigWithTransaction(db *gorm.DB, tx helius.EnhancedTransaction, cfg models.PumpfuninternalConfig) (*models.PumpfunAmmPoolConfig, *models.TransactionsMonitorConfig, error) {
	userPubkey := tx.FeePayer
	baseMintPubkey := cfg.Mint

	// Find creator by iterating through tokenTransfers
	var creatorPubkey string
	for _, tokenTransfer := range tx.TokenTransfers {
		// Check conditions: mint matches, from is bonding curve, amount > 200000000
		if tokenTransfer.Mint == cfg.Mint &&
			tokenTransfer.FromUserAccount == cfg.BondingCurvePda &&
			tokenTransfer.TokenAmount > 200000000 {
			creatorPubkey = tokenTransfer.ToUserAccount
			break
		}
	}

	if creatorPubkey == "" {
		return nil, nil, fmt.Errorf("creator not found in token transfers")
	}

	// Get Creator from TokenConfig
	var tokenConfig models.TokenConfig
	if err := db.Where("mint = ?", baseMintPubkey).First(&tokenConfig).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to find TokenConfig for mint %s: %v", baseMintPubkey, err)
	}

	coinCreator := tokenConfig.Creator
	if coinCreator == "" {
		return nil, nil, fmt.Errorf("TokenConfig has no creator specified for mint %s", baseMintPubkey)
	}

	// Parse public keys
	userPubkeyParsed, err := solana.PublicKeyFromBase58(userPubkey)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid user pubkey: %v", err)
	}

	creatorPubkeyParsed, err := solana.PublicKeyFromBase58(creatorPubkey)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid creator pubkey: %v", err)
	}

	baseMintPubkeyParsed, err := solana.PublicKeyFromBase58(baseMintPubkey)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid base mint pubkey: %v", err)
	}

	coinCreatorPubkeyParsed, err := solana.PublicKeyFromBase58(coinCreator)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid coin creator pubkey: %v", err)
	}

	// Call GetAllPumpSwapPDAs to get pool information
	pdaInfo, err := pumpsolana.GetAllPumpSwapPDAs(userPubkeyParsed, creatorPubkeyParsed, baseMintPubkeyParsed, coinCreatorPubkeyParsed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get PumpSwap PDAs: %v", err)
	}

	poolAddress := pdaInfo.Pool.Address.String()

	// Validate by checking if pool address appears in tokenTransfers
	poolAddressFound := false
	for _, tokenTransfer := range tx.TokenTransfers {
		if tokenTransfer.ToUserAccount == poolAddress {
			poolAddressFound = true
			break
		}
	}

	if !poolAddressFound {
		return nil, nil, fmt.Errorf("pool address %s not found in token transfers", poolAddress)
	}

	// Create PumpfunAmmPoolConfig
	pumpfunAmmPoolConfig := &models.PumpfunAmmPoolConfig{
		PoolAddress:           pdaInfo.Pool.Address.String(),
		PoolBump:              pdaInfo.Pool.Bump,
		Index:                 0, // Default index
		Creator:               creatorPubkey,
		BaseMint:              baseMintPubkey,
		QuoteMint:             "So11111111111111111111111111111111111111112", // WSOL
		LpMint:                pdaInfo.PoolLpMint.Address.String(),
		PoolBaseTokenAccount:  pdaInfo.PoolBaseTokenAccount.String(),
		PoolQuoteTokenAccount: pdaInfo.PoolQuoteTokenAccount.String(),
		LpSupply:              0, // Initial LP supply
		CoinCreator:           coinCreator,
		Status:                "active",
	}

	// Save to database
	if err := db.Create(pumpfunAmmPoolConfig).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create PumpfunAmmPoolConfig: %v", err)
	}

	// Get or create TransactionsMonitorConfig
	monitorConfig, err := GetOrCreateTransactionsMonitorConfig(db, poolAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get or create monitor config: %v", err)
	}

	return pumpfunAmmPoolConfig, monitorConfig, nil
}

// CreatePumpfunAmmPoolMigrateTxData processes transaction data for AMM pool migration
func CreatePumpfunAmmPoolMigrateTxData(db *gorm.DB, tx helius.EnhancedTransaction, pumpfunAmmPoolConfig *models.PumpfunAmmPoolConfig, monitorConfig *models.TransactionsMonitorConfig) error {
	// 获取余额变化数据（不写入数据库）
	balanceChanges := CreateAddressBalanceChange(tx, pumpfunAmmPoolConfig.PoolAddress)

	// 去除 Address 为 pumpfunAmmPoolConfig.Creator 的 balanceChanges
	var filteredBalanceChanges []models.AddressBalanceChange
	for _, change := range balanceChanges {
		if change.Address != pumpfunAmmPoolConfig.Creator {
			filteredBalanceChanges = append(filteredBalanceChanges, change)
		}
	}

	// Create AMM pool swap record using the filtered balance changes from memory
	swaps, err := CreatePumpfunAmmPoolSwap(db, tx, *pumpfunAmmPoolConfig, filteredBalanceChanges)
	if err != nil {
		logrus.Errorf("Failed to create swap record %s: %v", tx.Signature, err)
		return err
	}

	// 实时更新持有者信息 (T+1 logic)
	for _, swap := range swaps {
		if err := UpdatePumpfunAmmPoolHolder(db, swap, *pumpfunAmmPoolConfig); err != nil {
			logrus.Errorf("Failed to update holder information %s: %v", tx.Signature, err)
		}
	}

	// Update monitor config
	txSlot := uint(tx.Slot)
	txTimestamp := uint(tx.Timestamp)

	if txSlot > monitorConfig.LastSlot {
		monitorConfig.LastSlot = txSlot
		monitorConfig.LastTimestamp = txTimestamp
		monitorConfig.LastSignature = tx.Signature
	}

	if monitorConfig.StartSlot == 0 || txSlot < monitorConfig.StartSlot {
		monitorConfig.StartSlot = txSlot
		monitorConfig.StartTimestamp = txTimestamp
		monitorConfig.StartSignature = tx.Signature
	}

	monitorConfig.TxCount++

	// Save updated monitor config
	if err := db.Save(monitorConfig).Error; err != nil {
		logrus.Errorf("Failed to save monitor config: %v", err)
		return err
	}

	return nil
}

// UpdateProjectConfig updates ProjectConfig when transitioning to AMM pool
func UpdateProjectConfig(cfg models.PumpfuninternalConfig, pumpfunAmmPoolConfig *models.PumpfunAmmPoolConfig) error {
	db := config.DB
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// 1. 找到 BondingCurvePda 为 cfg.BondingCurvePda 的 PumpfuninternalConfig，获取其 ID
	var pumpfuninternalConfig models.PumpfuninternalConfig
	result := db.Where("bonding_curve_pda = ?", cfg.BondingCurvePda).First(&pumpfuninternalConfig)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			logrus.Warnf("No PumpfuninternalConfig found for bonding_curve_pda %s", cfg.BondingCurvePda)
			return nil // Not an error, just no config exists
		}
		return fmt.Errorf("error finding PumpfuninternalConfig: %v", result.Error)
	}

	// 2. 找到 ProjectConfig.PoolID 为该 ID 的 ProjectConfig
	var projectConfigs []models.ProjectConfig
	result = db.Where("pool_id = ?", pumpfuninternalConfig.ID).Find(&projectConfigs)

	if result.Error != nil {
		return fmt.Errorf("error finding ProjectConfig: %v", result.Error)
	}

	if len(projectConfigs) == 0 {
		logrus.Warnf("No ProjectConfig found for pool_id %d", pumpfuninternalConfig.ID)
		return nil // Not an error, just no project configs exist
	}

	// 3. 更新所有符合条件的 ProjectConfig
	updatedCount := 0
	for i := range projectConfigs {
		projectConfigs[i].PoolPlatform = "pumpfun_amm"
		projectConfigs[i].PoolID = pumpfunAmmPoolConfig.ID

		if err := db.Save(&projectConfigs[i]).Error; err != nil {
			logrus.Errorf("Failed to update ProjectConfig ID %d: %v", projectConfigs[i].ID, err)
			continue
		}
		updatedCount++

		logrus.Printf("Updated ProjectConfig ID %d: PoolPlatform=%s, PoolID=%d",
			projectConfigs[i].ID, projectConfigs[i].PoolPlatform, projectConfigs[i].PoolID)
	}

	logrus.Printf("Successfully updated %d ProjectConfig records for bonding_curve_pda %s",
		updatedCount, cfg.BondingCurvePda)

	return nil
}
