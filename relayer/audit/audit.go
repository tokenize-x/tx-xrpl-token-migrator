package audit

import (
	"context"
	"encoding/json"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/xrpl"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/finder"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// DiscrepancyType is the bridge audit discrepancy type.
type DiscrepancyType string

// DiscrepancyTypes.
const (
	DiscrepancyTypeContractDoubleSpend        = "ContractDoubleSpend"
	DiscrepancyTypeContractSentAmountMismatch = "ContractSentAmountMismatch"
	DiscrepancyTypeOrphanCoreumTx             = "OrphanCoreumTx"
	DiscrepancyTypeInvalidRecipient           = "InvalidRecipient"
	DiscrepancyTypeNotBurningXRPLTransaction  = "NotBurningTransfer"
	DiscrepancyTypeAmountMismatch             = "AmountMismatch"
)

// Discrepancy is the bridge audit discrepancy.
type Discrepancy struct {
	CoreumTx    *sdk.TxResponse
	XRPLTx      xrpl.Transaction
	Type        DiscrepancyType
	Description string
}

// ContractCallTx is type which holds contract call payload alongside with the transaction info.
type ContractCallTx[T any] struct {
	Tx      *sdk.TxResponse
	Msg     sdk.Msg
	Payload T
}

// ContractCallReport is the contract calls report which aggregates all calls to the contract.
type ContractCallReport struct {
	Txs                       []*sdk.TxResponse
	ThresholdBankSendRequests []ContractCallTx[coreum.ThresholdBankSendRequest]
	ExecutePendingRequests    []ContractCallTx[coreum.ExecutePendingRequest]
	UpdateMinAmountRequests   []ContractCallTx[coreum.UpdateMinAmountRequest]
	UpdateMaxAmountRequests   []ContractCallTx[coreum.UpdateMaxAmountRequest]
	WithdrawRequests          []ContractCallTx[coreum.WithdrawRequest]
}

// NewContractCallReport creates initialised ContractCallReport.
func NewContractCallReport() *ContractCallReport {
	return &ContractCallReport{
		Txs:                       make([]*sdk.TxResponse, 0),
		ThresholdBankSendRequests: make([]ContractCallTx[coreum.ThresholdBankSendRequest], 0),
		ExecutePendingRequests:    make([]ContractCallTx[coreum.ExecutePendingRequest], 0),
		UpdateMinAmountRequests:   make([]ContractCallTx[coreum.UpdateMinAmountRequest], 0),
		UpdateMaxAmountRequests:   make([]ContractCallTx[coreum.UpdateMaxAmountRequest], 0),
		WithdrawRequests:          make([]ContractCallTx[coreum.WithdrawRequest], 0),
	}
}

// CoreumChainClient is coreum chain client.
type CoreumChainClient interface {
	GetSpendingTransactions(ctx context.Context, fromAddress string) ([]*sdk.TxResponse, error)
}

// XRPLRPCClient is XRPL client.
type XRPLRPCClient interface {
	GetTransactions(ctx context.Context, hashes []string) (map[string]xrpl.Transaction, error)
}

// AuditorConfig is Auditor config.
type AuditorConfig struct {
	ContractAddress string
	CoreumDenom     string
	CoreumDecimals  int
	XRPLMemoSuffix  string
	XRPLCurrency    string
	XRPLIssuer      string
}

// Auditor is the bridge auditor.
type Auditor struct {
	cfg               AuditorConfig
	log               logger.Logger
	coreumChainClient CoreumChainClient
	xrplRPCClient     XRPLRPCClient
}

// NewAuditor returns a new instance of the Auditor.
func NewAuditor(cfg AuditorConfig, log logger.Logger, coreumChainClient CoreumChainClient, xrplRPCClient XRPLRPCClient) *Auditor {
	return &Auditor{
		cfg:               cfg,
		log:               log,
		coreumChainClient: coreumChainClient,
		xrplRPCClient:     xrplRPCClient,
	}
}

// Audit analyses the bridge transactions and returns discrepancy results.
func (a *Auditor) Audit(ctx context.Context) ([]Discrepancy, error) {
	contractCallReport, err := a.buildContractCallReport(ctx)
	if err != nil {
		return nil, err
	}
	discrepancies, err := a.analizeContractCallDiscrepancies(contractCallReport)
	if err != nil {
		return nil, err
	}
	xrplTxHashes := lo.Map(contractCallReport.ThresholdBankSendRequests, func(req ContractCallTx[coreum.ThresholdBankSendRequest], _ int) string {
		return req.Payload.ID
	})
	a.log.Info("Fetching xrpl transactions.", zap.Int("count", len(xrplTxHashes)))
	xrplTxs, err := a.xrplRPCClient.GetTransactions(ctx, xrplTxHashes)
	if err != nil {
		return nil, err
	}
	a.log.Info("Fetched xrpl transactions.", zap.Int("count", len(xrplTxHashes)))
	discrepancies = append(discrepancies, a.analiseXrplToCoreumDiscrepancies(contractCallReport, xrplTxs)...)

	return discrepancies, nil
}

func (a *Auditor) buildContractCallReport(ctx context.Context) (ContractCallReport, error) {
	a.log.Info("Fetching contact transactions to analise.")
	txs, err := a.coreumChainClient.GetSpendingTransactions(ctx, a.cfg.ContractAddress)
	if err != nil {
		return ContractCallReport{}, err
	}
	a.log.Info("Fetched contact transactions to analise.", zap.Int("count", len(txs)))

	report := NewContractCallReport()
	for _, txAny := range txs {
		tx, ok := txAny.Tx.GetCachedValue().(*sdktx.Tx)
		if !ok {
			return ContractCallReport{}, errors.Errorf("failed to get cached tx value, tx:%v", tx)
		}
		for _, msg := range tx.GetMsgs() {
			if err := decodeMessagePayloadToReport(txAny, msg, report); err != nil {
				return ContractCallReport{}, errors.Wrapf(err, "failed to decode transaction, tx:%v", tx)
			}
		}
		report.Txs = append(report.Txs, txAny)
	}

	return *report, nil
}

func (a *Auditor) analizeContractCallDiscrepancies(contractCallReport ContractCallReport) ([]Discrepancy, error) {
	discrepancies := make([]Discrepancy, 0)

	totalBridged := sdk.ZeroInt()
	foundTxHashes := make(map[string]struct{})
	for _, thresholdBankSendRequest := range contractCallReport.ThresholdBankSendRequests {
		xrplTxHash := thresholdBankSendRequest.Payload.ID
		if _, ok := foundTxHashes[xrplTxHash]; ok {
			discrepancies = append(discrepancies, Discrepancy{
				CoreumTx:    thresholdBankSendRequest.Tx,
				Type:        DiscrepancyTypeContractDoubleSpend,
				Description: fmt.Sprintf("Found duplicated XRPL tx hash in processed transactions, hash:%s", xrplTxHash),
			})
		}
		foundTxHashes[xrplTxHash] = struct{}{}

		totalBridged = totalBridged.Add(thresholdBankSendRequest.Payload.Amount.Amount)
		sentCoins, err := sumCoinsEventsSiblingAttributeValues(
			thresholdBankSendRequest.Tx.Events,
			banktypes.EventTypeTransfer,
			banktypes.AttributeKeySender,
			a.cfg.ContractAddress,
			sdk.AttributeKeyAmount,
		)
		if err != nil {
			return nil, err
		}
		if len(sentCoins) != 1 && (sentCoins.AmountOf(a.cfg.CoreumDenom).String() != thresholdBankSendRequest.Payload.Amount.String()) {
			discrepancies = append(discrepancies, Discrepancy{
				CoreumTx:    thresholdBankSendRequest.Tx,
				Type:        DiscrepancyTypeContractSentAmountMismatch,
				Description: fmt.Sprintf("The amount in the tx is different from sent, txAmount:%s, sentAmount:%s", thresholdBankSendRequest.Payload.Amount.String(), sentCoins.String()),
			})
		}
	}
	a.log.Info("Total bridged", zap.String("amount", fmt.Sprintf("%s%s", totalBridged.String(), a.cfg.CoreumDenom)))

	return discrepancies, nil
}

func (a *Auditor) analiseXrplToCoreumDiscrepancies(contractCallReport ContractCallReport, xrplTransactions map[string]xrpl.Transaction) []Discrepancy {
	discrepancies := make([]Discrepancy, 0)
	for _, thresholdBankSendRequest := range contractCallReport.ThresholdBankSendRequests {
		xrplTxHash := thresholdBankSendRequest.Payload.ID
		xrplTx, found := xrplTransactions[xrplTxHash]
		if !found {
			discrepancies = append(discrepancies, Discrepancy{
				CoreumTx:    thresholdBankSendRequest.Tx,
				Type:        DiscrepancyTypeOrphanCoreumTx,
				Description: fmt.Sprintf("XRPL tx not found, hash:%s", xrplTxHash),
			})
			continue
		}

		xrplRecipient, found := finder.ExtractAddressFromMemo(xrplTx.Memos, a.cfg.XRPLMemoSuffix)
		if !found || (thresholdBankSendRequest.Payload.Recipient != xrplRecipient.String()) {
			discrepancies = append(discrepancies, Discrepancy{
				CoreumTx:    thresholdBankSendRequest.Tx,
				XRPLTx:      xrplTx,
				Type:        DiscrepancyTypeInvalidRecipient,
				Description: fmt.Sprintf("XRPL recipient different from corem, xrplRecipient:%s, coreumRecipient:%s", xrplRecipient.String(), thresholdBankSendRequest.Payload.Recipient),
			})
		}

		if xrplTx.Destination != a.cfg.XRPLIssuer ||
			xrplTx.DeliveryAmount.Issuer != a.cfg.XRPLIssuer ||
			xrplTx.DeliveryAmount.Currency != a.cfg.XRPLCurrency {
			discrepancies = append(discrepancies, Discrepancy{
				CoreumTx:    thresholdBankSendRequest.Tx,
				XRPLTx:      xrplTx,
				Type:        DiscrepancyTypeNotBurningXRPLTransaction,
				Description: "XRPL tx is not a burning tx",
			})
		}

		xrplAmount := finder.ConvertXRPLAmountToCoreumAmount(xrplTx.DeliveryAmount.Value, a.cfg.CoreumDecimals)
		coreumAmount := thresholdBankSendRequest.Payload.Amount.Amount
		if xrplAmount.String() != coreumAmount.String() {
			discrepancies = append(discrepancies, Discrepancy{
				CoreumTx:    thresholdBankSendRequest.Tx,
				XRPLTx:      xrplTx,
				Type:        DiscrepancyTypeAmountMismatch,
				Description: fmt.Sprintf("XRPL tx amount if different from coreum, xrplAmount:%s, coreumAmount:%s", xrplAmount.String(), coreumAmount.String()),
			})
		}
	}

	return discrepancies
}

func decodeMessagePayloadToReport(tx *sdk.TxResponse, msg sdk.Msg, report *ContractCallReport) error {
	executeContractMsg, ok := msg.(*wasmtypes.MsgExecuteContract)
	if !ok {
		return errors.Errorf("unexpected message type for the message, msg:%v", msg)
	}
	payload := executeContractMsg.Msg
	callMap := make(map[string]json.RawMessage)
	if err := json.Unmarshal(payload, &callMap); err != nil {
		return errors.Wrapf(err, "failed to decode contract payload to map, raw payload:%s, tx:%v", string(payload), tx)
	}

	for methodName, methodPayload := range callMap {
		switch coreum.ExecMethod(methodName) {
		case coreum.ExecMethodThresholdBankSend:
			var req coreum.ThresholdBankSendRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.ThresholdBankSendRequests = append(report.ThresholdBankSendRequests, ContractCallTx[coreum.ThresholdBankSendRequest]{
				Tx:      tx,
				Msg:     msg,
				Payload: req,
			})
		case coreum.ExecMethodExecutePending:
			var req coreum.ExecutePendingRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.ExecutePendingRequests = append(report.ExecutePendingRequests, ContractCallTx[coreum.ExecutePendingRequest]{
				Tx:      tx,
				Msg:     msg,
				Payload: req,
			})
		case coreum.ExecMethodUpdateMinAmount:
			var req coreum.UpdateMinAmountRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.UpdateMinAmountRequests = append(report.UpdateMinAmountRequests, ContractCallTx[coreum.UpdateMinAmountRequest]{
				Tx:      tx,
				Msg:     msg,
				Payload: req,
			})
		case coreum.ExecMethodUpdateMaxAmount:
			var req coreum.UpdateMaxAmountRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.UpdateMaxAmountRequests = append(report.UpdateMaxAmountRequests, ContractCallTx[coreum.UpdateMaxAmountRequest]{
				Tx:      tx,
				Msg:     msg,
				Payload: req,
			})
		case coreum.ExecMethodWithdraw:
			var req coreum.WithdrawRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.WithdrawRequests = append(report.WithdrawRequests, ContractCallTx[coreum.WithdrawRequest]{
				Tx:      tx,
				Msg:     msg,
				Payload: req,
			})
		default:
			return errors.Errorf("exec method not found, method:%s", methodName)
		}
	}

	return nil
}

func sumCoinsEventsSiblingAttributeValues(events []tmtypes.Event, etype, siblingKey, siblingValue, nextAttrKey string) (sdk.Coins, error) {
	values := findEventsSiblingAttributeValues(events, etype, siblingKey, siblingValue, nextAttrKey)
	coins := sdk.NewCoins()
	for _, v := range values {
		coin, err := sdk.ParseCoinNormalized(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse string to count, value:%s", v)
		}
		coins = coins.Add(coin)
	}

	return coins, nil
}

func findEventsSiblingAttributeValues(events []tmtypes.Event, etype, siblingKey, siblingValue, nextAttrKey string) []string {
	values := make([]string, 0)
	for _, ev := range sdk.StringifyEvents(events) {
		if ev.Type == etype {
			values = append(values, findEventSiblingAttributeValues(ev, siblingKey, siblingValue, nextAttrKey)...)
		}
	}

	return values
}

// findEventSiblingAttributeValues find and returns all attribute values if sibling key with values is found in the same event.
func findEventSiblingAttributeValues(ev sdk.StringEvent, siblingKey, siblingValue, nextAttrKey string) []string {
	attrValues := make([]string, 0)
	for i, attrItem := range ev.Attributes {
		if attrItem.Key == siblingKey && attrItem.Value == siblingValue {
			if i < len(ev.Attributes) {
				nextAttrItem := ev.Attributes[i+1]
				if nextAttrItem.Key == nextAttrKey {
					attrValues = append(attrValues, nextAttrItem.Value)
				}
			}
		}
	}

	return attrValues
}
