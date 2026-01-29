package tx

import (
	"context"
	"encoding/json"
	"strings"

	sdkmath "cosmossdk.io/math"
	"github.com/CoreumFoundation/coreum/v5/pkg/client"
	"github.com/CoreumFoundation/coreum/v5/testutil/event"
	feemodeltypes "github.com/CoreumFoundation/coreum/v5/x/feemodel/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	contractembed "github.com/tokenize-x/tx-xrpl-token-migrator/contract"
)

// ExecMethod is contract exec method.
type ExecMethod string

// ExecMethods.
const (
	ExecMethodThresholdBankSend      ExecMethod = "threshold_bank_send"
	ExecMethodExecutePending         ExecMethod = "execute_pending"
	ExecMethodUpdateMinAmount        ExecMethod = "update_min_amount"
	ExecMethodUpdateMaxAmount        ExecMethod = "update_max_amount"
	ExecMethodUpdateTrustedAddresses ExecMethod = "update_trusted_addresses"
	ExecMethodAddXRPLTokens          ExecMethod = "add_xrpl_tokens"
)

// QueryMethod is contract query method.
type QueryMethod string

// QueryMethods.
const (
	QueryMethodGetConfig              QueryMethod = "get_config"
	QueryMethodGetPendingTransaction  QueryMethod = "get_pending_transaction"
	QueryMethodGetPendingTransactions QueryMethod = "get_pending_transactions"
	QueryMethodGetSentTransaction     QueryMethod = "get_sent_transaction"
	QueryMethodGetSentTransactions    QueryMethod = "get_sent_transactions"
)

const (
	unauthorizedErrorString            = "Unauthorized"
	evidenceProvidedErrorString        = "Sender already provided the evidence"
	transferSentErrorString            = "Transfer already sent"
	transactionNotFoundErrorString     = "Transaction not found"
	transactionNotConfirmedErrorString = "Transaction not confirmed"
	lowAmountErrorString               = "The amount is too low"
	fundsMismatchErrorString           = "Funds mismatch"
)

// DeployAndInstantiateConfig holds attributes used for the contract deployment and instantiation.
type DeployAndInstantiateConfig struct {
	Owner            string
	Admin            string
	TrustedAddresses []string
	Threshold        uint32
	MinAmount        sdkmath.Int
	MaxAmount        sdkmath.Int
	XRPLTokens       []XRPLToken
	Label            string
}

// XRPLToken represents XRPL token configuration.
//
//nolint:tagliatelle //contract spec
type XRPLToken struct {
	Currency       string `json:"currency"`
	Issuer         string `json:"issuer"`
	ActivationDate uint64 `json:"activation_date"`
	Multiplier     string `json:"multiplier"`
}

// BSCToken represents BSC token configuration.
//
//nolint:tagliatelle //contract spec
type BSCToken struct {
	BridgeAddress  string `json:"bridge_address"`
	ActivationDate uint64 `json:"activation_date"`
	Decimals       uint32 `json:"decimals"`
}

// Config represents contract config.
//
//nolint:tagliatelle //contract spec
type Config struct {
	Owner            string      `json:"owner"`
	TrustedAddresses []string    `json:"trusted_addresses"`
	Threshold        uint32      `json:"threshold"`
	MinAmount        sdkmath.Int `json:"min_amount"`
	MaxAmount        sdkmath.Int `json:"max_amount"`
	XRPLTokens       []XRPLToken `json:"xrpl_tokens"`
	BSCTokens        []BSCToken  `json:"bsc_tokens"`
	Version          uint64      `json:"version"`
}

// ThresholdBankSendRequest holds attributes for the send transaction.
type ThresholdBankSendRequest struct {
	ID        string   `json:"id"`
	Amount    sdk.Coin `json:"amount"`
	Recipient string   `json:"recipient"`
}

// ExecutePendingRequest is the `execute_pending` request payload.
type ExecutePendingRequest struct {
	EvidenceID string `json:"evidence_id"` //nolint:tagliatelle //contract spec
}

// UpdateMinAmountRequest is the `update_min_amount_request` request payload.
type UpdateMinAmountRequest struct {
	MinAmount sdkmath.Int `json:"min_amount"` //nolint:tagliatelle //contract spec
}

// UpdateMaxAmountRequest is the `update_max_amount_request` request payload.
type UpdateMaxAmountRequest struct {
	MaxAmount sdkmath.Int `json:"max_amount"` //nolint:tagliatelle //contract spec
}

// UpdateUpdateTrustedAddressesRequest is the `update_trusted_addresses` request payload.
type UpdateUpdateTrustedAddressesRequest struct {
	TrustedAddresses []sdk.AccAddress `json:"trusted_addresses"` //nolint:tagliatelle //contract spec
}

// AddXRPLTokensRequest is the `add_xrpl_tokens` request payload.
type AddXRPLTokensRequest struct {
	XRPLTokens []XRPLToken `json:"xrpl_tokens"` //nolint:tagliatelle //contract spec
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

//nolint:tagliatelle //contract spec
type instantiateRequest struct {
	Owner            string      `json:"owner"`
	TrustedAddresses []string    `json:"trusted_addresses"`
	Threshold        uint32      `json:"threshold"`
	MinAmount        sdkmath.Int `json:"min_amount"`
	MaxAmount        sdkmath.Int `json:"max_amount"`
	XRPLTokens       []XRPLToken `json:"xrpl_tokens"`
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
	ContractAddress    sdk.AccAddress
	TXDenom            string
	GasMultiplier      float64
	GasPriceMultiplier sdkmath.LegacyDec
	ContractPageSize   uint32
}

// DefaultContractClientConfig returns default ContractClient config.
func DefaultContractClientConfig(contractAddress sdk.AccAddress, txDenom string) ContractClientConfig {
	return ContractClientConfig{
		ContractAddress:    contractAddress,
		TXDenom:            txDenom,
		GasMultiplier:      1.5,
		GasPriceMultiplier: sdkmath.LegacyMustNewDecFromStr("1.2"),
		ContractPageSize:   500,
	}
}

// ContractClient is the wasm contract client.
type ContractClient struct {
	cfg                 ContractClientConfig
	clientCtx           client.Context
	wasmClient          wasmtypes.QueryClient
	feemodelQueryClient feemodeltypes.QueryClient
}

// NewContractClient returns a new instance of the ContractClient.
func NewContractClient(cfg ContractClientConfig, clientCtx client.Context) *ContractClient {
	return &ContractClient{
		cfg: cfg,
		clientCtx: clientCtx.
			WithBroadcastMode(flags.BroadcastSync).
			WithAwaitTx(true).
			WithGasPriceAdjustment(cfg.GasPriceMultiplier),
		wasmClient:          wasmtypes.NewQueryClient(clientCtx),
		feemodelQueryClient: feemodeltypes.NewQueryClient(clientCtx),
	}
}

// DeployAndInstantiate deploys the contract bytecode and instantiate it.
func (c *ContractClient) DeployAndInstantiate(
	ctx context.Context,
	sender sdk.AccAddress,
	config DeployAndInstantiateConfig,
) (sdk.AccAddress, error) {
	codeID, err := c.Deploy(ctx, sender)
	if err != nil {
		return nil, err
	}

	reqPayload, err := json.Marshal(instantiateRequest{
		Owner:            config.Owner,
		TrustedAddresses: config.TrustedAddresses,
		Threshold:        config.Threshold,
		MinAmount:        config.MinAmount,
		MaxAmount:        config.MaxAmount,
		XRPLTokens:       config.XRPLTokens,
	})
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal instantiate payload")
	}

	msg := &wasmtypes.MsgInstantiateContract{
		Sender: sender.String(),
		Admin:  config.Admin,
		CodeID: codeID,
		Label:  config.Label,
		Msg:    reqPayload,
	}

	res, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msg)
	if err != nil {
		return nil, errors.Wrap(err, "can't instantiate bytecode")
	}

	contractAddr, err := event.FindStringEventAttribute(
		res.Events,
		wasmtypes.EventTypeInstantiate,
		wasmtypes.AttributeKeyContractAddr,
	)
	if err != nil {
		return nil, errors.Wrap(err, "can't find contract address in the tx result")
	}

	sdkContractAddr, err := sdk.AccAddressFromBech32(contractAddr)
	if err != nil {
		return nil, errors.Wrap(err, "can't convert contrac address to sdk.AccAddress")
	}

	return sdkContractAddr, nil
}

// Deploy deploys the contract bytecode.
func (c *ContractClient) Deploy(
	ctx context.Context,
	sender sdk.AccAddress,
) (uint64, error) {
	msgStoreCode := &wasmtypes.MsgStoreCode{
		Sender:       sender.String(),
		WASMByteCode: contractembed.Bytecode,
	}

	res, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msgStoreCode)
	if err != nil {
		return 0, errors.Wrap(err, "can't deploy wasm bytecode")
	}
	codeID, err := event.FindUint64EventAttribute(res.Events, wasmtypes.EventTypeStoreCode, wasmtypes.AttributeKeyCodeID)
	if err != nil {
		return 0, errors.Wrap(err, "can't find code ID in the tx result")
	}

	return codeID, nil
}

// MigrateContract calls the executes the contract migration.
func (c *ContractClient) MigrateContract(
	ctx context.Context,
	sender sdk.AccAddress,
	codeID uint64,
) (*sdk.TxResponse, error) {
	msg := c.BuildMigrateContractMessage(sender, codeID)
	txRes, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to migrate contract, codeID:%d", codeID)
	}

	return txRes, nil
}

// BuildMigrateContractMessage builds migrate contract message.
func (c *ContractClient) BuildMigrateContractMessage(
	sender sdk.AccAddress,
	codeID uint64,
) sdk.Msg {
	return &wasmtypes.MsgMigrateContract{
		Sender:   sender.String(),
		Contract: c.cfg.ContractAddress.String(),
		CodeID:   codeID,
		Msg:      []byte("{}"),
	}
}

// UpdateTrustedAddresses executes update_trusted_addresses Method of the contract.
func (c *ContractClient) UpdateTrustedAddresses(
	ctx context.Context,
	sender sdk.AccAddress,
	newTrustedAddresses []sdk.AccAddress,
) (*sdk.TxResponse, error) {
	msg, err := c.BuildUpdateTrustedAddressesTransaction(sender, newTrustedAddresses)
	if err != nil {
		return nil, err
	}

	txRes, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update trusted addresses")
	}

	return txRes, nil
}

// BuildUpdateTrustedAddressesTransaction build update_trusted_addresses Method transaction.
func (c *ContractClient) BuildUpdateTrustedAddressesTransaction(
	sender sdk.AccAddress,
	newTrustedAddresses []sdk.AccAddress,
) (sdk.Msg, error) {
	msg, err := c.buildExecuteWithFunds(sender, sdk.NewCoins(), map[ExecMethod]UpdateUpdateTrustedAddressesRequest{
		ExecMethodUpdateTrustedAddresses: {
			TrustedAddresses: newTrustedAddresses,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build tx for %s Method", ExecMethodUpdateTrustedAddresses)
	}

	return msg, nil
}

// AddXRPLTokens executes add_xrpl_tokens Method of the contract.
func (c *ContractClient) AddXRPLTokens(
	ctx context.Context,
	sender sdk.AccAddress,
	xrplTokens []XRPLToken,
) (*sdk.TxResponse, error) {
	msg, err := c.BuildAddXRPLTokensTransaction(sender, xrplTokens)
	if err != nil {
		return nil, err
	}

	txRes, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add XRPL tokens")
	}

	return txRes, nil
}

// BuildAddXRPLTokensTransaction build add_xrpl_tokens Method transaction.
func (c *ContractClient) BuildAddXRPLTokensTransaction(
	sender sdk.AccAddress,
	xrplTokens []XRPLToken,
) (sdk.Msg, error) {
	msg, err := c.buildExecuteWithFunds(sender, sdk.NewCoins(), map[ExecMethod]AddXRPLTokensRequest{
		ExecMethodAddXRPLTokens: {
			XRPLTokens: xrplTokens,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build tx for %s Method", ExecMethodAddXRPLTokens)
	}

	return msg, nil
}

// SetContractAddress sets the client contract address if it was not set before.
func (c *ContractClient) SetContractAddress(contractAddress sdk.AccAddress) error {
	if c.cfg.ContractAddress != nil {
		return errors.New("contract address is already set")
	}

	c.cfg.ContractAddress = contractAddress

	return nil
}

// ThresholdBankSend executes threshold_bank_send Method of the contract.
func (c *ContractClient) ThresholdBankSend(
	ctx context.Context,
	sender sdk.AccAddress,
	requests ...ThresholdBankSendRequest,
) (*sdk.TxResponse, error) {
	msgs := make([]interface{}, 0)
	for _, req := range requests {
		msgs = append(msgs, map[ExecMethod]ThresholdBankSendRequest{
			ExecMethodThresholdBankSend: {
				ID:        req.ID,
				Amount:    req.Amount,
				Recipient: req.Recipient,
			},
		})
	}
	txRes, err := c.execute(ctx, sender, msgs...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s Method", ExecMethodThresholdBankSend)
	}

	return txRes, nil
}

// UpdateMinAmount executes update_min_amount Method of the contract.
func (c *ContractClient) UpdateMinAmount(
	ctx context.Context,
	sender sdk.AccAddress,
	minAmount sdkmath.Int,
) (*sdk.TxResponse, error) {
	txRes, err := c.execute(ctx, sender, map[ExecMethod]UpdateMinAmountRequest{
		ExecMethodUpdateMinAmount: {
			MinAmount: minAmount,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s Method", ExecMethodUpdateMinAmount)
	}

	return txRes, nil
}

// UpdateMaxAmount executes update_max_amount Method of the contract.
func (c *ContractClient) UpdateMaxAmount(
	ctx context.Context,
	sender sdk.AccAddress,
	maxAmount sdkmath.Int,
) (*sdk.TxResponse, error) {
	txRes, err := c.execute(ctx, sender, map[ExecMethod]UpdateMaxAmountRequest{
		ExecMethodUpdateMaxAmount: {
			MaxAmount: maxAmount,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s Method", ExecMethodUpdateMaxAmount)
	}

	return txRes, nil
}

// ExecutePending executes execute_pending Method of the contract.
func (c *ContractClient) ExecutePending(
	ctx context.Context,
	sender sdk.AccAddress,
	funds sdk.Coin,
	evidenceID string,
) (*sdk.TxResponse, error) {
	txRes, err := c.executeWithFunds(ctx, sender, sdk.NewCoins(funds), map[ExecMethod]ExecutePendingRequest{
		ExecMethodExecutePending: {
			EvidenceID: evidenceID,
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute %s Method", ExecMethodExecutePending)
	}

	return txRes, nil
}

// BuildExecutePendingMessages builds execute_pending messages of the contract.
func (c *ContractClient) BuildExecutePendingMessages(
	ctx context.Context,
	sender sdk.AccAddress,
	evidenceIDs []string,
) ([]sdk.Msg, error) {
	_, approvedTransactions, err := c.GetAllPendingTransactions(ctx)
	if err != nil {
		return nil, err
	}

	evidenceIDsMap := lo.SliceToMap(evidenceIDs, func(item string) (string, struct{}) {
		return item, struct{}{}
	})

	msgs := make([]sdk.Msg, 0)
	includeAll := len(evidenceIDsMap) == 0
	for _, txn := range approvedTransactions {
		if _, ok := evidenceIDsMap[txn.EvidenceID]; includeAll || ok {
			msg, err := c.buildExecuteWithFunds(sender, sdk.NewCoins(txn.Amount), map[ExecMethod]ExecutePendingRequest{
				ExecMethodExecutePending: {
					EvidenceID: txn.EvidenceID,
				},
			})
			if err != nil {
				return nil, errors.Wrapf(err, "failed to execute %s Method", ExecMethodExecutePending)
			}
			msgs = append(msgs, msg)
			delete(evidenceIDsMap, txn.EvidenceID)
		}
	}

	if len(evidenceIDsMap) != 0 {
		return nil, errors.Errorf("some of evidence ids are not found in the approved pending transactions: %v",
			lo.MapToSlice(evidenceIDsMap, func(id string, _ struct{}) string {
				return id
			}))
	}

	return msgs, nil
}

// EstimateExecuteMessages estimates the cost for execute contract messages.
func (c *ContractClient) EstimateExecuteMessages(
	ctx context.Context,
	sender sdk.AccAddress,
	msgs ...sdk.Msg,
) (sdk.Coin, uint64, error) {
	gas, err := c.calculateGas(ctx, sender, c.txFactory(), msgs...)
	if err != nil {
		return sdk.Coin{}, 0, err
	}
	feemodelParamsRes, err := c.feemodelQueryClient.Params(ctx, &feemodeltypes.QueryParamsRequest{})
	if err != nil {
		return sdk.Coin{}, 0, err
	}
	gasInt := sdkmath.LegacyNewDecFromInt(sdkmath.NewIntFromUint64(gas))
	amount := feemodelParamsRes.Params.Model.InitialGasPrice.Mul(gasInt).TruncateInt()

	return sdk.NewCoin(c.cfg.TXDenom, amount), gas, nil
}

// GetContractConfig returns contract config.
func (c *ContractClient) GetContractConfig(ctx context.Context) (Config, error) {
	var config Config
	err := c.query(ctx, map[QueryMethod]struct{}{
		QueryMethodGetConfig: {},
	}, &config)
	if err != nil {
		return Config{}, errors.Wrapf(err, "failed to query %s", QueryMethodGetConfig)
	}

	return config, nil
}

// GetPendingTx returns a pending transaction.
func (c *ContractClient) GetPendingTx(ctx context.Context, evidenceID string) (Transaction, error) {
	var txn Transaction
	err := c.query(ctx, map[QueryMethod]pendingTxQueryRequest{
		QueryMethodGetPendingTransaction: {
			EvidenceID: evidenceID,
		},
	}, &txn)
	if err != nil {
		return Transaction{}, errors.Wrapf(err, "failed to query %s", QueryMethodGetPendingTransaction)
	}

	return txn, nil
}

// GetPendingTxs returns a list of pending transactions.
func (c *ContractClient) GetPendingTxs(
	ctx context.Context,
	offset *uint64,
	limit *uint32,
) ([]PendingTransaction, error) {
	var txs queryTxsResponse[PendingTransaction]
	err := c.query(ctx, map[QueryMethod]pagingRequest{
		QueryMethodGetPendingTransactions: {
			Offset: offset,
			Limit:  limit,
		},
	}, &txs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query %s", QueryMethodGetPendingTransactions)
	}

	return txs.Transactions, nil
}

// GetSentTx returns a sent transaction.
func (c *ContractClient) GetSentTx(ctx context.Context, id string) (Transaction, error) {
	var txn Transaction
	err := c.query(ctx, map[QueryMethod]sentTxQueryRequest{
		QueryMethodGetSentTransaction: {
			ID: id,
		},
	}, &txn)
	if err != nil {
		return Transaction{}, errors.Wrapf(err, "failed to query %s", QueryMethodGetSentTransaction)
	}

	return txn, nil
}

// GetSentTxs returns a list of sent transactions.
func (c *ContractClient) GetSentTxs(ctx context.Context, offset *uint64, limit *uint32) ([]SentTransaction, error) {
	var txs queryTxsResponse[SentTransaction]
	err := c.query(ctx, map[QueryMethod]pagingRequest{
		QueryMethodGetSentTransactions: {
			Offset: offset,
			Limit:  limit,
		},
	}, &txs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query %s", QueryMethodGetSentTransactions)
	}

	return txs.Transactions, nil
}

// GetAllPendingTransactions queries all unapproved and approved pending transactions.
func (c *ContractClient) GetAllPendingTransactions(ctx context.Context) (
	[]PendingTransaction, []PendingTransaction, error,
) {
	offset := uint64(0)
	limit := c.cfg.ContractPageSize

	unapprovedTransactions := make([]PendingTransaction, 0)
	approvedTransactions := make([]PendingTransaction, 0)

	contractCfg, err := c.GetContractConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	for {
		pendingTxs, err := c.GetPendingTxs(ctx, &offset, &limit)
		if err != nil {
			return nil, nil, err
		}
		if len(pendingTxs) == 0 {
			break
		}

		for _, pendingTx := range pendingTxs {
			if uint32(len(pendingTx.EvidenceProviders)) < contractCfg.Threshold {
				unapprovedTransactions = append(unapprovedTransactions, pendingTx)
				continue
			}
			approvedTransactions = append(approvedTransactions, pendingTx)
		}

		offset += uint64(c.cfg.ContractPageSize)
		limit += c.cfg.ContractPageSize
	}

	return unapprovedTransactions, approvedTransactions, nil
}

// calculateGas calculates gas using legacy amino codec to cover both multisig and basic accounts.
func (c *ContractClient) calculateGas(
	ctx context.Context,
	sender sdk.AccAddress,
	txf client.Factory,
	msgs ...sdk.Msg,
) (uint64, error) {
	modeInfo, signature := authtx.SignatureDataToModeInfoAndSig(&signing.MultiSignatureData{
		Signatures: []signing.SignatureData{
			&signing.SingleSignatureData{
				SignMode: signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON,
			},
		},
	})

	pubKeyAny, err := codectypes.NewAnyWithValue(&secp256k1.PubKey{})
	if err != nil {
		return 0, errors.WithStack(err)
	}

	acc, err := client.GetAccountInfo(ctx, c.clientCtx, sender)
	if err != nil {
		return 0, errors.WithStack(err)
	}

	simAuthInfoBytes, err := proto.Marshal(&sdktx.AuthInfo{
		SignerInfos: []*sdktx.SignerInfo{
			{
				PublicKey: pubKeyAny,
				ModeInfo:  modeInfo,
				Sequence:  acc.GetSequence(),
			},
		},
		Fee: &sdktx.Fee{},
	})
	if err != nil {
		return 0, errors.WithStack(err)
	}

	anyMessages := make([]*codectypes.Any, 0, len(msgs))
	for _, msg := range msgs {
		anyMsg, err := codectypes.NewAnyWithValue(msg)
		if err != nil {
			return 0, errors.WithStack(err)
		}
		anyMessages = append(anyMessages, anyMsg)
	}

	bodyBytes, err := proto.Marshal(&sdktx.TxBody{
		Messages: anyMessages,
	})
	if err != nil {
		return 0, errors.WithStack(err)
	}

	txBytes, err := proto.Marshal(&sdktx.TxRaw{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: simAuthInfoBytes,
		Signatures: [][]byte{
			signature,
		},
	})
	if err != nil {
		return 0, errors.WithStack(err)
	}

	txSvcClient := sdktx.NewServiceClient(c.clientCtx)
	simRes, err := txSvcClient.Simulate(ctx, &sdktx.SimulateRequest{
		TxBytes: txBytes,
	})
	if err != nil {
		return 0, errors.Wrap(err, "transaction estimation failed")
	}

	return uint64(txf.GasAdjustment() * float64(simRes.GasInfo.GasUsed)), nil
}

// IsUnauthorizedError returns true if error is Unauthorized error.
func IsUnauthorizedError(err error) bool {
	return isError(err, unauthorizedErrorString)
}

// IsEvidenceProvidedError returns true if error is EvidenceProvided error.
func IsEvidenceProvidedError(err error) bool {
	return isError(err, evidenceProvidedErrorString)
}

// IsTransferSentError returns true if error is TransferSent error.
func IsTransferSentError(err error) bool {
	return isError(err, transferSentErrorString)
}

// IsTransactionNotFoundError returns true if error is TransactionNotFound error.
func IsTransactionNotFoundError(err error) bool {
	return isError(err, transactionNotFoundErrorString)
}

// IsTransactionNotConfirmedError returns true if error is TransactionNotConfirmed error.
func IsTransactionNotConfirmedError(err error) bool {
	return isError(err, transactionNotConfirmedErrorString)
}

// IsLowAmountError returns true if error is LowAmount error.
func IsLowAmountError(err error) bool {
	return isError(err, lowAmountErrorString)
}

// IsFundsMismatchError returns true if error is FundsMismatch error.
func IsFundsMismatchError(err error) bool {
	return isError(err, fundsMismatchErrorString)
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
			Msg:      payload,
		}
		msgs = append(msgs, msg)
	}

	res, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msgs...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ContractClient) executeWithFunds(
	ctx context.Context,
	sender sdk.AccAddress,
	funds sdk.Coins,
	req any,
) (*sdk.TxResponse, error) {
	msg, err := c.buildExecuteWithFunds(sender, funds, req)
	if err != nil {
		return nil, err
	}

	res, err := client.BroadcastTx(ctx, c.clientCtx.WithFromAddress(sender), c.txFactory(), msg)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c *ContractClient) buildExecuteWithFunds(
	sender sdk.AccAddress,
	funds sdk.Coins,
	req any,
) (*wasmtypes.MsgExecuteContract, error) {
	if c.cfg.ContractAddress == nil {
		return nil, errors.New("failed to execute with empty contract address")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal payload")
	}
	return &wasmtypes.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: c.cfg.ContractAddress.String(),
		Msg:      payload,
		Funds:    funds,
	}, nil
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
		QueryData: payload,
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
