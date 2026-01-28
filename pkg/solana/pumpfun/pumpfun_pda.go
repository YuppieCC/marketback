package pumpfun

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
)

// PumpFun 程序常量
var (
	// PumpFun 程序地址
	PUMP_FUN_PROGRAM_ID = solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P")
	
	// Metaplex Token Metadata 程序地址
	MPL_TOKEN_METADATA_PROGRAM_ID = solana.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")
)

// PDA 种子常量
var (
	SEED_GLOBAL                    = []byte("global")
	SEED_MINT_AUTHORITY           = []byte("mint-authority")
	SEED_BONDING_CURVE            = []byte("bonding-curve")
	SEED_CREATOR_VAULT            = []byte("creator-vault")
	SEED_EVENT_AUTHORITY          = []byte("__event_authority")
	SEED_GLOBAL_VOLUME_ACCUMULATOR = []byte("global_volume_accumulator")
	SEED_USER_VOLUME_ACCUMULATOR   = []byte("user_volume_accumulator")
	SEED_METADATA                 = []byte("metadata")
)

// PDAResult 表示 PDA 计算结果
type PDAResult struct {
	Address solana.PublicKey
	Bump    uint8
}

// GetEventAuthorityPDA 获取事件权限 PDA
func GetEventAuthorityPDA() (PDAResult, error) {
	seeds := [][]byte{SEED_EVENT_AUTHORITY}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find event authority PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetGlobalPDA 获取全局状态 PDA
func GetGlobalPDA() (PDAResult, error) {
	seeds := [][]byte{SEED_GLOBAL}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find global PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetCreatorVaultPDA 获取创建者 vault PDA
func GetCreatorVaultPDA(user solana.PublicKey) (PDAResult, error) {
	seeds := [][]byte{
		SEED_CREATOR_VAULT,
		user[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find creator vault PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetMintAuthorityPDA 获取铸币权限 PDA
func GetMintAuthorityPDA() (PDAResult, error) {
	seeds := [][]byte{SEED_MINT_AUTHORITY}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find mint authority PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetBondingCurvePDA 获取联合曲线 PDA
func GetBondingCurvePDA(mint solana.PublicKey) (PDAResult, error) {
	seeds := [][]byte{
		SEED_BONDING_CURVE,
		mint[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find bonding curve PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetGlobalVolumeAccumulatorPDA 获取全局交易量累加器 PDA
func GetGlobalVolumeAccumulatorPDA() (PDAResult, error) {
	seeds := [][]byte{SEED_GLOBAL_VOLUME_ACCUMULATOR}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find global volume accumulator PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetUserVolumeAccumulatorPDA 获取用户交易量累加器 PDA
func GetUserVolumeAccumulatorPDA(user solana.PublicKey) (PDAResult, error) {
	seeds := [][]byte{
		SEED_USER_VOLUME_ACCUMULATOR,
		user[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, PUMP_FUN_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find user volume accumulator PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetMetadataPDA 获取代币元数据 PDA
func GetMetadataPDA(mint solana.PublicKey) (PDAResult, error) {
	seeds := [][]byte{
		SEED_METADATA,
		MPL_TOKEN_METADATA_PROGRAM_ID[:],
		mint[:],
	}
	
	address, bump, err := solana.FindProgramAddress(seeds, MPL_TOKEN_METADATA_PROGRAM_ID)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find metadata PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// PumpFunPDAInfo 包含所有 PDA 信息的结构体
type PumpFunPDAInfo struct {
	EventAuthority          PDAResult
	Global                  PDAResult
	CreatorVault            PDAResult
	MintAuthority          PDAResult
	BondingCurve           PDAResult
	GlobalVolumeAccumulator PDAResult
	UserVolumeAccumulator   PDAResult
	Metadata               PDAResult
}

// GetAllPDAs 获取指定用户和代币的所有相关 PDA
func GetAllPDAs(user solana.PublicKey, mint solana.PublicKey) (*PumpFunPDAInfo, error) {
	info := &PumpFunPDAInfo{}
	var err error
	
	// 获取事件权限 PDA
	info.EventAuthority, err = GetEventAuthorityPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get event authority PDA: %w", err)
	}
	
	// 获取全局 PDA
	info.Global, err = GetGlobalPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global PDA: %w", err)
	}
	
	// 获取创建者保险库 PDA
	info.CreatorVault, err = GetCreatorVaultPDA(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator vault PDA: %w", err)
	}
	
	// 获取铸币权限 PDA
	info.MintAuthority, err = GetMintAuthorityPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get mint authority PDA: %w", err)
	}
	
	// 获取联合曲线 PDA
	info.BondingCurve, err = GetBondingCurvePDA(mint)
	if err != nil {
		return nil, fmt.Errorf("failed to get bonding curve PDA: %w", err)
	}
	
	// 获取全局交易量累加器 PDA
	info.GlobalVolumeAccumulator, err = GetGlobalVolumeAccumulatorPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global volume accumulator PDA: %w", err)
	}
	
	// 获取用户交易量累加器 PDA
	info.UserVolumeAccumulator, err = GetUserVolumeAccumulatorPDA(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user volume accumulator PDA: %w", err)
	}
	
	// 获取元数据 PDA
	info.Metadata, err = GetMetadataPDA(mint)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata PDA: %w", err)
	}
	
	return info, nil
}

// PrintPDAInfo 打印 PDA 信息
func (info *PumpFunPDAInfo) PrintPDAInfo() {
	fmt.Println("=== PumpFun PDA Information ===")
	fmt.Printf("Event Authority:           %s (bump: %d)\n", info.EventAuthority.Address, info.EventAuthority.Bump)
	fmt.Printf("Global:                    %s (bump: %d)\n", info.Global.Address, info.Global.Bump)
	fmt.Printf("Creator Vault:             %s (bump: %d)\n", info.CreatorVault.Address, info.CreatorVault.Bump)
	fmt.Printf("Mint Authority:            %s (bump: %d)\n", info.MintAuthority.Address, info.MintAuthority.Bump)
	fmt.Printf("Bonding Curve:             %s (bump: %d)\n", info.BondingCurve.Address, info.BondingCurve.Bump)
	fmt.Printf("Global Volume Accumulator: %s (bump: %d)\n", info.GlobalVolumeAccumulator.Address, info.GlobalVolumeAccumulator.Bump)
	fmt.Printf("User Volume Accumulator:   %s (bump: %d)\n", info.UserVolumeAccumulator.Address, info.UserVolumeAccumulator.Bump)
	fmt.Printf("Metadata:                  %s (bump: %d)\n", info.Metadata.Address, info.Metadata.Bump)
	fmt.Println("===============================")
}

// 示例用法函数
func ExampleUsage() {
	// 示例用户和代币地址
	userPubkey := solana.MustPublicKeyFromBase58("GnYrYW9KPtUws8yQ19ftnuSQGWJotaLKwesS1VoRsFoF")
	mintPubkey := solana.MustPublicKeyFromBase58("BYYg1btkqB9P1kMyxHCtMYW7cHJdUcGosd2rKduFpump")
	
	fmt.Println("PumpFun PDA Calculator - Go Version")
	fmt.Printf("User: %s\n", userPubkey)
	fmt.Printf("Mint: %s\n", mintPubkey)
	fmt.Println()
	
	// 获取所有 PDA
	pdaInfo, err := GetAllPDAs(userPubkey, mintPubkey)
	if err != nil {
		fmt.Printf("Error getting PDAs: %v\n", err)
		return
	}
	
	// 打印所有 PDA 信息
	pdaInfo.PrintPDAInfo()
	
	// 单独测试各个 PDA 函数
	fmt.Println("\n=== Individual PDA Tests ===")
	
	// 测试事件权限 PDA
	eventAuthority, err := GetEventAuthorityPDA()
	if err != nil {
		fmt.Printf("Error getting event authority PDA: %v\n", err)
	} else {
		fmt.Printf("✅ Event Authority PDA: %s (bump: %d)\n", eventAuthority.Address, eventAuthority.Bump)
	}
	
	// 测试联合曲线 PDA
	bondingCurve, err := GetBondingCurvePDA(mintPubkey)
	if err != nil {
		fmt.Printf("Error getting bonding curve PDA: %v\n", err)
	} else {
		fmt.Printf("✅ Bonding Curve PDA: %s (bump: %d)\n", bondingCurve.Address, bondingCurve.Bump)
	}
	
	// 测试元数据 PDA
	metadata, err := GetMetadataPDA(mintPubkey)
	if err != nil {
		fmt.Printf("Error getting metadata PDA: %v\n", err)
	} else {
		fmt.Printf("✅ Metadata PDA: %s (bump: %d)\n", metadata.Address, metadata.Bump)
	}
	
	// 测试用户交易量累加器 PDA
	userVolumeAccumulator, err := GetUserVolumeAccumulatorPDA(userPubkey)
	if err != nil {
		fmt.Printf("Error getting user volume accumulator PDA: %v\n", err)
	} else {
		fmt.Printf("✅ User Volume Accumulator PDA: %s (bump: %d)\n", userVolumeAccumulator.Address, userVolumeAccumulator.Bump)
	}
}
