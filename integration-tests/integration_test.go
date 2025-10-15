//go:build integrationtests

package integrationtests

import (
	"context"
	"sort"
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

type payment struct {
	address  string
	amounts  []string
	issuer   rippledata.Account
	currency rippledata.Currency
}

func TestXRPLToCoreumBridgingMultiTokenSending(t *testing.T) {
	t.Parallel()

	ctx, chains := NewTestingContext(t)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	coreumChain := chains.Coreum

	coreIssuer := xrplChain.GenAccount(ctx, t, 10)
	coreCurrency, err := rippledata.NewCurrency(xrplCORECurrency)
	requireT.NoError(err)

	xCoreIssuer := xrplChain.GenAccount(ctx, t, 10)
	xCoreCurrency, err := rippledata.NewCurrency(xrplXCORECurrency)
	requireT.NoError(err)

	soloIssuer := xrplChain.GenAccount(ctx, t, 10)
	soloCurrency, err := rippledata.NewCurrency(xrplSOLOCurrency)
	requireT.NoError(err)

	enableDefaultRippling(ctx, t, chains, coreIssuer, soloIssuer)

	tokens := []service.XRPLTokenConfig{
		{
			XRPLIssuer:     coreIssuer.String(),
			XRPLCurrency:   xrplCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:     xCoreIssuer.String(),
			XRPLCurrency:   xrplXCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:     soloIssuer.String(),
			XRPLCurrency:   xrplSOLOCurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.25",
		},
	}

	xrplSender := prepareXRPLSender(ctx, t, xrplChain, tokens)

	recipient1Address := coreumChain.GenAccount()
	recipient2Address := coreumChain.GenAccount()
	recipient3Address := coreumChain.GenAccount()

	sendPayments(ctx, t, xrplChain, xrplSender, []payment{
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
		{
			address:  recipient1Address.String(),
			amounts:  []string{"15.0", "250.0", "7.0"}, // Should receive 17.75 and 8.75 because of the 1.25 multiplier
			issuer:   soloIssuer,
			currency: soloCurrency,
		},
	})

	instances := buildAndStartDevEnv(ctx, t, chains, tokens)

	awaitForBalance(
		ctx, t, coreumChain.ClientContext, recipient1Address.String(), coreumChain.NewCoin(
			sdk.NewInt(
				// SOLO
				18750000+8750000+
					// XCORE
					30000000+12599999+
					// CORE
					150000000+7654321,
			)),
	)

	awaitForBalance(
		ctx, t, coreumChain.ClientContext, recipient2Address.String(), coreumChain.NewCoin(
			sdk.NewInt(
				// CORE
				42345000+
					// XCORE
					3100000),
		),
	)
	// the third sender includes the low and high amount checks, the low amount will be skipped the high will be locked
	// in the pending transactions. We use multiple amounts here since the low and high amounts are between
	// the transactions with the valid amounts.
	recipient3ExpectedBalance := sdk.NewInt(
		// CORE
		15000000 + 7000000,
	)
	awaitForBalance(
		ctx, t, coreumChain.ClientContext, recipient3Address.String(), coreumChain.NewCoin(recipient3ExpectedBalance),
	)

	// check that one xCore transaction is pending due to amount limit
	highAmount := coreumChain.NewCoin(sdk.NewInt(250000000))

	pendingTxs, err := instances[0].CoreumContractClient.GetPendingTxs(ctx, nil, nil)
	require.NoError(t, err)
	require.Len(t, pendingTxs, 2) // one xCore and one solo

	// sort pending transactions, so xCore with less amount would be the first, and solo would be the second
	sort.Slice(pendingTxs, func(i, j int) bool {
		return pendingTxs[i].Amount.IsLTE(pendingTxs[j].Amount)
	})

	highAmountPendingTx := pendingTxs[0]
	expectedHighAmountPendingTx := coreum.Transaction{
		Amount:    highAmount,
		Recipient: recipient3Address.String(),
		EvidenceProviders: lo.Map(instances, func(instance *service.Services, _ int) string {
			return instance.Config.CoreumSenderAddress
		}),
	}
	slices.Sort(expectedHighAmountPendingTx.EvidenceProviders)
	slices.Sort(highAmountPendingTx.EvidenceProviders)
	requireT.Equal(expectedHighAmountPendingTx, highAmountPendingTx.Transaction)

	// execute the pending transaction
	_, err = instances[0].CoreumContractClient.ExecutePending(
		ctx,
		sdk.MustAccAddressFromBech32(instances[0].Config.CoreumSenderAddress),
		highAmount,
		highAmountPendingTx.EvidenceID,
	)
	requireT.NoError(err)
	awaitForBalance(
		ctx,
		t,
		coreumChain.ClientContext,
		recipient3Address.String(),
		coreumChain.NewCoin(recipient3ExpectedBalance.Add(highAmount.Amount)),
	)

	// check that one solo transaction is pending due to amount limit
	highAmount = coreumChain.NewCoin(sdk.NewInt(312500000))

	highAmountPendingTx = pendingTxs[1]
	expectedHighAmountPendingTx = coreum.Transaction{
		Amount:    highAmount,
		Recipient: recipient1Address.String(),
		EvidenceProviders: lo.Map(instances, func(instance *service.Services, _ int) string {
			return instance.Config.CoreumSenderAddress
		}),
	}
	slices.Sort(expectedHighAmountPendingTx.EvidenceProviders)
	slices.Sort(highAmountPendingTx.EvidenceProviders)
	requireT.Equal(expectedHighAmountPendingTx, highAmountPendingTx.Transaction)

	// execute the pending transaction
	_, err = instances[0].CoreumContractClient.ExecutePending(
		ctx,
		sdk.MustAccAddressFromBech32(instances[0].Config.CoreumSenderAddress),
		highAmount,
		highAmountPendingTx.EvidenceID,
	)
	requireT.NoError(err)
}

func TestXRPLToCoreumBridgingTokenActivationDate(t *testing.T) {
	t.Parallel()

	ctx, chains := NewTestingContext(t)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	coreumChain := chains.Coreum

	coreIssuer := xrplChain.GenAccount(ctx, t, 10)
	coreCurrency, err := rippledata.NewCurrency(xrplCORECurrency)
	requireT.NoError(err)

	xCoreIssuer := xrplChain.GenAccount(ctx, t, 10)
	xCoreCurrency, err := rippledata.NewCurrency(xrplXCORECurrency)
	requireT.NoError(err)

	soloIssuer := xrplChain.GenAccount(ctx, t, 10)
	soloCurrency, err := rippledata.NewCurrency(xrplSOLOCurrency)
	requireT.NoError(err)

	enableDefaultRippling(ctx, t, chains, coreIssuer, soloIssuer)

	tokens := []service.XRPLTokenConfig{
		{
			XRPLIssuer:     coreIssuer.String(),
			XRPLCurrency:   xrplCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:   xCoreIssuer.String(),
			XRPLCurrency: xrplXCORECurrency,
			// the XCORE will be activated in the future
			ActivationDate: time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:   soloIssuer.String(),
			XRPLCurrency: xrplSOLOCurrency,
			// the SOLO will be activated in the future
			ActivationDate: time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.25",
		},
	}

	xrplSender := prepareXRPLSender(ctx, t, xrplChain, tokens)

	recipientAddress := coreumChain.GenAccount()

	sendPayments(ctx, t, xrplChain, xrplSender, []payment{
		{
			address:  recipientAddress.String(),
			amounts:  []string{"0.00000001", "30.0", "12.5999999999"},
			issuer:   xCoreIssuer,
			currency: xCoreCurrency,
		},
		{
			address:  recipientAddress.String(),
			amounts:  []string{"150.0"},
			issuer:   coreIssuer,
			currency: coreCurrency,
		},
		{
			address:  recipientAddress.String(),
			amounts:  []string{"10.0"},
			issuer:   xCoreIssuer,
			currency: xCoreCurrency,
		},
		{
			address:  recipientAddress.String(),
			amounts:  []string{"35.0"},
			issuer:   coreIssuer,
			currency: coreCurrency,
		},
		{
			address:  recipientAddress.String(),
			amounts:  []string{"0.00000001", "30.0", "12.5999999999"},
			issuer:   soloIssuer,
			currency: soloCurrency,
		},
	})

	buildAndStartDevEnv(ctx, t, chains, tokens)

	awaitForBalance(
		ctx, t, coreumChain.ClientContext, recipientAddress.String(), coreumChain.NewCoin(
			sdk.NewInt(
				// only CORE related balance is expected, the XCORE and SOLO are not activated
				150000000+35000000,
			)),
	)
}

func enableDefaultRippling(
	ctx context.Context,
	t *testing.T,
	chains Chains,
	accounts ...rippledata.Account,
) {
	requireT := require.New(t)

	requireT.NotEmpty(accounts)
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
	tokens []service.XRPLTokenConfig,
) rippledata.Account {
	t.Log("Preparing XRPL sender")

	xrplSender := xrplChain.GenAccount(ctx, t, 10)
	requireT := require.New(t)

	valueToFund, err := rippledata.NewValue("1000000000000", false)
	requireT.NoError(err)

	for _, token := range tokens {
		issuer, err := rippledata.NewAccountFromAddress(token.XRPLIssuer)
		requireT.NoError(err)

		currency, err := rippledata.NewCurrency(token.XRPLCurrency)
		requireT.NoError(err)

		trustSetForCoreumTx := rippledata.TrustSet{
			LimitAmount: rippledata.Amount{
				Value:    valueToFund,
				Issuer:   *issuer,
				Currency: currency,
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
				Issuer:   *issuer,
				Currency: currency,
			},
			TxBase: rippledata.TxBase{
				TransactionType: rippledata.PAYMENT,
			},
		}
		requireT.NoError(xrplChain.AutoFillSignAndSubmitTx(ctx, t, &fundXRPLSenderTx, *issuer))
	}

	return xrplSender
}

func sendPayments(
	ctx context.Context,
	t *testing.T,
	xrplChain XRPLChain,
	xrplSender rippledata.Account,
	payments []payment,
) {
	requireT := require.New(t)
	for _, p := range payments {
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
}

func buildAndStartDevEnv(
	ctx context.Context,
	t *testing.T,
	chains Chains,
	tokens []service.XRPLTokenConfig,
) []*service.Services {
	ctx, cancel := context.WithCancel(ctx)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	coreumChain := chains.Coreum

	owner := coreumChain.GenAccount()

	trustedAddress1 := coreumChain.GenAccount()
	trustedAddress2 := coreumChain.GenAccount()
	trustedAddress3 := coreumChain.GenAccount()

	t.Log("Funding trusted addresses.")
	coreumChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, coreumChain.NewCoin(sdk.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress1, coreumChain.NewCoin(sdk.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress2, coreumChain.NewCoin(sdk.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress3, coreumChain.NewCoin(sdk.NewInt(5000000000))),
	)

	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), coreumChain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	trustedAddresses := []string{
		trustedAddress1.String(),
		trustedAddress2.String(),
		trustedAddress3.String(),
	}
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner:            owner.String(),
		Admin:            owner.String(),
		TrustedAddresses: trustedAddresses,
		Threshold:        2,
		MinAmount:        sdk.NewInt(100),
		MaxAmount:        sdk.NewInt(200_000_000),
		XRPLTokens:       convertServiceTokensToContractTokens(tokens),
		Label:            "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := coreumChain.NewCoin(sdk.NewInt(10_000_000_000))
	coreumChain.Faucet.FundAccounts(ctx, t, integration.NewFundedAccount(contractAddr, coinToFundContract))

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	t.Log("Building and starting services.")
	instances := lo.Map(
		[]sdk.AccAddress{trustedAddress1, trustedAddress2, trustedAddress3},
		func(trustedAddress sdk.AccAddress, index int) *service.Services {
			return buildTestingServices(
				t,
				zaptest.NewLogger(t),
				coreumChain.ChainSettings.ChainID,
				coreumChain.ClientContext.Keyring(),
				coreumChain.Config().RPCAddress,
				coreumChain.Config().GRPCAddress,
				xrplChain.Config().RPCAddress,
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
				// the service will be terminated by the context canceled by the `t.Cleanup`
				mu.Lock()
				executionErrors = append(executionErrors, err)
				mu.Unlock()
			}
		}(instance)
	}

	t.Cleanup(func() {
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
	})

	return instances
}

func convertServiceTokensToContractTokens(tokens []service.XRPLTokenConfig) []coreum.XRPLToken {
	contractTokens := make([]coreum.XRPLToken, 0, len(tokens))
	for _, token := range tokens {
		contractTokens = append(contractTokens, coreum.XRPLToken{
			Currency:       token.XRPLCurrency,
			Issuer:         token.XRPLIssuer,
			ActivationDate: uint64(token.ActivationDate.Unix()),
			Multiplier:     token.Multiplier,
		})
	}
	return contractTokens
}

func buildTestingServices(
	t *testing.T,
	zapLogger *zap.Logger,
	chainID string,
	kr keyring.Keyring,
	coreumRPCURL, coreumGRPCURL, xrplRPCAddress string,
	senderAddress, contractAddress sdk.AccAddress,
) *service.Services {
	services, err := service.NewServices(context.Background(), service.Config{
		XRPLRPCURL:                    xrplRPCAddress,
		XRPLHistoryScanStartLedger:    0,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 0,
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

	t.Logf("Waiting for account %s balance, expected balance: %s.", address, expectedBalance.String())
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

	t.Logf("Received expected balance: %s.", expectedBalance)
}
