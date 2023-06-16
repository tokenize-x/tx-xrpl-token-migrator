package xrpl

import (
	"math/big"
	"time"
)

const (
	// TransactionTypePayment is Payment type of the transaction.
	TransactionTypePayment = "Payment"
	// TransactionResultSuccess is success result of the transaction.
	TransactionResultSuccess = "tesSUCCESS"
	// MessageTypeTransaction is transaction message type.
	MessageTypeTransaction = "transaction"
)

// DeliveredAmount is delivered amount of the transaction.
type DeliveredAmount struct {
	Currency string // the currency might be empty if it's native XRP coin
	Issuer   string
	Value    *big.Float
}

// Transaction is general transaction struct.
type Transaction struct { //nolint:musttag //json used in the tests only
	Account           string
	Destination       string
	DeliveryAmount    DeliveredAmount
	Memos             []string
	Hash              string
	TransactionType   string
	TransactionResult string
	LedgerIndex       int64
	Sequence          int64
	Date              time.Time
	Validated         bool
}
