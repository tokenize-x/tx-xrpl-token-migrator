//nolint:tagliatelle //contract spec
package xrpl

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/gammazero/workerpool"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"go.uber.org/zap"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
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

// ******************** RPC method objects ********************

// AccountDataWithSigners is account data with signers model.
type AccountDataWithSigners struct {
	rippledata.AccountRoot
	SignerList []rippledata.SignerList `json:"signer_lists"`
}

// AccountInfoRequest is `account_info` method request.
type AccountInfoRequest struct {
	Account     rippledata.Account `json:"account"`
	SignerLists bool               `json:"signer_lists"`
}

// AccountInfoResult is `account_info` method result.
type AccountInfoResult struct {
	LedgerSequence uint32                 `json:"ledger_current_index"`
	AccountData    AccountDataWithSigners `json:"account_data"`
}

// SubmitRequest is `submit` method request.
type SubmitRequest struct {
	TxBlob string `json:"tx_blob"`
}

// SubmitResult is `submit` method result.
type SubmitResult struct {
	EngineResult        rippledata.TransactionResult `json:"engine_result"`
	EngineResultCode    int                          `json:"engine_result_code"`
	EngineResultMessage string                       `json:"engine_result_message"`
	TxBlob              string                       `json:"tx_blob"`
	Tx                  any                          `json:"tx_json"`
}

//nolint:tagliatelle //contract spec
type metaRes struct {
	DeliveredAmount   rippledata.Amount `json:"delivered_amount"`
	TransactionResult string            `json:"TransactionResult"`
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

type txResTransactionsItem struct {
	accountTxResTransactionTx
	Meta      metaRes `json:"meta"`
	Validated bool    `json:"validated"`
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

type txReq struct {
	Hash   string `json:"transaction"`
	Binary bool   `json:"binary"`
}

func convertHexMemosToStrings(memos []memoRes) ([]string, error) {
	memoStrings := make([]string, 0, len(memos))
	for _, memo := range memos {
		memoStr, err := hex.DecodeString(memo.Memo.MemoData)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to decode xrpl memo")
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

func convertTxInfoToTransaction(
	txn baseTx, meta metaRes, ledgerIndex int64, validated bool,
) (Transaction, bool, error) {
	memos, err := convertHexMemosToStrings(txn.Memos)
	if err != nil {
		return Transaction{}, false, err
	}

	return Transaction{
		Account:           txn.Account,
		Destination:       txn.Destination,
		DeliveryAmount:    meta.DeliveredAmount,
		Memos:             memos,
		Hash:              txn.Hash,
		TransactionType:   txn.TransactionType,
		TransactionResult: meta.TransactionResult,
		LedgerIndex:       ledgerIndex,
		Sequence:          txn.Sequence,
		Date:              convertXRPLDateToTime(txn.Date),
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
	URL            string
	PageLimit      int
	WorkerPoolSize int
}

// DefaultRPCClientConfig returns default RPCClientConfig.
func DefaultRPCClientConfig(url string) RPCClientConfig {
	return RPCClientConfig{
		URL:            url,
		PageLimit:      100,
		WorkerPoolSize: 20,
	}
}

// RPCClient implement the XRPL RPC client.
type RPCClient struct {
	cfg        RPCClientConfig
	log        logger.Logger
	httpClient HTTPClient
	workerPool *workerpool.WorkerPool
}

// NewRPCClient returns new instance of the RPCClient.
func NewRPCClient(cfg RPCClientConfig, log logger.Logger, httpClient HTTPClient) *RPCClient {
	return &RPCClient{
		cfg:        cfg,
		log:        log,
		httpClient: httpClient,
		workerPool: workerpool.New(cfg.WorkerPoolSize),
	}
}

// SubscribeAccountTransactions sends the list of all account transitions using pagination to the inout channel.
func (c *RPCClient) SubscribeAccountTransactions(
	ctx context.Context,
	account rippledata.Account,
	startLedger, endLedger int64,
	ch chan<- Transaction,
) (int64, error) {
	c.log.Debug(
		"Subscribing RPC account transactions",
		zap.String("account", account.String()),
		zap.Int64("startLedger", startLedger),
		zap.Int64("endLedger", endLedger),
	)

	var (
		marker         *PageMarker
		maxLatestIndex int64
	)
	for {
		nextMarker, latestIndex, err := c.GetAccountTransactions(ctx, account, startLedger, endLedger, marker, ch)
		// handing of case when last page is empty so no transactions are indexed
		if latestIndex > maxLatestIndex {
			maxLatestIndex = latestIndex
		}
		if err != nil {
			return maxLatestIndex, err
		}
		// reached the end
		if nextMarker.Seq == 0 && nextMarker.Ledger == 0 {
			return maxLatestIndex, nil
		}
		marker = nextMarker
	}
}

// GetAccountTransactions returns the list account transactions with fully filled delivery amount using pagination.
func (c *RPCClient) GetAccountTransactions(
	ctx context.Context,
	account rippledata.Account,
	startLedger,
	endLedger int64,
	marker *PageMarker,
	ch chan<- Transaction,
) (*PageMarker, int64, error) {
	c.log.Debug(
		"Getting account transactions",
		append(convertMarkerToZapFields(marker), zap.String("account", account.String()))...,
	)

	if endLedger <= 0 {
		endLedger = -1
	}
	if startLedger <= 0 {
		startLedger = -1
	}
	accountTxReqParam := accountTxReq{
		Account:        account.String(),
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
	err := c.callRPC(ctx, accountTxRPCReq, &accountTxRPCRes)
	latestIndex := startLedger
	if err != nil {
		return nil, latestIndex - 1, errors.Wrap(err, "failed to call `account_tx` method")
	}

	totalValidCount := 0
	for _, txItem := range accountTxRPCRes.Result.Transactions {
		latestIndex = txItem.Tx.LedgerIndex
		txn, ok, err := convertTxInfoToTransaction(txItem.Tx.baseTx, txItem.Meta, txItem.Tx.LedgerIndex, txItem.Validated)
		if err != nil {
			return nil, latestIndex - 1, err
		}
		// we keep only transactions which fully fill the expected transaction struct
		if !ok {
			continue
		}
		totalValidCount++

		select {
		case <-ctx.Done():
			return nil, 0, errors.WithStack(ctx.Err())
		case ch <- txn:
		}
	}

	marker = &PageMarker{
		Ledger: accountTxRPCRes.Result.Marker.Ledger,
		Seq:    accountTxRPCRes.Result.Marker.Seq,
	}
	c.log.Debug(
		"Got account transactions, and received next marker",
		append(convertMarkerToZapFields(marker),
			zap.Int("total", len(accountTxRPCRes.Result.Transactions)),
			zap.Int("totalValid", totalValidCount))...)

	return marker, latestIndex - 1, nil
}

// GetCurrentLedger returns the current ledger index.
func (c *RPCClient) GetCurrentLedger(ctx context.Context) (int64, error) {
	ledgerCurrentRPCReq := rpcReq{
		Method: "ledger_current",
	}

	var ledgerCurrentRPCRes rpcRes[ledgerCurrentTxRes]
	err := c.callRPC(ctx, ledgerCurrentRPCReq, &ledgerCurrentRPCRes)
	if err != nil {
		return 0, errors.Wrap(err, "failed to call `ledger_current` method")
	}

	return ledgerCurrentRPCRes.Result.LedgerCurrentIndex, nil
}

// GetTransactions return the transactions by hashes.
func (c *RPCClient) GetTransactions(ctx context.Context, hashes []string) (map[string]Transaction, error) {
	txs := make(map[string]Transaction, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(len(hashes))
	var fetchErr error
	for _, hash := range hashes {
		hashToFetch := hash
		c.workerPool.Submit(func() {
			defer wg.Done()
			// if the sub-process is already set error no need to continue
			if fetchErr != nil {
				return
			}
			txn, ok, err := c.GetTransaction(ctx, hashToFetch)
			if err != nil {
				fetchErr = err
				return
			}
			if !ok {
				c.log.Warn("The transaction will be skipped, since its data isn't fully valid", zap.String("hash", hashToFetch))
			}
			mu.Lock()
			defer mu.Unlock()
			txs[hashToFetch] = txn
		})
	}
	wg.Wait()
	if fetchErr != nil {
		return nil, fetchErr
	}

	return txs, nil
}

// GetTransaction returns transaction by its hash.
func (c *RPCClient) GetTransaction(ctx context.Context, hash string) (Transaction, bool, error) {
	txRPCReq := rpcReq{
		Method: "tx",
		Params: []any{
			txReq{
				Hash:   strings.ToUpper(hash),
				Binary: false,
			},
		},
	}

	var txRPCRes rpcRes[txResTransactionsItem]
	if err := c.callRPC(ctx, txRPCReq, &txRPCRes); err != nil {
		return Transaction{}, false, errors.Wrap(err, "failed to call `tx` method")
	}

	return convertTxInfoToTransaction(
		txRPCRes.Result.baseTx,
		txRPCRes.Result.Meta,
		txRPCRes.Result.LedgerIndex,
		txRPCRes.Result.Validated,
	)
}

// AccountInfo returns the account information for the given account.
func (c *RPCClient) AccountInfo(ctx context.Context, acc rippledata.Account) (AccountInfoResult, error) {
	req := rpcReq{
		Method: "account_info",
		Params: []any{
			AccountInfoRequest{
				Account:     acc,
				SignerLists: true,
			},
		},
	}

	var result rpcRes[AccountInfoResult]
	if err := c.callRPC(ctx, req, &result); err != nil {
		return AccountInfoResult{}, err
	}

	return result.Result, nil
}

// AutoFillTx add seq number and fee for the transaction.
func (c *RPCClient) AutoFillTx(
	ctx context.Context,
	txn rippledata.Transaction,
	sender rippledata.Account,
	txSignatureCount uint32,
) error {
	accInfo, err := c.AccountInfo(ctx, sender)
	if err != nil {
		return err
	}
	// update base settings
	base := txn.GetBase()
	fee, err := c.calculateFee(txSignatureCount, DefaultXRPLBaseFee)
	if err != nil {
		return err
	}
	base.Fee = *fee
	base.Account = sender
	base.Sequence = *accInfo.AccountData.Sequence

	return nil
}

// SubmitAndAwaitSuccess submits tx a waits for its result, if result is not success returns an error.
func (c *RPCClient) SubmitAndAwaitSuccess(ctx context.Context, txn rippledata.Transaction) error {
	c.log.Info("Submitting XRPL transaction", zap.String("txHash", strings.ToUpper(txn.GetHash().String())))
	// submit the transaction
	res, err := c.Submit(ctx, txn)
	if err != nil {
		return err
	}
	if !res.EngineResult.Success() {
		return errors.Errorf("the tx submition is failed, %+v", res)
	}

	retryCtx, retryCtxCancel := context.WithTimeout(ctx, time.Minute)
	defer retryCtxCancel()
	c.log.Info(
		"Transaction is submitted, waiting for tx to be accepted",
		zap.String("txHash", strings.ToUpper(txn.GetHash().String())),
	)
	return retry.Do(retryCtx, 250*time.Millisecond, func() error {
		reqCtx, reqCtxCancel := context.WithTimeout(ctx, 3*time.Second)
		defer reqCtxCancel()
		txRes, ok, err := c.GetTransaction(reqCtx, txn.GetHash().String())
		if err != nil {
			return retry.Retryable(err)
		}
		if !ok {
			return retry.Retryable(errors.Errorf("transaction is not valid"))
		}
		if !txRes.Validated {
			return retry.Retryable(errors.Errorf("transaction is not validated"))
		}
		return nil
	})
}

// Submit submits a transaction to the RPC server.
func (c *RPCClient) Submit(ctx context.Context, txn rippledata.Transaction) (SubmitResult, error) {
	_, raw, err := rippledata.Raw(txn)
	if err != nil {
		return SubmitResult{}, errors.Wrapf(err, "failed to convert transaction to raw data")
	}
	req := rpcReq{
		Method: "submit",
		Params: []any{
			SubmitRequest{
				TxBlob: fmt.Sprintf("%X", raw),
			},
		},
	}
	var result rpcRes[SubmitResult]

	if err := c.callRPC(ctx, req, &result); err != nil {
		return SubmitResult{}, err
	}

	return result.Result, nil
}

func (c *RPCClient) callRPC(ctx context.Context, req rpcReq, res any) error {
	c.log.Debug("Executing XRPL RPC request", zap.Any("request", req))

	err := c.httpClient.DoJSON(ctx, http.MethodPost, c.cfg.URL, req, func(resBytes []byte) error {
		var rpcErrRes rpcRes[rpcErrResult]
		if err := json.Unmarshal(resBytes, &rpcErrRes); err != nil {
			return errors.Wrapf(err, "failed to decode http result to error result, raw http result:%s", string(resBytes))
		}
		if rpcErrRes.Result.ErrorCode != 0 {
			return errors.Errorf(
				"failed to call xrpl RPC, error:%s, error code:%d, error message:%s",
				rpcErrRes.Result.Error, rpcErrRes.Result.ErrorCode, rpcErrRes.Result.ErrorMessage,
			)
		}

		if err := json.Unmarshal(resBytes, res); err != nil {
			return errors.Wrapf(err, "failed to decode http result to expected struct, raw http result:%s", string(resBytes))
		}

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to call xrpl RPC")
	}

	return nil
}

func (c *RPCClient) calculateFee(txSignatureCount, baseFee uint32) (*rippledata.Value, error) {
	if txSignatureCount == 0 {
		return nil, errors.New("tx signature count must be greater than 0")
	} else if txSignatureCount == 1 {
		// Single sig: base_fee
		return rippledata.NewNativeValue(int64(baseFee))
	}

	// Multisig: base_fee × (1 + Number of Signatures Provided)
	return rippledata.NewNativeValue(int64(baseFee * (1 + txSignatureCount)))
}
