package solana

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// LaunchpadConfig represents the decoded launchpad config data structure
type LaunchpadConfig struct {
	Discriminator         uint64
	Epoch                 uint64
	CurveType             uint8
	Index                 uint16
	MigrateFee            uint64
	TradeFeeRate          uint64
	MaxShareFeeRate       uint64
	MinSupplyA            uint64
	MaxLockRate           uint64
	MinSellRateA          uint64
	MinMigrateRateA       uint64
	MinFundRaisingB       uint64
	MintB                 solana.PublicKey
	ProtocolFeeOwner      solana.PublicKey
	MigrateFeeOwner       solana.PublicKey
	MigrateToAmmWallet    solana.PublicKey
	MigrateToCpmmWallet   solana.PublicKey
	Reserved              [16]uint64
}

// LaunchpadConfigInfo represents the parsed launchpad config data for JSON output
type LaunchpadConfigInfo struct {
	Epoch                 int     `json:"epoch"`
	CurveType             uint8   `json:"curveType"`
	Index                 uint16  `json:"index"`
	MigrateFee            int     `json:"migrateFee"`
	TradeFeeRate          float64 `json:"tradeFeeRate"`
	MaxShareFeeRate       float64 `json:"maxShareFeeRate"`
	MinSupplyA            float64 `json:"minSupplyA"`
	MaxLockRate           float64 `json:"maxLockRate"`
	MinSellRateA          float64 `json:"minSellRateA"`
	MinMigrateRateA       float64 `json:"minMigrateRateA"`
	MinFundRaisingB       float64 `json:"minFundRaisingB"`
	MintB                 string  `json:"mintB"`
	ProtocolFeeOwner      string  `json:"protocolFeeOwner"`
	MigrateFeeOwner       string  `json:"migrateFeeOwner"`
	MigrateToAmmWallet    string  `json:"migrateToAmmWallet"`
	MigrateToCpmmWallet   string  `json:"migrateToCpmmWallet"`
}

// Helper functions for parsing rates and amounts (matching TypeScript logic)
func parseRate(value uint64, denominator float64) float64 {
	return float64(value) / denominator
}

func parseLargeAmount(value uint64, decimals int) float64 {
	return float64(value) / float64(uint64(1)<<uint(decimals*10/3)) // approximation of 10^decimals
}

// More precise version for common decimal places
func parseLargeAmountPrecise(value uint64, decimals int) float64 {
	divisor := float64(1)
	for i := 0; i < decimals; i++ {
		divisor *= 10
	}
	return float64(value) / divisor
}

// decodeLaunchpadConfig decodes the raw account data into a LaunchpadConfig struct
func decodeLaunchpadConfig(data []byte) (*LaunchpadConfig, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for LaunchpadConfig")
	}

	config := &LaunchpadConfig{}
	reader := bytes.NewReader(data)

	// Read discriminator and basic fields
	if err := binary.Read(reader, binary.LittleEndian, &config.Discriminator); err != nil {
		return nil, fmt.Errorf("failed to read discriminator: %w", err)
	}
	
	if err := binary.Read(reader, binary.LittleEndian, &config.Epoch); err != nil {
		return nil, fmt.Errorf("failed to read epoch: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &config.CurveType); err != nil {
		return nil, fmt.Errorf("failed to read curveType: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &config.Index); err != nil {
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	// Read u64 fields
	fields := []*uint64{
		&config.MigrateFee, &config.TradeFeeRate, &config.MaxShareFeeRate,
		&config.MinSupplyA, &config.MaxLockRate, &config.MinSellRateA,
		&config.MinMigrateRateA, &config.MinFundRaisingB,
	}

	for _, field := range fields {
		if err := binary.Read(reader, binary.LittleEndian, field); err != nil {
			return nil, fmt.Errorf("failed to read u64 field: %w", err)
		}
	}

	// Read PublicKey fields (32 bytes each)
	publicKeys := []*solana.PublicKey{
		&config.MintB, &config.ProtocolFeeOwner, &config.MigrateFeeOwner,
		&config.MigrateToAmmWallet, &config.MigrateToCpmmWallet,
	}

	for _, pk := range publicKeys {
		var keyBytes [32]byte
		if err := binary.Read(reader, binary.LittleEndian, &keyBytes); err != nil {
			return nil, fmt.Errorf("failed to read public key: %w", err)
		}
		*pk = solana.PublicKeyFromBytes(keyBytes[:])
	}

	// Read reserved fields (16 u64 values)
	if err := binary.Read(reader, binary.LittleEndian, &config.Reserved); err != nil {
		return nil, fmt.Errorf("failed to read reserved: %w", err)
	}

	return config, nil
}

// parseLaunchpadPoolConfig converts raw LaunchpadConfig to LaunchpadConfigInfo for JSON output
func parseLaunchpadPoolConfig(config *LaunchpadConfig) *LaunchpadConfigInfo {
	return &LaunchpadConfigInfo{
		Epoch:                 int(config.Epoch),
		CurveType:             config.CurveType,
		Index:                 config.Index,
		MigrateFee:            int(config.MigrateFee),
		TradeFeeRate:          parseRate(config.TradeFeeRate, 10000),         // e.g. 0.0025
		MaxShareFeeRate:       parseRate(config.MaxShareFeeRate, 10000),     // e.g. 1.0
		MinSupplyA:            parseLargeAmountPrecise(config.MinSupplyA, 6), // 10_000_000 => 10 tokens with 6 decimals
		MaxLockRate:           parseRate(config.MaxLockRate, 100000),         // e.g. 3.0%
		MinSellRateA:          parseRate(config.MinSellRateA, 100000),        // e.g. 2.0%
		MinMigrateRateA:       parseRate(config.MinMigrateRateA, 100000),
		MinFundRaisingB:       parseLargeAmountPrecise(config.MinFundRaisingB, 9), // e.g. 30.0 USDC
		MintB:                 config.MintB.String(),
		ProtocolFeeOwner:      config.ProtocolFeeOwner.String(),
		MigrateFeeOwner:       config.MigrateFeeOwner.String(),
		MigrateToAmmWallet:    config.MigrateToAmmWallet.String(),
		MigrateToCpmmWallet:   config.MigrateToCpmmWallet.String(),
	}
}

// GetLaunchpadPoolConfig fetches and decodes launchpad pool config data from Solana
func GetLaunchpadPoolConfig(rpcEndpoint string, configId solana.PublicKey) (*LaunchpadConfigInfo, error) {
	// Create RPC client
	client := rpc.New(rpcEndpoint)
	
	// Get account info
	accountInfo, err := client.GetAccountInfo(
		context.Background(),
		configId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	if accountInfo == nil || accountInfo.Value == nil {
		return nil, fmt.Errorf("account not found or has no data")
	}

	// Decode the account data
	config, err := decodeLaunchpadConfig(accountInfo.Value.Data.GetBinary())
	if err != nil {
		return nil, fmt.Errorf("failed to decode launchpad config data: %w", err)
	}

	// Parse to readable format
	return parseLaunchpadPoolConfig(config), nil
}

// Example function that mimics the TypeScript getLaunchpadPoolConfig
func getLaunchpadPoolConfigExample(configIdStr string) (*LaunchpadConfigInfo, error) {
	// Parse the config ID string
	configId, err := solana.PublicKeyFromBase58(configIdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid config ID: %w", err)
	}

	// RPC endpoint (you may want to make this configurable)
	rpcEndpoint := "https://red-wider-scion.solana-mainnet.quiknode.pro/7d63bea9a0a2d0a3664671d551a2d3565bef43b6/"
	
	// Get launchpad config data
	return GetLaunchpadPoolConfig(rpcEndpoint, configId)
}
