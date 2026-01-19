package finder

import (
	"context"
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bnb"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bnb/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"
)

type BNBScanner interface {
	Subscribe(ctx context.Context, ch chan<- *abi.TxBridgeBridgeInitiated) error
}

type BNBFinderConfig struct {
	ChainSuffix string
	TXDenom     string
	TXDecimals  int
}

// finds valid BNB bridge transactions and converts them to PendingTXSendTransaction.
type BNBFinder struct {
	cfg     BNBFinderConfig
	log     logger.Logger
	scanner BNBScanner
}

func NewBNBFinder(cfg BNBFinderConfig, log logger.Logger, scanner BNBScanner) *BNBFinder {
	return &BNBFinder{
		cfg:     cfg,
		log:     log,
		scanner: scanner,
	}
}

// subscribes to BNB bridge events and sends valid transactions to the channel.
func (f *BNBFinder) SubscribeTXSendTransactions(ctx context.Context, ch chan<- PendingTXSendTransaction) error {
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

func (f *BNBFinder) buildPendingTransaction(event *abi.TxBridgeBridgeInitiated) (PendingTXSendTransaction, bool) {
	txHash := event.Raw.TxHash.Hex()

	// extract address by stripping the chain suffix
	address := extractAddressFromTxchainAddress(event.TxchainAddress, f.cfg.ChainSuffix)

	destAddr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		f.log.Error("invalid BNB bridge destination address",
			zap.String("txHash", txHash),
			zap.String("txchainAddress", event.TxchainAddress),
			zap.String("extractedAddress", address),
			zap.Error(err),
		)
		return PendingTXSendTransaction{}, false
	}

	// Convert amount from wei (18 decimals) to TX amount (6 decimals)
	txAmount := convertBNBAmountToTXCoin(event.Amount, f.cfg.TXDenom, f.cfg.TXDecimals)
	if txAmount.IsZero() {
		f.log.Info("BNB bridge zero amount after conversion", zap.String("txHash", txHash))
		return PendingTXSendTransaction{}, false
	}

	f.log.Debug("BNB bridge event converted to PendingTXSendTransaction",
		zap.String("txHash", txHash),
		zap.String("originalTxchainAddress", event.TxchainAddress),
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

// extracts the bech32 address by removing the chain suffix.
// Input format: "testcore1abc.../coreum-testnet-1/v1" -> "testcore1abc..."
func extractAddressFromTxchainAddress(txchainAddress, suffix string) string {
	return strings.TrimSuffix(txchainAddress, suffix)
}

// converts BNB amount (18 decimals) to TX coin (6 decimals).
func convertBNBAmountToTXCoin(weiAmount *big.Int, denom string, txDecimals int) sdk.Coin {
	if weiAmount == nil || weiAmount.Sign() <= 0 {
		return sdk.NewCoin(denom, sdkmath.ZeroInt())
	}

	// BNB/ERC20 uses 18 decimals, TX uses 6 decimals
	// Divide by 10^(18-6) = 10^12
	bnbDecimals := 18
	decimalDiff := bnbDecimals - txDecimals

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalDiff)), nil)
	txAmount := new(big.Int).Div(weiAmount, divisor)

	return sdk.NewCoin(denom, sdkmath.NewIntFromBigInt(txAmount))
}

// ensure BNBFinder implements the scanner interface requirement
var _ BNBScanner = (*bnb.Scanner)(nil)
