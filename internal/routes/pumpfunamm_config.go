package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupPumpfunAmmConfigRoutes sets up routes for managing Pumpfun AMM pool configurations
func SetupPumpfunAmmConfigRoutes(r *gin.Engine) {
	pumpfunAmm := r.Group("/pumpfun-amm")
	{
		pool := pumpfunAmm.Group("/pool")
		{
			pool.POST("", handlers.CreatePumpfunAmmPoolConfig)
			pool.POST("/auto-create", handlers.AutoCreatePumpfunAmmPoolConfig)
			pool.GET("/:id", handlers.GetPumpfunAmmPoolConfig)
			pool.GET("", handlers.ListPumpfunAmmPoolConfigs)
			pool.GET("/slice", handlers.ListPumpfunAmmPoolConfigsBySlice)
			pool.PUT("/:id", handlers.UpdatePumpfunAmmPoolConfig)
			pool.PATCH("/:id/status", handlers.UpdatePumpfunAmmPoolConfigStatus)
			pool.DELETE("/:id", handlers.DeletePumpfunAmmPoolConfig)
		}
	}
} 