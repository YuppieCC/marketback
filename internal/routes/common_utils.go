package routes

import (
	"marketcontrol/internal/handlers"
	"marketcontrol/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupCommonUtilsRoutes(r *gin.Engine) {
	amm := r.Group("/common_utils/constant_product_amm")
	{
		amm.POST("/simulate_amount_out", handlers.SimulateAmountOutHandler)
		amm.POST("/simulate_amount_in", handlers.SimulateAmountInHandler)
	}

	bondingCurve := r.Group("/common_utils/bonding_curve")
	{
		bondingCurve.POST("/estimate_buy_cost", handlers.EstimateBuyCostWithIncreaseHandler)
		bondingCurve.POST("/estimate_sell_return", handlers.EstimateSellReturnWithDecreaseHandler)
		bondingCurve.POST("/simulate_amount_out", handlers.SimulateBondingCurveAmountOutHandler)
		bondingCurve.POST("/simulate_amount_in", handlers.SimulateBondingCurveAmountInHandler)
		bondingCurve.POST("/get_virtual_reserves", handlers.GetVirtualReservesHandler)
	}

	helius := r.Group("/common_utils/helius")
	{
		helius.POST("/transactions", handlers.GetEnhancedTransactionsByAddressHandler)
		helius.POST("/transactions_without_address", handlers.GetEnhancedTransactionsWithoutAddressHandler)
		helius.POST("/get_token_supply", handlers.GetTokenSupplyHandler)
	}

	token := r.Group("/common_utils/token")
	{
		token.POST("/info", handlers.GetTokenInfoHandler)
		token.POST("/das-info", handlers.GetDasInfoHandler)
	}

	raydium := r.Group("/common_utils/raydium")
	{
		raydium.POST("/launchpad-cpmm-pool-id", handlers.GetLaunchpadAndCpmmId)
		raydium.POST("/launchpad-pool-info", handlers.GetLaunchpadPoolInfo)
		raydium.POST("/launchpad-pool-config", handlers.GetLaunchpadPoolConfig)
		raydium.POST("/cpmm-pool-info", handlers.GetCpmmPoolInfo)
	}

	pumpfun := r.Group("/common_utils/pumpfun")
	{
		pumpfun.POST("/pda", handlers.GetPumpFunPDAHandler)
		pumpfun.POST("/pumpswap-pda", handlers.GetPumpSwapPDAHandler)
	}

	jupiter := r.Group("/common_utils/jupiter")
	{
		jupiter.POST("/swap", handlers.GetJupiterSwapResult)
		jupiter.POST("/price", handlers.GetTokenPrice)
	}

	account := r.Group("/common_utils/account")
	{
		account.POST("/info", handlers.GetAccountInfo)
		account.POST("/multi-account-info", handlers.GetMultiAccountsInfo)
	}

	websocket := r.Group("/common_utils/websocket")
	{
		websocket.POST("/pool-monitor", handlers.ControlPoolMonitor)
	}

	// RPC status check endpoint with rate limiting
	// Limit: 0.5 requests per second per IP (1 request every 2 seconds)
	// This ensures the endpoint doesn't consume too much server capacity
	rpcStatusGroup := r.Group("/common_utils")
	rpcStatusGroup.Use(middleware.RateLimiterMiddleware(middleware.RateLimiterConfig{
		RequestsPerSecond: 0.5, // 1 request every 2 seconds
		Burst:             1,   // Allow 1 burst request
	}))
	rpcStatusGroup.POST("/rps-status", handlers.GetRPCStatusHandler)

	vpnController := r.Group("/common_utils/vpn")
	{
		vpnController.POST("/proxies", handlers.VpnControllerGetProxies)
		vpnController.PUT("/change", handlers.VpnControllerChangeProxy)
	}
}
