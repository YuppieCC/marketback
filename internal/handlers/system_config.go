package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

// CreateSystemLogRequest represents the request payload for creating a system log
type CreateSystemLogRequest struct {
	ProjectID  *uint           `json:"project_id"`
	Level      string          `json:"level" binding:"required"`   // DEBUG, INFO, WARN, ERROR, FATAL
	Message    string          `json:"message" binding:"required"` // log body
	Module     string          `json:"module"`                     // e.g. "auth", "payment"
	ErrorStack string          `json:"error_stack"`
	Meta       json.RawMessage `json:"meta"` // optional json payload
}

// ListSystemLogs returns paginated system logs with optional filters
func ListSystemLogs(c *gin.Context) {
	// Parse query parameters
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 10
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	orderField := "id"
	if of := c.Query("order_field"); of != "" {
		valid := map[string]bool{
			"id": true, "project_id": true, "level": true, "created_at": true,
		}
		if valid[of] {
			orderField = of
		}
	}
	orderType := "desc"
	if ot := c.Query("order_type"); ot == "asc" || ot == "desc" {
		orderType = ot
	}

	var query = dbconfig.DB.Model(&models.SystemLog{})
	// Filters
	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", level)
	}
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}
	if pid := c.Query("project_id"); pid != "" {
		if parsed, err := strconv.Atoi(pid); err == nil {
			query = query.Where("project_id = ?", parsed)
		}
	}

	// Get total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * pageSize

	var logs []models.SystemLog
	if err := query.Order(orderField + " " + orderType).
		Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	})
}

// GetSystemLog returns a specific system log by ID
func GetSystemLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var log models.SystemLog
	if err := dbconfig.DB.First(&log, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, log)
}

// CreateSystemLog creates a new system log
func CreateSystemLog(c *gin.Context) {
	var req CreateSystemLogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var metaMap map[string]interface{}
	if len(req.Meta) > 0 {
		_ = json.Unmarshal(req.Meta, &metaMap)
	}

	projectID := uint(0)
	if req.ProjectID != nil {
		projectID = *req.ProjectID
	}

	log := models.SystemLog{
		ProjectID:  projectID,
		Level:      req.Level,
		Message:    req.Message,
		Module:     req.Module,
		ErrorStack: req.ErrorStack,
		Meta:       models.JSONMap(metaMap),
	}

	if err := dbconfig.DB.Create(&log).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, log)
}

// DeleteSystemLog deletes a system log by ID
func DeleteSystemLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	if err := dbconfig.DB.Delete(&models.SystemLog{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "System log deleted successfully"})
}

// ListSystemLogsByProject lists logs filtered by project_id with pagination
func ListSystemLogsByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Reuse pagination/order query params
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 10
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	orderField := "id"
	if of := c.Query("order_field"); of != "" {
		valid := map[string]bool{
			"id": true, "project_id": true, "level": true, "created_at": true,
		}
		if valid[of] {
			orderField = of
		}
	}
	orderType := "desc"
	if ot := c.Query("order_type"); ot == "asc" || ot == "desc" {
		orderType = ot
	}

	var query = dbconfig.DB.Model(&models.SystemLog{}).Where("project_id = ?", projectID)
	// Optional filters still allowed
	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", level)
	}
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * pageSize
	var logs []models.SystemLog
	if err := query.Order(orderField + " " + orderType).
		Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	})
}

// CreateSystemParamsRequest represents the request payload for creating system params
type CreateSystemParamsRequest struct {
	Name         string          `json:"name" binding:"required"`
	IsActive     *bool           `json:"is_active"`
	PresetName   string          `json:"preset_name"`
	ParamsConfig json.RawMessage `json:"params_config"` // optional json payload
}

// UpdateSystemParamsRequest represents the request payload for updating system params
type UpdateSystemParamsRequest struct {
	IsActive     *bool           `json:"is_active"`
	PresetName   *string         `json:"preset_name"`
	ParamsConfig json.RawMessage `json:"params_config"`
}

// ListSystemParams returns paginated system params with optional filters
func ListSystemParams(c *gin.Context) {
	// Parse query parameters
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 10
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	orderField := "id"
	if of := c.Query("order_field"); of != "" {
		valid := map[string]bool{
			"id": true, "name": true, "preset_id": true, "is_active": true, "created_at": true, "updated_at": true,
		}
		if valid[of] {
			orderField = of
		}
	}
	orderType := "desc"
	if ot := c.Query("order_type"); ot == "asc" || ot == "desc" {
		orderType = ot
	}

	var query = dbconfig.DB.Model(&models.SystemParams{})
	// Filters
	if name := c.Query("name"); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if isActive := c.Query("is_active"); isActive != "" {
		if parsed, err := strconv.ParseBool(isActive); err == nil {
			query = query.Where("is_active = ?", parsed)
		}
	}
	if presetID := c.Query("preset_id"); presetID != "" {
		if parsed, err := strconv.Atoi(presetID); err == nil {
			query = query.Where("preset_id = ?", parsed)
		}
	}
	if presetName := c.Query("preset_name"); presetName != "" {
		query = query.Where("preset_name LIKE ?", "%"+presetName+"%")
	}

	// Get total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * pageSize

	var params []models.SystemParams
	if err := query.Order(orderField + " " + orderType).
		Offset(offset).Limit(pageSize).Find(&params).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	c.JSON(http.StatusOK, gin.H{
		"data": params,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	})
}

// GetSystemParamsByName returns all system params filtered by name (no pagination)
func GetSystemParamsByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name parameter is required"})
		return
	}

	var params []models.SystemParams
	if err := dbconfig.DB.Where("name = ?", name).Find(&params).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, params)
}

// GetSystemParamsByPresetName returns a system params by preset_name
// If multiple records exist, returns the one with the smallest ID
func GetSystemParamsByPresetName(c *gin.Context) {
	presetName := c.Param("name")
	if presetName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Preset name parameter is required"})
		return
	}

	var param models.SystemParams
	if err := dbconfig.DB.
		Where("preset_name = ?", presetName).
		Order("id asc").
		First(&param).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, param)
}

// GetSystemParams returns a specific system params by ID
func GetSystemParams(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var param models.SystemParams
	if err := dbconfig.DB.First(&param, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, param)
}

// CreateSystemParams creates a new system params
func CreateSystemParams(c *gin.Context) {
	var req CreateSystemParamsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var paramsConfigMap map[string]interface{}
	if len(req.ParamsConfig) > 0 {
		if err := json.Unmarshal(req.ParamsConfig, &paramsConfigMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid params_config JSON format"})
			return
		}
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Calculate PresetID: find max preset_id for the same name, then +1
	var maxPresetID uint
	if err := dbconfig.DB.Model(&models.SystemParams{}).
		Where("name = ?", req.Name).
		Select("COALESCE(MAX(preset_id), 0)").
		Scan(&maxPresetID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate preset_id: " + err.Error()})
		return
	}
	newPresetID := maxPresetID + 1

	param := models.SystemParams{
		Name:         req.Name,
		IsActive:     isActive,
		PresetID:     newPresetID,
		PresetName:   req.PresetName,
		ParamsConfig: models.JSONMap(paramsConfigMap),
	}

	if err := dbconfig.DB.Create(&param).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, param)
}

// UpdateSystemParams updates an existing system params
func UpdateSystemParams(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req UpdateSystemParamsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var param models.SystemParams
	if err := dbconfig.DB.First(&param, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Update fields if provided
	// Note: Name and PresetID cannot be updated after creation
	if req.IsActive != nil {
		param.IsActive = *req.IsActive
	}
	if req.PresetName != nil {
		param.PresetName = *req.PresetName
	}
	if len(req.ParamsConfig) > 0 {
		var paramsConfigMap map[string]interface{}
		if err := json.Unmarshal(req.ParamsConfig, &paramsConfigMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid params_config JSON format"})
			return
		}
		param.ParamsConfig = models.JSONMap(paramsConfigMap)
	}

	if err := dbconfig.DB.Save(&param).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, param)
}

// DeleteSystemParams deletes a system params by ID
func DeleteSystemParams(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	if err := dbconfig.DB.Delete(&models.SystemParams{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "System params deleted successfully"})
}

// CreateSystemCommandRequest represents the request payload for creating a system command
type CreateSystemCommandRequest struct {
	IsActive      *bool           `json:"is_active"`
	ProjectID     *uint           `json:"project_id"`
	CommandName   string          `json:"command_name"`
	CommandParams json.RawMessage `json:"command_params"`
	VerifyParams  json.RawMessage `json:"verify_params"`
	IsSuccess     *bool           `json:"is_success"`
}

// UpdateSystemCommandRequest represents the request payload for updating a system command
type UpdateSystemCommandRequest struct {
	IsActive      *bool           `json:"is_active"`
	ProjectID     *uint           `json:"project_id"`
	CommandName   *string         `json:"command_name"`
	CommandParams json.RawMessage `json:"command_params"`
	VerifyParams  json.RawMessage `json:"verify_params"`
	IsSuccess     *bool           `json:"is_success"`
}

// ListSystemCommands returns paginated system commands with optional filters
func ListSystemCommands(c *gin.Context) {
	// Parse query parameters
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 10
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	orderField := "id"
	if of := c.Query("order_field"); of != "" {
		valid := map[string]bool{
			"id": true, "project_id": true, "command_name": true, "is_active": true, "is_success": true, "created_at": true, "updated_at": true,
		}
		if valid[of] {
			orderField = of
		}
	}
	orderType := "desc"
	if ot := c.Query("order_type"); ot == "asc" || ot == "desc" {
		orderType = ot
	}

	var query = dbconfig.DB.Model(&models.SystemCommand{})
	// Filters
	if commandName := c.Query("command_name"); commandName != "" {
		query = query.Where("command_name LIKE ?", "%"+commandName+"%")
	}
	if isActive := c.Query("is_active"); isActive != "" {
		if parsed, err := strconv.ParseBool(isActive); err == nil {
			query = query.Where("is_active = ?", parsed)
		}
	}
	if isSuccess := c.Query("is_success"); isSuccess != "" {
		if parsed, err := strconv.ParseBool(isSuccess); err == nil {
			query = query.Where("is_success = ?", parsed)
		}
	}
	if projectID := c.Query("project_id"); projectID != "" {
		if parsed, err := strconv.Atoi(projectID); err == nil {
			query = query.Where("project_id = ?", parsed)
		}
	}

	// Get total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * pageSize

	var commands []models.SystemCommand
	if err := query.Order(orderField + " " + orderType).
		Offset(offset).Limit(pageSize).Find(&commands).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	c.JSON(http.StatusOK, gin.H{
		"data": commands,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	})
}

// GetLatestSystemCommand returns the latest system command (highest ID)
func GetLatestSystemCommand(c *gin.Context) {
	var command models.SystemCommand
	if err := dbconfig.DB.Order("id DESC").First(&command).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No system command found"})
		return
	}
	c.JSON(http.StatusOK, command)
}

// ListSystemCommandsByProject lists commands filtered by project_id with pagination
func ListSystemCommandsByProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Reuse pagination/order query params
	page := 1
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	pageSize := 10
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}
	orderField := "id"
	if of := c.Query("order_field"); of != "" {
		valid := map[string]bool{
			"id": true, "project_id": true, "command_name": true, "is_active": true, "is_success": true, "created_at": true, "updated_at": true,
		}
		if valid[of] {
			orderField = of
		}
	}
	orderType := "desc"
	if ot := c.Query("order_type"); ot == "asc" || ot == "desc" {
		orderType = ot
	}

	var query = dbconfig.DB.Model(&models.SystemCommand{}).Where("project_id = ?", projectID)
	// Optional filters still allowed
	if commandName := c.Query("command_name"); commandName != "" {
		query = query.Where("command_name LIKE ?", "%"+commandName+"%")
	}
	if isActive := c.Query("is_active"); isActive != "" {
		if parsed, err := strconv.ParseBool(isActive); err == nil {
			query = query.Where("is_active = ?", parsed)
		}
	}
	if isSuccess := c.Query("is_success"); isSuccess != "" {
		if parsed, err := strconv.ParseBool(isSuccess); err == nil {
			query = query.Where("is_success = ?", parsed)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	offset := (page - 1) * pageSize
	var commands []models.SystemCommand
	if err := query.Order(orderField + " " + orderType).
		Offset(offset).Limit(pageSize).Find(&commands).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	c.JSON(http.StatusOK, gin.H{
		"data": commands,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	})
}

// GetSystemCommand returns a specific system command by ID
func GetSystemCommand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	var command models.SystemCommand
	if err := dbconfig.DB.First(&command, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, command)
}

// CreateSystemCommand creates a new system command
func CreateSystemCommand(c *gin.Context) {
	var req CreateSystemCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var commandParamsMap map[string]interface{}
	if len(req.CommandParams) > 0 {
		if err := json.Unmarshal(req.CommandParams, &commandParamsMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command_params JSON format"})
			return
		}
	}

	var verifyParamsMap map[string]interface{}
	if len(req.VerifyParams) > 0 {
		if err := json.Unmarshal(req.VerifyParams, &verifyParamsMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verify_params JSON format"})
			return
		}
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	projectID := uint(0)
	if req.ProjectID != nil {
		projectID = *req.ProjectID
	}

	isSuccess := false
	if req.IsSuccess != nil {
		isSuccess = *req.IsSuccess
	}

	command := models.SystemCommand{
		IsActive:      isActive,
		ProjectID:     projectID,
		CommandName:   req.CommandName,
		CommandParams: models.JSONMap(commandParamsMap),
		VerifyParams:  models.JSONMap(verifyParamsMap),
		IsSuccess:     isSuccess,
	}

	if err := dbconfig.DB.Create(&command).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, command)
}

// UpdateSystemCommand updates an existing system command
func UpdateSystemCommand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var req UpdateSystemCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var command models.SystemCommand
	if err := dbconfig.DB.First(&command, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Update fields if provided
	if req.IsActive != nil {
		command.IsActive = *req.IsActive
	}
	if req.ProjectID != nil {
		command.ProjectID = *req.ProjectID
	}
	if req.CommandName != nil {
		command.CommandName = *req.CommandName
	}
	if len(req.CommandParams) > 0 {
		var commandParamsMap map[string]interface{}
		if err := json.Unmarshal(req.CommandParams, &commandParamsMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid command_params JSON format"})
			return
		}
		command.CommandParams = models.JSONMap(commandParamsMap)
	}
	if len(req.VerifyParams) > 0 {
		var verifyParamsMap map[string]interface{}
		if err := json.Unmarshal(req.VerifyParams, &verifyParamsMap); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verify_params JSON format"})
			return
		}
		command.VerifyParams = models.JSONMap(verifyParamsMap)
	}
	if req.IsSuccess != nil {
		command.IsSuccess = *req.IsSuccess
	}

	if err := dbconfig.DB.Save(&command).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, command)
}

// DeleteSystemCommand deletes a system command by ID
func DeleteSystemCommand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}
	if err := dbconfig.DB.Delete(&models.SystemCommand{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "System command deleted successfully"})
}
