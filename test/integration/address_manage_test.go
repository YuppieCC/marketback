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

type AddressManage struct {
	ID         uint   `json:"id"`
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
}

type GenerateAddressResponse struct {
	Message    string         `json:"message"`
	Addresses  []AddressManage `json:"addresses"`
}

func TestAddressManageAPI(t *testing.T) {
	var addressID uint

	// Test Case 1: Generate Addresses
	t.Run("Generate Addresses", func(t *testing.T) {
		request := struct {
			Count int    `json:"count"`
			Tag   string `json:"tag"`
		}{
			Count: 1,
			Tag:   "marketmaker",
		}

		payload, err := json.Marshal(request)
		require.NoError(t, err)

		resp, err := http.Post(BaseURL+"/address-manage/generate", "application/json", bytes.NewBuffer(payload))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response GenerateAddressResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Successfully generated 1 addresses", response.Message)
		assert.Len(t, response.Addresses, 1)
		
		addressID = response.Addresses[0].ID
		assert.Equal(t, "marketmaker", response.Addresses[0].Tag)
		assert.NotEmpty(t, response.Addresses[0].Address)
		assert.NotEmpty(t, response.Addresses[0].PrivateKey)
	})

	// Test Case 2: List Addresses
	t.Run("List Addresses", func(t *testing.T) {
		resp, err := http.Get(BaseURL + "/address-manage")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var addresses []AddressManage
		err = json.NewDecoder(resp.Body).Decode(&addresses)
		require.NoError(t, err)
		assert.NotEmpty(t, addresses)
	})

	// Test Case 3: Get Address
	t.Run("Get Address", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/address-manage/%d", BaseURL, addressID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var address AddressManage
		err = json.NewDecoder(resp.Body).Decode(&address)
		require.NoError(t, err)
		assert.Equal(t, addressID, address.ID)
		assert.Equal(t, "marketmaker", address.Tag)
	})

	// Test Case 4: Update Address
	t.Run("Update Address", func(t *testing.T) {
		address := AddressManage{
			Address:    "0x0987654321098765432109876543210987654321",
			Tag:        "brush",
			PrivateKey: "encrypted_private_key_2",
		}

		payload, err := json.Marshal(address)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%s/address-manage/%d", BaseURL, addressID), bytes.NewBuffer(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response AddressManage
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, addressID, response.ID)
		assert.Equal(t, address.Address, response.Address)
		assert.Equal(t, address.Tag, response.Tag)
		assert.Equal(t, address.PrivateKey, response.PrivateKey)
	})

	// Test Case 5: Delete Address
	t.Run("Delete Address", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/address-manage/%d", BaseURL, addressID), nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// Test Case 6: Generate Multiple Addresses
	t.Run("Generate Multiple Addresses", func(t *testing.T) {
		request := struct {
			Count int    `json:"count"`
			Tag   string `json:"tag"`
		}{
			Count: 3,
			Tag:   "gas-distributor",
		}

		payload, err := json.Marshal(request)
		require.NoError(t, err)

		resp, err := http.Post(BaseURL+"/address-manage/generate", "application/json", bytes.NewBuffer(payload))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response GenerateAddressResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "Successfully generated 3 addresses", response.Message)
		assert.Len(t, response.Addresses, 3)
		for _, addr := range response.Addresses {
			assert.Equal(t, "gas-distributor", addr.Tag)
			assert.NotEmpty(t, addr.Address)
			assert.NotEmpty(t, addr.PrivateKey)
		}
	})

	// Test Case 7: Get Non-existent Address
	t.Run("Get Non-existent Address", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/address-manage/999", BaseURL))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}