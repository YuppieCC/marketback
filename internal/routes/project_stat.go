package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupProjectStatRoutes 设置项目统计相关的路由
func SetupProjectStatRoutes(r *gin.Engine) {
	walletStat := r.Group("/wallet-token-stat")
	{
		walletStat.GET("", handlers.ListWalletTokenStats)
		walletStat.GET("/:id", handlers.GetWalletTokenStat)
		walletStat.GET("/by-address/:address", handlers.GetWalletTokenStatsByAddress)
		walletStat.GET("/by-mint/:mint", handlers.GetWalletTokenStatsByMint)
		walletStat.POST("/by-role/:role_id", handlers.GetWalletTokenStatsByRole)
		walletStat.POST("/total/by-role/:role_id", handlers.GetTotalWalletTokenStatsByRole)
		walletStat.POST("/aggregate/by-role/:role_id", handlers.GetAggregateWalletTokenStatsByRole)
		walletStat.POST("/aggregate/by-project/:project_id", handlers.GetAggregateWalletTokenStatsByProject)
		walletStat.POST("/update/:address", handlers.UpdateWalletTokenStatsByAddress)
		walletStat.POST("/update-by-role/:role_id", handlers.UpdateWalletTokenStatsByRole)
		walletStat.POST("/batch-update", handlers.BatchUpdateWalletTokenStatsByAddressList)
		walletStat.POST("/batch-update-v2", handlers.BatchUpdateWalletTokenStatsByAddressListV2)
		walletStat.POST("/update/by-filter", handlers.UpdateAddressByFilter)
		walletStat.POST("/review/by-filter", handlers.ReviewAddressByFilter)
		walletStat.POST("/remove-duplicate", handlers.RemoveDuplicateWalletTokenStat)
		walletStat.POST("/locate-duplicate", handlers.LocateDuplicateWalletTokenStat)
		walletStat.POST("/update-from-source", handlers.UpdateWalletTokenStatFromSource)
	}

	poolStat := r.Group("/pool-stat")
	{
		poolStat.GET("", handlers.ListPoolStats)
		poolStat.GET("/:id", handlers.GetPoolStat)
		poolStat.GET("/by-pool/:pool_id", handlers.GetPoolStatsByPoolID)
		poolStat.GET("/by-project/:project_id", handlers.GetPoolStatByProjectID)
	}

	// 新增 pumpfun internal 统计路由
	pumpfunStat := r.Group("/pumpfuninternal-stat")
	{
		pumpfunStat.GET("", handlers.ListPumpfuninternalStats)
		pumpfunStat.GET("/:id", handlers.GetPumpfuninternalStat)
		pumpfunStat.GET("/by-pool/:pool_id", handlers.GetPumpfuninternalStatsByPoolID)
		pumpfunStat.GET("/by-project/:project_id", handlers.GetPumpfuninternalStatByProjectID)
		pumpfunStat.GET("/by-mint/:mint", handlers.GetPumpfuninternalStatsByMint)
	}

	// 新增结算统计路由
	settleStat := r.Group("/settle-stat")
	{
		settleStat.GET("/by-project/:project_id", handlers.GetSettleStatsByProject)
		settleStat.GET("/get-retail-sol-amount/by-project/:project_id", handlers.GetRetailSolAmountByProject)
	}

	// PumpfunAmmPool Stat routes
	pumpfunAmmPoolStatRoutes := r.Group("/pumpfunamm-pool-stat")
	{
		pumpfunAmmPoolStatRoutes.GET("", handlers.ListPumpfunAmmPoolStats)
		pumpfunAmmPoolStatRoutes.GET("/:id", handlers.GetPumpfunAmmPoolStat)
		pumpfunAmmPoolStatRoutes.GET("/pool/:pool_id", handlers.GetPumpfunAmmPoolStatByPoolID)
		pumpfunAmmPoolStatRoutes.GET("/by-project/:project_id", handlers.GetPumpfunAmmPoolStatByProjectID)
		pumpfunAmmPoolStatRoutes.DELETE("/:id", handlers.DeletePumpfunAmmPoolStat)
	}

	// RaydiumLaunchpadPoolStat routes
	raydiumLaunchpadPoolStatRoutes := r.Group("/raydium-launchpad-pool-stat")
	{
		raydiumLaunchpadPoolStatRoutes.GET("", handlers.ListRaydiumLaunchpadPoolStats)
		raydiumLaunchpadPoolStatRoutes.POST("", handlers.CreateRaydiumLaunchpadPoolStat)
		raydiumLaunchpadPoolStatRoutes.GET("/:id", handlers.GetRaydiumLaunchpadPoolStat)
		raydiumLaunchpadPoolStatRoutes.PUT("/:id", handlers.UpdateRaydiumLaunchpadPoolStat)
		raydiumLaunchpadPoolStatRoutes.DELETE("/:id", handlers.DeleteRaydiumLaunchpadPoolStat)
		raydiumLaunchpadPoolStatRoutes.GET("/pool/:pool_id", handlers.GetRaydiumLaunchpadPoolStatByPoolID)
		raydiumLaunchpadPoolStatRoutes.GET("/by-project/:project_id", handlers.GetRaydiumLaunchpadPoolStatByProjectID)
		raydiumLaunchpadPoolStatRoutes.GET("/by-mint/:mint", handlers.GetRaydiumLaunchpadPoolStatsByMint)
	}

	// RaydiumCpmmPoolStat routes
	raydiumCpmmPoolStatRoutes := r.Group("/raydium-cpmm-pool-stat")
	{
		raydiumCpmmPoolStatRoutes.GET("", handlers.ListRaydiumCpmmPoolStats)
		raydiumCpmmPoolStatRoutes.POST("", handlers.CreateRaydiumCpmmPoolStat)
		raydiumCpmmPoolStatRoutes.GET("/:id", handlers.GetRaydiumCpmmPoolStat)
		raydiumCpmmPoolStatRoutes.PUT("/:id", handlers.UpdateRaydiumCpmmPoolStat)
		raydiumCpmmPoolStatRoutes.DELETE("/:id", handlers.DeleteRaydiumCpmmPoolStat)
		raydiumCpmmPoolStatRoutes.GET("/pool/:pool_id", handlers.GetRaydiumCpmmPoolStatByPoolID)
		raydiumCpmmPoolStatRoutes.GET("/by-project/:project_id", handlers.GetRaydiumCpmmPoolStatByProjectID)
	}

	// MeteoradbcPoolStat routes
	meteoradbcPoolStatRoutes := r.Group("/meteoradbc-pool-stat")
	{
		meteoradbcPoolStatRoutes.GET("", handlers.ListMeteoradbcPoolStats)
		meteoradbcPoolStatRoutes.POST("", handlers.CreateMeteoradbcPoolStat)
		meteoradbcPoolStatRoutes.GET("/:id", handlers.GetMeteoradbcPoolStat)
		meteoradbcPoolStatRoutes.PUT("/:id", handlers.UpdateMeteoradbcPoolStat)
		meteoradbcPoolStatRoutes.DELETE("/:id", handlers.DeleteMeteoradbcPoolStat)
		meteoradbcPoolStatRoutes.GET("/by-pool-address/:pool_address", handlers.GetMeteoradbcPoolStatByPoolAddress)
		meteoradbcPoolStatRoutes.GET("/by-project/:project_id", handlers.GetMeteoradbcPoolStatByProjectID)
	}

	// MeteoracpmmPoolStat routes
	meteoracpmmPoolStatRoutes := r.Group("/meteoracpmm-pool-stat")
	{
		meteoracpmmPoolStatRoutes.GET("", handlers.ListMeteoracpmmPoolStats)
		meteoracpmmPoolStatRoutes.POST("", handlers.CreateMeteoracpmmPoolStat)
		meteoracpmmPoolStatRoutes.GET("/:id", handlers.GetMeteoracpmmPoolStat)
		meteoracpmmPoolStatRoutes.PUT("/:id", handlers.UpdateMeteoracpmmPoolStat)
		meteoracpmmPoolStatRoutes.DELETE("/:id", handlers.DeleteMeteoracpmmPoolStat)
		meteoracpmmPoolStatRoutes.GET("/by-pool-address/:pool_address", handlers.GetMeteoracpmmPoolStatByPoolAddress)
		meteoracpmmPoolStatRoutes.GET("/by-project/:project_id", handlers.GetMeteoracpmmPoolStatByProjectID)
	}
}
