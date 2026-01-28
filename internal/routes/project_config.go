package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupProjectConfigRoutes sets up all routes related to Project Config management
func SetupProjectConfigRoutes(r *gin.Engine) {
	project := r.Group("/project-config")
	{
		project.GET("", handlers.ListProjectConfigs)
		project.GET("/slice", handlers.ListProjectConfigsBySlice)
		project.GET("/latest", handlers.GetLatestProjectConfig)
		project.GET("/latest/active", handlers.GetLatestActiveProjectConfig)
		project.GET("/:id", handlers.GetProjectConfig)
		project.POST("", handlers.CreateProjectConfig)
		project.PUT("/:id", handlers.UpdateProjectConfig)
		project.DELETE("/:id", handlers.DeleteProjectConfig)
		project.GET("/address/count/:project_id", handlers.GetAddressCountByProjectID)
		project.POST("/auto-create-pumpfuninternal", handlers.AutoCreatePumpfuninternalProject)
		project.POST("/auto-create-pumpfunamm", handlers.AutoCreatePumpfunAmmProject)
		project.POST("/auto-create-meteoradbc", handlers.AutoCreateMeteoradbcProject)
		project.POST("/auto-create-meteoradbc-v2", handlers.AutoCreateMeteoradbcProjectV2)
		project.POST("/refill-token-metadata-id", handlers.RefillTokenMetadataID)
		project.POST("/update-assets-balance", handlers.UpdateAssetsBalance)
		project.POST("/update-vesting", handlers.UpdateVesting)
		project.POST("/toggle/:id", handlers.ToggleProjectConfigLocker)
	}
}

// SetupProjectTransferRoutes sets up all routes related to Project Fund Transfer Record management
func SetupProjectTransferRoutes(r *gin.Engine) {
	transfer := r.Group("/project-transfer")
	{
		transfer.GET("", handlers.ListProjectFundTransferRecords)
		transfer.GET("/:id", handlers.GetProjectFundTransferRecord)
		transfer.POST("", handlers.CreateProjectFundTransferRecord)
		transfer.PUT("/:id", handlers.UpdateProjectFundTransferRecord)
		transfer.DELETE("/:id", handlers.DeleteProjectFundTransferRecord)
		transfer.GET("/project/:project_id", handlers.GetProjectFundTransferRecordsByProjectID)
		transfer.GET("/project/initial-sol/:project_id", handlers.GetProjectInitialSol)
	}
}

// SetupProjectExtraAddressRoutes sets up all routes related to Project Extra Address management
func SetupProjectExtraAddressRoutes(r *gin.Engine) {
	extraAddress := r.Group("/project-extra-address")
	{
		extraAddress.GET("", handlers.ListProjectExtraAddresses)
		extraAddress.GET("/:id", handlers.GetProjectExtraAddress)
		extraAddress.POST("", handlers.CreateProjectExtraAddress)
		extraAddress.PUT("/:id", handlers.UpdateProjectExtraAddress)
		extraAddress.DELETE("/:id", handlers.DeleteProjectExtraAddress)
		extraAddress.GET("/project/:project_id", handlers.GetProjectExtraAddressesByProjectID)
	}
}
