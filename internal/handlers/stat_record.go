package handlers

import (
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/internal/models"
)

// ProjectSettleRecordFilterRequest 项目结算记录筛选请求
type ProjectSettleRecordFilterRequest struct {
	ProjectID      uint     `json:"project_id" binding:"required"`
	StartTimestamp int64    `json:"start_timestamp" binding:"required"`
	EndTimestamp   int64    `json:"end_timestamp" binding:"required"`
	Fields         []string `json:"fields" binding:"required"`
}

// ProjectSettleRecordFilterResponse 项目结算记录筛选响应
type ProjectSettleRecordFilterResponse map[string]interface{}

// ListProjectSettleRecords 获取项目结算记录列表
func ListProjectSettleRecords(c *gin.Context) {
	var records []models.ProjectSettleRecord
	if err := dbconfig.DB.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

// GetProjectSettleRecord 获取指定ID的项目结算记录
func GetProjectSettleRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var record models.ProjectSettleRecord
	if err := dbconfig.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, record)
}

// GetProjectSettleRecordsByProjectID 获取指定项目的结算记录
func GetProjectSettleRecordsByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var records []models.ProjectSettleRecord
	if err := dbconfig.DB.Where("project_id = ?", projectID).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

// GetLatestProjectSettleRecord 获取指定项目的最新结算记录
func GetLatestProjectSettleRecord(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var record models.ProjectSettleRecord
	if err := dbconfig.DB.Where("project_id = ?", projectID).
		Order("created_at DESC").
		First(&record).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// 创建响应结构
	response := make(map[string]interface{})
	recordValue := reflect.ValueOf(record)
	recordType := reflect.TypeOf(record)

	// 遍历所有字段
	for i := 0; i < recordValue.NumField(); i++ {
		field := recordType.Field(i)
		value := recordValue.Field(i)

		// 检查是否是时间字段
		if value.Type() == reflect.TypeOf(time.Time{}) {
			// 转换时间为毫秒级时间戳
			timeValue := value.Interface().(time.Time)
			response[field.Tag.Get("json")] = timeValue.UnixMilli()
		} else {
			// 其他字段保持原值
			response[field.Tag.Get("json")] = value.Interface()
		}
	}

	// 新增 tvl_by_project_sol 字段
	tvlByProject, ok1 := response["tvl_by_project"].(float64)
	tvlByProjectToken, ok2 := response["tvl_by_project_token"].(float64)
	if ok1 && ok2 {
		response["tvl_by_project_sol"] = tvlByProject - tvlByProjectToken
	} else {
		response["tvl_by_project_sol"] = nil
	}

	c.JSON(http.StatusOK, response)
}

// GetProjectSettleRecordListsByFilter 根据时间范围和字段筛选项目结算记录
func GetProjectSettleRecordListsByFilter(c *gin.Context) {
	var request ProjectSettleRecordFilterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证时间戳范围
	if request.EndTimestamp <= request.StartTimestamp {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_timestamp must be greater than start_timestamp"})
		return
	}

	// 转换时间戳为时间
	startTime := time.Unix(request.StartTimestamp/1000, 0)
	endTime := time.Unix(request.EndTimestamp/1000, 0)

	// 查询数据
	var records []models.ProjectSettleRecord
	query := dbconfig.DB.Where("project_id = ? AND created_at_by_zero_sec BETWEEN ? AND ?",
		request.ProjectID, startTime, endTime)

	if err := query.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构建响应
	response := ProjectSettleRecordFilterResponse{
		"created_at_by_zero_sec": make([]int64, len(records)),
	}

	// 初始化请求的字段切片
	for _, field := range request.Fields {
		response[field] = make([]float64, len(records))
	}

	// 填充数据
	for i, record := range records {
		// 总是包含时间戳
		response["created_at_by_zero_sec"].([]int64)[i] = record.CreatedAtByZeroSec.UnixMilli()

		// 填充请求的字段
		recordValue := reflect.ValueOf(record)
		for _, field := range request.Fields {
			// 获取字段值
			fieldValue := recordValue.FieldByName(field)
			if fieldValue.IsValid() && fieldValue.CanInterface() {
				// 确保字段存在且可以获取值
				if fieldValue.Kind() == reflect.Float64 {
					response[field].([]float64)[i] = fieldValue.Float()
				}
			}
		}
	}

	c.JSON(http.StatusOK, response)
}