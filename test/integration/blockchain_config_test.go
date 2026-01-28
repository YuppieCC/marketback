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

type BlockchainConfig struct {
	ID        uint   `json:"id"`
	ChainID   uint   `json:"chain_id"`
	Name      string `json:"name"`
	Network   string `json:"network"`
}

func TestBlockchainConfigAPI(t *testing.T) {
	var configID uint
	var chainID uint = 1

	// Test Case 1: Create Blockchain Config
	t.Run("Create Blockchain Config", func(t *testing.T) {
		config := BlockchainConfig{
			ChainID: chainID,
			Name:    "Ethereum",
			Network: "mainnet",
		}

		payload, err := json.Marshal(config)
		require.NoError(t, err)

		resp, err := http.Post(BaseURL+"/blockchain-config", "application/json", bytes.NewBuffer(payload))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response BlockchainConfig
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.NotZero(t, response.ID)
		assert.Equal(t, chainID, response.ChainID)
		configID = response.ID
	})

	// Test Case 2: Get Blockchain Config
	t.Run("Get Blockchain Config", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/blockchain-config/%d", BaseURL, configID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var config BlockchainConfig
		err = json.NewDecoder(resp.Body).Decode(&config)
		require.NoError(t, err)
		assert.Equal(t, configID, config.ID)
		assert.Equal(t, chainID, config.ChainID)
		assert.Equal(t, "Ethereum", config.Name)
	})

	// Test Case 3: Update Blockchain Config
	t.Run("Update Blockchain Config", func(t *testing.T) {
		config := BlockchainConfig{
			ChainID: chainID,
			Name:    "Ethereum Updated",
			Network: "mainnet",
		}

		payload, err := json.Marshal(config)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/blockchain-config/%d", BaseURL, configID), bytes.NewBuffer(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response BlockchainConfig
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, configID, response.ID)
		assert.Equal(t, chainID, response.ChainID)
		assert.Equal(t, "Ethereum Updated", response.Name)
	})

	// Test Case 4: Delete Blockchain Config
	t.Run("Delete Blockchain Config", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/blockchain-config/%d", BaseURL, configID), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test Case 5: Get Non-existent Blockchain Config
	t.Run("Get Non-existent Blockchain Config", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/blockchain-config/999", BaseURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
} 