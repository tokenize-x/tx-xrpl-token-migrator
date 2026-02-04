package finder

import (
	"context"
	"math/big"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rippledata "github.com/rubblelabs/ripple/data"
	"go.uber.org/zap"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/xrpl"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

// PendingTXSendTransaction represents the pending transaction to be sent to the TX blockchain.
type PendingTXSendTransaction struct {
	TXDestination sdk.AccAddress
	TXAmount      sdk.Coin
	XRPLTxHash    string
}

// XRPLScanner is XRPL scanner which provides XRPL transactions.
type XRPLScanner interface {
	Subscribe(
		ctx context.Context,
		account rippledata.Account,
		historyScanStartLedger,
		recentScanIndexesBack int64,
		ch chan<- xrpl.Transaction,
	) error
}

// Config is Finder config.
type Config struct {
	XRPLIssuer                 rippledata.Account
	XRPLCurrency               rippledata.Currency
	ActivationDate             time.Time
	Multiplier                 string
	XRPLHistoryScanStartLedger int64
	XRPLRecentScanIndexesBack  int64
	XRPLMemoSuffix             string

	TXDenom    string
	TXDecimals int
}

// Finder is a finder for the valid transactions.
type Finder struct {
	cfg         Config
	log         logger.Logger
	xrplScanner XRPLScanner
}

// NewFinder returns a new instance of the Finder.
func NewFinder(cfg Config, log logger.Logger, xrplScanner XRPLScanner) *Finder {
	return &Finder{
		cfg:         cfg,
		log:         log,
		xrplScanner: xrplScanner,
	}
}

// SubscribeTXSendTransactions subscribes XRPL transactions and sends to the channel only valid transactions.
func (f *Finder) SubscribeTXSendTransactions(ctx context.Context, ch chan<- PendingTXSendTransaction) error {
	xrplTxsCh := make(chan xrpl.Transaction)
	if err := f.xrplScanner.Subscribe(
		ctx,
		f.cfg.XRPLIssuer,
		f.cfg.XRPLHistoryScanStartLedger,
		f.cfg.XRPLRecentScanIndexesBack,
		xrplTxsCh,
	); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case xrplTx := <-xrplTxsCh:
				pendingTx, matches := f.buildPendingTransaction(xrplTx)
				if !matches {
					continue
				}
				ch <- pendingTx
			}
		}
	}()

	return nil
}

func (f *Finder) buildPendingTransaction(txn xrpl.Transaction) (PendingTXSendTransaction, bool) {
	if !txn.Validated ||
		txn.TransactionResult != xrpl.TransactionResultSuccess ||
		txn.TransactionType != xrpl.TransactionTypePayment {
		return PendingTXSendTransaction{}, false
	}

	// check activation date
	if txn.Date.Before(f.cfg.ActivationDate) {
		return PendingTXSendTransaction{}, false
	}

	// extract destination if there is a memo
	if len(txn.Memos) == 0 {
		return PendingTXSendTransaction{}, false
	}
	txDestination, matches := ExtractAddressFromMemo(txn.Memos, f.cfg.XRPLMemoSuffix)
	if !matches {
		return PendingTXSendTransaction{}, false
	}
	// we don't include the native coins
	if txn.DeliveryAmount.IsNative() {
		return PendingTXSendTransaction{}, false
	}

	if txn.DeliveryAmount.Currency.String() != f.cfg.XRPLCurrency.String() ||
		txn.DeliveryAmount.Issuer.String() != f.cfg.XRPLIssuer.String() {
		return PendingTXSendTransaction{}, false
	}

	txCoin := f.convertXRPLAmountToTXCoin(txn.DeliveryAmount.Value)
	if txCoin.IsZero() {
		f.log.Info("Zero amount to send", zap.String("xrplTxHash", txn.Hash))
		return PendingTXSendTransaction{}, false
	}

	return PendingTXSendTransaction{
		TXDestination: txDestination,
		TXAmount:      txCoin,
		XRPLTxHash:    txn.Hash,
	}, true
}

func (f *Finder) convertXRPLAmountToTXCoin(xrplAmount *rippledata.Value) sdk.Coin {
	amount := ConvertXRPLAmountToTXAmount(xrplAmount, f.cfg.TXDecimals, f.cfg.Multiplier)
	return sdk.NewCoin(f.cfg.TXDenom, amount)
}

// ExtractAddressFromMemo extracts the TX blockchain sdk address from the transaction.
func ExtractAddressFromMemo(memos []string, suffix string) (sdk.AccAddress, bool) {
	for _, memo := range memos {
		if !strings.HasSuffix(memo, suffix) {
			continue
		}
		addressString := strings.TrimSuffix(memo, suffix)
		accAddress, err := sdk.AccAddressFromBech32(addressString)
		if err != nil {
			continue
		}

		return accAddress, true
	}

	return sdk.AccAddress{}, false
}

// ConvertXRPLAmountToTXAmount converts xrpl amount to TX using the TX decimals.
func ConvertXRPLAmountToTXAmount(xrplAmount *rippledata.Value, decimals int, multiplier string) sdkmath.Int {
	if xrplAmount == nil {
		return sdkmath.NewInt(0)
	}

	if len(multiplier) == 0 || multiplier == "0" {
		multiplier = "1.0"
	}

	multiplierRat, ok := new(big.Rat).SetString(multiplier)
	if !ok {
		return sdkmath.NewInt(0)
	}

	if multiplierRat.Num().Cmp(big.NewInt(0)) == 0 {
		multiplierRat = big.NewRat(1, 1)
	}

	// 10^TXDecimals
	tenPowerDecimals := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	xrplRatAmount := xrplAmount.Rat()
	xrplRatAmount = new(big.Rat).Mul(xrplRatAmount, multiplierRat)
	xrplRatAmountNumerator := xrplRatAmount.Num()
	xrplRatAmountDenominator := xrplRatAmount.Denom()
	txAmount := big.NewInt(0).Quo(
		big.NewInt(0).Mul(
			tenPowerDecimals, xrplRatAmountNumerator,
		),
		xrplRatAmountDenominator)

	return sdkmath.NewIntFromBigInt(txAmount)
}
