package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"

	"github.com/gin-gonic/gin"
)

// ProjectSettleResp represents the response structure for project settle profit ranking
type ProjectSettleResp struct {
	ID                uint                `json:"id"`
	Name              string              `json:"name"`
	PoolPlatform      string              `json:"pool_platform"`
	PoolID            uint                `json:"pool_id"`
	TokenID           uint                `json:"token_id"`
	TokenMetadataID   uint                `json:"token_metadata_id"`
	IsActive          bool                `json:"is_active"`
	UpdateStatEnabled bool                `json:"update_stat_enabled"`
	IsMigrated        bool                `json:"is_migrated"`
	IsLocked          bool                `json:"is_locked"`
	AssetsBalance     float64             `json:"assets_balance"`
	RetailSolAmount   float64             `json:"retail_sol_amount"`
	PoolConfig        string              `json:"pool_config"`
	ProjectProfit     float64             `json:"project_profit"`
	CreatedAt         string              `json:"created_at"`
	UpdatedAt         string              `json:"updated_at"`
	Token             *models.TokenConfig `json:"token"`
}

// IGNORE_EXTREMUM_RANGE defines the valid range for project profit filtering
// Only projects with projectProfit within [Min, Max] will be included
const (
	IGNORE_EXTREMUM_RANGE_MIN = -5.0
	IGNORE_EXTREMUM_RANGE_MAX = 30.0
)

// GetProjectProfitRanking returns paginated projects sorted by project profit
// Only includes projects with projectProfit within IGNORE_EXTREMUM_RANGE [Min, Max]
// Query parameters: page (default: 1), page_size (default: 10, max: 100), order_type (default: desc, options: asc/desc)
func GetProjectProfitRanking(c *gin.Context) {
	// Parse pagination parameters
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

	// Parse order_type parameter (asc or desc, default: desc)
	orderType := "desc"
	if ot := c.Query("order_type"); ot != "" {
		otLower := strings.ToLower(ot)
		if otLower == "asc" || otLower == "desc" {
			orderType = otLower
		}
	}

	// 1. Get all ProjectConfigs
	var projects []models.ProjectConfig
	if err := dbconfig.DB.Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. Calculate project profit for each project and build response
	var results []ProjectSettleResp
	for i := range projects {
		project := &projects[i]

		// Calculate project_profit: current id's assets_balance minus previous id's assets_balance
		projectProfit := 0.0
		if project.ID > 1 {
			var previousProject models.ProjectConfig
			if err := dbconfig.DB.First(&previousProject, project.ID-1).Error; err == nil {
				projectProfit = project.AssetsBalance - previousProject.AssetsBalance
			}
		}

		// Filter by IGNORE_EXTREMUM_RANGE: only include projects within [Min, Max]
		if projectProfit < IGNORE_EXTREMUM_RANGE_MIN || projectProfit > IGNORE_EXTREMUM_RANGE_MAX {
			continue
		}

		// Get TokenConfig
		var token models.TokenConfig
		if err := dbconfig.DB.First(&token, project.TokenID).Error; err != nil {
			// If token not found, continue with empty token
			token = models.TokenConfig{}
		}

		// Build response
		resp := ProjectSettleResp{
			ID:                project.ID,
			Name:              project.Name,
			PoolPlatform:      project.PoolPlatform,
			PoolID:            project.PoolID,
			TokenID:           project.TokenID,
			TokenMetadataID:   project.TokenMetadataID,
			IsActive:          project.IsActive,
			UpdateStatEnabled: project.UpdateStatEnabled,
			IsMigrated:        project.IsMigrated,
			IsLocked:          project.IsLocked,
			AssetsBalance:     project.AssetsBalance,
			RetailSolAmount:   project.RetailSolAmount,
			PoolConfig:        project.PoolConfig,
			ProjectProfit:     projectProfit,
			CreatedAt:         project.CreatedAt.Format("2006-01-02T15:04:05.999999Z"),
			UpdatedAt:         project.UpdatedAt.Format("2006-01-02T15:04:05.999999Z"),
			Token:             &token,
		}

		results = append(results, resp)
	}

	// 3. Sort by project_profit based on order_type
	sort.Slice(results, func(i, j int) bool {
		if orderType == "asc" {
			return results[i].ProjectProfit < results[j].ProjectProfit
		}
		// default: desc
		return results[i].ProjectProfit > results[j].ProjectProfit
	})

	// 4. Apply pagination
	total := len(results)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// Calculate offset and limit
	offset := (page - 1) * pageSize
	end := offset + pageSize
	if end > total {
		end = total
	}

	// Get paginated data
	var paginatedResults []ProjectSettleResp
	if offset < total {
		paginatedResults = results[offset:end]
	}

	// 5. Return paginated response
	response := gin.H{
		"data": paginatedResults,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < totalPages,
			"has_prev":     page > 1,
		},
	}

	c.JSON(http.StatusOK, response)
}

// VestingReviewRequest represents the request body for vesting review
type VestingReviewRequest struct {
	StartID     uint  `json:"start_id" binding:"required"`
	EndID       uint  `json:"end_id" binding:"required"`
	OnlySuccess *bool `json:"onlySuccess"`
}

// VestingReview returns vesting summary for ProjectConfigs in [start_id, end_id]
// If onlySuccess is true, only includes vesting entries where status == "done"
func VestingReview(c *gin.Context) {
	var req VestingReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StartID > req.EndID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_id must be <= end_id"})
		return
	}

	onlySuccess := req.OnlySuccess != nil && *req.OnlySuccess

	// Query projects in range (inclusive)
	var projects []models.ProjectConfig
	if err := dbconfig.DB.
		Preload("Token").
		Where("id >= ? AND id <= ?", req.StartID, req.EndID).
		Order("id asc").
		Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	projectCount := 0
	var sumPoolQuoteBalance float64
	var sumPoolRemoveAmount float64

	for i := range projects {
		vesting := projects[i].Vesting
		if len(vesting) == 0 {
			continue
		}
		// Skip explicit JSON null
		if string(vesting) == "null" {
			continue
		}

		var m map[string]interface{}
		if err := json.Unmarshal(vesting, &m); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      "failed to parse vesting json",
				"project_id": projects[i].ID,
				"detail":     err.Error(),
			})
			return
		}

		status, _ := m["status"].(string)
		if onlySuccess && status != "done" {
			continue
		}

		// Extract numeric fields (JSON numbers decode to float64 by default)
		pqb, ok := m["pool_quote_balance"].(float64)
		if !ok {
			// allow integers decoded as float64 (still float64) or missing (treated as 0)
			pqb = 0
		}

		prm, ok := m["pool_remove_amount"].(float64)
		if !ok {
			prm = 0
		}

		sumPoolQuoteBalance += pqb
		sumPoolRemoveAmount += prm
		projectCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"start_id":           req.StartID,
		"end_id":             req.EndID,
		"project_count":      projectCount,
		"pool_quote_balance": sumPoolQuoteBalance,
		"pool_remove_amount": sumPoolRemoveAmount,
		"data":               projects,
	})
}
