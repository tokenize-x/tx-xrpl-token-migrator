package xrpl

import (
	"time"

	rippledata "github.com/rubblelabs/ripple/data"
)

const (
	// TransactionTypePayment is Payment type of the transaction.
	TransactionTypePayment = "Payment"
	// TransactionResultSuccess is success result of the transaction.
	TransactionResultSuccess = "tesSUCCESS"
)

// Transaction is general transaction struct.
type Transaction struct { //nolint:musttag //json used in the tests only
	Account           string
	Destination       string
	DeliveryAmount    rippledata.Amount
	Memos             []string
	Hash              string
	TransactionType   string
	TransactionResult string
	LedgerIndex       int64
	Sequence          int64
	Date              time.Time
	Validated         bool
}
