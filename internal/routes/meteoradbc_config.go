package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupMeteoradbcConfigRoutes sets up the routes for meteoradbc configuration management
func SetupMeteoradbcConfigRoutes(r *gin.Engine) {
	meteoradbc := r.Group("/meteoradbc-config")
	{
		// List all meteoradbc configurations
		meteoradbc.GET("/", handlers.ListMeteoradbcConfigs)

		meteoradbc.GET("/pool/slice", handlers.ListMeteoradbcConfigsBySlice)

		// Get meteoradbc configuration by mint address
		meteoradbc.GET("/mint/:mint_address", handlers.GetMeteoradbcConfigByMint)

		// Get meteoradbc configuration by creator
		meteoradbc.GET("/creator/:creator", handlers.GetMeteoradbcConfigByCreator)

		// Get meteoradbc configuration by ID
		meteoradbc.GET("/:id", handlers.GetMeteoradbcConfig)

		// Get meteoradbc configuration by pool address
		meteoradbc.GET("/pool/:pool_address", handlers.GetMeteoradbcConfigByPoolAddress)

		// Create new meteoradbc configuration
		meteoradbc.POST("/", handlers.CreateMeteoradbcConfig)

		// Update meteoradbc configuration
		meteoradbc.PUT("/:id", handlers.UpdateMeteoradbcConfig)

		// Update meteoradbc configuration status
		meteoradbc.PATCH("/:id/status", handlers.UpdateMeteoradbcConfigStatus)

		// Delete meteoradbc configuration
		meteoradbc.DELETE("/:id", handlers.DeleteMeteoradbcConfig)
	}
}
