package solana

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// PumpSwap (PumpAMM) program constants
var (
	PumpAmmProgramID = solana.MustPublicKeyFromBase58("pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA")
	WSolMint        = solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")
)

// PumpSwap PDA seeds
var (
	SeedGlobalConfig              = []byte("global_config")
	SeedPool                      = []byte("pool")
	SeedPoolLpMint               = []byte("pool_lp_mint")
	SeedEventAuthorityPumpSwap   = []byte("__event_authority")
	SeedCreatorVaultPumpSwap     = []byte("creator_vault")
	SeedGlobalVolumeAccumulatorPumpSwap = []byte("global_volume_accumulator")
	SeedUserVolumeAccumulatorPumpSwap   = []byte("user_volume_accumulator")
)

// PumpSwapPDAInfo contains all PumpSwap PDA information
type PumpSwapPDAInfo struct {
	GlobalConfig              PDAResult `json:"globalConfig"`
	EventAuthority           PDAResult `json:"eventAuthority"`
	Pool                     PDAResult `json:"pool"`
	PoolLpMint              PDAResult `json:"poolLpMint"`
	CoinCreatorVaultAuthority PDAResult `json:"coinCreatorVaultAuthority"`
	GlobalVolumeAccumulator  PDAResult `json:"globalVolumeAccumulator"`
	UserVolumeAccumulator    PDAResult `json:"userVolumeAccumulator"`
	BondingCurve             PDAResult `json:"bondingCurve"`
	Metadata                 PDAResult `json:"metadata"`
	
	// Token Accounts
	UserBaseTokenAccount   solana.PublicKey `json:"userBaseTokenAccount"`
	UserQuoteTokenAccount  solana.PublicKey `json:"userQuoteTokenAccount"`
	UserPoolTokenAccount   solana.PublicKey `json:"userPoolTokenAccount"`
	PoolBaseTokenAccount   solana.PublicKey `json:"poolBaseTokenAccount"`
	PoolQuoteTokenAccount  solana.PublicKey `json:"poolQuoteTokenAccount"`
	CoinCreatorVaultATA    solana.PublicKey `json:"coinCreatorVaultATA"`
}

// PoolParams represents pool parameters
type PoolParams struct {
	Index     uint16 `json:"index"`
	Creator   solana.PublicKey `json:"creator"`
	BaseMint  solana.PublicKey `json:"baseMint"`
	QuoteMint solana.PublicKey `json:"quoteMint"`
}

// GetGlobalConfigPDA gets global config PDA
func GetGlobalConfigPDA() (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedGlobalConfig},
		PumpAmmProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find global config PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetEventAuthorityPumpSwapPDA gets event authority PDA for PumpSwap
func GetEventAuthorityPumpSwapPDA() (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedEventAuthorityPumpSwap},
		PumpAmmProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find event authority PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetPoolPDA gets pool PDA
func GetPoolPDA(index uint16, creator, baseMint, quoteMint solana.PublicKey) (PDAResult, error) {
	// Create index buffer (u16, little endian)
	indexBuffer := make([]byte, 2)
	binary.LittleEndian.PutUint16(indexBuffer, index)
	
	seeds := [][]byte{
		SeedPool,
		indexBuffer,
		creator[:],
		baseMint[:],
		quoteMint[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PumpAmmProgramID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find pool PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetPoolLpMintPDA gets pool LP mint PDA
func GetPoolLpMintPDA(pool solana.PublicKey) (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedPoolLpMint, pool[:]},
		PumpAmmProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find pool LP mint PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetCoinCreatorVaultAuthorityPDA gets coin creator vault authority PDA
func GetCoinCreatorVaultAuthorityPDA(coinCreator solana.PublicKey) (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedCreatorVaultPumpSwap, coinCreator[:]},
		PumpAmmProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find coin creator vault authority PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetGlobalVolumeAccumulatorPumpSwapPDA gets global volume accumulator PDA for PumpSwap
func GetGlobalVolumeAccumulatorPumpSwapPDA() (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedGlobalVolumeAccumulatorPumpSwap},
		PumpAmmProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find global volume accumulator PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetUserVolumeAccumulatorPumpSwapPDA gets user volume accumulator PDA for PumpSwap
func GetUserVolumeAccumulatorPumpSwapPDA(user solana.PublicKey) (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedUserVolumeAccumulatorPumpSwap, user[:]},
		PumpAmmProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find user volume accumulator PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetAssociatedTokenAddress gets associated token account address
func GetAssociatedTokenAddress(mint, owner solana.PublicKey) (solana.PublicKey, error) {
	seeds := [][]byte{
		owner[:],
		solana.TokenProgramID[:],
		mint[:],
	}
	
	address, _, err := solana.FindProgramAddress(seeds, solana.SPLAssociatedTokenAccountProgramID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to find associated token address: %w", err)
	}
	
	return address, nil
}

// GetAllPumpSwapPDAs gets all PumpSwap related PDAs and token accounts
func GetAllPumpSwapPDAs(user solana.PublicKey, creatorPubkey solana.PublicKey, baseMintPubkey solana.PublicKey, coinCreatorPubkey solana.PublicKey) (*PumpSwapPDAInfo, error) {
	info := &PumpSwapPDAInfo{}
	var err error
	
	// For PumpSwap, we typically use WSOL as quote mint and index 0
	quoteMint := WSolMint
	poolParams := PoolParams{
		Index:     0,
		Creator:   creatorPubkey,
		BaseMint:  baseMintPubkey,
		QuoteMint: quoteMint,
	}
	
	// Get global config PDA
	info.GlobalConfig, err = GetGlobalConfigPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global config PDA: %w", err)
	}
	
	// Get event authority PDA
	info.EventAuthority, err = GetEventAuthorityPumpSwapPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get event authority PDA: %w", err)
	}
	
	// Get pool PDA
	info.Pool, err = GetPoolPDA(poolParams.Index, poolParams.Creator, poolParams.BaseMint, poolParams.QuoteMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool PDA: %w", err)
	}
	
	// Get LP mint PDA
	info.PoolLpMint, err = GetPoolLpMintPDA(info.Pool.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool LP mint PDA: %w", err)
	}
	
	// Get coin creator vault authority PDA
	info.CoinCreatorVaultAuthority, err = GetCoinCreatorVaultAuthorityPDA(coinCreatorPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to get coin creator vault authority PDA: %w", err)
	}
	
	// Get global volume accumulator PDA
	info.GlobalVolumeAccumulator, err = GetGlobalVolumeAccumulatorPumpSwapPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global volume accumulator PDA: %w", err)
	}
	
	// Get user volume accumulator PDA
	info.UserVolumeAccumulator, err = GetUserVolumeAccumulatorPumpSwapPDA(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user volume accumulator PDA: %w", err)
	}
	
	// Get bonding curve PDA (from PumpFun program)
	bondingCurveAddr, bondingCurveBump, err := GetBondingCurvePDA(poolParams.BaseMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get bonding curve PDA: %w", err)
	}
	info.BondingCurve = PDAResult{Address: bondingCurveAddr, Bump: bondingCurveBump}
	
	// Get metadata PDA
	metadataAddr, metadataBump, err := GetMetadataPDA(poolParams.BaseMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata PDA: %w", err)
	}
	info.Metadata = PDAResult{Address: metadataAddr, Bump: metadataBump}
	
	// Get user token accounts
	info.UserBaseTokenAccount, err = GetAssociatedTokenAddress(poolParams.BaseMint, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user base token account: %w", err)
	}
	
	info.UserQuoteTokenAccount, err = GetAssociatedTokenAddress(poolParams.QuoteMint, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user quote token account: %w", err)
	}
	
	info.UserPoolTokenAccount, err = GetAssociatedTokenAddress(info.PoolLpMint.Address, user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user pool token account: %w", err)
	}
	
	// Get pool token accounts
	info.PoolBaseTokenAccount, err = GetAssociatedTokenAddress(poolParams.BaseMint, info.Pool.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool base token account: %w", err)
	}
	
	info.PoolQuoteTokenAccount, err = GetAssociatedTokenAddress(poolParams.QuoteMint, info.Pool.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool quote token account: %w", err)
	}
	
	// Get coin creator vault ATA
	info.CoinCreatorVaultATA, err = GetAssociatedTokenAddress(poolParams.QuoteMint, info.CoinCreatorVaultAuthority.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get coin creator vault ATA: %w", err)
	}
	
	return info, nil
}

// PumpfunAmmPoolConfigByMint represents the complete pool configuration data for a mint
type PumpfunAmmPoolConfigByMint struct {
	PoolAddress           string `json:"pool_address"`
	PoolBump              uint8  `json:"pool_bump"`
	Index                 uint16 `json:"index"`
	Creator               string `json:"creator"`
	BaseMint              string `json:"base_mint"`
	QuoteMint             string `json:"quote_mint"`
	LpMint                string `json:"lp_mint"`
	PoolBaseTokenAccount  string `json:"pool_base_token_account"`
	PoolQuoteTokenAccount string `json:"pool_quote_token_account"`
	LpSupply              uint64 `json:"lp_supply"`
	CoinCreator           string `json:"coin_creator"`
	Status                string `json:"status"`
}

// GetPumpfunAmmPoolConfigByMint generates complete pool configuration data for a given mint and creator
func GetPumpfunAmmPoolConfigByMint(mintPubkey, creator, coinCreator string) (*PumpfunAmmPoolConfigByMint, error) {
	// Parse the BaseMint public key
	baseMint, err := solana.PublicKeyFromBase58(mintPubkey)
	if err != nil {
		return nil, fmt.Errorf("invalid mint public key: %w", err)
	}

	// Parse and validate Creator public key
	creatorPubkey, err := solana.PublicKeyFromBase58(creator)
	if err != nil {
		return nil, fmt.Errorf("invalid creator public key: %w", err)
	}

	// Validate CoinCreator public key
	_, err = solana.PublicKeyFromBase58(coinCreator)
	if err != nil {
		return nil, fmt.Errorf("invalid coin creator public key: %w", err)
	}

	// Set default values
	index := uint16(0)
	quoteMint := solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112") // WSOL

	// Calculate Pool Address using GetPoolPDA
	poolResult, err := GetPoolPDA(index, creatorPubkey, baseMint, quoteMint)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pool address: %w", err)
	}

	// Generate LP Mint PDA
	lpMintResult, err := GetPoolLpMintPDA(poolResult.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to generate LP mint address: %w", err)
	}

	// Generate Pool Token Accounts (Associated Token Accounts)
	poolBaseTokenAccount, err := GetAssociatedTokenAddress(baseMint, poolResult.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pool base token account: %w", err)
	}

	poolQuoteTokenAccount, err := GetAssociatedTokenAddress(quoteMint, poolResult.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pool quote token account: %w", err)
	}

	// Create the pool config data
	config := &PumpfunAmmPoolConfigByMint{
		PoolAddress:           poolResult.Address.String(),
		PoolBump:              poolResult.Bump,
		Index:                 index,
		Creator:               creator,
		BaseMint:              mintPubkey,
		QuoteMint:             quoteMint.String(),
		LpMint:                lpMintResult.Address.String(),
		PoolBaseTokenAccount:  poolBaseTokenAccount.String(),
		PoolQuoteTokenAccount: poolQuoteTokenAccount.String(),
		LpSupply:              0, // Initial LP supply
		CoinCreator:           coinCreator,
		Status:                "active",
	}

	return config, nil
}