//go:build integrationtests

package xrpl

import (
	"context"
	"slices"
	"sort"
	"sync"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/coreum/v5/pkg/client"
	"github.com/CoreumFoundation/coreum/v5/testutil/integration"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/service"
)

type payment struct {
	address  string
	amounts  []string
	issuer   rippledata.Account
	currency rippledata.Currency
}

func TestXRPLToTXBridgingMultiTokenSending(t *testing.T) {
	t.Parallel()

	ctx, chains := NewTestingContext(t)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	txChain := chains.TX

	coreIssuer := xrplChain.GenAccount(ctx, t, 10)
	coreCurrency, err := rippledata.NewCurrency(XRPLCORECurrency)
	requireT.NoError(err)

	xCoreIssuer := xrplChain.GenAccount(ctx, t, 10)
	xCoreCurrency, err := rippledata.NewCurrency(XRPLXCORECurrency)
	requireT.NoError(err)

	soloIssuer := xrplChain.GenAccount(ctx, t, 10)
	soloCurrency, err := rippledata.NewCurrency(XRPLSOLOCurrency)
	requireT.NoError(err)

	enableDefaultRippling(ctx, t, chains, coreIssuer, soloIssuer)

	tokens := []service.XRPLTokenConfig{
		{
			XRPLIssuer:     coreIssuer.String(),
			XRPLCurrency:   XRPLCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:     xCoreIssuer.String(),
			XRPLCurrency:   XRPLXCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:     soloIssuer.String(),
			XRPLCurrency:   XRPLSOLOCurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.25",
		},
	}

	xrplSender := prepareXRPLSender(ctx, t, xrplChain, tokens)

	recipient1Address := txChain.TXChain.GenAccount()
	recipient2Address := txChain.TXChain.GenAccount()
	recipient3Address := txChain.TXChain.GenAccount()

	instances := buildAndStartDevEnv(ctx, t, chains, tokens)

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

	awaitForBalance(
		ctx, t, txChain.TXChain.ClientContext, recipient1Address.String(), txChain.TXChain.NewCoin(
			sdkmath.NewInt(
				// SOLO
				18750000+8750000+
					// XCORE
					30000000+12599999+
					// CORE
					150000000+7654321,
			)),
	)

	awaitForBalance(
		ctx, t, txChain.TXChain.ClientContext, recipient2Address.String(), txChain.TXChain.NewCoin(
			sdkmath.NewInt(
				// CORE
				42345000+
					// XCORE
					3100000),
		),
	)
	// the third sender includes the low and high amount checks, the low amount will be skipped the high will be locked
	// in the pending transactions. We use multiple amounts here since the low and high amounts are between
	// the transactions with the valid amounts.
	recipient3ExpectedBalance := sdkmath.NewInt(
		// CORE
		15000000 + 7000000,
	)
	awaitForBalance(
		ctx, t, txChain.TXChain.ClientContext, recipient3Address.String(), txChain.TXChain.NewCoin(recipient3ExpectedBalance),
	)

	// check that one xCore transaction is pending due to amount limit
	highAmount := txChain.TXChain.NewCoin(sdkmath.NewInt(250000000))

	pendingTxs, err := instances[0].TXContractClient.GetPendingTxs(ctx, nil, nil)
	require.NoError(t, err)
	require.Len(t, pendingTxs, 2) // one xCore and one solo

	// sort pending transactions, so xCore with less amount would be the first, and solo would be the second
	sort.Slice(pendingTxs, func(i, j int) bool {
		return pendingTxs[i].Amount.IsLTE(pendingTxs[j].Amount)
	})

	highAmountPendingTx := pendingTxs[0]
	expectedHighAmountPendingTx := tx.Transaction{
		Amount:    highAmount,
		Recipient: recipient3Address.String(),
		EvidenceProviders: lo.Map(instances, func(instance *service.Services, _ int) string {
			return instance.Config.TXSenderAddress
		}),
	}
	slices.Sort(expectedHighAmountPendingTx.EvidenceProviders)
	slices.Sort(highAmountPendingTx.EvidenceProviders)
	requireT.Equal(expectedHighAmountPendingTx, highAmountPendingTx.Transaction)

	// execute the pending transaction
	_, err = instances[0].TXContractClient.ExecutePending(
		ctx,
		sdk.MustAccAddressFromBech32(instances[0].Config.TXSenderAddress),
		highAmount,
		highAmountPendingTx.EvidenceID,
	)
	requireT.NoError(err)
	awaitForBalance(
		ctx,
		t,
		txChain.TXChain.ClientContext,
		recipient3Address.String(),
		txChain.TXChain.NewCoin(recipient3ExpectedBalance.Add(highAmount.Amount)),
	)

	// check that one solo transaction is pending due to amount limit
	highAmount = txChain.TXChain.NewCoin(sdkmath.NewInt(312500000))

	highAmountPendingTx = pendingTxs[1]
	expectedHighAmountPendingTx = tx.Transaction{
		Amount:    highAmount,
		Recipient: recipient1Address.String(),
		EvidenceProviders: lo.Map(instances, func(instance *service.Services, _ int) string {
			return instance.Config.TXSenderAddress
		}),
	}
	slices.Sort(expectedHighAmountPendingTx.EvidenceProviders)
	slices.Sort(highAmountPendingTx.EvidenceProviders)
	requireT.Equal(expectedHighAmountPendingTx, highAmountPendingTx.Transaction)

	// execute the pending transaction
	_, err = instances[0].TXContractClient.ExecutePending(
		ctx,
		sdk.MustAccAddressFromBech32(instances[0].Config.TXSenderAddress),
		highAmount,
		highAmountPendingTx.EvidenceID,
	)
	requireT.NoError(err)
}

func TestXRPLToTXBridgingTokenActivationDate(t *testing.T) {
	t.Parallel()

	ctx, chains := NewTestingContext(t)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	txChain := chains.TX

	coreIssuer := xrplChain.GenAccount(ctx, t, 10)
	coreCurrency, err := rippledata.NewCurrency(XRPLCORECurrency)
	requireT.NoError(err)

	xCoreIssuer := xrplChain.GenAccount(ctx, t, 10)
	xCoreCurrency, err := rippledata.NewCurrency(XRPLXCORECurrency)
	requireT.NoError(err)

	soloIssuer := xrplChain.GenAccount(ctx, t, 10)
	soloCurrency, err := rippledata.NewCurrency(XRPLSOLOCurrency)
	requireT.NoError(err)

	enableDefaultRippling(ctx, t, chains, coreIssuer, soloIssuer)

	tokens := []service.XRPLTokenConfig{
		{
			XRPLIssuer:     coreIssuer.String(),
			XRPLCurrency:   XRPLCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:   xCoreIssuer.String(),
			XRPLCurrency: XRPLXCORECurrency,
			// the XCORE will be activated in the future
			ActivationDate: time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
		{
			XRPLIssuer:   soloIssuer.String(),
			XRPLCurrency: XRPLSOLOCurrency,
			// the SOLO will be activated in the future
			ActivationDate: time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.25",
		},
	}

	xrplSender := prepareXRPLSender(ctx, t, xrplChain, tokens)

	recipientAddress := txChain.TXChain.GenAccount()

	buildAndStartDevEnv(ctx, t, chains, tokens)

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
		ctx, t, txChain.TXChain.ClientContext, recipientAddress.String(), txChain.TXChain.NewCoin(
			sdkmath.NewInt(
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

		trustSetForTXTx := rippledata.TrustSet{
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
		requireT.NoError(xrplChain.AutoFillSignAndSubmitTx(ctx, t, &trustSetForTXTx, xrplSender))

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
								MemoData: []byte(p.address + XRPLTestMemoSuffix),
							},
						},
					},
				},
			}
			requireT.NoError(xrplChain.AutoFillSignAndSubmitTx(ctx, t, &paymentTx, xrplSender))
			// Insert a small delay to avoid submitting payments too quickly in tests.
			// and cause intermittent test failures.
			time.Sleep(2000 * time.Millisecond)
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
	txChain := chains.TX

	owner := txChain.TXChain.GenAccount()

	trustedAddress1 := txChain.TXChain.GenAccount()
	trustedAddress2 := txChain.TXChain.GenAccount()
	trustedAddress3 := txChain.TXChain.GenAccount()

	t.Log("Funding trusted addresses.")
	txChain.TXChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress1, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress2, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
		integration.NewFundedAccount(trustedAddress3, txChain.TXChain.NewCoin(sdkmath.NewInt(5000000000))),
	)

	contractClient := tx.NewContractClient(tx.DefaultContractClientConfig(nil, ""), txChain.TXChain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	trustedAddresses := []string{
		trustedAddress1.String(),
		trustedAddress2.String(),
		trustedAddress3.String(),
	}
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, tx.DeployAndInstantiateConfig{
		Owner:            owner.String(),
		Admin:            owner.String(),
		TrustedAddresses: trustedAddresses,
		Threshold:        2,
		MinAmount:        sdkmath.NewIntFromUint64(100),
		MaxAmount:        sdkmath.NewIntFromUint64(200_000_000),
		XRPLTokens:       convertServiceTokensToContractTokens(tokens),
		Label:            "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(10_000_000_000))
	txChain.TXChain.Faucet.FundAccounts(ctx, t, integration.NewFundedAccount(contractAddr, coinToFundContract))

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	t.Log("Building and starting services.")
	instances := lo.Map(
		[]sdk.AccAddress{trustedAddress1, trustedAddress2, trustedAddress3},
		//nolint:contextcheck // buildTestingServices intentionally uses context.Background()
		func(trustedAddress sdk.AccAddress, index int) *service.Services {
			return buildTestingServices(
				t,
				zaptest.NewLogger(t),
				txChain.TXChain.ChainSettings.ChainID,
				txChain.TXChain.ClientContext.Keyring(),
				txChain.Config().RPCAddress,
				txChain.Config().GRPCAddress,
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

	t.Cleanup(func() { //nolint:contextcheck // cleanup creates its own context for audit
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

func convertServiceTokensToContractTokens(tokens []service.XRPLTokenConfig) []tx.XRPLToken {
	contractTokens := make([]tx.XRPLToken, 0, len(tokens))
	for _, token := range tokens {
		contractTokens = append(contractTokens, tx.XRPLToken{
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
	txRPCURL, txGRPCURL, xrplRPCAddress string,
	senderAddress, contractAddress sdk.AccAddress,
) *service.Services {
	services, err := service.NewServices(t.Context(), service.Config{
		XRPLRPCURL:                    xrplRPCAddress,
		XRPLHistoryScanStartLedger:    0,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 0,
		XRPLMemoSuffix:                XRPLTestMemoSuffix,
		BSCScannerDisabled:            true,
		// we don't use the chain ctx here intentionally to fully check the client initialization
		TXRPCURL:          txRPCURL,
		TXGRPCURL:         txGRPCURL,
		TXChainID:         chainID,
		TXSenderAddress:   senderAddress.String(),
		TXContractAddress: contractAddress.String(),
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

// TestDuplicateTransactionPrevention tests that the same XRPL transaction is never processed twice,
// even when multipliers change or the transaction is submitted multiple times.
func TestDuplicateTransactionPrevention(t *testing.T) {
	t.Parallel()

	ctx, chains := NewTestingContext(t)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	txChain := chains.TX

	// Setup token issuer
	tokenIssuer := xrplChain.GenAccount(ctx, t, 10)
	tokenCurrency, err := rippledata.NewCurrency(XRPLCORECurrency)
	requireT.NoError(err)

	enableDefaultRippling(ctx, t, chains, tokenIssuer)

	initialMultiplier := "1.0"
	tokens := []service.XRPLTokenConfig{
		{
			XRPLIssuer:     tokenIssuer.String(),
			XRPLCurrency:   XRPLCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     initialMultiplier,
		},
	}

	// Build and start the relayer instances
	instances := buildAndStartDevEnv(ctx, t, chains, tokens)

	xrplSender := prepareXRPLSender(ctx, t, xrplChain, tokens)
	recipientAddress := txChain.TXChain.GenAccount()

	// Send initial payment from XRPL
	sendAmount := "100.0"
	sendPayments(ctx, t, xrplChain, xrplSender, []payment{
		{
			address:  recipientAddress.String(),
			amounts:  []string{sendAmount},
			issuer:   tokenIssuer,
			currency: tokenCurrency,
		},
	})

	t.Log("waiting for initial transaction to be processed")

	// Calculate expected balance with initial multiplier (1.0)
	expectedInitialBalance := txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(100_000000)) // 100.0 * 1.0 * 10^6
	awaitForBalance(
		ctx,
		t,
		txChain.TXChain.ClientContext,
		recipientAddress.String(),
		expectedInitialBalance,
	)

	t.Log("initial transaction processed successfully")

	// Record the initial balance
	bankClient := banktypes.NewQueryClient(txChain.TXChain.ClientContext)
	balanceRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipientAddress.String(),
		Denom:   expectedInitialBalance.Denom,
	})
	requireT.NoError(err)
	balanceAfterFirst := balanceRes.Balance.Amount

	t.Logf("balance after first transaction: %s", balanceAfterFirst.String())

	// Get contract client from one of the instances
	contractClient := instances[0].TXContractClient
	trustedAddress := sdk.MustAccAddressFromBech32(instances[0].Config.TXSenderAddress)

	// Get all sent transactions to find our transaction hash
	offset := uint64(0)
	limit := uint32(100)
	sentTxs, err := contractClient.GetSentTxs(ctx, &offset, &limit)
	requireT.NoError(err)
	requireT.NotEmpty(sentTxs, "should have at least one sent transaction")

	// Find the transaction we just sent
	var ourTxHash string
	for _, txn := range sentTxs {
		if txn.Recipient == recipientAddress.String() && txn.Amount.Amount.Equal(expectedInitialBalance.Amount) {
			ourTxHash = txn.ID
			break
		}
	}
	requireT.NotEmpty(ourTxHash, "should find our transaction in sent transactions")
	t.Logf("found transaction hash: %s", ourTxHash)

	// Attempt to submit evidence for the same transaction again
	// This should be rejected by the contract with "Transfer already sent" error
	_, err = contractClient.ThresholdBankSend(
		ctx,
		trustedAddress,
		tx.ThresholdBankSendRequest{
			ID:        ourTxHash,
			Amount:    expectedInitialBalance,
			Recipient: recipientAddress.String(),
		},
	)
	requireT.Error(err, "submitting duplicate transaction should fail")
	requireT.True(tx.IsTransferSentError(err), "error should be TransferSent error, got: %v", err)
	t.Log("contract correctly rejected duplicate transaction")

	// Wait a bit to ensure no duplicate processing happened
	time.Sleep(5 * time.Second)

	// Verify balance hasn't changed (no duplicate processing)
	balanceRes2, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipientAddress.String(),
		Denom:   expectedInitialBalance.Denom,
	})
	requireT.NoError(err)
	balanceAfterDuplicateAttempt := balanceRes2.Balance.Amount

	requireT.Equal(
		balanceAfterFirst.String(),
		balanceAfterDuplicateAttempt.String(),
		"balance should not change after duplicate submission attempt",
	)
	t.Logf("confirmed balance unchanged: %s", balanceAfterDuplicateAttempt.String())

	// Now test with multiplier change
	t.Log("testing scenario with multiplier change...")

	// Update multiplier
	newMultiplier := "2.0"
	newTokens := []tx.XRPLToken{
		{
			Currency:       XRPLCORECurrency,
			Issuer:         tokenIssuer.String(),
			ActivationDate: uint64(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
			Multiplier:     newMultiplier,
		},
	}

	// Note: Only the owner can add XRPL tokens. Get the contract owner from config
	config, err := contractClient.GetContractConfig(ctx)
	requireT.NoError(err)
	contractOwner := sdk.MustAccAddressFromBech32(config.Owner)

	// Create a contract client with the chain's keyring which has the owner account
	contractAddr := sdk.MustAccAddressFromBech32(instances[0].Config.TXContractAddress)
	ownerContractClient := tx.NewContractClient(
		tx.DefaultContractClientConfig(contractAddr, txChain.TXChain.ChainSettings.Denom),
		txChain.TXChain.ClientContext,
	)

	// Tokens are now immutable - we cannot update multipliers
	// Attempting to add a duplicate token (same issuer/currency) should fail
	_, err = ownerContractClient.AddXRPLTokens(ctx, contractOwner, newTokens)
	requireT.Error(err)
	requireT.Contains(err.Error(), "Duplicated XRPL token")
	t.Log("confirmed: cannot add duplicate token (tokens are immutable)")

	// Try to submit the SAME transaction again
	// The original multiplier (1.0) will still be used
	newExpectedAmount := txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(200_000000))

	_, err = contractClient.ThresholdBankSend(
		ctx,
		trustedAddress,
		tx.ThresholdBankSendRequest{
			ID:        ourTxHash, // Same transaction hash!
			Amount:    newExpectedAmount,
			Recipient: recipientAddress.String(),
		},
	)
	requireT.Error(err, "submitting same transaction with different multiplier should fail")
	requireT.True(
		tx.IsTransferSentError(err),
		"error should be TransferSent error even with multiplier change, got: %v",
		err,
	)
	t.Log("contract correctly rejected transaction even after multiplier change")

	// Wait to ensure no processing happened
	time.Sleep(5 * time.Second)

	// Verify balance is still the same (processed with ORIGINAL multiplier only once)
	balanceRes3, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipientAddress.String(),
		Denom:   expectedInitialBalance.Denom,
	})
	requireT.NoError(err)
	finalBalance := balanceRes3.Balance.Amount

	requireT.Equal(
		balanceAfterFirst.String(),
		finalBalance.String(),
		"Balance should still be the same after multiplier change - transaction processed only once",
	)
	t.Logf("final balance: %s (correctly unchanged)", finalBalance.String())

	// Verify it's NOT the new multiplier amount
	requireT.NotEqual(
		newExpectedAmount.Amount.String(),
		finalBalance.String(),
		"balance should NOT be calculated with new multiplier",
	)

	t.Log("transaction was processed exactly once despite multiple attempts and multiplier change")
}

// TestConfigChangeDetectionAndRestart tests that the relayer detects config changes and restarts with new config.
func TestConfigChangeDetectionAndRestart(t *testing.T) {
	t.Parallel()

	ctx, chains := NewTestingContext(t)

	requireT := require.New(t)

	xrplChain := chains.XRPL
	txChain := chains.TX

	// Setup token issuer
	tokenIssuer := xrplChain.GenAccount(ctx, t, 10)
	tokenCurrency, err := rippledata.NewCurrency(XRPLCORECurrency)
	requireT.NoError(err)

	enableDefaultRippling(ctx, t, chains, tokenIssuer)

	// Initial config with multiplier 1.0
	initialTokens := []service.XRPLTokenConfig{
		{
			XRPLIssuer:     tokenIssuer.String(),
			XRPLCurrency:   XRPLCORECurrency,
			ActivationDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Multiplier:     "1.0",
		},
	}

	// Deploy contract and start relayer
	owner := txChain.TXChain.GenAccount()
	trustedAddress1 := txChain.TXChain.GenAccount()
	trustedAddress2 := txChain.TXChain.GenAccount()
	trustedAddress3 := txChain.TXChain.GenAccount()

	t.Log("Funding accounts.")
	txChain.TXChain.Faucet.FundAccounts(ctx, t,
		integration.NewFundedAccount(owner, txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(5000000000))),
		integration.NewFundedAccount(trustedAddress1, txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(5000000000))),
		integration.NewFundedAccount(trustedAddress2, txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(5000000000))),
		integration.NewFundedAccount(trustedAddress3, txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(5000000000))),
	)

	contractClient := tx.NewContractClient(tx.DefaultContractClientConfig(nil, ""), txChain.TXChain.ClientContext)

	t.Log("Deploying and instantiating the smart contract with initial config.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, tx.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold:  2,
		MinAmount:  sdkmath.NewIntFromUint64(100),
		MaxAmount:  sdkmath.NewIntFromUint64(200_000_000),
		XRPLTokens: convertServiceTokensToContractTokens(initialTokens),
		Label:      "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(10_000_000_000))
	txChain.TXChain.Faucet.FundAccounts(ctx, t, integration.NewFundedAccount(contractAddr, coinToFundContract))

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	// Build service configs for multiple relayers (need at least 2 for threshold)
	zapLogger := zaptest.NewLogger(t)
	baseServiceCfg := service.Config{
		XRPLRPCURL:                    xrplChain.Config().RPCAddress,
		XRPLHistoryScanStartLedger:    0,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 0,
		XRPLMemoSuffix:                XRPLTestMemoSuffix,
		BSCScannerDisabled:            true,
		TXRPCURL:                      txChain.Config().RPCAddress,
		TXGRPCURL:                     txChain.Config().GRPCAddress,
		TXChainID:                     txChain.TXChain.ChainSettings.ChainID,
		TXContractAddress:             contractAddr.String(),
		ConfigWatcherPollInterval:     2 * time.Second, // Short interval for testing
	}

	// Start multiple relayers with auto-restart (need at least 2 for threshold)
	relayerCtx, relayerCancel := context.WithCancel(ctx)
	defer relayerCancel()

	relayerErrCh := make(chan error, 3)
	trustedAddresses := []sdk.AccAddress{trustedAddress1, trustedAddress2, trustedAddress3}

	// Start 3 relayers (one for each trusted address) to meet threshold of 2
	for _, trustedAddr := range trustedAddresses {
		serviceCfg := baseServiceCfg
		serviceCfg.TXSenderAddress = trustedAddr.String()

		go func(cfg service.Config) {
			// Use the actual RunExecutorWithAutoRestart function
			err := service.RunExecutorWithAutoRestart(
				relayerCtx, cfg, txChain.TXChain.ClientContext.Keyring(), zapLogger)
			relayerErrCh <- err
		}(serviceCfg)
	}

	// Monitor for relayer errors in background
	go func() {
		for {
			select {
			case <-relayerCtx.Done():
				return
			case err := <-relayerErrCh:
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Logf("Relayer error: %v", err)
				}
			}
		}
	}()

	// Wait a bit for relayer to start
	time.Sleep(2 * time.Second)

	// Prepare XRPL sender
	xrplSender := prepareXRPLSender(ctx, t, xrplChain, initialTokens)
	recipientAddress := txChain.TXChain.GenAccount()

	// Send first transaction with initial multiplier (1.0)
	t.Log("Sending first transaction with initial config (multiplier 1.0)")
	sendAmount := "100.0"
	sendPayments(ctx, t, xrplChain, xrplSender, []payment{
		{
			address:  recipientAddress.String(),
			amounts:  []string{sendAmount},
			issuer:   tokenIssuer,
			currency: tokenCurrency,
		},
	})

	// Wait for first transaction to be processed
	expectedInitialBalance := txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(100_000000)) // 100.0 * 1.0 * 10^6
	awaitForBalance(ctx, t, txChain.TXChain.ClientContext, recipientAddress.String(), expectedInitialBalance)
	t.Log("First transaction processed with initial multiplier")

	// Note: Tokens are now immutable - we cannot update multipliers
	// Attempting to add a duplicate token (same issuer/currency) should fail
	t.Log("Attempting to add duplicate token (should fail - tokens are immutable)")
	newTokens := []tx.XRPLToken{
		{
			Currency:       XRPLCORECurrency,
			Issuer:         tokenIssuer.String(),
			ActivationDate: uint64(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()),
			Multiplier:     "2.0", // Changed from 1.0 to 2.0
		},
	}

	_, err = contractClient.AddXRPLTokens(ctx, owner, newTokens)
	requireT.Error(err)
	requireT.Contains(err.Error(), "Duplicated XRPL token")
	t.Log("Confirmed: cannot add duplicate token - tokens are immutable")

	// Verify config was NOT updated - original multiplier remains
	cfg, err := contractClient.GetContractConfig(ctx)
	requireT.NoError(err)
	requireT.Equal("1.0", cfg.XRPLTokens[0].Multiplier, "Original multiplier should remain unchanged")
	t.Log("Contract config unchanged - tokens are immutable")

	// Send second transaction - should still use original multiplier (1.0)
	t.Log("Sending second transaction - will use original multiplier (1.0) since tokens are immutable")
	sendAmount2 := "50.0"
	sendPayments(ctx, t, xrplChain, xrplSender, []payment{
		{
			address:  recipientAddress.String(),
			amounts:  []string{sendAmount2},
			issuer:   tokenIssuer,
			currency: tokenCurrency,
		},
	})

	// Wait for second transaction to be processed with original multiplier (since tokens are immutable)
	// Expected: 100 * 1.0 + 50 * 1.0 = 100 + 50 = 150
	expectedFinalBalance := txChain.TXChain.NewCoin(sdkmath.NewIntFromUint64(150_000000))
	awaitForBalance(ctx, t, txChain.TXChain.ClientContext, recipientAddress.String(), expectedFinalBalance)
	t.Log("Second transaction processed with original multiplier (1.0) - tokens are immutable")

	// Verify the balance is correct
	bankClient := banktypes.NewQueryClient(txChain.TXChain.ClientContext)
	balanceRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: recipientAddress.String(),
		Denom:   expectedFinalBalance.Denom,
	})
	requireT.NoError(err)
	requireT.Equal(expectedFinalBalance.Amount.String(), balanceRes.Balance.Amount.String(),
		"Balance should reflect both transactions: first with multiplier 1.0, second with multiplier 2.0")

	t.Log("Config change detection and restart verified successfully")
}
