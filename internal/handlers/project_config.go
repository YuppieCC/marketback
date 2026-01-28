package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	// "log"
	"os"
	"time"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	dbconfig "marketcontrol/pkg/config"
	pumpsolana "marketcontrol/pkg/solana"
	"marketcontrol/pkg/solana/meteora"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ProjectConfigRequest represents the request body for creating/updating a project config
type ProjectConfigRequest struct {
	Name              *string         `json:"name"`
	PoolPlatform      *string         `json:"pool_platform"`
	PoolID            *uint           `json:"pool_id"`
	TokenID           *uint           `json:"token_id"`
	TokenMetadataID   *uint           `json:"token_metadata_id"`
	SnapshotEnabled   *bool           `json:"snapshot_enabled"`
	SnapshotCount     *int            `json:"snapshot_count"`
	IsActive          *bool           `json:"is_active"`
	UpdateStatEnabled *bool           `json:"update_stat_enabled"`
	IsMigrated        *bool           `json:"is_migrated"`
	IsLocked          *bool           `json:"is_locked"`
	AssetsBalance     *float64        `json:"assets_balance"`
	RetailSolAmount   *float64        `json:"retail_sol_amount"`
	PoolConfig        *string         `json:"pool_config"`
	Event             json.RawMessage `json:"event"`
	Vesting           json.RawMessage `json:"vesting"`
}

// ProjectConfigResp represents the response structure for a project config
type ProjectConfigResp struct {
	ID              uint                `json:"id"`
	Name            string              `json:"name"`
	PoolPlatform    string              `json:"pool_platform"`
	PoolID          uint                `json:"pool_id"`
	TokenID         uint                `json:"token_id"`
	TokenMetadataID uint                `json:"token_metadata_id"`
	IsActive        bool                `json:"is_active"`
	IsMigrated      bool                `json:"is_migrated"`
	IsLocked        bool                `json:"is_locked"`
	AssetsBalance   float64             `json:"assets_balance"`
	RetailSolAmount float64             `json:"retail_sol_amount"`
	PoolConfig      string              `json:"pool_config"`
	Event           json.RawMessage     `json:"event"`
	Vesting         json.RawMessage     `json:"vesting"`
	ProjectProfit   float64             `json:"project_profit"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	Pool            interface{}         `json:"pool,omitempty"`
	Token           *models.TokenConfig `json:"token,omitempty"`
	PoolRelation    interface{}         `json:"pool_relation"`
}

// ListProjectConfigs returns a list of all project configs
func ListProjectConfigs(c *gin.Context) {
	var projects []models.ProjectConfig
	if err := dbconfig.DB.Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var respList []ProjectConfigResp
	for _, project := range projects {
		proj := project // Create a new variable to avoid pointer issues
		if resp := buildProjectConfigResp(&proj); resp != nil {
			respList = append(respList, *resp)
		}
	}

	c.JSON(http.StatusOK, respList)
}

// GetProjectConfig returns a specific project config by ID
func GetProjectConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	resp := buildProjectConfigResp(&project)
	if resp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build response"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreateProjectConfig creates a new project config
func CreateProjectConfig(c *gin.Context) {
	var request ProjectConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证必填字段
	if request.Name == nil || request.PoolPlatform == nil || request.PoolID == nil || request.TokenID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, pool_platform, pool_id, token_id 是必填字段"})
		return
	}

	// 验证池子平台类型
	if *request.PoolPlatform != "raydium" && *request.PoolPlatform != "pumpfun_internal" && *request.PoolPlatform != "pumpfun_amm" &&
		*request.PoolPlatform != "raydium_launchpad" && *request.PoolPlatform != "raydium_cpmm" && *request.PoolPlatform != "meteora_dbc" && *request.PoolPlatform != "meteora_cpmm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pool_platform 必须是 raydium, pumpfun_internal, pumpfun_amm, raydium_launchpad, raydium_cpmm, meteora_dbc 或 meteora_cpmm 之一"})
		return
	}

	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}
	updateStatEnabled := true
	if request.UpdateStatEnabled != nil {
		updateStatEnabled = *request.UpdateStatEnabled
	}
	snapshotEnabled := false
	if request.SnapshotEnabled != nil {
		snapshotEnabled = *request.SnapshotEnabled
	}

	// Verify pool exists based on platform
	switch *request.PoolPlatform {
	case "raydium":
		var pool models.PoolConfig
		if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium pool not found"})
			return
		}
	case "pumpfun_internal":
		var pool models.PumpfuninternalConfig
		if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Pumpfun pool not found"})
			return
		}
	case "pumpfun_amm":
		var pool models.PumpfunAmmPoolConfig
		if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: PumpfunAmm pool not found"})
			return
		}
	case "raydium_launchpad":
		var pool models.RaydiumLaunchpadPoolConfig
		if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium Launchpad pool not found"})
			return
		}
	case "raydium_cpmm":
		var pool models.RaydiumCpmmPoolConfig
		if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium CPMM pool not found"})
			return
		}
	case "meteora_dbc":
		var pool models.MeteoradbcConfig
		if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Meteoradbc pool not found"})
			return
		}
	case "meteora_cpmm":
		var pool models.MeteoracpmmConfig
		if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Meteoracpmm pool not found"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
		return
	}

	// Verify token exists
	var token models.TokenConfig
	if err := dbconfig.DB.First(&token, *request.TokenID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token_id: Token not found"})
		return
	}

	poolConfig := ""
	if request.PoolConfig != nil {
		poolConfig = *request.PoolConfig
	}

	tokenMetadataID := uint(0)
	if request.TokenMetadataID != nil {
		tokenMetadataID = *request.TokenMetadataID
	}

	project := models.ProjectConfig{
		Name:              *request.Name,
		PoolPlatform:      *request.PoolPlatform,
		PoolID:            *request.PoolID,
		TokenID:           *request.TokenID,
		TokenMetadataID:   tokenMetadataID,
		SnapshotEnabled:   snapshotEnabled,
		SnapshotCount:     0,
		IsActive:          isActive,
		UpdateStatEnabled: updateStatEnabled,
		AssetsBalance:     0,
		PoolConfig:        poolConfig,
		Event:             request.Event,
		Vesting:           request.Vesting,
	}

	if err := dbconfig.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载项目并使用新的响应结构
	if err := dbconfig.DB.Preload("Token").First(&project, project.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project associations"})
		return
	}

	// 构建响应
	resp := buildProjectConfigResp(&project)
	c.JSON(http.StatusCreated, resp)
}

// UpdateProjectConfig updates an existing project config
func UpdateProjectConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request ProjectConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ID is required (from path parameter), all other fields are optional
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// 验证池子平台和ID的关联性
	if request.PoolPlatform != nil && request.PoolID != nil {
		// Verify pool exists based on platform
		switch *request.PoolPlatform {
		case "raydium":
			var pool models.PoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium pool not found"})
				return
			}
		case "pumpfun_internal":
			var pool models.PumpfuninternalConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Pumpfun pool not found"})
				return
			}
		case "pumpfun_amm":
			var pool models.PumpfunAmmPoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: PumpfunAmm pool not found"})
				return
			}
		case "raydium_launchpad":
			var pool models.RaydiumLaunchpadPoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium Launchpad pool not found"})
				return
			}
		case "raydium_cpmm":
			var pool models.RaydiumCpmmPoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium CPMM pool not found"})
				return
			}
		case "meteora_dbc":
			var pool models.MeteoradbcConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Meteoradbc pool not found"})
				return
			}
		case "meteora_cpmm":
			var pool models.MeteoracpmmConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Meteoracpmm pool not found"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
			return
		}
	} else if request.PoolPlatform != nil {
		// 如果只提供了平台，验证现有池子ID是否匹配
		switch *request.PoolPlatform {
		case "raydium":
			var pool models.PoolConfig
			if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Existing pool_id does not match Raydium platform"})
				return
			}
		case "pumpfun_internal":
			var pool models.PumpfuninternalConfig
			if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Existing pool_id does not match Pumpfun platform"})
				return
			}
		case "pumpfun_amm":
			var pool models.PumpfunAmmPoolConfig
			if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Existing pool_id does not match PumpfunAmm platform"})
				return
			}
		case "raydium_launchpad":
			var pool models.RaydiumLaunchpadPoolConfig
			if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Existing pool_id does not match Raydium Launchpad platform"})
				return
			}
		case "raydium_cpmm":
			var pool models.RaydiumCpmmPoolConfig
			if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Existing pool_id does not match Raydium CPMM platform"})
				return
			}
		case "meteora_dbc":
			var pool models.MeteoradbcConfig
			if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Existing pool_id does not match Meteoradbc platform"})
				return
			}
		case "meteora_cpmm":
			var pool models.MeteoracpmmConfig
			if err := dbconfig.DB.First(&pool, project.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Existing pool_id does not match Meteoracpmm platform"})
				return
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported pool platform"})
			return
		}
	} else if request.PoolID != nil {
		// 如果只提供了池子ID，验证是否匹配现有平台
		switch project.PoolPlatform {
		case "raydium":
			var pool models.PoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium pool not found"})
				return
			}
		case "pumpfun_internal":
			var pool models.PumpfuninternalConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Pumpfun pool not found"})
				return
			}
		case "pumpfun_amm":
			var pool models.PumpfunAmmPoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: PumpfunAmm pool not found"})
				return
			}
		case "raydium_launchpad":
			var pool models.RaydiumLaunchpadPoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium Launchpad pool not found"})
				return
			}
		case "raydium_cpmm":
			var pool models.RaydiumCpmmPoolConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Raydium CPMM pool not found"})
				return
			}
		case "meteora_dbc":
			var pool models.MeteoradbcConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Meteoradbc pool not found"})
				return
			}
		case "meteora_cpmm":
			var pool models.MeteoracpmmConfig
			if err := dbconfig.DB.First(&pool, *request.PoolID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pool_id: Meteoracpmm pool not found"})
				return
			}
		}
	}

	// 验证代币是否存在
	if request.TokenID != nil {
		var token models.TokenConfig
		if err := dbconfig.DB.First(&token, *request.TokenID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid token_id: Token not found"})
			return
		}
	}

	// 部分更新字段
	if request.Name != nil {
		project.Name = *request.Name
	}
	if request.PoolPlatform != nil {
		project.PoolPlatform = *request.PoolPlatform
	}
	if request.PoolID != nil {
		project.PoolID = *request.PoolID
	}
	if request.TokenID != nil {
		project.TokenID = *request.TokenID
	}
	if request.TokenMetadataID != nil {
		project.TokenMetadataID = *request.TokenMetadataID
	}
	if request.SnapshotEnabled != nil {
		project.SnapshotEnabled = *request.SnapshotEnabled
	}
	if request.SnapshotCount != nil {
		project.SnapshotCount = *request.SnapshotCount
	}
	if request.IsActive != nil {
		project.IsActive = *request.IsActive
	}
	if request.UpdateStatEnabled != nil {
		project.UpdateStatEnabled = *request.UpdateStatEnabled
	}
	if request.IsMigrated != nil {
		project.IsMigrated = *request.IsMigrated
	}
	if request.IsLocked != nil {
		project.IsLocked = *request.IsLocked
	}
	if request.AssetsBalance != nil {
		project.AssetsBalance = *request.AssetsBalance
	}
	if request.RetailSolAmount != nil {
		project.RetailSolAmount = *request.RetailSolAmount
	}
	if request.PoolConfig != nil {
		project.PoolConfig = *request.PoolConfig
	}
	if len(request.Event) > 0 {
		project.Event = request.Event
	}
	if len(request.Vesting) > 0 {
		project.Vesting = request.Vesting
	}

	// 如果提供了 is_active，则同步更新对应池子的 status
	if request.IsActive != nil {
		if err := UpdatePoolStatus(project.PoolPlatform, project.PoolID, *request.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pool status: " + err.Error()})
			return
		}
	}

	// 当 is_active 为 false 时，关闭该项目下所有策略
	if request.IsActive != nil && !*request.IsActive {
		if err := CloseAllStrategyStatus(project.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close strategies: " + err.Error()})
			return
		}
	}

	if err := dbconfig.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 重新加载项目并使用新的响应结构
	if err := dbconfig.DB.Preload("Token").First(&project, project.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project associations"})
		return
	}

	// 构建响应
	resp := buildProjectConfigResp(&project)
	c.JSON(http.StatusOK, resp)
}

// DeleteProjectConfig deletes a project config
func DeleteProjectConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// 1. 检查是否存在依赖的角色配置
	var roleCount int64
	if err := dbconfig.DB.Model(&models.RoleConfigRelation{}).Where("project_id = ?", id).Count(&roleCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check role dependencies"})
		return
	}

	if roleCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Cannot delete project: there are roles depending on this project",
			"role_count": roleCount,
		})
		return
	}

	// 2. 检查是否存在依赖的策略配置
	var strategyCount int64
	if err := dbconfig.DB.Model(&models.StrategyConfig{}).Where("project_id = ?", id).Count(&strategyCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check strategy dependencies"})
		return
	}

	if strategyCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":          "Cannot delete project: there are strategies depending on this project",
			"strategy_count": strategyCount,
		})
		return
	}

	// 3. 执行删除操作
	if err := dbconfig.DB.Delete(&models.ProjectConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// buildProjectConfigResp 构建项目配置响应
func buildProjectConfigResp(project *models.ProjectConfig) *ProjectConfigResp {
	if project == nil {
		return nil
	}

	// 查询池子
	var pool interface{}
	// 用于 meteora_dbc 衍生关系
	var meteoradbcForRelation *models.MeteoradbcConfig
	switch project.PoolPlatform {
	case "raydium":
		var raydiumPool models.PoolConfig
		if err := dbconfig.DB.First(&raydiumPool, project.PoolID).Error; err == nil {
			pool = raydiumPool
		}
	case "pumpfun_internal":
		var pumpfunPool models.PumpfuninternalConfig
		if err := dbconfig.DB.First(&pumpfunPool, project.PoolID).Error; err == nil {
			pool = pumpfunPool
		}
	case "pumpfun_amm":
		var pumpfunAmmPool models.PumpfunAmmPoolConfig
		if err := dbconfig.DB.First(&pumpfunAmmPool, project.PoolID).Error; err == nil {
			pool = pumpfunAmmPool
		}
	case "raydium_launchpad":
		var raydiumLaunchpadPool models.RaydiumLaunchpadPoolConfig
		if err := dbconfig.DB.First(&raydiumLaunchpadPool, project.PoolID).Error; err == nil {
			pool = raydiumLaunchpadPool
		}
	case "raydium_cpmm":
		var raydiumCpmmPool models.RaydiumCpmmPoolConfig
		if err := dbconfig.DB.First(&raydiumCpmmPool, project.PoolID).Error; err == nil {
			pool = raydiumCpmmPool
		}
	case "meteora_dbc":
		var meteoradbcPool models.MeteoradbcConfig
		if err := dbconfig.DB.First(&meteoradbcPool, project.PoolID).Error; err == nil {
			pool = meteoradbcPool
			meteoradbcForRelation = &meteoradbcPool
			// 检查 IsMigrated 是否为真
			if meteoradbcPool.IsMigrated && meteoradbcPool.DammV2PoolAddress != "" {
				// 查询对应的 MeteoracpmmConfig
				var meteoracpmmConfig models.MeteoracpmmConfig
				if err := dbconfig.DB.Where("pool_address = ?", meteoradbcPool.DammV2PoolAddress).First(&meteoracpmmConfig).Error; err == nil {
					// 直接修改 pool_platform, pool_id 和 pool 的数据
					project.PoolPlatform = "meteora_cpmm"
					project.PoolID = meteoracpmmConfig.ID
					pool = meteoracpmmConfig
				}
			}
		}
	case "meteora_cpmm":
		var meteoracpmmPool models.MeteoracpmmConfig
		if err := dbconfig.DB.First(&meteoracpmmPool, project.PoolID).Error; err == nil {
			pool = meteoracpmmPool
		}
	}

	// 查询 Token
	var token models.TokenConfig
	dbconfig.DB.First(&token, project.TokenID)

	// 查询 PoolRelation，默认为空字典
	var poolRelation interface{} = map[string]interface{}{}

	// 当 project.PoolPlatform 为 raydium_launchpad 时，尝试查询 RaydiumPoolRelation
	if project.PoolPlatform == "raydium_launchpad" {
		if raydiumLaunchpadPool, ok := pool.(models.RaydiumLaunchpadPoolConfig); ok {
			var relation models.RaydiumPoolRelation
			if err := dbconfig.DB.Where("launchpad_pool_id = ?", raydiumLaunchpadPool.PoolAddress).First(&relation).Error; err == nil {
				// 如果找到 RaydiumPoolRelation，尝试获取对应的 RaydiumCpmmPoolConfig
				var cpmmPoolConfig models.RaydiumCpmmPoolConfig
				if err := dbconfig.DB.Where("pool_address = ?", relation.CpmmPoolID).First(&cpmmPoolConfig).Error; err == nil {
					// 构建 PoolRelation 响应
					poolRelation = map[string]interface{}{
						"relation":         relation,
						"cpmm_pool_config": cpmmPoolConfig,
					}
				} else {
					// 如果没有找到 CPMM 池配置，只返回 relation
					poolRelation = map[string]interface{}{
						"relation": relation,
					}
				}
			}
		}
	}

	// 当平台为 meteora_dbc 时，返回对应 DammV2PoolAddress 的 MeteoracpmmConfig 于 PoolRelation
	if meteoradbcForRelation != nil && meteoradbcForRelation.DammV2PoolAddress != "" {
		var meteoracpmmConfig models.MeteoracpmmConfig
		if err := dbconfig.DB.Where("pool_address = ?", meteoradbcForRelation.DammV2PoolAddress).First(&meteoracpmmConfig).Error; err == nil {
			if pr, ok := poolRelation.(map[string]interface{}); ok {
				pr["meteoracpmm_config"] = meteoracpmmConfig
				poolRelation = pr
			} else {
				poolRelation = map[string]interface{}{
					"meteoracpmm_config": meteoracpmmConfig,
				}
			}
		}
	}

	// 计算 project_profit: 当前 id 的 assets_balance 减去上一个 id 的 assets_balance
	projectProfit := 0.0
	if project.ID > 1 {
		var previousProject models.ProjectConfig
		if err := dbconfig.DB.First(&previousProject, project.ID-1).Error; err == nil {
			projectProfit = project.AssetsBalance - previousProject.AssetsBalance
		}
		// 如果找不到上一个 id，projectProfit 保持为 0.0（默认值）
	}

	return &ProjectConfigResp{
		ID:              project.ID,
		Name:            project.Name,
		PoolPlatform:    project.PoolPlatform,
		PoolID:          project.PoolID,
		TokenID:         project.TokenID,
		TokenMetadataID: project.TokenMetadataID,
		IsActive:        project.IsActive,
		IsMigrated:      project.IsMigrated,
		IsLocked:        project.IsLocked,
		AssetsBalance:   project.AssetsBalance,
		RetailSolAmount: project.RetailSolAmount,
		PoolConfig:      project.PoolConfig,
		Event:           project.Event,
		Vesting:         project.Vesting,
		ProjectProfit:   projectProfit,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
		Pool:            pool,
		Token:           &token,
		PoolRelation:    poolRelation,
	}
}

// ProjectFundTransferRecordRequest represents the request body for creating a project fund transfer record
type ProjectFundTransferRecordRequest struct {
	ProjectID  uint    `json:"project_id" binding:"required"`
	Mint       string  `json:"mint" binding:"required"`
	Direction  string  `json:"direction" binding:"required,oneof=in out"`
	Amount     float64 `json:"amount" binding:"required,min=0"`
	TargetName string  `json:"target_name" binding:"required,oneof=project pool retail_investors"` // 必选: "project", "pool", "retail_investors"
}

// CreateProjectFundTransferRecord creates a new project fund transfer record
func CreateProjectFundTransferRecord(c *gin.Context) {
	var request ProjectFundTransferRecordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify project exists
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, request.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id: Project not found"})
		return
	}

	record := models.ProjectFundTransferRecord{
		ProjectID:  request.ProjectID,
		Mint:       request.Mint,
		Direction:  request.Direction,
		Amount:     request.Amount,
		TargetName: request.TargetName,
	}

	if err := dbconfig.DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, record)
}

// ListProjectFundTransferRecords returns all project fund transfer records
func ListProjectFundTransferRecords(c *gin.Context) {
	var records []models.ProjectFundTransferRecord
	if err := dbconfig.DB.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

// GetProjectFundTransferRecord returns a specific project fund transfer record by ID
func GetProjectFundTransferRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var record models.ProjectFundTransferRecord
	if err := dbconfig.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, record)
}

// GetProjectFundTransferRecordsByProjectID returns all records for a specific project
func GetProjectFundTransferRecordsByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var records []models.ProjectFundTransferRecord
	if err := dbconfig.DB.Where("project_id = ?", projectID).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

// UpdateProjectFundTransferRecord updates an existing project fund transfer record
func UpdateProjectFundTransferRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request ProjectFundTransferRecordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var record models.ProjectFundTransferRecord
	if err := dbconfig.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Verify project exists
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, request.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id: Project not found"})
		return
	}

	record.ProjectID = request.ProjectID
	record.Mint = request.Mint
	record.Direction = request.Direction
	record.Amount = request.Amount

	if err := dbconfig.DB.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

// DeleteProjectFundTransferRecord deletes a project fund transfer record
func DeleteProjectFundTransferRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.ProjectFundTransferRecord{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// GetProjectInitialSol returns the aggregated SOL amount for a project (in - out)
func GetProjectInitialSol(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Verify project exists
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Query all SOL records for this project
	var records []models.ProjectFundTransferRecord
	if err := dbconfig.DB.Where("project_id = ? AND mint = ?", projectID, "sol").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate net amount (in - out)
	var projectInitialSol float64
	for _, record := range records {
		if record.Direction == "in" {
			projectInitialSol += record.Amount
		} else if record.Direction == "out" {
			projectInitialSol -= record.Amount
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"project_id":          projectID,
		"project_initial_sol": projectInitialSol,
		"records_count":       len(records),
	})
}

// GetAddressCountByProjectID returns the count of unique addresses for a project
func GetAddressCountByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	// Verify project exists
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, projectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Get unique address count using a subquery to get role IDs for the project
	var count int64
	if err := dbconfig.DB.Model(&models.RoleAddress{}).
		Distinct("address").
		Joins("JOIN role_config ON role_address.role_id = role_config.id").
		Where("role_config.project_id = ?", projectID).
		Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project_id": projectID,
		"count":      count,
	})
}

// ListProjectConfigsBySlice returns a paginated list of project configs
func ListProjectConfigsBySlice(c *gin.Context) {
	// Parse query parameters with defaults
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
		// Validate order field to prevent SQL injection
		validFields := []string{
			"id", "name", "pool_platform", "pool_id", "token_id",
			"snapshot_enabled", "snapshot_count", "is_active", "update_stat_enabled",
			"is_migrated", "created_at", "updated_at",
		}
		for _, field := range validFields {
			if of == field {
				orderField = of
				break
			}
		}
	}

	orderType := "desc"
	if ot := c.Query("order_type"); ot == "asc" || ot == "desc" {
		orderType = ot
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count
	var total int64
	if err := dbconfig.DB.Model(&models.ProjectConfig{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated results
	var configs []models.ProjectConfig
	if err := dbconfig.DB.Order(orderField + " " + orderType).
		Offset(offset).
		Limit(pageSize).
		Find(&configs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to ProjectConfigResp with project_profit calculation
	var respList []ProjectConfigResp
	for _, config := range configs {
		proj := config // Create a new variable to avoid pointer issues
		if resp := buildProjectConfigResp(&proj); resp != nil {
			respList = append(respList, *resp)
		}
	}

	// Calculate pagination info
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)

	response := gin.H{
		"data": respList,
		"pagination": gin.H{
			"current_page": page,
			"page_size":    pageSize,
			"total_pages":  totalPages,
			"total_count":  total,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetLatestProjectConfig returns the latest ProjectConfig by ID desc
func GetLatestProjectConfig(c *gin.Context) {
	var project models.ProjectConfig
	if err := dbconfig.DB.Order("id desc").First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "ProjectConfig not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := buildProjectConfigResp(&project)
	if resp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build response"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetLatestActiveProjectConfig returns the latest active ProjectConfig
// First gets the latest 5 ProjectConfigs, then finds the latest one with IsActive = true
func GetLatestActiveProjectConfig(c *gin.Context) {
	// First, get the latest 5 ProjectConfigs ordered by ID desc
	var projects []models.ProjectConfig
	if err := dbconfig.DB.Order("id desc").Limit(5).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Find the latest one with IsActive = true
	var activeProject *models.ProjectConfig
	for i := range projects {
		if projects[i].IsActive {
			activeProject = &projects[i]
			break
		}
	}

	if activeProject == nil {
		// Return empty data instead of 404 error
		emptyResp := &ProjectConfigResp{}
		c.JSON(http.StatusOK, emptyResp)
		return
	}

	resp := buildProjectConfigResp(activeProject)
	if resp == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build response"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ProjectExtraAddressRequest 项目额外地址请求结构
type ProjectExtraAddressRequest struct {
	ProjectID       uint   `json:"project_id" binding:"required"`
	Address         string `json:"address" binding:"required"`
	Enabled         *bool  `json:"enabled"`
	PrivateKeyVaild *bool  `json:"private_key_vaild"`
	PrivateKey      string `json:"private_key"`
}

// ListProjectExtraAddresses 获取所有项目额外地址
func ListProjectExtraAddresses(c *gin.Context) {
	var addresses []models.ProjectExtraAddress
	if err := dbconfig.DB.Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, addresses)
}

// GetProjectExtraAddress 获取指定ID的项目额外地址
func GetProjectExtraAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var address models.ProjectExtraAddress
	if err := dbconfig.DB.First(&address, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, address)
}

// GetProjectExtraAddressesByProjectID 获取指定项目的所有额外地址
func GetProjectExtraAddressesByProjectID(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Param("project_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id format"})
		return
	}

	var addresses []models.ProjectExtraAddress
	if err := dbconfig.DB.Where("project_id = ?", projectID).Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, addresses)
}

// CreateProjectExtraAddress 创建项目额外地址
func CreateProjectExtraAddress(c *gin.Context) {
	var request ProjectExtraAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证项目是否存在
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, request.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id: Project not found"})
		return
	}

	// 检验同一个 ProjectID 是否已存在 Address
	var existingAddress models.ProjectExtraAddress
	if err := dbconfig.DB.Where("project_id = ? AND address = ?", request.ProjectID, request.Address).First(&existingAddress).Error; err == nil {
		// 如果找到了记录，说明已存在相同的 ProjectID 和 Address 组合
		c.JSON(http.StatusConflict, gin.H{"error": "Address already exists for this project"})
		return
	}

	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}

	privateKeyVaild := false
	if request.PrivateKeyVaild != nil {
		privateKeyVaild = *request.PrivateKeyVaild
	}

	address := models.ProjectExtraAddress{
		ProjectID:       request.ProjectID,
		Address:         request.Address,
		Enabled:         enabled,
		PrivateKeyVaild: privateKeyVaild,
		PrivateKey:      request.PrivateKey,
	}

	if err := dbconfig.DB.Create(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, address)
}

// UpdateProjectExtraAddress 更新项目额外地址
func UpdateProjectExtraAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request ProjectExtraAddressRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var address models.ProjectExtraAddress
	if err := dbconfig.DB.First(&address, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// 验证项目是否存在
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, request.ProjectID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project_id: Project not found"})
		return
	}

	address.ProjectID = request.ProjectID
	address.Address = request.Address
	if request.Enabled != nil {
		address.Enabled = *request.Enabled
	}
	if request.PrivateKeyVaild != nil {
		address.PrivateKeyVaild = *request.PrivateKeyVaild
	}
	if request.PrivateKey != "" {
		address.PrivateKey = request.PrivateKey
	}

	if err := dbconfig.DB.Save(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, address)
}

// DeleteProjectExtraAddress 删除项目额外地址
func DeleteProjectExtraAddress(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.ProjectExtraAddress{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project extra address deleted successfully"})
}

// AutoCreatePumpfuninternalProjectRequest represents the request body for auto-creating a pumpfun internal project
type AutoCreatePumpfuninternalProjectRequest struct {
	Mint            string   `json:"mint" binding:"required"`
	TokenMetadataID uint     `json:"token_metadata_id" binding:"required"`
	RoleID          uint     `json:"role_id" binding:"required"`
	ProjectName     string   `json:"project_name"`
	FeeRecipient    string   `json:"fee_recipient"`                   // Optional, will use default if not provided
	FeeRate         *float64 `json:"fee_rate"`                        // Optional, will use default if not provided
	CoinCreator     string   `json:"coin_creator" binding:"required"` // Required coin creator address
}

// AutoCreatePumpfuninternalProject creates a complete project setup including TokenConfig, PumpfuninternalConfig, ProjectConfig, and RoleConfigRelation
func AutoCreatePumpfuninternalProject(c *gin.Context) {
	var request AutoCreatePumpfuninternalProjectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start a database transaction
	tx := dbconfig.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Get TokenMetadata by ID
	var tokenMetadata models.TokenMetadata
	if err := tx.First(&tokenMetadata, request.TokenMetadataID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "TokenMetadata not found"})
		return
	}

	// 2. Create or get TokenConfig
	var tokenConfig models.TokenConfig
	err := tx.Where("mint = ?", request.Mint).First(&tokenConfig).Error
	if err != nil {
		// TokenConfig doesn't exist, create it
		tokenConfig = models.TokenConfig{
			Mint:        request.Mint,
			Symbol:      tokenMetadata.Symbol,
			Name:        tokenMetadata.Name,
			Decimals:    6, // Default decimals for most tokens
			LogoURI:     tokenMetadata.Image,
			TotalSupply: 1000000000,          // Will be updated later if needed
			Creator:     request.CoinCreator, // Set the creator from request
		}
		if err := tx.Create(&tokenConfig).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create TokenConfig"})
			return
		}
	} else {
		// TokenConfig exists, update Creator if it's empty
		if tokenConfig.Creator == "" {
			tokenConfig.Creator = request.CoinCreator
			if err := tx.Save(&tokenConfig).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update TokenConfig creator"})
				return
			}
		}
	}

	// 3. Create PumpfuninternalConfig with on-chain data
	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create client
	client := rpc.New(solanaRPC)

	// Parse mint address
	mintPubkey, err := solana.PublicKeyFromBase58(request.Mint)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mint address"})
		return
	}

	// Validate CoinCreator address
	_, err = solana.PublicKeyFromBase58(request.CoinCreator)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coin creator address"})
		return
	}

	// Use default fee recipient (you may want to make this configurable)
	defaultFeeRecipient := "62qc2CNXwrYqQScmEdiZFFAnJR262PxWEuNQtxfafNgV" // System Program
	feeRecipient := request.FeeRecipient
	if feeRecipient == "" {
		feeRecipient = defaultFeeRecipient
	}
	feeRecipientPubkey, err := solana.PublicKeyFromBase58(feeRecipient)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid fee recipient address"})
		return
	}

	// Use default fee rate if not provided
	feeRate := 0.01
	if request.FeeRate != nil && *request.FeeRate != 0 {
		feeRate = *request.FeeRate
	}

	// Get on-chain data
	poolStat, err := pumpsolana.GetPumpFunInternalPoolStat(client, mintPubkey, feeRate, feeRecipientPubkey)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get on-chain data: " + err.Error()})
		return
	}

	// Create PumpfuninternalConfig with on-chain data
	pumpfunConfig := models.PumpfuninternalConfig{
		Platform:               "pumpfun_internal",
		Mint:                   poolStat.Mint,
		BondingCurvePda:        poolStat.BondingCurvePDA,
		AssociatedBondingCurve: poolStat.AssociatedBondingCurve,
		CreatorVaultPda:        poolStat.CreatorVaultPDA,
		FeeRecipient:           poolStat.FeeRecipient,
		FeeRate:                poolStat.FeeRate,
		Status:                 "active",
	}
	if err := tx.Create(&pumpfunConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PumpfuninternalConfig"})
		return
	}

	// 4. Generate project name if not provided
	projectName := request.ProjectName
	if projectName == "" {
		// Rule: ${TokenConfig.symbol}-${TokenConfig.mint.slice(0, 5)}
		mintPrefix := request.Mint
		if len(mintPrefix) > 5 {
			mintPrefix = mintPrefix[:5]
		}
		projectName = fmt.Sprintf("%s-%s", tokenConfig.Symbol, mintPrefix)
	}

	// 5. Create ProjectConfig
	projectConfig := models.ProjectConfig{
		Name:              projectName,
		PoolPlatform:      "pumpfun_internal",
		PoolID:            pumpfunConfig.ID,
		TokenID:           tokenConfig.ID,
		TokenMetadataID:   request.TokenMetadataID,
		SnapshotEnabled:   true,
		SnapshotCount:     0,
		IsActive:          true,
		UpdateStatEnabled: true,
	}
	if err := tx.Create(&projectConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ProjectConfig"})
		return
	}

	// 6. Create RoleConfigRelation
	roleConfigRelation := models.RoleConfigRelation{
		RoleID:    request.RoleID,
		ProjectID: projectConfig.ID,
	}
	if err := tx.Create(&roleConfigRelation).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create RoleConfigRelation"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Build response
	response := gin.H{
		"message": "Project created successfully",
		"data": gin.H{
			"project_config": gin.H{
				"id":                  projectConfig.ID,
				"name":                projectConfig.Name,
				"pool_platform":       projectConfig.PoolPlatform,
				"pool_id":             projectConfig.PoolID,
				"token_id":            projectConfig.TokenID,
				"token_metadata_id":   projectConfig.TokenMetadataID,
				"snapshot_enabled":    projectConfig.SnapshotEnabled,
				"is_migrated":         projectConfig.IsMigrated,
				"is_active":           projectConfig.IsActive,
				"update_stat_enabled": projectConfig.UpdateStatEnabled,
				"created_at":          projectConfig.CreatedAt,
				"updated_at":          projectConfig.UpdatedAt,
			},
			"token_config": gin.H{
				"id":           tokenConfig.ID,
				"mint":         tokenConfig.Mint,
				"symbol":       tokenConfig.Symbol,
				"name":         tokenConfig.Name,
				"decimals":     tokenConfig.Decimals,
				"logo_uri":     tokenConfig.LogoURI,
				"total_supply": tokenConfig.TotalSupply,
			},
			"pumpfun_config": gin.H{
				"id":                       pumpfunConfig.ID,
				"platform":                 pumpfunConfig.Platform,
				"mint":                     pumpfunConfig.Mint,
				"bonding_curve_pda":        pumpfunConfig.BondingCurvePda,
				"associated_bonding_curve": pumpfunConfig.AssociatedBondingCurve,
				"creator_vault_pda":        pumpfunConfig.CreatorVaultPda,
				"fee_recipient":            pumpfunConfig.FeeRecipient,
				"fee_rate":                 pumpfunConfig.FeeRate,
				"status":                   pumpfunConfig.Status,
			},
			"role_config_relation": gin.H{
				"role_id":    roleConfigRelation.RoleID,
				"project_id": roleConfigRelation.ProjectID,
			},
		},
	}

	c.JSON(http.StatusCreated, response)
}

// UpdatePoolStatus 根据平台与池ID更新池状态，同时处理 meteora_cpmm 对应的 dbc 状态联动
func UpdatePoolStatus(poolPlatform string, poolID uint, active bool) error {
	statusVal := "inactive"
	if active {
		statusVal = "active"
	}

	switch poolPlatform {
	case "meteora_cpmm":
		var cpmm models.MeteoracpmmConfig
		if err := dbconfig.DB.First(&cpmm, poolID).Error; err != nil {
			return fmt.Errorf("MeteoracpmmConfig not found: %v", err)
		}
		if err := dbconfig.DB.Model(&models.MeteoracpmmConfig{}).Where("id = ?", poolID).Update("status", statusVal).Error; err != nil {
			return fmt.Errorf("failed to update MeteoracpmmConfig status: %v", err)
		}
		// 级联更新对应 DBC 池（按 DbcPoolAddress 匹配 MeteoradbcConfig.PoolAddress）
		if cpmm.DbcPoolAddress != "" {
			if err := dbconfig.DB.Model(&models.MeteoradbcConfig{}).
				Where("pool_address = ?", cpmm.DbcPoolAddress).
				Update("status", statusVal).Error; err != nil {
				return fmt.Errorf("failed to cascade update MeteoradbcConfig status: %v", err)
			}
		}
	case "meteora_dbc":
		if err := dbconfig.DB.Model(&models.MeteoradbcConfig{}).Where("id = ?", poolID).Update("status", statusVal).Error; err != nil {
			return fmt.Errorf("failed to update MeteoradbcConfig status: %v", err)
		}
	case "pumpfun_amm":
		if err := dbconfig.DB.Model(&models.PumpfunAmmPoolConfig{}).Where("id = ?", poolID).Update("status", statusVal).Error; err != nil {
			return fmt.Errorf("failed to update PumpfunAmmPoolConfig status: %v", err)
		}
	case "pumpfun_internal":
		if err := dbconfig.DB.Model(&models.PumpfuninternalConfig{}).Where("id = ?", poolID).Update("status", statusVal).Error; err != nil {
			return fmt.Errorf("failed to update PumpfuninternalConfig status: %v", err)
		}
	default:
		return fmt.Errorf("unsupported pool_platform: %s", poolPlatform)
	}
	return nil
}

// CloseAllStrategyStatus 关闭指定项目下所有策略（Enabled=false）
func CloseAllStrategyStatus(projectConfigId uint) error {
	if err := dbconfig.DB.Model(&models.StrategyConfig{}).
		Where("project_id = ?", projectConfigId).
		Update("enabled", false).Error; err != nil {
		return fmt.Errorf("failed to update StrategyConfig enabled=false: %v", err)
	}
	return nil
}

// AutoCreatePumpfunAmmProjectRequest represents the request body for auto-creating a pumpfun amm project
type AutoCreatePumpfunAmmProjectRequest struct {
	PoolPlatform        string  `json:"pool_platform" binding:"required"`
	Mint                string  `json:"mint" binding:"required"`
	ProjectInitialToken float64 `json:"project_initial_token" binding:"required"`
	RoleID              uint    `json:"role_id" binding:"required"`
	PoolConfig          struct {
		PoolAddress           string `json:"pool_address" binding:"required"`
		PoolBump              uint8  `json:"pool_bump" binding:"required"`
		Index                 uint16 `json:"index"`
		Creator               string `json:"creator" binding:"required"`
		BaseMint              string `json:"base_mint" binding:"required"`
		QuoteMint             string `json:"quote_mint" binding:"required"`
		LpMint                string `json:"lp_mint" binding:"required"`
		PoolBaseTokenAccount  string `json:"pool_base_token_account" binding:"required"`
		PoolQuoteTokenAccount string `json:"pool_quote_token_account" binding:"required"`
		LpSupply              uint64 `json:"lp_supply" binding:"required"`
		CoinCreator           string `json:"coin_creator" binding:"required"`
	} `json:"pool_config" binding:"required"`
	ProjectName string `json:"project_name"` // Optional
}

// AutoCreatePumpfunAmmProject automatically creates a complete project setup for Pumpfun AMM
func AutoCreatePumpfunAmmProject(c *gin.Context) {
	var request AutoCreatePumpfunAmmProjectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate pool_platform
	if request.PoolPlatform != "pumpfun_amm" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pool_platform must be 'pumpfun_amm'"})
		return
	}

	// Validate project_initial_token
	if request.ProjectInitialToken < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_initial_token must be non-negative"})
		return
	}

	// Start a database transaction
	tx := dbconfig.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Find or create TokenConfig by mint
	var tokenConfig models.TokenConfig
	err := tx.Where("mint = ?", request.Mint).First(&tokenConfig).Error
	if err != nil {
		// TokenConfig doesn't exist, create it with default values
		tokenConfig = models.TokenConfig{
			Mint:        request.Mint,
			Symbol:      "TOKEN",         // Default symbol, will be updated if needed
			Name:        "Unknown Token", // Default name, will be updated if needed
			Decimals:    6,               // Default decimals for most tokens
			LogoURI:     "",              // Empty logo URI
			TotalSupply: 1000000000,      // Default total supply
		}
		if err := tx.Create(&tokenConfig).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create TokenConfig: " + err.Error()})
			return
		}
	}

	// 2. Create PumpfunAmmPoolConfig
	pumpfunAmmConfig := models.PumpfunAmmPoolConfig{
		PoolAddress:           request.PoolConfig.PoolAddress,
		PoolBump:              request.PoolConfig.PoolBump,
		Index:                 request.PoolConfig.Index,
		Creator:               request.PoolConfig.Creator,
		BaseMint:              request.PoolConfig.BaseMint,
		QuoteMint:             request.PoolConfig.QuoteMint,
		LpMint:                request.PoolConfig.LpMint,
		PoolBaseTokenAccount:  request.PoolConfig.PoolBaseTokenAccount,
		PoolQuoteTokenAccount: request.PoolConfig.PoolQuoteTokenAccount,
		LpSupply:              request.PoolConfig.LpSupply,
		CoinCreator:           request.PoolConfig.CoinCreator,
		Status:                "active",
	}
	if err := tx.Create(&pumpfunAmmConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create PumpfunAmmPoolConfig: " + err.Error()})
		return
	}

	// 3. Generate project name if not provided
	projectName := request.ProjectName
	if projectName == "" {
		// Rule: ${TokenConfig.symbol}-${TokenConfig.mint.slice(0, 5)}
		mintPrefix := request.Mint
		if len(mintPrefix) > 5 {
			mintPrefix = mintPrefix[:5]
		}
		projectName = fmt.Sprintf("%s-%s", tokenConfig.Symbol, mintPrefix)
	}

	// 4. Create ProjectConfig
	projectConfig := models.ProjectConfig{
		Name:              projectName,
		PoolPlatform:      "pumpfun_amm",
		PoolID:            pumpfunAmmConfig.ID,
		TokenID:           tokenConfig.ID,
		TokenMetadataID:   0,
		SnapshotEnabled:   true,
		SnapshotCount:     0,
		IsActive:          true,
		UpdateStatEnabled: true,
	}
	if err := tx.Create(&projectConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ProjectConfig: " + err.Error()})
		return
	}

	// 5. Create ProjectFundTransferRecord for initial token amount
	var fundTransferRecord *models.ProjectFundTransferRecord
	if request.ProjectInitialToken > 0 {
		fundTransferRecord = &models.ProjectFundTransferRecord{
			ProjectID:  projectConfig.ID,
			Mint:       request.Mint,
			Direction:  "in",
			Amount:     request.ProjectInitialToken,
			TargetName: "project",
		}
		if err := tx.Create(fundTransferRecord).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ProjectFundTransferRecord: " + err.Error()})
			return
		}
	}

	// 6. Create RoleConfigRelation
	roleConfigRelation := models.RoleConfigRelation{
		RoleID:    request.RoleID,
		ProjectID: projectConfig.ID,
	}
	if err := tx.Create(&roleConfigRelation).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create RoleConfigRelation: " + err.Error()})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	// Build response
	response := gin.H{
		"message": "Pumpfun AMM project created successfully",
		"data": gin.H{
			"project_config": gin.H{
				"id":                  projectConfig.ID,
				"name":                projectConfig.Name,
				"pool_platform":       projectConfig.PoolPlatform,
				"pool_id":             projectConfig.PoolID,
				"token_id":            projectConfig.TokenID,
				"token_metadata_id":   projectConfig.TokenMetadataID,
				"snapshot_enabled":    projectConfig.SnapshotEnabled,
				"snapshot_count":      projectConfig.SnapshotCount,
				"is_migrated":         projectConfig.IsMigrated,
				"is_active":           projectConfig.IsActive,
				"update_stat_enabled": projectConfig.UpdateStatEnabled,
			},
			"token_config": gin.H{
				"id":           tokenConfig.ID,
				"mint":         tokenConfig.Mint,
				"symbol":       tokenConfig.Symbol,
				"name":         tokenConfig.Name,
				"decimals":     tokenConfig.Decimals,
				"logo_uri":     tokenConfig.LogoURI,
				"total_supply": tokenConfig.TotalSupply,
			},
			"pumpfun_amm_pool_config": gin.H{
				"id":                       pumpfunAmmConfig.ID,
				"pool_address":             pumpfunAmmConfig.PoolAddress,
				"pool_bump":                pumpfunAmmConfig.PoolBump,
				"index":                    pumpfunAmmConfig.Index,
				"creator":                  pumpfunAmmConfig.Creator,
				"base_mint":                pumpfunAmmConfig.BaseMint,
				"quote_mint":               pumpfunAmmConfig.QuoteMint,
				"lp_mint":                  pumpfunAmmConfig.LpMint,
				"pool_base_token_account":  pumpfunAmmConfig.PoolBaseTokenAccount,
				"pool_quote_token_account": pumpfunAmmConfig.PoolQuoteTokenAccount,
				"lp_supply":                pumpfunAmmConfig.LpSupply,
				"coin_creator":             pumpfunAmmConfig.CoinCreator,
				"status":                   pumpfunAmmConfig.Status,
			},
			"role_config_relation": gin.H{
				"role_id":    roleConfigRelation.RoleID,
				"project_id": roleConfigRelation.ProjectID,
			},
		},
	}

	// Add fund_transfer_record to response if it was created
	if fundTransferRecord != nil {
		response["data"].(gin.H)["project_fund_transfer_record"] = gin.H{
			"id":          fundTransferRecord.ID,
			"project_id":  fundTransferRecord.ProjectID,
			"mint":        fundTransferRecord.Mint,
			"direction":   fundTransferRecord.Direction,
			"amount":      fundTransferRecord.Amount,
			"target_name": fundTransferRecord.TargetName,
		}
	}

	c.JSON(http.StatusCreated, response)
}

// CpmmPoolConfig represents the configuration for a Meteoracpmm pool
type CpmmPoolConfig struct {
	PoolAddress           string `json:"pool_address" binding:"required"`
	DbcPoolAddress        string `json:"dbc_pool_address"`
	Creator               string `json:"creator" binding:"required"`
	BaseMint              string `json:"base_mint" binding:"required"`
	QuoteMint             string `json:"quote_mint" binding:"required"`
	PoolBaseTokenAccount  string `json:"pool_base_token_account" binding:"required"`
	PoolQuoteTokenAccount string `json:"pool_quote_token_account" binding:"required"`
	Status                string `json:"status" binding:"required"`
}

// AutoCreateMeteoradbcProjectRequest represents the request body for auto-creating a meteora dbc project
type AutoCreateMeteoradbcProjectRequest struct {
	RoleID     uint `json:"role_id" binding:"required"`
	MintConfig struct {
		Mint        string  `json:"mint" binding:"required"`
		Symbol      string  `json:"symbol" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		Decimals    int     `json:"decimals" binding:"required"`
		LogoURI     string  `json:"logo_uri" binding:"required"`
		TotalSupply float64 `json:"total_supply" binding:"required"`
	} `json:"mint_config" binding:"required"`
	PoolConfig struct {
		PoolAddress           string         `json:"pool_address" binding:"required"`
		Creator               string         `json:"creator" binding:"required"`
		PoolConfig            string         `json:"pool_config" binding:"required"`
		BaseMint              string         `json:"base_mint" binding:"required"`
		QuoteMint             string         `json:"quote_mint" binding:"required"`
		PoolBaseTokenAccount  string         `json:"pool_base_token_account" binding:"required"`
		PoolQuoteTokenAccount string         `json:"pool_quote_token_account" binding:"required"`
		FirstBuyer            string         `json:"first_buyer"`
		Status                string         `json:"status" binding:"required"`
		CpmmPoolConfig        CpmmPoolConfig `json:"cpmm_pool_config"`
		DammV2PoolAddress     string         `json:"damm_v2_pool_address"`
		IsMigrated            bool           `json:"is_migrated"`
	} `json:"pool_config" binding:"required"`
	ProjectName     string `json:"project_name"`      // Optional
	TokenMetadataID uint   `json:"token_metadata_id"` // Optional, defaults to 0
}

// AutoCreateMeteoradbcProjectRequestV2 represents the request body for auto-creating a meteora dbc project (V2 with strategy configs)
type AutoCreateMeteoradbcProjectRequestV2 struct {
	RoleID     uint `json:"role_id" binding:"required"`
	MintConfig struct {
		Mint        string  `json:"mint" binding:"required"`
		Symbol      string  `json:"symbol" binding:"required"`
		Name        string  `json:"name" binding:"required"`
		Decimals    int     `json:"decimals" binding:"required"`
		LogoURI     string  `json:"logo_uri" binding:"required"`
		TotalSupply float64 `json:"total_supply" binding:"required"`
	} `json:"mint_config" binding:"required"`
	PoolConfig struct {
		PoolAddress           string         `json:"pool_address" binding:"required"`
		Creator               string         `json:"creator" binding:"required"`
		PoolConfig            string         `json:"pool_config" binding:"required"`
		BaseMint              string         `json:"base_mint" binding:"required"`
		QuoteMint             string         `json:"quote_mint" binding:"required"`
		PoolBaseTokenAccount  string         `json:"pool_base_token_account" binding:"required"`
		PoolQuoteTokenAccount string         `json:"pool_quote_token_account" binding:"required"`
		FirstBuyer            string         `json:"first_buyer"`
		Status                string         `json:"status" binding:"required"`
		CpmmPoolConfig        CpmmPoolConfig `json:"cpmm_pool_config"`
		DammV2PoolAddress     string         `json:"damm_v2_pool_address"`
		IsMigrated            bool           `json:"is_migrated"`
	} `json:"pool_config" binding:"required"`
	ProjectName     string                  `json:"project_name"`      // Optional
	TokenMetadataID uint                    `json:"token_metadata_id"` // Optional, defaults to 0
	StrategyConfigs []StrategyConfigRequest `json:"strategy_configs"`  // Optional: list of strategy configs to create
}

// AutoCreateMeteoradbcProject automatically creates a complete project setup for Meteora DBC
func AutoCreateMeteoradbcProject(c *gin.Context) {
	var request AutoCreateMeteoradbcProjectRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start a database transaction
	tx := dbconfig.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Find or create TokenConfig by mint
	var tokenConfig models.TokenConfig
	err := tx.Where("mint = ?", request.MintConfig.Mint).First(&tokenConfig).Error
	if err != nil {
		// TokenConfig doesn't exist, create it with provided values
		tokenConfig = models.TokenConfig{
			Mint:        request.MintConfig.Mint,
			Symbol:      request.MintConfig.Symbol,
			Name:        request.MintConfig.Name,
			Decimals:    request.MintConfig.Decimals,
			LogoURI:     request.MintConfig.LogoURI,
			TotalSupply: request.MintConfig.TotalSupply,
		}
		if err := tx.Create(&tokenConfig).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create TokenConfig: " + err.Error()})
			return
		}
	}

	// 2. Create MeteoradbcConfig
	// If CpmmPoolConfig is provided, set DammV2PoolAddress to CpmmPoolConfig.PoolAddress
	dammV2PoolAddress := request.PoolConfig.DammV2PoolAddress
	if request.PoolConfig.CpmmPoolConfig.PoolAddress != "" {
		dammV2PoolAddress = request.PoolConfig.CpmmPoolConfig.PoolAddress
	}

	meteoradbcConfig := models.MeteoradbcConfig{
		PoolAddress:           request.PoolConfig.PoolAddress,
		Creator:               request.PoolConfig.Creator,
		PoolConfig:            request.PoolConfig.PoolConfig,
		BaseMint:              request.PoolConfig.BaseMint,
		QuoteMint:             request.PoolConfig.QuoteMint,
		PoolBaseTokenAccount:  request.PoolConfig.PoolBaseTokenAccount,
		PoolQuoteTokenAccount: request.PoolConfig.PoolQuoteTokenAccount,
		FirstBuyer:            request.PoolConfig.FirstBuyer,
		DammV2PoolAddress:     dammV2PoolAddress,
		IsMigrated:            request.PoolConfig.IsMigrated,
		Status:                request.PoolConfig.Status,
	}
	if err := tx.Create(&meteoradbcConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create MeteoradbcConfig: " + err.Error()})
		return
	}

	// 2.1. Create MeteoracpmmConfig if CpmmPoolConfig is provided
	// 同时创建 MeteoradbcConfig 和 MeteoracpmmConfig
	var meteoracpmmConfig *models.MeteoracpmmConfig
	if request.PoolConfig.CpmmPoolConfig.PoolAddress != "" {
		meteoracpmmConfig = &models.MeteoracpmmConfig{
			PoolAddress:           request.PoolConfig.CpmmPoolConfig.PoolAddress,
			DbcPoolAddress:        request.PoolConfig.CpmmPoolConfig.DbcPoolAddress,
			Creator:               request.PoolConfig.CpmmPoolConfig.Creator,
			BaseMint:              request.PoolConfig.CpmmPoolConfig.BaseMint,
			QuoteMint:             request.PoolConfig.CpmmPoolConfig.QuoteMint,
			PoolBaseTokenAccount:  request.PoolConfig.CpmmPoolConfig.PoolBaseTokenAccount,
			PoolQuoteTokenAccount: request.PoolConfig.CpmmPoolConfig.PoolQuoteTokenAccount,
			Status:                request.PoolConfig.CpmmPoolConfig.Status,
		}
		// If DbcPoolAddress is empty, set it to MeteoradbcConfig.PoolAddress
		if meteoracpmmConfig.DbcPoolAddress == "" {
			meteoracpmmConfig.DbcPoolAddress = meteoradbcConfig.PoolAddress
		}
		if err := tx.Create(meteoracpmmConfig).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create MeteoracpmmConfig: " + err.Error()})
			return
		}
	}

	// 3. Generate project name if not provided
	projectName := request.ProjectName
	if projectName == "" {
		// Rule: ${TokenConfig.symbol}-${TokenConfig.mint.slice(0, 5)}
		mintPrefix := request.MintConfig.Mint
		if len(mintPrefix) > 5 {
			mintPrefix = mintPrefix[:5]
		}
		projectName = fmt.Sprintf("%s-%s", tokenConfig.Symbol, mintPrefix)
	}

	// 4. Create ProjectConfig
	projectConfig := models.ProjectConfig{
		Name:              projectName,
		PoolPlatform:      "meteora_dbc",
		PoolID:            meteoradbcConfig.ID,
		TokenID:           tokenConfig.ID,
		TokenMetadataID:   request.TokenMetadataID,
		SnapshotEnabled:   true,
		SnapshotCount:     0,
		IsActive:          true,
		UpdateStatEnabled: true,
		PoolConfig:        request.PoolConfig.PoolConfig,
	}
	if err := tx.Create(&projectConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ProjectConfig: " + err.Error()})
		return
	}

	// 5. Create RoleConfigRelation
	roleConfigRelation := models.RoleConfigRelation{
		RoleID:    request.RoleID,
		ProjectID: projectConfig.ID,
	}
	if err := tx.Create(&roleConfigRelation).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create RoleConfigRelation: " + err.Error()})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	// Publish monitoring task to RabbitMQ (async, non-blocking)
	go func() {
		if config.RabbitMQ != nil {
			publisher, err := config.NewPublisher()
			if err != nil {
				log.Errorf("Failed to create RabbitMQ publisher: %v", err)
				return
			}
			defer publisher.Close()

			// Prepare monitoring message
			monitorMsg := meteora.PoolMonitorMessage{
				Action:               "start_monitoring",
				MeteoradbcAddress:    meteoradbcConfig.PoolAddress,
				ProjectID:            projectConfig.ID,
				BaseTokenMint:        meteoradbcConfig.BaseMint,
				QuoteTokenMint:       meteoradbcConfig.QuoteMint,
				MeteoraDbcAuthority:  "FhVo3mqL8PW5pH5U2CN4XE33DokiyZnUwuGpH2hmHLuM",
				MeteoraCpmmAuthority: "HLnpSz9h2S4hiLQ43rnSD9XkcUThA7B8hQMKmDaiTLcC",
			}

			// Add Meteoracpmm address if it exists
			if meteoracpmmConfig != nil {
				monitorMsg.MeteoracpmmAddress = meteoracpmmConfig.PoolAddress
				// Use Meteoracpmm token info if available
				if meteoracpmmConfig.BaseMint != "" {
					monitorMsg.BaseTokenMint = meteoracpmmConfig.BaseMint
				}
				if meteoracpmmConfig.QuoteMint != "" {
					monitorMsg.QuoteTokenMint = meteoracpmmConfig.QuoteMint
				}
			}

			// Publish message
			if err := publisher.Publish("meteora_pool_monitor", monitorMsg); err != nil {
				log.Errorf("Failed to publish monitoring message: %v", err)
			} else {
				meteoracpmmAddr := ""
				if meteoracpmmConfig != nil {
					meteoracpmmAddr = meteoracpmmConfig.PoolAddress
				}
				log.Infof("Published monitoring task for project %d: Meteoradbc=%s, Meteoracpmm=%s",
					projectConfig.ID, meteoradbcConfig.PoolAddress, meteoracpmmAddr)
			}
		} else {
			log.Warn("RabbitMQ not initialized, skipping monitoring task publication")
		}
	}()

	// Build response
	response := gin.H{
		"message": "Meteora DBC project created successfully",
		"data": gin.H{
			"project_config": gin.H{
				"id":                  projectConfig.ID,
				"name":                projectConfig.Name,
				"pool_platform":       projectConfig.PoolPlatform,
				"pool_id":             projectConfig.PoolID,
				"token_id":            projectConfig.TokenID,
				"token_metadata_id":   projectConfig.TokenMetadataID,
				"snapshot_enabled":    projectConfig.SnapshotEnabled,
				"snapshot_count":      projectConfig.SnapshotCount,
				"is_migrated":         projectConfig.IsMigrated,
				"is_active":           projectConfig.IsActive,
				"update_stat_enabled": projectConfig.UpdateStatEnabled,
			},
			"token_config": gin.H{
				"id":           tokenConfig.ID,
				"mint":         tokenConfig.Mint,
				"symbol":       tokenConfig.Symbol,
				"name":         tokenConfig.Name,
				"decimals":     tokenConfig.Decimals,
				"logo_uri":     tokenConfig.LogoURI,
				"total_supply": tokenConfig.TotalSupply,
			},
			"meteoradbc_config": gin.H{
				"id":                       meteoradbcConfig.ID,
				"pool_address":             meteoradbcConfig.PoolAddress,
				"creator":                  meteoradbcConfig.Creator,
				"pool_config":              meteoradbcConfig.PoolConfig,
				"base_mint":                meteoradbcConfig.BaseMint,
				"quote_mint":               meteoradbcConfig.QuoteMint,
				"pool_base_token_account":  meteoradbcConfig.PoolBaseTokenAccount,
				"pool_quote_token_account": meteoradbcConfig.PoolQuoteTokenAccount,
				"first_buyer":              meteoradbcConfig.FirstBuyer,
				"damm_v2_pool_address":     meteoradbcConfig.DammV2PoolAddress,
				"is_migrated":              meteoradbcConfig.IsMigrated,
				"status":                   meteoradbcConfig.Status,
			},
			"role_config_relation": gin.H{
				"role_id":    roleConfigRelation.RoleID,
				"project_id": roleConfigRelation.ProjectID,
			},
		},
	}

	// Add MeteoracpmmConfig to response if it was created
	if meteoracpmmConfig != nil {
		response["data"].(gin.H)["meteoracpmm_config"] = gin.H{
			"id":                       meteoracpmmConfig.ID,
			"pool_address":             meteoracpmmConfig.PoolAddress,
			"dbc_pool_address":         meteoracpmmConfig.DbcPoolAddress,
			"creator":                  meteoracpmmConfig.Creator,
			"base_mint":                meteoracpmmConfig.BaseMint,
			"quote_mint":               meteoracpmmConfig.QuoteMint,
			"pool_base_token_account":  meteoracpmmConfig.PoolBaseTokenAccount,
			"pool_quote_token_account": meteoracpmmConfig.PoolQuoteTokenAccount,
			"status":                   meteoracpmmConfig.Status,
		}
	}

	c.JSON(http.StatusCreated, response)
}

// AutoCreateMeteoradbcProjectV2 automatically creates a complete project setup for Meteora DBC with strategy configs
func AutoCreateMeteoradbcProjectV2(c *gin.Context) {
	var request AutoCreateMeteoradbcProjectRequestV2
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Start a database transaction
	tx := dbconfig.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Find or create TokenConfig by mint
	var tokenConfig models.TokenConfig
	err := tx.Where("mint = ?", request.MintConfig.Mint).First(&tokenConfig).Error
	if err != nil {
		// TokenConfig doesn't exist, create it with provided values
		tokenConfig = models.TokenConfig{
			Mint:        request.MintConfig.Mint,
			Symbol:      request.MintConfig.Symbol,
			Name:        request.MintConfig.Name,
			Decimals:    request.MintConfig.Decimals,
			LogoURI:     request.MintConfig.LogoURI,
			TotalSupply: request.MintConfig.TotalSupply,
		}
		if err := tx.Create(&tokenConfig).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create TokenConfig: " + err.Error()})
			return
		}
	}

	// 2. Create MeteoradbcConfig
	// If CpmmPoolConfig is provided, set DammV2PoolAddress to CpmmPoolConfig.PoolAddress
	dammV2PoolAddress := request.PoolConfig.DammV2PoolAddress
	if request.PoolConfig.CpmmPoolConfig.PoolAddress != "" {
		dammV2PoolAddress = request.PoolConfig.CpmmPoolConfig.PoolAddress
	}

	meteoradbcConfig := models.MeteoradbcConfig{
		PoolAddress:           request.PoolConfig.PoolAddress,
		Creator:               request.PoolConfig.Creator,
		PoolConfig:            request.PoolConfig.PoolConfig,
		BaseMint:              request.PoolConfig.BaseMint,
		QuoteMint:             request.PoolConfig.QuoteMint,
		PoolBaseTokenAccount:  request.PoolConfig.PoolBaseTokenAccount,
		PoolQuoteTokenAccount: request.PoolConfig.PoolQuoteTokenAccount,
		FirstBuyer:            request.PoolConfig.FirstBuyer,
		DammV2PoolAddress:     dammV2PoolAddress,
		IsMigrated:            request.PoolConfig.IsMigrated,
		Status:                request.PoolConfig.Status,
	}
	if err := tx.Create(&meteoradbcConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create MeteoradbcConfig: " + err.Error()})
		return
	}

	// 2.1. Create MeteoracpmmConfig if CpmmPoolConfig is provided
	// 同时创建 MeteoradbcConfig 和 MeteoracpmmConfig
	var meteoracpmmConfig *models.MeteoracpmmConfig
	if request.PoolConfig.CpmmPoolConfig.PoolAddress != "" {
		meteoracpmmConfig = &models.MeteoracpmmConfig{
			PoolAddress:           request.PoolConfig.CpmmPoolConfig.PoolAddress,
			DbcPoolAddress:        request.PoolConfig.CpmmPoolConfig.DbcPoolAddress,
			Creator:               request.PoolConfig.CpmmPoolConfig.Creator,
			BaseMint:              request.PoolConfig.CpmmPoolConfig.BaseMint,
			QuoteMint:             request.PoolConfig.CpmmPoolConfig.QuoteMint,
			PoolBaseTokenAccount:  request.PoolConfig.CpmmPoolConfig.PoolBaseTokenAccount,
			PoolQuoteTokenAccount: request.PoolConfig.CpmmPoolConfig.PoolQuoteTokenAccount,
			Status:                request.PoolConfig.CpmmPoolConfig.Status,
		}
		// If DbcPoolAddress is empty, set it to MeteoradbcConfig.PoolAddress
		if meteoracpmmConfig.DbcPoolAddress == "" {
			meteoracpmmConfig.DbcPoolAddress = meteoradbcConfig.PoolAddress
		}
		if err := tx.Create(meteoracpmmConfig).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create MeteoracpmmConfig: " + err.Error()})
			return
		}
	}

	// 3. Generate project name if not provided
	projectName := request.ProjectName
	if projectName == "" {
		// Rule: ${TokenConfig.symbol}-${TokenConfig.mint.slice(0, 5)}
		mintPrefix := request.MintConfig.Mint
		if len(mintPrefix) > 5 {
			mintPrefix = mintPrefix[:5]
		}
		projectName = fmt.Sprintf("%s-%s", tokenConfig.Symbol, mintPrefix)
	}

	// 4. Create ProjectConfig
	projectConfig := models.ProjectConfig{
		Name:              projectName,
		PoolPlatform:      "meteora_dbc",
		PoolID:            meteoradbcConfig.ID,
		TokenID:           tokenConfig.ID,
		TokenMetadataID:   request.TokenMetadataID,
		SnapshotEnabled:   true,
		SnapshotCount:     0,
		IsActive:          true,
		UpdateStatEnabled: true,
		PoolConfig:        request.PoolConfig.PoolConfig,
	}
	if err := tx.Create(&projectConfig).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ProjectConfig: " + err.Error()})
		return
	}

	// 5. Create RoleConfigRelation
	roleConfigRelation := models.RoleConfigRelation{
		RoleID:    request.RoleID,
		ProjectID: projectConfig.ID,
	}
	if err := tx.Create(&roleConfigRelation).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create RoleConfigRelation: " + err.Error()})
		return
	}

	// 6. Create StrategyConfigs if provided
	var createdStrategies []models.StrategyConfig
	if len(request.StrategyConfigs) > 0 {
		for _, strategyReq := range request.StrategyConfigs {
			// Use projectConfig.ID for ProjectID, use strategyReq.RoleID for RoleID
			strategy := models.StrategyConfig{
				ProjectID:      projectConfig.ID,
				RoleID:         strategyReq.RoleID,
				StrategyName:   strategyReq.StrategyName,
				StrategyType:   strategyReq.StrategyType,
				StrategyParams: strategyReq.StrategyParams,
				StrategyStat:   strategyReq.StrategyStat,
			}

			// Set Enabled field - if not provided, use the model's default value (false)
			if strategyReq.Enabled != nil {
				strategy.Enabled = *strategyReq.Enabled
			}

			if err := tx.Create(&strategy).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create StrategyConfig: " + err.Error()})
				return
			}
			createdStrategies = append(createdStrategies, strategy)
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	// Publish monitoring task to RabbitMQ (async, non-blocking)
	go func() {
		if config.RabbitMQ != nil {
			publisher, err := config.NewPublisher()
			if err != nil {
				log.Errorf("Failed to create RabbitMQ publisher: %v", err)
				return
			}
			defer publisher.Close()

			// Prepare monitoring message
			monitorMsg := meteora.PoolMonitorMessage{
				Action:               "start_monitoring",
				MeteoradbcAddress:    meteoradbcConfig.PoolAddress,
				ProjectID:            projectConfig.ID,
				BaseTokenMint:        meteoradbcConfig.BaseMint,
				QuoteTokenMint:       meteoradbcConfig.QuoteMint,
				MeteoraDbcAuthority:  "FhVo3mqL8PW5pH5U2CN4XE33DokiyZnUwuGpH2hmHLuM",
				MeteoraCpmmAuthority: "HLnpSz9h2S4hiLQ43rnSD9XkcUThA7B8hQMKmDaiTLcC",
			}

			// Add Meteoracpmm address if it exists
			if meteoracpmmConfig != nil {
				monitorMsg.MeteoracpmmAddress = meteoracpmmConfig.PoolAddress
				// Use Meteoracpmm token info if available
				if meteoracpmmConfig.BaseMint != "" {
					monitorMsg.BaseTokenMint = meteoracpmmConfig.BaseMint
				}
				if meteoracpmmConfig.QuoteMint != "" {
					monitorMsg.QuoteTokenMint = meteoracpmmConfig.QuoteMint
				}
			}

			// Publish message
			if err := publisher.Publish("meteora_pool_monitor", monitorMsg); err != nil {
				log.Errorf("Failed to publish monitoring message: %v", err)
			} else {
				meteoracpmmAddr := ""
				if meteoracpmmConfig != nil {
					meteoracpmmAddr = meteoracpmmConfig.PoolAddress
				}
				log.Infof("Published monitoring task for project %d: Meteoradbc=%s, Meteoracpmm=%s",
					projectConfig.ID, meteoradbcConfig.PoolAddress, meteoracpmmAddr)
			}
		} else {
			log.Warn("RabbitMQ not initialized, skipping monitoring task publication")
		}
	}()

	// Build response
	response := gin.H{
		"message": "Meteora DBC project created successfully",
		"data": gin.H{
			"project_config": gin.H{
				"id":                  projectConfig.ID,
				"name":                projectConfig.Name,
				"pool_platform":       projectConfig.PoolPlatform,
				"pool_id":             projectConfig.PoolID,
				"token_id":            projectConfig.TokenID,
				"snapshot_enabled":    projectConfig.SnapshotEnabled,
				"snapshot_count":      projectConfig.SnapshotCount,
				"is_migrated":         projectConfig.IsMigrated,
				"is_active":           projectConfig.IsActive,
				"update_stat_enabled": projectConfig.UpdateStatEnabled,
				"pool_config":         projectConfig.PoolConfig,
				"token_metadata_id":   projectConfig.TokenMetadataID,
			},
			"token_config": gin.H{
				"id":           tokenConfig.ID,
				"mint":         tokenConfig.Mint,
				"symbol":       tokenConfig.Symbol,
				"name":         tokenConfig.Name,
				"decimals":     tokenConfig.Decimals,
				"logo_uri":     tokenConfig.LogoURI,
				"total_supply": tokenConfig.TotalSupply,
			},
			"meteoradbc_config": gin.H{
				"id":                       meteoradbcConfig.ID,
				"pool_address":             meteoradbcConfig.PoolAddress,
				"creator":                  meteoradbcConfig.Creator,
				"pool_config":              meteoradbcConfig.PoolConfig,
				"base_mint":                meteoradbcConfig.BaseMint,
				"quote_mint":               meteoradbcConfig.QuoteMint,
				"pool_base_token_account":  meteoradbcConfig.PoolBaseTokenAccount,
				"pool_quote_token_account": meteoradbcConfig.PoolQuoteTokenAccount,
				"first_buyer":              meteoradbcConfig.FirstBuyer,
				"damm_v2_pool_address":     meteoradbcConfig.DammV2PoolAddress,
				"is_migrated":              meteoradbcConfig.IsMigrated,
				"status":                   meteoradbcConfig.Status,
			},
			"role_config_relation": gin.H{
				"role_id":    roleConfigRelation.RoleID,
				"project_id": roleConfigRelation.ProjectID,
			},
		},
	}

	// Add MeteoracpmmConfig to response if it was created
	if meteoracpmmConfig != nil {
		response["data"].(gin.H)["meteoracpmm_config"] = gin.H{
			"id":                       meteoracpmmConfig.ID,
			"pool_address":             meteoracpmmConfig.PoolAddress,
			"dbc_pool_address":         meteoracpmmConfig.DbcPoolAddress,
			"creator":                  meteoracpmmConfig.Creator,
			"base_mint":                meteoracpmmConfig.BaseMint,
			"quote_mint":               meteoracpmmConfig.QuoteMint,
			"pool_base_token_account":  meteoracpmmConfig.PoolBaseTokenAccount,
			"pool_quote_token_account": meteoracpmmConfig.PoolQuoteTokenAccount,
			"status":                   meteoracpmmConfig.Status,
		}
	}

	// Add StrategyConfigs to response if they were created
	if len(createdStrategies) > 0 {
		strategyConfigsList := make([]gin.H, 0, len(createdStrategies))
		for _, strategy := range createdStrategies {
			strategyConfigsList = append(strategyConfigsList, gin.H{
				"id":              strategy.ID,
				"project_id":      strategy.ProjectID,
				"role_id":         strategy.RoleID,
				"strategy_name":   strategy.StrategyName,
				"strategy_type":   strategy.StrategyType,
				"strategy_params": strategy.StrategyParams,
				"strategy_stat":   strategy.StrategyStat,
				"enabled":         strategy.Enabled,
				"created_at":      strategy.CreatedAt,
				"updated_at":      strategy.UpdatedAt,
			})
		}
		response["data"].(gin.H)["strategy_configs"] = strategyConfigsList
	}

	c.JSON(http.StatusCreated, response)
}

// RefillTokenMetadataID fills TokenMetadataID for all ProjectConfigs where TokenMetadataID is 0
func RefillTokenMetadataID(c *gin.Context) {
	// 1. Find all ProjectConfigs where TokenMetadataID is 0
	var projects []models.ProjectConfig
	if err := dbconfig.DB.Where("token_metadata_id = ?", 0).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to query ProjectConfigs: %v", err)})
		return
	}

	if len(projects) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "No ProjectConfigs with TokenMetadataID = 0 found",
			"updated": 0,
		})
		return
	}

	// Start a transaction
	tx := dbconfig.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	updatedCount := 0
	notFoundCount := 0
	var errorMessages []string

	// 2. Process each project
	for _, project := range projects {
		// 2.1. Get TokenConfig by TokenID
		var tokenConfig models.TokenConfig
		if err := tx.First(&tokenConfig, project.TokenID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				notFoundCount++
				errorMessages = append(errorMessages, fmt.Sprintf("ProjectConfig ID %d: TokenConfig not found for TokenID %d", project.ID, project.TokenID))
				continue
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get TokenConfig for ProjectConfig ID %d: %v", project.ID, err)})
			return
		}

		// 2.2. Find TokenMetadata by Name and Symbol
		var tokenMetadata models.TokenMetadata
		if err := tx.Where("name = ? AND symbol = ?", tokenConfig.Name, tokenConfig.Symbol).First(&tokenMetadata).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				notFoundCount++
				errorMessages = append(errorMessages, fmt.Sprintf("ProjectConfig ID %d: TokenMetadata not found for Name='%s', Symbol='%s'", project.ID, tokenConfig.Name, tokenConfig.Symbol))
				continue
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get TokenMetadata for ProjectConfig ID %d: %v", project.ID, err)})
			return
		}

		// 2.3. Update ProjectConfig with TokenMetadata.ID
		if err := tx.Model(&models.ProjectConfig{}).Where("id = ?", project.ID).Update("token_metadata_id", tokenMetadata.ID).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update ProjectConfig ID %d: %v", project.ID, err)})
			return
		}

		updatedCount++
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to commit transaction: %v", err)})
		return
	}

	// Build response
	response := gin.H{
		"message":     "Refill TokenMetadataID completed",
		"total_found": len(projects),
		"updated":     updatedCount,
		"not_found":   notFoundCount,
	}

	// Include errors if any
	if len(errorMessages) > 0 {
		response["error_details"] = errorMessages
	}

	c.JSON(http.StatusOK, response)
}

// UpdateAssetsBalanceRequest represents the request body for updating assets balance
type UpdateAssetsBalanceRequest struct {
	ProjectID     uint    `json:"project_id" binding:"required"`
	AssetsBalance float64 `json:"assets_balance" binding:"required"`
}

// UpdateAssetsBalance updates the assets balance for a project
func UpdateAssetsBalance(c *gin.Context) {
	var request UpdateAssetsBalanceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify project exists
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, request.ProjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Update assets balance
	project.AssetsBalance = request.AssetsBalance
	if err := dbconfig.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload project with associations
	if err := dbconfig.DB.Preload("Token").First(&project, project.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project associations"})
		return
	}

	// Build response
	resp := buildProjectConfigResp(&project)

	// If ProjectProfit < -0.2, set IsLocked to true
	// if resp != nil && resp.ProjectProfit < -0.25 {
	// 	project.IsLocked = true
	// 	if err := dbconfig.DB.Save(&project).Error; err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update IsLocked: " + err.Error()})
	// 		return
	// 	}
	// 	// Reload project and rebuild response
	// 	if err := dbconfig.DB.Preload("Token").First(&project, project.ID).Error; err == nil {
	// 		resp = buildProjectConfigResp(&project)
	// 	}
	// }

	c.JSON(http.StatusOK, resp)
}

// UpdateVestingRequest represents the request body for updating vesting
type UpdateVestingRequest struct {
	ProjectID uint            `json:"project_id" binding:"required"`
	Vesting   json.RawMessage `json:"vesting" binding:"required"`
}

// UpdateVesting updates the vesting for a project
func UpdateVesting(c *gin.Context) {
	var request UpdateVestingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify project exists
	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, request.ProjectID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Update vesting
	project.Vesting = request.Vesting
	if err := dbconfig.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload project with associations
	if err := dbconfig.DB.Preload("Token").First(&project, project.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project associations"})
		return
	}

	// Build response
	resp := buildProjectConfigResp(&project)
	c.JSON(http.StatusOK, resp)
}

// ToggleProjectConfigLocker toggles the IsLocked field for a project config
func ToggleProjectConfigLocker(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var project models.ProjectConfig
	if err := dbconfig.DB.First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Toggle IsLocked
	project.IsLocked = !project.IsLocked
	if err := dbconfig.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reload project with associations
	if err := dbconfig.DB.Preload("Token").First(&project, project.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load project associations"})
		return
	}

	// Build response
	resp := buildProjectConfigResp(&project)
	c.JSON(http.StatusOK, resp)
}
