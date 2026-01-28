package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"marketcontrol/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB() {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(50)           // 设置空闲连接池中的最大连接数
	sqlDB.SetMaxOpenConns(200)          // 设置打开数据库连接的最大数量
	sqlDB.SetConnMaxLifetime(time.Hour) // 设置连接可复用的最大时间

	DB = db

	// Auto migrate all models
	err = DB.AutoMigrate(
		&models.BlockchainConfig{},
		&models.RpcConfig{},
		&models.AddressManage{},
		&models.DisposableAddressManage{},
		&models.WashMap{},
		&models.AddressNode{},
		&models.AddressEdge{},
		&models.WashTask{},
		&models.WashTaskManage{},
		&models.StrategyConfig{},
		&models.StrategySignal{},
		&models.StrategyTransaction{},
		&models.RoleConfig{},
		&models.RoleAddress{},
		&models.ProjectConfig{},
		&models.ProjectFundTransferRecord{},
		&models.PoolConfig{},
		&models.TokenConfig{},
		&models.PumpfuninternalConfig{},
		&models.WalletTokenStat{},
		&models.PoolStat{},
		&models.PumpfuninternalStat{},
		&models.WalletTokenSnapshot{},
		&models.PoolSnapshot{},
		&models.PumpfuninternalSnapshot{},
		&models.TokenAccount{},
		&models.TokenMetadata{},
		&models.TransactionsMonitorConfig{},
		&models.AddressTransaction{},
		&models.AddressBalanceChange{},
		&models.PumpfuninternalSwap{},
		&models.PumpfuninternalHolder{},
		&models.PumpfunAmmPoolConfig{},
		&models.PumpfunAmmPoolSwap{},
		&models.PumpfunAmmpoolHolder{},
		&models.PumpfunAmmPoolStat{},
		&models.TemplateRoleConfig{},
		&models.TemplateRoleAddress{},
		&models.ProjectSettleRecord{},
		&models.RoleConfigRelation{},
		&models.ProjectExtraAddress{},
		&models.RaydiumLaunchpadPoolConfig{},
		&models.RaydiumCpmmPoolConfig{},
		&models.RaydiumLaunchpadPoolStat{},
		&models.RaydiumCpmmPoolStat{},
		&models.RaydiumPoolHolder{},
		&models.RaydiumPoolSwap{},
		&models.RaydiumPoolRelation{},
		&models.AddressConfig{},
		&models.MeteoradbcConfig{},
		&models.MeteoradbcHolder{},
		&models.MeteoradbcSwap{},
		&models.MeteoradbcPoolStat{},
		&models.MeteoracpmmConfig{},
		&models.MeteoracpmmHolder{},
		&models.MeteoracpmmSwap{},
		&models.MeteoracpmmPoolStat{},
		&models.SystemLog{},
		&models.SwapTransaction{},
		&models.SystemParams{},
		&models.SystemCommand{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}
