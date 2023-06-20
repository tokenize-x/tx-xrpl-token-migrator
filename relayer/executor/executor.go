package executor

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/finder"
)

// ContractClient is coreum contract client interface.
type ContractClient interface {
	ThresholdBankSend(ctx context.Context, sender sdk.AccAddress, requests ...coreum.ThresholdBankSendRequest) (*sdk.TxResponse, error)
}

// Finder is transactions finder interface.
type Finder interface {
	SubscribeCoreumSendTransactions(ctx context.Context, ch chan<- finder.PendingCoreumSendTransaction) error
}

// Config represents the executor config.
type Config struct {
	SenderAddress sdk.AccAddress
	RetryDelay    time.Duration
}

// DefaultConfig returns default Config.
func DefaultConfig(senderAddress sdk.AccAddress) Config {
	return Config{
		SenderAddress: senderAddress,
		RetryDelay:    30 * time.Second,
	}
}

// Executor is coreum transaction executor.
type Executor struct {
	cfg            Config
	contractClient ContractClient
	finder         Finder
}

// NewExecutor returns a new instance of the Executor.
func NewExecutor(cfg Config, contractClient ContractClient, finder Finder) *Executor {
	return &Executor{
		cfg:            cfg,
		contractClient: contractClient,
		finder:         finder,
	}
}

// Start starts an executor.
func (e *Executor) Start(ctx context.Context) error {
	log := logger.Get(ctx)
	log.Info("Starting executor.")

	txsCh := make(chan finder.PendingCoreumSendTransaction)
	if err := e.finder.SubscribeCoreumSendTransactions(ctx, txsCh); err != nil {
		return err
	}

	executionDoneCh := make(chan struct{})
	go func() {
		defer close(executionDoneCh)
		for {
			select {
			case <-ctx.Done():
				return
			case tx := <-txsCh:
				err := retry.Do(ctx, e.cfg.RetryDelay, func() error {
					sendReq := coreum.ThresholdBankSendRequest{
						ID:        tx.XRPLTxHash,
						Amount:    tx.CoreumAmount,
						Recipient: tx.CoreumDestination.String(),
					}
					_, err := e.contractClient.ThresholdBankSend(ctx, e.cfg.SenderAddress, sendReq)
					if err == nil {
						log.Info(
							"Submitted new evidence.",
							zap.String("senderAddress", e.cfg.SenderAddress.String()),
							zap.Any("request", sendReq),
						)
						return nil
					}
					if coreum.IsEvidenceProvidedError(err) {
						log.Debug(
							"Evidence has been already submitted.",
							zap.String("senderAddress", e.cfg.SenderAddress.String()),
							zap.String("xrplTxHash", tx.XRPLTxHash),
						)
						return nil
					}
					if coreum.IsTransferSentError(err) {
						log.Debug(
							"Transfer has been already sent.",
							zap.String("senderAddress", e.cfg.SenderAddress.String()),
							zap.String("xrplTxHash", tx.XRPLTxHash),
						)
						return nil
					}
					log.Error("Can't execute coreum contract transaction, the execution will be repeated", zap.Any("request", sendReq), zap.String("delay", e.cfg.RetryDelay.String()), zap.Error(err))
					return retry.Retryable(err)
				})
				// unexpected error
				if err != nil && !errors.Is(err, context.Canceled) {
					panic(err)
				}
			}
		}
	}()

	<-executionDoneCh

	return nil
}
