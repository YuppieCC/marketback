package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

func SetupTemplateRoleConfigRoutes(r *gin.Engine) {
	templateRoleGroup := r.Group("/template-role-config")
	{
		templateRoleGroup.POST("", handlers.CreateTemplateRoleConfig)
		templateRoleGroup.GET("/:id", handlers.GetTemplateRoleConfig)
		templateRoleGroup.GET("", handlers.ListTemplateRoleConfigs)
		templateRoleGroup.PUT("/:id", handlers.UpdateTemplateRoleConfig)
		templateRoleGroup.DELETE("/:id", handlers.DeleteTemplateRoleConfig)
		templateRoleGroup.POST("/copy/:role_id", handlers.CreateTemplateRoleConfigByCopy)
		templateRoleGroup.POST("/with-address", handlers.CreateTemplateRoleConfigByWithAddress)
	}
} 