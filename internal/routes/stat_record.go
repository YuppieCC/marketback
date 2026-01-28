package routes

import (
	"github.com/gin-gonic/gin"
	"marketcontrol/internal/handlers"
)

// SetupProjectSettleRecordRoutes 设置项目结算记录相关路由
func SetupProjectSettleRecordRoutes(r *gin.Engine) {
	v1 := r.Group("/project-settle-records")
	{
		v1.GET("/:project_id", handlers.GetProjectSettleRecordsByProjectID)
		v1.GET("/latest/:project_id", handlers.GetLatestProjectSettleRecord)
		v1.POST("/lists-by-filter", handlers.GetProjectSettleRecordListsByFilter)
	}
}
