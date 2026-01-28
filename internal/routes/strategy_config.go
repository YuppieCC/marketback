package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupStrategyConfigRoutes sets up all routes related to Strategy Config management
func SetupStrategyConfigRoutes(r *gin.Engine) {
	strategy := r.Group("/strategy-config")
	{
		// Standard CRUD operations
		strategy.GET("", handlers.ListStrategyConfigs)
		strategy.GET("/:id", handlers.GetStrategyConfig)
		strategy.POST("", handlers.CreateStrategyConfig)
		strategy.PUT("/:id", handlers.UpdateStrategyConfig)
		strategy.DELETE("/:id", handlers.DeleteStrategyConfig)

		// Special operations requested by user
		strategy.GET("/project/:project_id", handlers.ListStrategyConfigsByProjectId)
		strategy.POST("/close-all/:project_id", handlers.CloseStrategyConfigsByProjectId)
		strategy.POST("/close-type", handlers.CloseStrategyTypeByProjectId)
		strategy.POST("/check-close", handlers.CheckStrategyCloseByProjectId)
		strategy.PATCH("/:id/params", handlers.UpdateStrategyParams)
		strategy.PATCH("/:id/stat", handlers.UpdateStrategyStat)
		strategy.POST("/toggle/:id", handlers.ToggleStrategyConfig)
	}
}
