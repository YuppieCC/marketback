package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupRaydiumRoutes sets up the Raydium-related routes
func SetupRaydiumRoutes(r *gin.Engine) {
	// Setup raydium routes
	raydiumGroup := r.Group("/raydium")
	{
		// Launchpad Pool Config routes
		launchpadPoolConfigRoutes := raydiumGroup.Group("/launchpad-pool-config")
		{
			launchpadPoolConfigRoutes.GET("", handlers.ListRaydiumLaunchpadPoolConfigs)
			launchpadPoolConfigRoutes.POST("", handlers.CreateRaydiumLaunchpadPoolConfig)
			launchpadPoolConfigRoutes.POST("/create/by-mint", handlers.CreateRaydiumLaunchpadPoolConfigByMint)
			launchpadPoolConfigRoutes.GET("/:id", handlers.GetRaydiumLaunchpadPoolConfig)
			launchpadPoolConfigRoutes.PUT("/:id", handlers.UpdateRaydiumLaunchpadPoolConfig)
			launchpadPoolConfigRoutes.DELETE("/:id", handlers.DeleteRaydiumLaunchpadPoolConfig)
			launchpadPoolConfigRoutes.GET("/by-pool-address/:pool_address", handlers.GetRaydiumLaunchpadPoolConfigByPoolAddress)
		}

		// CPMM Pool Config routes
		cpmmPoolConfigRoutes := raydiumGroup.Group("/cpmm-pool-config")
		{
			cpmmPoolConfigRoutes.GET("", handlers.ListRaydiumCpmmPoolConfigs)
			cpmmPoolConfigRoutes.POST("", handlers.CreateRaydiumCpmmPoolConfig)
			cpmmPoolConfigRoutes.GET("/:id", handlers.GetRaydiumCpmmPoolConfig)
			cpmmPoolConfigRoutes.PUT("/:id", handlers.UpdateRaydiumCpmmPoolConfig)
			cpmmPoolConfigRoutes.DELETE("/:id", handlers.DeleteRaydiumCpmmPoolConfig)
			cpmmPoolConfigRoutes.GET("/by-pool-address/:pool_address", handlers.GetRaydiumCpmmPoolConfigByPoolAddress)
		}

		// Pool Relation routes
		poolRelationRoutes := raydiumGroup.Group("/pool-relation")
		{
			poolRelationRoutes.GET("", handlers.ListRaydiumPoolRelations)
			poolRelationRoutes.POST("", handlers.CreateRaydiumPoolRelation)
			poolRelationRoutes.GET("/:id", handlers.GetRaydiumPoolRelation)
			poolRelationRoutes.PUT("/:id", handlers.UpdateRaydiumPoolRelation)
			poolRelationRoutes.DELETE("/:id", handlers.DeleteRaydiumPoolRelation)
			poolRelationRoutes.GET("/by-mints/:mint_a/:mint_b", handlers.GetRaydiumPoolRelationByMints)
		}
	}
} 