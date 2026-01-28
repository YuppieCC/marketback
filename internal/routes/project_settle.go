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
		projectSettle.POST("/vesting-reivew", handlers.VestingReview)
	}
}
