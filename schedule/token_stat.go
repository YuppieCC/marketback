package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"gorm.io/gorm"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	solanaUtils "marketcontrol/pkg/solana"

	log "github.com/sirupsen/logrus"
)

const (
	ROLE_MAX_CONCURRENT      = 2
	ADDRESS_MAX_CONCURRENT   = 2
	ADDRESSS_UPDATE_INTERVAL = 1
	WSOl_MINT                = "So11111111111111111111111111111111111111112"
	MaxLastUpdatedAt         = 24 * 60 * 60 // 24小时的秒数
)

// 需要更新的代币 mint 列表
var UPDATE_MINT = []string{
	"6n1kJcgg6xrSiG1dF4rBnvffraeBCB4NNVhyVVTHWygp",
}

var RPC_URLS = []string{
	// mainnet
	"https://cold-crimson-aura.solana-mainnet.quiknode.pro/e7db90e833b5b3ca15ce04663544243d256a3a80/",
	"https://red-wider-scion.solana-mainnet.quiknode.pro/7d63bea9a0a2d0a3664671d551a2d3565bef43b6/",
	"https://still-boldest-daylight.solana-mainnet.quiknode.pro/00fa699b3a64d931c80b6802a8aa44086bf4e54f/",
	"https://bitter-bitter-liquid.solana-mainnet.quiknode.pro/42dc87e6ee0d0ed68ccc7b50382f540f410d637e/",
	// devnet
	// "https://intensive-floral-night.solana-devnet.quiknode.pro/72da73fdff26c7bf461329b393efbf9e3b267678/",
}

// 获取随机 RPC URL
func getRandomRPCURL() string {
	rand.Seed(time.Now().UnixNano())
	return RPC_URLS[rand.Intn(len(RPC_URLS))]
}

// UpdateAllAddressBalance 更新所有地址的指定代币余额
func UpdateAllAddressBalance(db *gorm.DB, client *rpc.Client) {
	log.Infof("> 开始更新所有地址的代币余额")
	log.Infof("> 需要更新的代币: %v", UPDATE_MINT)

	// 获取所有 AddressManage
	var addresses []models.AddressManage
	if err := db.Find(&addresses).Error; err != nil {
		log.Errorf("> 查询所有 AddressManage 失败: %v", err)
		return
	}
	log.Infof("> 找到 %d 个地址", len(addresses))

	if len(addresses) == 0 {
		log.Warnf("> 没有找到任何地址，跳过更新")
		return
	}

	// 使用信号量控制并发
	sem := make(chan struct{}, ADDRESS_MAX_CONCURRENT)
	var wg sync.WaitGroup

	for i, addr := range addresses {
		wg.Add(1)
		sem <- struct{}{} // acquire

		// 添加并发间隔控制，除了第一个地址
		if i > 0 {
			time.Sleep(100 * time.Millisecond) // 0.1秒间隔
		}

		go func(address models.AddressManage) {
			defer wg.Done()
			defer func() { <-sem }() // release

			// 解析地址
			pubkey, err := solana.PublicKeyFromBase58(address.Address)
			if err != nil {
				log.Errorf("> 无效地址: %s, 错误: %v", address.Address, err)
				return
			}

			log.Infof("> 开始更新地址: %s 的代币余额", address.Address)
			currentTime := time.Now()

			// 检查并更新 SOL 余额
			var solStat models.WalletTokenStat
			needUpdateSol := true
			if err := db.Where("owner_address = ? AND mint = ?", address.Address, "sol").First(&solStat).Error; err == nil {
				timeSinceLastUpdate := currentTime.Sub(solStat.UpdatedAt).Seconds()
				if timeSinceLastUpdate <= MaxLastUpdatedAt {
					needUpdateSol = false
					log.Infof("> 地址 %s SOL 余额更新时间未超过 %d 秒，跳过更新", address.Address, MaxLastUpdatedAt)
				}
			}

			if needUpdateSol {
				solBalance, solUpdateTime, err := solanaUtils.GetSolBalance(client, pubkey)
				if err != nil {
					log.Errorf("> 查询地址 %s SOL 余额失败: %v", address.Address, err)
				} else {
					UpdateWalletTokenStat(db, address.Address, "sol", 1e9, solBalance, solUpdateTime)
					log.Infof("> 地址 %s SOL 余额: %d", address.Address, solBalance)
				}
			}

			// 检查并更新 WSOL 余额
			var wsolStat models.WalletTokenStat
			needUpdateWsol := true
			if err := db.Where("owner_address = ? AND mint = ?", address.Address, WSOl_MINT).First(&wsolStat).Error; err == nil {
				timeSinceLastUpdate := currentTime.Sub(wsolStat.UpdatedAt).Seconds()
				if timeSinceLastUpdate <= MaxLastUpdatedAt {
					needUpdateWsol = false
					log.Infof("> 地址 %s WSOL 余额更新时间未超过 %d 秒，跳过更新", address.Address, MaxLastUpdatedAt)
				}
			}

			if needUpdateWsol {
				wsolBalance, wsolUpdateTime, err := solanaUtils.GetTokenBalance(db, client, pubkey, WSOl_MINT)
				if err != nil {
					log.Errorf("> 查询地址 %s WSOL 余额失败: %v", address.Address, err)
				} else {
					UpdateWalletTokenStat(db, address.Address, WSOl_MINT, 1e9, wsolBalance, wsolUpdateTime)
					log.Infof("> 地址 %s WSOL 余额: %d", address.Address, wsolBalance)
				}
			}

			// 检查并更新 UPDATE_MINT 中指定的代币余额
			for _, mint := range UPDATE_MINT {
				// 添加延迟避免 RPC 限制
				time.Sleep(time.Duration(ADDRESSS_UPDATE_INTERVAL * float64(time.Second)))

				// 检查该代币是否需要更新
				var tokenStat models.WalletTokenStat
				needUpdateToken := true
				if err := db.Where("owner_address = ? AND mint = ?", address.Address, mint).First(&tokenStat).Error; err == nil {
					timeSinceLastUpdate := currentTime.Sub(tokenStat.UpdatedAt).Seconds()
					if timeSinceLastUpdate <= MaxLastUpdatedAt {
						needUpdateToken = false
						log.Infof("> 地址 %s 代币 %s 余额更新时间未超过 %d 秒，跳过更新", address.Address, mint, MaxLastUpdatedAt)
					}
				}

				if needUpdateToken {
					// 使用默认的 decimals (9) 来计算可读余额
					decimalsWithPow := math.Pow(10, 9)
					balance, updateTime, err := solanaUtils.GetTokenBalance(db, client, pubkey, mint)
					if err != nil {
						log.Errorf("> 查询地址 %s 代币 %s 余额失败: %v", address.Address, mint, err)
					} else {
						UpdateWalletTokenStat(db, address.Address, mint, decimalsWithPow, balance, updateTime)
						log.Infof("> 地址 %s 代币 %s 余额: %d", address.Address, mint, balance)
					}
				}
			}

			log.Infof("> 完成地址: %s 的代币余额更新", address.Address)
		}(addr)
	}

	wg.Wait()
	log.Infof("> 完成所有地址的代币余额更新")
}

func main() {
	// 日志输出到文件
	os.MkdirAll("logs", 0755)
	file, err := os.OpenFile("logs/update_project_stat_schedule.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	log.SetLevel(log.InfoLevel)
	log.Infof("> 初始化程序完成")

	if err == nil {
		log.SetOutput(file)
	} else {
		log.Warn("无法打开日志文件，日志将输出到标准输出")
	}

	config.InitDB()

	// 使用随机 RPC URL 创建 client
	solanaRPC := getRandomRPCURL()
	client := rpc.New(solanaRPC)
	log.Infof("> 使用 RPC 节点: %s", solanaRPC)
	log.Infof("> 初始化程序完成")

	// 运行 UpdateAllAddressBalance 函数
	UpdateAllAddressBalance(config.DB, client)
}

func UpdateWalletTokenStat(db *gorm.DB, address, mint string, deciamls_with_pow float64, balance uint64, updateTime time.Time) {
	var stat models.WalletTokenStat
	if err := db.Where("owner_address = ? AND mint = ?", address, mint).First(&stat).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			stat = models.WalletTokenStat{
				OwnerAddress:    address,
				Mint:            mint,
				Balance:         balance,
				BalanceReadable: float64(balance) / deciamls_with_pow,
				Slot:            0,
				BlockTime:       updateTime,
			}
			if err := db.Create(&stat).Error; err != nil {
				log.Errorf("> 创建 地址: %s, 代币: %s, WalletTokenStat 失败: %v", address, mint, err)
			}
		} else {
			log.Errorf("> 查询 地址: %s, 代币: %s, WalletTokenStat 失败: %v", address, mint, err)
		}
	} else {
		stat.Balance = balance
		stat.BalanceReadable = float64(balance) / deciamls_with_pow
		stat.Slot = 0
		stat.BlockTime = updateTime
		if err := db.Save(&stat).Error; err != nil {
			log.Errorf("> 更新 地址: %s, 代币: %s, WalletTokenStat 失败: %v", address, mint, err)
		}
	}
}

// checkTokenStatsNeedUpdate 检查地址的代币状态是否需要更新
func checkTokenStatsNeedUpdate(db *gorm.DB, address string, mint string, currentTime time.Time) (bool, error) {
	// 检查并清理重复数据
	mints := []string{mint, WSOl_MINT, "sol"}
	for _, tokenMint := range mints {
		var duplicates []models.WalletTokenStat
		if err := db.Where("owner_address = ? AND mint = ?", address, tokenMint).
			Order("updated_at DESC").Find(&duplicates).Error; err != nil { // 按更新时间降序排序
			return false, fmt.Errorf("查询地址 %s 的代币 %s 状态失败: %v", address, tokenMint, err)
		}

		// 如果有重复记录，保留最新的一条，删除其他的
		if len(duplicates) > 1 {
			log.Infof("> 地址: %s 的代币 %s 存在 %d 条重复记录，保留最新的记录，删除其他记录",
				address, tokenMint, len(duplicates))

			// 获取要删除的记录的ID列表（除了第一条最新的记录）
			var idsToDelete []uint
			for i := 1; i < len(duplicates); i++ { // 从第二条记录开始删除（保留第一条最新的记录）
				idsToDelete = append(idsToDelete, duplicates[i].ID)
			}

			// 批量删除重复记录
			if err := db.Delete(&models.WalletTokenStat{}, idsToDelete).Error; err != nil {
				return false, fmt.Errorf("删除地址 %s 的代币 %s 重复记录失败: %v", address, tokenMint, err)
			}
			log.Infof("> 已删除地址: %s 的代币 %s 的 %d 条重复记录",
				address, tokenMint, len(idsToDelete))
		}
	}

	// 重新查询清理后的记录
	var stats []models.WalletTokenStat
	if err := db.Where("owner_address = ? AND mint IN (?, ?, ?)",
		address, mint, WSOl_MINT, "sol").Find(&stats).Error; err != nil {
		return false, fmt.Errorf("查询地址 %s 的代币状态失败: %v", address, err)
	}

	// 检查是否需要更新
	if len(stats) < 3 { // 应该有3条记录：项目代币、WSOL和SOL
		log.Infof("> 地址: %s 的代币状态记录不完整（当前: %d, 期望: 3），需要更新", address, len(stats))
		return true, nil
	}

	// 检查每条记录的更新时间
	for _, stat := range stats {
		timeSinceLastUpdate := currentTime.Sub(stat.UpdatedAt).Seconds()
		if timeSinceLastUpdate > MaxLastUpdatedAt {
			log.Infof("> 地址: %s 的代币 %s 状态更新时间超过 %d 秒，需要更新",
				address, stat.Mint, MaxLastUpdatedAt)
			return true, nil
		}
	}

	log.Infof("> 地址: %s 的所有代币状态更新时间均未超过 %d 秒，跳过更新", address, MaxLastUpdatedAt)
	return false, nil
}

func UpdateRolesStat(db *gorm.DB, roles []models.RoleConfig, mint string, deciamls_with_pow float64, client *rpc.Client) {
	sem := make(chan struct{}, ROLE_MAX_CONCURRENT)
	var wg sync.WaitGroup
	currentTime := time.Now()

	for _, role := range roles {
		// Skip if updates are disabled
		if !role.UpdateEnabled {
			log.Infof("> 角色: %d 已禁用更新", role.ID)
			continue
		}

		// Check if it's time to update based on UpdateInterval and LastUpdateAt
		update_interval := role.UpdateInterval
		if update_interval == 0 {
			update_interval = 1
		}

		// Skip if not time to update yet
		if role.LastUpdateAt != nil {
			timeSinceLastUpdate := currentTime.Sub(*role.LastUpdateAt).Seconds()
			if timeSinceLastUpdate < update_interval {
				log.Infof("> 角色: %d 未到更新时间, 距离上次更新: %.2f秒, 更新间隔: %.2f秒", role.ID, timeSinceLastUpdate, update_interval)
				continue
			}
		}

		wg.Add(1)
		sem <- struct{}{} // acquire
		go func(role models.RoleConfig) {
			defer wg.Done()
			defer func() { <-sem }() // release

			var addresses []models.RoleAddress
			if err := db.Where("role_id = ?", role.ID).Find(&addresses).Error; err != nil {
				log.Errorf("> 查询 角色: %d, 地址失败: %v", role.ID, err)
				return
			}

			if len(addresses) == 0 {
				log.Errorf("> 角色: %d, 没有地址", role.ID)
				return
			}

			addrSem := make(chan struct{}, ADDRESS_MAX_CONCURRENT)
			var addrWg sync.WaitGroup
			for _, addr := range addresses {
				addrWg.Add(1)
				addrSem <- struct{}{}
				go func(addr models.RoleAddress) {
					defer addrWg.Done()
					defer func() { <-addrSem }()

					needUpdate, err := checkTokenStatsNeedUpdate(db, addr.Address, mint, currentTime)
					if err != nil {
						log.Errorf("%v", err)
						return
					}

					if !needUpdate {
						return
					}

					pubkey, err := solana.PublicKeyFromBase58(addr.Address)
					if err != nil {
						log.Errorf("> 角色: %d, 无效地址: %s", role.ID, addr.Address)
						return
					}

					// 查询 SPL Token 余额
					time.Sleep(time.Duration(ADDRESSS_UPDATE_INTERVAL * float64(time.Second))) // 启动下一个 goroutine 前 sleep
					balance, updateTime, err := solanaUtils.GetTokenBalance(db, client, pubkey, mint)
					log.Infof("> 角色: %d, 查询地址 %s 的代币: %s 余额, balance: %d, updateTime: %s error: %v", role.ID, addr.Address, mint, balance, updateTime, err)
					if err != nil {
						log.Errorf("> 角色: %d, 查询地址 %s 余额失败: %v", role.ID, addr.Address, err)
					} else {
						UpdateWalletTokenStat(db, addr.Address, mint, deciamls_with_pow, balance, updateTime)
					}

					// 查询 SOL 余额
					time.Sleep(time.Duration(ADDRESSS_UPDATE_INTERVAL * float64(time.Second)))
					solBalance, solUpdateTime, err := solanaUtils.GetSolBalance(client, pubkey)
					log.Infof("> 角色: %d, 查询地址 %s SOL 余额, solBalance: %d, solUpdateTime: %s error: %v", role.ID, addr.Address, solBalance, solUpdateTime, err)
					if err != nil {
						log.Errorf("> 查询地址 %s SOL 余额失败: %v", addr.Address, err)
					} else {
						UpdateWalletTokenStat(db, addr.Address, "sol", 1e9, solBalance, solUpdateTime)
					}

					// 查询 WSOL 余额
					// time.Sleep(time.Duration(ADDRESSS_UPDATE_INTERVAL * float64(time.Second)))
					// wsolBalance, wsolUpdateTime, err := solanaUtils.GetTokenBalance(db, client, pubkey, WSOl_MINT)
					// log.Infof("> 角色: %d, 查询地址 %s WSOL 余额, wsolBalance: %d, wsolUpdateTime: %s error: %v", role.ID, addr.Address, wsolBalance, wsolUpdateTime, err)
					// if err != nil {
					// 	log.Errorf("> 查询地址 %s WSOL 余额失败: %v", addr.Address, err)
					// } else {
					// 	UpdateWalletTokenStat(db, addr.Address, WSOl_MINT, 1e9, wsolBalance, wsolUpdateTime)
					// }
				}(addr)
			}
			addrWg.Wait()

			// Update LastUpdateAt after successful processing
			if err := db.Model(&role).Update("last_update_at", currentTime).Error; err != nil {
				log.Errorf("> 更新角色: %d 的 LastUpdateAt 失败: %v", role.ID, err)
			}
		}(role)
	}
	wg.Wait()
}
