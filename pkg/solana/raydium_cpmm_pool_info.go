package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gagliardetto/solana-go"
)

// CpmmPoolInfo represents the CPMM pool information structure
type CpmmPoolInfo struct {
	Type          string             `json:"type"`
	ProgramId     string             `json:"programId"`
	Id            string             `json:"id"`
	MintA         TokenInfo          `json:"mintA"`
	MintB         TokenInfo          `json:"mintB"`
	Price         float64            `json:"price"`
	MintAmountA   float64            `json:"mintAmountA"`
	MintAmountB   float64            `json:"mintAmountB"`
	FeeRate       float64            `json:"feeRate"`
	OpenTime      string             `json:"openTime"`
	TvlUsd        float64            `json:"tvlUsd"`
	Day           PriceChangeInfo    `json:"day"`
	Week          PriceChangeInfo    `json:"week"`
	Month         PriceChangeInfo    `json:"month"`
	PoolType      []string           `json:"pooltype"`
	RewardDefaultInfos []interface{} `json:"rewardDefaultInfos"`
	FarmUpcomingCount  int           `json:"farmUpcomingCount"`
	FarmOngoingCount   int           `json:"farmOngoingCount"`
	FarmFinishedCount  int           `json:"farmFinishedCount"`
	MarketId      string             `json:"marketId"`
	LpMint        TokenInfo          `json:"lpMint"`
	LpPrice       float64            `json:"lpPrice"`
	LpAmount      float64            `json:"lpAmount"`
	BurnPercent   float64            `json:"burnPercent"`
}

// TokenInfo represents token information
type TokenInfo struct {
	ChainId     int    `json:"chainId"`
	Address     string `json:"address"`
	ProgramId   string `json:"programId"`
	LogoURI     string `json:"logoURI"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    int    `json:"decimals"`
	Tags        []string `json:"tags"`
	Extensions  map[string]interface{} `json:"extensions"`
}

// PriceChangeInfo represents price change information
type PriceChangeInfo struct {
	Volume      float64 `json:"volume"`
	VolumeQuote float64 `json:"volumeQuote"`
	VolumeFee   float64 `json:"volumeFee"`
	Apr         float64 `json:"apr"`
	FeeApr      float64 `json:"feeApr"`
	PriceChange float64 `json:"priceChange"`
	PriceMin    float64 `json:"priceMin"`
	PriceMax    float64 `json:"priceMax"`
}

// RaydiumAPIResponse represents the response from Raydium API
type RaydiumAPIResponse struct {
	Id      string         `json:"id"`
	Success bool           `json:"success"`
	Data    []CpmmPoolInfo `json:"data"`
}

// RaydiumAPIClient handles API calls to Raydium
type RaydiumAPIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewRaydiumAPIClient creates a new Raydium API client
func NewRaydiumAPIClient() *RaydiumAPIClient {
	return &RaydiumAPIClient{
		BaseURL: "https://api-v3.raydium.io",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchPoolById fetches pool information by pool ID using Raydium API
func (client *RaydiumAPIClient) FetchPoolById(poolIds []string) ([]CpmmPoolInfo, error) {
	if len(poolIds) == 0 {
		return nil, fmt.Errorf("pool IDs cannot be empty")
	}

	// Create request URL with pool IDs as query parameters
	url := fmt.Sprintf("%s/pools/info/ids", client.BaseURL)
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	for _, id := range poolIds {
		q.Add("ids", id)
	}
	req.URL.RawQuery = q.Encode()

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "MarketStream-Go/1.0")

	// Make the request
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var apiResponse RaydiumAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResponse.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	return apiResponse.Data, nil
}

// GetCpmmPoolInfo fetches CPMM pool information by pool ID
func GetCpmmPoolInfo(poolId string) (*CpmmPoolInfo, error) {
	// Validate pool ID
	if poolId == "" {
		return nil, fmt.Errorf("pool ID cannot be empty")
	}

	// Validate that it's a valid Solana public key
	_, err := solana.PublicKeyFromBase58(poolId)
	if err != nil {
		return nil, fmt.Errorf("invalid pool ID format: %w", err)
	}

	// Create API client
	client := NewRaydiumAPIClient()

	// Fetch pool data
	pools, err := client.FetchPoolById([]string{poolId})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pool data: %w", err)
	}

	if len(pools) == 0 {
		return nil, fmt.Errorf("pool not found")
	}

	// Return the first (and should be only) pool
	return &pools[0], nil
}

// GetCpmmPoolInfoBatch fetches multiple CPMM pool information by pool IDs
func GetCpmmPoolInfoBatch(poolIds []string) ([]CpmmPoolInfo, error) {
	if len(poolIds) == 0 {
		return nil, fmt.Errorf("pool IDs cannot be empty")
	}

	// Validate all pool IDs
	for i, poolId := range poolIds {
		if poolId == "" {
			return nil, fmt.Errorf("pool ID at index %d cannot be empty", i)
		}
		_, err := solana.PublicKeyFromBase58(poolId)
		if err != nil {
			return nil, fmt.Errorf("invalid pool ID format at index %d: %w", i, err)
		}
	}

	// Create API client
	client := NewRaydiumAPIClient()

	// Fetch pool data
	return client.FetchPoolById(poolIds)
}

// Example function that mimics the TypeScript getCpmmPoolInfo
func getCpmmPoolInfoExample() (*CpmmPoolInfo, error) {
	// Example pool ID from TypeScript code (SOL-USDC pool)
	poolId := "5d5nHo5FsT2wGxpi4SryiTohmkebs8Ro4XJ3ofpeKPBq"
	
	poolInfo, err := GetCpmmPoolInfo(poolId)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPMM pool info: %w", err)
	}

	// Print pool info (similar to console.log in TypeScript)
	fmt.Printf("CPMM Pool Info: %+v\n", poolInfo)
	
	return poolInfo, nil
}