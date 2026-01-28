package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type RPCConfig struct {
	ID                uint   `json:"id"`
	Endpoint          string `json:"endpoint"`
	IsActive          bool   `json:"is_active"`
	BlockchainConfigID uint  `json:"blockchain_config_id"`
	BlockchainConfig   BlockchainConfig `json:"blockchain_config"`
}

func TestRPCConfigAPI(t *testing.T) {
	var blockchainConfigID uint = 26

	// Test Case 1: Create RPC Config
	t.Run("Create RPC Config", func(t *testing.T) {
		config := RPCConfig{
			Endpoint:          "https://eth-mainnet.alchemyapi.io/v2/your-api-key",
			IsActive:          true,
			BlockchainConfigID: blockchainConfigID,
		}

		payload, err := json.Marshal(config)
		require.NoError(t, err)

		resp, err := http.Post(BaseURL+"/rpc-config", "application/json", bytes.NewBuffer(payload))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response RPCConfig
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotZero(t, response.ID)
		assert.Equal(t, config.Endpoint, response.Endpoint)
		assert.Equal(t, config.IsActive, response.IsActive)
		assert.Equal(t, config.BlockchainConfigID, response.BlockchainConfigID)
	})

	// Test Case 2: Get RPC Config
	t.Run("Get RPC Config", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/rpc-config/1", BaseURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var config RPCConfig
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		assert.Equal(t, "https://eth-mainnet.alchemyapi.io/v2/your-api-key", config.Endpoint)
		assert.True(t, config.IsActive)
		assert.Equal(t, blockchainConfigID, config.BlockchainConfigID)
	})

	// Test Case 3: Update RPC Config
	t.Run("Update RPC Config", func(t *testing.T) {
		config := RPCConfig{
			Endpoint:          "https://eth-mainnet.alchemyapi.io/v2/new-api-key",
			IsActive:          true,
			BlockchainConfigID: blockchainConfigID,
		}

		payload, err := json.Marshal(config)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/rpc-config/1", BaseURL), bytes.NewBuffer(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response RPCConfig
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, config.Endpoint, response.Endpoint)
		assert.Equal(t, config.IsActive, response.IsActive)
		assert.Equal(t, config.BlockchainConfigID, response.BlockchainConfigID)
	})

	// Test Case 4: Delete RPC Config
	t.Run("Delete RPC Config", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/rpc-config/1", BaseURL), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test Case 5: Get Non-existent RPC Config
	t.Run("Get Non-existent RPC Config", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/rpc-config/999", BaseURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}