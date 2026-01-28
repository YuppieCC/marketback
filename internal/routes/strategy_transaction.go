package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupStrategyTransactionRoutes sets up all routes related to Strategy Transaction management
func SetupStrategyTransactionRoutes(r *gin.Engine) {
	transaction := r.Group("/strategy-transaction")
	{
		// Standard CRUD operations
		transaction.GET("", handlers.ListStrategyTransactions)
		transaction.GET("/:id", handlers.GetStrategyTransaction)
		transaction.POST("", handlers.CreateStrategyTransaction)
		transaction.PUT("/:id", handlers.UpdateStrategyTransaction)
		transaction.DELETE("/:id", handlers.DeleteStrategyTransaction)
		
		// Filter operations
		transaction.GET("/project/:project_id", handlers.GetStrategyTransactionsByProjectID)
		transaction.GET("/strategy/:strategy_id", handlers.GetStrategyTransactionsByStrategyID)
		transaction.GET("/signal/:signal_id", handlers.GetStrategyTransactionsBySignalID)
		transaction.GET("/wallet/:wallet", handlers.GetStrategyTransactionsByWallet)
	}
} 