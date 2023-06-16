package xrpl

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/http"
)

func TestTxScanner_Scan(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	t.Cleanup(cancel)

	rpcClientConfig := DefaultRPCClientConfig(mainnetRPCURL)
	httpClient := http.NewRetryableClient(http.DefaultClientConfig())
	rpcClient := NewRPCClient(rpcClientConfig, httpClient)

	ctx = logger.WithLogger(ctx, zaptest.NewLogger(t))

	txScanner := NewTxScanner(DefaultTxScannerConfig(), rpcClient)

	txsCh := make(chan Transaction)
	err := txScanner.Subscribe(
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
