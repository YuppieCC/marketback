package main

import (
	"os"
	"time"
	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	log "github.com/sirupsen/logrus"
)

const WSOl_MINT = "So11111111111111111111111111111111111111112"
const SOL_MINT = "sol"

func main() {
	// 日志输出到文件
	os.MkdirAll("logs", 0755)
	file, err := os.OpenFile("logs/snapshot_schedule.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
	log.Infof("> 初始化程序完成")
	for {
		time.Sleep(900 * time.Second)
		var projects []models.ProjectConfig
		if err := config.DB.Preload("Token").Where("is_active = ? AND snapshot_enabled = ?", true, true).Find(&projects).Error; err != nil {
			log.Errorf("> 查询项目失败: %v", err)
		}

		for _, project := range projects {
			// 根据池子平台检查相应的池子配置
			var skipProject bool
			switch project.PoolPlatform {
			case "raydium":
				var pool models.PoolConfig
				if err := config.DB.First(&pool, project.PoolID).Error; err != nil {
					log.Warnf("项目 %d (raydium) 未找到 Pool，跳过: %v", project.ID, err)
					skipProject = true
				}
			case "pumpfun_internal":
				var pumpfunPool models.PumpfuninternalConfig
				if err := config.DB.First(&pumpfunPool, project.PoolID).Error; err != nil {
					log.Warnf("项目 %d (pumpfun_internal) 未找到 PumpfunPool，跳过: %v", project.ID, err)
					skipProject = true
				}
			default:
				log.Warnf("项目 %d 的池子平台 %s 不支持，跳过", project.ID, project.PoolPlatform)
				skipProject = true
			}

			if skipProject {
				continue
			}

			// 更新快照ID
			project.SnapshotCount += 1
			snapshotID := uint(project.SnapshotCount)

			// 根据不同的池子平台创建不同类型的快照
			switch project.PoolPlatform {
			case "pumpfun_internal":
				if err := createPumpfunInternalSnapshot(project, snapshotID); err != nil {
					log.Errorf("> 创建 PumpfunInternalSnapshot 失败: %v", err)
					continue
				}
			case "raydium":
				if err := createPoolSnapshot(project, snapshotID); err != nil {
					log.Errorf("> 创建 PoolSnapshot 失败: %v", err)
					continue
				}
			default:
				log.Warnf("项目 %d 的池子平台 %s 不支持，跳过", project.ID, project.PoolPlatform)
				continue
			}

			// 创建钱包快照
			if err := createWalletSnapshots(project, snapshotID); err != nil {
				log.Errorf("> 创建钱包快照失败: %v", err)
				continue
			}

			// 更新项目的快照计数
			if err := config.DB.Model(&models.ProjectConfig{}).Where("id = ?", project.ID).UpdateColumn("snapshot_count", project.SnapshotCount).Error; err != nil {
				log.Errorf("> 更新 ProjectConfig 的 snapshot_count 失败: %v", err)
				continue
			}
		}
	}
}

func createPumpfunInternalSnapshot(project models.ProjectConfig, snapshotID uint) error {
	var pumpfunStat models.PumpfuninternalStat
	if err := config.DB.Where("pumpfuninternal_id = ?", project.PoolID).First(&pumpfunStat).Error; err != nil {
		return err
	}

	snapshot := models.PumpfuninternalSnapshot{
		ProjectID:            project.ID,
		SnapshotID:          snapshotID,
		PumpfuninternalID:   project.PoolID,
		Mint:                pumpfunStat.Mint,
		UnknownData:         pumpfunStat.UnknownData,
		VirtualTokenReserves: pumpfunStat.VirtualTokenReserves,
		VirtualSolReserves:  pumpfunStat.VirtualSolReserves,
		RealTokenReserves:   pumpfunStat.RealTokenReserves,
		RealSolReserves:     pumpfunStat.RealSolReserves,
		TokenTotalSupply:    pumpfunStat.TokenTotalSupply,
		Complete:            pumpfunStat.Complete,
		Creator:             pumpfunStat.Creator,
		Price:               pumpfunStat.Price,
		FeeRecipient:        pumpfunStat.FeeRecipient,
		SolBalance:          pumpfunStat.SolBalance,
		TokenBalance:        pumpfunStat.TokenBalance,
		Slot:               pumpfunStat.Slot,
		SourceUpdatedAt:     pumpfunStat.UpdatedAt,
		CreatedAt:           time.Now(),
	}

	return config.DB.Create(&snapshot).Error
}

func createPoolSnapshot(project models.ProjectConfig, snapshotID uint) error {
	var poolStat models.PoolStat
	if err := config.DB.Where("pool_id = ?", project.PoolID).First(&poolStat).Error; err != nil {
		return err
	}

	// 获取池子信息
	var pool models.PoolConfig
	if err := config.DB.First(&pool, project.PoolID).Error; err != nil {
		return err
	}

	snapshot := models.PoolSnapshot{
		ProjectID:           project.ID,
		SnapshotID:         snapshotID,
		PoolAddress:        pool.PoolAddress,
		BaseAmountReadable: poolStat.BaseAmountReadable,
		QuoteAmountReadable: poolStat.QuoteAmountReadable,
		MarketValue:        poolStat.MarketValue,
		LpSupply:           poolStat.LpSupply,
		Price:              poolStat.Price,
		SourceUpdatedAt:    poolStat.UpdatedAt,
		CreatedAt:          time.Now(),
	}

	return config.DB.Create(&snapshot).Error
}

func createWalletSnapshots(project models.ProjectConfig, snapshotID uint) error {
	if project.Token == nil {
		log.Warnf("项目 %d 未配置 Token，跳过钱包快照", project.ID)
		return nil
	}

	var roles []models.RoleConfig
	if err := config.DB.Where("project_id = ?", project.ID).Find(&roles).Error; err != nil {
		return err
	}

	var roleIDs []uint
	for _, role := range roles {
		roleIDs = append(roleIDs, role.ID)
	}

	if len(roleIDs) == 0 {
		log.Warnf("> 项目 %d 没有角色，跳过钱包快照", project.ID)
		return nil
	}

	var roleAddresses []models.RoleAddress
	if err := config.DB.Where("role_id IN ?", roleIDs).Find(&roleAddresses).Error; err != nil {
		return err
	}

	for _, addr := range roleAddresses {
		CreateWalletTokenSnapshot(project.ID, snapshotID, project.Token.Mint, addr)
		CreateWalletTokenSnapshot(project.ID, snapshotID, SOL_MINT, addr)
		CreateWalletTokenSnapshot(project.ID, snapshotID, WSOl_MINT, addr)
	}

	return nil
}

func CreateWalletTokenSnapshot(projectID uint, snapshotID uint, tokenMint string, addr models.RoleAddress) {
	var walletStat models.WalletTokenStat
	if err := config.DB.Where("owner_address = ? AND mint = ?", addr.Address, tokenMint).First(&walletStat).Error; err != nil {
		log.Warnf("> 未找到 WalletTokenStat, address=%s, mint=%s, 跳过: %v", addr.Address, tokenMint, err)
		return
	}

	walletSnapshot := models.WalletTokenSnapshot{
		ProjectID:       projectID,
		SnapshotID:     snapshotID,
		RoleID:         addr.RoleID,
		OwnerAddress:   addr.Address,
		Mint:           tokenMint,
		BalanceReadable: walletStat.BalanceReadable,
		Slot:           walletStat.Slot,
		BlockTime:      walletStat.BlockTime,
		SourceUpdatedAt: walletStat.UpdatedAt,
		CreatedAt:      time.Now(),
	}

	if err := config.DB.Create(&walletSnapshot).Error; err != nil {
		log.Errorf("> 创建 WalletTokenSnapshot 失败: %v", err)
		return
	}

	log.Infof("> 钱包快照成功，ProjectID: %d, SnapshotID: %d, Address: %s", projectID, snapshotID, addr.Address)
}
