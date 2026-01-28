package solana

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	log "github.com/sirupsen/logrus"
)

// var (
// 	rpcClient *rpc.Client
// 	once      sync.Once
// )

// initRPCClient 初始化 RPC 客户端（只会执行一次）
// func initRPCClient() {
// 	once.Do(func() {
// 		rpcEndpoint := os.Getenv("DEFAULT_SOLANA_RPC")
// 		if rpcEndpoint == "" {
// 			panic("DEFAULT_SOLANA_RPC environment variable is not set")
// 		}
// 		rpcClient = rpc.New(rpcEndpoint)
// 	})
// }

// CheckTransactionStatus 检查 Solana 交易状态
func CheckTransactionStatus(signature string) (string, error) {
	// initRPCClient()

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// sig, err := solana.SignatureFromBase58(signature)
	// if err != nil {
	// 	return "", fmt.Errorf("invalid signature format: %v", err)
	// }

	// res, err := rpcClient.GetSignatureStatuses(ctx, []solana.Signature{sig})
	// if err != nil {
	// 	return "", fmt.Errorf("failed to get signature status: %v", err)
	// }

	// if len(res.Value) == 0 || res.Value[0] == nil {
	// 	return "pending", nil
	// }

	// status := res.Value[0]

	// if status.Err != nil {
	// 	errJSON, _ := json.Marshal(status.Err)
	// 	return "error", fmt.Errorf("transaction failed: %s", string(errJSON))
	// }

	// switch status.ConfirmationStatus {
	// case rpc.ConfirmationStatusFinalized:
	// 	return "finalized", nil
	// case rpc.ConfirmationStatusConfirmed:
	// 	return "confirmed", nil
	// case rpc.ConfirmationStatusProcessed:
	// 	return "pending", nil
	// }

	return "pending", nil
}

type TransferResult struct {
	AccountAddress string
	Success        bool
	Signature      string
	Error          error
}

type transferTask struct {
	AccountAddress string
	AccountPubkey  solana.PublicKey
	PrivateKey     *solana.PrivateKey
	SourceATA      solana.PublicKey
	Balance        uint64
}

func MultiTransferMintToTarget(
	client *rpc.Client,
	accounts []string,
	mint string,
	targetAddress string,
	rps int,
	accountToPrivateKey map[string]*solana.PrivateKey,
	decimals uint8,
) ([]TransferResult, error) {
	ctx := context.Background()
	var results []TransferResult

	// --- Validate and prepare ---
	mintPubkey, err := solana.PublicKeyFromBase58(mint)
	if err != nil {
		return nil, fmt.Errorf("invalid mint: %w", err)
	}
	targetPubkey, err := solana.PublicKeyFromBase58(targetAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid target address: %w", err)
	}

	// --- Ensure target ATA exists ---
	targetATA, err := GetAssociatedTokenAddress(mintPubkey, targetPubkey)
	if err != nil {
		return nil, err
	}
	targetInfo, _ := client.GetAccountInfo(ctx, targetATA)
	if targetInfo == nil || targetInfo.Value == nil {
		if err := createATA(client, targetPubkey, mintPubkey, accountToPrivateKey); err != nil {
			return nil, fmt.Errorf("failed to create target ATA: %w", err)
		}
		log.Infof("Target ATA created: %s", targetATA)
	}

	// --- Prepare transfer tasks ---
	accountMap := make(map[string]solana.PublicKey)
	for _, addr := range accounts {
		pub, err := solana.PublicKeyFromBase58(addr)
		if err == nil {
			accountMap[addr] = pub
		}
	}
	balances, err := GetMultiAccountsMint(client, accountMap, mint, decimals)
	if err != nil {
		return nil, fmt.Errorf("get balances error: %w", err)
	}

	var tasks []transferTask
	for addr, bal := range balances {
		if bal.Balance == 0 {
			continue
		}
		sourceATA, err := GetAssociatedTokenAddress(mintPubkey, accountMap[addr])
		if err != nil {
			continue
		}
		tasks = append(tasks, transferTask{
			AccountAddress: addr,
			AccountPubkey:  accountMap[addr],
			PrivateKey:     accountToPrivateKey[addr],
			SourceATA:      sourceATA,
			Balance:        bal.Balance,
		})
	}

	if len(tasks) == 0 {
		log.Infof("No accounts with balance")
		return nil, nil
	}

	// --- Execute in parallel ---
	limiter := rate.NewLimiter(rate.Limit(rps), rps) // 每秒允许 rps 次请求，突发上限为 rps

	resultCh := make(chan TransferResult, len(tasks))
	var wg sync.WaitGroup

	for _, task := range tasks {
		wg.Add(1)
		go func(t transferTask) {
			defer wg.Done()

			// 速率限制：等待获取 token
			if err := limiter.Wait(context.Background()); err != nil {
				resultCh <- TransferResult{
					AccountAddress: t.AccountAddress,
					Success:        false,
					Error:          fmt.Errorf("rate limiter wait failed: %w", err),
				}
				return
			}

			// 执行转账，带重试机制（最多重试 3 次）
			maxRetries := 3
			var res TransferResult
			for attempt := 0; attempt <= maxRetries; attempt++ {
				res = transferAndClose(client, t, targetATA, mintPubkey)

				// 如果成功，直接返回
				if res.Success {
					resultCh <- res
					return
				}

				// 如果失败且还有重试机会，等待后重试
				if attempt < maxRetries {
					// 等待速率限制器允许下一次请求
					if err := limiter.Wait(context.Background()); err != nil {
						resultCh <- TransferResult{
							AccountAddress: t.AccountAddress,
							Success:        false,
							Error:          fmt.Errorf("rate limiter wait failed on retry: %w", err),
						}
						return
					}
					// 添加短暂延迟，避免立即重试
					time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
					log.Warnf("Transfer failed for account %s, attempt %d/%d, retrying... Error: %v",
						t.AccountAddress, attempt+1, maxRetries, res.Error)
				}
			}

			// 所有重试都失败，返回最后一次的结果
			log.Errorf("Transfer failed for account %s after %d attempts, giving up. Error: %v",
				t.AccountAddress, maxRetries+1, res.Error)
			resultCh <- res
		}(task)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		results = append(results, res)
	}
	return results, nil
}

func createATA(client *rpc.Client, owner, mint solana.PublicKey, privMap map[string]*solana.PrivateKey) error {
	ctx := context.Background()

	var payer solana.PublicKey
	var payerPriv *solana.PrivateKey
	for _, key := range privMap {
		if key != nil {
			payer = key.PublicKey()
			payerPriv = key
			break
		}
	}
	if payerPriv == nil {
		return fmt.Errorf("no payer available")
	}

	ix := associatedtokenaccount.NewCreateInstruction(payer, owner, mint).Build()
	bh, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return err
	}
	tx, err := solana.NewTransaction([]solana.Instruction{ix}, bh.Value.Blockhash, solana.TransactionPayer(payer))
	if err != nil {
		return err
	}
	if _, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(payer) {
			return payerPriv
		}
		return nil
	}); err != nil {
		return err
	}
	if _, err := client.SendTransaction(ctx, tx); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

func transferAndClose(
	client *rpc.Client,
	task transferTask,
	targetATA solana.PublicKey,
	mint solana.PublicKey,
) TransferResult {
	ctx := context.Background()

	bh, err := client.GetLatestBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return TransferResult{AccountAddress: task.AccountAddress, Success: false, Error: err}
	}

	ixTransfer := token.NewTransferInstruction(task.Balance, task.SourceATA, targetATA, task.AccountPubkey, nil).Build()
	ixClose := token.NewCloseAccountInstruction(task.SourceATA, task.AccountPubkey, task.AccountPubkey, nil).Build()
	tx, err := solana.NewTransaction([]solana.Instruction{ixTransfer, ixClose}, bh.Value.Blockhash, solana.TransactionPayer(task.AccountPubkey))
	if err != nil {
		return TransferResult{AccountAddress: task.AccountAddress, Success: false, Error: err}
	}
	if _, err := tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(task.AccountPubkey) {
			return task.PrivateKey
		}
		return nil
	}); err != nil {
		return TransferResult{AccountAddress: task.AccountAddress, Success: false, Error: err}
	}
	sig, err := client.SendTransaction(ctx, tx)
	if err != nil {
		return TransferResult{AccountAddress: task.AccountAddress, Success: false, Error: err}
	}
	return TransferResult{AccountAddress: task.AccountAddress, Success: true, Signature: sig.String()}
}
