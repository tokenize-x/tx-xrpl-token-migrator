//go:build integrationtests

package integrationtests

import (
	"context"
	"strconv"
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
	xrplCoreumCurrency = "434F524500000000000000000000000000000000"
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
	xrplIssuer := xrplChain.GenAccount(ctx, t, 10)
	coreumCurrency, err := rippledata.NewCurrency(xrplCoreumCurrency)
	requireT.NoError(err)

	// enable rippling on this account's trust lines by default.
	defaultRippleAccountSetForCoreumIssuerTx := rippledata.AccountSet{
		SetFlag: lo.ToPtr(uint32(rippledata.TxDefaultRipple)),
		TxBase: rippledata.TxBase{
			TransactionType: rippledata.ACCOUNT_SET,
		},
	}
	requireT.NoError(chains.XRPL.AutoFillSignAndSubmitTx(ctx, t, &defaultRippleAccountSetForCoreumIssuerTx, xrplIssuer))

	// fund the XRPL sender
	xrplSender := xrplChain.GenAccount(ctx, t, 10)

	toSendToXRPLSender, err := rippledata.NewValue("1000000000000", false)
	requireT.NoError(err)

	trustSetForCoreumTx := rippledata.TrustSet{
		LimitAmount: rippledata.Amount{
			Value:    toSendToXRPLSender,
			Currency: coreumCurrency,
			Issuer:   xrplIssuer,
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
			Value:    toSendToXRPLSender,
			Currency: coreumCurrency,
			Issuer:   xrplIssuer,
		},
		TxBase: rippledata.TxBase{
			TransactionType: rippledata.PAYMENT,
		},
	}
	requireT.NoError(xrplChain.AutoFillSignAndSubmitTx(ctx, t, &fundXRPLSenderTx, xrplIssuer))

	owner := coreumChain.GenAccount()
	trustedAddress1 := coreumChain.GenAccount()
	trustedAddress2 := coreumChain.GenAccount()
	trustedAddress3 := coreumChain.GenAccount()

	recipient1Address := coreumChain.GenAccount()
	recipient2Address := coreumChain.GenAccount()
	recipient3Address := coreumChain.GenAccount()

	type accountToPay struct {
		address string
		amounts []float64
	}

	// sending payments to the issuer with the memo of the recipients
	for _, accToPay := range []accountToPay{
		{
			address: recipient1Address.String(),
			amounts: []float64{150, 7.654321},
		},
		{
			address: recipient2Address.String(),
			amounts: []float64{42.345679},
		},
		{
			address: recipient3Address.String(),
			amounts: []float64{15.000000, 250.000000, 7.000000},
		},
	} {
		for _, amt := range accToPay.amounts {
			valueToPay, err := rippledata.NewValue(strconv.FormatFloat(amt, 'f', 6, 64), false)
			requireT.NoError(err)
			paymentTx := rippledata.Payment{
				Destination: xrplIssuer,
				Amount: rippledata.Amount{
					Value:    valueToPay,
					Currency: coreumCurrency,
					Issuer:   xrplIssuer,
				},
				TxBase: rippledata.TxBase{
					TransactionType: rippledata.PAYMENT,
					Memos: rippledata.Memos{
						rippledata.Memo{
							Memo: rippledata.MemoItem{
								MemoData: []byte(accToPay.address + xrplTestMemoSuffix),
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

	instances := []*service.Services{
		buildTestingServices(
			t,
			log,
			coreumChain.ChainSettings.ChainID,
			coreumChain.ClientContext.Keyring(),
			coreumChain.Config().RPCAddress,
			coreumChain.Config().GRPCAddress,
			xrplChain.Config().RPCAddress,
			xrplIssuer.String(),
			trustedAddress1,
			contractAddr,
		),
		buildTestingServices(
			t,
			log,
			coreumChain.ChainSettings.ChainID,
			coreumChain.ClientContext.Keyring(),
			coreumChain.Config().RPCAddress,
			coreumChain.Config().GRPCAddress,
			xrplChain.Config().RPCAddress,
			xrplIssuer.String(),
			trustedAddress2,
			contractAddr,
		),
		buildTestingServices(
			t,
			log,
			coreumChain.ChainSettings.ChainID,
			coreumChain.ClientContext.Keyring(),
			coreumChain.Config().RPCAddress,
			coreumChain.Config().GRPCAddress,
			xrplChain.Config().RPCAddress,
			xrplIssuer.String(),
			trustedAddress3,
			contractAddr,
		),
	}

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

	awaitForBalance(ctx, t, coreumChain.ClientContext, recipient1Address.String(), coreumChain.NewCoin(sdk.NewInt(150000000+7654321)))
	awaitForBalance(ctx, t, coreumChain.ClientContext, recipient2Address.String(), coreumChain.NewCoin(sdk.NewInt(42345679)))
	// the third sender includes the low and high amount checks, the low amount will be skipped the high will be locked
	// in the pending transactions. We use multiple amounts here since the low and high amounts are between
	// the transactions with the valid amounts.
	recipient3ExpectedBalance := sdk.NewInt(15000000 + 7000000)
	awaitForBalance(ctx, t, coreumChain.ClientContext, recipient3Address.String(), coreumChain.NewCoin(recipient3ExpectedBalance))

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
	_, err = instances[0].CoreumContractClient.ExecutePending(ctx, trustedAddress1, highAmount, highAmountPendingTx.EvidenceID)
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

func buildTestingServices(
	t *testing.T,
	zapLogger *zap.Logger,
	chainID string,
	kr keyring.Keyring,
	coreumRPCURL, coreumGRPCURL, xrplRPCAddress string,
	xrplIssuer string,
	senderAddress, contractAddress sdk.AccAddress,
) *service.Services {
	services, err := service.NewServices(service.Config{
		XRPLRPCURL:                    xrplRPCAddress,
		XRPLHistoryScanStartLedger:    0,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 0,
		XRPLAccount:                   xrplIssuer,
		XRPLCurrency:                  xrplCoreumCurrency,
		XRPLIssuer:                    xrplIssuer,
		XRPLMemoSuffix:                xrplTestMemoSuffix,
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
