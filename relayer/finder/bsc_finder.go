package finder

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"
)

type BSCScanner interface {
	Subscribe(ctx context.Context, ch chan<- *abi.TXBridgeSentToTXChain) error
}

type BSCFinderConfig struct {
	TXDenom    string
	TXDecimals int
}

// finds valid BSC bridge transactions and converts them to PendingTXSendTransaction.
type BSCFinder struct {
	cfg     BSCFinderConfig
	log     logger.Logger
	scanner BSCScanner
}

// ensure bsc.Scanner implements the BSCScanner interface
var _ BSCScanner = (*bsc.Scanner)(nil)

func NewBSCFinder(cfg BSCFinderConfig, log logger.Logger, scanner BSCScanner) *BSCFinder {
	return &BSCFinder{
		cfg:     cfg,
		log:     log,
		scanner: scanner,
	}
}

// subscribes to BSC bridge events and sends valid transactions to the channel.
func (f *BSCFinder) SubscribeTXSendTransactions(ctx context.Context, ch chan<- PendingTXSendTransaction) error {
	eventsCh := make(chan *abi.TXBridgeSentToTXChain)

	if err := f.scanner.Subscribe(ctx, eventsCh); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-eventsCh:
				pendingTx, ok := f.buildPendingTransaction(event)
				if !ok {
					continue
				}
				ch <- pendingTx
			}
		}
	}()

	return nil
}

func (f *BSCFinder) buildPendingTransaction(event *abi.TXBridgeSentToTXChain) (PendingTXSendTransaction, bool) {
	txHash := event.Raw.TxHash.Hex()

	destAddr, err := sdk.AccAddressFromBech32(event.TxAddress)
	if err != nil {
		f.log.Warn("invalid BSC bridge destination address",
			zap.String("txHash", txHash),
			zap.String("txAddress", event.TxAddress),
			zap.Error(err),
		)
		return PendingTXSendTransaction{}, false
	}

	if event.Amount == nil || event.Amount.Sign() <= 0 {
		f.log.Info("BSC bridge zero or invalid amount", zap.String("txHash", txHash))
		return PendingTXSendTransaction{}, false
	}
	txAmount := sdk.NewCoin(f.cfg.TXDenom, sdkmath.NewIntFromBigInt(event.Amount))

	f.log.Debug("BSC bridge event converted to PendingTXSendTransaction",
		zap.String("txHash", txHash),
		zap.String("txAddress", event.TxAddress),
		zap.String("destination", destAddr.String()),
		zap.String("amount", txAmount.String()),
	)

	return PendingTXSendTransaction{
		TXDestination: destAddr,
		TXAmount:      txAmount,
		XRPLTxHash:    txHash,
	}, true
}
