package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupStrategySignalRoutes sets up all routes related to Strategy Signal management
func SetupStrategySignalRoutes(r *gin.Engine) {
	signal := r.Group("/strategy-signal")
	{
		// Standard CRUD operations
		signal.GET("", handlers.ListStrategySignals)
		signal.GET("/:id", handlers.GetStrategySignal)
		signal.POST("", handlers.CreateStrategySignal)
		signal.PUT("/:id", handlers.UpdateStrategySignal)
		signal.DELETE("/:id", handlers.DeleteStrategySignal)
		
		// Filter operations
		signal.GET("/project/:project_id", handlers.GetStrategySignalsByProjectID)
		signal.GET("/strategy/:strategy_id", handlers.GetStrategySignalsByStrategyID)
		signal.GET("/role/:role_id", handlers.GetStrategySignalsByRoleID)
	}
} 