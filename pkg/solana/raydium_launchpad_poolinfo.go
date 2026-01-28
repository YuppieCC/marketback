package solana

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// LaunchpadPool represents the decoded launchpad pool data structure
type LaunchpadPool struct {
	Discriminator         uint64
	Epoch                 uint64
	Bump                  uint8
	Status                uint8
	MintDecimalsA         uint8
	MintDecimalsB         uint8
	MigrateType           uint8
	Supply                uint64
	TotalSellA            uint64
	VirtualA              uint64
	VirtualB              uint64
	RealA                 uint64
	RealB                 uint64
	TotalFundRaisingB     uint64
	ProtocolFee           uint64
	PlatformFee           uint64
	MigrateFee            uint64
	VestingSchedule       VestingSchedule
	ConfigId              solana.PublicKey
	PlatformId            solana.PublicKey
	MintA                 solana.PublicKey
	MintB                 solana.PublicKey
	VaultA                solana.PublicKey
	VaultB                solana.PublicKey
	Creator               solana.PublicKey
	MintProgramFlag       uint8
	Reserved              [63]uint8
}

// VestingSchedule represents the vesting schedule structure
type VestingSchedule struct {
	TotalLockedAmount    uint64
	CliffPeriod          uint64
	UnlockPeriod         uint64
	StartTime            uint64
	TotalAllocatedShare  uint64
}

// LaunchpadPoolInfo represents the parsed launch pool data for JSON output
type LaunchpadPoolInfo struct {
	Epoch                 string            `json:"epoch"`
	Bump                  uint8             `json:"bump"`
	Status                uint8             `json:"status"`
	MintDecimalsA         uint8             `json:"mintDecimalsA"`
	MintDecimalsB         uint8             `json:"mintDecimalsB"`
	MigrateType           uint8             `json:"migrateType"`
	Supply                string            `json:"supply"`
	TotalSellA            string            `json:"totalSellA"`
	VirtualA              string            `json:"virtualA"`
	VirtualB              string            `json:"virtualB"`
	RealA                 string            `json:"realA"`
	RealB                 string            `json:"realB"`
	TotalFundRaisingB     string            `json:"totalFundRaisingB"`
	ProtocolFee           string            `json:"protocolFee"`
	PlatformFee           string            `json:"platformFee"`
	MigrateFee            string            `json:"migrateFee"`
	VestingSchedule       VestingScheduleInfo `json:"vestingSchedule"`
	ConfigId              string            `json:"configId"`
	PlatformId            string            `json:"platformId"`
	MintA                 string            `json:"mintA"`
	MintB                 string            `json:"mintB"`
	VaultA                string            `json:"vaultA"`
	VaultB                string            `json:"vaultB"`
	Creator               string            `json:"creator"`
	MintProgramFlag       uint8             `json:"mintProgramFlag"`
}

// VestingScheduleInfo represents the parsed vesting schedule for JSON output
type VestingScheduleInfo struct {
	TotalLockedAmount    string `json:"totalLockedAmount"`
	CliffPeriod          string `json:"cliffPeriod"`
	UnlockPeriod         string `json:"unlockPeriod"`
	StartTime            string `json:"startTime"`
	TotalAllocatedShare  string `json:"totalAllocatedShare"`
}

// decodeLaunchpadPool decodes the raw account data into a LaunchpadPool struct
func decodeLaunchpadPool(data []byte) (*LaunchpadPool, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for LaunchpadPool")
	}

	pool := &LaunchpadPool{}
	reader := bytes.NewReader(data)

	// Read discriminator and basic fields
	if err := binary.Read(reader, binary.LittleEndian, &pool.Discriminator); err != nil {
		return nil, fmt.Errorf("failed to read discriminator: %w", err)
	}
	
	if err := binary.Read(reader, binary.LittleEndian, &pool.Epoch); err != nil {
		return nil, fmt.Errorf("failed to read epoch: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &pool.Bump); err != nil {
		return nil, fmt.Errorf("failed to read bump: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &pool.Status); err != nil {
		return nil, fmt.Errorf("failed to read status: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &pool.MintDecimalsA); err != nil {
		return nil, fmt.Errorf("failed to read mintDecimalsA: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &pool.MintDecimalsB); err != nil {
		return nil, fmt.Errorf("failed to read mintDecimalsB: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &pool.MigrateType); err != nil {
		return nil, fmt.Errorf("failed to read migrateType: %w", err)
	}

	// Read u64 fields
	fields := []*uint64{
		&pool.Supply, &pool.TotalSellA, &pool.VirtualA, &pool.VirtualB,
		&pool.RealA, &pool.RealB, &pool.TotalFundRaisingB,
		&pool.ProtocolFee, &pool.PlatformFee, &pool.MigrateFee,
	}

	for _, field := range fields {
		if err := binary.Read(reader, binary.LittleEndian, field); err != nil {
			return nil, fmt.Errorf("failed to read u64 field: %w", err)
		}
	}

	// Read vesting schedule
	vestingFields := []*uint64{
		&pool.VestingSchedule.TotalLockedAmount,
		&pool.VestingSchedule.CliffPeriod,
		&pool.VestingSchedule.UnlockPeriod,
		&pool.VestingSchedule.StartTime,
		&pool.VestingSchedule.TotalAllocatedShare,
	}

	for _, field := range vestingFields {
		if err := binary.Read(reader, binary.LittleEndian, field); err != nil {
			return nil, fmt.Errorf("failed to read vesting field: %w", err)
		}
	}

	// Read PublicKey fields (32 bytes each)
	publicKeys := []*solana.PublicKey{
		&pool.ConfigId, &pool.PlatformId, &pool.MintA, &pool.MintB,
		&pool.VaultA, &pool.VaultB, &pool.Creator,
	}

	for _, pk := range publicKeys {
		var keyBytes [32]byte
		if err := binary.Read(reader, binary.LittleEndian, &keyBytes); err != nil {
			return nil, fmt.Errorf("failed to read public key: %w", err)
		}
		*pk = solana.PublicKeyFromBytes(keyBytes[:])
	}

	// Read final fields
	if err := binary.Read(reader, binary.LittleEndian, &pool.MintProgramFlag); err != nil {
		return nil, fmt.Errorf("failed to read mintProgramFlag: %w", err)
	}

	if err := binary.Read(reader, binary.LittleEndian, &pool.Reserved); err != nil {
		return nil, fmt.Errorf("failed to read reserved: %w", err)
	}

	return pool, nil
}

// parseLaunchpadPoolInfo converts raw LaunchpadPool to LaunchpadPoolInfo for JSON output
func parseLaunchpadPoolInfo(pool *LaunchpadPool) *LaunchpadPoolInfo {
	return &LaunchpadPoolInfo{
		Epoch:                 fmt.Sprintf("%d", pool.Epoch),
		Bump:                  pool.Bump,
		Status:                pool.Status,
		MintDecimalsA:         pool.MintDecimalsA,
		MintDecimalsB:         pool.MintDecimalsB,
		MigrateType:           pool.MigrateType,
		Supply:                fmt.Sprintf("%d", pool.Supply),
		TotalSellA:            fmt.Sprintf("%d", pool.TotalSellA),
		VirtualA:              fmt.Sprintf("%d", pool.VirtualA),
		VirtualB:              fmt.Sprintf("%d", pool.VirtualB),
		RealA:                 fmt.Sprintf("%d", pool.RealA),
		RealB:                 fmt.Sprintf("%d", pool.RealB),
		TotalFundRaisingB:     fmt.Sprintf("%d", pool.TotalFundRaisingB),
		ProtocolFee:           fmt.Sprintf("%d", pool.ProtocolFee),
		PlatformFee:           fmt.Sprintf("%d", pool.PlatformFee),
		MigrateFee:            fmt.Sprintf("%d", pool.MigrateFee),
		VestingSchedule: VestingScheduleInfo{
			TotalLockedAmount:    fmt.Sprintf("%d", pool.VestingSchedule.TotalLockedAmount),
			CliffPeriod:          fmt.Sprintf("%d", pool.VestingSchedule.CliffPeriod),
			UnlockPeriod:         fmt.Sprintf("%d", pool.VestingSchedule.UnlockPeriod),
			StartTime:            fmt.Sprintf("%d", pool.VestingSchedule.StartTime),
			TotalAllocatedShare:  fmt.Sprintf("%d", pool.VestingSchedule.TotalAllocatedShare),
		},
		ConfigId:              pool.ConfigId.String(),
		PlatformId:            pool.PlatformId.String(),
		MintA:                 pool.MintA.String(),
		MintB:                 pool.MintB.String(),
		VaultA:                pool.VaultA.String(),
		VaultB:                pool.VaultB.String(),
		Creator:               pool.Creator.String(),
		MintProgramFlag:       pool.MintProgramFlag,
	}
}

// GetLaunchpadPoolInfo fetches and decodes launch pool data from Solana
func GetLaunchpadPoolInfo(rpcEndpoint string, poolId solana.PublicKey) (*LaunchpadPoolInfo, error) {
	// Create RPC client
	client := rpc.New(rpcEndpoint)
	
	// Get account info
	accountInfo, err := client.GetAccountInfo(
		context.Background(),
		poolId,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	if accountInfo == nil || accountInfo.Value == nil {
		return nil, fmt.Errorf("account not found or has no data")
	}

	// Decode the account data
	pool, err := decodeLaunchpadPool(accountInfo.Value.Data.GetBinary())
	if err != nil {
		return nil, fmt.Errorf("failed to decode launch pool data: %w", err)
	}

	// Parse to readable format
	return parseLaunchpadPoolInfo(pool), nil
}

// Example function that mimics the TypeScript getLaunchpadPoolInfo
func getLaunchpadPoolInfoExample() (*LaunchpadPoolInfo, error) {
	// Example mint addresses from the TypeScript code
	mintA := solana.MustPublicKeyFromBase58("9AQxgnDbNRBcmFyMWQ7xdtVWLeR85aQDfpADKpd7bonk")
	mintB := solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112") // WSOL
	
	// Get pool ID using our existing function
	poolIds, err := getLaunchpadAndCpmmId(mintA, mintB)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool IDs: %w", err)
	}

	// RPC endpoint (you may want to make this configurable)
	rpcEndpoint := "https://red-wider-scion.solana-mainnet.quiknode.pro/7d63bea9a0a2d0a3664671d551a2d3565bef43b6/"
	
	// Get launch pool data
	return GetLaunchpadPoolInfo(rpcEndpoint, poolIds.LaunchpadPoolId)
}

// Helper function to convert uint64 to big.Int for larger numbers
func uint64ToBigInt(val uint64) *big.Int {
	return new(big.Int).SetUint64(val)
}
