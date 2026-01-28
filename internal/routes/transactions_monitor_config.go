package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupTransactionsMonitorConfigRoutes sets up all routes related to Transactions Monitor Config management
func SetupTransactionsMonitorConfigRoutes(r *gin.Engine) {
	// Setup monitor config routes
	monitorGroup := r.Group("/api/transactions-monitor-config")
	{
		monitorGroup.POST("", handlers.CreateTransactionsMonitorConfig)
		monitorGroup.GET("/:id", handlers.GetTransactionsMonitorConfig)
		monitorGroup.GET("", handlers.ListTransactionsMonitorConfigs)
		monitorGroup.PUT("/:id", handlers.UpdateTransactionsMonitorConfig)
		monitorGroup.DELETE("/:id", handlers.DeleteTransactionsMonitorConfig)
		monitorGroup.POST("/delete-with-data", handlers.DeleteTransactionsMonitorConfigWithData)
	}

	// Setup address transaction routes
	transactionGroup := r.Group("/api/address-transaction")
	{
		transactionGroup.POST("", handlers.CreateAddressTransaction)
		transactionGroup.GET("/:id", handlers.GetAddressTransaction)
		transactionGroup.GET("", handlers.ListAddressTransactions)
		transactionGroup.PUT("/:id", handlers.UpdateAddressTransaction)
		transactionGroup.DELETE("/:id", handlers.DeleteAddressTransaction)
	}

	// Setup address balance change routes
	balanceGroup := r.Group("/api/address-balance-change")
	{
		balanceGroup.POST("", handlers.CreateAddressBalanceChange)
		balanceGroup.GET("/:id", handlers.GetAddressBalanceChange)
		balanceGroup.GET("", handlers.ListAddressBalanceChanges)
		balanceGroup.PUT("/:id", handlers.UpdateAddressBalanceChange)
		balanceGroup.DELETE("/:id", handlers.DeleteAddressBalanceChange)
		balanceGroup.POST("/filter", handlers.FilterListAddressBalanceChanges)
	}

	// Setup pumpfuninternal swap routes
	swapGroup := r.Group("/api/pumpfuninternal-swap")
	{
		swapGroup.POST("", handlers.CreatePumpfuninternalSwap)
		swapGroup.GET("/:id", handlers.GetPumpfuninternalSwap)
		swapGroup.GET("", handlers.ListPumpfuninternalSwaps)
		swapGroup.PUT("/:id", handlers.UpdatePumpfuninternalSwap)
		swapGroup.DELETE("/:id", handlers.DeletePumpfuninternalSwap)
		swapGroup.POST("/filter", handlers.FilterPumpfuninternalSwaps)
		swapGroup.GET("/pool/:pool_id", handlers.ListPumpfuninternalSwapsByPoolID)
	}

	// Setup pumpfuninternal holder routes
	holderGroup := r.Group("/api/pumpfuninternal-holder")
	{
		holderGroup.POST("", handlers.CreatePumpfuninternalHolder)
		holderGroup.GET("/:id", handlers.GetPumpfuninternalHolder)
		holderGroup.GET("", handlers.ListPumpfuninternalHolders)
		holderGroup.PUT("/:id", handlers.UpdatePumpfuninternalHolder)
		holderGroup.DELETE("/:id", handlers.DeletePumpfuninternalHolder)
		holderGroup.POST("/filter", handlers.FilterPumpfuninternalHolders)
		holderGroup.POST("/project/:project_id", handlers.GetPumpfuninternalHolderByProjectID)
	}

	// Setup pumpfunammpool swap routes
	ammSwapGroup := r.Group("/api/pumpfunammpool-swap")
	{
		ammSwapGroup.POST("", handlers.CreatePumpfunAmmPoolSwap)
		ammSwapGroup.GET("/:id", handlers.GetPumpfunAmmPoolSwap)
		ammSwapGroup.GET("", handlers.ListPumpfunAmmPoolSwaps)
		ammSwapGroup.PUT("/:id", handlers.UpdatePumpfunAmmPoolSwap)
		ammSwapGroup.DELETE("/:id", handlers.DeletePumpfunAmmPoolSwap)
		ammSwapGroup.POST("/filter", handlers.FilterPumpfunAmmPoolSwaps)
		ammSwapGroup.GET("/pool/:pool_id", handlers.ListPumpfunAmmPoolSwapsByPoolID)
	}

	// Setup pumpfunammpool holder routes
	ammHolderGroup := r.Group("/api/pumpfunammpool-holder")
	{
		ammHolderGroup.POST("", handlers.CreatePumpfunAmmpoolHolder)
		ammHolderGroup.GET("/:id", handlers.GetPumpfunAmmpoolHolder)
		ammHolderGroup.GET("", handlers.ListPumpfunAmmpoolHolders)
		ammHolderGroup.PUT("/:id", handlers.UpdatePumpfunAmmpoolHolder)
		ammHolderGroup.DELETE("/:id", handlers.DeletePumpfunAmmpoolHolder)
		ammHolderGroup.POST("/filter", handlers.FilterPumpfunAmmpoolHolders)
		ammHolderGroup.POST("/project/:project_id", handlers.GetPumpfunAmmpoolHolderByProjectID)
	}

	// Setup raydium pool holder routes
	raydiumHolderGroup := r.Group("/api/raydium-pool-holder")
	{
		raydiumHolderGroup.POST("", handlers.CreateRaydiumPoolHolder)
		raydiumHolderGroup.GET("/:id", handlers.GetRaydiumPoolHolder)
		raydiumHolderGroup.GET("", handlers.ListRaydiumPoolHolders)
		raydiumHolderGroup.PUT("/:id", handlers.UpdateRaydiumPoolHolder)
		raydiumHolderGroup.DELETE("/:id", handlers.DeleteRaydiumPoolHolder)
		raydiumHolderGroup.POST("/filter", handlers.FilterRaydiumPoolHolders)
	}

	// Setup raydium pool swap routes
	raydiumSwapGroup := r.Group("/api/raydium-pool-swap")
	{
		raydiumSwapGroup.POST("", handlers.CreateRaydiumPoolSwap)
		raydiumSwapGroup.GET("/:id", handlers.GetRaydiumPoolSwap)
		raydiumSwapGroup.GET("", handlers.ListRaydiumPoolSwaps)
		raydiumSwapGroup.PUT("/:id", handlers.UpdateRaydiumPoolSwap)
		raydiumSwapGroup.DELETE("/:id", handlers.DeleteRaydiumPoolSwap)
		raydiumSwapGroup.POST("/filter", handlers.FilterRaydiumPoolSwaps)
	}

	// Setup meteoradbc holder routes
	meteoradbcHolderGroup := r.Group("/api/meteoradbc-holder")
	{
		meteoradbcHolderGroup.POST("", handlers.CreateMeteoradbcHolder)
		meteoradbcHolderGroup.GET("/:id", handlers.GetMeteoradbcHolder)
		meteoradbcHolderGroup.GET("", handlers.ListMeteoradbcHolders)
		meteoradbcHolderGroup.PUT("/:id", handlers.UpdateMeteoradbcHolder)
		meteoradbcHolderGroup.DELETE("/:id", handlers.DeleteMeteoradbcHolder)
		meteoradbcHolderGroup.POST("/filter", handlers.FilterMeteoradbcHolders)
		meteoradbcHolderGroup.POST("/project/:project_id", handlers.GetMeteoradbcHolderByProjectID)
		meteoradbcHolderGroup.POST("/migrate/:poolAddress", handlers.MigrateHolderByPoolAddress)
	}

	// Setup meteoradbc swap routes
	meteoradbcSwapGroup := r.Group("/api/meteoradbc-swap")
	{
		meteoradbcSwapGroup.POST("", handlers.CreateMeteoradbcSwap)
		meteoradbcSwapGroup.GET("/:id", handlers.GetMeteoradbcSwap)
		meteoradbcSwapGroup.GET("", handlers.ListMeteoradbcSwaps)
		meteoradbcSwapGroup.PUT("/:id", handlers.UpdateMeteoradbcSwap)
		meteoradbcSwapGroup.DELETE("/:id", handlers.DeleteMeteoradbcSwap)
		meteoradbcSwapGroup.POST("/filter", handlers.FilterMeteoradbcSwaps)
		meteoradbcSwapGroup.GET("/pool/:pool_id", handlers.ListMeteoradbcSwapsByPoolID)
	}

	// Setup meteoracpmm holder routes
	meteoracpmmHolderGroup := r.Group("/api/meteoracpmm-holder")
	{
		meteoracpmmHolderGroup.POST("", handlers.CreateMeteoracpmmHolder)
		meteoracpmmHolderGroup.GET("/:id", handlers.GetMeteoracpmmHolder)
		meteoracpmmHolderGroup.GET("", handlers.ListMeteoracpmmHolders)
		meteoracpmmHolderGroup.PUT("/:id", handlers.UpdateMeteoracpmmHolder)
		meteoracpmmHolderGroup.DELETE("/:id", handlers.DeleteMeteoracpmmHolder)
		meteoracpmmHolderGroup.POST("/filter", handlers.FilterMeteoracpmmHolders)
		meteoracpmmHolderGroup.POST("/project/:project_id", handlers.GetMeteoracpmmHolderByProjectID)
	}

	// Setup meteoracpmm swap routes
	meteoracpmmSwapGroup := r.Group("/api/meteoracpmm-swap")
	{
		meteoracpmmSwapGroup.POST("", handlers.CreateMeteoracpmmSwap)
		meteoracpmmSwapGroup.GET("/:id", handlers.GetMeteoracpmmSwap)
		meteoracpmmSwapGroup.GET("", handlers.ListMeteoracpmmSwaps)
		meteoracpmmSwapGroup.PUT("/:id", handlers.UpdateMeteoracpmmSwap)
		meteoracpmmSwapGroup.DELETE("/:id", handlers.DeleteMeteoracpmmSwap)
		meteoracpmmSwapGroup.POST("/filter", handlers.FilterMeteoracpmmSwaps)
		meteoracpmmSwapGroup.GET("/pool/:pool_id", handlers.ListMeteoracpmmSwapsByPoolID)
	}

	// Setup swap transaction routes
	swapTransactionGroup := r.Group("/api/swap-transaction")
	{
		swapTransactionGroup.POST("", handlers.CreateSwapTransaction)
		swapTransactionGroup.GET("/:id", handlers.GetSwapTransaction)
		swapTransactionGroup.GET("", handlers.ListSwapTransactions)
		swapTransactionGroup.PUT("/:id", handlers.UpdateSwapTransaction)
		swapTransactionGroup.DELETE("/:id", handlers.DeleteSwapTransaction)
		swapTransactionGroup.POST("/filter", handlers.FilterSwapTransactions)
		swapTransactionGroup.GET("/pool/:pool_id", handlers.ListSwapTransactionsByPoolID)
		swapTransactionGroup.GET("/project/:project_id", handlers.GetSwapTransactionsByProject)
	}

}
