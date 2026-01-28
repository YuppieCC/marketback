package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupPumpfuninternalConfigRoutes sets up all routes related to Pumpfuninternal Config management
func SetupPumpfuninternalConfigRoutes(r *gin.Engine) {
	pumpfun := r.Group("/pumpfuninternal-config")
	{
		pumpfun.GET("", handlers.ListPumpfuninternalConfigs)
		pumpfun.GET("/slice", handlers.ListPumpfuninternalConfigsBySlice)
		pumpfun.GET("/:id", handlers.GetPumpfuninternalConfig)
		pumpfun.GET("/mint/:mint", handlers.GetPumpfuninternalConfigByMint)
		pumpfun.POST("", handlers.CreatePumpfuninternalConfig)
		// pumpfun.PUT("/:id", handlers.UpdatePumpfuninternalConfig)
		pumpfun.PATCH("/:id/status", handlers.UpdatePumpfuninternalConfigStatus)
		pumpfun.DELETE("/:id", handlers.DeletePumpfuninternalConfig)
	}
} 