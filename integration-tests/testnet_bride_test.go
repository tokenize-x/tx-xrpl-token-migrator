//go:build integrationtests

package integrationtests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/exp/slices"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	integrationtests "github.com/CoreumFoundation/coreum/integration-tests"
	"github.com/CoreumFoundation/coreum/pkg/client"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/service"
)

func TestWASMTestnetBridging(t *testing.T) {
	t.Parallel()

	ctx, chain := integrationtests.NewCoreumTestingContext(t)
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()
	trustedAddress2 := chain.GenAccount()
	trustedAddress3 := chain.GenAccount()

	recipient1Address := "devcore1k0vuxw2d835u56u64rerjfnkgdpm88n2zl596z"
	recipient2Address := "devcore1ppc3az9z429hflver2gj8ervnlgx2s7gued0cs"
	recipient3Address := "devcore15rteqcut6w2vr3edhwqu8eh0ncv30j05sj7jck"

	requireT := require.New(t)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)

	balanceRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipient1Address,
		Denom:   chain.Chain.ChainSettings.Denom,
	})
	requireT.NoError(err)
	requireT.True(balanceRes.Balance.IsZero())

	balanceRes, err = bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipient2Address,
		Denom:   chain.Chain.ChainSettings.Denom,
	})
	requireT.NoError(err)
	requireT.True(balanceRes.Balance.IsZero())

	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress2, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress3, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), chain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold: 2,
		MinAmount: sdk.NewInt(100),
		MaxAmount: sdk.NewInt(200_000_000),
		Label:     "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000_000_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(contractAddr, coinToFundContract))

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	log := zaptest.NewLogger(t)
	trustedAddress1Service := buildTestingServices(t, log, chain.ChainSettings.ChainID, chain.ClientContext.Keyring(), trustedAddress1, contractAddr)
	trustedAddress2Service := buildTestingServices(t, log, chain.ChainSettings.ChainID, chain.ClientContext.Keyring(), trustedAddress2, contractAddr)
	trustedAddress3Service := buildTestingServices(t, log, chain.ChainSettings.ChainID, chain.ClientContext.Keyring(), trustedAddress3, contractAddr)

	startFunctions := []func(context.Context) error{
		trustedAddress1Service.Executor.Start,
		trustedAddress2Service.Executor.Start,
		trustedAddress3Service.Executor.Start,
	}
	executionErrors := make([]error, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(startFunctions))
	for _, f := range startFunctions {
		go func(f func(context.Context) error) {
			defer wg.Done()
			if err := f(ctx); err != nil && !errors.Is(err, context.Canceled) {
				mu.Lock()
				executionErrors = append(executionErrors, err)
				mu.Unlock()
			}
		}(f)
	}

	awaitForBalance(ctx, t, chain.ClientContext, recipient1Address, chain.NewCoin(sdk.NewInt(150000000+7654321)))
	awaitForBalance(ctx, t, chain.ClientContext, recipient2Address, chain.NewCoin(sdk.NewInt(42345679)))
	// the third sender includes the low and high amount checks, the low amount will be skipped the high will be locked
	// in the pending transactions. We use multiple amounts here since the low and high amounts are between
	// the transactions with the valid amounts.
	recipient3ExpectedBalance := sdk.NewInt(15000000 + 30000000 + 7000000)
	awaitForBalance(ctx, t, chain.ClientContext, recipient3Address, chain.NewCoin(recipient3ExpectedBalance))

	// check that one transaction is pending due to amount limit
	const highAmountTxHash = "798C6F9FF80B794F869242529423CB43FCA8A3CF456AF2141B3D7AEE999A6165"
	highAmount := chain.NewCoin(sdk.NewInt(250000000))
	highAmountTxEvidenceID := strings.ToLower(fmt.Sprintf("%s-%s-%s", highAmountTxHash, highAmount, recipient3Address))
	highAmountPendingTx, err := trustedAddress1Service.CoreumContractClient.GetPendingTx(ctx, highAmountTxEvidenceID)
	requireT.NoError(err)
	expectedHighAmountPendingTx := coreum.Transaction{
		Amount:    highAmount,
		Recipient: recipient3Address,
		EvidenceProviders: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
	}
	slices.Sort(expectedHighAmountPendingTx.EvidenceProviders)
	slices.Sort(highAmountPendingTx.EvidenceProviders)
	requireT.Equal(expectedHighAmountPendingTx, highAmountPendingTx)

	// execute the pending transaction
	_, err = trustedAddress1Service.CoreumContractClient.ExecutePending(ctx, trustedAddress1, highAmount, highAmountTxEvidenceID)
	requireT.NoError(err)
	awaitForBalance(ctx, t, chain.ClientContext, recipient3Address, chain.NewCoin(recipient3ExpectedBalance.Add(highAmount.Amount)))

	cancel()
	wg.Wait()

	requireT.Empty(executionErrors)
}

func buildTestingServices(
	t *testing.T,
	zapLogger *zap.Logger,
	chainID string,
	kr keyring.Keyring,
	senderAddress, contractAddress sdk.AccAddress,
) *service.Services {
	const XRPLRPCURLEnvVariableName = "XRPL_TESTNET_RPC_URL"
	rpcURL := os.Getenv(XRPLRPCURLEnvVariableName)
	if rpcURL == "" {
		t.Fatalf("It is required to set %q env variable to use that function", XRPLRPCURLEnvVariableName)
	}

	services, err := service.NewServices(service.Config{
		XRPLRPCURL:                 rpcURL,
		XRPLHistoryScanStartLedger: 38500000,
		XRPLRecentScanIndexesBack:  30_000,
		XRPLAccount:                "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA",
		XRPLCurrency:               "434F524500000000000000000000000000000000",
		XRPLIssuer:                 "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA",
		XRPLMemoSuffix:             "/integration-test",
		CoreumGRPCURL:              "http://localhost:9090", // we don't use the chain ctx here intentionally to fully check the client initialisation
		CoreumChainID:              chainID,
		CoreumSenderAddress:        senderAddress.String(),
		CoreumContractAddress:      contractAddress.String(),
	}, kr, true, zapLogger)
	require.NoError(t, err)

	return services
}

func awaitForBalance(
	ctx context.Context,
	t *testing.T,
	clientCtx client.Context,
	address string,
	expectedBalance sdk.Coin,
) {
	t.Helper()

	t.Logf("Waiting for account %s balance, expected amount: %s.", address, expectedBalance.String())
	bankClient := banktypes.NewQueryClient(clientCtx)
	retryCtx, retryCancel := context.WithTimeout(ctx, time.Minute)
	defer retryCancel()
	require.NoError(t, retry.Do(retryCtx, time.Second, func() error {
		requestCtx, requestCancel := context.WithTimeout(retryCtx, 5*time.Second)
		defer requestCancel()

		// We intentionally query all balances instead of single denom here to include this info inside error message.
		balancesRes, err := bankClient.AllBalances(requestCtx, &banktypes.QueryAllBalancesRequest{
			Address: address,
		})
		if err != nil {
			return err
		}

		if balancesRes.Balances.AmountOf(expectedBalance.Denom).String() != expectedBalance.Amount.String() {
			return retry.Retryable(errors.Errorf("account %s %s balance is still not equal to expected, all balances: %s", address, expectedBalance, balancesRes))
		}

		return nil
	}))

	t.Logf("Received expected balance of %s.", expectedBalance.Denom)
}
