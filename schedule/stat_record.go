package main

import (
	"os"
	"time"

	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/internal/models"
	"marketcontrol/internal/handlers/business"

	"github.com/robfig/cron/v3"
	logger "github.com/sirupsen/logrus"
)

// getZeroSecondTime 获取当前时间的零秒时间戳
func getZeroSecondTime(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, t.Location())
}

// RecordProjectSettle 记录项目结算数据
func RecordProjectSettle() error {
	logger.Info("> 开始记录项目结算数据")

	// 1. 获取所有活跃项目
	var projects []models.ProjectConfig
	if err := dbconfig.DB.Preload("Token").Find(&projects).Error; err != nil {
		logger.Errorf("> 查询项目失败: %v", err)
		return err
	}

	logger.Infof("> 共找到 %d 个项目", len(projects))

	// 2. 遍历每个项目
	for _, project := range projects {
		if project.Token == nil {
			logger.Warnf("> 项目 %d 未配置 Token，跳过", project.ID)
			continue
		}

		var settleStats *business.SettleStats
		var err error
		var poolPrice float64
		var poolSlot uint64
		var poolUpdatedAt time.Time

		// 3. 根据池子类型获取数据
		switch project.PoolPlatform {
		case "pumpfun_internal":
			var pumpfunStat models.PumpfuninternalStat
			if err := dbconfig.DB.Preload("PumpfunPool").
				Where("pumpfuninternal_id = ?", project.PoolID).
				First(&pumpfunStat).Error; err != nil {
				logger.Errorf("> 获取项目 %d 的 PumpfuninternalStat 失败: %v", project.ID, err)
				continue
			}
			settleStats, err = business.CalculatePumpfunPoolSettle(&project, &pumpfunStat)
			if err != nil {
				logger.Errorf("> 计算项目 %d 的结算数据失败: %v", project.ID, err)
				continue
			}
			poolPrice = pumpfunStat.Price
			poolSlot = pumpfunStat.Slot
			poolUpdatedAt = pumpfunStat.UpdatedAt
			logger.Infof("> 项目 %d (pumpfun_internal) 数据获取成功", project.ID)

		case "pumpfun_amm":
			var pumpfunStat models.PumpfunAmmPoolStat
			if err := dbconfig.DB.Where("pool_id = ?", project.PoolID).
				First(&pumpfunStat).Error; err != nil {
				logger.Errorf("> 获取项目 %d 的 PumpfunAmmPoolStat 失败: %v", project.ID, err)
				continue
			}
			settleStats, err = business.CalculatePumpfunCombinedPoolSettle(&project, &pumpfunStat)
			if err != nil {
				logger.Errorf("> 计算项目 %d 的结算数据失败: %v", project.ID, err)
				continue
			}
			poolPrice = pumpfunStat.Price
			poolSlot = pumpfunStat.Slot
			poolUpdatedAt = pumpfunStat.UpdatedAt
			logger.Infof("> 项目 %d (pumpfun_amm) 数据获取成功", project.ID)

		default:
			logger.Warnf("> 不支持的池子类型 %s (项目 %d)", project.PoolPlatform, project.ID)
			continue
		}

		// 4. 创建记录
		now := time.Now()
		record := models.ProjectSettleRecord{
			ProjectID:                      project.ID,
			PoolPlatform:                   project.PoolPlatform,
			PoolPrice:                      poolPrice,
			PoolSlot:                       poolSlot,
			PoolUpdatedAt:                  poolUpdatedAt,
			TokenByProject:                 settleStats.TokenByProject,
			TokenByPool:                    settleStats.TokenByPool,
			TokenByRetailInvestors:         settleStats.TokenByRetailInvestors,
			TokenAllocationByRetailInvestors: settleStats.TokenAllocationByRetailInvestors,
			TokenAllocationByProject:       settleStats.TokenAllocationByProject,
			TokenAllocationByPool:          settleStats.TokenAllocationByPool,
			TvlByProjectToken:             settleStats.TvlByProjectToken,
			TvlByProject:                  settleStats.TvlByProject,
			TvlByRetailInvestors:         settleStats.TvlByRetailInvestors,
			ProjectPnl:                    settleStats.ProjectPnl,
			ProjectMinPnl:                 settleStats.ProjectMinPnl,
			CreatedAtByZeroSec:           getZeroSecondTime(now),
		}

		if err := dbconfig.DB.Create(&record).Error; err != nil {
			logger.Errorf("> 创建项目 %d 的结算记录失败: %v", project.ID, err)
			continue
		}
		logger.Infof("> 项目 %d 的结算记录创建成功", project.ID)
	}

	logger.Info("> 项目结算数据记录完成")
	return nil
}

func main() {
	// 配置日志输出到文件
	os.MkdirAll("logs", 0755)
	file, err := os.OpenFile("logs/project_settle_record.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.SetOutput(file)
	} else {
		logger.Warn("无法打开日志文件，日志将输出到标准输出")
	}

	// 配置日志格式
	logger.SetFormatter(&logger.TextFormatter{
		FullTimestamp: true,
	})
	// logger.SetLevel(logger.InfoLevel)  // 修改为 InfoLevel 以便查看更多日志
	logger.SetLevel(logger.ErrorLevel)
	logger.Info("> 开始初始化程序...")

	// 初始化数据库连接
	dbconfig.InitDB()
	logger.Info("> 数据库连接初始化完成")

	// 创建定时任务
	c := cron.New(cron.WithSeconds())
	
	// 每15分钟执行一次
	_, err = c.AddFunc("0 */15 * * * *", func() {
		if err := RecordProjectSettle(); err != nil {
			logger.Errorf("> 记录项目结算数据失败: %v", err)
		}
	})
	if err != nil {
		logger.Fatalf("> 添加定时任务失败: %v", err)
	}

	logger.Info("> 定时任务已启动，每15分钟执行一次")
	
	// 启动定时任务
	c.Start()

	// 保持程序运行
	select {}
} 