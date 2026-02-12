//go:build integrationtests

// Package bsc provides BSC integration tests.
package bsc

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/coreum/v5/pkg/client"
	"github.com/CoreumFoundation/coreum/v5/testutil/integration"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	integrationtests "github.com/tokenize-x/tx-xrpl-token-migrator/integration-tests"
	"github.com/tokenize-x/tx-xrpl-token-migrator/integration-tests/bsc/evm"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc"
	bscabi "github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/executor"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/finder"
)

// holds an executor and its cancel function.
type executorInstance struct {
	executor *executor.Executor
	cancel   context.CancelFunc
}

func tokensToAmount(tokens int64) *big.Int {
	amount := big.NewInt(tokens)
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	return amount.Mul(amount, multiplier)
}

func ethToWei(eth int64) *big.Int {
	amount := big.NewInt(eth)
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	return amount.Mul(amount, multiplier)
}

// TestBSCLiveScanner tests the real BSC scanner against a local BSC node.
func TestBSCLiveScanner(t *testing.T) {
	requireT := require.New(t)
	ctx, chains := NewTestingContext(t)
	logger := zaptest.NewLogger(t)

	rpcClient := chains.BSC.RPCClient()

	// Get deployer private key
	deployer := chains.BSC.GenAccount(t)
	chains.BSC.FundAccount(ctx, t, rpcClient, deployer.Address, ethToWei(10))

	// Configure bridge for testing
	bridgeCfg := evm.DefaultBridgeConfig()

	// Deploy contracts
	t.Log("Deploying TXToken and TXBridge contracts...")
	contracts, err := evm.SetupBridgeEnvironment(
		ctx, rpcClient, chains.BSC.ChainID(), chains.BSC.KeyStore(), deployer, bridgeCfg,
	)
	requireT.NoError(err)
	t.Logf("Token deployed at: %s", contracts.TokenAddress.Hex())
	t.Logf("Bridge deployed at: %s", contracts.BridgeAddress.Hex())

	// Get user key and fund with ETH for gas
	user := chains.BSC.GenAccount(t)
	chains.BSC.FundAccount(ctx, t, rpcClient, user.Address, ethToWei(1))

	// Mint tokens to user
	mintAmount := tokensToAmount(100) // 100 tokens (6 decimals)
	t.Logf("Minting %s tokens to user %s", mintAmount.String(), user.Address.Hex())
	err = evm.MintTokens(
		ctx,
		rpcClient,
		chains.BSC.ChainID(),
		chains.BSC.KeyStore(),
		deployer,
		contracts.Token,
		user.Address,
		mintAmount,
	)
	requireT.NoError(err)

	// Verify user balance
	balance, err := contracts.Token.BalanceOf(nil, user.Address)
	requireT.NoError(err)
	requireT.Equal(mintAmount.String(), balance.String(), "user should have minted tokens")

	// TX destination address (valid bech32)
	txAddress := "devcore1cz8x502s930v0ux8m6lpfw6s3l5tydz3gsx87w"

	// Bridge
	bridgeAmount := tokensToAmount(10) // 10 tokens
	t.Logf("Bridging %s tokens to %s", bridgeAmount.String(), txAddress)

	bridgeTx, err := evm.SendToTxChain(
		ctx,
		rpcClient,
		chains.BSC.ChainID(),
		chains.BSC.KeyStore(),
		user,
		contracts.Bridge,
		bridgeAmount,
		txAddress,
	)
	requireT.NoError(err)
	t.Logf("Bridge transaction: %s", bridgeTx.TxHash.Hex())

	// Create scanner
	scannerCfg := bsc.ScannerConfig{
		RPCURL:        chains.BSC.cfg.RPCAddress,
		BridgeAddress: contracts.BridgeAddress,
		StartBlock:    0,
		PollInterval:  500 * time.Millisecond,
		Confirmations: 0,
	}

	scanner, err := bsc.NewScanner(scannerCfg, logger, rpcClient, nil)
	requireT.NoError(err)

	// Subscribe to events
	eventCh := make(chan *bscabi.TXBridgeSentToTXChain, 10)
	scanCtx, scanCancel := context.WithTimeout(ctx, 10*time.Second)
	defer scanCancel()

	err = scanner.Subscribe(scanCtx, eventCh)
	requireT.NoError(err)

	// Wait for event
	t.Log("Waiting for SentToTXChain event...")
	select {
	case event := <-eventCh:
		t.Logf("Received event: from=%s, amount=%s, payload=%s",
			event.From.Hex(), event.Amount.String(), event.TxAddress)
		requireT.Equal(user.Address, event.From, "event should be from user")
		requireT.Equal(bridgeAmount.String(), event.Amount.String(), "amount should match")
		requireT.Equal(txAddress, event.TxAddress, "txAddress should match")
	case <-scanCtx.Done():
		t.Fatal("timeout waiting for bridge event")
	}

	t.Log("Live scanner test passed!")
}

// TestBSCLiveMultipleTransactions tests multiple bridge transactions through the live flow.
func TestBSCLiveMultipleTransactions(t *testing.T) {
	ctx, chains := NewTestingContext(t)
	requireT := require.New(t)
	txChain := chains.TX
	logger := zaptest.NewLogger(t)

	rpcClient := chains.BSC.RPCClient()

	// Get deployer key
	deployer := chains.BSC.GenAccount(t)
	chains.BSC.FundAccount(ctx, t, rpcClient, deployer.Address, ethToWei(10))

	// Configure bridge
	bridgeCfg := evm.DefaultBridgeConfig()

	// Deploy EVM contracts
	t.Log("Deploying EVM contracts...")
	contracts, err := evm.SetupBridgeEnvironment(
		ctx, rpcClient, chains.BSC.ChainID(), chains.BSC.KeyStore(), deployer, bridgeCfg,
	)
	requireT.NoError(err)

	// Setup TX chain side
	owner := txChain.TXChain.GenAccount()
	trustedAddress1 := txChain.TXChain.GenAccount()
	trustedAddress2 := txChain.TXChain.GenAccount()

	txChain.TXChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress1, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress2, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
	)

	contractClient := tx.NewContractClient(tx.DefaultContractClientConfig(nil, ""), txChain.TXChain.ClientContext)

	trustedAddresses := []string{
		trustedAddress1.String(),
		trustedAddress2.String(),
	}

	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, tx.DeployAndInstantiateConfig{
		Owner:            owner.String(),
		Admin:            owner.String(),
		TrustedAddresses: trustedAddresses,
		Threshold:        2,
		MinAmount:        sdkmath.NewIntFromUint64(100),
		MaxAmount:        sdkmath.NewIntFromUint64(500_000_000),
		XRPLTokens:       []tx.XRPLToken{},
		Label:            "bsc_live_multi_test",
	})
	requireT.NoError(err)

	coinToFundContract := txChain.TXChain.NewCoin(sdkmath.NewInt(50_000_000_000))
	txChain.TXChain.Faucet.FundAccounts(ctx, t, integration.NewFundedAccount(contractAddr, coinToFundContract))
	requireT.NoError(contractClient.SetContractAddress(contractAddr))

	// Create recipients
	recipient1 := txChain.TXChain.GenAccount()
	recipient2 := txChain.TXChain.GenAccount()

	// Get multiple EVM user keys and fund with ETH for gas
	user1 := chains.BSC.GenAccount(t)
	user2 := chains.BSC.GenAccount(t)
	chains.BSC.FundAccount(ctx, t, rpcClient, user1.Address, ethToWei(1))
	chains.BSC.FundAccount(ctx, t, rpcClient, user2.Address, ethToWei(1))

	// Mint tokens to users
	mintAmount := tokensToAmount(100) // 100 tokens each
	err = evm.MintTokens(
		ctx,
		rpcClient,
		chains.BSC.ChainID(),
		chains.BSC.KeyStore(),
		deployer,
		contracts.Token,
		user1.Address,
		mintAmount,
	)
	requireT.NoError(err)
	err = evm.MintTokens(
		ctx,
		rpcClient,
		chains.BSC.ChainID(),
		chains.BSC.KeyStore(),
		deployer,
		contracts.Token,
		user2.Address,
		mintAmount,
	)
	requireT.NoError(err)

	// Create scanner and start executors
	scanner, err := bsc.NewScanner(bsc.ScannerConfig{
		RPCURL:        chains.BSC.cfg.RPCAddress,
		BridgeAddress: contracts.BridgeAddress,
		StartBlock:    0,
		PollInterval:  500 * time.Millisecond,
		Confirmations: 0,
	}, logger, rpcClient, nil)
	requireT.NoError(err)

	instances := buildAndStartBSCLiveExecutors(
		ctx, t, txChain, contractAddr,
		[]sdk.AccAddress{trustedAddress1, trustedAddress2},
		scanner,
	)

	// Bridge transaction 1: 30 tokens from user1 to recipient1
	_, err = evm.SendToTxChain(
		ctx, rpcClient,
		chains.BSC.ChainID(),
		chains.BSC.KeyStore(),
		user1,
		contracts.Bridge,
		tokensToAmount(30),
		recipient1.String(),
	)
	requireT.NoError(err)
	t.Log("Submitted bridge tx 1: 30 tokens to recipient1")

	// Bridge transaction 2: 45 tokens from user2 to recipient2
	_, err = evm.SendToTxChain(
		ctx, rpcClient,
		chains.BSC.ChainID(),
		chains.BSC.KeyStore(),
		user2,
		contracts.Bridge,
		tokensToAmount(45),
		recipient2.String(),
	)
	requireT.NoError(err)
	t.Log("Submitted bridge tx 2: 45 tokens to recipient2")

	// Bridge transaction 3: 20 tokens from user1 to recipient1
	_, err = evm.SendToTxChain(
		ctx, rpcClient,
		chains.BSC.ChainID(),
		chains.BSC.KeyStore(),
		user1,
		contracts.Bridge,
		tokensToAmount(20),
		recipient1.String(),
	)
	requireT.NoError(err)
	t.Log("Submitted bridge tx 3: 20 tokens to recipient1")

	// wait tx
	// recipient1: 30 + 20 = 50 tokens = 50_000_000
	expectedBalance1 := txChain.TXChain.NewCoin(sdkmath.NewInt(50_000_000))
	awaitForBalance(ctx, t, txChain.TXChain.ClientContext, recipient1.String(), expectedBalance1)

	// recipient2: 45 tokens = 45_000_000
	expectedBalance2 := txChain.TXChain.NewCoin(sdkmath.NewInt(45_000_000))
	awaitForBalance(ctx, t, txChain.TXChain.ClientContext, recipient2.String(), expectedBalance2)

	t.Log("Multiple live bridge transactions processed successfully!")

	t.Cleanup(func() {
		time.Sleep(2 * time.Second)
		for _, instance := range instances {
			instance.cancel()
		}
	})
}

// helper that creates and starts executors with a real scanner.
func buildAndStartBSCLiveExecutors(
	ctx context.Context,
	t *testing.T,
	txChain integrationtests.TXChain,
	contractAddr sdk.AccAddress,
	trustedAddresses []sdk.AccAddress,
	scanner *bsc.Scanner,
) []*executorInstance {
	t.Helper()

	logger := zaptest.NewLogger(t)
	instances := make([]*executorInstance, 0, len(trustedAddresses))

	executionErrors := make([]error, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, trustedAddr := range trustedAddresses {
		// Create BSC finder with scanner
		bscFinder := finder.NewBSCFinder(
			finder.BSCFinderConfig{
				TXDenom:    txChain.TXChain.ChainSettings.Denom,
				TXDecimals: 6,
			},
			logger,
			scanner,
		)

		// Create contract client
		contractClient := tx.NewContractClient(
			tx.DefaultContractClientConfig(contractAddr, txChain.TXChain.ChainSettings.Denom),
			txChain.TXChain.ClientContext,
		)

		// Create executor
		exec := executor.NewExecutor(
			executor.DefaultConfig(trustedAddr),
			logger,
			contractClient,
			[]executor.Finder{bscFinder},
		)

		execCtx, execCancel := context.WithCancel(ctx)
		instances = append(instances, &executorInstance{
			executor: exec,
			cancel:   execCancel,
		})

		wg.Add(1)
		go func(e *executor.Executor, addr sdk.AccAddress) {
			defer wg.Done()
			if err := e.Start(execCtx); err != nil && !errors.Is(err, context.Canceled) {
				mu.Lock()
				executionErrors = append(executionErrors, fmt.Errorf("executor %s: %w", addr, err))
				mu.Unlock()
			}
		}(exec, trustedAddr)
	}

	t.Cleanup(func() {
		for _, instance := range instances {
			instance.cancel()
		}
		wg.Wait()
		require.Empty(t, executionErrors, "executor errors: %v", executionErrors)
	})

	time.Sleep(1 * time.Second)

	return instances
}

func awaitForBalance(
	ctx context.Context,
	t *testing.T,
	clientCtx client.Context,
	address string,
	expectedBalance sdk.Coin,
) {
	t.Helper()

	t.Logf("Waiting for account %s balance, expected balance: %s.", address, expectedBalance.String())
	bankClient := banktypes.NewQueryClient(clientCtx)
	retryCtx, retryCancel := context.WithTimeout(ctx, time.Minute)
	defer retryCancel()
	require.NoError(t, retry.Do(retryCtx, time.Second, func() error {
		requestCtx, requestCancel := context.WithTimeout(retryCtx, 5*time.Second)
		defer requestCancel()

		balancesRes, err := bankClient.AllBalances(requestCtx, &banktypes.QueryAllBalancesRequest{
			Address: address,
		})
		if err != nil {
			return err
		}

		if balancesRes.Balances.AmountOf(expectedBalance.Denom).String() != expectedBalance.Amount.String() {
			return retry.Retryable(
				errors.Errorf(
					"account %s %s balance is still not equal to expected, all balances: %s",
					address, expectedBalance, balancesRes,
				),
			)
		}

		return nil
	}))

	t.Logf("Received expected balance: %s.", expectedBalance)
}
