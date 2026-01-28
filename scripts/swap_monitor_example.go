//go:build ignore

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"marketcontrol/pkg/solana/meteora"

	log "github.com/sirupsen/logrus"
)

func main() {
	// 配置日志格式
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	log.SetLevel(log.InfoLevel)

	// 配置参数
	rpcEndpoint := getEnv("SOLANA_RPC_URL", "https://red-wider-scion.solana-mainnet.quiknode.pro/7d63bea9a0a2d0a3664671d551a2d3565bef43b6/")
	// wsEndpoint := "wss://red-wider-scion.solana-mainnet.quiknode.pro/7d63bea9a0a2d0a3664671d551a2d3565bef43b6/"
	wsEndpoint := "wss://red-wider-scion.solana-mainnet.quiknode.pro/7d63bea9a0a2d0a3664671d551a2d3565bef43b6/"

	// 检查 SOLANA_WS_URL 是否设置
	if wsEndpoint == "" {
		log.Fatal("❌ SOLANA_WS_URL 环境变量未设置")
	}

	// 池子地址
	poolAddress := getEnv("POOL_ADDRESS", "4h4zwhCgLRdiAcd2fw1viPeCR8AKxaC9G1MHbpLnSoYX")

	// 基础代币地址
	baseTokenAddress := getEnv("BASE_TOKEN_ADDRESS", "7iTEa7P9GnmQdyDiztUVHu4StndS9hUmYmJbeXvXUsjP")

	// 报价代币地址（SOL）
	quoteTokenAddress := getEnv("QUOTE_TOKEN_ADDRESS", "So11111111111111111111111111111111111111112")

	// Authority 地址（可选，用于过滤特定 owner 的代币余额变化）
	// 内盘: "FhVo3mqL8PW5pH5U2CN4XE33DokiyZnUwuGpH2hmHLuM"
	// 外盘: "HLnpSz9h2S4hiLQ43rnSD9XkcUThA7B8hQMKmDaiTLcC"
	authority := getEnv("AUTHORITY", "HLnpSz9h2S4hiLQ43rnSD9XkcUThA7B8hQMKmDaiTLcC")

	log.Info("🚀 初始化 Swap Monitor...")
	log.Infof("📍 池子地址: %s", poolAddress)
	log.Infof("🪙 基础代币: %s", baseTokenAddress)
	log.Infof("💵 报价代币: %s", quoteTokenAddress)
	if authority != "" {
		log.Infof("🔐 Authority: %s", authority)
	}
	log.Info("")

	// 设置 RPC endpoint 环境变量（如果未设置）
	if os.Getenv("DEFAULT_SOLANA_RPC") == "" {
		os.Setenv("DEFAULT_SOLANA_RPC", rpcEndpoint)
	}

	// 创建 PoolMonitorManager 实例
	manager, err := meteora.NewPoolMonitorManager()
	if err != nil {
		log.Fatalf("❌ 创建 PoolMonitorManager 失败: %v", err)
	}

	// 定义处理 Swap 交易的回调函数
	handleSwapTransaction := func(swap *meteora.SwapTransaction) {
		log.Info("\n═══════════════════════════════════════")
		log.Infof("[%s] 🔄 Swap 交易检测到", time.Unix(swap.Timestamp/1000, 0).Format(time.RFC3339))
		log.Info("─────────────────────────────────────")
		log.Infof("📝 交易签名: %s", swap.Signature)
		log.Infof("🎯 交易类型: %s", swap.Action)
		log.Infof("🎯 交易代币: %s", swap.BaseToken.Address)
		log.Infof("🪙 基础代币: %f %s", swap.BaseToken.Amount, swap.BaseToken.Symbol)
		log.Infof("💵 报价代币: %f %s", swap.QuoteToken.Amount, swap.QuoteToken.Symbol)
		log.Infof("💰 交易价值: %.6f %s", swap.Value, swap.QuoteToken.Symbol)
		log.Infof("💳 支付者 (Payer): %s", swap.Payer)
		if len(swap.Signers) > 1 {
			log.Infof("✍️  所有签名者 (%d):", len(swap.Signers))
			for index, signer := range swap.Signers {
				log.Infof("   [%d] %s", index+1, signer)
			}
		}
		log.Info("─────────────────────────────────────")
		log.Info("═══════════════════════════════════════\n")

		// 这里可以添加你的业务逻辑
		// 例如：发送 webhook、存储到数据库、触发其他操作等
		// sendWebhook(swap)
		// saveToDatabase(swap)
	}

	// 启动监控
	err = manager.StartMonitoring(
		poolAddress,
		baseTokenAddress,
		quoteTokenAddress,
		authority,
		handleSwapTransaction,
	)
	if err != nil {
		log.Fatalf("❌ 启动监控失败: %v", err)
	}

	log.Info("✅ Swap 交易监控已启动")
	log.Info("👂 正在监听新的 Swap 交易...")
	log.Info("按 Ctrl+C 停止监控\n")

	// 优雅关闭处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 等待信号
	<-sigChan

	log.Info("\n\n🛑 正在停止监控...")
	err = manager.StopMonitoring(poolAddress)
	if err != nil {
		log.Errorf("❌ 停止监控时出错: %v", err)
		os.Exit(1)
	}

	log.Info("✅ 监控已停止")
	os.Exit(0)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
