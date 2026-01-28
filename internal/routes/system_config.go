package routes

import (
	"marketcontrol/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupSystemConfigRoutes sets up routes for system logs and related system configurations
func SetupSystemConfigRoutes(r *gin.Engine) {
	logs := r.Group("/system-logs")
	{
		logs.GET("", handlers.ListSystemLogs)
		logs.GET("/project/:project_id", handlers.ListSystemLogsByProject)
		logs.GET("/:id", handlers.GetSystemLog)
		logs.POST("", handlers.CreateSystemLog)
		logs.DELETE("/:id", handlers.DeleteSystemLog)
	}

	params := r.Group("/system-params")
	{
		params.GET("", handlers.ListSystemParams)
		params.GET("/name/:name", handlers.GetSystemParamsByName)
		params.GET("/preset_name/:name", handlers.GetSystemParamsByPresetName)
		params.GET("/:id", handlers.GetSystemParams)
		params.POST("", handlers.CreateSystemParams)
		params.PUT("/:id", handlers.UpdateSystemParams)
		params.DELETE("/:id", handlers.DeleteSystemParams)
	}

	commands := r.Group("/system-command")
	{
		commands.GET("", handlers.ListSystemCommands)
		commands.GET("/latest", handlers.GetLatestSystemCommand)
		commands.GET("/project/:project_id", handlers.ListSystemCommandsByProject)
		commands.GET("/:id", handlers.GetSystemCommand)
		commands.POST("", handlers.CreateSystemCommand)
		commands.PUT("/:id", handlers.UpdateSystemCommand)
		commands.DELETE("/:id", handlers.DeleteSystemCommand)
	}
}
