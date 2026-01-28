package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupRpcConfigRoutes sets up all routes related to RPC Config management
func SetupRpcConfigRoutes(r *gin.Engine) {
	rpc := r.Group("/rpc-config")
	{
		rpc.GET("", handlers.ListRpcConfigs)
		rpc.GET("/:id", handlers.GetRpcConfig)
		rpc.POST("", handlers.CreateRpcConfig)
		rpc.PUT("/:id", handlers.UpdateRpcConfig)
		rpc.DELETE("/:id", handlers.DeleteRpcConfig)
	}
} 