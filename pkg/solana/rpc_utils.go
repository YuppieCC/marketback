package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Global HTTP client with connection pooling for better performance
var (
	rpcCheckClient *http.Client
	clientOnce     sync.Once
)

// getRPCClient returns a shared HTTP client with optimized settings
func getRPCClient() *http.Client {
	clientOnce.Do(func() {
		transport := &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		}
		rpcCheckClient = &http.Client{
			Transport: transport,
			Timeout:   2 * time.Second, // Default timeout
		}
	})
	return rpcCheckClient
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	Jsonrpc string           `json:"jsonrpc"`
	Result  interface{}      `json:"result"`
	Error   *json.RawMessage `json:"error"`
	ID      int              `json:"id"`
}

// RPCCheckResult represents the result of checking an RPC endpoint
type RPCCheckResult struct {
	URL     string        `json:"url"`
	OK      bool          `json:"ok"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

// checkRPCAsync checks a single RPC endpoint asynchronously with context support
func checkRPCAsync(ctx context.Context, url string, timeout time.Duration, ch chan<- RPCCheckResult, wg *sync.WaitGroup) {
	defer wg.Done()

	start := time.Now()

	// Create request with context
	req := RPCRequest{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "getHealth",
		Params:  []interface{}{},
	}
	body, _ := json.Marshal(req)

	// Use shared HTTP client with context
	client := getRPCClient()

	// Create request with context
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		ch <- RPCCheckResult{URL: url, OK: false, Latency: 0, Error: err.Error()}
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Create a client with custom timeout for this request
	requestClient := &http.Client{
		Transport: client.Transport,
		Timeout:   timeout,
	}

	resp, err := requestClient.Do(httpReq)
	if err != nil {
		latency := time.Since(start)
		ch <- RPCCheckResult{URL: url, OK: false, Latency: latency, Error: err.Error()}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		latency := time.Since(start)
		ch <- RPCCheckResult{URL: url, OK: false, Latency: latency, Error: fmt.Sprintf("status code: %d", resp.StatusCode)}
		return
	}

	var result RPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		latency := time.Since(start)
		ch <- RPCCheckResult{URL: url, OK: false, Latency: latency, Error: err.Error()}
		return
	}
	if result.Error != nil {
		latency := time.Since(start)
		ch <- RPCCheckResult{URL: url, OK: false, Latency: latency, Error: fmt.Sprintf("rpc error: %s", string(*result.Error))}
		return
	}

	latency := time.Since(start)
	ch <- RPCCheckResult{URL: url, OK: true, Latency: latency, Error: ""}
}

// CheckRPCListAsync checks multiple RPC endpoints asynchronously with context support
func CheckRPCListAsync(ctx context.Context, rpcList []string, timeout time.Duration) []RPCCheckResult {
	var wg sync.WaitGroup
	resultCh := make(chan RPCCheckResult, len(rpcList))

	for _, url := range rpcList {
		wg.Add(1)
		go checkRPCAsync(ctx, url, timeout, resultCh, &wg)
	}

	// Wait for all goroutines to complete or context to be cancelled
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, return partial results
		close(resultCh)
		var results []RPCCheckResult
		for res := range resultCh {
			results = append(results, res)
		}
		return results
	case <-done:
		close(resultCh)
		var results []RPCCheckResult
		for res := range resultCh {
			results = append(results, res)
		}
		return results
	}
}
