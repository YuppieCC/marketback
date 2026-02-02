package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"marketcontrol/internal/models"
	"marketcontrol/pkg/config"
	dbconfig "marketcontrol/pkg/config"
	"marketcontrol/pkg/helius"
	mcsolana "marketcontrol/pkg/solana"
	"marketcontrol/pkg/solana/meteora"
	"marketcontrol/pkg/utils"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type AmmRequest struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	DX   float64 `json:"dx"`
	DY   float64 `json:"dy"`
	Fee  float64 `json:"fee"`
	Mode string  `json:"mode"`
}

type AmmResponse struct {
	Result float64 `json:"result"`
}

// SimulateAmountOutRequest 用于模拟已知输入的兑换
type SimulateAmountOutRequest struct {
	AmountIn  float64 `json:"amountIn"`
	InputType string  `json:"inputType"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Fee       float64 `json:"fee"`
}

// SimulateAmountInRequest 用于模拟已知输出的兑换
type SimulateAmountInRequest struct {
	AmountOut  float64 `json:"amountOut"`
	OutputType string  `json:"outputType"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Fee        float64 `json:"fee"`
}

// GetEnhancedTransactionsRequest represents the request body for getting enhanced transactions
type GetEnhancedTransactionsRequest struct {
	Address string                    `json:"address" binding:"required"`
	Options helius.TransactionOptions `json:"options"`
}

// GetEnhancedTransactionsByAddressHandler handles requests to get enhanced transactions from Helius API
func GetEnhancedTransactionsByAddressHandler(c *gin.Context) {
	var request GetEnhancedTransactionsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get API key from environment variable
	apiKey := os.Getenv("HELIUS_API_KEY")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helius API key not configured"})
		return
	}

	// Create Helius client
	client := helius.NewClient(apiKey)

	// Get transactions
	transactions, err := client.GetEnhancedTransactionsByAddress(request.Address, &request.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// SimulateAmountOutHandler 处理已知输入模拟
func SimulateAmountOutHandler(c *gin.Context) {
	var req SimulateAmountOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := utils.SimulateConstantProductAmountOut(req.AmountIn, req.InputType, req.X, req.Y, req.Fee)
	c.JSON(http.StatusOK, AmmResponse{Result: result})
}

// SimulateAmountInHandler 处理已知输出模拟
func SimulateAmountInHandler(c *gin.Context) {
	var req SimulateAmountInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := utils.SimulateConstantProductAmountIn(req.AmountOut, req.OutputType, req.X, req.Y, req.Fee)
	c.JSON(http.StatusOK, AmmResponse{Result: result})
}

// GetTokenSupplyRequest represents the request body for getting token supply
type GetTokenSupplyRequest struct {
	Mint string `json:"mint" binding:"required"`
}

// GetTokenSupplyResponse represents the response for getting token supply
type GetTokenSupplyResponse struct {
	Amount         string  `json:"amount"`
	Decimals       int     `json:"decimals"`
	UiAmount       float64 `json:"uiAmount"`
	UiAmountString string  `json:"uiAmountString"`
}

// GetTokenSupplyHandler handles requests to get token supply
func GetTokenSupplyHandler(c *gin.Context) {
	var req GetTokenSupplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Helius API key from environment
	heliusApiKey := os.Getenv("HELIUS_API_KEY")
	if heliusApiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helius API key not configured"})
		return
	}

	// Create Helius client
	client := helius.NewClient(heliusApiKey)

	// Get token supply
	supply, err := client.GetTokenSupply(req.Mint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token supply: %v", err)})
		return
	}

	if supply == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Token supply not found"})
		return
	}

	// Return response
	c.JSON(http.StatusOK, GetTokenSupplyResponse{
		Amount:         supply.Amount,
		Decimals:       supply.Decimals,
		UiAmount:       supply.UiAmount,
		UiAmountString: supply.UiAmountString,
	})
}

// GetTokenInfoRequest represents the request body for getting token information
type GetTokenInfoRequest struct {
	Mint string `json:"mint" binding:"required"`
}

// GetTokenInfoResponse represents the combined token information
type GetTokenInfoResponse struct {
	// Metadata fields
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Description string `json:"description"`
	Image       string `json:"image"`

	// Supply fields
	TotalSupply         string  `json:"totalSupply"`
	ToTalSupplyReadable float64 `json:"totalSupplyReadable"`
	Decimals            int     `json:"decimals"`
	UiSupply            float64 `json:"uiSupply"`
	UiSupplyString      string  `json:"uiSupplyString"`
}

// GetTokenInfoHandler handles requests to get combined token information
func GetTokenInfoHandler(c *gin.Context) {
	var req GetTokenInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create client
	solanaClient := rpc.New(solanaRPC)

	// Create Helius client
	heliusApiKey := os.Getenv("HELIUS_API_KEY")
	if heliusApiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helius API key not configured"})
		return
	}

	// Create clients
	heliusClient := helius.NewClient(heliusApiKey)

	// Parse mint address
	mintPubkey := solana.MustPublicKeyFromBase58(req.Mint)

	// Get token metadata
	metadata, err := mcsolana.GetTokenMetadata(solanaClient, mintPubkey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token metadata: %v", err)})
		return
	}

	// Get token supply
	supply, err := heliusClient.GetTokenSupply(req.Mint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token supply: %v", err)})
		return
	}

	// Combine the information
	response := GetTokenInfoResponse{
		// Metadata fields
		Name:        metadata.Name,
		Symbol:      metadata.Symbol,
		Description: "", // Metadata doesn't include description
		Image:       metadata.Uri,

		// Supply fields
		TotalSupply:         supply.Amount,
		ToTalSupplyReadable: supply.UiAmount,
		Decimals:            supply.Decimals,
		UiSupply:            supply.UiAmount,
		UiSupplyString:      supply.UiAmountString,
	}

	c.JSON(http.StatusOK, response)
}

// GetDasInfoRequest represents the request body for getting DAS (Digital Asset Standard) info
type GetDasInfoRequest struct {
	Mint string `json:"mint" binding:"required"`
}

// GetDasInfoHandler handles requests to get DAS info from Helius API
func GetDasInfoHandler(c *gin.Context) {
	var req GetDasInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
			"id": req.Mint,
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

	// Parse response as JSON and return it
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		// If parsing fails, return raw response
		c.Data(http.StatusOK, "application/json", bodyBytes)
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetEnhancedTransactionsWithoutAddressRequest represents the request body for getting enhanced transactions
type GetEnhancedTransactionsWithoutAddressRequest struct {
	Signatures []string `json:"signatures" binding:"required,min=1"`
}

// GetEnhancedTransactionsWithoutAddressHandler handles requests to get enhanced transactions by signatures
func GetEnhancedTransactionsWithoutAddressHandler(c *gin.Context) {
	var req GetEnhancedTransactionsWithoutAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Helius API key from environment
	heliusApiKey := os.Getenv("HELIUS_API_KEY")
	if heliusApiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helius API key not configured"})
		return
	}

	// Create Helius client
	client := helius.NewClient(heliusApiKey)

	// Get enhanced transactions
	transactions, err := client.GetEnhancedTransactions(req.Signatures)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get transactions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, transactions)
}

// BondingCurveEstimateRequest represents the request for bonding curve estimation
type BondingCurveEstimateRequest struct {
	Percentage float64 `json:"percentage" binding:"required"`
	VSOL       float64 `json:"v_sol" binding:"required"`
	VToken     float64 `json:"v_token" binding:"required"`
	FeeRate    float64 `json:"fee_rate" binding:"required"`
}

// BondingCurveSimulateRequest represents the request for bonding curve simulation
type BondingCurveSimulateRequest struct {
	Amount  float64 `json:"amount" binding:"required"`
	Type    string  `json:"type" binding:"required"`
	VSOL    float64 `json:"v_sol" binding:"required"`
	VToken  float64 `json:"v_token" binding:"required"`
	FeeRate float64 `json:"fee_rate" binding:"required"`
}

// GetVirtualReservesRequest represents the request for getting virtual reserves
type GetVirtualReservesRequest struct {
	TokenAmount float64 `json:"token_amount" binding:"required"`
}

// GetVirtualReservesResponse represents the response for virtual reserves
type GetVirtualReservesResponse struct {
	VSOL   float64 `json:"v_sol"`
	VToken float64 `json:"v_token"`
}

// EstimateBuyCostWithIncreaseHandler handles the estimation of buy cost
func EstimateBuyCostWithIncreaseHandler(c *gin.Context) {
	var req BondingCurveEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := utils.EstimateBuyCostWithIncrease(req.Percentage, req.VSOL, req.VToken, req.FeeRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// EstimateSellReturnWithDecreaseHandler handles the estimation of sell return
func EstimateSellReturnWithDecreaseHandler(c *gin.Context) {
	var req BondingCurveEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := utils.EstimateSellReturnWithDecrease(req.Percentage, req.VSOL, req.VToken, req.FeeRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SimulateBondingCurveAmountOutHandler handles the simulation of amount out
func SimulateBondingCurveAmountOutHandler(c *gin.Context) {
	var req BondingCurveSimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := utils.SimulateBondingCurveAmountOut(req.Amount, req.Type, req.VSOL, req.VToken, req.FeeRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// SimulateBondingCurveAmountInHandler handles the simulation of amount in
func SimulateBondingCurveAmountInHandler(c *gin.Context) {
	var req BondingCurveSimulateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := utils.SimulateBondingCurveAmountIn(req.Amount, req.Type, req.VSOL, req.VToken, req.FeeRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetVirtualReservesHandler handles getting virtual reserves
func GetVirtualReservesHandler(c *gin.Context) {
	var req GetVirtualReservesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vSol, vToken := utils.GetVirtualReserves(req.TokenAmount)
	c.JSON(http.StatusOK, GetVirtualReservesResponse{
		VSOL:   vSol,
		VToken: vToken,
	})
}

// GetLaunchpadAndCpmmIdRequest represents the request body for getting launchpad and CPMM pool IDs
type GetLaunchpadAndCpmmIdRequest struct {
	MintA string `json:"mintA" binding:"required"`
	MintB string `json:"mintB" binding:"required"`
}

// GetLaunchpadAndCpmmIdResponse represents the response for launchpad and CPMM pool IDs
type GetLaunchpadAndCpmmIdResponse struct {
	CpmmPoolId         string `json:"cpmmPoolId"`
	LaunchpadPoolId    string `json:"launchpadPoolId"`
	CpmmBaseVault      string `json:"cpmmBaseVault"`
	CpmmQuoteVault     string `json:"cpmmQuoteVault"`
	CpmmLpMint         string `json:"cpmmLpMint"`
	CpmmPdaAmmConfigId string `json:"cpmmPdaAmmConfigId"`
}

// GetLaunchpadAndCpmmId handles requests to get launchpad and CPMM pool IDs
func GetLaunchpadAndCpmmId(c *gin.Context) {
	var req GetLaunchpadAndCpmmIdRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse mint addresses
	mintA, err := solana.PublicKeyFromBase58(req.MintA)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid mintA address: %v", err)})
		return
	}

	mintB, err := solana.PublicKeyFromBase58(req.MintB)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid mintB address: %v", err)})
		return
	}

	// Get pool IDs
	poolIds, err := mcsolana.GetLaunchpadAndCpmmId(mintA, mintB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get pool IDs: %v", err)})
		return
	}

	// Get CPMM pool vault addresses
	vaultResult, err := mcsolana.GetCpmmPoolVault(poolIds.CpmmPoolId, mintB, mintA)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get CPMM pool vaults: %v", err)})
		return
	}

	// Get CPMM LP mint address
	lpMintResult, err := mcsolana.GetPdaLpMint(mcsolana.CREATE_CPMM_POOL_PROGRAM, poolIds.CpmmPoolId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get CPMM LP mint: %v", err)})
		return
	}

	// Get CPMM PDA AMM Config ID
	ammConfigResult, err := mcsolana.GetCpmmPdaAmmConfigId(mcsolana.CREATE_CPMM_POOL_PROGRAM, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get CPMM AMM config ID: %v", err)})
		return
	}

	// Return response
	c.JSON(http.StatusOK, GetLaunchpadAndCpmmIdResponse{
		CpmmPoolId:         poolIds.CpmmPoolId.String(),
		LaunchpadPoolId:    poolIds.LaunchpadPoolId.String(),
		CpmmBaseVault:      vaultResult.BaseVault.String(),
		CpmmQuoteVault:     vaultResult.QuoteVault.String(),
		CpmmLpMint:         lpMintResult.PublicKey.String(),
		CpmmPdaAmmConfigId: ammConfigResult.PublicKey.String(),
	})
}

// GetLaunchpadPoolInfoRequest represents the request body for getting launchpad pool data
type GetLaunchpadPoolInfoRequest struct {
	RpcEndpoint string `json:"rpcEndpoint" binding:"required"`
	PoolId      string `json:"poolId" binding:"required"`
}

// GetLaunchpadPoolInfo handles requests to get launchpad pool data
func GetLaunchpadPoolInfo(c *gin.Context) {
	var req GetLaunchpadPoolInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse pool ID
	poolId, err := solana.PublicKeyFromBase58(req.PoolId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid poolId address: %v", err)})
		return
	}

	// Get launchpad pool data
	poolData, err := mcsolana.GetLaunchpadPoolInfo(req.RpcEndpoint, poolId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get launchpad pool data: %v", err)})
		return
	}

	// Return response
	c.JSON(http.StatusOK, poolData)
}

// GetLaunchpadPoolConfigRequest represents the request body for getting launchpad pool config
type GetLaunchpadPoolConfigRequest struct {
	RpcEndpoint string `json:"rpcEndpoint" binding:"required"`
	ConfigId    string `json:"configId" binding:"required"`
}

// GetLaunchpadPoolConfig handles requests to get launchpad pool config
func GetLaunchpadPoolConfig(c *gin.Context) {
	var req GetLaunchpadPoolConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse pool ID (config ID)
	configId, err := solana.PublicKeyFromBase58(req.ConfigId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid poolId address: %v", err)})
		return
	}

	// Get launchpad pool config
	configData, err := mcsolana.GetLaunchpadPoolConfig(req.RpcEndpoint, configId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get launchpad pool config: %v", err)})
		return
	}

	// Return response
	c.JSON(http.StatusOK, configData)
}

// GetCpmmPoolInfoRequest represents the request body for getting CPMM pool info
type GetCpmmPoolInfoRequest struct {
	PoolId string `json:"poolId" binding:"required"`
}

// GetCpmmPoolInfo handles requests to get CPMM pool info
func GetCpmmPoolInfo(c *gin.Context) {
	var req GetCpmmPoolInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that it's a valid Solana public key format
	_, err := solana.PublicKeyFromBase58(req.PoolId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid poolId format: %v", err)})
		return
	}

	// Get CPMM pool info using batch function with single pool ID
	poolInfos, err := mcsolana.GetCpmmPoolInfoBatch([]string{req.PoolId})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get CPMM pool info: %v", err)})
		return
	}

	// Check if pool was found
	if len(poolInfos) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pool not found"})
		return
	}

	// Return the first (and should be only) pool info
	c.JSON(http.StatusOK, poolInfos[0])
}

// GetPumpFunPDARequest represents the request body for getting PumpFun PDAs
type GetPumpFunPDARequest struct {
	UserPubkey string `json:"userPubkey" binding:"required"`
	MintPubkey string `json:"mintPubkey" binding:"required"`
}

// GetPumpFunPDAHandler handles requests to get PumpFun PDAs
func GetPumpFunPDAHandler(c *gin.Context) {
	var req GetPumpFunPDARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse public keys
	userPubkey, err := solana.PublicKeyFromBase58(req.UserPubkey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid userPubkey: %v", err)})
		return
	}

	mintPubkey, err := solana.PublicKeyFromBase58(req.MintPubkey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid mintPubkey: %v", err)})
		return
	}

	// Get all PumpFun PDAs
	pdaInfo, err := mcsolana.GetAllPumpFunPDAs(userPubkey, mintPubkey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get PumpFun PDAs: %v", err)})
		return
	}

	c.JSON(http.StatusOK, pdaInfo)
}

// GetPumpSwapPDARequest represents the request body for getting PumpSwap PDAs
type GetPumpSwapPDARequest struct {
	UserPubkey        string `json:"userPubkey" binding:"required"`
	CreatorPubkey     string `json:"creatorPubkey" binding:"required"`
	BaseMintPubkey    string `json:"baseMintPubkey" binding:"required"`
	CoinCreatorPubkey string `json:"coinCreatorPubkey" binding:"required"`
}

// GetPumpSwapPDAHandler handles requests to get PumpSwap PDAs
func GetPumpSwapPDAHandler(c *gin.Context) {
	var req GetPumpSwapPDARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse public keys
	userPubkey, err := solana.PublicKeyFromBase58(req.UserPubkey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid userPubkey: %v", err)})
		return
	}

	creatorPubkey, err := solana.PublicKeyFromBase58(req.CreatorPubkey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid creatorPubkey: %v", err)})
		return
	}

	baseMintPubkey, err := solana.PublicKeyFromBase58(req.BaseMintPubkey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid baseMintPubkey: %v", err)})
		return
	}

	coinCreatorPubkey, err := solana.PublicKeyFromBase58(req.CoinCreatorPubkey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid coinCreatorPubkey: %v", err)})
		return
	}

	// Get all PumpSwap PDAs
	pdaInfo, err := mcsolana.GetAllPumpSwapPDAs(userPubkey, creatorPubkey, baseMintPubkey, coinCreatorPubkey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get PumpSwap PDAs: %v", err)})
		return
	}

	c.JSON(http.StatusOK, pdaInfo)
}

// GetJupiterSwapResultRequest represents the request body for getting Jupiter swap result
type GetJupiterSwapResultRequest struct {
	InputMint                  string `json:"inputMint" binding:"required"`
	OutputMint                 string `json:"outputMint" binding:"required"`
	Amount                     string `json:"amount" binding:"required"`
	SlippageBps                int    `json:"slippageBps" binding:"required"`
	RestrictIntermediateTokens *bool  `json:"restrictIntermediateTokens"`
}

// GetJupiterSwapResult handles requests to get Jupiter swap result
func GetJupiterSwapResult(c *gin.Context) {
	var req GetJupiterSwapResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default value for restrictIntermediateTokens if not provided
	restrictIntermediateTokens := true
	if req.RestrictIntermediateTokens != nil {
		restrictIntermediateTokens = *req.RestrictIntermediateTokens
	}

	// Call Jupiter client
	swapResult, err := utils.GetSwapResult(
		req.InputMint,
		req.OutputMint,
		req.Amount,
		req.SlippageBps,
		restrictIntermediateTokens,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get Jupiter swap result: %v", err)})
		return
	}

	c.JSON(http.StatusOK, swapResult)
}

// GetTokenPriceRequest represents the request body for getting token price
type GetTokenPriceRequest struct {
	Mint string `json:"mint" binding:"required"`
}

// GetTokenPrice handles requests to get token price from Jupiter
func GetTokenPrice(c *gin.Context) {
	var req GetTokenPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call Jupiter client to get token price
	price, useCached, err := utils.GetTokenPrice(req.Mint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token price: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price":      price,
		"use_cached": useCached,
	})
}

// GetAccountInfoRequest represents the request body for getting account information
type GetAccountInfoRequest struct {
	Address string `json:"address" binding:"required"`
}

// TokenBalanceItem represents a simple token balance item
type TokenBalanceItem struct {
	Mint            string  `json:"mint"`
	TokenAccount    string  `json:"token_account"`
	Balance         uint64  `json:"balance"`
	BalanceReadable float64 `json:"balance_readable"`
}

// GetAccountInfoResponse represents the response for account information
type GetAccountInfoResponse struct {
	Address            string             `json:"address"`
	SolBalance         uint64             `json:"sol_balance"`
	SolBalanceReadable float64            `json:"sol_balance_readable"`
	TokenBalances      []TokenBalanceItem `json:"token_balances"`
	LastUpdated        string             `json:"last_updated"`
}

// GetAccountInfo handles requests to get account information including SOL and token balances
func GetAccountInfo(c *gin.Context) {
	var req GetAccountInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse the address
	ownerPubkey, err := solana.PublicKeyFromBase58(req.Address)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address format"})
		return
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create RPC client
	client := rpc.New(solanaRPC)

	// Get SOL balance
	solBalance, solUpdateTime, err := mcsolana.GetSolBalance(client, ownerPubkey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get SOL balance: %v", err)})
		return
	}

	// Get all token configs from database
	var tokenConfigs []models.TokenConfig
	if err := dbconfig.DB.Find(&tokenConfigs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get token configs: %v", err)})
		return
	}

	// Get token balances for each token config
	var tokenBalances []TokenBalanceItem
	for _, tokenConfig := range tokenConfigs {
		balance, _, err := mcsolana.GetTokenBalance(dbconfig.DB, client, ownerPubkey, tokenConfig.Mint)
		if err != nil {
			// Log error but continue with other tokens
			fmt.Printf("Failed to get balance for token %s (%s): %v\n", tokenConfig.Symbol, tokenConfig.Mint, err)
			continue
		}

		// Only include tokens with non-zero balance
		if balance > 0 {
			// Calculate readable balance using token's decimals
			divisor := uint64(1) << tokenConfig.Decimals
			balanceReadable := float64(balance) / float64(divisor)

			tokenBalance := TokenBalanceItem{
				Mint:            tokenConfig.Mint,
				TokenAccount:    tokenConfig.Mint, // Use mint as token_account for now
				Balance:         balance,
				BalanceReadable: balanceReadable,
			}
			tokenBalances = append(tokenBalances, tokenBalance)
		}
	}

	// Calculate SOL balance in readable format (SOL has 9 decimals)
	solBalanceReadable := float64(solBalance) / 1e9

	// Create response
	response := GetAccountInfoResponse{
		Address:            req.Address,
		SolBalance:         solBalance,
		SolBalanceReadable: solBalanceReadable,
		TokenBalances:      tokenBalances,
		LastUpdated:        solUpdateTime.Format("2006-01-02 15:04:05"),
	}

	c.JSON(http.StatusOK, response)
}

// FetchAddressBalanceChangeRequest represents the request body for address balance change from signature
type FetchAddressBalanceChangeRequest struct {
	Signature   string   `json:"signature" binding:"required"`
	AddressList []string `json:"address_list" binding:"required"`
	Mint        string   `json:"mint" binding:"required"` // "sol" for native SOL, or token mint address
	Decimals    uint     `json:"decimals"`               // precision for readable amount (e.g. 9 for SOL, 6 for token)
}

// FetchAddressBalanceChangeFromSignature fetches the transaction by signature and returns balance changes for each address
func FetchAddressBalanceChangeFromSignature(c *gin.Context) {
	var req FetchAddressBalanceChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}
	client := rpc.New(solanaRPC)
	txResult, err := mcsolana.GetTransactionBySignature(client, req.Signature)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to get transaction: %v", err)})
		return
	}
	changes, err := mcsolana.ParseAddressBalanceChangesFromTransaction(txResult, req.AddressList, req.Mint, req.Decimals)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to parse balance changes: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": changes})
}

// PoolMonitorRequest represents the request body for controlling pool monitoring
type PoolMonitorRequest struct {
	Action               string `json:"action" binding:"required"`       // "start" or "stop"
	PoolAddress          string `json:"pool_address" binding:"required"` // Pool address to monitor
	BaseToken            string `json:"base_token"`                      // Base token mint address (required for start)
	QuoteToken           string `json:"quote_token"`                     // Quote token mint address (required for start)
	MeteoraDbcAuthority  string `json:"meteora_dbc_authority"`           // Meteora DBC authority address
	MeteoraCpmmAuthority string `json:"meteora_cpmm_authority"`          // Meteora CPMM authority address
}

// ControlPoolMonitor handles pool monitoring control requests
func ControlPoolMonitor(c *gin.Context) {
	var request PoolMonitorRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate action
	if request.Action != "start" && request.Action != "stop" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "action must be 'start' or 'stop'"})
		return
	}

	// Validate required fields for start action
	if request.Action == "start" {
		if request.BaseToken == "" || request.QuoteToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "base_token and quote_token are required for start action"})
			return
		}
		if request.MeteoraDbcAuthority == "" && request.MeteoraCpmmAuthority == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "at least one authority (meteora_dbc_authority or meteora_cpmm_authority) is required for start action"})
			return
		}
	}

	// Check if RabbitMQ is initialized
	if config.RabbitMQ == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "RabbitMQ not initialized"})
		return
	}

	// Handle in goroutine to avoid blocking
	go func() {
		publisher, err := config.NewPublisher()
		if err != nil {
			log.Errorf("Failed to create RabbitMQ publisher: %v", err)
			return
		}
		defer publisher.Close()

		// Prepare monitoring message
		var monitorMsg meteora.PoolMonitorMessage
		if request.Action == "start" {
			monitorMsg = meteora.PoolMonitorMessage{
				Action:               "start_monitoring",
				MeteoradbcAddress:    request.PoolAddress,
				BaseTokenMint:        request.BaseToken,
				QuoteTokenMint:       request.QuoteToken,
				MeteoraDbcAuthority:  request.MeteoraDbcAuthority,
				MeteoraCpmmAuthority: request.MeteoraCpmmAuthority,
			}
		} else {
			// For stop action, we need to send stop_monitoring message
			monitorMsg = meteora.PoolMonitorMessage{
				Action:            "stop_monitoring",
				MeteoradbcAddress: request.PoolAddress,
			}
		}

		// Publish message
		if err := publisher.Publish("meteora_pool_monitor", monitorMsg); err != nil {
			log.Errorf("Failed to publish monitoring message: %v", err)
		} else {
			log.Infof("Published %s monitoring task for pool: %s",
				request.Action, request.PoolAddress)
		}
	}()

	// Return success response immediately
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Pool monitoring %s request submitted successfully", request.Action),
		"action":  request.Action,
		"pool":    request.PoolAddress,
	})
}

// GetMultiAccountsInfoRequest represents the request body for getting multiple accounts information
type GetMultiAccountsInfoRequest struct {
	Accounts []string `json:"accounts" binding:"required,min=1"`
	Mint     string   `json:"mint" binding:"required"`
}

// GetMultiAccountsInfoResponse represents the response for multiple accounts information
type GetMultiAccountsInfoResponse struct {
	Mint     string                         `json:"mint"`
	Balances []mcsolana.MultiAccountBalance `json:"balances"`
}

// GetMultiAccountsInfo handles requests to get multiple accounts token balance information
func GetMultiAccountsInfo(c *gin.Context) {
	var req GetMultiAccountsInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Solana RPC endpoint from environment
	solanaRPC := os.Getenv("DEFAULT_SOLANA_RPC")
	if solanaRPC == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Solana RPC endpoint not configured"})
		return
	}

	// Create RPC client
	client := rpc.New(solanaRPC)

	// Get token decimals from database or default to 9 (SOL) or 6 (most tokens)
	var decimals uint8 = 6 // Default to 6 decimals for most tokens
	var tokenConfig models.TokenConfig
	if err := dbconfig.DB.Where("mint = ?", req.Mint).First(&tokenConfig).Error; err == nil {
		decimals = uint8(tokenConfig.Decimals)
	} else {
		// If token not found in database, try to get decimals from chain
		// For now, we'll use default 6, but could extend to query mint account
		log.Warnf("Token %s not found in database, using default decimals: %d", req.Mint, decimals)
	}

	// Get multiple accounts info
	balances, err := mcsolana.GetMultiAccountsInfo(client, req.Accounts, req.Mint, decimals)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get multi accounts info: %v", err)})
		return
	}

	// Create response
	response := GetMultiAccountsInfoResponse{
		Mint:     req.Mint,
		Balances: balances,
	}

	c.JSON(http.StatusOK, response)
}

// RPCStatusRequest represents the request body for checking RPC status
type RPCStatusRequest struct {
	RPCList []string `json:"rpc-list" binding:"required"`
}

// GetRPCStatusHandler handles requests to check RPC endpoint status
func GetRPCStatusHandler(c *gin.Context) {
	var request RPCStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Limit the number of RPC endpoints to check to prevent abuse
	if len(request.RPCList) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 20 RPC endpoints allowed per request"})
		return
	}

	// Create context with timeout (max 2 seconds for all checks)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	// Check RPC list with reduced timeout (1.5 seconds per endpoint)
	// This ensures faster response times
	results := mcsolana.CheckRPCListAsync(ctx, request.RPCList, 1*time.Second)

	// Convert results to response format
	response := make([]map[string]interface{}, len(results))
	for i, res := range results {
		resultMap := map[string]interface{}{
			"url":     res.URL,
			"ok":      res.OK,
			"latency": res.Latency.String(),
		}
		if res.Error != "" {
			resultMap["error"] = res.Error
		}
		response[i] = resultMap
	}

	c.JSON(http.StatusOK, gin.H{
		"results": response,
		"count":   len(results),
	})
}

// VpnGetProxiesRequest represents the request body for getting Clash proxies
type VpnGetProxiesRequest struct {
	BaseURL     string `json:"base_url" binding:"required"`
	BearerToken string `json:"bearer_token" binding:"required"`
}

// VpnControllerGetProxies handles requests to get Clash proxies list
func VpnControllerGetProxies(c *gin.Context) {
	var req VpnGetProxiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build the Clash API URL
	clashURL := fmt.Sprintf("http://%s/proxies", req.BaseURL)

	// Create HTTP request
	httpReq, err := http.NewRequest("GET", clashURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create request: %v", err)})
		return
	}

	// Set Authorization header
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", req.BearerToken))

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send request
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send request: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read response: %v", err)})
		return
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Clash API returned status code: %d", resp.StatusCode),
			"body":  string(bodyBytes),
		})
		return
	}

	// Parse response as JSON
	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		// If parsing fails, return raw response
		c.Data(http.StatusOK, "application/json", bodyBytes)
		return
	}

	c.JSON(http.StatusOK, result)
}

// VpnChangeProxyRequest represents the request body for changing Clash proxy
type VpnChangeProxyRequest struct {
	BaseURL     string `json:"base_url" binding:"required"`
	BearerToken string `json:"bearer_token" binding:"required"`
	ProxyName   string `json:"proxy_name" binding:"required"`
}

// VpnControllerChangeProxy handles requests to change Clash proxy
func VpnControllerChangeProxy(c *gin.Context) {
	var req VpnChangeProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build the Clash API URL (always use GLOBAL as the selector name)
	clashURL := fmt.Sprintf("http://%s/proxies/GLOBAL", req.BaseURL)

	// Prepare request body
	requestBody := map[string]interface{}{
		"name": req.ProxyName,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to marshal request: %v", err)})
		return
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("PUT", clashURL, bytes.NewBuffer(jsonData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create request: %v", err)})
		return
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", req.BearerToken))

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send request
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send request: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read response: %v", err)})
		return
	}

	// Check response status
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Clash API returned status code: %d", resp.StatusCode),
			"body":  string(bodyBytes),
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message":    "Proxy changed successfully",
		"proxy_name": req.ProxyName,
	})
}
