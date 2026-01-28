package helius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"bytes"
	"time"
)

// Client represents a Helius API client
type Client struct {
	apiKey     string
	baseURL    string
	rpcURL     string
	httpClient *http.Client
}

// NewClient creates a new Helius API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		baseURL:    "https://api.helius.xyz/v0",
		rpcURL:     "https://mainnet.helius-rpc.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				IdleConnTimeout:       10 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}
}

// TransactionOptions represents the query parameters for GetEnhancedTransactionsByAddress
type TransactionOptions struct {
	Limit  *int    `json:"limit,omitempty"`
	Before *string `json:"before,omitempty"`
	Until  *string `json:"until,omitempty"`
	Source *string `json:"source,omitempty"`
	Type   *string `json:"type,omitempty"`
}

// TokenTransfer represents a token transfer in the transaction
type TokenTransfer struct {
	FromTokenAccount string  `json:"fromTokenAccount"`
	ToTokenAccount   string  `json:"toTokenAccount"`
	FromUserAccount  string  `json:"fromUserAccount"`
	ToUserAccount    string  `json:"toUserAccount"`
	TokenAmount     float64 `json:"tokenAmount"`
	Mint            string  `json:"mint"`
	TokenStandard   string  `json:"tokenStandard"`
}

// NativeTransfer represents a native SOL transfer in the transaction
type NativeTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount         int64  `json:"amount"`
}

// RawTokenAmount represents the token amount with decimals
type RawTokenAmount struct {
	TokenAmount string `json:"tokenAmount"`
	Decimals    int    `json:"decimals"`
}

// TokenBalanceChange represents a token balance change
type TokenBalanceChange struct {
	UserAccount     string         `json:"userAccount"`
	TokenAccount    string         `json:"tokenAccount"`
	RawTokenAmount  RawTokenAmount `json:"rawTokenAmount"`
	Mint            string         `json:"mint"`
}

// AccountData represents the account data in the transaction
type AccountData struct {
	Account              string               `json:"account"`
	NativeBalanceChange  int64                `json:"nativeBalanceChange"`
	TokenBalanceChanges  []TokenBalanceChange `json:"tokenBalanceChanges"`
}

// InnerInstruction represents an inner instruction in the transaction
type InnerInstruction struct {
	Accounts   []string `json:"accounts"`
	Data       string   `json:"data"`
	ProgramId  string   `json:"programId"`
}

// Instruction represents an instruction in the transaction
type Instruction struct {
	Accounts           []string            `json:"accounts"`
	Data              string              `json:"data"`
	ProgramId         string              `json:"programId"`
	InnerInstructions []InnerInstruction  `json:"innerInstructions"`
}

// EnhancedTransaction represents the response structure for a transaction
type EnhancedTransaction struct {
	Description      string            `json:"description"`
	Type            string            `json:"type"`
	Source          string            `json:"source"`
	Fee             int64             `json:"fee"`
	FeePayer        string            `json:"feePayer"`
	Signature       string            `json:"signature"`
	Slot            uint64            `json:"slot"`
	Timestamp       int64             `json:"timestamp"`
	TokenTransfers  []TokenTransfer   `json:"tokenTransfers"`
	NativeTransfers []NativeTransfer  `json:"nativeTransfers"`
	AccountData     []AccountData     `json:"accountData"`
	TransactionError interface{}      `json:"transactionError"`
	Instructions    []Instruction     `json:"instructions"`
	Events          map[string]interface{} `json:"events"`
}

// GetEnhancedTransactionsByAddress retrieves enhanced transactions for a specific address
func (c *Client) GetEnhancedTransactionsByAddress(address string, opts *TransactionOptions) ([]EnhancedTransaction, error) {
	// Build the URL with query parameters
	baseURL := fmt.Sprintf("%s/addresses/%s/transactions", c.baseURL, address)
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	q := u.Query()
	q.Add("api-key", c.apiKey)

	if opts != nil {
		if opts.Limit != nil {
			q.Add("limit", fmt.Sprintf("%d", *opts.Limit))
		}
		if opts.Before != nil {
			q.Add("before", *opts.Before)
		}
		if opts.Until != nil {
			q.Add("until", *opts.Until)
		}
		if opts.Source != nil {
			q.Add("source", *opts.Source)
		}
		if opts.Type != nil {
			q.Add("type", *opts.Type)
		}
	}
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Parse response
	var transactions []EnhancedTransaction
	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return transactions, nil
}

// Helper function to create an int pointer
func IntPtr(i int) *int {
	return &i
}

// Helper function to create a string pointer
func StringPtr(s string) *string {
	return &s
}

// TokenSupplyValue represents the token supply information
type TokenSupplyValue struct {
	Amount         string  `json:"amount"`
	Decimals      int     `json:"decimals"`
	UiAmount      float64 `json:"uiAmount"`
	UiAmountString string  `json:"uiAmountString"`
}

// TokenSupplyResult represents the result of a getTokenSupply request
type TokenSupplyResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value TokenSupplyValue `json:"value"`
}

// TokenSupplyResponse represents the JSON-RPC response for getTokenSupply
type TokenSupplyResponse struct {
	JsonRPC string           `json:"jsonrpc"`
	ID      string           `json:"id"`
	Result  TokenSupplyResult `json:"result"`
}

// GetTokenSupply retrieves the current supply of a SPL Token
func (c *Client) GetTokenSupply(mint string) (*TokenSupplyValue, error) {
	// Prepare the request payload
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "1",
		"method":  "getTokenSupply",
		"params":  []string{mint},
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create request with RPC URL
	rpcURLWithKey := fmt.Sprintf("%s/?api-key=%s", c.rpcURL, c.apiKey)
	req, err := http.NewRequest("POST", rpcURLWithKey, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Parse response
	var tokenSupplyResp TokenSupplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenSupplyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenSupplyResp.Result.Value, nil
}

// GetEnhancedTransactions retrieves enhanced transactions by their signatures
func (c *Client) GetEnhancedTransactions(signatures []string) ([]EnhancedTransaction, error) {
	// Build the URL
	url := fmt.Sprintf("%s/transactions/?api-key=%s", c.baseURL, c.apiKey)

	// Prepare request payload
	payload := map[string]interface{}{
		"transactions": signatures,
	}

	// Convert payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Parse response
	var transactions []EnhancedTransaction
	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return transactions, nil
} 