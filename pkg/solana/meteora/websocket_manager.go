package meteora

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"marketcontrol/internal/models"
	dbconfig "marketcontrol/pkg/config"
)

const (
	// Connection states
	StateDisconnected = "disconnected"
	StateConnecting   = "connecting"
	StateConnected    = "connected"

	// Reconnect settings
	maxReconnectAttempts = 10
	reconnectDelay       = 5 * time.Second

	// Error threshold
	maxErrorCount = 6 // Maximum consecutive errors before stopping monitoring

	// Transaction retry settings
	maxTransactionRetries  = 3                      // Maximum retry attempts for getting transaction
	initialRetryDelay      = 500 * time.Millisecond // Initial delay before first retry
	maxRetryDelay          = 5 * time.Second        // Maximum delay between retries
	retryBackoffMultiplier = 2.0                    // Exponential backoff multiplier
)

// PoolMonitorMessage represents a message for starting pool monitoring
type PoolMonitorMessage struct {
	Action             string `json:"action"`
	MeteoradbcAddress  string `json:"meteoradbc_address,omitempty"`
	MeteoracpmmAddress string `json:"meteoracpmm_address,omitempty"`
	ProjectID          uint   `json:"project_id,omitempty"`
	// Token information
	BaseTokenMint        string `json:"base_token_mint,omitempty"`
	QuoteTokenMint       string `json:"quote_token_mint,omitempty"`
	MeteoraDbcAuthority  string `json:"meteora_dbc_authority,omitempty"`
	MeteoraCpmmAuthority string `json:"meteora_cpmm_authority,omitempty"`
}

// TokenInfo represents token information in a swap transaction
type TokenInfo struct {
	Symbol  string  `json:"symbol"`
	Amount  float64 `json:"amount"`
	Address string  `json:"address"`
}

// SwapTransaction represents a parsed swap transaction
type SwapTransaction struct {
	Signature  string    `json:"signature"`
	Slot       uint64    `json:"slot"`
	Timestamp  int64     `json:"timestamp"`
	Action     string    `json:"action"` // "buy" | "sell" | "add liquidity" | "remove liquidity" | "unknown"
	BaseToken  TokenInfo `json:"base_token"`
	QuoteToken TokenInfo `json:"quote_token"`
	Value      float64   `json:"value"`
	Payer      string    `json:"payer"`
	Signers    []string  `json:"signers"`
	Success    bool      `json:"success"`           // Whether the transaction succeeded
	Error      string    `json:"error,omitempty"`   // Error message if transaction failed
	TxMeta     string    `json:"tx_meta,omitempty"` // Transaction metadata as JSON string
}

// SwapCallback is a function type for handling swap transactions
type SwapCallback func(swap *SwapTransaction)

// PoolConnection represents a WebSocket connection for monitoring a pool address
type PoolConnection struct {
	Address              string
	BaseTokenMint        string
	QuoteTokenMint       string
	MeteoraDbcAuthority  string
	MeteoraCpmmAuthority string
	Conn                 *websocket.Conn
	RPCClient            *rpc.Client
	Status               string
	LastMessage          time.Time
	ReconnectCh          chan bool
	StopCh               chan bool
	SubscriptionID       interface{}
	SwapCallback         SwapCallback
	mu                   sync.RWMutex
	wsEndpoint           string
	rpcEndpoint          string
	roleAddressMap       map[string]bool // Cached RoleAddress map for filtering
	errorCount           int             // Error counter for tracking consecutive errors
}

// PoolMonitorManager manages WebSocket connections for pool monitoring
type PoolMonitorManager struct {
	connections sync.Map // map[string]*PoolConnection
	wsEndpoint  string
	rpcEndpoint string
	mu          sync.RWMutex
}

// NewPoolMonitorManager creates a new pool monitor manager
func NewPoolMonitorManager() (*PoolMonitorManager, error) {
	rpcEndpoint := os.Getenv("DEFAULT_SOLANA_RPC")
	if rpcEndpoint == "" {
		rpcEndpoint = "https://responsive-prettiest-star.solana-mainnet.quiknode.pro/a89ffa3856d2b33cbe2601c466ca8c884c9f3e3b/"
	}

	// Get WebSocket endpoint from environment (required)
	wsEndpoint := os.Getenv("DEFAULT_SOLANA_WSS")
	if wsEndpoint == "" {
		wsEndpoint = "wss://responsive-prettiest-star.solana-mainnet.quiknode.pro/a89ffa3856d2b33cbe2601c466ca8c884c9f3e3b/"

	}

	return &PoolMonitorManager{
		wsEndpoint:  wsEndpoint,
		rpcEndpoint: rpcEndpoint,
	}, nil
}

// StartMonitoring starts monitoring a pool address for swap transactions
func (m *PoolMonitorManager) StartMonitoring(address, baseTokenMint, quoteTokenMint, meteoraDbcAuthority, meteoraCpmmAuthority string, callback SwapCallback) error {
	// Check if connection already exists
	if _, exists := m.connections.Load(address); exists {
		log.WithFields(log.Fields{
			"pool_address": address,
		}).Info("Connection already exists, skipping")
		return nil
	}

	// At least one authority is required
	if meteoraDbcAuthority == "" && meteoraCpmmAuthority == "" {
		return fmt.Errorf("at least one authority (meteoraDbcAuthority or meteoraCpmmAuthority) is required for monitoring")
	}

	// Load RoleAddress map into memory for filtering
	roleAddressMap, err := m.loadRoleAddressMap()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Warn("Failed to load RoleAddress map, will retry on each transaction")
		// Continue anyway, will load on each transaction if map is nil
		roleAddressMap = nil
	}

	conn := &PoolConnection{
		Address:              address,
		BaseTokenMint:        baseTokenMint,
		QuoteTokenMint:       quoteTokenMint,
		MeteoraDbcAuthority:  meteoraDbcAuthority,
		MeteoraCpmmAuthority: meteoraCpmmAuthority,
		Status:               StateDisconnected,
		ReconnectCh:          make(chan bool, 1),
		StopCh:               make(chan bool, 1),
		SwapCallback:         callback,
		wsEndpoint:           m.wsEndpoint,
		rpcEndpoint:          m.rpcEndpoint,
		RPCClient:            rpc.New(m.rpcEndpoint),
		roleAddressMap:       roleAddressMap,
		errorCount:           0,
	}

	m.connections.Store(address, conn)

	// Start connection in goroutine
	go m.connectAndMonitor(conn)

	log.WithFields(log.Fields{
		"pool_address": address,
	}).Info("交易监控已创建")
	return nil
}

// StopMonitoring stops monitoring a pool address
func (m *PoolMonitorManager) StopMonitoring(address string) error {
	value, exists := m.connections.Load(address)
	if !exists {
		return fmt.Errorf("connection for address %s not found", address)
	}

	conn := value.(*PoolConnection)
	close(conn.StopCh)
	m.connections.Delete(address)
	log.WithFields(log.Fields{
		"pool_address": address,
	}).Info("Swap交易监控已停止")

	// Clean up RabbitMQ resources
	m.cleanupRabbitMQResources(address)

	return nil
}

// incrementErrorCount increments the error count and checks if threshold is reached
// Returns true if error count exceeds threshold and monitoring should be stopped
func (m *PoolMonitorManager) incrementErrorCount(conn *PoolConnection) bool {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.errorCount++
	log.WithFields(log.Fields{
		"pool_address": conn.Address,
		"error_count":  conn.errorCount,
		"max_errors":   maxErrorCount,
	}).Warn("Error count increased")

	if conn.errorCount >= maxErrorCount {
		log.WithFields(log.Fields{
			"pool_address": conn.Address,
			"error_count":  conn.errorCount,
			"max_errors":   maxErrorCount,
		}).Error("Error count exceeded threshold, stopping monitoring and cleaning up RabbitMQ resources")
		return true
	}

	return false
}

// resetErrorCount resets the error count (called on successful operations)
func (m *PoolMonitorManager) resetErrorCount(conn *PoolConnection) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.errorCount > 0 {
		log.WithFields(log.Fields{
			"pool_address": conn.Address,
			"error_count":  conn.errorCount,
		}).Debug("Resetting error count")
		conn.errorCount = 0
	}
}

// cleanupRabbitMQResources cleans up RabbitMQ resources for a pool address
func (m *PoolMonitorManager) cleanupRabbitMQResources(address string) {
	if dbconfig.RabbitMQ == nil {
		log.WithFields(log.Fields{
			"pool_address": address,
		}).Debug("RabbitMQ not initialized, skipping cleanup")
		return
	}

	// Try to delete queue named after the address (if it exists)
	// This handles cases where a dedicated queue was created for this address
	queueName := fmt.Sprintf("meteora_pool_monitor_%s", address)
	if err := dbconfig.DeleteQueue(queueName); err != nil {
		// Queue might not exist, which is fine - log as debug
		log.WithFields(log.Fields{
			"pool_address": address,
			"queue_name":   queueName,
			"error":        err.Error(),
		}).Debug("Queue does not exist or failed to delete")
	} else {
		log.WithFields(log.Fields{
			"pool_address": address,
			"queue_name":   queueName,
		}).Info("Deleted RabbitMQ queue")
	}

	// Also try to purge messages from the main shared queue that might be related to this address
	// Note: This is a best-effort cleanup since we can't selectively delete messages by content
	// The main queue "meteora_pool_monitor" is shared, so we don't delete it
	// Individual messages will be processed and ignored if monitoring is already stopped
}

// connectAndMonitor handles the WebSocket connection and monitoring
func (m *PoolMonitorManager) connectAndMonitor(conn *PoolConnection) {
	reconnectAttempts := 0

	for {
		select {
		case <-conn.StopCh:
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
			}).Info("Stopping monitoring")
			if conn.Conn != nil {
				conn.Conn.Close()
			}
			return
		default:
			// Update status to connecting
			conn.mu.Lock()
			conn.Status = StateConnecting
			conn.mu.Unlock()

			// Connect to Solana WebSocket
			c, _, err := websocket.DefaultDialer.Dial(conn.wsEndpoint, nil)
			if err != nil {
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
					"error":        err.Error(),
				}).Error("Failed to connect to Solana WebSocket")
				reconnectAttempts++

				// Increment error count and check if we should stop
				if m.incrementErrorCount(conn) {
					log.WithFields(log.Fields{
						"pool_address": conn.Address,
					}).Error("Stopping monitoring due to excessive errors")
					m.StopMonitoring(conn.Address)
					return
				}

				if reconnectAttempts >= maxReconnectAttempts {
					log.WithFields(log.Fields{
						"pool_address":           conn.Address,
						"reconnect_attempts":     reconnectAttempts,
						"max_reconnect_attempts": maxReconnectAttempts,
					}).Error("Max reconnect attempts reached, stopping")
					m.StopMonitoring(conn.Address)
					return
				}
				time.Sleep(reconnectDelay)
				continue
			}

			conn.mu.Lock()
			conn.Conn = c
			conn.Status = StateConnected
			conn.mu.Unlock()

			reconnectAttempts = 0
			// Reset error count on successful connection
			m.resetErrorCount(conn)
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
			}).Info("Connected to Solana WebSocket")

			// Subscribe to logs for this address (equivalent to onLogs in TypeScript)
			poolPubkey, err := solana.PublicKeyFromBase58(conn.Address)
			if err != nil {
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
					"error":        err.Error(),
				}).Error("Invalid pool address")
				c.Close()
				// Increment error count and check if we should stop
				if m.incrementErrorCount(conn) {
					log.WithFields(log.Fields{
						"pool_address": conn.Address,
					}).Error("Stopping monitoring due to excessive errors")
					m.StopMonitoring(conn.Address)
					return
				}
				time.Sleep(reconnectDelay)
				continue
			}

			subscribeMsg := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "logsSubscribe",
				"params": []interface{}{
					map[string]interface{}{
						"mentions": []string{poolPubkey.String()},
					},
					map[string]interface{}{
						"commitment": "confirmed",
					},
				},
			}

			if err := c.WriteJSON(subscribeMsg); err != nil {
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
					"error":        err.Error(),
				}).Error("Failed to send subscription message")
				c.Close()
				// Increment error count and check if we should stop
				if m.incrementErrorCount(conn) {
					log.WithFields(log.Fields{
						"pool_address": conn.Address,
					}).Error("Stopping monitoring due to excessive errors")
					m.StopMonitoring(conn.Address)
					return
				}
				time.Sleep(reconnectDelay)
				continue
			}

			log.WithFields(log.Fields{
				"pool_address": conn.Address,
			}).Info("开始监控Swap交易...")

			// Start reading messages
			go m.readMessages(conn)

			// Wait for reconnect signal or stop signal
			select {
			case <-conn.ReconnectCh:
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
				}).Info("Reconnect requested")
				c.Close()
				time.Sleep(reconnectDelay)
			case <-conn.StopCh:
				c.Close()
				return
			}
		}
	}
}

// readMessages reads messages from WebSocket connection
func (m *PoolMonitorManager) readMessages(conn *PoolConnection) {
	defer func() {
		conn.mu.Lock()
		if conn.Conn != nil {
			conn.Conn.Close()
		}
		conn.Status = StateDisconnected
		conn.mu.Unlock()

		// Trigger reconnect
		select {
		case conn.ReconnectCh <- true:
		default:
		}
	}()

	for {
		conn.mu.RLock()
		c := conn.Conn
		conn.mu.RUnlock()

		if c == nil {
			return
		}

		_, message, err := c.ReadMessage()
		if err != nil {
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
				"error":        err.Error(),
			}).Error("Error reading message")
			// Increment error count and check if we should stop
			if m.incrementErrorCount(conn) {
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
				}).Error("Stopping monitoring due to excessive errors")
				m.StopMonitoring(conn.Address)
			}
			return
		}

		// Reset error count on successful message read
		m.resetErrorCount(conn)

		conn.mu.Lock()
		conn.LastMessage = time.Now()
		conn.mu.Unlock()

		// Process message
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
				"error":        err.Error(),
			}).Error("Failed to unmarshal message")
			// Increment error count and check if we should stop
			if m.incrementErrorCount(conn) {
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
				}).Error("Stopping monitoring due to excessive errors")
				m.StopMonitoring(conn.Address)
				return
			}
			continue
		}

		// Log all received messages for debugging
		// log.Infof("Received message for %s: %+v", conn.Address, msg)

		// Handle subscription confirmation (response to logsSubscribe)
		// The response format is: {"jsonrpc":"2.0","result":<subscription_id>,"id":1}
		if id, hasID := msg["id"]; hasID {
			if result, ok := msg["result"].(float64); ok {
				conn.mu.Lock()
				conn.SubscriptionID = result
				conn.mu.Unlock()
				log.WithFields(log.Fields{
					"pool_address":    conn.Address,
					"subscription_id": result,
					"request_id":      id,
				}).Info("Subscription confirmed")
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
				}).Info("Swap交易监控已启动")
				continue
			}
			// Check if result is in a different format
			if result, ok := msg["result"]; ok {
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
					"result_type":  fmt.Sprintf("%T", result),
					"result_value": result,
				}).Info("Subscription response received but result type unexpected")
			}
		}

		// Handle log notification (subscription update)
		// The notification format is: {"jsonrpc":"2.0","method":"logsNotification","params":{"result":{...},"subscription":<id>}}
		if method, ok := msg["method"].(string); ok {
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
				"method":       method,
			}).Info("Received method")
			if method == "logsNotification" {
				if params, ok := msg["params"].(map[string]interface{}); ok {
					// log.Infof("Log notification params: %+v", params)
					if result, ok := params["result"].(map[string]interface{}); ok {
						// Extract error information if present (but still process the transaction)
						var txError string
						if err, hasErr := result["err"]; hasErr && err != nil {
							// Convert error to string for logging
							if errStr, ok := err.(string); ok {
								txError = errStr
							} else if errMap, ok := err.(map[string]interface{}); ok {
								// Try to extract error message from error object
								if errBytes, err := json.Marshal(errMap); err == nil {
									txError = string(errBytes)
								} else {
									txError = fmt.Sprintf("%v", err)
								}
							} else {
								txError = fmt.Sprintf("%v", err)
							}
							log.WithFields(log.Fields{
								"pool_address": conn.Address,
								"error":        txError,
							}).Warn("Transaction error detected")
						}

						// Extract signature from logs
						// For logsSubscribe, the signature is typically in result.value.signature
						if value, ok := result["value"].(map[string]interface{}); ok {
							if signature, ok := value["signature"].(string); ok {
								// log.Infof("Received log notification for %s, signature: %s (error: %v)", conn.Address, signature, txError != "")
								// Process transaction in goroutine to avoid blocking, pass error info
								go m.processTransactionWithError(conn, signature, txError)
								continue
							}
							log.WithFields(log.Fields{
								"pool_address": conn.Address,
								"value":        value,
							}).Info("Value object in log notification")
						}

						// Also try direct signature field
						if signature, ok := result["signature"].(string); ok {
							log.WithFields(log.Fields{
								"pool_address": conn.Address,
								"signature":    signature,
								"has_error":    txError != "",
							}).Info("Received log notification")
							go m.processTransactionWithError(conn, signature, txError)
							continue
						}

						log.WithFields(log.Fields{
							"pool_address": conn.Address,
							"result":       result,
						}).Info("Log notification received but no signature found")
					} else {
						log.WithFields(log.Fields{
							"pool_address": conn.Address,
							"result_type":  fmt.Sprintf("%T", params["result"]),
							"result_value": params["result"],
						}).Info("Log notification params.result is not a map")
					}
				}
			}
		}

		// Handle errors
		if err, ok := msg["error"].(map[string]interface{}); ok {
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
				"error":        err,
			}).Error("WebSocket error")
			// Increment error count and check if we should stop
			if m.incrementErrorCount(conn) {
				log.WithFields(log.Fields{
					"pool_address": conn.Address,
				}).Error("Stopping monitoring due to excessive errors")
				m.StopMonitoring(conn.Address)
				return
			}
		}
	}
}

// getTransactionWithRetry fetches a transaction with exponential backoff retry mechanism
// This is useful for handling "not found" errors which may occur when transactions are still pending
func (m *PoolMonitorManager) getTransactionWithRetry(ctx context.Context, conn *PoolConnection, sig solana.Signature, signature string) (*rpc.GetParsedTransactionResult, error) {
	var lastErr error
	delay := initialRetryDelay

	for attempt := 0; attempt <= maxTransactionRetries; attempt++ {
		// Try to get transaction
		tx, err := conn.RPCClient.GetParsedTransaction(ctx, sig, &rpc.GetParsedTransactionOpts{
			MaxSupportedTransactionVersion: func() *uint64 { v := uint64(0); return &v }(),
		})

		if err == nil {
			// Success - return transaction
			if attempt > 0 {
				log.WithFields(log.Fields{
					"signature":      signature,
					"retry_attempts": attempt,
				}).Info("Successfully retrieved transaction after retries")
			}
			return tx, nil
		}

		lastErr = err
		errStr := strings.ToLower(err.Error())
		isNotFound := strings.Contains(errStr, "not found")

		// If it's not a "not found" error, don't retry
		if !isNotFound {
			log.WithFields(log.Fields{
				"signature": signature,
				"error":     err.Error(),
			}).Error("Failed to get transaction (non-retryable error)")
			return nil, err
		}

		// If it's the last attempt, return the error
		if attempt >= maxTransactionRetries {
			log.WithFields(log.Fields{
				"signature":      signature,
				"retry_attempts": maxTransactionRetries + 1,
				"error":          err.Error(),
			}).Debug("Transaction not found after all retry attempts")
			return nil, err
		}

		// Log retry attempt
		log.WithFields(log.Fields{
			"signature":      signature,
			"attempt":        attempt + 1,
			"max_attempts":   maxTransactionRetries,
			"retry_delay_ms": delay.Milliseconds(),
		}).Debug("Transaction not found, retrying")

		// Wait before retry with exponential backoff
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * retryBackoffMultiplier)
		if delay > maxRetryDelay {
			delay = maxRetryDelay
		}
	}

	return nil, lastErr
}

// processTransaction fetches and parses a transaction (backward compatibility)
func (m *PoolMonitorManager) processTransaction(conn *PoolConnection, signature string) {
	m.processTransactionWithError(conn, signature, "")
}

// processTransactionWithError fetches and parses a transaction, including failed ones
func (m *PoolMonitorManager) processTransactionWithError(conn *PoolConnection, signature string, txError string) {
	// Use longer timeout to accommodate retries
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get parsed transaction
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		log.WithFields(log.Fields{
			"signature": signature,
			"error":     err.Error(),
		}).Error("Invalid signature format")
		return
	}

	// Get transaction with retry mechanism
	tx, err := m.getTransactionWithRetry(ctx, conn, sig, signature)
	if err != nil {
		// Check if error is "not found" after all retries
		errStr := strings.ToLower(err.Error())
		isNotFound := strings.Contains(errStr, "not found")

		if isNotFound {
			// After all retries, transaction still not found - this is acceptable
			// Don't increment error count, just log as debug
			log.WithFields(log.Fields{
				"signature": signature,
				"error":     err.Error(),
			}).Debug("Transaction not found after retries (may be pending or dropped)")
			return
		}

		// For other errors, log as error and increment error count
		log.WithFields(log.Fields{
			"signature": signature,
			"error":     err.Error(),
		}).Error("Failed to get transaction after retries")
		// Increment error count and check if we should stop
		if m.incrementErrorCount(conn) {
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
			}).Error("Stopping monitoring due to excessive errors")
			m.StopMonitoring(conn.Address)
		}
		return
	}

	if tx == nil {
		return
	}

	// Check if transaction failed (from RPC response or from log notification)
	isSuccess := txError == ""
	if tx.Meta != nil && tx.Meta.Err != nil {
		isSuccess = false
		// If we don't have error from log notification, extract from meta
		if txError == "" {
			if errBytes, err := json.Marshal(tx.Meta.Err); err == nil {
				txError = string(errBytes)
			} else {
				txError = fmt.Sprintf("%v", tx.Meta.Err)
			}
		}
	}

	// Serialize transaction meta to JSON string for TxMeta field (not logged)
	var txMeta string
	if tx.Meta != nil {
		if metaBytes, err := json.Marshal(tx.Meta); err == nil {
			txMeta = string(metaBytes)
		} else {
			log.WithFields(log.Fields{
				"signature": signature,
				"error":     err.Error(),
			}).Warn("Failed to marshal transaction meta")
			txMeta = fmt.Sprintf("%+v", tx.Meta)
		}
	}

	// Parse swap transaction (even if failed, we still try to extract information)
	swapTx := m.parseSwapTransaction(conn, tx, signature, isSuccess, txError, txMeta)
	if swapTx != nil {
		// Save to database with filtering
		go m.saveSwapTransactionToDB(swapTx, conn)

		// Call callback if provided
		if conn.SwapCallback != nil {
			conn.SwapCallback(swapTx)
		}

		// If action is "remove liquidity" and transaction succeeded, stop monitoring
		if swapTx.Action == "remove liquidity" && swapTx.Success {
			log.WithFields(log.Fields{
				"pool_address": conn.Address,
				"signature":    swapTx.Signature,
				"action":       swapTx.Action,
			}).Info("Remove liquidity detected, stopping monitor")
			m.StopMonitoring(conn.Address)
		}
	}
}

// parseSwapTransaction parses a transaction to extract swap information
func (m *PoolMonitorManager) parseSwapTransaction(conn *PoolConnection, tx *rpc.GetParsedTransactionResult, signature string, isSuccess bool, txError string, txMeta string) *SwapTransaction {
	if tx.Transaction == nil {
		return nil
	}

	// At least one authority is required
	if conn.MeteoraDbcAuthority == "" && conn.MeteoraCpmmAuthority == "" {
		return nil
	}

	// Get transaction slot
	slot := tx.Slot

	// Get transaction timestamp from block time
	var timestamp int64
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	blockTime, err := conn.RPCClient.GetBlockTime(ctx, slot)
	if err != nil || blockTime == nil {
		timestamp = time.Now().UnixMilli()
	} else {
		timestamp = int64(*blockTime) * 1000 // Convert to milliseconds
	}

	// Extract signers and payer (always available, even for failed transactions)
	payer, signers := m.extractSignersAndPayer(tx)

	// Skip if payer is empty (cannot track balance changes without payer)
	if payer == "" {
		log.WithFields(log.Fields{
			"signature": signature,
		}).Debug("Payer is empty, skipping balance change calculation")
		return nil
	}

	// For transactions without meta, we still create a SwapTransaction with minimal info
	// But we can't calculate balance changes without meta
	if tx.Meta == nil {
		return &SwapTransaction{
			Signature: signature,
			Slot:      slot,
			Timestamp: timestamp,
			Action:    "unknown",
			BaseToken: TokenInfo{
				Symbol:  "Token",
				Amount:  0,
				Address: conn.BaseTokenMint,
			},
			QuoteToken: TokenInfo{
				Symbol:  "SOL",
				Amount:  0,
				Address: conn.QuoteTokenMint,
			},
			Value:   0,
			Payer:   payer,
			Signers: signers,
			Success: isSuccess,
			Error:   txError,
			TxMeta:  txMeta,
		}
	}

	// Get token balance changes (available even for failed transactions if meta exists)
	// Always calculate balance changes regardless of transaction success status
	preTokenBalances := tx.Meta.PreTokenBalances
	postTokenBalances := tx.Meta.PostTokenBalances

	// Find token balance changes for base and quote tokens for authorities (pool)
	// This works for both successful and failed transactions
	// Calculate pool's balance changes to track liquidity changes
	// Sum up balance changes from both meteoraDbcAuthority and meteoraCpmmAuthority
	baseChange := m.getTokenBalanceChanges(preTokenBalances, postTokenBalances, conn.BaseTokenMint, conn.MeteoraDbcAuthority, conn.MeteoraCpmmAuthority)
	quoteChange := m.getTokenBalanceChanges(preTokenBalances, postTokenBalances, conn.QuoteTokenMint, conn.MeteoraDbcAuthority, conn.MeteoraCpmmAuthority)

	// If no balance changes found, still return transaction info (might be a failed attempt)
	baseChangeValue := 0.0
	if baseChange != nil {
		baseChangeValue = baseChange.Change
	}

	quoteChangeValue := 0.0
	if quoteChange != nil {
		quoteChangeValue = quoteChange.Change
	}

	// Determine action type (if we have balance changes)
	action := "unknown"
	if baseChange != nil || quoteChange != nil {
		action = m.determineAction(baseChangeValue, quoteChangeValue)
	}

	return &SwapTransaction{
		Signature: signature,
		Slot:      slot,
		Timestamp: timestamp,
		Action:    action,
		BaseToken: TokenInfo{
			Symbol:  "Token",          // TODO: Get actual token symbol
			Amount:  -baseChangeValue, // Negative value as per TypeScript
			Address: conn.BaseTokenMint,
		},
		QuoteToken: TokenInfo{
			Symbol:  "SOL",             // TODO: Get actual token symbol
			Amount:  -quoteChangeValue, // Negative value as per TypeScript
			Address: conn.QuoteTokenMint,
		},
		Value:   abs(quoteChangeValue),
		Payer:   payer,
		Signers: signers,
		Success: isSuccess,
		Error:   txError,
		TxMeta:  txMeta,
	}
}

// TokenBalanceChange represents a token balance change
type TokenBalanceChange struct {
	Change float64
}

// getTokenBalanceChanges calculates the token balance change by summing changes from both authorities
func (m *PoolMonitorManager) getTokenBalanceChanges(preBalances, postBalances []rpc.TokenBalance, mintStr, meteoraDbcAuthority, meteoraCpmmAuthority string) *TokenBalanceChange {
	// Convert string to PublicKey for comparison
	mintPK, err := solana.PublicKeyFromBase58(mintStr)
	if err != nil {
		return nil
	}

	var totalChange float64
	var foundAny bool

	// Helper function to calculate balance change for a single authority
	calculateChangeForAuthority := func(authorityStr string) float64 {
		if authorityStr == "" {
			return 0.0
		}

		ownerPK, err := solana.PublicKeyFromBase58(authorityStr)
		if err != nil {
			return 0.0
		}

		var preAmount, postAmount float64
		var foundPre, foundPost bool

		// Find pre-balance
		for _, bal := range preBalances {
			if bal.Mint.Equals(mintPK) && bal.Owner != nil && bal.Owner.Equals(ownerPK) {
				foundPre = true
				if bal.UiTokenAmount != nil && bal.UiTokenAmount.UiAmount != nil {
					preAmount = *bal.UiTokenAmount.UiAmount
				} else {
					// Account exists but uiAmount is null, treat as 0
					preAmount = 0.0
				}
				break
			}
		}

		// Find post-balance
		for _, bal := range postBalances {
			if bal.Mint.Equals(mintPK) && bal.Owner != nil && bal.Owner.Equals(ownerPK) {
				foundPost = true
				if bal.UiTokenAmount != nil && bal.UiTokenAmount.UiAmount != nil {
					postAmount = *bal.UiTokenAmount.UiAmount
				} else {
					// Account exists but uiAmount is null, treat as 0
					postAmount = 0.0
				}
				break
			}
		}

		// If post-balance not found, return 0 (no change for this authority)
		if !foundPost {
			return 0.0
		}

		// If pre-balance not found but post-balance exists, it means the account was created in this transaction
		// In this case, preAmount is 0 (new account starts with 0 balance)
		if !foundPre {
			preAmount = 0.0
			log.WithFields(log.Fields{
				"mint":      mintStr,
				"authority": authorityStr,
			}).Debug("Token account was created in this transaction, treating pre-balance as 0")
		}

		return postAmount - preAmount
	}

	// Calculate changes for both authorities and sum them up
	dbcChange := calculateChangeForAuthority(meteoraDbcAuthority)
	cpmmChange := calculateChangeForAuthority(meteoraCpmmAuthority)
	totalChange = dbcChange + cpmmChange

	// Check if we found any balance changes
	if meteoraDbcAuthority != "" || meteoraCpmmAuthority != "" {
		foundAny = true
	}

	// If no balance changes found, return nil
	if !foundAny {
		return nil
	}

	return &TokenBalanceChange{
		Change: totalChange,
	}
}

// determineAction determines the action type based on token balance changes
func (m *PoolMonitorManager) determineAction(baseChange, quoteChange float64) string {
	// log.Infof("baseChange: %f, quoteChange: %f", baseChange, quoteChange)

	if quoteChange <= -70 {
		return "remove liquidity"
	}

	if baseChange > 0 && quoteChange < 0 {
		return "sell"
	}

	if baseChange < 0 && quoteChange > 0 {
		return "buy"
	}

	if baseChange > 0 && quoteChange > 0 {
		return "add liquidity"
	}

	return "unknown"
}

// extractSignersAndPayer extracts signer addresses from transaction
func (m *PoolMonitorManager) extractSignersAndPayer(tx *rpc.GetParsedTransactionResult) (string, []string) {
	var signers []string
	payer := ""

	if tx == nil || tx.Transaction == nil || tx.Transaction.Message.AccountKeys == nil {
		return payer, signers
	}

	// Extract signers from AccountKeys where Signer == true
	for _, account := range tx.Transaction.Message.AccountKeys {
		if account.Signer {
			signerAddress := account.PublicKey.String()
			signers = append(signers, signerAddress)
		}
	}

	// First signer is the payer
	if len(signers) > 0 {
		payer = signers[0]
	}

	return payer, signers
}

// abs returns absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetConnectionStatus returns the status of a connection
func (m *PoolMonitorManager) GetConnectionStatus(address string) (string, error) {
	value, exists := m.connections.Load(address)
	if !exists {
		return StateDisconnected, fmt.Errorf("connection not found")
	}

	conn := value.(*PoolConnection)
	conn.mu.RLock()
	defer conn.mu.RUnlock()
	return conn.Status, nil
}

// GetAllConnections returns all active connections
func (m *PoolMonitorManager) GetAllConnections() map[string]string {
	result := make(map[string]string)
	m.connections.Range(func(key, value interface{}) bool {
		address := key.(string)
		conn := value.(*PoolConnection)
		conn.mu.RLock()
		status := conn.Status
		conn.mu.RUnlock()
		result[address] = status
		return true
	})
	return result
}

// loadRoleAddressMap loads all RoleAddress records and creates a map for quick lookup
func (m *PoolMonitorManager) loadRoleAddressMap() (map[string]bool, error) {
	var roleAddresses []models.RoleAddress
	if err := dbconfig.DB.Find(&roleAddresses).Error; err != nil {
		return nil, err
	}

	// Create a map for quick lookup
	roleAddressMap := make(map[string]bool)
	for _, roleAddr := range roleAddresses {
		roleAddressMap[roleAddr.Address] = true
	}

	return roleAddressMap, nil
}

// saveSwapTransactionToDB saves swap transaction to database after filtering by RoleAddress
func (m *PoolMonitorManager) saveSwapTransactionToDB(swapTx *SwapTransaction, conn *PoolConnection) {
	// Get roleAddressMap from connection (cached in memory)
	conn.mu.RLock()
	roleAddressMap := conn.roleAddressMap
	conn.mu.RUnlock()

	// If map is nil (failed to load initially), try to load it now
	if roleAddressMap == nil {
		var err error
		roleAddressMap, err = m.loadRoleAddressMap()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Failed to get RoleAddress list")
			return
		}
		// Update the connection with the loaded map
		conn.mu.Lock()
		conn.roleAddressMap = roleAddressMap
		conn.mu.Unlock()
	}

	// Filter: skip if payer is in RoleAddress list
	if roleAddressMap[swapTx.Payer] {
		log.WithFields(log.Fields{
			"signature": swapTx.Signature,
			"payer":     swapTx.Payer,
			"action":    swapTx.Action,
		}).Debug("Skipping swap transaction: payer is in RoleAddress list")
		return
	}

	// Convert meteora.SwapTransaction to models.SwapTransaction
	// Check if transaction already exists
	var existingTx models.SwapTransaction
	err := dbconfig.DB.Where("signature = ?", swapTx.Signature).First(&existingTx).Error
	if err == nil {
		log.WithFields(log.Fields{
			"signature": swapTx.Signature,
		}).Debug("Swap transaction already exists, skipping")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.WithFields(log.Fields{
			"signature": swapTx.Signature,
			"error":     err.Error(),
		}).Error("Failed to check if swap transaction exists")
		return
	}

	// Determine payer type based on action
	payerType := ""
	switch swapTx.Action {
	case "buy":
		payerType = "buyer"
	case "sell":
		payerType = "seller"
	case "add liquidity":
		payerType = "liquidity_provider"
	case "remove liquidity":
		payerType = "liquidity_remover"
	default:
		payerType = "unknown"
	}

	// Convert timestamp from milliseconds to seconds
	timestamp := uint(swapTx.Timestamp / 1000)
	if swapTx.Timestamp < 0 {
		// Handle negative timestamp (shouldn't happen, but be safe)
		timestamp = 0
	}

	// Create database record
	dbSwapTx := models.SwapTransaction{
		Signature:   swapTx.Signature,
		Slot:        uint(swapTx.Slot),
		Timestamp:   timestamp,
		PayerType:   payerType,
		Payer:       swapTx.Payer,
		PoolAddress: conn.Address,
		BaseMint:    conn.BaseTokenMint,
		QuoteMint:   conn.QuoteTokenMint,
		BaseChange:  swapTx.BaseToken.Amount,
		QuoteChange: swapTx.QuoteToken.Amount,
		IsSuccess:   swapTx.Success,
		TxMeta:      swapTx.TxMeta,
		TxError:     swapTx.Error,
	}

	// Save to database
	if err := dbconfig.DB.Create(&dbSwapTx).Error; err != nil {
		log.WithFields(log.Fields{
			"signature": swapTx.Signature,
			"error":     err.Error(),
		}).Error("Failed to save swap transaction to database")
		return
	}

	// Log with structured fields, excluding TxMeta
	log.WithFields(log.Fields{
		"signature":    swapTx.Signature,
		"pool_address": conn.Address,
		"payer":        swapTx.Payer,
		"action":       swapTx.Action,
		"base_mint":    conn.BaseTokenMint,
		"quote_mint":   conn.QuoteTokenMint,
		"base_amount":  swapTx.BaseToken.Amount,
		"quote_amount": swapTx.QuoteToken.Amount,
		"value":        swapTx.Value,
		"slot":         swapTx.Slot,
		"timestamp":    swapTx.Timestamp,
		"success":      swapTx.Success,
		"payer_type":   payerType,
	}).Info("Saved swap transaction to database")
}
