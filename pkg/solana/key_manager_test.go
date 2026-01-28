package solana

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"bytes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


func TestKeyManager(t *testing.T) {
	km := NewKeyManager()

	// Test key pair generation
	t.Run("Generate Key Pair", func(t *testing.T) {
		account, err := km.GenerateKeyPair()
		require.NoError(t, err)
		assert.NotNil(t, account)
		assert.NotEmpty(t, account.PublicKey.ToBase58())
		assert.NotEmpty(t, account.PrivateKey)
		assert.Equal(t, 64, len(account.PrivateKey), "Private key should be 64 bytes")
	})

	// Test encryption and decryption
	t.Run("Encrypt and Decrypt Private Key", func(t *testing.T) {
		account, err := km.GenerateKeyPair()
		require.NoError(t, err)

		password := "test-password"
		encrypted, err := km.EncryptPrivateKey(account.PrivateKey, password)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		decrypted, err := km.DecryptPrivateKey(encrypted, password)

		// check if the decrypted key is the same as the original key
        assert.True(t, bytes.Equal(account.PrivateKey[:], decrypted), "Decrypted private key should match original")

		require.NoError(t, err)
		assert.Equal(t, len(account.PrivateKey), len(decrypted), "Decrypted key length should match original")
		for i := 0; i < len(account.PrivateKey); i++ {
			assert.Equal(t, account.PrivateKey[i], decrypted[i], "Byte at index %d should match", i)
		}
	})

	// Test file operations with JSON format
	t.Run("Save and Load Encrypted Key as JSON", func(t *testing.T) {
		account, err := km.GenerateKeyPair()
		require.NoError(t, err)

		password := "test-password"
		encrypted, err := km.EncryptPrivateKey(account.PrivateKey, password)
		require.NoError(t, err)

		// Create a keystore entry
		address := account.PublicKey.ToBase58()
		entry := KeyStoreEntry{
			Address:      address,
			EncryptedKey: encrypted,
			Version:      1,
		}

		// Convert to JSON
		jsonData, err := json.MarshalIndent(entry, "", "  ")
		require.NoError(t, err)

		// Create a temporary directory for test files
		tempDir, err := os.MkdirTemp("", "solana-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Save the JSON file with the address as the filename
		filename := filepath.Join(tempDir, address+".json")
		err = os.WriteFile(filename, jsonData, 0600)
		require.NoError(t, err)

		// Read the JSON file
		loadedData, err := os.ReadFile(filename)
		require.NoError(t, err)

		// Parse the JSON
		var loadedEntry KeyStoreEntry
		err = json.Unmarshal(loadedData, &loadedEntry)
		require.NoError(t, err)

		// Verify the loaded data
		assert.Equal(t, address, loadedEntry.Address)
		assert.Equal(t, encrypted, loadedEntry.EncryptedKey)
		assert.Equal(t, 1, loadedEntry.Version)

		// Decrypt the key
		decrypted, err := km.DecryptPrivateKey(loadedEntry.EncryptedKey, password)
		
		// check if the decrypted key is the same as the original key
		assert.True(t, bytes.Equal(account.PrivateKey[:], decrypted), "Decrypted private key should match original")

		require.NoError(t, err)
		assert.Equal(t, len(account.PrivateKey), len(decrypted), "Decrypted key length should match original")
		for i := 0; i < len(account.PrivateKey); i++ {
			assert.Equal(t, account.PrivateKey[i], decrypted[i], "Byte at index %d should match", i)
		}
	})

	// Test address derivation
	t.Run("Get Solana Address", func(t *testing.T) {
		account, err := km.GenerateKeyPair()
		require.NoError(t, err)

		address, err := km.GetSolanaAddressFromPrivateKey(account.PrivateKey)
		require.NoError(t, err)
		assert.Equal(t, account.PublicKey.ToBase58(), address)
	})

	// Test error cases
	t.Run("Error Cases", func(t *testing.T) {
		// Test invalid password
		account, err := km.GenerateKeyPair()
		require.NoError(t, err)

		encrypted, err := km.EncryptPrivateKey(account.PrivateKey, "password1")
		require.NoError(t, err)

		_, err = km.DecryptPrivateKey(encrypted, "password2")
		assert.Error(t, err)

		// Test invalid file
		_, err = km.LoadEncryptedKeyFromFile("nonexistent.enc")
		assert.Error(t, err)

		// Test invalid private key
		_, err = km.GetSolanaAddressFromPrivateKey([]byte("invalid-key"))
		assert.Error(t, err)
	})

	// Test multiple key generation
	t.Run("Multiple Key Generation", func(t *testing.T) {
		// Generate multiple keys and ensure they are unique
		keys := make(map[string]bool)
		for i := 0; i < 10; i++ {
			account, err := km.GenerateKeyPair()
			require.NoError(t, err)
			
			address := account.PublicKey.ToBase58()
			assert.False(t, keys[address], "Generated duplicate address")
			keys[address] = true
		}
	})
} 