package solana

import (
	"context"
	"math"

	// "encoding/json"
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"marketcontrol/internal/models"
)

// TokenMetadata represents the metadata of a token
type TokenMetadata struct {
	Key                  uint8
	UpdateAuthority      solana.PublicKey
	Mint                 solana.PublicKey
	Name                 string
	Symbol               string
	Uri                  string
	SellerFeeBasisPoints uint16
	Creator              string
	// 其他字段（如 creators、flags）可按需扩展
}

// TokenBalance represents the balance information of a token
type TokenBalance struct {
	Mint            string    `json:"mint"`
	AccountAddress  string    `json:"account_address"`
	Balance         uint64    `json:"balance"`
	BalanceReadable float64   `json:"balance_readable"`
	Decimals        uint8     `json:"decimals"`
	LastUpdated     time.Time `json:"last_updated"`
}

// GetSolBalance 查询 owner 的 SOL 余额
func GetSolBalance(client *rpc.Client, owner solana.PublicKey) (uint64, time.Time, error) {
	resp, err := client.GetBalance(context.Background(), owner, rpc.CommitmentFinalized)
	if err != nil {
		log.Errorf("> 查询 owner %s 的 SOL 余额失败: %v", owner.String(), err)
		return 0, time.Time{}, err
	}
	return resp.Value, time.Now(), nil
}

// GetTokenBalance 通过 TokenAccount 表获取 AccountAddress，再查余额
func GetTokenBalance(db *gorm.DB, client *rpc.Client, owner solana.PublicKey, mint string) (uint64, time.Time, error) {
	var tokenAccounts []string
	var tokenAccount models.TokenAccount
	err := db.Where("owner_address = ? AND mint = ? AND is_close = ?", owner.String(), mint, false).First(&tokenAccount).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			mintPubkey := solana.MustPublicKeyFromBase58(mint)
			resp, err := client.GetTokenAccountsByOwner(context.Background(), owner, &rpc.GetTokenAccountsConfig{
				Mint: &mintPubkey,
			}, &rpc.GetTokenAccountsOpts{Encoding: solana.EncodingBase64})
			if err != nil {
				log.Errorf("> 链上查询 owner %s 的 token %s 账户失败: %v", owner.String(), mint, err)
				return 0, time.Now(), err
			}
			if len(resp.Value) == 0 {
				return 0, time.Now(), nil // 链上没有代币帐户
			}
			// 批量写入所有链上账户，并收集 account address
			for _, v := range resp.Value {
				accountAddress := v.Pubkey.String()
				tokenAccount := models.TokenAccount{
					OwnerAddress:   owner.String(),
					Mint:           mint,
					AccountAddress: accountAddress,
					IsClose:        false,
				}
				if err := db.Create(&tokenAccount).Error; err != nil {
					return 0, time.Now(), err
				}
				tokenAccounts = append(tokenAccounts, accountAddress)
			}
		}
	} else {
		// 数据库已有，查所有 account address
		var dbTokenAccounts []models.TokenAccount
		db.Where("owner_address = ? AND mint = ? AND is_close = ?", owner.String(), mint, false).Find(&dbTokenAccounts)
		for _, ta := range dbTokenAccounts {
			tokenAccounts = append(tokenAccounts, ta.AccountAddress)
		}
	}

	// 查询余额，遍历所有账户，累加余额
	totalAmt := uint64(0)
	for _, accAddr := range tokenAccounts {
		accountPubkey, err := solana.PublicKeyFromBase58(accAddr)
		if err != nil {
			log.Errorf("> 解析 accountAddress %s 失败: %v", accAddr, err)
			continue
		}
		balResp, err := client.GetTokenAccountBalance(context.Background(), accountPubkey, rpc.CommitmentFinalized)
		if err != nil {
			log.Errorf("> 查询 account %s 的余额失败: %v", accAddr, err)
			continue
		}
		if balResp == nil || balResp.Value == nil {
			log.Errorf("> 查询 account %s 的余额返回空值", accAddr)
			continue
		}
		log.Infof("> 查询 account %s 的代币余额成功: %s", accAddr, balResp.Value.Amount)
		amt, err := strconv.ParseUint(balResp.Value.Amount, 10, 64)
		if err != nil {
			log.Errorf("> 解析余额失败: %v", err)
			continue
		}
		totalAmt += amt
	}
	return totalAmt, time.Now(), nil
}

func readString(buf *bytes.Buffer) (string, error) {
	var strLen uint32
	if err := binary.Read(buf, binary.LittleEndian, &strLen); err != nil {
		return "", err
	}
	strBytes := make([]byte, strLen)
	if _, err := buf.Read(strBytes); err != nil {
		return "", err
	}
	return string(strBytes), nil
}

func GetTokenMetadata(client *rpc.Client, mint solana.PublicKey) (*TokenMetadata, error) {
	// Metaplex Metadata Program
	metadataProgramID := solana.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")

	// 标准 PDA seeds: ["metadata", programID, mint]
	seeds := [][]byte{
		[]byte("metadata"),
		metadataProgramID.Bytes(),
		mint.Bytes(),
	}

	metadataAddress, _, err := solana.FindProgramAddress(seeds, metadataProgramID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive metadata address: %w", err)
	}

	accountInfo, err := client.GetAccountInfo(context.Background(), metadataAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metadata: %w", err)
	}
	if accountInfo == nil || accountInfo.Value == nil || accountInfo.Value.Data == nil {
		return nil, fmt.Errorf("no metadata found for mint: %s", mint.String())
	}

	data := accountInfo.Value.Data.GetBinary()
	buf := bytes.NewBuffer(data)

	var meta TokenMetadata
	if err := binary.Read(buf, binary.LittleEndian, &meta.Key); err != nil {
		return nil, err
	}
	if _, err := buf.Read(meta.UpdateAuthority[:]); err != nil {
		return nil, err
	}
	if _, err := buf.Read(meta.Mint[:]); err != nil {
		return nil, err
	}

	if meta.Name, err = readString(buf); err != nil {
		return nil, err
	}
	if meta.Symbol, err = readString(buf); err != nil {
		return nil, err
	}
	if meta.Uri, err = readString(buf); err != nil {
		return nil, err
	}

	if err := binary.Read(buf, binary.LittleEndian, &meta.SellerFeeBasisPoints); err != nil {
		return nil, err
	}

	// Parse creators if they exist
	// First, check if creators data exists
	var hasCreators uint8
	if err := binary.Read(buf, binary.LittleEndian, &hasCreators); err != nil {
		// If we can't read hasCreators, assume no creators and use UpdateAuthority as creator
		meta.Creator = meta.UpdateAuthority.String()
		return &meta, nil
	}

	if hasCreators == 1 {
		// Read number of creators
		var numCreators uint32
		if err := binary.Read(buf, binary.LittleEndian, &numCreators); err != nil {
			meta.Creator = meta.UpdateAuthority.String()
			return &meta, nil
		}

		if numCreators > 0 {
			// Read first creator address (32 bytes)
			var creatorPubkey [32]byte
			if _, err := buf.Read(creatorPubkey[:]); err != nil {
				meta.Creator = meta.UpdateAuthority.String()
				return &meta, nil
			}
			creator := solana.PublicKeyFromBytes(creatorPubkey[:])
			meta.Creator = creator.String()

			// Skip remaining creator data (verified flag, share)
			buf.Next(1 + 1) // 1 byte for verified, 1 byte for share

			// Skip remaining creators if any
			for i := uint32(1); i < numCreators; i++ {
				buf.Next(32 + 1 + 1) // 32 bytes address + 1 byte verified + 1 byte share
			}
		} else {
			meta.Creator = meta.UpdateAuthority.String()
		}
	} else {
		// No creators, use UpdateAuthority as creator
		meta.Creator = meta.UpdateAuthority.String()
	}

	return &meta, nil
}

// GetAllTokenBalance 查询指定地址的所有 SPL 代币余额
func GetAllTokenBalance(db *gorm.DB, client *rpc.Client, owner solana.PublicKey) ([]TokenBalance, error) {
	var tokenBalances []TokenBalance
	ownerStr := owner.String()

	// 从数据库查询该地址的所有代币账户
	var tokenAccounts []models.TokenAccount
	err := db.Where("owner_address = ? AND is_close = ?", ownerStr, false).Find(&tokenAccounts).Error
	if err != nil {
		log.Errorf("> 查询地址 %s 的代币账户失败: %v", ownerStr, err)
		return nil, err
	}

	if len(tokenAccounts) == 0 {
		log.Infof("> 地址 %s 没有找到任何代币账户", ownerStr)
		return tokenBalances, nil
	}

	log.Infof("> 找到地址 %s 的 %d 个代币账户", ownerStr, len(tokenAccounts))

	// 按 mint 分组，处理同一 mint 的多个账户
	mintGroups := make(map[string][]models.TokenAccount)
	for _, ta := range tokenAccounts {
		mintGroups[ta.Mint] = append(mintGroups[ta.Mint], ta)
	}

	// 遍历每个 mint，查询余额
	for mint, accounts := range mintGroups {
		totalBalance := uint64(0)
		var accountAddresses []string

		// 查询该 mint 的所有账户余额
		for _, account := range accounts {
			accountPubkey, err := solana.PublicKeyFromBase58(account.AccountAddress)
			if err != nil {
				log.Errorf("> 解析账户地址 %s 失败: %v", account.AccountAddress, err)
				continue
			}

			balResp, err := client.GetTokenAccountBalance(context.Background(), accountPubkey, rpc.CommitmentFinalized)
			if err != nil {
				log.Errorf("> 查询账户 %s 的余额失败: %v", account.AccountAddress, err)
				continue
			}

			if balResp == nil || balResp.Value == nil {
				log.Errorf("> 查询账户 %s 的余额返回空值", account.AccountAddress)
				continue
			}

			balance, err := strconv.ParseUint(balResp.Value.Amount, 10, 64)
			if err != nil {
				log.Errorf("> 解析账户 %s 的余额失败: %v", account.AccountAddress, err)
				continue
			}

			totalBalance += balance
			accountAddresses = append(accountAddresses, account.AccountAddress)

			log.Infof("> 账户 %s (mint: %s) 余额: %s", account.AccountAddress, mint, balResp.Value.Amount)
		}

		// 如果总余额大于0，添加到结果中
		if totalBalance > 0 {
			// 获取代币的 decimals 信息
			decimals := uint8(9) // 默认值
			// 这里可以扩展获取代币的 decimals，暂时使用默认值
			// 在实际应用中，可能需要查询代币的 mint 账户信息

			// 计算可读余额
			divisor := uint64(1) << decimals
			balanceReadable := float64(totalBalance) / float64(divisor)

			tokenBalance := TokenBalance{
				Mint:            mint,
				AccountAddress:  accountAddresses[0], // 使用第一个账户地址作为代表
				Balance:         totalBalance,
				BalanceReadable: balanceReadable,
				Decimals:        decimals,
				LastUpdated:     time.Now(),
			}

			tokenBalances = append(tokenBalances, tokenBalance)
		}
	}

	log.Infof("> 地址 %s 共有 %d 种代币余额", ownerStr, len(tokenBalances))
	return tokenBalances, nil
}

// MultiAccountBalance represents the balance information for a single account
type MultiAccountBalance struct {
	AccountAddress  string  `json:"account_address"`
	Lamports        uint64  `json:"lamports"`
	Balance         uint64  `json:"balance"`
	BalanceReadable float64 `json:"balance_readable"`
	Decimals        uint8   `json:"decimals"`
}

// mintBalanceInfo represents token balance information for a single account
type mintBalanceInfo struct {
	Balance         uint64
	BalanceReadable float64
}

// GetMultiAccountsSol 批量获取多个账户的 SOL 余额（lamports）
func GetMultiAccountsSol(client *rpc.Client, accountPubkeys []solana.PublicKey) (map[string]uint64, error) {
	if len(accountPubkeys) == 0 {
		return make(map[string]uint64), nil
	}

	ctx := context.Background()
	args := make([]reflect.Value, 0, len(accountPubkeys)+1)
	args = append(args, reflect.ValueOf(ctx))
	for _, addr := range accountPubkeys {
		args = append(args, reflect.ValueOf(addr))
	}

	// Call GetMultipleAccounts using reflection to get SOL balances
	method := reflect.ValueOf(client).MethodByName("GetMultipleAccounts")
	if !method.IsValid() {
		return nil, fmt.Errorf("GetMultipleAccounts method not found")
	}

	result := method.Call(args)
	if len(result) != 2 {
		return nil, fmt.Errorf("unexpected return value count")
	}

	// Check for error
	if errVal := result[1]; !errVal.IsNil() {
		if err, ok := errVal.Interface().(error); ok {
			return nil, fmt.Errorf("failed to get multiple accounts info: %w", err)
		}
	}

	// Get the result for SOL balances
	solAccountsInfoVal := result[0]
	if !solAccountsInfoVal.IsValid() || solAccountsInfoVal.IsNil() {
		return nil, fmt.Errorf("GetMultipleAccounts returned nil result")
	}
	solAccountsInfo, ok := solAccountsInfoVal.Interface().(*rpc.GetMultipleAccountsResult)
	if !ok {
		return nil, fmt.Errorf("failed to convert GetMultipleAccounts result")
	}

	// Create a map to store SOL balances (lamports) by account address
	lamportsMap := make(map[string]uint64)
	for i, accountInfo := range solAccountsInfo.Value {
		accountPubkey := accountPubkeys[i]
		accountStr := accountPubkey.String()

		if accountInfo == nil {
			lamportsMap[accountStr] = 0
		} else {
			lamportsMap[accountStr] = accountInfo.Lamports
		}
	}

	return lamportsMap, nil
}

// GetMultiAccountsMint 批量获取多个账户的代币余额（通过 ATA）
func GetMultiAccountsMint(client *rpc.Client, accountStrToPubkey map[string]solana.PublicKey, mint string, decimals uint8) (map[string]mintBalanceInfo, error) {
	if len(accountStrToPubkey) == 0 {
		return make(map[string]mintBalanceInfo), nil
	}

	// Parse mint address
	mintPubkey, err := solana.PublicKeyFromBase58(mint)
	if err != nil {
		return nil, fmt.Errorf("invalid mint address: %w", err)
	}

	// Calculate ATA addresses for all accounts
	var ataAddresses []solana.PublicKey
	accountToATA := make(map[string]solana.PublicKey) // Map from account address to ATA

	for accountStr, accountPubkey := range accountStrToPubkey {
		// Use GetAssociatedTokenAddress from pumpswap.go
		ata, err := GetAssociatedTokenAddress(mintPubkey, accountPubkey)
		if err != nil {
			log.Warnf("Failed to calculate ATA for account %s: %v", accountStr, err)
			continue
		}

		ataAddresses = append(ataAddresses, ata)
		accountToATA[accountStr] = ata
	}

	if len(ataAddresses) == 0 {
		// Return empty map if no ATA addresses
		return make(map[string]mintBalanceInfo), nil
	}

	// Get multiple ATA account info in batch
	ctx := context.Background()
	ataArgs := make([]reflect.Value, 0, len(ataAddresses)+1)
	ataArgs = append(ataArgs, reflect.ValueOf(ctx))
	for _, addr := range ataAddresses {
		ataArgs = append(ataArgs, reflect.ValueOf(addr))
	}

	method := reflect.ValueOf(client).MethodByName("GetMultipleAccounts")
	if !method.IsValid() {
		return nil, fmt.Errorf("GetMultipleAccounts method not found")
	}

	ataResult := method.Call(ataArgs)
	if len(ataResult) != 2 {
		return nil, fmt.Errorf("unexpected return value count for ATA accounts")
	}

	// Check for error
	if errVal := ataResult[1]; !errVal.IsNil() {
		if err, ok := errVal.Interface().(error); ok {
			return nil, fmt.Errorf("failed to get multiple ATA accounts info: %w", err)
		}
	}

	// Get the result for ATA balances
	ataAccountsInfoVal := ataResult[0]
	if !ataAccountsInfoVal.IsValid() || ataAccountsInfoVal.IsNil() {
		return nil, fmt.Errorf("GetMultipleAccounts returned nil result for ATA accounts")
	}
	ataAccountsInfo, ok := ataAccountsInfoVal.Interface().(*rpc.GetMultipleAccountsResult)
	if !ok {
		return nil, fmt.Errorf("failed to convert GetMultipleAccounts result for ATA accounts")
	}

	// Parse balances from account data
	mintBalanceMap := make(map[string]mintBalanceInfo)
	for i, accountInfo := range ataAccountsInfo.Value {
		ataAddr := ataAddresses[i]

		// Find the corresponding account address
		var accountStr string
		for accStr, ata := range accountToATA {
			if ata.Equals(ataAddr) {
				accountStr = accStr
				break
			}
		}

		if accountStr == "" {
			log.Warnf("Could not find account address for ATA: %s", ataAddr.String())
			continue
		}

		if accountInfo == nil {
			// ATA account doesn't exist, token balance is 0
			mintBalanceMap[accountStr] = mintBalanceInfo{
				Balance:         0,
				BalanceReadable: 0,
			}
			continue
		}

		// Parse account data to get token balance
		// SPL Token account layout: balance is at offset 64 (8 bytes, uint64 little-endian)
		data := accountInfo.Data.GetBinary()
		if len(data) < 72 {
			log.Warnf("Account data too short: %d bytes for account %s", len(data), accountStr)
			mintBalanceMap[accountStr] = mintBalanceInfo{
				Balance:         0,
				BalanceReadable: 0,
			}
			continue
		}

		// Read balance from offset 64
		balanceBytes := data[64:72]
		balance := binary.LittleEndian.Uint64(balanceBytes)

		// Calculate readable balance
		divisor := math.Pow(10, float64(decimals))
		balanceReadable := float64(balance) / float64(divisor)

		mintBalanceMap[accountStr] = mintBalanceInfo{
			Balance:         balance,
			BalanceReadable: balanceReadable,
		}
	}

	return mintBalanceMap, nil
}

// GetMultiAccountsInfo 批量获取多个账户的代币余额信息
func GetMultiAccountsInfo(client *rpc.Client, accounts []string, mint string, decimals uint8) ([]MultiAccountBalance, error) {
	if len(accounts) == 0 {
		return []MultiAccountBalance{}, nil
	}

	// Parse all account addresses
	var accountPubkeys []solana.PublicKey
	accountStrToPubkey := make(map[string]solana.PublicKey)

	for _, accountStr := range accounts {
		accountPubkey, err := solana.PublicKeyFromBase58(accountStr)
		if err != nil {
			log.Warnf("Invalid account address %s: %v", accountStr, err)
			continue
		}
		accountPubkeys = append(accountPubkeys, accountPubkey)
		accountStrToPubkey[accountStr] = accountPubkey
	}

	if len(accountPubkeys) == 0 {
		return []MultiAccountBalance{}, nil
	}

	// Step 1: Get SOL balances (lamports) for all accounts
	lamportsMap, err := GetMultiAccountsSol(client, accountPubkeys)
	if err != nil {
		return nil, fmt.Errorf("failed to get SOL balances: %w", err)
	}

	// Step 2: Get token balances (ATA balances)
	mintBalanceMap, err := GetMultiAccountsMint(client, accountStrToPubkey, mint, decimals)
	if err != nil {
		return nil, fmt.Errorf("failed to get token balances: %w", err)
	}

	// Combine SOL and token balances
	var results []MultiAccountBalance
	for accountStr, lamports := range lamportsMap {
		mintBalance, exists := mintBalanceMap[accountStr]
		if !exists {
			// If no token balance info, set to 0
			mintBalance = mintBalanceInfo{
				Balance:         0,
				BalanceReadable: 0,
			}
		}

		results = append(results, MultiAccountBalance{
			AccountAddress:  accountStr,
			Lamports:        lamports,
			Balance:         mintBalance.Balance,
			BalanceReadable: mintBalance.BalanceReadable,
			Decimals:        decimals,
		})
	}

	return results, nil
}

// AddressBalanceChange represents the balance change for an address from a transaction
type AddressBalanceChange struct {
	Address          string  `json:"address"`
	Mint             string  `json:"mint"`                // "sol" for native SOL
	DeltaLamports    int64   `json:"delta_lamports"`      // SOL change in lamports (post - pre)
	DeltaTokenAmount float64 `json:"delta_token_amount"`  // Token change (post - pre), 0 for SOL
	DeltaTokenRaw    string  `json:"delta_token_raw"`     // Raw token amount string if needed
	DeltaReadable    float64 `json:"delta_readable"`      // Human-readable delta using Decimals (SOL: lamports/10^decimals; token: ui amount)
}

// GetTransactionBySignature fetches a transaction by signature from Solana RPC
func GetTransactionBySignature(client *rpc.Client, signature string) (*rpc.GetTransactionResult, error) {
	sig, err := solana.SignatureFromBase58(signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}
	ctx := context.Background()
	maxVer := rpc.MaxSupportedTransactionVersion1
	opts := &rpc.GetTransactionOpts{
		Encoding:                     solana.EncodingBase64,
		MaxSupportedTransactionVersion: &maxVer,
	}
	txResult, err := client.GetTransaction(ctx, sig, opts)
	if err != nil {
		return nil, fmt.Errorf("getTransaction: %w", err)
	}
	if txResult == nil || txResult.Transaction == nil {
		return nil, fmt.Errorf("transaction not found")
	}
	return txResult, nil
}

// ParseAddressBalanceChangesFromTransaction parses balance changes for the given addresses and mint from a transaction result.
// If mint is "sol" (case-insensitive), parses SOL balance changes; otherwise parses SPL token balance changes for that mint.
// decimals is used for DeltaReadable: for SOL, readable = lamports/10^decimals (default 9 if 0); for token, readable = ui amount.
func ParseAddressBalanceChangesFromTransaction(txResult *rpc.GetTransactionResult, addressList []string, mint string, decimals uint) ([]AddressBalanceChange, error) {
	if txResult == nil || txResult.Transaction == nil || txResult.Meta == nil {
		return nil, fmt.Errorf("transaction or meta is nil")
	}
	tx, err := txResult.Transaction.GetTransaction()
	if err != nil {
		return nil, fmt.Errorf("decode transaction: %w", err)
	}
	meta := txResult.Meta
	addressSet := make(map[string]bool)
	for _, a := range addressList {
		addressSet[a] = true
	}
		var results []AddressBalanceChange
	mintLower := strings.ToLower(strings.TrimSpace(mint))
	if mintLower == "sol" {
		dec := decimals
		if dec == 0 {
			dec = 9
		}
		divisor := math.Pow(10, float64(dec))
		// Build full account key list (static + loaded addresses for versioned tx)
		accountKeys := make([]solana.PublicKey, 0, len(tx.Message.AccountKeys)+32)
		accountKeys = append(accountKeys, tx.Message.AccountKeys...)
		if len(meta.LoadedAddresses.Writable) > 0 || len(meta.LoadedAddresses.ReadOnly) > 0 {
			accountKeys = append(accountKeys, meta.LoadedAddresses.Writable...)
			accountKeys = append(accountKeys, meta.LoadedAddresses.ReadOnly...)
		}
		preBalances := meta.PreBalances
		postBalances := meta.PostBalances
		if len(preBalances) != len(accountKeys) || len(postBalances) != len(accountKeys) {
			log.Warnf("preBalances/postBalances length mismatch with account keys: keys=%d pre=%d post=%d", len(accountKeys), len(preBalances), len(postBalances))
		}
		for _, addr := range addressList {
			change := AddressBalanceChange{Address: addr, Mint: "sol"}
			pk, err := solana.PublicKeyFromBase58(addr)
			if err != nil {
				change.DeltaLamports = 0
				results = append(results, change)
				continue
			}
			for i, key := range accountKeys {
				if !key.Equals(pk) {
					continue
				}
				var pre, post uint64
				if i < len(preBalances) {
					pre = preBalances[i]
				}
				if i < len(postBalances) {
					post = postBalances[i]
				}
				change.DeltaLamports = int64(post) - int64(pre)
				change.DeltaReadable = float64(change.DeltaLamports) / divisor
				break
			}
			results = append(results, change)
		}
		return results, nil
	}
	// SPL token balance changes: filter by Owner and Mint
	mintPK, err := solana.PublicKeyFromBase58(mint)
	if err != nil {
		return nil, fmt.Errorf("invalid mint address: %w", err)
	}
	preToken := meta.PreTokenBalances
	postToken := meta.PostTokenBalances
	for _, addr := range addressList {
		change := AddressBalanceChange{Address: addr, Mint: mint}
		ownerPK, err := solana.PublicKeyFromBase58(addr)
		if err != nil {
			results = append(results, change)
			continue
		}
		var preAmount, postAmount float64
		for _, bal := range preToken {
			if bal.Mint.Equals(mintPK) && bal.Owner != nil && bal.Owner.Equals(ownerPK) {
				if bal.UiTokenAmount != nil && bal.UiTokenAmount.UiAmount != nil {
					preAmount = *bal.UiTokenAmount.UiAmount
				}
				break
			}
		}
		for _, bal := range postToken {
			if bal.Mint.Equals(mintPK) && bal.Owner != nil && bal.Owner.Equals(ownerPK) {
				if bal.UiTokenAmount != nil && bal.UiTokenAmount.UiAmount != nil {
					postAmount = *bal.UiTokenAmount.UiAmount
				}
				break
			}
		}
		change.DeltaTokenAmount = postAmount - preAmount
		change.DeltaReadable = change.DeltaTokenAmount
		results = append(results, change)
	}
	return results, nil
}
