package solana

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/blocto/solana-go-sdk/types"
)

// KeyStoreEntry represents a keystore entry with metadata
type KeyStoreEntry struct {
	Address      string `json:"address"`
	EncryptedKey string `json:"encrypted_key"`
	Version      int    `json:"version"`
}

// KeyManager handles Solana key pair generation, encryption, and decryption
type KeyManager struct {
	// No fields needed for now
}

// NewKeyManager creates a new KeyManager instance
func NewKeyManager() *KeyManager {
	return &KeyManager{}
}

// GenerateKeyPair generates a new Solana key pair
func (km *KeyManager) GenerateKeyPair() (*types.Account, error) {
	account := types.NewAccount()
	return &account, nil
}

// EncryptPrivateKey encrypts a private key using AES-256-GCM
func (km *KeyManager) EncryptPrivateKey(privateKey []byte, password string) (string, error) {
	key := deriveKey(password) // 32-byte key for AES-256
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Combine nonce and ciphertext for storage
	ciphertext := gcm.Seal(nonce, nonce, privateKey, nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encoded, nil
}

// DecryptPrivateKey decrypts a private key using AES-256-GCM
func (km *KeyManager) DecryptPrivateKey(encryptedKey string, password string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// SaveEncryptedKeyToFile saves an encrypted private key to a file
func (km *KeyManager) SaveEncryptedKeyToFile(encryptedKey string, filename string) error {
	// Ensure the keystore directory exists
	keystoreDir := "configs/keystore"
	if err := os.MkdirAll(keystoreDir, 0700); err != nil {
		return fmt.Errorf("failed to create keystore directory: %w", err)
	}

	// Create the full path for the file
	fullPath := filepath.Join(keystoreDir, filename)

	// Write the encrypted key to the file
	if err := os.WriteFile(fullPath, []byte(encryptedKey), 0600); err != nil {
		return fmt.Errorf("failed to write encrypted key to file: %w", err)
	}

	return nil
}

// LoadEncryptedKeyFromFile loads an encrypted private key from a file
func (km *KeyManager) LoadEncryptedKeyFromFile(filename string) (string, error) {
	// Create the full path for the file
	fullPath := filepath.Join("configs/keystore", filename)

	// Read the encrypted key from the file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read encrypted key from file: %w", err)
	}

	return string(data), nil
}

// SaveKeyStoreEntry saves a keystore entry to a JSON file
func (km *KeyManager) SaveKeyStoreEntry(account *types.Account, password string) error {
	// Encrypt the private key
	encrypted, err := km.EncryptPrivateKey(account.PrivateKey, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt private key: %w", err)
	}

	// Create a keystore entry
	address := account.PublicKey.ToBase58()
	entry := KeyStoreEntry{
		Address:      address,
		EncryptedKey: encrypted,
		Version:      1,
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keystore entry: %w", err)
	}

	// Ensure the keystore directory exists
	keystoreDir := "configs/keystore"
	if err := os.MkdirAll(keystoreDir, 0700); err != nil {
		return fmt.Errorf("failed to create keystore directory: %w", err)
	}

	// Save the JSON file with the address as the filename
	filename := filepath.Join(keystoreDir, address+".json")
	if err := os.WriteFile(filename, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write keystore entry to file: %w", err)
	}

	return nil
}

// LoadKeyStoreEntry loads a keystore entry from a JSON file
func (km *KeyManager) LoadKeyStoreEntry(address string, password string) (*types.Account, error) {
	// Create the full path for the file
	filename := filepath.Join("configs/keystore", address+".json")

	// Read the JSON file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore entry: %w", err)
	}

	// Parse the JSON
	var entry KeyStoreEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal keystore entry: %w", err)
	}

	// Verify the address
	if entry.Address != address {
		return nil, fmt.Errorf("address mismatch: expected %s, got %s", address, entry.Address)
	}

	// Decrypt the private key
	privateKey, err := km.DecryptPrivateKey(entry.EncryptedKey, password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	// Create an account from the private key
	account, err := types.AccountFromBytes(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create account from private key: %w", err)
	}

	return &account, nil
}

// GetSolanaAddressFromPrivateKey returns the Solana address for a private key
func (km *KeyManager) GetSolanaAddressFromPrivateKey(privateKey []byte) (string, error) {
	account, err := types.AccountFromBytes(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create account from private key: %w", err)
	}
	return account.PublicKey.ToBase58(), nil
}

// deriveKey creates a 32-byte key from a password using SHA-256
func deriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}
