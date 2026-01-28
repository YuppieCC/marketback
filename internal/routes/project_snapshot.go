package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

func SetupProjectSnapshotRoutes(r *gin.Engine) {
	wallet := r.Group("/wallet-token-snapshot")
	{
		wallet.GET("", handlers.ListWalletTokenSnapshots)
		wallet.GET(":id", handlers.GetWalletTokenSnapshot)
		wallet.GET("/by-project/:project_id", handlers.ListWalletTokenSnapshotsByProject)
		wallet.GET("/by-snapshot/:snapshot_id", handlers.ListWalletTokenSnapshotsBySnapshotID)
		wallet.POST("/by-role/:role_id", handlers.ListWalletTokenSnapshotsByRoleID)
		wallet.POST("/aggregate/by-role/:role_id", handlers.ListAggregateWalletTokenSnapshotsByRoleID)
	}

	pool := r.Group("/pool-snapshot")
	{
		pool.GET("", handlers.ListPoolSnapshots)
		pool.GET(":id", handlers.GetPoolSnapshot)
		pool.POST("/by-project/:project_id", handlers.GetPoolSnapshotByProject)
		pool.GET("/by-snapshot/:snapshot_id", handlers.ListPoolSnapshotsBySnapshotID)
	}

	settle := r.Group("/settle-snapshot")
	{
		settle.POST("/by-project/:project_id", handlers.GetSettleSnapshotByProject)
	}

	pumpfuninternal := r.Group("/pumpfuninternal-snapshot")
	{
		pumpfuninternal.GET("", handlers.ListPumpfuninternalSnapshots)
		pumpfuninternal.GET(":id", handlers.GetPumpfuninternalSnapshot)
		pumpfuninternal.POST("/by-project/:project_id", handlers.GetPumpfuninternalSnapshotByProject)
		pumpfuninternal.GET("/by-snapshot/:snapshot_id", handlers.ListPumpfuninternalSnapshotsBySnapshotID)
	}
}
