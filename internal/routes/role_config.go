package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoleConfigRoutes sets up all routes related to Role Config and Role Address management
func SetupRoleConfigRoutes(r *gin.Engine) {
	role := r.Group("/role-config")
	{
		role.GET("", handlers.ListRoleConfigs)
		role.GET("/:id", handlers.GetRoleConfig)
		role.POST("", handlers.CreateRoleConfig)
		role.PUT("/:id", handlers.UpdateRoleConfig)
		role.DELETE("/:id", handlers.DeleteRoleConfig)
		role.GET("/by-project/:project_id", handlers.GetRoleConfigByProjectID)
		role.DELETE("/with-address/:role_id", handlers.DeleteRoleConfigWithAddressByRoleID)
		role.POST("/by-template", handlers.CreateRoleConfigByTemplateID)

	}

	roleAddr := r.Group("/role-address")
	{
		roleAddr.GET("", handlers.ListRoleAddresses)
		roleAddr.GET("/:id", handlers.GetRoleAddress)
		roleAddr.POST("", handlers.CreateRoleAddress)
		roleAddr.DELETE("/:id", handlers.DeleteRoleAddress)
		roleAddr.GET("/by-role/:role_id", handlers.GetRoleAddressByRoleID)
		roleAddr.POST("/batch", handlers.BatchCreateRoleAddress)
		roleAddr.DELETE("/by-role/:role_id", handlers.DeleteRoleAddressByRoleID)
		roleAddr.GET("/count/:role_id", handlers.GetRoleAddressCountByRoleID)
		roleAddr.GET("/total/:role_id", handlers.GetTotalRoleAddressByRoleID)
		roleAddr.POST("/batch-add", handlers.BatchAddRoleAddress)
		roleAddr.POST("/batch-delete", handlers.BatchDeleteRoleAddress)
		roleAddr.GET("/export/:role_id", handlers.ExportAddressByRoleID)
		roleAddr.POST("/transfer-mint", handlers.TransferMintToTargetByRole)
		roleAddr.GET("/sol-balances/:role_id", handlers.GetRoleAddressSolBalances)
		roleAddr.POST("/check-exists", handlers.CheckRoleAddressExist)
		roleAddr.POST("/safe-delete", handlers.SafeDeleteAddressByRole)
		roleAddr.POST("/select-random-roleaddress-transfer", handlers.SelectRandomRoleAddressTransfer)
	}

	roleConfigRelation := r.Group("/role-config-relation")
	{
		roleConfigRelation.POST("/create", handlers.CreateRoleConfigRelation)
		roleConfigRelation.DELETE("/delete/:id", handlers.DeleteRoleConfigRelation)
		roleConfigRelation.POST("/migrate", handlers.MigrateAllRoleConfig)
		roleConfigRelation.DELETE("/delete-by-filter", handlers.DeleteRoleConfigRelationByFilter)
		roleConfigRelation.GET("/by-role/:role_id", handlers.GetRoleConfigRelationByRoleID)
		roleConfigRelation.GET("/by-project/:project_id", handlers.GetRoleConfigRelationByProjectID)
		roleConfigRelation.GET("/lists", handlers.ListRoleConfigRelations)
	}
}
