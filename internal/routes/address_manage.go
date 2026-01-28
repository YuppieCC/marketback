package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupAddressManageRoutes sets up all routes related to Address Management
func SetupAddressManageRoutes(r *gin.Engine) {
	address := r.Group("/address-manage")
	{
		address.GET("", handlers.ListAddresses)
		address.GET("/:address", handlers.GetAddress)
		address.GET("/role/:role_id", handlers.ListAddressesByRole)
		address.POST("/generate", handlers.GenerateAddresses)
		address.DELETE("/:id", handlers.DeleteAddress)
		address.POST("/decrypt", handlers.DecryptPrivateKey)
		address.POST("/export-with-new-password", handlers.ExportWithNewPassword)
		address.POST("/export-with-new-password/role/:rold_id", handlers.ExportWithNewPasswordFromRole)
		address.POST("/export-with-gmgn-track-format/role/:role_id", handlers.ExportWithGmgnTrackFormatFromRole)
		address.POST("/import-and-verify-password", handlers.ImportAndVerifyPassword)
		address.GET("/review-by-role-count", handlers.ReviewAddressesByRoleCount)
		address.POST("/review-by-token-stat", handlers.ReviewAddressesByTokenStat)
		address.POST("/check-exists", handlers.CheckAddressExists)
		address.POST("/multi-transfer-sol", handlers.MultiTransferSol)
		address.POST("/import-csv", handlers.ImportCsv)
	}

	// Address Config routes
	addressConfig := r.Group("/address-config")
	{
		addressConfig.POST("", handlers.CreateAddressConfig)
		addressConfig.GET("", handlers.ListAddressConfigs)
		addressConfig.GET("/id/:id", handlers.GetAddressConfig)
		addressConfig.PUT("/id/:id", handlers.UpdateAddressConfig)
		addressConfig.DELETE("/id/:id", handlers.DeleteAddressConfig)
		addressConfig.GET("/role/:role_id", handlers.ListAddressConfigByRole)
		addressConfig.GET("/by-address-mint/:address/:mint", handlers.GetAddressConfigByAddressAndMint)
		addressConfig.POST("/filter", handlers.GetAddressConfigByFilter)
		addressConfig.POST("/create-or-update", handlers.CreateOrUpdateAddressConfig)
	}

	// Disposable Address Manage routes
	disposableAddress := r.Group("/disposable-address-manage")
	{
		disposableAddress.GET("", handlers.ListDisposableAddresses)
		disposableAddress.GET("/:id", handlers.GetDisposableAddress)
		disposableAddress.POST("", handlers.CreateDisposableAddress)
		disposableAddress.POST("/generate", handlers.GenerateDisposableAddresses)
		disposableAddress.POST("/get-and-replace", handlers.GetAndReplaceDisposableAddress)
		disposableAddress.PUT("/:id", handlers.UpdateDisposableAddress)
		disposableAddress.DELETE("/:id", handlers.DeleteDisposableAddress)
		disposableAddress.POST("/export-with-new-password", handlers.ExportWithNewPasswordInDisposableAddressManage)
		disposableAddress.POST("/import-and-verify-password", handlers.ImportAndVerifyPasswordInDisposableAddressManage)
		disposableAddress.POST("/import-csv", handlers.ImportCsvInDisposableAddressManage)
	}
}
