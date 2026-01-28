package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// JupiterQuoteResponse represents the response structure from Jupiter API
type JupiterQuoteResponse struct {
	InputMint                     string      `json:"inputMint"`
	InAmount                      string      `json:"inAmount"`
	OutputMint                    string      `json:"outputMint"`
	OutAmount                     string      `json:"outAmount"`
	OtherAmountThreshold          string      `json:"otherAmountThreshold"`
	SwapMode                      string      `json:"swapMode"`
	SlippageBps                   int         `json:"slippageBps"`
	PlatformFee                   any         `json:"platformFee"`
	PriceImpactPct                string      `json:"priceImpactPct"`
	RoutePlan                     []RoutePlan `json:"routePlan"`
	ContextSlot                   int         `json:"contextSlot"`
	TimeTaken                     float64     `json:"timeTaken"`
	SwapUsdValue                  string      `json:"swapUsdValue"`
	SimplerRouteUsed              bool        `json:"simplerRouteUsed"`
	MostReliableAmmsQuoteReport   any         `json:"mostReliableAmmsQuoteReport"`
	UseIncurredSlippageForQuoting any         `json:"useIncurredSlippageForQuoting"`
	OtherRoutePlans               any         `json:"otherRoutePlans"`
	LoadedLongtailToken           bool        `json:"loadedLongtailToken"`
	InstructionVersion            any         `json:"instructionVersion"`
}

// RoutePlan represents a route plan in the Jupiter response
type RoutePlan struct {
	SwapInfo SwapInfo `json:"swapInfo"`
	Percent  int      `json:"percent"`
	Bps      int      `json:"bps"`
}

// SwapInfo represents swap information in a route plan
type SwapInfo struct {
	AmmKey     string `json:"ammKey"`
	Label      string `json:"label"`
	InputMint  string `json:"inputMint"`
	OutputMint string `json:"outputMint"`
	InAmount   string `json:"inAmount"`
	OutAmount  string `json:"outAmount"`
	FeeAmount  string `json:"feeAmount"`
	FeeMint    string `json:"feeMint"`
}

// GetSwapResult retrieves swap quote from Jupiter API
func GetSwapResult(inputMint, outputMint, amount string, slippageBps int, restrictIntermediateTokens ...bool) (*JupiterQuoteResponse, error) {
	// Check if amount is less than or equal to 0
	amountFloat, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse amount: %w", err)
	}

	if amountFloat <= 100 {
		// Return zero response for zero or negative amounts
		return &JupiterQuoteResponse{
			InputMint:                     inputMint,
			InAmount:                      amount,
			OutputMint:                    outputMint,
			OutAmount:                     "0",
			OtherAmountThreshold:          "0",
			SwapMode:                      "ExactIn",
			SlippageBps:                   slippageBps,
			PlatformFee:                   nil,
			PriceImpactPct:                "0",
			RoutePlan:                     []RoutePlan{},
			ContextSlot:                   0,
			TimeTaken:                     0,
			SwapUsdValue:                  "0",
			SimplerRouteUsed:              false,
			MostReliableAmmsQuoteReport:   nil,
			UseIncurredSlippageForQuoting: nil,
			OtherRoutePlans:               nil,
			LoadedLongtailToken:           false,
			InstructionVersion:            nil,
		}, nil
	}

	// Set default value for restrictIntermediateTokens
	restrict := true
	if len(restrictIntermediateTokens) > 0 {
		restrict = restrictIntermediateTokens[0]
	}

	// Build the API URL
	baseURL := "https://lite-api.jup.ag/swap/v1/quote"
	params := url.Values{}
	params.Add("inputMint", inputMint)
	params.Add("outputMint", outputMint)
	params.Add("amount", amount)
	params.Add("slippageBps", strconv.Itoa(slippageBps))
	params.Add("restrictIntermediateTokens", strconv.FormatBool(restrict))

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Parse JSON response
	var quoteResponse JupiterQuoteResponse
	if err := json.NewDecoder(resp.Body).Decode(&quoteResponse); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return &quoteResponse, nil
}

// token price cache (in-memory)
type tokenPriceCacheEntry struct {
	price     float64
	updatedAt time.Time
}

var (
	tokenPriceCache   = make(map[string]tokenPriceCacheEntry)
	tokenPriceCacheMu sync.RWMutex
)

// GetTokenPrice retrieves the price of a token in SOL
// Returns: price, useCached, error
func GetTokenPrice(mint string) (float64, bool, error) {
	// Check if mint is SOL
	solMint := "So11111111111111111111111111111111111111112"
	if mint == solMint {
		return 1.0, false, nil
	}

	// Call GetSwapResult to get swap quote
	// Using 1000000000000 (1e12) tokens as input amount
	quote, err := GetSwapResult(mint, solMint, "1000000000000", 50)
	if err != nil {
		// fallback to cached price if available
		tokenPriceCacheMu.RLock()
		entry, ok := tokenPriceCache[mint]
		tokenPriceCacheMu.RUnlock()
		if ok {
			return entry.price, true, nil
		}
		return 0, false, fmt.Errorf("failed to get swap result and no cached price: %w", err)
	}

	// Parse outAmount to float64
	outAmount, err := strconv.ParseFloat(quote.OutAmount, 64)
	if err != nil {
		return 0, false, fmt.Errorf("failed to parse outAmount: %w", err)
	}

	// Calculate token price by dividing outAmount by 1e15
	// outAmount is in lamports (1e9), so dividing by 1e15 gives price per token
	const divisor = 1e15
	tokenPrice := outAmount / divisor

	// update cache
	tokenPriceCacheMu.Lock()
	tokenPriceCache[mint] = tokenPriceCacheEntry{
		price:     tokenPrice,
		updatedAt: time.Now(),
	}
	tokenPriceCacheMu.Unlock()

	return tokenPrice, false, nil
}
