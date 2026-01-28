package solana

import (
	"bytes"
	"context"
	"encoding/binary"
	// "encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Program IDs
var (
	PumpFunProgramID           = solana.MustPublicKeyFromBase58("6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P")
	EventAuthority             = solana.MustPublicKeyFromBase58("Ce6TQqeHC9p8KetsN6JsjHK7UTZk7nasjjnr7XxXp9F1")
	MPLTokenMetadataProgramID  = solana.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")
	TokenProgramID             = solana.TokenProgramID
	AssociatedTokenProgramID   = solana.SPLAssociatedTokenAccountProgramID
	SystemProgramID            = solana.SystemProgramID
	SysvarRentPubkey          = solana.SysVarRentPubkey
)

// Instruction discriminators
var (
	InstructionInitialize = []byte{175, 175, 109, 31, 13, 152, 155, 237}
	InstructionSetParams  = []byte{165, 31, 134, 53, 189, 180, 130, 255}
	InstructionCreate     = []byte{24, 30, 200, 40, 5, 28, 7, 119}
	InstructionBuy        = []byte{102, 6, 61, 18, 1, 218, 235, 234}
	InstructionSell       = []byte{51, 230, 133, 164, 1, 127, 131, 173}
	InstructionWithdraw   = []byte{183, 18, 70, 156, 148, 109, 161, 34}
)

// Seeds for PDAs
var (
	SeedGlobal        = []byte("global")
	SeedMintAuthority = []byte("mint-authority")
	SeedBondingCurve  = []byte("bonding-curve")
	SeedCreatorVault  = []byte("creator-vault")
	SeedVault         = []byte("vault")
	SeedMetadata      = []byte("metadata")
)

// PDA helper functions
func GetGlobalPDA() (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{SeedGlobal},
		PumpFunProgramID,
	)
}

func GetCreatorVaultPDA(user solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{
			SeedCreatorVault,
			user.Bytes(),
		},
		PumpFunProgramID,
	)
}

func GetMintAuthorityPDA() (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{SeedMintAuthority},
		PumpFunProgramID,
	)
}

func GetBondingCurvePDA(mint solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{SeedBondingCurve, mint.Bytes()},
		PumpFunProgramID,
	)
}

func GetMetadataPDA(mint solana.PublicKey) (solana.PublicKey, uint8, error) {
	return solana.FindProgramAddress(
		[][]byte{
			SeedMetadata,
			MPLTokenMetadataProgramID.Bytes(),
			mint.Bytes(),
		},
		MPLTokenMetadataProgramID,
	)
}

// Serialization helpers
func serializeString(str string) []byte {
	strBytes := []byte(str)
	lengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBytes, uint32(len(strBytes)))
	return append(lengthBytes, strBytes...)
}

func serializeU64(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, value)
	return bytes
}

func serializePubkey(pubkey solana.PublicKey) []byte {
	return pubkey.Bytes()
}

// Instruction builders

// CreateInitializeInstruction creates an initialize instruction
func CreateInitializeInstruction(user solana.PublicKey) (solana.Instruction, error) {
	globalPDA, _, err := GetGlobalPDA()
	if err != nil {
		return nil, err
	}

	accounts := []*solana.AccountMeta{
		{PublicKey: globalPDA, IsWritable: true, IsSigner: false},
		{PublicKey: user, IsWritable: true, IsSigner: true},
		{PublicKey: SystemProgramID, IsWritable: false, IsSigner: false},
	}

	return solana.NewInstruction(
		PumpFunProgramID,
		accounts,
		InstructionInitialize,
	), nil
}

// CreateSetParamsInstruction creates a set params instruction
func CreateSetParamsInstruction(
	user solana.PublicKey,
	feeRecipient solana.PublicKey,
	initialVirtualTokenReserves uint64,
	initialVirtualSolReserves uint64,
	initialRealTokenReserves uint64,
	tokenTotalSupply uint64,
	feeBasisPoints uint64,
) (solana.Instruction, error) {
	globalPDA, _, err := GetGlobalPDA()
	if err != nil {
		return nil, err
	}

	accounts := []*solana.AccountMeta{
		{PublicKey: globalPDA, IsWritable: true, IsSigner: false},
		{PublicKey: user, IsWritable: true, IsSigner: true},
		{PublicKey: SystemProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: EventAuthority, IsWritable: false, IsSigner: false},
		{PublicKey: PumpFunProgramID, IsWritable: false, IsSigner: false},
	}

	data := bytes.Join([][]byte{
		InstructionSetParams,
		serializePubkey(feeRecipient),
		serializeU64(initialVirtualTokenReserves),
		serializeU64(initialVirtualSolReserves),
		serializeU64(initialRealTokenReserves),
		serializeU64(tokenTotalSupply),
		serializeU64(feeBasisPoints),
	}, nil)

	return solana.NewInstruction(
		PumpFunProgramID,
		accounts,
		data,
	), nil
}

// CreateCreateInstruction creates a create token instruction
func CreateCreateInstruction(
	mint solana.PublicKey,
	user solana.PublicKey,
	name string,
	symbol string,
	uri string,
	creator solana.PublicKey,
) (solana.Instruction, error) {
	globalPDA, _, err := GetGlobalPDA()
	if err != nil {
		return nil, err
	}

	mintAuthorityPDA, _, err := GetMintAuthorityPDA()
	if err != nil {
		return nil, err
	}

	bondingCurvePDA, _, err := GetBondingCurvePDA(mint)
	if err != nil {
		return nil, err
	}

	metadataPDA, _, err := GetMetadataPDA(mint)
	if err != nil {
		return nil, err
	}

	associatedBondingCurve, _, err := solana.FindAssociatedTokenAddress(bondingCurvePDA, mint)
	if err != nil {
		return nil, err
	}

	accounts := []*solana.AccountMeta{
		{PublicKey: mint, IsWritable: true, IsSigner: true},
		{PublicKey: mintAuthorityPDA, IsWritable: false, IsSigner: false},
		{PublicKey: bondingCurvePDA, IsWritable: true, IsSigner: false},
		{PublicKey: associatedBondingCurve, IsWritable: true, IsSigner: false},
		{PublicKey: globalPDA, IsWritable: false, IsSigner: false},
		{PublicKey: MPLTokenMetadataProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: metadataPDA, IsWritable: true, IsSigner: false},
		{PublicKey: user, IsWritable: true, IsSigner: true},
		{PublicKey: SystemProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: TokenProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: AssociatedTokenProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: SysvarRentPubkey, IsWritable: false, IsSigner: false},
		{PublicKey: EventAuthority, IsWritable: false, IsSigner: false},
		{PublicKey: PumpFunProgramID, IsWritable: false, IsSigner: false},
	}

	data := bytes.Join([][]byte{
		InstructionCreate,
		serializeString(name),
		serializeString(symbol),
		serializeString(uri),
		serializePubkey(creator),
	}, nil)

	return solana.NewInstruction(
		PumpFunProgramID,
		accounts,
		data,
	), nil
}

// CreateBuyInstruction creates a buy instruction
func CreateBuyInstruction(
	mint solana.PublicKey,
	bondingCurvePDA solana.PublicKey,
	associatedBondingCurve solana.PublicKey,
	associatedUser solana.PublicKey,
	creatorVaultPDA solana.PublicKey,
	user solana.PublicKey,
	feeRecipient solana.PublicKey,
	amount uint64,
	maxSolCost uint64,
) (solana.Instruction, error) {
	globalPDA, _, err := GetGlobalPDA()
	if err != nil {
		return nil, err
	}

	accounts := []*solana.AccountMeta{
		{PublicKey: globalPDA, IsWritable: false, IsSigner: false},
		{PublicKey: feeRecipient, IsWritable: true, IsSigner: false},
		{PublicKey: mint, IsWritable: false, IsSigner: false},
		{PublicKey: bondingCurvePDA, IsWritable: true, IsSigner: false},
		{PublicKey: associatedBondingCurve, IsWritable: true, IsSigner: false},
		{PublicKey: associatedUser, IsWritable: true, IsSigner: false},
		{PublicKey: user, IsWritable: true, IsSigner: true},
		{PublicKey: SystemProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: TokenProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: creatorVaultPDA, IsWritable: true, IsSigner: false},
		{PublicKey: EventAuthority, IsWritable: false, IsSigner: false},
		{PublicKey: PumpFunProgramID, IsWritable: false, IsSigner: false},
	}

	data := bytes.Join([][]byte{
		InstructionBuy,
		serializeU64(amount),
		serializeU64(maxSolCost),
	}, nil)

	return solana.NewInstruction(
		PumpFunProgramID,
		accounts,
		data,
	), nil
}

// CreateSellInstruction creates a sell instruction
func CreateSellInstruction(
	mint solana.PublicKey,
	bondingCurvePDA solana.PublicKey,
	associatedBondingCurve solana.PublicKey,
	associatedUser solana.PublicKey,
	creatorVaultPDA solana.PublicKey,
	user solana.PublicKey,
	feeRecipient solana.PublicKey,
	amount uint64,
	minSolOutput uint64,
) (solana.Instruction, error) {
	globalPDA, _, err := GetGlobalPDA()
	if err != nil {
		return nil, err
	}

	accounts := []*solana.AccountMeta{
		{PublicKey: globalPDA, IsWritable: false, IsSigner: false},
		{PublicKey: feeRecipient, IsWritable: true, IsSigner: false},
		{PublicKey: mint, IsWritable: false, IsSigner: false},
		{PublicKey: bondingCurvePDA, IsWritable: true, IsSigner: false},
		{PublicKey: associatedBondingCurve, IsWritable: true, IsSigner: false},
		{PublicKey: associatedUser, IsWritable: true, IsSigner: false},
		{PublicKey: user, IsWritable: true, IsSigner: true},
		{PublicKey: SystemProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: creatorVaultPDA, IsWritable: true, IsSigner: false},
		{PublicKey: TokenProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: EventAuthority, IsWritable: false, IsSigner: false},
		{PublicKey: PumpFunProgramID, IsWritable: false, IsSigner: false},
	}

	data := bytes.Join([][]byte{
		InstructionSell,
		serializeU64(amount),
		serializeU64(minSolOutput),
	}, nil)

	return solana.NewInstruction(
		PumpFunProgramID,
		accounts,
		data,
	), nil
}

// CreateWithdrawInstruction creates a withdraw instruction
func CreateWithdrawInstruction(
	mint solana.PublicKey,
	user solana.PublicKey,
	lastWithdraw solana.PublicKey,
) (solana.Instruction, error) {
	globalPDA, _, err := GetGlobalPDA()
	if err != nil {
		return nil, err
	}

	bondingCurvePDA, _, err := GetBondingCurvePDA(mint)
	if err != nil {
		return nil, err
	}

	associatedBondingCurve, _, err := solana.FindAssociatedTokenAddress(bondingCurvePDA, mint)
	if err != nil {
		return nil, err
	}

	associatedUser, _, err := solana.FindAssociatedTokenAddress(user, mint)
	if err != nil {
		return nil, err
	}

	accounts := []*solana.AccountMeta{
		{PublicKey: globalPDA, IsWritable: false, IsSigner: false},
		{PublicKey: lastWithdraw, IsWritable: true, IsSigner: false},
		{PublicKey: mint, IsWritable: false, IsSigner: false},
		{PublicKey: bondingCurvePDA, IsWritable: true, IsSigner: false},
		{PublicKey: associatedBondingCurve, IsWritable: true, IsSigner: false},
		{PublicKey: associatedUser, IsWritable: true, IsSigner: false},
		{PublicKey: user, IsWritable: true, IsSigner: true},
		{PublicKey: SystemProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: TokenProgramID, IsWritable: false, IsSigner: false},
		{PublicKey: SysvarRentPubkey, IsWritable: false, IsSigner: false},
		{PublicKey: EventAuthority, IsWritable: false, IsSigner: false},
		{PublicKey: PumpFunProgramID, IsWritable: false, IsSigner: false},
	}

	return solana.NewInstruction(
		PumpFunProgramID,
		accounts,
		InstructionWithdraw,
	), nil
}

// Utility function to print accounts (for debugging)
func PrintAccounts(accounts []*solana.AccountMeta) {
	fmt.Println("Accounts:")
	for i, account := range accounts {
		fmt.Printf("%d: %s (writable: %t, signer: %t)\n", 
			i, account.PublicKey.String(), account.IsWritable, account.IsSigner)
	}
}

// BondingState represents the state of a bonding curve
type BondingState struct {
	UnknownData           uint64
	VirtualTokenReserves  uint64
	VirtualSolReserves    uint64
	RealTokenReserves     uint64
	RealSolReserves       uint64
	TokenTotalSupply      uint64
	Complete              bool
	Creator               solana.PublicKey
}

// PumpFunInternalPoolStat represents the complete state of a pump pool
type PumpFunInternalPoolStat struct {
	Timestamp             int64   `json:"timestamp"`
	Mint                  string  `json:"mint"`
	FeeRate               float64 `json:"feeRate"`
	UnknownData           uint64  `json:"unknownData"`
	VirtualTokenReserves  uint64  `json:"virtualTokenReserves"`
	VirtualSolReserves    uint64  `json:"virtualSolReserves"`
	RealTokenReserves     uint64  `json:"realTokenReserves"`
	RealSolReserves       uint64  `json:"realSolReserves"`
	TokenTotalSupply      uint64  `json:"tokenTotalSupply"`
	Complete              bool    `json:"complete"`
	Creator               string  `json:"creator"`
	Price                 float64 `json:"price"`
	FeeRecipient          string  `json:"feeRecipient"`
	BondingCurvePDA       string  `json:"bondingCurvePDA"`
	AssociatedBondingCurve string `json:"associatedBondingCurve"`
	CreatorVaultPDA       string  `json:"creatorVaultPDA"`
}

// DecodeBondingState decodes the bonding state from raw data
func DecodeBondingState(data []byte) (*BondingState, error) {
	buf := bytes.NewReader(data)
	var s BondingState

	if err := binary.Read(buf, binary.LittleEndian, &s.UnknownData); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &s.VirtualTokenReserves); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &s.VirtualSolReserves); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &s.RealTokenReserves); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &s.RealSolReserves); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &s.TokenTotalSupply); err != nil {
		return nil, err
	}
	var completeByte byte
	if err := binary.Read(buf, binary.LittleEndian, &completeByte); err != nil {
		return nil, err
	}
	s.Complete = completeByte != 0

	creatorBytes := make([]byte, 32)
	if _, err := buf.Read(creatorBytes); err != nil {
		return nil, err
	}
	s.Creator = solana.PublicKeyFromBytes(creatorBytes)

	if len(s.Creator) != 32 {
		return nil, errors.New("invalid creator pubkey")
	}

	return &s, nil
}

// GetPumpFunInternalPoolStat retrieves and decodes the pool state for a given mint
func GetPumpFunInternalPoolStat(
	client *rpc.Client,
	mint solana.PublicKey,
	feeRate float64,
	feeRecipient solana.PublicKey,
) (*PumpFunInternalPoolStat, error) {
	bondingPDA, _, err := GetBondingCurvePDA(mint)
	if err != nil {
		return nil, err
	}

	accountInfo, err := client.GetAccountInfo(context.Background(), bondingPDA)
	if err != nil {
		return nil, err
	}
	if accountInfo == nil || accountInfo.Value == nil {
		return nil, errors.New("bonding PDA not found")
	}

	state, err := DecodeBondingState(accountInfo.Value.Data.GetBinary())
	if err != nil {
		return nil, err
	}

	virtualToken := state.VirtualTokenReserves
	virtualSol := state.VirtualSolReserves
	if virtualToken == 0 {
		virtualToken = 1 // 设置一个默认值
	}
	price := float64(virtualSol) / 1e9 / (float64(virtualToken) / 1e6)

	associatedBondingCurve, _, err := solana.FindAssociatedTokenAddress(bondingPDA, mint)
	if err != nil {
		return nil, err
	}

	creatorVaultPDA, _, err := GetCreatorVaultPDA(state.Creator)
	if err != nil {
		return nil, err
	}

	stat := &PumpFunInternalPoolStat{
		Timestamp:              time.Now().Unix(),
		Mint:                   mint.String(),
		FeeRate:                feeRate,
		UnknownData:            state.UnknownData,
		VirtualTokenReserves:   state.VirtualTokenReserves,
		VirtualSolReserves:     state.VirtualSolReserves,
		RealTokenReserves:      state.RealTokenReserves,
		RealSolReserves:        state.RealSolReserves,
		TokenTotalSupply:       state.TokenTotalSupply,
		Complete:               state.Complete,
		Creator:                state.Creator.String(),
		Price:                  price,
		FeeRecipient:           feeRecipient.String(),
		BondingCurvePDA:        bondingPDA.String(),
		AssociatedBondingCurve: associatedBondingCurve.String(),
		CreatorVaultPDA:        creatorVaultPDA.String(),
	}

	// jsonBytes, _ := json.MarshalIndent(stat, "", "  ")
	// fmt.Println("\n✅ Pump Pool Stat:")
	// fmt.Println(string(jsonBytes))
	return stat, nil
}

// PDAResult represents a PDA calculation result
type PDAResult struct {
	Address solana.PublicKey `json:"address"`
	Bump    uint8            `json:"bump"`
}

// PumpFunPDAInfo contains all PDA information
type PumpFunPDAInfo struct {
	EventAuthority          PDAResult `json:"eventAuthority"`
	Global                  PDAResult `json:"global"`
	CreatorVault            PDAResult `json:"creatorVault"`
	MintAuthority          PDAResult `json:"mintAuthority"`
	BondingCurve           PDAResult `json:"bondingCurve"`
	GlobalVolumeAccumulator PDAResult `json:"globalVolumeAccumulator"`
	UserVolumeAccumulator   PDAResult `json:"userVolumeAccumulator"`
	Metadata               PDAResult `json:"metadata"`
}

// Seeds for additional PDAs
var (
	SeedEventAuthority         = []byte("__event_authority")
	SeedGlobalVolumeAccumulator = []byte("global_volume_accumulator")
	SeedUserVolumeAccumulator   = []byte("user_volume_accumulator")
)

// GetEventAuthorityPDA gets event authority PDA
func GetEventAuthorityPDA() (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedEventAuthority},
		PumpFunProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find event authority PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetGlobalVolumeAccumulatorPDA gets global volume accumulator PDA
func GetGlobalVolumeAccumulatorPDA() (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedGlobalVolumeAccumulator},
		PumpFunProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find global volume accumulator PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetUserVolumeAccumulatorPDA gets user volume accumulator PDA
func GetUserVolumeAccumulatorPDA(user solana.PublicKey) (PDAResult, error) {
	address, bump, err := solana.FindProgramAddress(
		[][]byte{SeedUserVolumeAccumulator, user[:]},
		PumpFunProgramID,
	)
	if err != nil {
		return PDAResult{}, fmt.Errorf("failed to find user volume accumulator PDA: %w", err)
	}
	
	return PDAResult{
		Address: address,
		Bump:    bump,
	}, nil
}

// GetAllPumpFunPDAs gets all PumpFun related PDAs for a user and mint
func GetAllPumpFunPDAs(user solana.PublicKey, mint solana.PublicKey) (*PumpFunPDAInfo, error) {
	info := &PumpFunPDAInfo{}
	var err error
	
	// Get event authority PDA
	info.EventAuthority, err = GetEventAuthorityPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get event authority PDA: %w", err)
	}
	
	// Get global PDA
	globalAddr, globalBump, err := GetGlobalPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global PDA: %w", err)
	}
	info.Global = PDAResult{Address: globalAddr, Bump: globalBump}
	
	// Get creator vault PDA
	creatorVaultAddr, creatorVaultBump, err := GetCreatorVaultPDA(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get creator vault PDA: %w", err)
	}
	info.CreatorVault = PDAResult{Address: creatorVaultAddr, Bump: creatorVaultBump}
	
	// Get mint authority PDA
	mintAuthorityAddr, mintAuthorityBump, err := GetMintAuthorityPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get mint authority PDA: %w", err)
	}
	info.MintAuthority = PDAResult{Address: mintAuthorityAddr, Bump: mintAuthorityBump}
	
	// Get bonding curve PDA
	bondingCurveAddr, bondingCurveBump, err := GetBondingCurvePDA(mint)
	if err != nil {
		return nil, fmt.Errorf("failed to get bonding curve PDA: %w", err)
	}
	info.BondingCurve = PDAResult{Address: bondingCurveAddr, Bump: bondingCurveBump}
	
	// Get global volume accumulator PDA
	info.GlobalVolumeAccumulator, err = GetGlobalVolumeAccumulatorPDA()
	if err != nil {
		return nil, fmt.Errorf("failed to get global volume accumulator PDA: %w", err)
	}
	
	// Get user volume accumulator PDA
	info.UserVolumeAccumulator, err = GetUserVolumeAccumulatorPDA(user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user volume accumulator PDA: %w", err)
	}
	
	// Get metadata PDA
	metadataAddr, metadataBump, err := GetMetadataPDA(mint)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata PDA: %w", err)
	}
	info.Metadata = PDAResult{Address: metadataAddr, Bump: metadataBump}
	
	return info, nil
}
