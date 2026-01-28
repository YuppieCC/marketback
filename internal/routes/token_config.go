package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupTokenConfigRoutes sets up all routes related to Token Config management
func SetupTokenConfigRoutes(r *gin.Engine) {
	token := r.Group("/token-config")
	{
		token.GET("", handlers.ListTokenConfigs)
		token.GET("/slice", handlers.ListTokenConfigsSlice)
		token.GET("/:id", handlers.GetTokenConfig)
		token.GET("/by-mint/:mint", handlers.GetTokenConfigByMint)
		token.POST("", handlers.CreateTokenConfig)
		token.PUT("/:id", handlers.UpdateTokenConfig)
		token.DELETE("/:id", handlers.DeleteTokenConfig)
	}

	// Token metadata routes
	tokenMetadata := r.Group("/token-metadata")
	{
		tokenMetadata.GET("", handlers.ListTokenMetadata)
		tokenMetadata.GET("/favorites", handlers.GetFavorites)
		tokenMetadata.GET("/:id", handlers.GetTokenMetadata)
		tokenMetadata.GET("/by-symbol/:symbol", handlers.GetTokenMetadataBySymbol)
		tokenMetadata.POST("", handlers.CreateTokenMetadata)
		tokenMetadata.PUT("/:id", handlers.UpdateTokenMetadata)
		tokenMetadata.DELETE("/:id", handlers.DeleteTokenMetadata)
		tokenMetadata.POST("/fetch-metadata-by-url", handlers.FetchTokenMetadataByURL)
		tokenMetadata.POST("/fetch-metadata-by-mint", handlers.FetchTokenMetadataByMint)
		tokenMetadata.POST("/random", handlers.GetRandomTokenMetadata)
	}
}
