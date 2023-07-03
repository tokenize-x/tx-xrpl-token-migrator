package coreum

import (
	"context"
	"encoding/json"
	"strings"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"github.com/CoreumFoundation/coreum/pkg/client"
	"github.com/CoreumFoundation/coreum/testutil/event"
	contractembed "github.com/CoreumFoundation/xrpl-bridge/contract"
)

type method string

const (
	methodThresholdBankSend method = "threshold_bank_send"
	methodWithdraw          method = "withdraw"

	methodGetPendingTransaction  method = "get_pending_transaction"
	methodGetPendingTransactions method = "get_pending_transactions"
	methodGetSentTransaction     method = "get_sent_transaction"
	methodGetSentTransactions    method = "get_sent_transactions"
)

const (
	unauthorizedErrorString     = "Unauthorized"
	evidenceProvidedErrorString = "Sender already provided the evidence"
	transferSentErrorString     = "Transfer already sent"
)

// DeployAndInstantiateConfig holds attributes used for the contract deployment and instantiation.
type DeployAndInstantiateConfig struct {
	Owner            string
	Admin            string
	TrustedAddresses []string
	Threshold        int
	Label            string
}

// ThresholdBankSendRequest holds attributes for the send transaction.
type ThresholdBankSendRequest struct {
	ID        string   `json:"id"`
	Amount    sdk.Coin `json:"amount"`
	Recipient string   `json:"recipient"`
}

// Transaction represents the transaction model.
type Transaction struct {
	Amount            sdk.Coin `json:"amount"`
	Recipient         string   `json:"recipient"`
	EvidenceProviders []string `json:"evidence_providers"` //nolint:tagliatelle //contract spec
}

// PendingTransaction represents the pending transaction model.
type PendingTransaction struct {
	EvidenceID string `json:"evidence_id"` //nolint:tagliatelle //contract spec
	Transaction
}

// SentTransaction represents the sent transaction model.
type SentTransaction struct {
	ID string `json:"id"`
	Transaction
}

type instantiateRequest struct {
	Owner            string   `json:"owner"`
	TrustedAddresses []string `json:"trusted_addresses"` //nolint:tagliatelle //contract spec
	Threshold        int      `json:"threshold"`
}

type queryTxsResponse[T any] struct {
	Transactions []T `json:"transactions"`
}

type pendingTxQueryRequest struct {
	EvidenceID string `json:"evidence_id"` //nolint:tagliatelle //contract spec
}

type pagingRequest struct {
	Offset *uint64 `json:"offset"`
	Limit  *uint32 `json:"limit"`
}

type sentTxQueryRequest struct {
	ID string `json:"id"`
}

// ******************** Client ********************

// ContractClientConfig represent the ContractClient config.
type ContractClientConfig struct {
	ContractAddress sdk.AccAddress
	GasMultiplier   float64
}

// DefaultContractClientConfig returns default ContractClient config.
func DefaultContractClientConfig(contractAddress sdk.AccAddress) ContractClientConfig {
	return ContractClientConfig{
		ContractAddress: contractAddress,
		GasMultiplier:   1.3,
	}
}

// ContractClient is the wasm contract client.
type ContractClient struct {
	cfg        ContractClientConfig
	clientCtx  client.Context
	wasmClient wasmtypes.QueryClient
}

// NewContractClient returns a new instance of the ContractClient.
func NewContractClient(cfg ContractClientConfig, clientCtx client.Context) *ContractClient {
	return &ContractClient{
		cfg:        cfg,
		clientCtx:  clientCtx.WithBroadcastMode(flags.BroadcastBlock),
		wasmClient: wasmtypes.NewQueryClient(clientCtx),
	}
}

// DeployAndInstantiate deploys the contract bytecode and instantiate it.
func (c *ContractClient) DeployAndInstantiate(ctx context.Context, sender sdk.AccAddress, config DeployAndInstantiateConfig) (sdk.AccAddress, error) {
	msgStoreCode := &wasmtypes.MsgStoreCode{
		Sender:       sender.String(),
		WASMByteCode: contractembed.Bytecode,
	}

	res, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msgStoreCode)
	if err != nil {
		return nil, errors.Wrap(err, "can't deploy wasm bytecode")
	}
	codeID, err := event.FindUint64EventAttribute(res.Events, wasmtypes.EventTypeStoreCode, wasmtypes.AttributeKeyCodeID)
	if err != nil {
		return nil, errors.Wrap(err, "can't find code ID in the tx result")
	}

	reqPayload, err := json.Marshal(instantiateRequest{
		Owner:            config.Owner,
		TrustedAddresses: config.TrustedAddresses,
		Threshold:        config.Threshold,
	})
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal instantiate payload")
	}

	msg := &wasmtypes.MsgInstantiateContract{
		Sender: sender.String(),
		Admin:  config.Admin,
		CodeID: codeID,
		Label:  config.Label,
		Msg:    wasmtypes.RawContractMessage(reqPayload),
	}

	res, err = client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msg)
	if err != nil {
		return nil, errors.Wrap(err, "can't instantiate bytecode")
	}

	contractAddr, err := event.FindStringEventAttribute(res.Events, wasmtypes.EventTypeInstantiate, wasmtypes.AttributeKeyContractAddr)
	if err != nil {
		return nil, errors.Wrap(err, "can't find contract address in the tx result")
	}

	sdkContractAddr, err := sdk.AccAddressFromBech32(contractAddr)
	if err != nil {
		return nil, errors.Wrap(err, "can't convert contrac address to sdk.AccAddress")
	}

	return sdkContractAddr, nil
}

// SetContractAddress sets the client contract address if it was not set before.
func (c *ContractClient) SetContractAddress(contractAddress sdk.AccAddress) error {
	if c.cfg.ContractAddress != nil {
		return errors.New("contract address is already set")
	}

	c.cfg.ContractAddress = contractAddress

	return nil
}

// ThresholdBankSend executes threshold_bank_send method of the contract.
func (c *ContractClient) ThresholdBankSend(ctx context.Context, sender sdk.AccAddress, requests ...ThresholdBankSendRequest) (*sdk.TxResponse, error) {
	msgs := make([]interface{}, 0)
	for _, req := range requests {
		msgs = append(msgs, map[method]ThresholdBankSendRequest{
			methodThresholdBankSend: {
				ID:        req.ID,
				Amount:    req.Amount,
				Recipient: req.Recipient,
			},
		})
	}
	txRes, err := c.execute(ctx, sender, msgs...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s method", methodThresholdBankSend)
	}

	return txRes, nil
}

// Withdraw executes withdraw method of the contract with will send the coins from the contract to recipient.
func (c *ContractClient) Withdraw(ctx context.Context, sender sdk.AccAddress) (*sdk.TxResponse, error) {
	txRes, err := c.execute(ctx, sender, map[method]struct{}{
		methodWithdraw: {},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s method", methodWithdraw)
	}

	return txRes, nil
}

// GetPendingTx returns a pending transaction.
func (c *ContractClient) GetPendingTx(ctx context.Context, evidenceID string) (Transaction, error) {
	var tx Transaction
	err := c.query(ctx, map[method]pendingTxQueryRequest{
		methodGetPendingTransaction: {
			EvidenceID: evidenceID,
		},
	}, &tx)
	if err != nil {
		return Transaction{}, errors.Wrapf(err, "failed to query %s", methodGetPendingTransaction)
	}

	return tx, nil
}

// GetPendingTxs returns a list of pending transactions.
func (c *ContractClient) GetPendingTxs(ctx context.Context, offset *uint64, limit *uint32) ([]PendingTransaction, error) {
	var txs queryTxsResponse[PendingTransaction]
	err := c.query(ctx, map[method]pagingRequest{
		methodGetPendingTransactions: {
			Offset: offset,
			Limit:  limit,
		},
	}, &txs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query %s", methodGetPendingTransactions)
	}

	return txs.Transactions, nil
}

// GetSentTx returns a sent transaction.
func (c *ContractClient) GetSentTx(ctx context.Context, id string) (Transaction, error) {
	var tx Transaction
	err := c.query(ctx, map[method]sentTxQueryRequest{
		methodGetSentTransaction: {
			ID: id,
		},
	}, &tx)
	if err != nil {
		return Transaction{}, errors.Wrapf(err, "failed to query %s", methodGetSentTransaction)
	}

	return tx, nil
}

// GetSentTxs returns a list of sent transactions.
func (c *ContractClient) GetSentTxs(ctx context.Context, offset *uint64, limit *uint32) ([]SentTransaction, error) {
	var txs queryTxsResponse[SentTransaction]
	err := c.query(ctx, map[method]pagingRequest{
		methodGetSentTransactions: {
			Offset: offset,
			Limit:  limit,
		},
	}, &txs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query %s", methodGetSentTransactions)
	}

	return txs.Transactions, nil
}

// IsUnauthorizedError returns true if error is Unauthorized error.
func IsUnauthorizedError(err error) bool {
	return isError(err, unauthorizedErrorString)
}

// IsEvidenceProvidedError returns true if error is EvidenceProvided error.
func IsEvidenceProvidedError(err error) bool {
	return isError(err, evidenceProvidedErrorString)
}

// IsTransferSentError returns true is error is TransferSent error.
func IsTransferSentError(err error) bool {
	return isError(err, transferSentErrorString)
}

func isError(err error, errorString string) bool {
	return err != nil && strings.Contains(err.Error(), errorString)
}

func (c *ContractClient) execute(ctx context.Context, sender sdk.AccAddress, requests ...any) (*sdk.TxResponse, error) {
	if c.cfg.ContractAddress == nil {
		return nil, errors.New("failed to execute with empty contract address")
	}

	msgs := make([]sdk.Msg, 0, len(requests))
	for _, req := range requests {
		payload, err := json.Marshal(req)
		if err != nil {
			return nil, errors.Wrap(err, "can't marshal payload")
		}
		msg := &wasmtypes.MsgExecuteContract{
			Sender:   sender.String(),
			Contract: c.cfg.ContractAddress.String(),
			Msg:      wasmtypes.RawContractMessage(payload),
		}
		msgs = append(msgs, msg)
	}

	res, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msgs...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ContractClient) txFactory() client.Factory {
	return client.Factory{}.
		WithKeybase(c.clientCtx.Keyring()).
		WithChainID(c.clientCtx.ChainID()).
		WithTxConfig(c.clientCtx.TxConfig()).
		WithGasAdjustment(c.cfg.GasMultiplier).
		WithSimulateAndExecute(true)
}

func (c *ContractClient) query(ctx context.Context, request, response any) error {
	if c.cfg.ContractAddress == nil {
		return errors.New("failed to execute with empty contract address")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return errors.Wrapf(err, "can't marshal query request")
	}

	query := &wasmtypes.QuerySmartContractStateRequest{
		Address:   c.cfg.ContractAddress.String(),
		QueryData: wasmtypes.RawContractMessage(payload),
	}
	resp, err := c.wasmClient.SmartContractState(ctx, query)
	if err != nil {
		return errors.Wrap(err, "query failed")
	}

	if err := json.Unmarshal(resp.Data, response); err != nil {
		return errors.Wrapf(err, "can't unmarshal wasm contract response, raw response:%s", string(resp.Data))
	}

	return nil
}
