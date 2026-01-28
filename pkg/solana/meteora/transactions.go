package meteora

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	"marketcontrol/pkg/helius"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	maxWorkers      = 3    // 最大并发工作协程数
	apiRequestDelay = 3000 // API 请求间隔（毫秒）
	MAX_PAGE        = 2    // 最大翻页次数
)

var (
	updateMutex  sync.Mutex // 添加互斥锁
	addressLocks sync.Map   // 地址级别的锁映射
)

// SortTransactions sorts enhanced transactions by the given field and order.
// Supported sortField: "slot", "timestamp", "fee", "signature"
// sortType: "asc" or "desc"
func SortTransactions(sortField, sortType string, transactions []helius.EnhancedTransaction) []helius.EnhancedTransaction {
	if len(transactions) < 2 {
		return transactions
	}
	field := strings.ToLower(sortField)
	desc := strings.ToLower(sortType) == "desc"

	switch field {
	case "slot":
		sort.Slice(transactions, func(i, j int) bool {
			if desc {
				return transactions[i].Slot > transactions[j].Slot
			}
			return transactions[i].Slot < transactions[j].Slot
		})
	case "timestamp":
		sort.Slice(transactions, func(i, j int) bool {
			if desc {
				return transactions[i].Timestamp > transactions[j].Timestamp
			}
			return transactions[i].Timestamp < transactions[j].Timestamp
		})
	case "fee":
		sort.Slice(transactions, func(i, j int) bool {
			if desc {
				return transactions[i].Fee > transactions[j].Fee
			}
			return transactions[i].Fee < transactions[j].Fee
		})
	case "signature":
		sort.Slice(transactions, func(i, j int) bool {
			if desc {
				return transactions[i].Signature > transactions[j].Signature
			}
			return transactions[i].Signature < transactions[j].Signature
		})
	default:
		// unsupported field; return as-is
		return transactions
	}
	return transactions
}

// MeteoradbcPoolConfig 定义通用的池配置接口
type MeteoradbcPoolConfig interface {
	GetPoolAddress() string
	GetBaseMint() string
	GetQuoteMint() string
	GetBaseVault() string
	GetQuoteVault() string
	GetStatus() string
}

// MeteoradbcConfig 包装类型
type MeteoradbcConfig struct {
	models.MeteoradbcConfig
}

// 为 MeteoradbcConfig 实现接口
func (cfg MeteoradbcConfig) GetPoolAddress() string { return cfg.PoolAddress }
func (cfg MeteoradbcConfig) GetBaseMint() string    { return cfg.BaseMint }
func (cfg MeteoradbcConfig) GetQuoteMint() string   { return cfg.QuoteMint }
func (cfg MeteoradbcConfig) GetBaseVault() string   { return cfg.PoolBaseTokenAccount }
func (cfg MeteoradbcConfig) GetQuoteVault() string  { return cfg.PoolQuoteTokenAccount }
func (cfg MeteoradbcConfig) GetStatus() string      { return cfg.Status }

// MeteoracpmmConfig 包装类型
type MeteoracpmmConfig struct {
	models.MeteoracpmmConfig
}

// 为 MeteoracpmmConfig 实现接口
func (cfg MeteoracpmmConfig) GetPoolAddress() string { return cfg.PoolAddress }
func (cfg MeteoracpmmConfig) GetBaseMint() string    { return cfg.BaseMint }
func (cfg MeteoracpmmConfig) GetQuoteMint() string   { return cfg.QuoteMint }
func (cfg MeteoracpmmConfig) GetBaseVault() string   { return cfg.PoolBaseTokenAccount }
func (cfg MeteoracpmmConfig) GetQuoteVault() string  { return cfg.PoolQuoteTokenAccount }
func (cfg MeteoracpmmConfig) GetStatus() string      { return cfg.Status }

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

// MeteoracpmmTxHandler 处理单个 Meteoracpmm 交易
func MeteoracpmmTxHandler(db *gorm.DB, tx helius.EnhancedTransaction, cfg MeteoradbcPoolConfig, monitorConfig *models.TransactionsMonitorConfig) error {
	// 获取余额变化数据（不写入数据库）
	balanceChanges := CreateAddressBalanceChange(tx, cfg)

	// Create Meteoracpmm pool swap record using the balance changes from memory
	swaps, err := CreateMeteoracpmmSwap(db, tx, cfg, balanceChanges)
	if err != nil {
		return fmt.Errorf("failed to create swap record %s: %v", tx.Signature, err)
	}

	// 实时更新持有者信息 (T+1 logic)
	for _, swap := range swaps {
		if err := UpdateMeteoracpmmHolder(db, swap, cfg); err != nil {
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
	return nil
}

// CreateAddressBalanceChangeWithMigration 创建地址余额变化记录（支持迁移检测），返回内存中的变化数据和是否迁移标志
func CreateAddressBalanceChangeWithMigration(db *gorm.DB, tx helius.EnhancedTransaction, cfg MeteoradbcPoolConfig) ([]models.AddressBalanceChange, bool, error) {
	var changes []models.AddressBalanceChange
	isMigrated := false

	// 获取基本交易信息
	slot := uint(tx.Slot)
	timestamp := uint(tx.Timestamp)
	signature := tx.Signature

	// 尝试获取对应的 MeteoracpmmConfig（如果 cfg 是 MeteoradbcConfig 且有 DammV2PoolAddress）
	var meteoracpmmConfig *models.MeteoracpmmConfig
	if meteoradbcCfg, ok := cfg.(MeteoradbcConfig); ok && meteoradbcCfg.DammV2PoolAddress != "" {
		var cpmmConfig models.MeteoracpmmConfig
		if err := db.Where("pool_address = ?", meteoradbcCfg.DammV2PoolAddress).First(&cpmmConfig).Error; err == nil {
			meteoracpmmConfig = &cpmmConfig
		}
	}

	// 处理代币转账 - 检查是否涉及目标交易所的逻辑
	for _, tokenTransfer := range tx.TokenTransfers {
		// 检查是否涉及目标交易所
		isTradeWithExchange := false

		// 当 Mint 为 base_mint 且 fromTokenAccount 或者 toTokenAccount 为 base_vault 时，为 true
		if tokenTransfer.Mint == cfg.GetBaseMint() &&
			(tokenTransfer.FromTokenAccount == cfg.GetBaseVault() || tokenTransfer.ToTokenAccount == cfg.GetBaseVault()) {
			isTradeWithExchange = true
		}

		// 当 Mint 为 quote_mint 且 fromTokenAccount 或者 toTokenAccount 为 quote_vault 时，为 true
		if tokenTransfer.Mint == cfg.GetQuoteMint() &&
			(tokenTransfer.FromTokenAccount == cfg.GetQuoteVault() || tokenTransfer.ToTokenAccount == cfg.GetQuoteVault()) {
			isTradeWithExchange = true
		}

		if !isTradeWithExchange {
			continue
		}

		// 创建发送方余额变化记录
		fromAddress := tokenTransfer.FromUserAccount
		// 当 tokenTransfer.fromTokenAccount 为 base_vault 或者 quote_vault，则 Address 为 cfg.PoolAddress
		if tokenTransfer.FromTokenAccount == cfg.GetBaseVault() || tokenTransfer.FromTokenAccount == cfg.GetQuoteVault() {
			fromAddress = cfg.GetPoolAddress()
		}

		fromChange := models.AddressBalanceChange{
			Slot:         slot,
			Timestamp:    timestamp,
			Signature:    signature,
			Address:      fromAddress,
			Mint:         tokenTransfer.Mint,
			AmountChange: tokenTransfer.TokenAmount * -1, // 负值表示转出
		}
		changes = append(changes, fromChange)

		// 检查是否为迁移交易
		// 如果 tokenTransfer.ToTokenAccount 为 MeteoracpmmConfig 的 base_vault 或 quote_vault，则设置 isMigrated = true
		if meteoracpmmConfig != nil {
			wrappedCpmmCfg := MeteoracpmmConfig{*meteoracpmmConfig}
			if tokenTransfer.ToTokenAccount == wrappedCpmmCfg.GetBaseVault() || tokenTransfer.ToTokenAccount == wrappedCpmmCfg.GetQuoteVault() {
				isMigrated = true
			}
		}

		// 创建接收方余额变化记录（默认创建）
		toAddress := tokenTransfer.ToUserAccount
		// 当 tokenTransfer.toTokenAccount 为 base_vault 或者 quote_vault，则 Address 为 cfg.PoolAddress
		if tokenTransfer.ToTokenAccount == cfg.GetBaseVault() || tokenTransfer.ToTokenAccount == cfg.GetQuoteVault() {
			toAddress = cfg.GetPoolAddress()
		}

		toChange := models.AddressBalanceChange{
			Slot:         slot,
			Timestamp:    timestamp,
			Signature:    signature,
			Address:      toAddress,
			Mint:         tokenTransfer.Mint,
			AmountChange: tokenTransfer.TokenAmount, // 正值表示转入
		}
		changes = append(changes, toChange)
	}

	// 处理原生代币(SOL)余额变化
	for _, accountData := range tx.AccountData {
		if accountData.NativeBalanceChange != 0 {
			// 确定地址：如果 accountData.Account 是 base_vault 或 quote_vault，则使用 pool_address
			address := accountData.Account
			if accountData.Account == cfg.GetBaseVault() || accountData.Account == cfg.GetQuoteVault() {
				address = cfg.GetPoolAddress()
			}

			// 创建 SOL 余额变化记录
			nativeChange := models.AddressBalanceChange{
				Slot:         slot,
				Timestamp:    timestamp,
				Signature:    signature,
				Address:      address,
				Mint:         "sol", // 使用 "sol" 表示原生代币
				AmountChange: float64(accountData.NativeBalanceChange),
			}
			changes = append(changes, nativeChange)
		}
	}

	return changes, isMigrated, nil
}

// CreateAddressBalanceChange 创建地址余额变化记录，返回内存中的变化数据而不写入数据库
func CreateAddressBalanceChange(tx helius.EnhancedTransaction, cfg MeteoradbcPoolConfig) []models.AddressBalanceChange {
	var changes []models.AddressBalanceChange

	// 获取基本交易信息
	slot := uint(tx.Slot)
	timestamp := uint(tx.Timestamp)
	signature := tx.Signature

	// 处理代币转账 - 检查是否涉及目标交易所的逻辑
	for _, tokenTransfer := range tx.TokenTransfers {
		// 检查是否涉及目标交易所
		isTradeWithExchange := false

		// 当 Mint 为 base_mint 且 fromTokenAccount 或者 toTokenAccount 为 base_vault 时，为 true
		if tokenTransfer.Mint == cfg.GetBaseMint() &&
			(tokenTransfer.FromTokenAccount == cfg.GetBaseVault() || tokenTransfer.ToTokenAccount == cfg.GetBaseVault()) {
			isTradeWithExchange = true
		}

		// 当 Mint 为 quote_mint 且 fromTokenAccount 或者 toTokenAccount 为 quote_vault 时，为 true
		if tokenTransfer.Mint == cfg.GetQuoteMint() &&
			(tokenTransfer.FromTokenAccount == cfg.GetQuoteVault() || tokenTransfer.ToTokenAccount == cfg.GetQuoteVault()) {
			isTradeWithExchange = true
		}

		if !isTradeWithExchange {
			continue
		}

		// 创建发送方余额变化记录
		fromAddress := tokenTransfer.FromUserAccount
		// 当 tokenTransfer.fromTokenAccount 为 base_vault 或者 quote_vault，则 Address 为 cfg.PoolAddress
		if tokenTransfer.FromTokenAccount == cfg.GetBaseVault() || tokenTransfer.FromTokenAccount == cfg.GetQuoteVault() {
			fromAddress = cfg.GetPoolAddress()
		}

		fromChange := models.AddressBalanceChange{
			Slot:         slot,
			Timestamp:    timestamp,
			Signature:    signature,
			Address:      fromAddress,
			Mint:         tokenTransfer.Mint,
			AmountChange: tokenTransfer.TokenAmount * -1, // 负值表示转出
		}
		changes = append(changes, fromChange)

		// 创建接收方余额变化记录
		toAddress := tokenTransfer.ToUserAccount
		// 当 tokenTransfer.toTokenAccount 为 base_vault 或者 quote_vault，则 Address 为 cfg.PoolAddress
		if tokenTransfer.ToTokenAccount == cfg.GetBaseVault() || tokenTransfer.ToTokenAccount == cfg.GetQuoteVault() {
			toAddress = cfg.GetPoolAddress()
		}

		toChange := models.AddressBalanceChange{
			Slot:         slot,
			Timestamp:    timestamp,
			Signature:    signature,
			Address:      toAddress,
			Mint:         tokenTransfer.Mint,
			AmountChange: tokenTransfer.TokenAmount, // 正值表示转入
		}
		changes = append(changes, toChange)
	}

	// 处理原生代币(SOL)余额变化
	for _, accountData := range tx.AccountData {
		if accountData.NativeBalanceChange != 0 {
			// 确定地址：如果 accountData.Account 是 base_vault 或 quote_vault，则使用 pool_address
			address := accountData.Account
			if accountData.Account == cfg.GetBaseVault() || accountData.Account == cfg.GetQuoteVault() {
				address = cfg.GetPoolAddress()
			}

			// 创建 SOL 余额变化记录
			nativeChange := models.AddressBalanceChange{
				Slot:         slot,
				Timestamp:    timestamp,
				Signature:    signature,
				Address:      address,
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

// CreateMeteoradbcSwap 创建 Meteoradbc 交换记录，返回 swap 数据数组
func CreateMeteoradbcSwap(db *gorm.DB, tx helius.EnhancedTransaction, cfg MeteoradbcPoolConfig, balanceChanges []models.AddressBalanceChange) ([]*models.MeteoradbcSwap, error) {
	var swaps []*models.MeteoradbcSwap

	// 检查是否已存在相同的 Signature 和 Address 记录（针对 tx.FeePayer）
	var existingSwap models.MeteoradbcSwap
	result := db.Where("signature = ? AND address = ?", tx.Signature, tx.FeePayer).First(&existingSwap)
	if result.Error == nil {
		// 记录已存在，返回错误
		return nil, fmt.Errorf("swap record already exists for signature %s and address %s", tx.Signature, tx.FeePayer)
	}

	// 检查是否是"记录未找到"的错误，如果是则继续创建
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing swap record: %v", result.Error)
	}

	// 检查 tx.FeePayer 是否为空，如果为空则跳过创建主要交换记录
	if tx.FeePayer == "" {
		logrus.Warnf("FeePayer is empty for signature %s, skipping main swap record creation", tx.Signature)
	} else {
		// 创建主要的交换记录（针对 tx.FeePayer），使用传入的余额变化数据
		mainSwap := models.MeteoradbcSwap{
			Slot:              uint(tx.Slot),
			Timestamp:         uint(tx.Timestamp),
			PoolAddress:       cfg.GetPoolAddress(),
			Signature:         tx.Signature,
			Fee:               float64(tx.Fee),
			Address:           tx.FeePayer,
			BaseMint:          cfg.GetBaseMint(),
			QuoteMint:         cfg.GetQuoteMint(),
			TraderBaseChange:  GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, cfg.GetBaseMint()),
			TraderQuoteChange: GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, cfg.GetQuoteMint()),
			TraderSolChange:   GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, "sol"),
			PoolBaseChange:    GetAggregateBalanceChanges(balanceChanges, cfg.GetPoolAddress(), cfg.GetBaseMint()),
			PoolQuoteChange:   GetAggregateBalanceChanges(balanceChanges, cfg.GetPoolAddress(), cfg.GetQuoteMint()),
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
			if err := UpdateWalletTokenStat(db, tx.FeePayer, cfg.GetBaseMint(), mainSwap.TraderBaseChange, 6); err != nil {
				logrus.Warnf("Failed to update wallet token stat for base mint %s, address %s: %v", cfg.GetBaseMint(), tx.FeePayer, err)
			}
		}

		// 更新主要交换记录的钱包代币统计 - quote mint
		if mainSwap.TraderQuoteChange != 0 {
			if err := UpdateWalletTokenStat(db, tx.FeePayer, cfg.GetQuoteMint(), mainSwap.TraderQuoteChange, 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for quote mint %s, address %s: %v", cfg.GetQuoteMint(), tx.FeePayer, err)
			}
		}

		// 更新主要交换记录的钱包代币统计 - sol
		if mainSwap.TraderSolChange != 0 {
			if err := UpdateWalletTokenStat(db, tx.FeePayer, "sol", mainSwap.TraderSolChange/math.Pow10(9), 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for sol, address %s: %v", tx.FeePayer, err)
			}
		}
	}

	// 迭代 balanceChanges，为每个地址（除了 tx.FeePayer）创建额外的交换记录
	processedAddresses := make(map[string]bool)
	processedAddresses[tx.FeePayer] = true          // 标记 tx.FeePayer 已处理
	processedAddresses[cfg.GetPoolAddress()] = true // 标记池子已处理

	for _, change := range balanceChanges {
		if change.Mint != cfg.GetBaseMint() {
			continue
		}

		// 跳过 tx.FeePayer 和已处理的地址
		if processedAddresses[change.Address] {
			continue
		}
		processedAddresses[change.Address] = true

		// 检查是否已存在相同的 Signature 和 Address 记录
		var existingAdditionalSwap models.MeteoradbcSwap
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

		// 检查 change.Address 是否为空，如果为空则跳过创建额外交换记录
		if change.Address == "" {
			logrus.Warnf("Address is empty for signature %s, skipping additional swap record creation", tx.Signature)
			continue
		}

		// 创建额外的交换记录
		additionalSwap := models.MeteoradbcSwap{
			Slot:              uint(tx.Slot),
			Timestamp:         uint(tx.Timestamp),
			PoolAddress:       cfg.GetPoolAddress(),
			Signature:         tx.Signature,
			Fee:               float64(tx.Fee),
			Address:           change.Address,
			BaseMint:          cfg.GetBaseMint(),
			QuoteMint:         cfg.GetQuoteMint(),
			TraderBaseChange:  GetAggregateBalanceChanges(balanceChanges, change.Address, cfg.GetBaseMint()),
			TraderQuoteChange: GetAggregateBalanceChanges(balanceChanges, change.Address, cfg.GetQuoteMint()),
			TraderSolChange:   GetAggregateBalanceChanges(balanceChanges, change.Address, "sol"),
			PoolBaseChange:    0,
			PoolQuoteChange:   0,
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
			if err := UpdateWalletTokenStat(db, change.Address, cfg.GetBaseMint(), additionalSwap.TraderBaseChange, 6); err != nil {
				logrus.Warnf("Failed to update wallet token stat for base mint %s, address %s: %v", cfg.GetBaseMint(), change.Address, err)
			}
		}

		// 更新钱包代币统计 - quote mint
		if additionalSwap.TraderQuoteChange != 0 {
			if err := UpdateWalletTokenStat(db, change.Address, cfg.GetQuoteMint(), additionalSwap.TraderQuoteChange, 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for quote mint %s, address %s: %v", cfg.GetQuoteMint(), change.Address, err)
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

// UpdateMeteoradbcHolder 更新持有者信息 (T+1 logic with individual swap)
func UpdateMeteoradbcHolder(db *gorm.DB, swap *models.MeteoradbcSwap, cfg MeteoradbcPoolConfig) error {
	// 确定 HolderType
	holderType := "retail_investors" // 默认值
	poolAddress := swap.PoolAddress

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
	var traderHolder models.MeteoradbcHolder
	result := db.Where("address = ? AND base_mint = ? AND quote_mint = ?",
		swap.Address, swap.BaseMint, swap.QuoteMint).First(&traderHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new record
			traderHolder = models.MeteoradbcHolder{
				Address:        swap.Address,
				HolderType:     holderType,
				PoolAddress:    poolAddress,
				BaseMint:       swap.BaseMint,
				QuoteMint:      swap.QuoteMint,
				LastSlot:       swap.Slot,
				StartSlot:      swap.Slot,
				LastTimestamp:  swap.Timestamp,
				StartTimestamp: swap.Timestamp,
				EndSignature:   swap.Signature,
				StartSignature: swap.Signature,
				BaseChange:     swap.TraderBaseChange,
				QuoteChange:    swap.TraderQuoteChange,
				SolChange:      swap.TraderSolChange,
				TxCount:        1,
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
		traderHolder.TxCount++

		// Update holder type if it changed
		traderHolder.HolderType = holderType

		if err := db.Save(&traderHolder).Error; err != nil {
			return fmt.Errorf("error updating trader holder record: %v", err)
		}
	}

	// Update pool holder record
	var poolHolder models.MeteoradbcHolder
	result = db.Where("address = ? AND pool_address = ? AND base_mint = ? AND quote_mint = ?",
		cfg.GetPoolAddress(), cfg.GetPoolAddress(), swap.BaseMint, swap.QuoteMint).First(&poolHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new pool holder record
			poolHolder = models.MeteoradbcHolder{
				Address:        cfg.GetPoolAddress(),
				HolderType:     "pool",
				PoolAddress:    cfg.GetPoolAddress(),
				BaseMint:       swap.BaseMint,
				QuoteMint:      swap.QuoteMint,
				LastSlot:       swap.Slot,
				StartSlot:      swap.Slot,
				LastTimestamp:  swap.Timestamp,
				StartTimestamp: swap.Timestamp,
				EndSignature:   swap.Signature,
				StartSignature: swap.Signature,
				BaseChange:     swap.PoolBaseChange,
				QuoteChange:    swap.PoolQuoteChange,
				SolChange:      0, // Meteoradbc pools don't have direct SOL changes like PumpFun
				TxCount:        1,
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

		// Aggregate pool changes
		poolHolder.BaseChange += swap.PoolBaseChange
		poolHolder.QuoteChange += swap.PoolQuoteChange
		poolHolder.TxCount++
		// poolHolder.SolChange remains 0 for Meteoradbc pools

		if err := db.Save(&poolHolder).Error; err != nil {
			return fmt.Errorf("error updating pool holder record: %v", err)
		}
	}

	return nil
}

// ProcessMeteoradbcConfig processes transactions for a single Meteoradbc pool configuration
func ProcessMeteoradbcConfig(ctx context.Context, db *gorm.DB, heliusClient *helius.Client, cfg MeteoradbcPoolConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get or create monitor config
	monitorConfig, err := GetOrCreateTransactionsMonitorConfig(db, cfg.GetPoolAddress())
	if err != nil {
		logrus.Errorf("Failed to get monitor config for %s: %v", cfg.GetPoolAddress(), err)
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
			logrus.Warnf("Context cancelled during saveConfig for address: %s", cfg.GetPoolAddress())
			return
		}

		// Update LastExecution if API was called
		if hasCalledAPI {
			monitorConfig.LastExecution = uint(time.Now().Unix())
		}

		// Save config
		if err := db.Save(monitorConfig).Error; err != nil {
			logrus.Errorf("Failed to update monitor config for %s: %v", cfg.GetPoolAddress(), err)
		}
	}

mainLoop:
	for {
		select {
		case <-ctx.Done():
			logrus.Warnf("Context cancelled for address: %s, reason: %v", cfg.GetPoolAddress(), ctx.Err())
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
			logrus.Printf("GetEnhancedTransactionsByAddress for address: %s, page: %d", cfg.GetPoolAddress(), currentPage)

			// Create timeout context
			apiCtx, apiCancel := context.WithTimeout(ctx, 30*time.Second)
			defer apiCancel()

			// Check if context is cancelled
			if apiCtx.Err() != nil {
				break mainLoop
			}

			transactions, err := heliusClient.GetEnhancedTransactionsByAddress(cfg.GetPoolAddress(), &opts)

			hasCalledAPI = true

			if err != nil {
				if ctx.Err() != nil {
					break mainLoop
				}
				logrus.Errorf("Failed to get transactions for %s: %v", cfg.GetPoolAddress(), err)

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

			// Sort by slot ascending before processing
			transactions = SortTransactions("slot", "asc", transactions)

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
					Address:   cfg.GetPoolAddress(),
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

				// 获取余额变化数据（不写入数据库），支持迁移检测
				balanceChanges, isMigrated, err := CreateAddressBalanceChangeWithMigration(db, tx, cfg)
				if err != nil {
					logrus.Errorf("Failed to create balance changes for %s: %v", tx.Signature, err)
					continue
				}

				// Create Meteoradbc pool swap record using the balance changes from memory
				swaps, err := CreateMeteoradbcSwap(db, tx, cfg, balanceChanges)
				if err != nil {
					logrus.Errorf("Failed to create swap record %s: %v", tx.Signature, err)
					continue
				}

				// 实时更新持有者信息 (T+1 logic)
				for _, swap := range swaps {
					if err := UpdateMeteoradbcHolder(db, swap, cfg); err != nil {
						logrus.Errorf("Failed to update holder information %s: %v", tx.Signature, err)
					}
				}

				// 如果检测到迁移，需要额外处理 Meteoracpmm 交易
				if isMigrated {
					// 获取对应的 MeteoracpmmConfig
					if meteoradbcCfg, ok := cfg.(MeteoradbcConfig); ok && meteoradbcCfg.DammV2PoolAddress != "" {
						// 如果 IsMigrated 标志还未设置，则执行 Holder 迁移
						if !meteoradbcCfg.IsMigrated {
							if err := MigrateHolder(db, meteoradbcCfg.MeteoradbcConfig); err != nil {
								logrus.Errorf("Failed to migrate holders for MeteoradbcConfig %s: %v", meteoradbcCfg.PoolAddress, err)
							} else {
								logrus.Infof("Successfully migrated holders for MeteoradbcConfig %s to MeteoracpmmConfig %s",
									meteoradbcCfg.PoolAddress, meteoradbcCfg.DammV2PoolAddress)
							}
						}

						var cpmmConfig models.MeteoracpmmConfig
						if err := db.Where("pool_address = ?", meteoradbcCfg.DammV2PoolAddress).First(&cpmmConfig).Error; err == nil {
							wrappedCpmmCfg := MeteoracpmmConfig{cpmmConfig}
							// 获取或创建 Meteoracpmm 的监控配置
							cpmmMonitorConfig, err := GetOrCreateTransactionsMonitorConfig(db, wrappedCpmmCfg.GetPoolAddress())
							if err != nil {
								logrus.Errorf("Failed to get monitor config for Meteoracpmm %s: %v", wrappedCpmmCfg.GetPoolAddress(), err)
							} else {
								// 使用 MeteoracpmmTxHandler 处理交易
								if err := MeteoracpmmTxHandler(db, tx, wrappedCpmmCfg, cpmmMonitorConfig); err != nil {
									logrus.Errorf("Failed to handle Meteoracpmm transaction %s: %v", tx.Signature, err)
								}
								// 保存 Meteoracpmm 监控配置
								if err := db.Save(cpmmMonitorConfig).Error; err != nil {
									logrus.Errorf("Failed to save Meteoracpmm monitor config: %v", err)
								}
							}
						}

						// 更新 MeteoradbcConfig 的 IsMigrated 标志
						meteoradbcCfg.IsMigrated = true
						if err := db.Save(&meteoradbcCfg.MeteoradbcConfig).Error; err != nil {
							logrus.Errorf("Failed to update MeteoradbcConfig IsMigrated flag: %v", err)
						} else {
							// 更新 IsMigrated 标志成功后，调用 ModifyProjectConfig
							if err := ModifyProjectConfig(meteoradbcCfg.MeteoradbcConfig); err != nil {
								logrus.Errorf("Failed to modify ProjectConfig for MeteoradbcConfig %s: %v", meteoradbcCfg.PoolAddress, err)
							}
						}
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
	logrus.Printf("Completed ProcessMeteoradbcConfig for address: %s", cfg.GetPoolAddress())
}

// UpdateMeteoradbcTransactions 更新 Meteoradbc 交易数据
func UpdateMeteoradbcTransactions(ctx context.Context) error {
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

	// Get the latest active MeteoradbcConfig (highest ID)
	var meteoradbcConfigs []models.MeteoradbcConfig
	if err := db.Where("status = ?", "active").Order("id DESC").Limit(1).Find(&meteoradbcConfigs).Error; err != nil {
		return err
	}

	// Use worker pool to process configurations
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	// Process Meteoradbc configs
	for _, config := range meteoradbcConfigs {
		// Try to acquire address lock
		if _, loaded := addressLocks.LoadOrStore(config.PoolAddress, true); loaded {
			logrus.Infof("Address %s is already being processed, skipping", config.PoolAddress)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(cfg models.MeteoradbcConfig) {
			defer func() {
				<-semaphore                          // Release semaphore
				addressLocks.Delete(cfg.PoolAddress) // Release address lock
			}()
			// 使用包装类型
			wrappedCfg := MeteoradbcConfig{cfg}
			ProcessMeteoradbcConfig(ctx, db, heliusClient, wrappedCfg, &wg)
		}(config)
	}

	wg.Wait()
	return nil
}

// CreateMeteoracpmmSwap 创建 Meteoracpmm 交换记录，返回 swap 数据数组
func CreateMeteoracpmmSwap(db *gorm.DB, tx helius.EnhancedTransaction, cfg MeteoradbcPoolConfig, balanceChanges []models.AddressBalanceChange) ([]*models.MeteoracpmmSwap, error) {
	var swaps []*models.MeteoracpmmSwap

	// 检查是否已存在相同的 Signature 和 Address 记录（针对 tx.FeePayer）
	var existingSwap models.MeteoracpmmSwap
	result := db.Where("signature = ? AND address = ?", tx.Signature, tx.FeePayer).First(&existingSwap)
	if result.Error == nil {
		// 记录已存在，返回错误
		return nil, fmt.Errorf("swap record already exists for signature %s and address %s", tx.Signature, tx.FeePayer)
	}

	// 检查是否是"记录未找到"的错误，如果是则继续创建
	if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing swap record: %v", result.Error)
	}

	// 检查 tx.FeePayer 是否为空，如果为空则跳过创建主要交换记录
	if tx.FeePayer == "" {
		logrus.Warnf("FeePayer is empty for signature %s, skipping main swap record creation", tx.Signature)
	} else {
		// 创建主要的交换记录（针对 tx.FeePayer），使用传入的余额变化数据
		mainSwap := models.MeteoracpmmSwap{
			Slot:              uint(tx.Slot),
			Timestamp:         uint(tx.Timestamp),
			PoolAddress:       cfg.GetPoolAddress(),
			Signature:         tx.Signature,
			Fee:               float64(tx.Fee),
			Address:           tx.FeePayer,
			BaseMint:          cfg.GetBaseMint(),
			QuoteMint:         cfg.GetQuoteMint(),
			TraderBaseChange:  GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, cfg.GetBaseMint()),
			TraderQuoteChange: GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, cfg.GetQuoteMint()),
			TraderSolChange:   GetAggregateBalanceChanges(balanceChanges, tx.FeePayer, "sol"),
			PoolBaseChange:    GetAggregateBalanceChanges(balanceChanges, cfg.GetPoolAddress(), cfg.GetBaseMint()),
			PoolQuoteChange:   GetAggregateBalanceChanges(balanceChanges, cfg.GetPoolAddress(), cfg.GetQuoteMint()),
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
			if err := UpdateWalletTokenStat(db, tx.FeePayer, cfg.GetBaseMint(), mainSwap.TraderBaseChange, 6); err != nil {
				logrus.Warnf("Failed to update wallet token stat for base mint %s, address %s: %v", cfg.GetBaseMint(), tx.FeePayer, err)
			}
		}

		// 更新主要交换记录的钱包代币统计 - quote mint
		if mainSwap.TraderQuoteChange != 0 {
			if err := UpdateWalletTokenStat(db, tx.FeePayer, cfg.GetQuoteMint(), mainSwap.TraderQuoteChange, 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for quote mint %s, address %s: %v", cfg.GetQuoteMint(), tx.FeePayer, err)
			}
		}

		// 更新主要交换记录的钱包代币统计 - sol
		if mainSwap.TraderSolChange != 0 {
			if err := UpdateWalletTokenStat(db, tx.FeePayer, "sol", mainSwap.TraderSolChange/math.Pow10(9), 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for sol, address %s: %v", tx.FeePayer, err)
			}
		}
	}

	// 迭代 balanceChanges，为每个地址（除了 tx.FeePayer）创建额外的交换记录
	processedAddresses := make(map[string]bool)
	processedAddresses[tx.FeePayer] = true          // 标记 tx.FeePayer 已处理
	processedAddresses[cfg.GetPoolAddress()] = true // 标记池子已处理

	for _, change := range balanceChanges {
		if change.Mint != cfg.GetBaseMint() {
			continue
		}

		// 跳过 tx.FeePayer 和已处理的地址
		if processedAddresses[change.Address] {
			continue
		}
		processedAddresses[change.Address] = true

		// 检查是否已存在相同的 Signature 和 Address 记录
		var existingAdditionalSwap models.MeteoracpmmSwap
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

		// 检查 change.Address 是否为空，如果为空则跳过创建额外交换记录
		if change.Address == "" {
			logrus.Warnf("Address is empty for signature %s, skipping additional swap record creation", tx.Signature)
			continue
		}

		// 创建额外的交换记录
		additionalSwap := models.MeteoracpmmSwap{
			Slot:              uint(tx.Slot),
			Timestamp:         uint(tx.Timestamp),
			PoolAddress:       cfg.GetPoolAddress(),
			Signature:         tx.Signature,
			Fee:               float64(tx.Fee),
			Address:           change.Address,
			BaseMint:          cfg.GetBaseMint(),
			QuoteMint:         cfg.GetQuoteMint(),
			TraderBaseChange:  GetAggregateBalanceChanges(balanceChanges, change.Address, cfg.GetBaseMint()),
			TraderQuoteChange: GetAggregateBalanceChanges(balanceChanges, change.Address, cfg.GetQuoteMint()),
			TraderSolChange:   GetAggregateBalanceChanges(balanceChanges, change.Address, "sol"),
			PoolBaseChange:    0,
			PoolQuoteChange:   0,
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
			if err := UpdateWalletTokenStat(db, change.Address, cfg.GetBaseMint(), additionalSwap.TraderBaseChange, 6); err != nil {
				logrus.Warnf("Failed to update wallet token stat for base mint %s, address %s: %v", cfg.GetBaseMint(), change.Address, err)
			}
		}

		// 更新钱包代币统计 - quote mint
		if additionalSwap.TraderQuoteChange != 0 {
			if err := UpdateWalletTokenStat(db, change.Address, cfg.GetQuoteMint(), additionalSwap.TraderQuoteChange, 9); err != nil {
				logrus.Warnf("Failed to update wallet token stat for quote mint %s, address %s: %v", cfg.GetQuoteMint(), change.Address, err)
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

// UpdateMeteoracpmmHolder 更新持有者信息 (T+1 logic with individual swap)
func UpdateMeteoracpmmHolder(db *gorm.DB, swap *models.MeteoracpmmSwap, cfg MeteoradbcPoolConfig) error {
	// 确定 HolderType
	holderType := "retail_investors" // 默认值
	poolAddress := swap.PoolAddress

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
	var traderHolder models.MeteoracpmmHolder
	result := db.Where("address = ? AND base_mint = ? AND quote_mint = ?",
		swap.Address, swap.BaseMint, swap.QuoteMint).First(&traderHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new record
			traderHolder = models.MeteoracpmmHolder{
				Address:        swap.Address,
				HolderType:     holderType,
				PoolAddress:    poolAddress,
				BaseMint:       swap.BaseMint,
				QuoteMint:      swap.QuoteMint,
				LastSlot:       swap.Slot,
				StartSlot:      swap.Slot,
				LastTimestamp:  swap.Timestamp,
				StartTimestamp: swap.Timestamp,
				EndSignature:   swap.Signature,
				StartSignature: swap.Signature,
				BaseChange:     swap.TraderBaseChange,
				QuoteChange:    swap.TraderQuoteChange,
				SolChange:      swap.TraderSolChange,
				TxCount:        1,
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
		traderHolder.TxCount++

		// Update holder type if it changed
		traderHolder.HolderType = holderType

		if err := db.Save(&traderHolder).Error; err != nil {
			return fmt.Errorf("error updating trader holder record: %v", err)
		}
	}

	// Update pool holder record
	var poolHolder models.MeteoracpmmHolder
	result = db.Where("address = ? AND pool_address = ? AND base_mint = ? AND quote_mint = ?",
		cfg.GetPoolAddress(), cfg.GetPoolAddress(), swap.BaseMint, swap.QuoteMint).First(&poolHolder)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new pool holder record
			poolHolder = models.MeteoracpmmHolder{
				Address:        cfg.GetPoolAddress(),
				HolderType:     "pool",
				PoolAddress:    cfg.GetPoolAddress(),
				BaseMint:       swap.BaseMint,
				QuoteMint:      swap.QuoteMint,
				LastSlot:       swap.Slot,
				StartSlot:      swap.Slot,
				LastTimestamp:  swap.Timestamp,
				StartTimestamp: swap.Timestamp,
				EndSignature:   swap.Signature,
				StartSignature: swap.Signature,
				BaseChange:     swap.PoolBaseChange,
				QuoteChange:    swap.PoolQuoteChange,
				SolChange:      0, // Meteoracpmm pools don't have direct SOL changes like PumpFun
				TxCount:        1,
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

		// Aggregate pool changes
		poolHolder.BaseChange += swap.PoolBaseChange
		poolHolder.QuoteChange += swap.PoolQuoteChange
		poolHolder.TxCount++
		// poolHolder.SolChange remains 0 for Meteoracpmm pools

		if err := db.Save(&poolHolder).Error; err != nil {
			return fmt.Errorf("error updating pool holder record: %v", err)
		}
	}

	return nil
}

// MigrateHolder 将 MeteoradbcHolder 的数据复制到 MeteoracpmmHolder
// 不包括 HolderType 为 "pool" 的数据，并将 PoolAddress 改为 DammV2PoolAddress
func MigrateHolder(db *gorm.DB, meteoradbcConfig models.MeteoradbcConfig) error {
	if meteoradbcConfig.DammV2PoolAddress == "" {
		return fmt.Errorf("DammV2PoolAddress is empty, cannot migrate")
	}

	// 查询所有 MeteoradbcHolder，排除 HolderType 为 "pool" 的数据
	var meteoradbcHolders []models.MeteoradbcHolder
	if err := db.Where("pool_address = ? AND holder_type != ?", meteoradbcConfig.PoolAddress, "pool").Find(&meteoradbcHolders).Error; err != nil {
		return fmt.Errorf("failed to query MeteoradbcHolder: %v", err)
	}

	// 批量复制数据到 MeteoracpmmHolder
	for _, dbcHolder := range meteoradbcHolders {
		// 检查是否已存在相同的 MeteoracpmmHolder 记录
		var existingCpmmHolder models.MeteoracpmmHolder
		result := db.Where("address = ? AND pool_address = ? AND base_mint = ? AND quote_mint = ?",
			dbcHolder.Address, meteoradbcConfig.DammV2PoolAddress, dbcHolder.BaseMint, dbcHolder.QuoteMint).First(&existingCpmmHolder)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// 创建新的 MeteoracpmmHolder 记录
				cpmmHolder := models.MeteoracpmmHolder{
					Address:        dbcHolder.Address,
					HolderType:     dbcHolder.HolderType,
					PoolAddress:    meteoradbcConfig.DammV2PoolAddress, // 使用 DammV2PoolAddress
					BaseMint:       dbcHolder.BaseMint,
					QuoteMint:      dbcHolder.QuoteMint,
					LastSlot:       dbcHolder.LastSlot,
					StartSlot:      dbcHolder.StartSlot,
					LastTimestamp:  dbcHolder.LastTimestamp,
					StartTimestamp: dbcHolder.StartTimestamp,
					EndSignature:   dbcHolder.EndSignature,
					StartSignature: dbcHolder.StartSignature,
					BaseChange:     dbcHolder.BaseChange,
					QuoteChange:    dbcHolder.QuoteChange,
					SolChange:      dbcHolder.SolChange,
					TxCount:        dbcHolder.TxCount,
				}
				if err := db.Create(&cpmmHolder).Error; err != nil {
					logrus.Errorf("Failed to create MeteoracpmmHolder for address %s: %v", dbcHolder.Address, err)
					continue
				}
				logrus.Infof("Migrated MeteoradbcHolder to MeteoracpmmHolder: address=%s, pool_address=%s -> %s",
					dbcHolder.Address, dbcHolder.PoolAddress, meteoradbcConfig.DammV2PoolAddress)
			} else {
				logrus.Errorf("Failed to check existing MeteoracpmmHolder for address %s: %v", dbcHolder.Address, result.Error)
				continue
			}
		} else {
			// 记录已存在，可以选择跳过或更新
			logrus.Infof("MeteoracpmmHolder already exists for address %s, pool_address %s, skipping migration",
				dbcHolder.Address, meteoradbcConfig.DammV2PoolAddress)
		}
	}

	logrus.Infof("Migration completed: migrated %d MeteoradbcHolder records to MeteoracpmmHolder", len(meteoradbcHolders))
	return nil
}

// ModifyProjectConfig 修改 ProjectConfig，将 meteora_dbc 项目迁移到 meteora_cpmm
func ModifyProjectConfig(cfg models.MeteoradbcConfig) error {
	// 查询 PoolPlatform 为 meteora_dbc 且 pool_id 为 cfg.ID 的项目
	var projectConfigs []models.ProjectConfig
	if err := config.DB.Where("pool_platform = ? AND pool_id = ?", "meteora_dbc", cfg.ID).Find(&projectConfigs).Error; err != nil {
		return fmt.Errorf("failed to query ProjectConfig: %v", err)
	}

	if len(projectConfigs) == 0 {
		logrus.Infof("No ProjectConfig found for MeteoradbcConfig ID %d, skipping modification", cfg.ID)
		return nil
	}

	// 查询对应 DbcPoolAddress 为 cfg.PoolAddress 的 MeteoracpmmConfig
	var meteoracpmmConfig models.MeteoracpmmConfig
	if err := config.DB.Where("dbc_pool_address = ?", cfg.PoolAddress).First(&meteoracpmmConfig).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("MeteoracpmmConfig not found for DbcPoolAddress %s", cfg.PoolAddress)
		}
		return fmt.Errorf("failed to query MeteoracpmmConfig: %v", err)
	}

	// 更新所有匹配的 ProjectConfig
	for _, projectConfig := range projectConfigs {
		projectConfig.PoolPlatform = "meteora_cpmm"
		projectConfig.PoolID = meteoracpmmConfig.ID
		if err := config.DB.Save(&projectConfig).Error; err != nil {
			logrus.Errorf("Failed to update ProjectConfig ID %d: %v", projectConfig.ID, err)
			continue
		}
		logrus.Infof("Updated ProjectConfig ID %d: PoolPlatform changed to meteora_cpmm, PoolID changed from %d to %d",
			projectConfig.ID, cfg.ID, meteoracpmmConfig.ID)
	}

	// 更新 MeteoracpmmConfig 的 Status 为 active
	meteoracpmmConfig.Status = "active"
	if err := config.DB.Save(&meteoracpmmConfig).Error; err != nil {
		logrus.Errorf("Failed to update MeteoracpmmConfig Status: %v", err)
		return fmt.Errorf("failed to update MeteoracpmmConfig Status: %v", err)
	}
	logrus.Infof("Updated MeteoracpmmConfig ID %d: Status changed to active", meteoracpmmConfig.ID)

	logrus.Infof("ModifyProjectConfig completed: updated %d ProjectConfig records and MeteoracpmmConfig Status", len(projectConfigs))
	return nil
}

// ProcessMeteoracpmmConfig processes transactions for a single Meteoracpmm pool configuration
func ProcessMeteoracpmmConfig(ctx context.Context, db *gorm.DB, heliusClient *helius.Client, cfg MeteoradbcPoolConfig, wg *sync.WaitGroup) {
	defer wg.Done()

	// Get or create monitor config
	monitorConfig, err := GetOrCreateTransactionsMonitorConfig(db, cfg.GetPoolAddress())
	if err != nil {
		logrus.Errorf("Failed to get monitor config for %s: %v", cfg.GetPoolAddress(), err)
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
			logrus.Warnf("Context cancelled during saveConfig for address: %s", cfg.GetPoolAddress())
			return
		}

		// Update LastExecution if API was called
		if hasCalledAPI {
			monitorConfig.LastExecution = uint(time.Now().Unix())
		}

		// Save config
		if err := db.Save(monitorConfig).Error; err != nil {
			logrus.Errorf("Failed to update monitor config for %s: %v", cfg.GetPoolAddress(), err)
		}
	}

mainLoop:
	for {
		select {
		case <-ctx.Done():
			logrus.Warnf("Context cancelled for address: %s, reason: %v", cfg.GetPoolAddress(), ctx.Err())
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
			logrus.Printf("GetEnhancedTransactionsByAddress for address: %s, page: %d", cfg.GetPoolAddress(), currentPage)

			// Create timeout context
			apiCtx, apiCancel := context.WithTimeout(ctx, 30*time.Second)
			defer apiCancel()

			// Check if context is cancelled
			if apiCtx.Err() != nil {
				break mainLoop
			}

			transactions, err := heliusClient.GetEnhancedTransactionsByAddress(cfg.GetPoolAddress(), &opts)

			hasCalledAPI = true

			if err != nil {
				if ctx.Err() != nil {
					break mainLoop
				}
				logrus.Errorf("Failed to get transactions for %s: %v", cfg.GetPoolAddress(), err)

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
					Address:   cfg.GetPoolAddress(),
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

				// 使用 MeteoracpmmTxHandler 处理交易
				if err := MeteoracpmmTxHandler(db, tx, cfg, monitorConfig); err != nil {
					logrus.Errorf("Failed to handle transaction %s: %v", tx.Signature, err)
					continue
				}
			}

			// Save config
			saveConfig()
		}
	}

	// Final config save
	saveConfig()
	logrus.Printf("Completed ProcessMeteoracpmmConfig for address: %s", cfg.GetPoolAddress())
}

// UpdateMeteoracpmmTransactions 更新 Meteoracpmm 交易数据
func UpdateMeteoracpmmTransactions(ctx context.Context) error {
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

	// Get the latest active MeteoracpmmConfig (highest ID)
	var meteoracpmmConfigs []models.MeteoracpmmConfig
	if err := db.Where("status = ?", "active").Order("id DESC").Limit(1).Find(&meteoracpmmConfigs).Error; err != nil {
		return err
	}

	// Use worker pool to process configurations
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)

	// Process Meteoracpmm configs
	for _, config := range meteoracpmmConfigs {
		// Try to acquire address lock
		if _, loaded := addressLocks.LoadOrStore(config.PoolAddress, true); loaded {
			logrus.Infof("Address %s is already being processed, skipping", config.PoolAddress)
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(cfg models.MeteoracpmmConfig) {
			defer func() {
				<-semaphore                          // Release semaphore
				addressLocks.Delete(cfg.PoolAddress) // Release address lock
			}()
			// 使用包装类型
			wrappedCfg := MeteoracpmmConfig{cfg}
			ProcessMeteoracpmmConfig(ctx, db, heliusClient, wrappedCfg, &wg)
		}(config)
	}

	wg.Wait()
	return nil
}
