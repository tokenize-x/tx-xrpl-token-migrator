package xrpl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
)

// ******************** RPC transport objects ********************

type rpcReq struct {
	Method string `json:"method"`
	Params []any  `json:"params,omitempty"`
}

type rpcRes[T any] struct {
	Result T `json:"result"`
}

//nolint:tagliatelle //contract spec
type rpcErrResult struct {
	Error        string `json:"error"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// ******************** Unexported common ********************

type metaDeliveredAmount struct {
	Currency string     `json:"currency"`
	Issuer   string     `json:"issuer"`
	Value    *big.Float `json:"value,string"` //nolint:staticcheck // expected string tag
}

//nolint:tagliatelle //contract spec
type metaRes struct {
	DeliveredAmount   json.RawMessage `json:"delivered_amount"` // can be float string or metaDeliveredAmount
	TransactionResult string          `json:"TransactionResult"`
}

//nolint:tagliatelle //contract spec
type memoItemRes struct {
	MemoData string `json:"MemoData"` // hex string
	MemoType string `json:"MemoType"` // hex string
}

//nolint:tagliatelle //contract spec
type memoRes struct {
	Memo memoItemRes `json:"Memo"`
}

//nolint:tagliatelle //contract spec
type baseTx struct {
	Account         string    `json:"Account"`
	Destination     string    `json:"Destination"`
	Hash            string    `json:"hash"`
	TransactionType string    `json:"TransactionType"`
	Date            int       `json:"date"`
	Sequence        int64     `json:"Sequence"`
	Memos           []memoRes `json:"memos"`
}

//nolint:tagliatelle //contract spec
type accountTxResTransactionTx struct {
	baseTx
	LedgerIndex int64 `json:"ledger_index"`
}

type pageMarker struct {
	Ledger int64 `json:"ledger"`
	Seq    int   `json:"seq"`
}

//nolint:tagliatelle //contract spec
type accountTxReq struct {
	Account        string      `json:"account"`
	Binary         bool        `json:"binary"`
	Forward        bool        `json:"forward"`
	LedgerIndexMin int64       `json:"ledger_index_min"`
	LedgerIndexMax int64       `json:"ledger_index_max"`
	Limit          int         `json:"limit"`
	Marker         *pageMarker `json:"marker,omitempty"`
}

type accountTxResTransactionsItem struct {
	Meta      metaRes                   `json:"meta"`
	Tx        accountTxResTransactionTx `json:"tx"`
	Validated bool                      `json:"validated"`
}

type accountTxRes struct {
	Transactions []accountTxResTransactionsItem `json:"transactions"`
	Marker       pageMarker                     `json:"marker"`
}

//nolint:tagliatelle //contract spec
type ledgerCurrentTxRes struct {
	LedgerCurrentIndex int64  `json:"ledger_current_index"`
	Status             string `json:"status"`
}

func convertJSONToDeliveredAmount(amount json.RawMessage) (DeliveredAmount, bool) {
	var amt metaDeliveredAmount
	err := json.Unmarshal(amount, &amt)
	if err != nil {
		return DeliveredAmount{}, false
	}

	return DeliveredAmount(amt), true
}

func convertHexMemosToStrings(memos []memoRes) ([]string, error) {
	memoStrings := make([]string, 0, len(memos))
	for _, memo := range memos {
		memoStr, err := hex.DecodeString(memo.Memo.MemoData)
		if err != nil {
			return nil, errors.Wrapf(err, "can't decode xrpl memo")
		}
		memoStrings = append(memoStrings, string(memoStr))
	}

	return memoStrings, nil
}

func convertXRPLDateToTime(date int) time.Time {
	txTime := time.Date(2000, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
	return txTime.Add(time.Duration(date) * time.Second)
}

func convertMarkerToZapFields(marker *PageMarker) []zap.Field {
	fields := make([]zap.Field, 0)
	if marker == nil {
		return fields
	}
	fields = append(
		fields,
		zap.Int64("marker.Ledger", marker.Ledger),
		zap.Int("marker.Seq", marker.Seq),
	)
	return fields
}

func convertTxInfoToTransaction(tx baseTx, meta metaRes, ledgerIndex int64, validated bool) (Transaction, bool, error) {
	memos, err := convertHexMemosToStrings(tx.Memos)
	if err != nil {
		return Transaction{}, false, err
	}

	deliveredAmount, ok := convertJSONToDeliveredAmount(meta.DeliveredAmount)
	if !ok {
		return Transaction{}, false, nil
	}
	return Transaction{
		Account:           tx.Account,
		Destination:       tx.Destination,
		DeliveryAmount:    deliveredAmount,
		Memos:             memos,
		Hash:              tx.Hash,
		TransactionType:   tx.TransactionType,
		TransactionResult: meta.TransactionResult,
		LedgerIndex:       ledgerIndex,
		Sequence:          tx.Sequence,
		Date:              convertXRPLDateToTime(tx.Date),
		Validated:         validated,
	}, true, nil
}

// ******************** XRPL RPC Client ********************

// HTTPClient is HTTP client interface.
type HTTPClient interface {
	DoJSON(ctx context.Context, method, url string, reqBody any, resDecoder func([]byte) error) error
}

// PageMarker is the pagination for the RPC client.
type PageMarker struct {
	Ledger int64
	Seq    int
}

// RPCClientConfig defines the config for the RPCClient.
type RPCClientConfig struct {
	URL       string
	PageLimit int
}

// DefaultRPCClientConfig returns default RPCClientConfig.
func DefaultRPCClientConfig(url string) RPCClientConfig {
	return RPCClientConfig{
		URL:       url,
		PageLimit: 100,
	}
}

// RPCClient implement the XRPL RPC client.
type RPCClient struct {
	cfg        RPCClientConfig
	httpClient HTTPClient
}

// NewRPCClient returns new instance of the RPCClient.
func NewRPCClient(cfg RPCClientConfig, httpClient HTTPClient) *RPCClient {
	return &RPCClient{
		cfg:        cfg,
		httpClient: httpClient,
	}
}

// SubscribeAccountTransactions sends the list of all account transitions using pagination to the inout channel.
func (c *RPCClient) SubscribeAccountTransactions(
	ctx context.Context,
	account string,
	startLedger, endLedger int64,
	ch chan<- Transaction,
) (int64, error) {
	log := logger.Get(ctx)
	log.Debug(
		"Subscribing RPC account transactions",
		zap.String("account", account),
		zap.Int64("startLedger", startLedger),
		zap.Int64("endLedger", endLedger),
	)

	var marker *PageMarker
	for {
		nextMarker, latestIndex, err := c.GetAccountTransactions(ctx, account, startLedger, endLedger, marker, ch)
		if err != nil {
			return latestIndex, err
		}
		// reached the end
		if nextMarker.Seq == 0 && nextMarker.Ledger == 0 {
			return latestIndex, nil
		}
		marker = nextMarker
	}
}

// GetAccountTransactions returns the list account transactions with fully filled delivery amount using pagination.
func (c *RPCClient) GetAccountTransactions(ctx context.Context, account string, startLedger, endLedger int64, marker *PageMarker, ch chan<- Transaction) (*PageMarker, int64, error) {
	log := logger.Get(ctx)
	log.Debug(
		"Getting account transactions",
		append(convertMarkerToZapFields(marker), zap.String("account", account))...,
	)

	if endLedger <= 0 {
		endLedger = -1
	}
	if startLedger <= 0 {
		startLedger = -1
	}
	accountTxReqParam := accountTxReq{
		Account:        account,
		Binary:         false,
		Forward:        true,
		LedgerIndexMin: startLedger,
		LedgerIndexMax: endLedger,
		Limit:          c.cfg.PageLimit,
	}
	if marker != nil {
		accountTxReqParam.Marker = &pageMarker{
			Ledger: marker.Ledger,
			Seq:    marker.Seq,
		}
	}
	accountTxRPCReq := rpcReq{
		Method: "account_tx",
		Params: []any{
			accountTxReqParam,
		},
	}

	var accountTxRPCRes rpcRes[accountTxRes]
	err := c.callPRC(ctx, accountTxRPCReq, &accountTxRPCRes)
	latestIndex := startLedger
	if err != nil {
		return nil, latestIndex - 1, errors.Wrap(err, "can't call `account_tx` method")
	}

	for _, txItem := range accountTxRPCRes.Result.Transactions {
		latestIndex = txItem.Tx.LedgerIndex
		tx, ok, err := convertTxInfoToTransaction(txItem.Tx.baseTx, txItem.Meta, txItem.Tx.LedgerIndex, txItem.Validated)
		if err != nil {
			return nil, latestIndex - 1, err
		}
		// we keep only transactions which fully fill the expected transaction struct
		if !ok {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, 0, errors.WithStack(ctx.Err())
		case ch <- tx:
		}
	}

	marker = &PageMarker{
		Ledger: accountTxRPCRes.Result.Marker.Ledger,
		Seq:    accountTxRPCRes.Result.Marker.Seq,
	}
	log.Debug("Got account transactions, and received next marker", convertMarkerToZapFields(marker)...)

	return marker, latestIndex - 1, nil
}

// GetCurrentLedger returns the current ledger index.
func (c *RPCClient) GetCurrentLedger(ctx context.Context) (int64, error) {
	ledgerCurrentRPCReq := rpcReq{
		Method: "ledger_current",
	}

	var ledgerCurrentRPCRes rpcRes[ledgerCurrentTxRes]
	err := c.callPRC(ctx, ledgerCurrentRPCReq, &ledgerCurrentRPCRes)
	if err != nil {
		return 0, errors.Wrap(err, "can't call `ledger_current` method")
	}

	return ledgerCurrentRPCRes.Result.LedgerCurrentIndex, nil
}

func (c *RPCClient) callPRC(ctx context.Context, req rpcReq, res any) error {
	err := c.httpClient.DoJSON(ctx, http.MethodPost, c.cfg.URL, req, func(resBytes []byte) error {
		var rpcErrRes rpcErrResult
		if err := json.Unmarshal(resBytes, &rpcErrRes); err != nil {
			return errors.Wrapf(err, "can't decode http result to error result, raw http result:%s", string(resBytes))
		}
		if rpcErrRes.ErrorCode != 0 {
			return errors.Errorf("can't call xrpl RPC, error:%s, error code:%d, error message:%s", rpcErrRes.Error, rpcErrRes.ErrorCode, rpcErrRes.ErrorMessage)
		}

		if err := json.Unmarshal(resBytes, res); err != nil {
			return errors.Wrapf(err, "can't decode http result to expected struct, raw http result:%s", string(resBytes))
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "can't call xrpl RPC")
	}

	return nil
}
