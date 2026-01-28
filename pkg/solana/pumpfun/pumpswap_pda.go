package pumpfun

import (
	"encoding/binary"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// PumpSwap (PumpAMM) 程序常量
var (
	// PumpAMM 程序地址
	PUMP_AMM_PROGRAM_ID = solana.MustPublicKeyFromBase58("pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA")
	
	// PumpFun 程序地址 (PumpSwap 版本)
	PUMP_FUN_PROGRAM_ID_PUMPSWAP = solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P")
	
	// WSOL Mint 地址
	WSOL_MINT = solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")
	
	// Metaplex Token Metadata 程序地址 (PumpSwap 版本)
	MPL_TOKEN_METADATA_PROGRAM_ID_PUMPSWAP = solana.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")
	
	// 固定的协议费接收者地址
	PROTOCOL_FEE_RECIPIENT = solana.MustPublicKeyFromBase58("9rPYyANsfQZw3DnDmKE3YCQF5E8oD89UXoHn9JFEhJUz")
	PROTOCOL_FEE_RECIPIENT_TOKEN_ACCOUNT = solana.MustPublicKeyFromBase58("Bvtgim23rfocUzxVX9j9QFxTbBnH8JZxnaGLCEkXvjKS")
	
	// 管理员地址
	ADMIN_ADDRESS = solana.MustPublicKeyFromBase58("8LWu7QM2dGR1G8nKDHthckea57bkCzXyBTAKPJUBDHo8")
	
	// Token 程序地址
	TOKEN_PROGRAM_ID = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	TOKEN_2022_PROGRAM_ID = solana.MustPublicKeyFromBase58("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")
	ASSOCIATED_TOKEN_PROGRAM_ID = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
)

// PDA 种子常量 (PumpSwap 特有的)
var (
	SEED_GLOBAL_CONFIG_PUMPSWAP              = []byte("global_config")
	SEED_POOL_PUMPSWAP                      = []byte("pool")
	SEED_POOL_LP_MINT_PUMPSWAP              = []byte("pool_lp_mint")
	SEED_EVENT_AUTHORITY_PUMPSWAP           = []byte("__event_authority")
	SEED_CREATOR_VAULT_PUMPSWAP             = []byte("creator_vault")
	SEED_GLOBAL_VOLUME_ACCUMULATOR_PUMPSWAP = []byte("global_volume_accumulator")
	SEED_USER_VOLUME_ACCUMULATOR_PUMPSWAP   = []byte("user_volume_accumulator")
	SEED_BONDING_CURVE_PUMPSWAP             = []byte("bonding-curve")
	SEED_METADATA_PUMPSWAP                  = []byte("metadata")
)

// PumpSwapPDAResult 表示 PDA 计算结果
type PumpSwapPDAResult struct {
	Address solana.PublicKey
	Bump    uint8
}

// GetGlobalConfigPDA 获取全局配置 PDA  
func GetGlobalConfigPDA() (PumpSwapPDAResult, error) {
	seeds := [][]byte{SEED_GLOBAL_CONFIG_PUMPSWAP}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_AMM_PROGRAM_ID)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find global config PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetEventAuthorityPumpSwapPDA 获取事件权限 PDA (PumpSwap 版本)
func GetEventAuthorityPumpSwapPDA() (PumpSwapPDAResult, error) {
	seeds := [][]byte{SEED_EVENT_AUTHORITY_PUMPSWAP}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_AMM_PROGRAM_ID)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find event authority PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetPoolPDA 获取池子 PDA
func GetPoolPDA(index uint16, creator, baseMint, quoteMint solana.PublicKey) (PumpSwapPDAResult, error) {
	// 创建 index buffer (u16, little endian)
	indexBuffer := make([]byte, 2)
	binary.LittleEndian.PutUint16(indexBuffer, index)
	
	seeds := [][]byte{
		SEED_POOL_PUMPSWAP,
		indexBuffer,
		creator[:],
		baseMint[:],
		quoteMint[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_AMM_PROGRAM_ID)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find pool PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetPoolLpMintPDA 获取 LP 代币铸造 PDA
func GetPoolLpMintPDA(pool solana.PublicKey) (PumpSwapPDAResult, error) {
	seeds := [][]byte{
		SEED_POOL_LP_MINT_PUMPSWAP,
		pool[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_AMM_PROGRAM_ID)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find pool LP mint PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetCoinCreatorVaultAuthorityPDA 获取币创建者保险库权限 PDA
func GetCoinCreatorVaultAuthorityPDA(coinCreator solana.PublicKey) (PumpSwapPDAResult, error) {
	seeds := [][]byte{
		SEED_CREATOR_VAULT_PUMPSWAP,
		coinCreator[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_AMM_PROGRAM_ID)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find coin creator vault authority PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetGlobalVolumeAccumulatorPumpSwapPDA 获取全局交易量累加器 PDA
func GetGlobalVolumeAccumulatorPumpSwapPDA() (PumpSwapPDAResult, error) {
	seeds := [][]byte{SEED_GLOBAL_VOLUME_ACCUMULATOR_PUMPSWAP}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_AMM_PROGRAM_ID)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find global volume accumulator PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetUserVolumeAccumulatorPumpSwapPDA 获取用户交易量累加器 PDA
func GetUserVolumeAccumulatorPumpSwapPDA(user solana.PublicKey) (PumpSwapPDAResult, error) {
	seeds := [][]byte{
		SEED_USER_VOLUME_ACCUMULATOR_PUMPSWAP,
		user[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_AMM_PROGRAM_ID)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find user volume accumulator PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetBondingCurvePumpSwapPDA 获取联合曲线 PDA (来自 Pump Fun 程序)
func GetBondingCurvePumpSwapPDA(mint solana.PublicKey) (PumpSwapPDAResult, error) {
	seeds := [][]byte{
		SEED_BONDING_CURVE_PUMPSWAP,
		mint[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID_PUMPSWAP)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find bonding curve PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetMetadataPumpSwapPDA 获取代币元数据 PDA
func GetMetadataPumpSwapPDA(mint solana.PublicKey) (PumpSwapPDAResult, error) {
	seeds := [][]byte{
		SEED_METADATA_PUMPSWAP,
		MPL_TOKEN_METADATA_PROGRAM_ID_PUMPSWAP[:],
		mint[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, MPL_TOKEN_METADATA_PROGRAM_ID_PUMPSWAP)
	if err != nil {
		return PumpSwapPDAResult{}, fmt.Errorf("failed to find metadata PDA: %w", err)
	}
	
	return PumpSwapPDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetAssociatedTokenAddress 获取关联代币账户地址 (ATA)
func GetAssociatedTokenAddress(mint, owner solana.PublicKey, allowOwnerOffCurve bool) (solana.PublicKey, error) {
	var programID solana.PublicKey
	
	// 根据 allowOwnerOffCurve 选择程序ID
	if allowOwnerOffCurve {
		programID = TOKEN_PROGRAM_ID
	} else {
		programID = TOKEN_PROGRAM_ID
	}
	
	seeds := [][]byte{
		owner[:],
		programID[:],
		mint[:],
	}
	
	address, _, err := solana.FindProgramAddress(seeds, ASSOCIATED_TOKEN_PROGRAM_ID)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("failed to find associated token address: %w", err)
	}
	
	return address, nil
}

// GetPoolTokenAccount 获取池子代币账户地址
func GetPoolTokenAccount(pool, mint solana.PublicKey) (solana.PublicKey, error) {
	return GetAssociatedTokenAddress(mint, pool, true)
}

// PumpSwapPDAInfo 包含所有 PDA 信息的结构体
type PumpSwapPDAInfo struct {
	GlobalConfig              PumpSwapPDAResult
	EventAuthority           PumpSwapPDAResult
	Pool                     PumpSwapPDAResult
	PoolLpMint              PumpSwapPDAResult
	CoinCreatorVaultAuthority PumpSwapPDAResult
	GlobalVolumeAccumulator  PumpSwapPDAResult
	UserVolumeAccumulator    PumpSwapPDAResult
	BondingCurve             PumpSwapPDAResult
	Metadata                 PumpSwapPDAResult
	
	// Token Accounts
	UserBaseTokenAccount   solana.PublicKey
	UserQuoteTokenAccount  solana.PublicKey
	UserPoolTokenAccount   solana.PublicKey
	PoolBaseTokenAccount   solana.PublicKey
	PoolQuoteTokenAccount  solana.PublicKey
	CoinCreatorVaultATA    solana.PublicKey
}

// PoolParams 池子参数
type PoolParams struct {
	Index     uint16
	Creator   solana.PublicKey
	BaseMint  solana.PublicKey
	QuoteMint solana.PublicKey
}

// GetAllPumpSwapPDAs 获取指定用户、池子和代币的所有相关 PDA 和代币账户
func GetAllPumpSwapPDAs(user solana.PublicKey, poolParams PoolParams, coinCreator solana.PublicKey) (*PumpSwapPDAInfo, error) {
	info := &PumpSwapPDAInfo{}
	var err error
	
	// 获取全局配置 PDA
	info.GlobalConfig, err = GetGlobalConfigPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global config PDA: %w", err)
	}
	
	// 获取事件权限 PDA
	info.EventAuthority, err = GetEventAuthorityPumpSwapPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get event authority PDA: %w", err)
	}
	
	// 获取池子 PDA
	info.Pool, err = GetPoolPDA(poolParams.Index, poolParams.Creator, poolParams.BaseMint, poolParams.QuoteMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool PDA: %w", err)
	}
	
	// 获取 LP 代币铸造 PDA
	info.PoolLpMint, err = GetPoolLpMintPDA(info.Pool.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool LP mint PDA: %w", err)
	}
	
	// 获取币创建者保险库权限 PDA
	info.CoinCreatorVaultAuthority, err = GetCoinCreatorVaultAuthorityPDA(coinCreator)
	if err != nil {
		return nil, fmt.Errorf("failed to get coin creator vault authority PDA: %w", err)
	}
	
	// 获取全局交易量累加器 PDA
	info.GlobalVolumeAccumulator, err = GetGlobalVolumeAccumulatorPumpSwapPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global volume accumulator PDA: %w", err)
	}
	
	// 获取用户交易量累加器 PDA
	info.UserVolumeAccumulator, err = GetUserVolumeAccumulatorPumpSwapPDA(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user volume accumulator PDA: %w", err)
	}
	
	// 获取联合曲线 PDA
	info.BondingCurve, err = GetBondingCurvePumpSwapPDA(poolParams.BaseMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get bonding curve PDA: %w", err)
	}
	
	// 获取元数据 PDA
	info.Metadata, err = GetMetadataPumpSwapPDA(poolParams.BaseMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata PDA: %w", err)
	}
	
	// 获取用户代币账户
	info.UserBaseTokenAccount, err = GetAssociatedTokenAddress(poolParams.BaseMint, user, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user base token account: %w", err)
	}
	
	info.UserQuoteTokenAccount, err = GetAssociatedTokenAddress(poolParams.QuoteMint, user, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user quote token account: %w", err)
	}
	
	info.UserPoolTokenAccount, err = GetAssociatedTokenAddress(info.PoolLpMint.Address, user, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user pool token account: %w", err)
	}
	
	// 获取池子代币账户
	info.PoolBaseTokenAccount, err = GetPoolTokenAccount(info.Pool.Address, poolParams.BaseMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool base token account: %w", err)
	}
	
	info.PoolQuoteTokenAccount, err = GetPoolTokenAccount(info.Pool.Address, poolParams.QuoteMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool quote token account: %w", err)
	}
	
	// 获取币创建者保险库 ATA
	info.CoinCreatorVaultATA, err = GetAssociatedTokenAddress(poolParams.QuoteMint, info.CoinCreatorVaultAuthority.Address, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get coin creator vault ATA: %w", err)
	}
	
	return info, nil
}

// PrintPumpSwapPDAInfo 打印 PumpSwap PDA 信息
func (info *PumpSwapPDAInfo) PrintPumpSwapPDAInfo() {
	fmt.Println("=== PumpSwap (PumpAMM) PDA Information ===")
	fmt.Printf("Global Config:               %s (bump: %d)\n", info.GlobalConfig.Address, info.GlobalConfig.Bump)
	fmt.Printf("Event Authority:             %s (bump: %d)\n", info.EventAuthority.Address, info.EventAuthority.Bump)
	fmt.Printf("Pool:                        %s (bump: %d)\n", info.Pool.Address, info.Pool.Bump)
	fmt.Printf("Pool LP Mint:                %s (bump: %d)\n", info.PoolLpMint.Address, info.PoolLpMint.Bump)
	fmt.Printf("Coin Creator Vault Authority: %s (bump: %d)\n", info.CoinCreatorVaultAuthority.Address, info.CoinCreatorVaultAuthority.Bump)
	fmt.Printf("Global Volume Accumulator:   %s (bump: %d)\n", info.GlobalVolumeAccumulator.Address, info.GlobalVolumeAccumulator.Bump)
	fmt.Printf("User Volume Accumulator:     %s (bump: %d)\n", info.UserVolumeAccumulator.Address, info.UserVolumeAccumulator.Bump)
	fmt.Printf("Bonding Curve:               %s (bump: %d)\n", info.BondingCurve.Address, info.BondingCurve.Bump)
	fmt.Printf("Metadata:                    %s (bump: %d)\n", info.Metadata.Address, info.Metadata.Bump)
	
	fmt.Println("\n=== Token Accounts ===")
	fmt.Printf("User Base Token Account:     %s\n", info.UserBaseTokenAccount)
	fmt.Printf("User Quote Token Account:    %s\n", info.UserQuoteTokenAccount)
	fmt.Printf("User Pool Token Account:     %s\n", info.UserPoolTokenAccount)
	fmt.Printf("Pool Base Token Account:     %s\n", info.PoolBaseTokenAccount)
	fmt.Printf("Pool Quote Token Account:    %s\n", info.PoolQuoteTokenAccount)
	fmt.Printf("Coin Creator Vault ATA:      %s\n", info.CoinCreatorVaultATA)
	
	fmt.Println("\n=== Fixed Addresses ===")
	fmt.Printf("Protocol Fee Recipient:      %s\n", PROTOCOL_FEE_RECIPIENT)
	fmt.Printf("Protocol Fee Recipient ATA:  %s\n", PROTOCOL_FEE_RECIPIENT_TOKEN_ACCOUNT)
	fmt.Printf("Admin Address:               %s\n", ADMIN_ADDRESS)
	fmt.Println("=========================================")
}

// TokenAccountInfo 代币账户信息
type TokenAccountInfo struct {
	UserBaseTokenAccount   solana.PublicKey
	UserQuoteTokenAccount  solana.PublicKey
	UserPoolTokenAccount   solana.PublicKey
	PoolBaseTokenAccount   solana.PublicKey
	PoolQuoteTokenAccount  solana.PublicKey
	CoinCreatorVaultATA    solana.PublicKey
}

// GetTokenAccounts 获取所有相关的代币账户
func GetTokenAccounts(user, pool, lpMint, baseMint, quoteMint, coinCreatorVaultAuthority solana.PublicKey) (*TokenAccountInfo, error) {
	info := &TokenAccountInfo{}
	var err error
	
	// 用户代币账户
	info.UserBaseTokenAccount, err = GetAssociatedTokenAddress(baseMint, user, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user base token account: %w", err)
	}
	
	info.UserQuoteTokenAccount, err = GetAssociatedTokenAddress(quoteMint, user, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user quote token account: %w", err)
	}
	
	info.UserPoolTokenAccount, err = GetAssociatedTokenAddress(lpMint, user, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user pool token account: %w", err)
	}
	
	// 池子代币账户
	info.PoolBaseTokenAccount, err = GetPoolTokenAccount(pool, baseMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool base token account: %w", err)
	}
	
	info.PoolQuoteTokenAccount, err = GetPoolTokenAccount(pool, quoteMint)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool quote token account: %w", err)
	}
	
	// 币创建者保险库 ATA
	info.CoinCreatorVaultATA, err = GetAssociatedTokenAddress(quoteMint, coinCreatorVaultAuthority, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get coin creator vault ATA: %w", err)
	}
	
	return info, nil
}

// 示例用法函数
func PumpSwapExampleUsage() {
	// 示例参数
	userPubkey := solana.MustPublicKeyFromBase58("GnYrYW9KPtUws8yQ19ftnuSQGWJotaLKwesS1VoRsFoF")
	creatorPubkey := solana.MustPublicKeyFromBase58("4SA8GUevgpg2EG7X9cLigXZpeafS124pp5gRnGRava4s")
	baseMintPubkey := solana.MustPublicKeyFromBase58("BYYg1btkqB9P1kMyxHCtMYW7cHJdUcGosd2rKduFpump")
	quoteMintPubkey := WSOL_MINT
	coinCreatorPubkey := solana.MustPublicKeyFromBase58("DRUDqK3C8ertN8mAg7vKyCJmX36aJbdqKAnNFLdFEneh")
	
	poolParams := PoolParams{
		Index:     0,
		Creator:   creatorPubkey,
		BaseMint:  baseMintPubkey,
		QuoteMint: quoteMintPubkey,
	}
	
	fmt.Println("PumpSwap (PumpAMM) PDA Calculator - Go Version")
	fmt.Printf("User: %s\n", userPubkey)
	fmt.Printf("Creator: %s\n", creatorPubkey)
	fmt.Printf("Base Mint: %s\n", baseMintPubkey)
	fmt.Printf("Quote Mint: %s\n", quoteMintPubkey)
	fmt.Printf("Coin Creator: %s\n", coinCreatorPubkey)
	fmt.Printf("Pool Index: %d\n", poolParams.Index)
	fmt.Println()
	
	// 获取所有 PDA 和代币账户
	pdaInfo, err := GetAllPumpSwapPDAs(userPubkey, poolParams, coinCreatorPubkey)
	if err != nil {
		fmt.Printf("Error getting PumpSwap PDAs: %v\n", err)
		return
	}
	
	// 打印所有 PDA 信息
	pdaInfo.PrintPumpSwapPDAInfo()
	
	// 单独测试各个 PDA 函数
	fmt.Println("\n=== Individual PDA Tests ===")
	
	// 测试全局配置 PDA
	globalConfig, err := GetGlobalConfigPDA()
	if err != nil {
		fmt.Printf("Error getting global config PDA: %v\n", err)
	} else {
		fmt.Printf("✅ Global Config PDA: %s (bump: %d)\n", globalConfig.Address, globalConfig.Bump)
	}
	
	// 测试池子 PDA
	pool, err := GetPoolPDA(poolParams.Index, poolParams.Creator, poolParams.BaseMint, poolParams.QuoteMint)
	if err != nil {
		fmt.Printf("Error getting pool PDA: %v\n", err)
	} else {
		fmt.Printf("✅ Pool PDA: %s (bump: %d)\n", pool.Address, pool.Bump)
	}
	
	// 测试池子代币账户
	poolBaseTokenAccount, err := GetPoolTokenAccount(pool.Address, baseMintPubkey)
	if err != nil {
		fmt.Printf("Error getting pool base token account: %v\n", err)
	} else {
		fmt.Printf("✅ Pool Base Token Account: %s\n", poolBaseTokenAccount)
	}
	
	poolQuoteTokenAccount, err := GetPoolTokenAccount(pool.Address, quoteMintPubkey)
	if err != nil {
		fmt.Printf("Error getting pool quote token account: %v\n", err)
	} else {
		fmt.Printf("✅ Pool Quote Token Account: %s\n", poolQuoteTokenAccount)
	}
	
	// 测试币创建者保险库权限 PDA
	coinCreatorVaultAuthority, err := GetCoinCreatorVaultAuthorityPDA(coinCreatorPubkey)
	if err != nil {
		fmt.Printf("Error getting coin creator vault authority PDA: %v\n", err)
	} else {
		fmt.Printf("✅ Coin Creator Vault Authority PDA: %s (bump: %d)\n", coinCreatorVaultAuthority.Address, coinCreatorVaultAuthority.Bump)
	}
	
	// 测试用户交易量累加器 PDA
	userVolumeAccumulator, err := GetUserVolumeAccumulatorPumpSwapPDA(userPubkey)
	if err != nil {
		fmt.Printf("Error getting user volume accumulator PDA: %v\n", err)
	} else {
		fmt.Printf("✅ User Volume Accumulator PDA: %s (bump: %d)\n", userVolumeAccumulator.Address, userVolumeAccumulator.Bump)
	}
}
