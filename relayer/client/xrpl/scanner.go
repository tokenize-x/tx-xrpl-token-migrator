package xrpl

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// RPCTxProvider is RPC transactions provider.
type RPCTxProvider interface {
	SubscribeAccountTransactions(ctx context.Context, account string, startLedger, endLedger int64, ch chan<- Transaction) (int64, error)
	GetCurrentLedger(ctx context.Context) (int64, error)
}

// MetricRecorder is coreum metric recorder interface.
type MetricRecorder interface {
	SetXRPLLatestAccountLedgerIndex(v int64)
}

// TxScannerConfig is the TxScanner config.
type TxScannerConfig struct {
	RetryDelay time.Duration
}

// DefaultTxScannerConfig returns the default TxScannerConfig.
func DefaultTxScannerConfig() TxScannerConfig {
	return TxScannerConfig{
		RetryDelay: 30 * time.Second,
	}
}

// TxScanner is XRPL transactions scanner.
type TxScanner struct {
	cfg            TxScannerConfig
	log            logger.Logger
	rpcTxProvider  RPCTxProvider
	metricRecorder MetricRecorder
}

// NewTxScanner returns a nw instance of the TxScanner.
func NewTxScanner(cfg TxScannerConfig, log logger.Logger, rpcTxScanner RPCTxProvider, metricRecorder MetricRecorder) *TxScanner {
	return &TxScanner{
		cfg:            cfg,
		log:            log,
		rpcTxProvider:  rpcTxScanner,
		metricRecorder: metricRecorder,
	}
}

// Subscribe subscribes on rpc and ws client account transactions.
func (t *TxScanner) Subscribe(
	ctx context.Context,
	account string,
	historyScanStartLedger,
	recentScanIndexesBack int64,
	ch chan<- Transaction,
) error {
	if recentScanIndexesBack <= 0 {
		return errors.New("recentScanIndexesBack must be greater than zero")
	}

	var initialLedger int64
	t.doWithResubscribe(ctx, false, func() error {
		t.log.Info("Fetching initial ledger")
		var err error
		initialLedger, err = t.rpcTxProvider.GetCurrentLedger(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	t.log.Info(
		"Subscribing xrpl scanner",
		zap.String("account", account),
		zap.Int64("historyScanStartLedger", historyScanStartLedger),
		zap.Int64("recentScanIndexesBack", recentScanIndexesBack),
		zap.Int64("initialLedger", initialLedger),
	)

	go func() {
		// handling of error retry to keep the latest start paging index
		startHistoricalScanIndex := historyScanStartLedger - 1
		t.doWithResubscribe(ctx, false, func() error {
			endLedger := initialLedger - recentScanIndexesBack
			t.log.Info("Scanning full history", zap.Int64("endLedger", endLedger))
			var err error
			startHistoricalScanIndex, err = t.rpcTxProvider.SubscribeAccountTransactions(
				ctx,
				account,
				startHistoricalScanIndex+1,
				endLedger,
				ch,
			)
			if err != nil {
				return err
			}
			t.log.Info("Scanning of full history is done")

			return nil
		})
	}()

	go func() {
		// we do it initially to do the rescan for about a day back
		// to sync latest transactions faster than the full history scan does it
		prevEndLedger := initialLedger - recentScanIndexesBack
		t.doWithResubscribe(ctx, true, func() error {
			startLedger := prevEndLedger + 1
			t.log.Info("Scanning recent history", zap.Int64("startLedger", startLedger))
			latestProcessedLedger, err := t.rpcTxProvider.SubscribeAccountTransactions(
				ctx,
				account,
				startLedger,
				0,
				ch,
			)
			t.metricRecorder.SetXRPLLatestAccountLedgerIndex(latestProcessedLedger)
			prevEndLedger = latestProcessedLedger
			if err != nil {
				return err
			}
			t.log.Info("Scanning of the recent history done", zap.Int64("latestProcessedLedger", latestProcessedLedger))

			return nil
		})
	}()

	return nil
}

func (t *TxScanner) doWithResubscribe(
	ctx context.Context,
	repeat bool,
	f func() error,
) {
	err := retry.Do(ctx, t.cfg.RetryDelay, func() error {
		if err := f(); err != nil {
			t.log.Error("Error on scan subscription", zap.Error(err))
			return retry.Retryable(err)
		}
		if repeat {
			t.log.Info("Waiting before the next execution.", zap.String("delay", t.cfg.RetryDelay.String()))
			return retry.Retryable(errors.New("repeat scan"))
		}

		return nil
	})
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	// this panic is unexpected
	panic(errors.Wrap(err, "unexpected error in scan with retry"))
}
