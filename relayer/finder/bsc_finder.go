package finder

import (
	"context"
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"
)

type BSCScanner interface {
	Subscribe(ctx context.Context, ch chan<- *abi.TxBridgeBridgeInitiated) error
}

type BSCFinderConfig struct {
	ChainID    string
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
	eventsCh := make(chan *abi.TxBridgeBridgeInitiated)

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

func (f *BSCFinder) buildPendingTransaction(event *abi.TxBridgeBridgeInitiated) (PendingTXSendTransaction, bool) {
	txHash := event.Raw.TxHash.Hex()

	// extract address by stripping the chainID from destinationPayload
	// format: {bech32Address}{chainID} e.g., "devcore1abc.../coreum-devnet-1"
	address := extractAddressFromDestinationPayload(event.DestinationPayload, f.cfg.ChainID)

	destAddr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		f.log.Warn("invalid BSC bridge destination address",
			zap.String("txHash", txHash),
			zap.String("destinationPayload", event.DestinationPayload),
			zap.String("extractedAddress", address),
			zap.Error(err),
		)
		return PendingTXSendTransaction{}, false
	}

	// Convert amount from wei (18 decimals) to TX amount (6 decimals)
	txAmount := convertBSCAmountToTXCoin(event.Amount, f.cfg.TXDenom, f.cfg.TXDecimals)
	if txAmount.IsZero() {
		f.log.Info("BSC bridge zero amount after conversion", zap.String("txHash", txHash))
		return PendingTXSendTransaction{}, false
	}

	f.log.Debug("BSC bridge event converted to PendingTXSendTransaction",
		zap.String("txHash", txHash),
		zap.String("destinationPayload", event.DestinationPayload),
		zap.String("extractedAddress", address),
		zap.String("destination", destAddr.String()),
		zap.String("originalAmountWei", event.Amount.String()),
		zap.String("convertedAmount", txAmount.String()),
	)

	return PendingTXSendTransaction{
		TXDestination: destAddr,
		TXAmount:      txAmount,
		XRPLTxHash:    txHash,
	}, true
}

// extractAddressFromDestinationPayload extracts the bech32 address by removing the chainID suffix.
// Input format: "devcore1abc.../coreum-devnet-1" -> "devcore1abc..."
func extractAddressFromDestinationPayload(destinationPayload, chainIDSuffix string) string {
	return strings.TrimSuffix(destinationPayload, chainIDSuffix)
}

// converts BSC amount (18 decimals) to TX coin (6 decimals).
func convertBSCAmountToTXCoin(weiAmount *big.Int, denom string, txDecimals int) sdk.Coin {
	if weiAmount == nil || weiAmount.Sign() <= 0 {
		return sdk.NewCoin(denom, sdkmath.ZeroInt())
	}

	// BSC/ERC20 uses 18 decimals, TX uses 6 decimals
	// Divide by 10^(18-6) = 10^12
	bscDecimals := 18
	decimalDiff := bscDecimals - txDecimals

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalDiff)), nil)
	txAmount := new(big.Int).Div(weiAmount, divisor)

	return sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(txAmount))
}
