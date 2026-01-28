package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupMeteoracpmmConfigRoutes sets up the routes for meteoracpmm configuration management
func SetupMeteoracpmmConfigRoutes(r *gin.Engine) {
	meteoracpmm := r.Group("/meteoracpmm-config")
	{
		// List all meteoracpmm configurations
		meteoracpmm.GET("/", handlers.ListMeteoracpmmConfigs)

		meteoracpmm.GET("/pool/slice", handlers.ListMeteoracpmmConfigsBySlice)

		// Get meteoracpmm configuration by mint address
		meteoracpmm.GET("/mint/:mint_address", handlers.GetMeteoracpmmConfigByMint)

		// Get meteoracpmm configuration by creator
		meteoracpmm.GET("/creator/:creator", handlers.GetMeteoracpmmConfigByCreator)

		// Get meteoracpmm configuration by ID
		meteoracpmm.GET("/:id", handlers.GetMeteoracpmmConfig)

		// Get meteoracpmm configuration by pool address
		meteoracpmm.GET("/pool/:pool_address", handlers.GetMeteoracpmmConfigByPoolAddress)

		// Create new meteoracpmm configuration
		meteoracpmm.POST("/", handlers.CreateMeteoracpmmConfig)

		// Update meteoracpmm configuration
		meteoracpmm.PUT("/:id", handlers.UpdateMeteoracpmmConfig)

		// Update meteoracpmm configuration status
		meteoracpmm.PATCH("/:id/status", handlers.UpdateMeteoracpmmConfigStatus)

		// Delete meteoracpmm configuration
		meteoracpmm.DELETE("/:id", handlers.DeleteMeteoracpmmConfig)
	}
}
