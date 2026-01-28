package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupBlockchainConfigRoutes sets up all routes related to Blockchain Config management
func SetupBlockchainConfigRoutes(r *gin.Engine) {
	blockchain := r.Group("/blockchain-config")
	{
		blockchain.GET("", handlers.ListBlockchainConfigs)
		blockchain.GET("/:id", handlers.GetBlockchainConfig)
		blockchain.POST("", handlers.CreateBlockchainConfig)
		blockchain.PUT("/:id", handlers.UpdateBlockchainConfig)
		blockchain.DELETE("/:id", handlers.DeleteBlockchainConfig)
	}
} 