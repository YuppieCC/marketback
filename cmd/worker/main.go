package main

import (
	"encoding/json"
	"log"
	"sync"

	"marketcontrol/pkg/config"
	"marketcontrol/pkg/solana/meteora"

	logrus "github.com/sirupsen/logrus"
)

const (
	maxErrorCount = 3 // Maximum consecutive errors before stopping monitoring
)

var (
	// errorCounts tracks error count per address
	errorCounts   = make(map[string]int)
	errorCountsMu sync.RWMutex
)

func main() {
	// Initialize logger
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	// Initialize database
	config.InitDB()

	// Initialize RabbitMQ
	config.InitRabbitMQ()
	defer config.RabbitMQ.Close()

	// Create pool monitor manager
	manager, err := meteora.NewPoolMonitorManager()
	if err != nil {
		logrus.Fatal("Failed to create pool monitor manager: ", err)
	}

	// Create consumer for meteora pool monitoring queue
	msgConsumer, err := config.NewConsumer("meteora_pool_monitor")
	if err != nil {
		logrus.Fatal("Failed to create consumer: ", err)
	}
	defer msgConsumer.Close()

	logrus.Info("Meteora Pool Monitor Worker started, waiting for messages...")

	// Start consuming messages
	err = msgConsumer.Consume(func(msg []byte) error {
		var monitorMsg meteora.PoolMonitorMessage
		if err := json.Unmarshal(msg, &monitorMsg); err != nil {
			logrus.Errorf("Failed to unmarshal message: %v", err)
			return err
		}

		logrus.Infof("Received monitoring request: %+v", monitorMsg)

		// Define swap callback
		swapCallback := func(swap *meteora.SwapTransaction) {
			// Log with structured fields, excluding TxMeta
			logFields := logrus.Fields{
				"signature": swap.Signature,
				"slot":      swap.Slot,
				"timestamp": swap.Timestamp,
				"action":    swap.Action,
				"base_token": logrus.Fields{
					"symbol":  swap.BaseToken.Symbol,
					"amount":  swap.BaseToken.Amount,
					"address": swap.BaseToken.Address,
				},
				"quote_token": logrus.Fields{
					"symbol":  swap.QuoteToken.Symbol,
					"amount":  swap.QuoteToken.Amount,
					"address": swap.QuoteToken.Address,
				},
				"value":   swap.Value,
				"payer":   swap.Payer,
				"signers": swap.Signers,
				"success": swap.Success,
			}
			// Only include error if present
			if swap.Error != "" {
				logFields["error"] = swap.Error
			}
			logrus.WithFields(logFields).Info("Swap transaction detected")
			// TODO: Add your business logic here
			// For example: save to database, trigger notifications, etc.
		}

		// Handle start monitoring action
		if monitorMsg.Action == "start_monitoring" {
			// Start monitoring Meteoradbc address
			if monitorMsg.MeteoradbcAddress != "" {
				if err := manager.StartMonitoring(
					monitorMsg.MeteoradbcAddress,
					monitorMsg.BaseTokenMint,
					monitorMsg.QuoteTokenMint,
					monitorMsg.MeteoraDbcAuthority,
					monitorMsg.MeteoraCpmmAuthority,
					swapCallback,
				); err != nil {
					logrus.Errorf("Failed to start monitoring Meteoradbc address %s: %v",
						monitorMsg.MeteoradbcAddress, err)

					// Increment error count and check if we should stop
					count := incrementErrorCount(monitorMsg.MeteoradbcAddress)
					if count >= maxErrorCount {
						logrus.Errorf("Error count exceeded threshold for %s, cleaning up RabbitMQ resources",
							monitorMsg.MeteoradbcAddress)
						cleanupRabbitMQResources(monitorMsg.MeteoradbcAddress)
						// Don't return error, just log and continue
						logrus.Warnf("Skipping monitoring for %s due to excessive errors", monitorMsg.MeteoradbcAddress)
					} else {
						return err
					}
				} else {
					// Reset error count on successful start
					resetErrorCount(monitorMsg.MeteoradbcAddress)
					logrus.Infof("Started monitoring Meteoradbc address: %s", monitorMsg.MeteoradbcAddress)
				}
			}

			// Start monitoring Meteoracpmm address
			if monitorMsg.MeteoracpmmAddress != "" {
				if err := manager.StartMonitoring(
					monitorMsg.MeteoracpmmAddress,
					monitorMsg.BaseTokenMint,
					monitorMsg.QuoteTokenMint,
					monitorMsg.MeteoraDbcAuthority,
					monitorMsg.MeteoraCpmmAuthority,
					swapCallback,
				); err != nil {
					logrus.Errorf("Failed to start monitoring Meteoracpmm address %s: %v",
						monitorMsg.MeteoracpmmAddress, err)

					// Increment error count and check if we should stop
					count := incrementErrorCount(monitorMsg.MeteoracpmmAddress)
					if count >= maxErrorCount {
						logrus.Errorf("Error count exceeded threshold for %s, cleaning up RabbitMQ resources",
							monitorMsg.MeteoracpmmAddress)
						cleanupRabbitMQResources(monitorMsg.MeteoracpmmAddress)
						// Don't return error, just log and continue
						logrus.Warnf("Skipping monitoring for %s due to excessive errors", monitorMsg.MeteoracpmmAddress)
					} else {
						return err
					}
				} else {
					// Reset error count on successful start
					resetErrorCount(monitorMsg.MeteoracpmmAddress)
					logrus.Infof("Started monitoring Meteoracpmm address: %s", monitorMsg.MeteoracpmmAddress)
				}
			}
		} else if monitorMsg.Action == "stop_monitoring" {
			// Handle stop monitoring action
			if monitorMsg.MeteoradbcAddress != "" {
				if err := manager.StopMonitoring(monitorMsg.MeteoradbcAddress); err != nil {
					logrus.Errorf("Failed to stop monitoring Meteoradbc address %s: %v",
						monitorMsg.MeteoradbcAddress, err)
				} else {
					logrus.Infof("Stopped monitoring Meteoradbc address: %s", monitorMsg.MeteoradbcAddress)
				}
			}

			if monitorMsg.MeteoracpmmAddress != "" {
				if err := manager.StopMonitoring(monitorMsg.MeteoracpmmAddress); err != nil {
					logrus.Errorf("Failed to stop monitoring Meteoracpmm address %s: %v",
						monitorMsg.MeteoracpmmAddress, err)
				} else {
					logrus.Infof("Stopped monitoring Meteoracpmm address: %s", monitorMsg.MeteoracpmmAddress)
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatal("Failed to start consumer: ", err)
	}
}

// incrementErrorCount increments the error count for an address
func incrementErrorCount(address string) int {
	errorCountsMu.Lock()
	defer errorCountsMu.Unlock()

	errorCounts[address]++
	count := errorCounts[address]
	logrus.Warnf("Error count for address %s: %d/%d", address, count, maxErrorCount)
	return count
}

// resetErrorCount resets the error count for an address
func resetErrorCount(address string) {
	errorCountsMu.Lock()
	defer errorCountsMu.Unlock()

	if errorCounts[address] > 0 {
		logrus.Debugf("Resetting error count for address %s (was %d)", address, errorCounts[address])
		errorCounts[address] = 0
	}
}

// cleanupRabbitMQResources cleans up RabbitMQ resources for an address
func cleanupRabbitMQResources(address string) {
	if config.RabbitMQ == nil {
		logrus.Debugf("RabbitMQ not initialized, skipping cleanup for address: %s", address)
		return
	}

	// Try to delete queue named after the address (if it exists)
	queueName := "meteora_pool_monitor_" + address
	if err := config.DeleteQueue(queueName); err != nil {
		// Queue might not exist, which is fine - log as debug
		logrus.Debugf("Queue %s does not exist or failed to delete: %v", queueName, err)
	} else {
		logrus.Infof("Deleted RabbitMQ queue for address %s: %s", address, queueName)
	}

	// Reset error count after cleanup
	errorCountsMu.Lock()
	delete(errorCounts, address)
	errorCountsMu.Unlock()
}
