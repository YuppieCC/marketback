package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gin-gonic/gin"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/helius"
	mcsolana "marketcontrol/pkg/solana"
)

// TokenConfigRequest represents the request body for token config operations
type TokenConfigRequest struct {
	Mint        string   `json:"mint" binding:"required"`
	Symbol      *string  `json:"symbol"`
	Name        *string  `json:"name"`
	Decimals    *int     `json:"decimals"`
	LogoURI     *string  `json:"logo_uri"`
	TotalSupply *float64 `json:"total_supply"`
	Creator     *string  `json:"creator"`
}

// sanitizeString removes null bytes and ensures valid UTF-8
func sanitizeString(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")

	// Ensure valid UTF-8
	if !utf8.ValidString(s) {
		// Convert to valid UTF-8 by replacing invalid sequences
		v := make([]rune, 0, len(s))
		for i, r := range s {
			if r == utf8.RuneError {
				if i+2 < len(s) {
					// Skip the next byte as it may be part of the invalid sequence
					continue
				}
				// Replace invalid runes with a space
				v = append(v, ' ')
			} else {
				v = append(v, r)
			}
		}
		return string(v)
	}
	return s
}

// validateAndSanitizeURL checks if the URL is valid and returns a sanitized version
func validateAndSanitizeURL(uri string) string {
	// Check if the URL is valid
	u, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	// Only allow http and https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}

	// Return the sanitized URL
	return u.String()
}

// ListTokenConfigs returns a list of all token configs
func ListTokenConfigs(c *gin.Context) {
	var tokens []models.TokenConfig
	if err := dbconfig.DB.Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tokens)
}

// ListTokenConfigsSlice returns a slice of token configs
func ListTokenConfigsSlice(c *gin.Context) {
	// Get pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Get sorting parameters
	orderField := c.DefaultQuery("order_field", "created_at")
	orderType := c.DefaultQuery("order_type", "desc")

	// Validate order_field (whitelist allowed fields for security)
	allowedFields := map[string]bool{
		"id":           true,
		"mint":         true,
		"symbol":       true,
		"name":         true,
		"decimals":     true,
		"total_supply": true,
		"creator":      true,
		"created_at":   true,
		"updated_at":   true,
	}

	if !allowedFields[orderField] {
		orderField = "created_at"
	}

	// Validate order_type
	if orderType != "asc" && orderType != "desc" {
		orderType = "desc"
	}

	// Build order clause
	orderClause := orderField + " " + orderType

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count
	var totalCount int64
	if err := dbconfig.DB.Model(&models.TokenConfig{}).Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated and sorted data
	var tokens []models.TokenConfig
	if err := dbconfig.DB.Order(orderClause).Offset(offset).Limit(pageSize).Find(&tokens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate pagination info
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	hasNext := page < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"data": tokens,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
		"sorting": gin.H{
			"order_field": orderField,
			"order_type":  orderType,
		},
	})
}

// GetTokenConfig returns a specific token config by ID
func GetTokenConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var token models.TokenConfig
	if err := dbconfig.DB.First(&token, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, token)
}

// GetTokenConfigByMint returns a specific token config by mint
func GetTokenConfigByMint(c *gin.Context) {
	mint := c.Param("mint")
	var token models.TokenConfig
	if err := dbconfig.DB.Where("mint = ?", mint).First(&token).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, token)
}

// CreateTokenConfig creates a new token config
func CreateTokenConfig(c *gin.Context) {
	var request TokenConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Initialize variables for token data
	var tokenSymbol, tokenName, tokenLogoURI, tokenCreator string
	var tokenDecimals int
	var tokenTotalSupply float64

	// Use provided values or fetch from blockchain
	needsMetadata := request.Symbol == nil || request.Name == nil || request.Creator == nil
	needsSupply := request.Decimals == nil || request.TotalSupply == nil

	if needsMetadata || needsSupply {
		// Get Helius API key from environment
		heliusApiKey := os.Getenv("HELIUS_API_KEY")
		if heliusApiKey == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Helius API key not configured"})
			return
		}

		// Get Solana RPC endpoint from environment
		solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
		if solanaRPC == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
			return
		}

		// Create clients
		heliusClient := helius.NewClient(heliusApiKey)
		solanaClient := rpc.New(solanaRPC)

		// Parse mint address
		mintPubkey := solana.MustPublicKeyFromBase58(request.Mint)

		// Get token metadata if needed
		if needsMetadata {
			metadata, err := mcsolana.GetTokenMetadata(solanaClient, mintPubkey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token metadata: %v", err)})
				return
			}

			// Use blockchain data as fallback
			if request.Symbol == nil {
				tokenSymbol = sanitizeString(metadata.Symbol)
			}
			if request.Name == nil {
				tokenName = sanitizeString(metadata.Name)
			}
			if request.Creator == nil {
				tokenCreator = sanitizeString(metadata.Creator)
			}
			// For LogoURI, use blockchain metadata URI as fallback only if not provided
			if request.LogoURI == nil {
				tokenLogoURI = validateAndSanitizeURL(metadata.Uri)
			}
		}

		// Get token supply if needed
		if needsSupply {
			supply, err := heliusClient.GetTokenSupply(request.Mint)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token supply: %v", err)})
				return
			}

			if request.Decimals == nil {
				tokenDecimals = supply.Decimals
			}
			if request.TotalSupply == nil {
				tokenTotalSupply = supply.UiAmount
			}
		}
	}

	// Use provided values or fallback to blockchain data
	if request.Symbol != nil {
		tokenSymbol = sanitizeString(*request.Symbol)
	}
	if request.Name != nil {
		tokenName = sanitizeString(*request.Name)
	}
	if request.LogoURI != nil {
		tokenLogoURI = validateAndSanitizeURL(*request.LogoURI)
	}
	if request.Creator != nil {
		tokenCreator = sanitizeString(*request.Creator)
	}
	if request.Decimals != nil {
		tokenDecimals = *request.Decimals
	}
	if request.TotalSupply != nil {
		tokenTotalSupply = *request.TotalSupply
	}

	// Create token config with final values
	token := models.TokenConfig{
		Mint:        request.Mint,
		Symbol:      tokenSymbol,
		Name:        tokenName,
		Decimals:    tokenDecimals,
		LogoURI:     tokenLogoURI,
		TotalSupply: tokenTotalSupply,
		Creator:     tokenCreator,
	}

	if err := dbconfig.DB.Create(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, token)
}

// UpdateTokenConfig updates an existing token config
func UpdateTokenConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request TokenConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var token models.TokenConfig
	if err := dbconfig.DB.First(&token, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Get Helius API key from environment
	heliusApiKey := os.Getenv("HELIUS_API_KEY")
	if heliusApiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helius API key not configured"})
		return
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create clients
	heliusClient := helius.NewClient(heliusApiKey)
	solanaClient := rpc.New(solanaRPC)

	// Parse mint address
	mintPubkey := solana.MustPublicKeyFromBase58(request.Mint)

	// Get token metadata
	metadata, err := mcsolana.GetTokenMetadata(solanaClient, mintPubkey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token metadata: %v", err)})
		return
	}

	// Get token supply
	supply, err := heliusClient.GetTokenSupply(request.Mint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token supply: %v", err)})
		return
	}

	// Sanitize the data
	sanitizedName := sanitizeString(metadata.Name)
	sanitizedSymbol := sanitizeString(metadata.Symbol)
	sanitizedLogoURI := validateAndSanitizeURL(metadata.Uri)

	// Update token config with sanitized data
	token.Mint = request.Mint
	token.Symbol = sanitizedSymbol
	token.Name = sanitizedName
	token.Decimals = supply.Decimals
	token.LogoURI = sanitizedLogoURI
	token.TotalSupply = supply.UiAmount

	if err := dbconfig.DB.Save(&token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, token)
}

// DeleteTokenConfig deletes a token config
func DeleteTokenConfig(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	// Check if any project is using this token
	var projectCount int64
	if err := dbconfig.DB.Model(&models.ProjectConfig{}).
		Where("token_id = ?", id).
		Count(&projectCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check project dependencies"})
		return
	}

	if projectCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Cannot delete token: there are projects using this token",
			"project_count": projectCount,
		})
		return
	}

	if err := dbconfig.DB.Delete(&models.TokenConfig{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token config deleted successfully"})
}

// TokenMetadataRequest represents the request body for token metadata operations
type TokenMetadataRequest struct {
	Name        string                 `json:"name"`
	Symbol      string                 `json:"symbol"`
	Description string                 `json:"description"`
	Image       string                 `json:"image"`
	Twitter     string                 `json:"twitter"`
	Telegram    string                 `json:"telegram"`
	Website     string                 `json:"website"`
	SourceURL   string                 `json:"source_url"`
	SourceData  map[string]interface{} `json:"source_data"`
	IsFavorite  *bool                  `json:"is_favorite"`
}

// ListTokenMetadata returns a list of all token metadata
func ListTokenMetadata(c *gin.Context) {
	// Get pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count
	var totalCount int64
	if err := dbconfig.DB.Model(&models.TokenMetadata{}).Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated data
	var metadata []models.TokenMetadata
	if err := dbconfig.DB.Order("id DESC").Offset(offset).Limit(pageSize).Find(&metadata).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate pagination info
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	hasNext := page < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"data": metadata,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	})
}

// GetFavorites returns all token metadata where IsFavorite is true
func GetFavorites(c *gin.Context) {
	// Get pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get total count of favorites
	var totalCount int64
	if err := dbconfig.DB.Model(&models.TokenMetadata{}).
		Where("is_favorite = ?", true).
		Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get paginated favorites data
	var metadata []models.TokenMetadata
	if err := dbconfig.DB.Where("is_favorite = ?", true).
		Order("id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&metadata).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate pagination info
	totalPages := int(math.Ceil(float64(totalCount) / float64(pageSize)))
	hasNext := page < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"data": metadata,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": totalPages,
			"has_next":    hasNext,
			"has_prev":    hasPrev,
		},
	})
}

// GetTokenMetadata returns a specific token metadata by ID
func GetTokenMetadata(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var metadata models.TokenMetadata
	if err := dbconfig.DB.First(&metadata, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, metadata)
}

// GetTokenMetadataBySymbol returns a specific token metadata by symbol
func GetTokenMetadataBySymbol(c *gin.Context) {
	symbol := c.Param("symbol")
	var metadata models.TokenMetadata
	if err := dbconfig.DB.Where("symbol = ?", symbol).First(&metadata).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}
	c.JSON(http.StatusOK, metadata)
}

// CreateTokenMetadata creates a new token metadata
func CreateTokenMetadata(c *gin.Context) {
	var request TokenMetadataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if symbol already exists (only if symbol is provided)
	// if request.Symbol != "" {
	// 	var existingCount int64
	// 	if err := dbconfig.DB.Model(&models.TokenMetadata{}).
	// 		Where("symbol = ?", request.Symbol).
	// 		Count(&existingCount).Error; err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing symbol"})
	// 		return
	// 	}

	// 	if existingCount > 0 {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol already exists"})
	// 		return
	// 	}
	// }

	// Set default IsFavorite value
	isFavorite := false
	if request.IsFavorite != nil {
		isFavorite = *request.IsFavorite
	}

	// Create token metadata
	metadata := models.TokenMetadata{
		Name:        request.Name,
		Symbol:      request.Symbol,
		Description: request.Description,
		Image:       request.Image,
		Twitter:     request.Twitter,
		Telegram:    request.Telegram,
		Website:     request.Website,
		SourceURL:   request.SourceURL,
		SourceData:  models.JSONB(request.SourceData),
		IsFavorite:  isFavorite,
	}

	if err := dbconfig.DB.Create(&metadata).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, metadata)
}

// UpdateTokenMetadata updates an existing token metadata
func UpdateTokenMetadata(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	var request TokenMetadataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var metadata models.TokenMetadata
	if err := dbconfig.DB.First(&metadata, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Check if the new symbol conflicts with existing records (excluding current record)
	// Only check if symbol is provided and different from current
	if request.Symbol != "" && request.Symbol != metadata.Symbol {
		var existingCount int64
		if err := dbconfig.DB.Model(&models.TokenMetadata{}).
			Where("symbol = ? AND id != ?", request.Symbol, id).
			Count(&existingCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing symbol"})
			return
		}

		if existingCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol already exists"})
			return
		}
	}

	// Update token metadata (only update fields that are provided)
	if request.Name != "" {
		metadata.Name = request.Name
	}
	if request.Symbol != "" {
		metadata.Symbol = request.Symbol
	}
	if request.Description != "" {
		metadata.Description = request.Description
	}
	if request.Image != "" {
		metadata.Image = request.Image
	}
	if request.Twitter != "" {
		metadata.Twitter = request.Twitter
	}
	if request.Telegram != "" {
		metadata.Telegram = request.Telegram
	}
	if request.Website != "" {
		metadata.Website = request.Website
	}
	if request.SourceURL != "" {
		metadata.SourceURL = request.SourceURL
	}
	if request.SourceData != nil {
		metadata.SourceData = models.JSONB(request.SourceData)
	}
	if request.IsFavorite != nil {
		metadata.IsFavorite = *request.IsFavorite
	}

	if err := dbconfig.DB.Save(&metadata).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metadata)
}

// DeleteTokenMetadata deletes a token metadata
func DeleteTokenMetadata(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
		return
	}

	if err := dbconfig.DB.Delete(&models.TokenMetadata{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Token metadata deleted successfully"})
}

// FetchTokenMetadataByURLRequest represents the request body for fetching token metadata by URL
type FetchTokenMetadataByURLRequest struct {
	URL string `json:"url" binding:"required"`
}

// URLMetadataResponse represents the response structure from the metadata URL
type URLMetadataResponse struct {
	Name        string      `json:"name"`
	Symbol      string      `json:"symbol"`
	Description string      `json:"description"`
	Image       string      `json:"image"`
	Twitter     string      `json:"twitter"`
	Telegram    string      `json:"telegram"`
	Website     string      `json:"website"`
	ShowName    interface{} `json:"showName"`
	CreatedOn   string      `json:"createdOn"`
}

// FetchTokenMetadataByURL fetches token metadata from a URL and stores it in the database
func FetchTokenMetadataByURL(c *gin.Context) {
	var request FetchTokenMetadataByURLRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch metadata from URL
	resp, err := http.Get(request.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch metadata from URL: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to fetch metadata: HTTP %d", resp.StatusCode)})
		return
	}

	// Parse the response
	var urlMetadata URLMetadataResponse
	if err := json.NewDecoder(resp.Body).Decode(&urlMetadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse metadata response: %v", err)})
		return
	}

	// Check if symbol already exists
	// var existingCount int64
	// if err := dbconfig.DB.Model(&models.TokenMetadata{}).
	// 	Where("symbol = ?", urlMetadata.Symbol).
	// 	Count(&existingCount).Error; err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing symbol"})
	// 	return
	// }

	// if existingCount > 0 {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Symbol already exists"})
	// 	return
	// }

	// Normalize ShowName to bool if provided as string/number
	var showNameBool bool
	switch v := urlMetadata.ShowName.(type) {
	case bool:
		showNameBool = v
	case string:
		showNameBool = strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
	case float64:
		showNameBool = v != 0
	default:
		showNameBool = false
	}

	// Prepare source data
	sourceData := models.JSONB{
		"showName":  showNameBool,
		"createdOn": urlMetadata.CreatedOn,
	}

	// Handle social media fields - use provided values or default to "null"
	twitter := getStringValue(urlMetadata.Twitter)
	telegram := getStringValue(urlMetadata.Telegram)
	website := getStringValue(urlMetadata.Website)

	// Create token metadata with source URL and optional social media fields
	desc := getStringValue(urlMetadata.Description)
	desc = truncateString(desc, 512)
	metadata := models.TokenMetadata{
		Name:        getStringValue(urlMetadata.Name),
		Symbol:      getStringValue(urlMetadata.Symbol),
		Description: desc,
		Image:       getStringValue(urlMetadata.Image),
		Twitter:     twitter,
		Telegram:    telegram,
		Website:     website,
		SourceURL:   request.URL,
		SourceData:  sourceData,
	}

	if err := dbconfig.DB.Create(&metadata).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, metadata)
}

// FetchTokenMetadataByMintRequest represents the request body for fetching token metadata by mint
type FetchTokenMetadataByMintRequest struct {
	Mint string `json:"mint" binding:"required"`
}

// HeliusAssetResponse represents the response structure from Helius getAsset API
type HeliusAssetResponse struct {
	JsonRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  struct {
		Content struct {
			JSONURI string `json:"json_uri"`
		} `json:"content"`
	} `json:"result"`
}

// FetchTokenMetadataByMint fetches token metadata by mint address using Helius API
func FetchTokenMetadataByMint(c *gin.Context) {
	var request FetchTokenMetadataByMintRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Helius API key from environment
	heliusApiKey := os.Getenv("HELIUS_API_KEY")
	if heliusApiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helius API key not configured"})
		return
	}

	// Prepare the JSON-RPC request payload
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "1",
		"method":  "getAsset",
		"params": map[string]interface{}{
			"id": request.Mint,
			"options": map[string]interface{}{
				"showUnverifiedCollections": false,
				"showCollectionMetadata":    false,
				"showFungible":              false,
				"showInscription":           false,
			},
		},
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to marshal request: %v", err)})
		return
	}

	// Create request with RPC URL
	rpcURLWithKey := fmt.Sprintf("%s/?api-key=%s", "https://mainnet.helius-rpc.com", heliusApiKey)
	httpReq, err := http.NewRequest("POST", rpcURLWithKey, bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create request: %v", err)})
		return
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request using default HTTP client
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send request: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("API request failed with status code: %d", resp.StatusCode),
			"body":  string(bodyBytes),
		})
		return
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read response: %v", err)})
		return
	}

	// Parse Helius response
	var heliusResp HeliusAssetResponse
	if err := json.Unmarshal(bodyBytes, &heliusResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse Helius response: %v", err)})
		return
	}

	// Check if json_uri exists
	jsonURI := heliusResp.Result.Content.JSONURI
	if jsonURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No json_uri found in asset metadata"})
		return
	}

	// Fetch metadata from json_uri
	metadataResp, err := http.Get(jsonURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch metadata from URL: %v", err)})
		return
	}
	defer metadataResp.Body.Close()

	if metadataResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to fetch metadata: HTTP %d", metadataResp.StatusCode)})
		return
	}

	// Parse the metadata response
	var urlMetadata URLMetadataResponse
	if err := json.NewDecoder(metadataResp.Body).Decode(&urlMetadata); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse metadata response: %v", err)})
		return
	}

	// Normalize ShowName to bool if provided as string/number
	var showNameBool bool
	switch v := urlMetadata.ShowName.(type) {
	case bool:
		showNameBool = v
	case string:
		showNameBool = strings.EqualFold(v, "true") || v == "1" || strings.EqualFold(v, "yes")
	case float64:
		showNameBool = v != 0
	default:
		showNameBool = false
	}

	// Prepare source data
	sourceData := models.JSONB{
		"showName":  showNameBool,
		"createdOn": urlMetadata.CreatedOn,
		"mint":      request.Mint,
	}

	// Handle social media fields - use provided values or default to "null"
	twitter := getStringValue(urlMetadata.Twitter)
	telegram := getStringValue(urlMetadata.Telegram)
	website := getStringValue(urlMetadata.Website)

	// Create token metadata with source URL and optional social media fields
	desc := getStringValue(urlMetadata.Description)
	desc = truncateString(desc, 512)
	metadata := models.TokenMetadata{
		Name:        getStringValue(urlMetadata.Name),
		Symbol:      getStringValue(urlMetadata.Symbol),
		Description: desc,
		Image:       getStringValue(urlMetadata.Image),
		Twitter:     twitter,
		Telegram:    telegram,
		Website:     website,
		SourceURL:   jsonURI,
		SourceData:  sourceData,
	}

	if err := dbconfig.DB.Create(&metadata).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, metadata)
}

// getStringValue returns "null" if the string is empty, otherwise returns the string
func getStringValue(s string) string {
	if s == "" {
		return "null"
	}
	return s
}

// truncateString limits a string to at most max runes (unicode-safe)
func truncateString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

// GetRandomTokenMetadataRequest represents the request body for getting a random token metadata
type GetRandomTokenMetadataRequest struct {
	IsFavorite *bool `json:"is_favorite"`
}

// GetRandomTokenMetadata returns a random token metadata record based on is_favorite filter
func GetRandomTokenMetadata(c *gin.Context) {
	var request GetRandomTokenMetadataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var metadata models.TokenMetadata
	query := dbconfig.DB.Model(&models.TokenMetadata{})

	// Filter by is_favorite if provided
	if request.IsFavorite != nil {
		query = query.Where("is_favorite = ?", *request.IsFavorite)
	}

	// Use PostgreSQL's RANDOM() function to get a random record
	// ORDER BY RANDOM() LIMIT 1
	if err := query.Order("RANDOM()").Limit(1).First(&metadata).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No token metadata found matching the criteria"})
		return
	}

	c.JSON(http.StatusOK, metadata)
}
