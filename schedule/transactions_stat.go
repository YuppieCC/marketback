package main

import (
	"context"
	"os"
	"time"

	"marketcontrol/pkg/config"
	"marketcontrol/pkg/solana/meteora"

	"github.com/sirupsen/logrus"
)

func main() {
	// 创建日志目录
	os.MkdirAll("logs", 0755)
	file, err := os.OpenFile("logs/update_transactions_schedule.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	// logrus.SetLevel(logrus.InfoLevel)
	logrus.SetLevel(logrus.ErrorLevel)

	if err == nil {
		logrus.SetOutput(file)
	} else {
		logrus.Warn("无法打开日志文件，日志将输出到标准输出")
	}

	// 初始化数据库
	config.InitDB()
	if config.DB == nil {
		logrus.Fatal("Failed to initialize database")
		return
	}

	// 初始化 Helius API 客户端
	heliusAPIKey := os.Getenv("HELIUS_API_KEY")
	if heliusAPIKey == "" {
		logrus.Fatal("HELIUS_API_KEY environment variable is not set")
		return
	}

	logrus.Info("> 初始化程序完成")

	for {
		// 等待一段时间后继续下一轮更新
		time.Sleep(2 * time.Second)

		// 创建带超时的 context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

		// // 执行 pumpfun 内盘交易更新
		// if err := pumpfun.UpdatePumpfunInternalTransactions(ctx); err != nil {
		// 	logrus.Errorf("Error updating internal transactions: %v", err)
		// }

		// // 执行 pumpfunamm交易更新
		// if err := pumpfun.UpdatePumpfunAmmPoolTransactions(ctx); err != nil {
		// 	logrus.Errorf("Error updating AMM pool transactions: %v", err)
		// }

		// // 执行 raydium 交易更新
		// if err := raydium.UpdateRaydiumPoolTransactions(ctx); err != nil {
		// 	logrus.Errorf("Error updating Raydium pool transactions: %v", err)
		// }

		// 执行 meteoradbc 交易更新
		if err := meteora.UpdateMeteoradbcTransactions(ctx); err != nil {
			logrus.Errorf("Error updating Meteoradbc transactions: %v", err)
		}

		// 执行 meteoraacpmm 交易更新
		if err := meteora.UpdateMeteoracpmmTransactions(ctx); err != nil {
			logrus.Errorf("Error updating Meteoracpmm transactions: %v", err)
		}

		// 取消 context 避免资源泄露
		cancel()
	}
}
