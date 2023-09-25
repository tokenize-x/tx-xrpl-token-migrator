package finder

import (
	"context"
	"math/big"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/xrpl"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// PendingCoreumSendTransaction represents the pending transaction to be sent to the coreum.
type PendingCoreumSendTransaction struct {
	CoreumDestination sdk.AccAddress
	CoreumAmount      sdk.Coin
	XRPLTxHash        string
}

// XRPLScanner is XRPL scanner which provides XRPL transactions.
type XRPLScanner interface {
	Subscribe(ctx context.Context, account string, historyScanStartLedger, recentScanIndexesBack int64, ch chan<- xrpl.Transaction) error
}

// Config is Finder config.
type Config struct {
	XRPLIssuer                 string
	XRPLCurrency               string
	XRPLHistoryScanStartLedger int64
	XRPLRecentScanIndexesBack  int64
	XRPLMemoSuffix             string

	CoreumDenom    string
	CoreumDecimals int
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

// SubscribeCoreumSendTransactions subscribes XRPL transactions and sends to the channel only valid transactions.
func (f *Finder) SubscribeCoreumSendTransactions(ctx context.Context, ch chan<- PendingCoreumSendTransaction) error {
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

func (f *Finder) buildPendingTransaction(tx xrpl.Transaction) (PendingCoreumSendTransaction, bool) {
	if !tx.Validated ||
		tx.TransactionResult != xrpl.TransactionResultSuccess ||
		tx.TransactionType != xrpl.TransactionTypePayment {
		return PendingCoreumSendTransaction{}, false
	}
	if len(tx.Memos) == 0 {
		return PendingCoreumSendTransaction{}, false
	}

	coreumDestination, matches := ExtractAddressFromMemo(tx.Memos, f.cfg.XRPLMemoSuffix)
	if !matches {
		return PendingCoreumSendTransaction{}, false
	}

	if tx.DeliveryAmount.Currency != f.cfg.XRPLCurrency || tx.DeliveryAmount.Issuer != f.cfg.XRPLIssuer {
		return PendingCoreumSendTransaction{}, false
	}

	coreumCoin := f.convertXRPLAmountToCoreumCoin(tx.DeliveryAmount.Value)
	if coreumCoin.IsZero() {
		f.log.Info("Zero amount to send", zap.String("xrplTxHash", tx.Hash))
		return PendingCoreumSendTransaction{}, false
	}

	return PendingCoreumSendTransaction{
		CoreumDestination: coreumDestination,
		CoreumAmount:      coreumCoin,
		XRPLTxHash:        tx.Hash,
	}, true
}

func (f *Finder) convertXRPLAmountToCoreumCoin(xrplAmount *big.Float) sdk.Coin {
	amount := ConvertXRPLAmountToCoreumAmount(xrplAmount, f.cfg.CoreumDecimals)
	return sdk.NewCoin(f.cfg.CoreumDenom, amount)
}

// ExtractAddressFromMemo extracts the coreum sdk address from the transaction.
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

// ConvertXRPLAmountToCoreumAmount converts xrpl amount to coreum using the coreum decimals.
func ConvertXRPLAmountToCoreumAmount(xrplAmount *big.Float, decimals int) sdk.Int {
	if xrplAmount == nil {
		return sdk.NewInt(0)
	}

	// 10^CoreumDecimals
	var tenPowerDecimals big.Int
	tenPowerDecimals.Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)

	var coreumFloatAmount big.Float
	coreumFloatAmount.Mul(big.NewFloat(0).SetInt(&tenPowerDecimals), xrplAmount)

	truncatedAmount, _ := coreumFloatAmount.Int(nil)
	return sdk.NewIntFromBigInt(truncatedAmount)
}
