package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/solana"
)

const (
	// CHECK_INTERVAL 检查间隔时间
	CHECK_INTERVAL = 3 * time.Second
	// TIME_WINDOW 时间窗口
	TIME_WINDOW = 3 * time.Minute
)

// RunStrategyTransactionStatusCheck 运行策略交易状态检查
func RunStrategyTransactionStatusCheck() {
	log.Printf("Starting strategy transaction status check scheduler...")

	ticker := time.NewTicker(CHECK_INTERVAL)
	defer ticker.Stop()

	for range ticker.C {
		if err := checkAndUpdateTransactionStatus(); err != nil {
			log.Printf("Error checking transaction status: %v", err)
		}
	}
}

// checkAndUpdateTransactionStatus 检查并更新交易状态
func checkAndUpdateTransactionStatus() error {
	// 计算时间窗口
	checkTime := time.Now().Add(-TIME_WINDOW)

	// 查询需要检查的交易
	var transactions []models.StrategyTransaction
	if err := dbconfig.DB.Where("created_at > ? AND status = ?", checkTime, "pending").
		Find(&transactions).Error; err != nil {
		return err
	}

	// 如果没有需要检查的交易，直接返回
	if len(transactions) == 0 {
		return nil
	}

	// 批量检查交易状态并更新
	for _, tx := range transactions {
		status, err := solana.CheckTransactionStatus(tx.Signature)
		if err != nil {
			log.Printf("Error checking transaction %s: %v", tx.Signature, err)
			continue
		}

		// 更新交易状态
		var newStatus string
		switch status {
		case "confirmed", "finalized":
			newStatus = "completed"
		case "pending":
			newStatus = "pending"
		default:
			newStatus = "error"
		}

		// 只有状态发生变化时才更新数据库
		if newStatus != tx.Status {
			if err := dbconfig.DB.Model(&tx).Update("status", newStatus).Error; err != nil {
				log.Printf("Error updating transaction %d status: %v", tx.ID, err)
				continue
			}
			log.Printf("Updated transaction %d status from %s to %s", tx.ID, tx.Status, newStatus)
		}
	}

	return nil
}

func main() {
	// 创建日志目录
	os.MkdirAll("logs", 0755)
	file, err := os.OpenFile("logs/strategy_stat.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	
	// 配置日志格式
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	// logrus.SetLevel(logrus.ErrorLevel)
	logrus.SetLevel(logrus.InfoLevel)

	if err == nil {
		logrus.SetOutput(file)
	} else {
		logrus.Warn("无法打开日志文件，日志将输出到标准输出")
	}

	// 初始化数据库
	dbconfig.InitDB()
	if dbconfig.DB == nil {
		logrus.Fatal("Failed to initialize database")
		return
	}

	logrus.Info("> 初始化程序完成")

	for {
		// 等待指定时间后继续下一轮更新
		time.Sleep(CHECK_INTERVAL)

		// 创建带超时的 context
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

		// 执行策略交易状态检查
		if err := checkAndUpdateTransactionStatus(); err != nil {
			logrus.Errorf("Error checking strategy transaction status: %v", err)
		}

		// 取消 context 避免资源泄露
		cancel()
	}
}
