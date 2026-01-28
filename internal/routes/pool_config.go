package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupPoolConfigRoutes sets up all routes related to Pool Config management
func SetupPoolConfigRoutes(r *gin.Engine) {
	pool := r.Group("/pool-config")
	{
		pool.GET("", handlers.ListPoolConfigs)
		pool.GET("/:id", handlers.GetPoolConfig)
		pool.POST("", handlers.CreatePoolConfig)
		pool.PUT("/:id", handlers.UpdatePoolConfig)
		pool.DELETE("/:id", handlers.DeletePoolConfig)
	}
}
