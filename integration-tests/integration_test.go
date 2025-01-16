//go:build integrationtests

package integrationtests

import (
	"context"
	"encoding/hex"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"golang.org/x/exp/slices"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/coreum/v4/pkg/client"
	"github.com/CoreumFoundation/coreum/v4/testutil/integration"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/service"
)

const (
	xrplTestMemoSuffix = "/integration-test"
	xrplCORECurrency   = "434F524500000000000000000000000000000000"
	xrplXCORECurrency  = "58434F5245000000000000000000000000000000"
)

func TestXRPLToCoreumBridging(t *testing.T) {
	t.Parallel()

	ctx, chains := NewTestingContext(t)
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	coreumChain := chains.Coreum

	bankClient := banktypes.NewQueryClient(coreumChain.ClientContext)

	// create XRPL coreum token to use for the test
	coreIssuer := xrplChain.GenAccount(ctx, t, 10)
	coreCurrency, err := rippledata.NewCurrency(xrplCORECurrency)
	requireT.NoError(err)

	xCoreIssuer := xrplChain.GenAccount(ctx, t, 10)
	xCoreCurrency, err := rippledata.NewCurrency(xrplXCORECurrency)
	requireT.NoError(err)

	enableDefaultRippling(ctx, t, chains, []rippledata.Account{coreIssuer, xCoreIssuer})

	xrplSender := prepareXRPLSender(ctx, t, xrplChain, coreIssuer, coreCurrency, xCoreIssuer, xCoreCurrency)

	owner := coreumChain.GenAccount()
	trustedAddress1 := coreumChain.GenAccount()
	trustedAddress2 := coreumChain.GenAccount()
	trustedAddress3 := coreumChain.GenAccount()

	recipient1Address := coreumChain.GenAccount()
	recipient2Address := coreumChain.GenAccount()
	recipient3Address := coreumChain.GenAccount()

	type testPayment struct {
		address  string
		amounts  []string
		issuer   rippledata.Account
		currency rippledata.Currency
	}

	// sending payments to the issuer with the memo of the recipients
	for _, p := range []testPayment{
		{
			address:  recipient1Address.String(),
			amounts:  []string{"30.0", "12.5999999999"},
			issuer:   xCoreIssuer,
			currency: xCoreCurrency,
		},
		{
			address:  recipient2Address.String(),
			amounts:  []string{"42.345"},
			issuer:   coreIssuer,
			currency: coreCurrency,
		},
		{
			address:  recipient1Address.String(),
			amounts:  []string{"150.0", "7.654321"},
			issuer:   coreIssuer,
			currency: coreCurrency,
		},
		{
			address:  recipient2Address.String(),
			amounts:  []string{"3.1"},
			issuer:   xCoreIssuer,
			currency: xCoreCurrency,
		},
		{
			address:  recipient3Address.String(),
			amounts:  []string{"15.0", "250.0", "7.0"},
			issuer:   coreIssuer,
			currency: coreCurrency,
		},
	} {
		for _, amt := range p.amounts {
			valueToPay, err := rippledata.NewValue(amt, false)
			requireT.NoError(err)
			paymentTx := rippledata.Payment{
				Destination: p.issuer,
				Amount: rippledata.Amount{
					Value:    valueToPay,
					Issuer:   p.issuer,
					Currency: p.currency,
				},
				TxBase: rippledata.TxBase{
					TransactionType: rippledata.PAYMENT,
					Memos: rippledata.Memos{
						rippledata.Memo{
							Memo: rippledata.MemoItem{
								MemoData: []byte(p.address + xrplTestMemoSuffix),
							},
						},
					},
				},
			}
			requireT.NoError(xrplChain.AutoFillSignAndSubmitTx(ctx, t, &paymentTx, xrplSender))
		}
	}

	balanceRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipient1Address.String(),
		Denom:   coreumChain.Chain.ChainSettings.Denom,
	})
	requireT.NoError(err)
	requireT.True(balanceRes.Balance.IsZero())

	balanceRes, err = bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipient2Address.String(),
		Denom:   coreumChain.Chain.ChainSettings.Denom,
	})
	requireT.NoError(err)
	requireT.True(balanceRes.Balance.IsZero())

	coreumChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, coreumChain.NewCoin(sdk.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress1, coreumChain.NewCoin(sdk.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress2, coreumChain.NewCoin(sdk.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress3, coreumChain.NewCoin(sdk.NewInt(5000000000))),
	)

	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), coreumChain.ClientContext)

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

	coinToFundContract := coreumChain.NewCoin(sdk.NewInt(10_000_000_000))
	coreumChain.Faucet.FundAccounts(ctx, t, integration.NewFundedAccount(contractAddr, coinToFundContract))

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	log := zaptest.NewLogger(t)

	instances := lo.Map(
		[]sdk.AccAddress{trustedAddress1, trustedAddress2, trustedAddress3},
		func(trustedAddress sdk.AccAddress, index int) *service.Services {
			return buildTestingServices(
				t,
				log,
				coreumChain.ChainSettings.ChainID,
				coreumChain.ClientContext.Keyring(),
				coreumChain.Config().RPCAddress,
				coreumChain.Config().GRPCAddress,
				xrplChain.Config().RPCAddress,
				coreIssuer, coreCurrency,
				xCoreIssuer, xCoreCurrency,
				trustedAddress,
				contractAddr,
			)
		},
	)

	executionErrors := make([]error, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(instances))
	for _, instance := range instances {
		go func(instance *service.Services) {
			defer wg.Done()
			if err := instance.Executor.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				mu.Lock()
				executionErrors = append(executionErrors, err)
				mu.Unlock()
			}
		}(instance)
	}

	awaitForBalance(
		ctx, t, coreumChain.ClientContext, recipient1Address.String(), coreumChain.NewCoin(
			sdk.NewInt(
				// xcore
				30000000+12599999+
					// core
					150000000+7654321,
			)),
	)

	awaitForBalance(
		ctx, t, coreumChain.ClientContext, recipient2Address.String(), coreumChain.NewCoin(
			sdk.NewInt(
				// core
				42345000+
					// xcore
					3100000),
		),
	)
	// the third sender includes the low and high amount checks, the low amount will be skipped the high will be locked
	// in the pending transactions. We use multiple amounts here since the low and high amounts are between
	// the transactions with the valid amounts.
	recipient3ExpectedBalance := sdk.NewInt(
		// core
		15000000 + 7000000,
	)
	awaitForBalance(
		ctx, t, coreumChain.ClientContext, recipient3Address.String(), coreumChain.NewCoin(recipient3ExpectedBalance),
	)

	// check that one transaction is pending due to amount limit
	highAmount := coreumChain.NewCoin(sdk.NewInt(250000000))

	pendingTxs, err := instances[0].CoreumContractClient.GetPendingTxs(ctx, nil, nil)
	require.NoError(t, err)
	require.Len(t, pendingTxs, 1)

	highAmountPendingTx := pendingTxs[0]
	expectedHighAmountPendingTx := coreum.Transaction{
		Amount:    highAmount,
		Recipient: recipient3Address.String(),
		EvidenceProviders: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
	}
	slices.Sort(expectedHighAmountPendingTx.EvidenceProviders)
	slices.Sort(highAmountPendingTx.EvidenceProviders)
	requireT.Equal(expectedHighAmountPendingTx, highAmountPendingTx.Transaction)

	// execute the pending transaction
	_, err = instances[0].CoreumContractClient.ExecutePending(
		ctx, trustedAddress1, highAmount, highAmountPendingTx.EvidenceID,
	)
	requireT.NoError(err)
	awaitForBalance(
		ctx,
		t,
		coreumChain.ClientContext,
		recipient3Address.String(),
		coreumChain.NewCoin(recipient3ExpectedBalance.Add(highAmount.Amount)),
	)

	cancel()
	wg.Wait()

	requireT.Empty(executionErrors)
	// validate that no error where produced
	for _, instance := range instances {
		totalErrors, err := instance.MetricRecorder.GetTotalErrors()
		requireT.NoError(err)
		requireT.Zero(totalErrors)
	}

	auditCtx, auditCtxCancel := context.WithTimeout(context.Background(), time.Minute)
	defer auditCtxCancel()
	discrepancies, err := instances[0].Auditor.Audit(auditCtx)
	requireT.NoError(err)
	requireT.Empty(discrepancies)
}

func enableDefaultRippling(
	ctx context.Context,
	t *testing.T,
	chains Chains,
	accounts []rippledata.Account,
) {
	requireT := require.New(t)
	for _, acc := range accounts {
		requireT.NoError(chains.XRPL.AutoFillSignAndSubmitTx(ctx, t, &rippledata.AccountSet{
			SetFlag: lo.ToPtr(uint32(rippledata.TxDefaultRipple)),
			TxBase: rippledata.TxBase{
				TransactionType: rippledata.ACCOUNT_SET,
			},
		}, acc))
	}
}

func prepareXRPLSender(
	ctx context.Context,
	t *testing.T,
	xrplChain XRPLChain,
	coreIssuer rippledata.Account, coreCurrency rippledata.Currency,
	xCoreIssuer rippledata.Account, xCoreCurrency rippledata.Currency,
) rippledata.Account {
	t.Log("Preparing XRPL sender")

	xrplSender := xrplChain.GenAccount(ctx, t, 10)
	requireT := require.New(t)

	valueToFund, err := rippledata.NewValue("1000000000000", false)
	requireT.NoError(err)

	for _, v := range []lo.Tuple2[rippledata.Account, rippledata.Currency]{
		{A: coreIssuer, B: coreCurrency},
		{A: xCoreIssuer, B: xCoreCurrency},
	} {
		trustSetForCoreumTx := rippledata.TrustSet{
			LimitAmount: rippledata.Amount{
				Value:    valueToFund,
				Issuer:   v.A,
				Currency: v.B,
			},
			TxBase: rippledata.TxBase{
				TransactionType: rippledata.TRUST_SET,
				Flags:           lo.ToPtr(rippledata.TxSetNoRipple),
			},
		}
		requireT.NoError(xrplChain.AutoFillSignAndSubmitTx(ctx, t, &trustSetForCoreumTx, xrplSender))

		fundXRPLSenderTx := rippledata.Payment{
			Destination: xrplSender,
			Amount: rippledata.Amount{
				Value:    valueToFund,
				Issuer:   v.A,
				Currency: v.B,
			},
			TxBase: rippledata.TxBase{
				TransactionType: rippledata.PAYMENT,
			},
		}
		requireT.NoError(xrplChain.AutoFillSignAndSubmitTx(ctx, t, &fundXRPLSenderTx, v.A))
	}

	return xrplSender
}

func buildTestingServices(
	t *testing.T,
	zapLogger *zap.Logger,
	chainID string,
	kr keyring.Keyring,
	coreumRPCURL, coreumGRPCURL, xrplRPCAddress string,
	coreIssuer rippledata.Account, coreCurrency rippledata.Currency,
	xCoreIssuer rippledata.Account, xCoreCurrency rippledata.Currency,
	senderAddress, contractAddress sdk.AccAddress,
) *service.Services {
	services, err := service.NewServices(service.Config{
		XRPLRPCURL:                    xrplRPCAddress,
		XRPLHistoryScanStartLedger:    0,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 0,
		XRPLTokens: []service.XRPLTokenConfig{
			{
				XRPLIssuer:   coreIssuer.String(),
				XRPLCurrency: convertCurrencyToString(coreCurrency),
			},
			{
				XRPLIssuer:   xCoreIssuer.String(),
				XRPLCurrency: convertCurrencyToString(xCoreCurrency),
			},
		},
		XRPLMemoSuffix: xrplTestMemoSuffix,
		// we don't use the chain ctx here intentionally to fully check the client initialisation
		CoreumRPCURL:          coreumRPCURL,
		CoreumGRPCURL:         coreumGRPCURL,
		CoreumChainID:         chainID,
		CoreumSenderAddress:   senderAddress.String(),
		CoreumContractAddress: contractAddress.String(),
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
			return retry.Retryable(
				errors.Errorf(
					"account %s %s balance is still not equal to expected, all balances: %s",
					address, expectedBalance, balancesRes,
				),
			)
		}

		return nil
	}))

	t.Logf("Received expected balance of %s.", expectedBalance.Denom)
}

func convertCurrencyToString(currency rippledata.Currency) string {
	currencyString := currency.String()
	if len(currencyString) == 3 {
		return currencyString
	}
	hexString := hex.EncodeToString([]byte(currencyString))
	// append tailing zeros to match the contract expectation
	hexString += strings.Repeat("0", 40-len(hexString))
	return strings.ToUpper(hexString)
}
