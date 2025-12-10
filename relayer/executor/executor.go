package executor

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/finder"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
)

// ContractClient is TX contract client interface.
type ContractClient interface {
	ThresholdBankSend(
		ctx context.Context,
		sender sdk.AccAddress,
		requests ...tx.ThresholdBankSendRequest,
	) (*sdk.TxResponse, error)
	GetContractConfig(ctx context.Context) (tx.Config, error)
}

// Finder is transactions finder interface.
type Finder interface {
	SubscribeTXSendTransactions(ctx context.Context, ch chan<- finder.PendingTXSendTransaction) error
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

// Executor is TX transaction executor.
type Executor struct {
	cfg            Config
	log            logger.Logger
	contractClient ContractClient
	finders        []Finder
}

// NewExecutor returns a new instance of the Executor.
func NewExecutor(cfg Config, log logger.Logger, contractClient ContractClient, finders []Finder) *Executor {
	return &Executor{
		cfg:            cfg,
		log:            log,
		contractClient: contractClient,
		finders:        finders,
	}
}

// Start starts an executor.
func (e *Executor) Start(ctx context.Context) error {
	e.log.Info("Starting executor.")

	txsCh := make(chan finder.PendingTXSendTransaction)

	for _, f := range e.finders {
		if err := f.SubscribeTXSendTransactions(ctx, txsCh); err != nil {
			return err
		}
	}

	executionDoneCh := make(chan struct{})
	go func() {
		defer close(executionDoneCh)
		for {
			select {
			case <-ctx.Done():
				return
			case txn := <-txsCh:
				err := retry.Do(ctx, e.cfg.RetryDelay, func() error {
					e.log.Info(
						"Found valid transaction.",
						zap.Any("tx", txn),
					)

					contractCfg, err := e.contractClient.GetContractConfig(ctx)
					if err != nil {
						return retry.Retryable(err)
					}

					sendReq := tx.ThresholdBankSendRequest{
						ID:        txn.XRPLTxHash,
						Amount:    txn.TXAmount,
						Recipient: txn.TXDestination.String(),
					}

					if contractCfg.MinAmount.GT(txn.TXAmount.Amount) {
						e.log.Info(
							"Low amount, execution is skipped.",
							zap.String("senderAddress", e.cfg.SenderAddress.String()),
							zap.Any("request", sendReq),
						)
						return nil
					}

					_, err = e.contractClient.ThresholdBankSend(ctx, e.cfg.SenderAddress, sendReq)
					if err == nil {
						e.log.Info(
							"Submitted new evidence.",
							zap.String("senderAddress", e.cfg.SenderAddress.String()),
							zap.Any("request", sendReq),
						)
						return nil
					}
					if tx.IsEvidenceProvidedError(err) {
						e.log.Debug(
							"Evidence has already been submitted.",
							zap.String("senderAddress", e.cfg.SenderAddress.String()),
							zap.String("xrplTxHash", txn.XRPLTxHash),
						)
						return nil
					}
					if tx.IsTransferSentError(err) {
						e.log.Debug(
							"Transfer has already been sent.",
							zap.String("senderAddress", e.cfg.SenderAddress.String()),
							zap.String("xrplTxHash", txn.XRPLTxHash),
						)
						return nil
					}

					// Account sequence mismatch is a transient error that occurs when multiple
					// transactions are submitted concurrently. It's expected and will be retried,
					// so we log it as a warning instead of an error to avoid incrementing the error counter.
					if tx.IsAccountSequenceMismatchError(err) {
						e.log.Warn(
							"Account sequence mismatch, retrying",
							zap.Any("request", sendReq),
							zap.String("delay", e.cfg.RetryDelay.String()),
							zap.Error(err),
						)
						return retry.Retryable(err)
					}

					e.log.Error(
						"Can't execute TX contract transaction, the execution will be repeated",
						zap.Any("request", sendReq),
						zap.String("delay", e.cfg.RetryDelay.String()),
						zap.Error(err),
					)

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
