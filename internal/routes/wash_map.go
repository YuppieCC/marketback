package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupWashMapRoutes 设置洗币图谱相关路由
func SetupWashMapRoutes(r *gin.Engine) {
	washMap := r.Group("/wash-map")
	{
		washMap.GET("/:id", handlers.GetWashMap)
		washMap.GET("/nodes/:id", handlers.ListWashMapNodes)
		washMap.GET("/edges/:id", handlers.ListWashMapEdges)
		washMap.DELETE("/:id", handlers.DeleteWashMap)
		
		washMap.POST("/create", handlers.CreateWashMap)
		washMap.POST("/list", handlers.ListWashMapsByProjectID)  // 新增路由
		washMap.POST("/export", handlers.ExportWashMap)

		washMap.POST("/create-task", handlers.CreateWashTask)  // create tasks and manage
		washMap.POST("/list-task-manage", handlers.ListWashTaskManageByProjectID)
		washMap.POST("/update-task-manage/:WashTaskManageID", handlers.UpdateWashTaskManagByID)

		washMap.GET("/list-tasks/:WashTaskManageID", handlers.GetWashTaskByManageID)
		washMap.POST("/update-task/:WashTaskID", handlers.UpdateTaskByID)
		washMap.GET("/task/:WashTaskID", handlers.GetWashTask)  // Changed path

		// 新增自动创建洗币图谱和任务的路由
		washMap.POST("/with-task/auto", handlers.AutoCreateWashMapWithTask)
		washMap.POST("/with-task/auto-v2", handlers.AutoCreateWashMapWithTaskV2)
	}
}