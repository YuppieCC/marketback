package routes

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// SetupRoutes initializes and returns the Gin router with all routes configured
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// Add health check endpoint
	r.Any("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// Configure CORS middleware
	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Get allowed origins from environment variable
		// Format: comma-separated list, e.g., "http://localhost:3000,http://localhost:3001"
		allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
		var allowedOrigins []string

		if allowedOriginsStr != "" {
			// Split by comma and trim whitespace
			origins := strings.Split(allowedOriginsStr, ",")
			for _, o := range origins {
				trimmed := strings.TrimSpace(o)
				if trimmed != "" {
					allowedOrigins = append(allowedOrigins, trimmed)
				}
			}
		}

		// Check if the request origin is in the allowed list
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		// Check if origin matches 192.168.*.*:3000 pattern
		// if !allowed && origin != "" {
		// 	if strings.HasPrefix(origin, "http://192.168.") && strings.HasSuffix(origin, ":3000") {
		// 		allowed = true
		// 	}
		// }

		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		// 确保包含所有必要的请求头
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup routes for each module
	SetupRpcConfigRoutes(r)
	SetupBlockchainConfigRoutes(r)
	SetupAddressManageRoutes(r)
	SetupWashMapRoutes(r)
	SetupTokenConfigRoutes(r)
	SetupPoolConfigRoutes(r)
	SetupProjectConfigRoutes(r)
	SetupProjectTransferRoutes(r)
	SetupProjectExtraAddressRoutes(r) // Add project extra address routes
	SetupProjectSettleRoutes(r)
	SetupRoleConfigRoutes(r)
	SetupProjectStatRoutes(r)
	SetupProjectSnapshotRoutes(r)
	SetupTokenAccountRoutes(r)
	SetupCommonUtilsRoutes(r)
	SetupStrategyConfigRoutes(r)
	SetupStrategySignalRoutes(r)
	SetupStrategyTransactionRoutes(r)
	SetupPumpfuninternalConfigRoutes(r)
	SetupTransactionsMonitorConfigRoutes(r)
	SetupPumpfunAmmConfigRoutes(r)
	SetupTemplateRoleConfigRoutes(r)
	SetupProjectSettleRecordRoutes(r)
	SetupRaydiumRoutes(r)
	SetupMeteoradbcConfigRoutes(r)
	SetupMeteoracpmmConfigRoutes(r)
	SetupSystemConfigRoutes(r)

	return r
}
