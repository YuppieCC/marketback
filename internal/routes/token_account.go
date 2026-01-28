package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupTokenAccountRoutes sets up all routes related to Token Account management
func SetupTokenAccountRoutes(r *gin.Engine) {
	tokenAccount := r.Group("/token-account")
	{
		tokenAccount.GET("", handlers.ListTokenAccounts)
		tokenAccount.GET(":id", handlers.GetTokenAccount)
		tokenAccount.POST("", handlers.CreateTokenAccount)
		tokenAccount.PUT(":id", handlers.UpdateTokenAccount)
		tokenAccount.DELETE(":id", handlers.DeleteTokenAccount)
	}
} 