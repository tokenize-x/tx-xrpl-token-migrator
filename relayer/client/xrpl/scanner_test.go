package xrpl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/http"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/metric"
)

func TestTxScanner_Scan(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	t.Cleanup(cancel)

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	metricRecorder, err := metric.NewRecorder()
	require.NoError(t, err)

	log := logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder)
	rpcClient := NewRPCClient(rpcClientConfig, log, httpClient)

	txScanner := NewTxScanner(DefaultTxScannerConfig(), log, rpcClient, metricRecorder)

	txsCh := make(chan Transaction)
	err = txScanner.Subscribe(
		ctx,
		mainnetCoreAccount,
		mainnetInitialBridgeLedgerIndex,
		10,
		txsCh)
	require.NoError(t, err)

	expectedTx := 30
	for {
		select {
		case <-ctx.Done():
			t.Fail()
		case <-txsCh:
			expectedTx--
			if expectedTx == 0 {
				return
			}
		}
	}
}
