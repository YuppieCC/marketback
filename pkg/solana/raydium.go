package solana

import (
	"encoding/binary"
	"fmt"
	"errors"

	"github.com/gagliardetto/solana-go"
	"gorm.io/gorm"
	"marketcontrol/internal/models"
)

// Program IDs
var (
	CREATE_CPMM_POOL_PROGRAM = solana.MustPublicKeyFromBase58("CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C")
	LAUNCHPAD_PROGRAM        = solana.MustPublicKeyFromBase58("LanMV9sAd7wArD4vJFi2qDdfnVhFxYSUg6eADduJ3uj")
)

// Seeds for PDA derivation
var (
	// CPMM seeds
	AUTH_SEED        = []byte("vault_and_lp_mint_auth_seed")
	AMM_CONFIG_SEED  = []byte("amm_config")
	POOL_SEED        = []byte("pool")
	POOL_LP_MINT_SEED = []byte("pool_lp_mint")
	POOL_VAULT_SEED  = []byte("pool_vault")
	OBSERVATION_SEED = []byte("observation")
	
	// Launchpad seeds
	LAUNCHPAD_POOL_SEED       = []byte("pool")
	LAUNCHPAD_POOL_VAULT_SEED = []byte("pool_vault")
)

// PdaResult represents the result of PDA derivation
type PdaResult struct {
	PublicKey solana.PublicKey
	Nonce     uint8
}

// PoolIds represents the result of getLaunchpadAndCpmmId
type PoolIds struct {
	CpmmPoolId      solana.PublicKey
	LaunchpadPoolId solana.PublicKey
}

// CpmmPoolVaultResult represents the result of GetCpmmPoolVault
type CpmmPoolVaultResult struct {
	BaseVault  solana.PublicKey
	QuoteVault solana.PublicKey
}

// u16ToBytes converts a uint16 to little-endian bytes
func u16ToBytes(num uint16) []byte {
	bytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(bytes, num)
	return bytes
}

func getPdaVault(programId, poolId, mint solana.PublicKey) (PdaResult, error) {
	// Seeds array matches TypeScript: [POOL_VAULT_SEED, poolId.toBuffer(), mint.toBuffer()]
	seeds := [][]byte{
		POOL_VAULT_SEED,
		poolId.Bytes(),
		mint.Bytes(),
	}
	
	// Find program address using Solana's PDA derivation
	pda, nonce, err := solana.FindProgramAddress(seeds, programId)
	if err != nil {
		return PdaResult{}, fmt.Errorf("failed to find program address for vault PDA: %w", err)
	}
	
	return PdaResult{
		PublicKey: pda,
		Nonce:     nonce,
	}, nil
}

// getCpmmPdaAmmConfigId derives the AMM config PDA for CPMM (Go equivalent of TypeScript getCpmmPdaAmmConfigId)
// This function generates a Program Derived Address (PDA) for AMM configuration
// Parameters match the TypeScript version: programId, index (number)
// TypeScript: return findProgramAddress([AMM_CONFIG_SEED, u16ToBytes(index)], programId);
func getCpmmPdaAmmConfigId(programId solana.PublicKey, index uint16) (PdaResult, error) {
	// Seeds array matches TypeScript: [AMM_CONFIG_SEED, u16ToBytes(index)]
	seeds := [][]byte{
		AMM_CONFIG_SEED,
		u16ToBytes(index),
	}
	
	// Find program address using Solana's PDA derivation
	pda, nonce, err := solana.FindProgramAddress(seeds, programId)
	if err != nil {
		return PdaResult{}, fmt.Errorf("failed to find program address for AMM config PDA: %w", err)
	}
	
	return PdaResult{
		PublicKey: pda,
		Nonce:     nonce,
	}, nil
}

// getCpmmPdaPoolId derives the pool PDA for CPMM
func getCpmmPdaPoolId(programId, ammConfigId, mintA, mintB solana.PublicKey) (PdaResult, error) {
	seeds := [][]byte{
		POOL_SEED,
		ammConfigId.Bytes(),
		mintA.Bytes(),
		mintB.Bytes(),
	}
	
	pda, nonce, err := solana.FindProgramAddress(seeds, programId)
	if err != nil {
		return PdaResult{}, err
	}
	
	return PdaResult{
		PublicKey: pda,
		Nonce:     nonce,
	}, nil
}

// GetCpmmPoolVault gets the CPMM pool vault addresses (Go equivalent of TypeScript getCpmmPoolVault)
func GetCpmmPoolVault(poolId solana.PublicKey, mintA, mintB solana.PublicKey) (*CpmmPoolVaultResult, error) {
	// Validate mint addresses
	if mintA.IsZero() || mintB.IsZero() {
		return nil, fmt.Errorf("mint addresses cannot be zero")
	}
	
	// Get pool ID
	// poolId, err := getCpmmPoolId(mintA, mintB)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get pool ID: %w", err)
	// }
	
	// Get base vault (vault for mintA)
	baseVaultResult, err := getPdaVault(CREATE_CPMM_POOL_PROGRAM, poolId, mintA)
	if err != nil {
		return nil, fmt.Errorf("failed to get base vault: %w", err)
	}
	
	// Get quote vault (vault for mintB)
	quoteVaultResult, err := getPdaVault(CREATE_CPMM_POOL_PROGRAM, poolId, mintB)
	if err != nil {
		return nil, fmt.Errorf("failed to get quote vault: %w", err)
	}
	
	return &CpmmPoolVaultResult{
		BaseVault:  baseVaultResult.PublicKey,
		QuoteVault: quoteVaultResult.PublicKey,
	}, nil
}

func getPdaLpMint(programId, poolId solana.PublicKey) (PdaResult, error) {
	// Seeds array matches TypeScript: [POOL_LP_MINT_SEED, poolId.toBuffer()]
	seeds := [][]byte{
		POOL_LP_MINT_SEED,
		poolId.Bytes(),
	}
	
	// Find program address using Solana's PDA derivation
	pda, nonce, err := solana.FindProgramAddress(seeds, programId)
	if err != nil {
		return PdaResult{}, fmt.Errorf("failed to find program address for LP mint PDA: %w", err)
	}
	
	return PdaResult{
		PublicKey: pda,
		Nonce:     nonce,
	}, nil
}

// GetPdaLpMint is the public exported version of getPdaLpMint
// This is the Go equivalent of the exported getPdaLpMint function in TypeScript
func GetPdaLpMint(programId, poolId solana.PublicKey) (*PdaResult, error) {
	result, err := getPdaLpMint(programId, poolId)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetCpmmPdaAmmConfigId is the public exported version of getCpmmPdaAmmConfigId
func GetCpmmPdaAmmConfigId(programId solana.PublicKey, index uint16) (*PdaResult, error) {
	result, err := getCpmmPdaAmmConfigId(programId, index)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// getPdaLaunchpadPoolId derives the pool PDA for Launchpad
func getPdaLaunchpadPoolId(programId, mintA, mintB solana.PublicKey) (PdaResult, error) {
	seeds := [][]byte{
		LAUNCHPAD_POOL_SEED,
		mintA.Bytes(),
		mintB.Bytes(),
	}
	
	pda, nonce, err := solana.FindProgramAddress(seeds, programId)
	if err != nil {
		return PdaResult{}, err
	}
	
	return PdaResult{
		PublicKey: pda,
		Nonce:     nonce,
	}, nil
}

// getLaunchpadAndCpmmId is the Go equivalent of the TypeScript function
func getLaunchpadAndCpmmId(mintA, mintB solana.PublicKey) (PoolIds, error) {
	// Get AMM config ID
	configResult, err := getCpmmPdaAmmConfigId(CREATE_CPMM_POOL_PROGRAM, 0)
	if err != nil {
		return PoolIds{}, fmt.Errorf("failed to get AMM config ID: %w", err)
	}
	
	// Get CPMM pool ID
	cpmmResult, err := getCpmmPdaPoolId(
		CREATE_CPMM_POOL_PROGRAM,
		configResult.PublicKey,
		mintB, // Note: mintB and mintA are swapped in the original TypeScript
		mintA,
	)
	if err != nil {
		return PoolIds{}, fmt.Errorf("failed to get CPMM pool ID: %w", err)
	}
	
	// Get Launchpad pool ID
	launchpadResult, err := getPdaLaunchpadPoolId(
		LAUNCHPAD_PROGRAM,
		mintA,
		mintB,
	)
	if err != nil {
		return PoolIds{}, fmt.Errorf("failed to get Launchpad pool ID: %w", err)
	}
	
	return PoolIds{
		CpmmPoolId:      cpmmResult.PublicKey,
		LaunchpadPoolId: launchpadResult.PublicKey,
	}, nil
}

// GetLaunchpadAndCpmmId is the exported version of getLaunchpadAndCpmmId
func GetLaunchpadAndCpmmId(mintA, mintB solana.PublicKey) (PoolIds, error) {
	return getLaunchpadAndCpmmId(mintA, mintB)
}

// CreateCpmmPoolConfig creates a RaydiumCpmmPoolConfig record in the database based on mintA and mintB
func CreateCpmmPoolConfig(db *gorm.DB, mintA, mintB solana.PublicKey) error {
	// Get pool IDs using getLaunchpadAndCpmmId
	poolIds, err := getLaunchpadAndCpmmId(mintA, mintB)
	if err != nil {
		return fmt.Errorf("failed to get pool IDs: %w", err)
	}

	// Check if pool_address already exists in the database
	var existingConfig models.RaydiumCpmmPoolConfig
	result := db.Where("pool_address = ?", poolIds.CpmmPoolId.String()).First(&existingConfig)
	if result.Error == nil {
		// Pool already exists, no need to create
		return fmt.Errorf("pool already exists with address: %s", poolIds.CpmmPoolId.String())
	}
	
	// If error is not "record not found", return the error
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing pool: %w", result.Error)
	}

	// Pool doesn't exist, proceed with creation
	// Get CPMM pool vault addresses
	vaultResult, err := GetCpmmPoolVault(poolIds.CpmmPoolId, mintA, mintB)
	if err != nil {
		return fmt.Errorf("failed to get CPMM pool vaults: %w", err)
	}

	// Get AMM Config ID
	ammConfigResult, err := GetCpmmPdaAmmConfigId(CREATE_CPMM_POOL_PROGRAM, 0)
	if err != nil {
		return fmt.Errorf("failed to get AMM config ID: %w", err)
	}

	// Get LP Mint
	lpMintResult, err := GetPdaLpMint(CREATE_CPMM_POOL_PROGRAM, poolIds.CpmmPoolId)
	if err != nil {
		return fmt.Errorf("failed to get LP mint: %w", err)
	}

	// Create the RaydiumCpmmPoolConfig record
	config := models.RaydiumCpmmPoolConfig{
		Platform:        "raydium_cpmm",
		ProgramID:       CREATE_CPMM_POOL_PROGRAM.String(),
		PoolAddress:     poolIds.CpmmPoolId.String(),
		BaseIsWsol:      false,
		BaseMint:        mintA.String(),
		QuoteMint:       mintB.String(),
		BaseVault:       vaultResult.BaseVault.String(),
		QuoteVault:      vaultResult.QuoteVault.String(),
		FeeRate:         0.0025,
		ConfigID:        ammConfigResult.PublicKey.String(),
		ConfigIndex:     uint64(0),
		ProtocolFeeRate: 0.0001,
		TradeFeeRate:    0.0025,
		FundFeeRate:     0.0005,
		CreatePoolFee:   float64(1000000000),
		LpMint:          lpMintResult.PublicKey.String(),
		BurnPercent:     float64(100),
		Status:          "active",
	}

	// Save to database
	if err := db.Create(&config).Error; err != nil {
		return fmt.Errorf("failed to create RaydiumCpmmPoolConfig: %w", err)
	}

	return nil
}
