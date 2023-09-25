package xrpl

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/CoreumFoundation/coreum-tools/pkg/http"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/metric"
)

const (
	mainnetRPCURL                   = "https://s2.ripple.com:51234/"
	mainnetCoreAccount              = "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D"
	mainnetInitialBridgeLedgerIndex = 80175264
)

func TestRPCClient_SubscribeAccountTransactions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	rpcClient := NewRPCClient(rpcClientConfig, logger.NewZapLogger(zaptest.NewLogger(t), nil), httpClient)

	txsCh := make(chan Transaction)

	startLedger := int64(mainnetInitialBridgeLedgerIndex)
	endLedger := int64(mainnetInitialBridgeLedgerIndex + 1000)

	txs := make([]Transaction, 0)

	// we read the channel until the SubscribeAccountTransactions is done
	doneRead := make(chan struct{})
	go func() {
		defer close(doneRead)
		for {
			select {
			case <-ctx.Done():
				return
			case tx, open := <-txsCh:
				if !open {
					return
				}
				txs = append(txs, tx)
			}
		}
	}()

	latestProcessedIndex, err := rpcClient.SubscribeAccountTransactions(ctx, mainnetCoreAccount, startLedger, endLedger, txsCh)
	require.NoError(t, err)
	require.Equal(t, int64(80176263), latestProcessedIndex)

	close(txsCh)

	select {
	case <-ctx.Done():
		t.FailNow()
	case <-doneRead:
	}
	require.Equal(t, 14, len(txs))
	for _, tx := range txs {
		require.GreaterOrEqual(t, tx.LedgerIndex, startLedger, fmt.Sprintf("tx:%+v", tx))
		require.LessOrEqual(t, tx.LedgerIndex, endLedger, fmt.Sprintf("tx:%+v", tx))
	}
}

func TestRPCClient_GetAccountTransactions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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
			case tx, open := <-txsCh:
				if !open {
					return
				}
				txs = append(txs, tx)
			}
		}
	}()
	_, latestProcessedIndex, err := rpcClient.GetAccountTransactions(ctx, mainnetCoreAccount, 0, 0, markerPtr, txsCh)
	require.NoError(t, err)
	require.Equal(t, int64(80175276), latestProcessedIndex)
	close(txsCh)

	select {
	case <-ctx.Done():
		t.FailNow()
	case <-doneRead:
	}

	require.Equal(t, 1, len(txs))
	tx := txs[0]

	expectedTime, err := time.Parse(time.DateTime, "2023-06-01 16:00:02")
	require.NoError(t, err)
	expectedTx := Transaction{
		Account:     "rL54wzknUXxqiC8Tzs6mzLi3QJTtX5uVK6",
		Destination: "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
		DeliveryAmount: DeliveredAmount{
			Currency: "434F524500000000000000000000000000000000",
			Issuer:   "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
			Value:    big.NewFloat(1),
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

	expectedTxBytes, err := json.Marshal(expectedTx)
	require.NoError(t, err)

	txBytes, err := json.Marshal(tx)
	require.NoError(t, err)
	require.Equal(t, string(expectedTxBytes), string(txBytes))

	// fetch same transaction by its hash
	txsMap, err := rpcClient.GetTransactions(ctx, []string{expectedTx.Hash})
	require.NoError(t, err)
	require.Len(t, txs, 1)
	tx = txsMap[expectedTx.Hash]
	txBytes, err = json.Marshal(tx)
	require.NoError(t, err)
	require.Equal(t, string(expectedTxBytes), string(txBytes))
}

func TestRPCClient_RPCErrorHandling(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
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
			case tx, open := <-txsCh:
				if !open {
					return
				}
				txs = append(txs, tx)
			}
		}
	}()
	_, _, err = rpcClient.GetAccountTransactions(ctx, "invalid-account", 0, 0, markerPtr, txsCh)
	require.ErrorContains(t, err, "actMalformed")
}

func TestRPCClient_GetCurrentLedger(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	t.Cleanup(cancel)

	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	metricRecorder, err := metric.NewRecorder()
	require.NoError(t, err)
	rpcClient := NewRPCClient(rpcClientConfig, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), httpClient)

	currentLedger, err := rpcClient.GetCurrentLedger(ctx)
	require.NoError(t, err)
	require.True(t, currentLedger > 0)
}

func Test_convertJsonToDeliveredAmount(t *testing.T) {
	t.Parallel()

	type args struct {
		amount json.RawMessage
	}
	tests := []struct {
		name       string
		args       args
		wantValid  bool
		wantAmount DeliveredAmount
	}{
		{
			name: "empty_string",
			args: args{
				amount: json.RawMessage(""),
			},
			wantValid:  false,
			wantAmount: DeliveredAmount{},
		},
		{
			name: "xrp_amount",
			args: args{
				amount: json.RawMessage("10000000"),
			},
			wantValid: false,
			// For the case with the native token we expect empty result
			wantAmount: DeliveredAmount{},
		},
		{
			name: "coreum_amount",
			args: args{
				amount: json.RawMessage(`{
           "currency": "434F524500000000000000000000000000000000",
           "issuer": "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
           "value": "1.72361980446674"
         }`),
			},
			wantValid: true,
			wantAmount: DeliveredAmount{
				Currency: "434F524500000000000000000000000000000000",
				Issuer:   "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
				Value: func() *big.Float {
					v, _ := big.NewFloat(0).SetString("1.72361980446674")
					return v
				}(),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotAmount, gotValid := convertJSONToDeliveredAmount(tt.args.amount)
			require.Equal(t, tt.wantValid, gotValid)
			require.Equal(t, tt.wantAmount.Currency, gotAmount.Currency)
			require.Equal(t, tt.wantAmount.Issuer, gotAmount.Issuer)
			require.Equal(t, fmt.Sprintf("%s", tt.wantAmount.Value), fmt.Sprintf("%s", gotAmount.Value))
		})
	}
}
