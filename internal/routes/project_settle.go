package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupProjectSettleRoutes sets up all routes related to Project Settle management
func SetupProjectSettleRoutes(r *gin.Engine) {
	projectSettle := r.Group("/project-settle")
	{
		projectSettle.GET("/profit-ranking", handlers.GetProjectProfitRanking)
		projectSettle.GET("/error-vesting", handlers.GetErrorVesting)
		projectSettle.POST("/fix-error-vesting", handlers.FixErrorVesting)
		projectSettle.POST("/fetch-creator-balance-change", handlers.FetchCreatorBalanceChange)
		projectSettle.POST("/vesting-reivew", handlers.VestingReview)
	}
}
