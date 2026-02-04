package xrpl

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CoreumFoundation/coreum-tools/pkg/http"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/metric"
)

const (
	mainnetRPCURL                   = "https://s2.ripple.com:51234/"
	mainnetCoreAccount              = "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D"
	mainnetInitialBridgeLedgerIndex = 80175264
)

func TestRPCClient_SubscribeAccountTransactions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)

	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	rpcClient := NewRPCClient(rpcClientConfig, logger.NewZapLogger(zaptest.NewLogger(t), nil), httpClient)

	txsCh := make(chan Transaction)

	startLedger := int64(mainnetInitialBridgeLedgerIndex)
	endLedger := int64(mainnetInitialBridgeLedgerIndex + 100)

	txs := make([]Transaction, 0)

	// we read the channel until the SubscribeAccountTransactions is done
	doneRead := make(chan struct{})
	go func() {
		defer close(doneRead)
		for {
			select {
			case <-ctx.Done():
				return
			case txn, open := <-txsCh:
				if !open {
					return
				}
				txs = append(txs, txn)
			}
		}
	}()

	latestProcessedIndex, err := rpcClient.SubscribeAccountTransactions(
		ctx,
		convertStringToRippleAccount(t, mainnetCoreAccount),
		startLedger,
		endLedger,
		txsCh,
	)
	require.NoError(t, err)
	require.Equal(t, int64(80175363), latestProcessedIndex)

	close(txsCh)

	select {
	case <-ctx.Done():
		t.FailNow()
	case <-doneRead:
	}
	require.Len(t, txs, 1028)
	for _, txn := range txs {
		require.GreaterOrEqual(t, txn.LedgerIndex, startLedger, "tx:%+v", txn)
		require.LessOrEqual(t, txn.LedgerIndex, endLedger, "tx:%+v", txn)
	}
}

func TestRPCClient_GetAccountTransactions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)

	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	metricRecorder, err := metric.NewRecorder()
	require.NoError(t, err)
	rpcClient := NewRPCClient(rpcClientConfig, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), httpClient)

	// get core payment transaction
	markerPtr := &PageMarker{
		Ledger: mainnetInitialBridgeLedgerIndex,
		Seq:    0,
	}
	txsCh := make(chan Transaction)
	txs := make([]Transaction, 0)
	// we read the channel until the SubscribeAccountTransactions is done
	doneRead := make(chan struct{})
	go func() {
		defer close(doneRead)
		for {
			select {
			case <-ctx.Done():
				return
			case txn, open := <-txsCh:
				if !open {
					return
				}
				txs = append(txs, txn)
			}
		}
	}()
	_, latestProcessedIndex, err := rpcClient.GetAccountTransactions(
		ctx,
		convertStringToRippleAccount(t, mainnetCoreAccount),
		0,
		0,
		markerPtr,
		txsCh,
	)
	require.NoError(t, err)
	require.Equal(t, int64(80175276), latestProcessedIndex)
	close(txsCh)

	select {
	case <-ctx.Done():
		t.FailNow()
	case <-doneRead:
	}

	require.Len(t, txs, 100)
	txn := txs[0]

	expectedTime, err := time.Parse(time.DateTime, "2023-06-01 16:00:02")
	require.NoError(t, err)
	expectedTxn := Transaction{
		Account:     "rL54wzknUXxqiC8Tzs6mzLi3QJTtX5uVK6",
		Destination: "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
		DeliveryAmount: rippledata.Amount{
			Currency: convertStringToRippleCurrency(t, "434F524500000000000000000000000000000000"),
			Issuer:   convertStringToRippleAccount(t, "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D"),
			Value:    convertStringToRippleValue(t, "1", false),
		},
		Memos: []string{
			"core12zs7nt7p73mzzplnl0ye8txzvxjgnqd4h2c8uy=coreum",
			"b7b05053-fd59-44e1-b028-fec12a49ab93",
		},
		Hash:              "4054DF1B39852E80D5C3E9AA71E2E99B5CD14D9F4A618FD76BE9280901D8B749",
		TransactionType:   TransactionTypePayment,
		TransactionResult: TransactionResultSuccess,
		LedgerIndex:       80175264,
		Sequence:          4436,
		Date:              expectedTime,
		Validated:         true,
	}

	expectedTxnBytes, err := json.Marshal(expectedTxn)
	require.NoError(t, err)

	txnBytes, err := json.Marshal(txn)
	require.NoError(t, err)
	require.Equal(t, string(expectedTxnBytes), string(txnBytes))

	// fetch same transaction by its hash
	txsMap, err := rpcClient.GetTransactions(ctx, []string{expectedTxn.Hash})
	require.NoError(t, err)
	require.Len(t, txsMap, 1)
	txn = txsMap[expectedTxn.Hash]
	txnBytes, err = json.Marshal(txn)
	require.NoError(t, err)
	require.Equal(t, string(expectedTxnBytes), string(txnBytes))
}

func TestRPCClient_RPCErrorHandling(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)

	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	metricRecorder, err := metric.NewRecorder()
	require.NoError(t, err)
	rpcClient := NewRPCClient(rpcClientConfig, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), httpClient)

	// get core payment transaction
	markerPtr := &PageMarker{
		Ledger: mainnetInitialBridgeLedgerIndex,
		Seq:    0,
	}
	txsCh := make(chan Transaction)
	txs := make([]Transaction, 0)
	// we read the channel until the SubscribeAccountTransactions is done
	doneRead := make(chan struct{})
	go func() {
		defer close(doneRead)
		for {
			select {
			case <-ctx.Done():
				return
			case txn, open := <-txsCh:
				if !open {
					return
				}
				txs = append(txs, txn)
			}
		}
	}()
	_, _, err = rpcClient.GetAccountTransactions(
		ctx,
		convertStringToRippleAccount(t, mainnetCoreAccount),
		1000000000000000000,
		0,
		markerPtr,
		txsCh,
	)
	require.ErrorContains(t, err, "error code:57, error message:Ledger indexes invalid")
}

func TestRPCClient_GetCurrentLedger(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)

	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	metricRecorder, err := metric.NewRecorder()
	require.NoError(t, err)
	rpcClient := NewRPCClient(rpcClientConfig, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), httpClient)

	currentLedger, err := rpcClient.GetCurrentLedger(ctx)
	require.NoError(t, err)
	require.Positive(t, currentLedger)
}

func convertStringToRippleCurrency(t *testing.T, s string) rippledata.Currency {
	currency, err := rippledata.NewCurrency(s)
	require.NoError(t, err)

	return currency
}

//nolint:unparam // helper func
func convertStringToRippleAccount(t *testing.T, s string) rippledata.Account {
	acc, err := rippledata.NewAccountFromAddress(s)
	require.NoError(t, err)

	return *acc
}

func convertStringToRippleValue(t *testing.T, s string, native bool) *rippledata.Value {
	v, err := rippledata.NewValue(s, native)
	require.NoError(t, err)

	return v
}
