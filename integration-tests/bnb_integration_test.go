//go:build integrationtests

package integrationtests

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/CoreumFoundation/coreum/v5/testutil/integration"

	"github.com/tokenize-x/tx-xrpl-token-migrator/integration-tests/evm"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bnb"
	bnbabi "github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bnb/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/executor"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/finder"
)

// holds an executor and its cancel function.
type executorInstance struct {
	executor *executor.Executor
	cancel   context.CancelFunc
}

// converts whole tokens to wei (18 decimals).
func tokensToWei(tokens int64) *big.Int {
	wei := big.NewInt(tokens)
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	return wei.Mul(wei, multiplier)
}

// TestBNBLiveScanner tests the real BNB scanner against a local Anvil node.
func TestBNBLiveScanner(t *testing.T) {
	requireT := require.New(t)
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	// Start Anvil
	t.Log("Starting Anvil...")
	anvil, err := evm.StartAnvil(evm.DefaultAnvilConfig())
	requireT.NoError(err)
	t.Cleanup(func() {
		_ = anvil.Stop()
	})

	client, err := anvil.Client()
	requireT.NoError(err)

	// Get deployer private key (first Anvil account)
	deployerKey, err := evm.GetPrivateKey(0)
	requireT.NoError(err)

	// Configure bridge for testing
	bridgeCfg := evm.DefaultBridgeConfig()

	// Deploy contracts
	t.Log("Deploying TXToken and TXBridge contracts...")
	contracts, err := evm.SetupBridgeEnvironment(client, deployerKey, anvil.ChainID(), bridgeCfg)
	requireT.NoError(err)
	t.Logf("Token deployed at: %s", contracts.TokenAddress.Hex())
	t.Logf("Bridge deployed at: %s", contracts.BridgeAddress.Hex())

	// Get user key (second Anvil account)
	userKey, err := evm.GetPrivateKey(1)
	requireT.NoError(err)
	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)

	// Mint tokens to user
	mintAmount := tokensToWei(100) // 100 tokens (18 decimals)
	t.Logf("Minting %s tokens to user %s", mintAmount.String(), userAddr.Hex())
	err = evm.MintTokens(client, deployerKey, anvil.ChainID(), contracts.Token, userAddr, mintAmount)
	requireT.NoError(err)

	// Verify user balance
	balance, err := contracts.Token.BalanceOf(nil, userAddr)
	requireT.NoError(err)
	requireT.Equal(mintAmount.String(), balance.String(), "user should have minted tokens")

	// Create destination payload (address is valid bech32)
	destinationPayload := "devcore1cz8x502s930v0ux8m6lpfw6s3l5tydz3gsx87w" + bridgeCfg.ChainID

	// Bridge
	bridgeAmount := tokensToWei(10) // 10 tokens
	t.Logf("Bridging %s tokens to %s", bridgeAmount.String(), destinationPayload)

	bridgeTx, err := evm.Bridge(
		client,
		userKey,
		anvil.ChainID(),
		contracts.Bridge,
		bridgeAmount,
		destinationPayload,
	)
	requireT.NoError(err)
	t.Logf("Bridge transaction: %s", bridgeTx.TxHash.Hex())

	// Create scanner
	scannerCfg := bnb.ScannerConfig{
		RPCURL:        anvil.RPCURL(),
		BridgeAddress: contracts.BridgeAddress,
		StartBlock:    0,
		PollInterval:  500 * time.Millisecond,
		Confirmations: 0,
		ChainSuffix:   bridgeCfg.ChainID,
	}

	scanner, err := bnb.NewScanner(scannerCfg, logger)
	requireT.NoError(err)

	// Subscribe to events
	eventCh := make(chan *bnbabi.TxBridgeBridgeInitiated, 10)
	scanCtx, scanCancel := context.WithTimeout(ctx, 10*time.Second)
	defer scanCancel()

	err = scanner.Subscribe(scanCtx, eventCh)
	requireT.NoError(err)

	// Wait for event
	t.Log("Waiting for BridgeInitiated event...")
	select {
	case event := <-eventCh:
		t.Logf("Received event: from=%s, amount=%s, payload=%s",
			event.From.Hex(), event.Amount.String(), event.DestinationPayload)
		requireT.Equal(userAddr, event.From, "event should be from user")
		requireT.Equal(bridgeAmount.String(), event.Amount.String(), "amount should match")
		requireT.Equal(destinationPayload, event.DestinationPayload, "payload should match")
	case <-scanCtx.Done():
		t.Fatal("timeout waiting for bridge event")
	}

	t.Log("Live scanner test passed!")
}

// TestBNBLiveEndToEnd tests the complete flow: EVM bridge tx -> TX Chain bank send.
func TestBNBLiveEndToEnd(t *testing.T) {
	ctx, chains := NewTestingContext(t)
	requireT := require.New(t)
	txChain := chains.TX
	logger := zaptest.NewLogger(t)

	// Start Anvil
	t.Log("Starting Anvil...")
	anvil, err := evm.StartAnvil(evm.DefaultAnvilConfig())
	requireT.NoError(err)
	t.Cleanup(func() {
		_ = anvil.Stop()
	})

	client, err := anvil.Client()
	requireT.NoError(err)

	// Get deployer private key
	deployerKey, err := evm.GetPrivateKey(0)
	requireT.NoError(err)

	// Configure bridge (same config for all tests)
	bridgeCfg := evm.DefaultBridgeConfig()

	// Deploy EVM contracts
	t.Log("Deploying TXToken and TXBridge contracts...")
	contracts, err := evm.SetupBridgeEnvironment(client, deployerKey, anvil.ChainID(), bridgeCfg)
	requireT.NoError(err)
	t.Logf("Token: %s, Bridge: %s", contracts.TokenAddress.Hex(), contracts.BridgeAddress.Hex())

	// Setup TX Chain side
	owner := txChain.TXChain.GenAccount()
	trustedAddress1 := txChain.TXChain.GenAccount()
	trustedAddress2 := txChain.TXChain.GenAccount()

	t.Log("Funding TX Chain accounts...")
	txChain.TXChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress1, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress2, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
	)

	// Deploy CosmWasm contract
	contractClient := tx.NewContractClient(tx.DefaultContractClientConfig(nil, ""), txChain.TXChain.ClientContext)

	trustedAddresses := []string{
		trustedAddress1.String(),
		trustedAddress2.String(),
	}

	t.Log("Deploying CosmWasm smart contract...")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, tx.DeployAndInstantiateConfig{
		Owner:            owner.String(),
		Admin:            owner.String(),
		TrustedAddresses: trustedAddresses,
		Threshold:        2,
		MinAmount:        sdkmath.NewIntFromUint64(100),
		MaxAmount:        sdkmath.NewIntFromUint64(200_000_000),
		XRPLTokens:       []tx.XRPLToken{},
		Label:            "bnb_live_bridge_test",
	})
	requireT.NoError(err)

	// Fund the contract
	coinToFundContract := txChain.TXChain.NewCoin(sdkmath.NewInt(10_000_000_000))
	txChain.TXChain.Faucet.FundAccounts(ctx, t, integration.NewFundedAccount(contractAddr, coinToFundContract))
	requireT.NoError(contractClient.SetContractAddress(contractAddr))

	// Create recipient on TX Chain
	recipientAddress := txChain.TXChain.GenAccount()
	t.Logf("Recipient address: %s", recipientAddress.String())

	// Get EVM user key
	userKey, err := evm.GetPrivateKey(1)
	requireT.NoError(err)
	userAddr := crypto.PubkeyToAddress(userKey.PublicKey)

	// Mint tokens to EVM user
	mintAmount := tokensToWei(100) // 100 tokens
	err = evm.MintTokens(client, deployerKey, anvil.ChainID(), contracts.Token, userAddr, mintAmount)
	requireT.NoError(err)

	// Submit bridge transaction on EVM
	bridgeAmount := tokensToWei(50) // 50 tokens
	destinationPayload := recipientAddress.String() + bridgeCfg.ChainID

	t.Logf("Bridging %s tokens to %s", bridgeAmount.String(), recipientAddress.String())
	bridgeTx, err := evm.Bridge(
		client,
		userKey,
		anvil.ChainID(),
		contracts.Bridge,
		bridgeAmount,
		destinationPayload,
	)
	requireT.NoError(err)
	t.Logf("EVM bridge tx: %s", bridgeTx.TxHash.Hex())

	// Create the scanner
	scanner, err := bnb.NewScanner(bnb.ScannerConfig{
		RPCURL:        anvil.RPCURL(),
		BridgeAddress: contracts.BridgeAddress,
		StartBlock:    0,
		PollInterval:  500 * time.Millisecond,
		Confirmations: 0,
		ChainSuffix:   bridgeCfg.ChainID,
	}, logger)
	requireT.NoError(err)

	// Build and start executors with scanner (2 executors, threshold=2)
	instances := buildAndStartBNBLiveExecutors(
		ctx, t, txChain, contractAddr,
		[]sdk.AccAddress{trustedAddress1, trustedAddress2},
		scanner,
		bridgeCfg.ChainID,
	)

	// wait tx to be processed
	// 50 tokens with 18 decimals -> 50_000_000 with 6 decimals
	expectedBalance := txChain.TXChain.NewCoin(sdkmath.NewInt(50_000_000))
	awaitForBalance(ctx, t, txChain.TXChain.ClientContext, recipientAddress.String(), expectedBalance)

	t.Log("BNB live end-to-end bridge test passed!")

	t.Cleanup(func() {
		// to fully complete processing before canceling context
		time.Sleep(2 * time.Second)
		for _, instance := range instances {
			instance.cancel()
		}
	})
}

// TestBNBLiveMultipleTransactions tests multiple bridge transactions through the live flow.
func TestBNBLiveMultipleTransactions(t *testing.T) {
	ctx, chains := NewTestingContext(t)
	requireT := require.New(t)
	txChain := chains.TX
	logger := zaptest.NewLogger(t)

	// Start Anvil
	t.Log("Starting Anvil...")
	anvil, err := evm.StartAnvil(evm.DefaultAnvilConfig())
	requireT.NoError(err)
	t.Cleanup(func() {
		_ = anvil.Stop()
	})

	client, err := anvil.Client()
	requireT.NoError(err)

	// Get deployer key
	deployerKey, err := evm.GetPrivateKey(0)
	requireT.NoError(err)

	// Configure bridge
	bridgeCfg := evm.DefaultBridgeConfig()

	// Deploy EVM contracts
	t.Log("Deploying EVM contracts...")
	contracts, err := evm.SetupBridgeEnvironment(client, deployerKey, anvil.ChainID(), bridgeCfg)
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
		Label:            "bnb_live_multi_test",
	})
	requireT.NoError(err)

	coinToFundContract := txChain.TXChain.NewCoin(sdkmath.NewInt(50_000_000_000))
	txChain.TXChain.Faucet.FundAccounts(ctx, t, integration.NewFundedAccount(contractAddr, coinToFundContract))
	requireT.NoError(contractClient.SetContractAddress(contractAddr))

	// Create recipients
	recipient1 := txChain.TXChain.GenAccount()
	recipient2 := txChain.TXChain.GenAccount()

	// Get multiple EVM user keys
	user1Key, err := evm.GetPrivateKey(1)
	requireT.NoError(err)
	user1Addr := crypto.PubkeyToAddress(user1Key.PublicKey)

	user2Key, err := evm.GetPrivateKey(2)
	requireT.NoError(err)
	user2Addr := crypto.PubkeyToAddress(user2Key.PublicKey)

	// Mint tokens to users
	mintAmount := tokensToWei(100) // 100 tokens each
	err = evm.MintTokens(client, deployerKey, anvil.ChainID(), contracts.Token, user1Addr, mintAmount)
	requireT.NoError(err)
	err = evm.MintTokens(client, deployerKey, anvil.ChainID(), contracts.Token, user2Addr, mintAmount)
	requireT.NoError(err)

	// Create scanner and start executors
	scanner, err := bnb.NewScanner(bnb.ScannerConfig{
		RPCURL:        anvil.RPCURL(),
		BridgeAddress: contracts.BridgeAddress,
		StartBlock:    0,
		PollInterval:  500 * time.Millisecond,
		Confirmations: 0,
		ChainSuffix:   bridgeCfg.ChainID,
	}, logger)
	requireT.NoError(err)

	instances := buildAndStartBNBLiveExecutors(
		ctx, t, txChain, contractAddr,
		[]sdk.AccAddress{trustedAddress1, trustedAddress2},
		scanner,
		bridgeCfg.ChainID,
	)

	// Bridge transaction 1: 30 tokens from user1 to recipient1
	_, err = evm.Bridge(
		client, user1Key, anvil.ChainID(),
		contracts.Bridge,
		tokensToWei(30),
		recipient1.String()+bridgeCfg.ChainID,
	)
	requireT.NoError(err)
	t.Log("Submitted bridge tx 1: 30 tokens to recipient1")

	// Bridge transaction 2: 45 tokens from user2 to recipient2
	_, err = evm.Bridge(
		client, user2Key, anvil.ChainID(),
		contracts.Bridge,
		tokensToWei(45),
		recipient2.String()+bridgeCfg.ChainID,
	)
	requireT.NoError(err)
	t.Log("Submitted bridge tx 2: 45 tokens to recipient2")

	// Bridge transaction 3: 20 tokens from user1 to recipient1
	_, err = evm.Bridge(
		client, user1Key, anvil.ChainID(),
		contracts.Bridge,
		tokensToWei(20),
		recipient1.String()+bridgeCfg.ChainID,
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
func buildAndStartBNBLiveExecutors(
	ctx context.Context,
	t *testing.T,
	txChain TXChain,
	contractAddr sdk.AccAddress,
	trustedAddresses []sdk.AccAddress,
	scanner *bnb.Scanner,
	chainSuffix string,
) []*executorInstance {
	t.Helper()

	logger := zaptest.NewLogger(t)
	instances := make([]*executorInstance, 0, len(trustedAddresses))

	executionErrors := make([]error, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, trustedAddr := range trustedAddresses {
		// Create BNB finder with scanner
		bnbFinder := finder.NewBNBFinder(
			finder.BNBFinderConfig{
				ChainSuffix: chainSuffix,
				TXDenom:     txChain.TXChain.ChainSettings.Denom,
				TXDecimals:  6,
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
			[]executor.Finder{bnbFinder},
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
