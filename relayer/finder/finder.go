package finder

import (
	"context"
	"math/big"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/xrpl"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"
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

	if txn.Destination != f.cfg.XRPLIssuer.String() {
		return PendingTXSendTransaction{}, false
	}

	// Confirm the delivered amount was really burned by checking the tx's ledger changes,
	// not just the reported delivered amount.
	txWithMeta := rippledata.TransactionWithMetaData{
		Transaction: &rippledata.Payment{TxBase: rippledata.TxBase{TransactionType: rippledata.PAYMENT}},
		MetaData:    rippledata.MetaData{AffectedNodes: txn.AffectedNodes},
	}
	burned, err := receivedAtLeastDeliveredAmount(txWithMeta, f.cfg.XRPLIssuer, txn.DeliveryAmount)
	if err != nil {
		// We could not verify the burn from the ledger; the tokens may already be burned.
		// It must be checked by operator.
		f.log.Error(
			"Skipping tx: could not verify the burn against the ledger metadata",
			zap.String("xrplTxHash", txn.Hash),
			zap.Error(err),
		)
		return PendingTXSendTransaction{}, false
	}
	if !burned {
		f.log.Error(
			"Skipping tx: delivered amount is not backed by an on-ledger burn",
			zap.String("xrplTxHash", txn.Hash),
			zap.String("deliveredAmount", txn.DeliveryAmount.String()),
		)
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

// balanceReconciliationEpsilon is the rounding tolerance for issued tokens, as a fraction of the balance.
// XRPL keeps ~15-16 significant digits, so 1e-14 is above that yet below any real amount.
var balanceReconciliationEpsilon = big.NewRat(1, 100_000_000_000_000)

// receivedAtLeastDeliveredAmount reports whether account's balance change in the delivered currency
// (from the tx's AffectedNodes) is at least the delivered amount.
// For a burn, account is the issuer: the token is returned to it, so its balance must rise by the
// delivered amount. Receiving more is fine.
func receivedAtLeastDeliveredAmount(
	txn rippledata.TransactionWithMetaData,
	account rippledata.Account,
	deliveredAmount rippledata.Amount,
) (bool, error) {
	balances, err := txn.Balances()
	if err != nil {
		return false, errors.Wrap(err, "failed to compute balance changes from the XRPL tx metadata")
	}

	accountBalances, ok := balances[account]
	if !ok || accountBalances == nil {
		return false, nil
	}

	var total rippledata.Value
	found := false
	maxBalanceMag := new(big.Rat)
	for _, balance := range *accountBalances {
		if !balance.Currency.Equals(deliveredAmount.Currency) {
			continue
		}
		// For a token the account does not issue, it holds it via a trust line, so the counterparty must
		// be the issuer. The account's own issued token has no such constraint.
		if !deliveredAmount.IsNative() &&
			!deliveredAmount.Issuer.Equals(account) &&
			!balance.CounterParty.Equals(deliveredAmount.Issuer) {
			continue
		}

		if mag := new(big.Rat).Abs(balance.Balance.Rat()); mag.Cmp(maxBalanceMag) > 0 {
			maxBalanceMag = mag
		}
		if !found {
			total = balance.Change
			found = true
			continue
		}
		sum, err := total.Add(balance.Change)
		if err != nil {
			return false, errors.Wrap(err, "failed to sum the balance changes")
		}
		total = *sum
	}

	if !found {
		return false, nil
	}

	if total.Compare(*deliveredAmount.Value) >= 0 {
		return true, nil
	}

	// XRP is exact, so any shortfall is real.
	if deliveredAmount.IsNative() {
		return false, nil
	}

	// Allow a shortfall only up to the rounding tolerance.
	// Scale it to the larger of the balance and the delivered amount, since a full burn ends at 0.
	scale := maxBalanceMag
	if d := new(big.Rat).Abs(deliveredAmount.Rat()); d.Cmp(scale) > 0 {
		scale = d
	}
	shortfall := new(big.Rat).Sub(deliveredAmount.Rat(), total.Rat())
	tolerance := new(big.Rat).Mul(scale, balanceReconciliationEpsilon)
	return shortfall.Cmp(tolerance) <= 0, nil
}
