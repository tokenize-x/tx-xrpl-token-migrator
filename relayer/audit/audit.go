package audit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tmtypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/xrpl"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/finder"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

// DiscrepancyType is the bridge audit discrepancy type.
type DiscrepancyType string

// DiscrepancyTypes.
const (
	DiscrepancyTypeContractDoubleSpend        = "ContractDoubleSpend"
	DiscrepancyTypeContractSentAmountMismatch = "ContractSentAmountMismatch"
	DiscrepancyTypeOrphanTXTx                 = "OrphanCoreumTx"
	DiscrepancyTypeInvalidRecipient           = "InvalidRecipient"
	DiscrepancyTypeNotBurningXRPLTransaction  = "NotBurningTransfer"
	DiscrepancyTypeAmountMismatch             = "AmountMismatch"
)

// KnownDiscrepancies are known discrepancies.
var KnownDiscrepancies = map[string]DiscrepancyType{
	// Those txs are expected to be difference since we had and issue with the conversion from the
	// XRPL to TX amount.
	// testnet
	"D0B28A44955C37F0E06D2CA63177461B18522639F1EF4E5AC171D2C45F7EA1FB": DiscrepancyTypeAmountMismatch,
	"F3E46D6FB811FAA57B57BA8DB4D345F9620BBAA40CDE2036DADBB24B6DBE66F3": DiscrepancyTypeAmountMismatch,
	"E3CB1679BA5F361F786B0FD2158DF6769C543E4DB3BC98CA4AE2C49F4D4A4BAC": DiscrepancyTypeAmountMismatch,
	"8CC3097C7D10E539488B981C6E314EE5717D5E0BB3764B099A072F4CE8E394C1": DiscrepancyTypeAmountMismatch,
	// mainnet
	"21B22F298BF359D43B3CBFB4CC0CEB93CEFC2C43CE889880262F5B2A8A5AEE1A": DiscrepancyTypeAmountMismatch,
	"D28775A1B3F4D17CD4F6CA5639E200B95F770E394558BACFBD91C0668A1BC384": DiscrepancyTypeAmountMismatch,
	"03A8ED4177CCFEA0D72C62FB2AC432C3CB9B29F06F8E0E377BF19270BED1A7E1": DiscrepancyTypeAmountMismatch,
	"269DB1DD2265F0115F92B1FB3865BA803BB29E8906D67A06ADAA66D550B5A107": DiscrepancyTypeAmountMismatch,
	"7FB045596AAC1CCB63D70B7D2688807A79910705F1CA8CE2A525756DBCB56BF8": DiscrepancyTypeAmountMismatch,
	"49E41F712B938FDB35A88555E9E6B172BA3CC5CC87AD9F663A1D9AAC5FF2C459": DiscrepancyTypeAmountMismatch,
	"01984404B349763D7D5A7716D9A85A4F26C359E03A6598C0BE7298CBFAAA4D06": DiscrepancyTypeAmountMismatch,
	"CA6D531A777E7F863A5B4B9A12E30F093043B6D83689FA361D741296E940440F": DiscrepancyTypeAmountMismatch,
	"E4D09529FC126F34A1780E172F1EB4F9FA4BC236EF45B008CF3970E46D820AF8": DiscrepancyTypeAmountMismatch,
	"78AA3733D4E1B906D09CD20CCC09D93979F95EFC8D4289C815A1CE2E06C18AE5": DiscrepancyTypeAmountMismatch,
	"FC17D74D459D201ABD8BB5C2F36335D7A04F6ACFE5C403A13DAAB22D81760DC4": DiscrepancyTypeAmountMismatch,
}

// Discrepancy is the bridge audit discrepancy.
type Discrepancy struct {
	TXTx        *sdk.TxResponse
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
	ThresholdBankSendRequests []ContractCallTx[tx.ThresholdBankSendRequest]
	ExecutePendingRequests    []ContractCallTx[tx.ExecutePendingRequest]
	UpdateMinAmountRequests   []ContractCallTx[tx.UpdateMinAmountRequest]
	UpdateMaxAmountRequests   []ContractCallTx[tx.UpdateMaxAmountRequest]
}

// NewContractCallReport creates initialised ContractCallReport.
func NewContractCallReport() *ContractCallReport {
	return &ContractCallReport{
		Txs:                       make([]*sdk.TxResponse, 0),
		ThresholdBankSendRequests: make([]ContractCallTx[tx.ThresholdBankSendRequest], 0),
		ExecutePendingRequests:    make([]ContractCallTx[tx.ExecutePendingRequest], 0),
		UpdateMinAmountRequests:   make([]ContractCallTx[tx.UpdateMinAmountRequest], 0),
		UpdateMaxAmountRequests:   make([]ContractCallTx[tx.UpdateMaxAmountRequest], 0),
	}
}

// TXChainClient is TX chain client.
type TXChainClient interface {
	GetSpendingTransactions(ctx context.Context, fromAddress string, startDate time.Time) ([]*sdk.TxResponse, error)
}

// XRPLRPCClient is XRPL client.
type XRPLRPCClient interface {
	GetTransactions(ctx context.Context, hashes []string) (map[string]xrpl.Transaction, error)
}

// XRPLTokenConfig is XRPL token config.
type XRPLTokenConfig struct {
	XRPLIssuer   rippledata.Account
	XRPLCurrency rippledata.Currency
	Multiplier   string
}

// AuditorConfig is Auditor config.
type AuditorConfig struct {
	ContractAddress string
	TXDenom         string
	TXDecimals      int
	XRPLMemoSuffix  string
	XRPLTokens      []XRPLTokenConfig
	StartDate       time.Time
}

// Auditor is the bridge auditor.
type Auditor struct {
	cfg           AuditorConfig
	log           logger.Logger
	txChainClient TXChainClient
	xrplRPCClient XRPLRPCClient
}

// NewAuditor returns a new instance of the Auditor.
func NewAuditor(
	cfg AuditorConfig,
	log logger.Logger,
	txChainClient TXChainClient,
	xrplRPCClient XRPLRPCClient,
) *Auditor {
	return &Auditor{
		cfg:           cfg,
		log:           log,
		txChainClient: txChainClient,
		xrplRPCClient: xrplRPCClient,
	}
}

// Audit analyses the bridge transactions and returns discrepancy results.
func (a *Auditor) Audit(ctx context.Context) ([]Discrepancy, error) {
	contractCallReport, err := a.buildContractCallReport(ctx)
	if err != nil {
		return nil, err
	}
	discrepancies, err := a.analyzeContractCallDiscrepancies(contractCallReport)
	if err != nil {
		return nil, err
	}
	xrplTxHashes := lo.Map(contractCallReport.ThresholdBankSendRequests,
		func(req ContractCallTx[tx.ThresholdBankSendRequest], _ int) string {
			return req.Payload.ID
		})
	a.log.Info("Fetching xrpl transactions.", zap.Int("count", len(xrplTxHashes)))
	xrplTxs, err := a.xrplRPCClient.GetTransactions(ctx, xrplTxHashes)
	if err != nil {
		return nil, err
	}
	a.log.Info("Fetched xrpl transactions.", zap.Int("count", len(xrplTxHashes)))
	discrepancies = append(discrepancies, a.analiseXrplToTXDiscrepancies(contractCallReport, xrplTxs)...)

	discrepancies = lo.Filter(discrepancies, func(d Discrepancy, _ int) bool {
		return d.Type != KnownDiscrepancies[d.XRPLTx.Hash]
	})

	return discrepancies, nil
}

func (a *Auditor) buildContractCallReport(ctx context.Context) (ContractCallReport, error) {
	a.log.Info("Fetching contact transactions to analise.")
	txs, err := a.txChainClient.GetSpendingTransactions(ctx, a.cfg.ContractAddress, a.cfg.StartDate)
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

func (a *Auditor) analyzeContractCallDiscrepancies(contractCallReport ContractCallReport) ([]Discrepancy, error) {
	discrepancies := make([]Discrepancy, 0)

	totalBridged := sdk.ZeroInt()
	foundTxHashes := make(map[string]struct{})
	for _, thresholdBankSendRequest := range contractCallReport.ThresholdBankSendRequests {
		xrplTxHash := thresholdBankSendRequest.Payload.ID
		if _, ok := foundTxHashes[xrplTxHash]; ok {
			discrepancies = append(discrepancies, Discrepancy{
				TXTx:        thresholdBankSendRequest.Tx,
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
		if sentCoins.Empty() {
			// workaround to use v3 client with v2 chain
			events, err := convertFromBase64ToStringAttributes(thresholdBankSendRequest.Tx.Events)
			if err != nil {
				return nil, err
			}
			sentCoins, err = sumCoinsEventsSiblingAttributeValues(
				events,
				banktypes.EventTypeTransfer,
				banktypes.AttributeKeySender,
				a.cfg.ContractAddress,
				sdk.AttributeKeyAmount,
			)
			if err != nil {
				return nil, err
			}
		}
		if sentCoins.Empty() {
			return nil, errors.Errorf(
				"the tx does not contain the sent coins, tx hash:%s, events:%v",
				thresholdBankSendRequest.Tx.TxHash, thresholdBankSendRequest.Tx.Events,
			)
		}

		if len(sentCoins) != 1 &&
			(sentCoins.AmountOf(a.cfg.TXDenom).String() != thresholdBankSendRequest.Payload.Amount.String()) {
			discrepancies = append(discrepancies, Discrepancy{
				TXTx: thresholdBankSendRequest.Tx,
				Type: DiscrepancyTypeContractSentAmountMismatch,
				Description: fmt.Sprintf(
					"The amount in the tx is different from sent, txAmount:%s, sentAmount:%s",
					thresholdBankSendRequest.Payload.Amount.String(), sentCoins.String(),
				),
			})
		}
	}
	a.log.Info("Total bridged", zap.String("amount", fmt.Sprintf("%s%s", totalBridged.String(), a.cfg.TXDenom)))

	return discrepancies, nil
}

func (a *Auditor) analiseXrplToTXDiscrepancies(
	contractCallReport ContractCallReport,
	xrplTransactions map[string]xrpl.Transaction,
) []Discrepancy {
	discrepancies := make([]Discrepancy, 0)
	for _, thresholdBankSendRequest := range contractCallReport.ThresholdBankSendRequests {
		xrplTxHash := thresholdBankSendRequest.Payload.ID
		xrplTx, found := xrplTransactions[xrplTxHash]
		if !found {
			discrepancies = append(discrepancies, Discrepancy{
				TXTx:        thresholdBankSendRequest.Tx,
				Type:        DiscrepancyTypeOrphanTXTx,
				Description: fmt.Sprintf("XRPL tx not found, hash:%s", xrplTxHash),
			})
			continue
		}

		xrplRecipient, found := finder.ExtractAddressFromMemo(xrplTx.Memos, a.cfg.XRPLMemoSuffix)
		if !found || (thresholdBankSendRequest.Payload.Recipient != xrplRecipient.String()) {
			discrepancies = append(discrepancies, Discrepancy{
				TXTx:   thresholdBankSendRequest.Tx,
				XRPLTx: xrplTx,
				Type:   DiscrepancyTypeInvalidRecipient,
				Description: fmt.Sprintf(
					"XRPL recipient different from TX, xrplRecipient:%s, txRecipient:%s",
					xrplRecipient.String(), thresholdBankSendRequest.Payload.Recipient,
				),
			})
		}

		// check if the recipient is issuer and tokes is allowed
		isBurningTx := false
		multiplier := "1.0"
		for _, tokenCfg := range a.cfg.XRPLTokens {
			if xrplTx.Destination == tokenCfg.XRPLIssuer.String() &&
				xrplTx.DeliveryAmount.Issuer.String() == tokenCfg.XRPLIssuer.String() &&
				xrplTx.DeliveryAmount.Currency.String() == tokenCfg.XRPLCurrency.String() {
				isBurningTx = true
				multiplier = tokenCfg.Multiplier
				break
			}
		}
		if !isBurningTx {
			discrepancies = append(discrepancies, Discrepancy{
				TXTx:        thresholdBankSendRequest.Tx,
				XRPLTx:      xrplTx,
				Type:        DiscrepancyTypeNotBurningXRPLTransaction,
				Description: "XRPL tx is not a burning tx",
			})
		}

		xrplAmount := finder.ConvertXRPLAmountToTXAmount(xrplTx.DeliveryAmount.Value, a.cfg.TXDecimals, multiplier)
		txAmount := thresholdBankSendRequest.Payload.Amount.Amount
		if xrplAmount.String() != txAmount.String() {
			discrepancies = append(discrepancies, Discrepancy{
				TXTx:   thresholdBankSendRequest.Tx,
				XRPLTx: xrplTx,
				Type:   DiscrepancyTypeAmountMismatch,
				Description: fmt.Sprintf(
					"XRPL tx amount is different from TX, xrplAmount:%s, txAmount:%s",
					xrplAmount.String(), txAmount.String(),
				),
			})
		}
	}

	return discrepancies
}

func decodeMessagePayloadToReport(txn *sdk.TxResponse, msg sdk.Msg, report *ContractCallReport) error {
	executeContractMsg, ok := msg.(*wasmtypes.MsgExecuteContract)
	if !ok {
		return errors.Errorf("unexpected message type for the message, msg:%v", msg)
	}
	payload := executeContractMsg.Msg
	callMap := make(map[string]json.RawMessage)
	if err := json.Unmarshal(payload, &callMap); err != nil {
		return errors.Wrapf(err, "failed to decode contract payload to map, raw payload:%s, tx:%v", string(payload), txn)
	}

	for methodName, methodPayload := range callMap {
		switch tx.ExecMethod(methodName) {
		case tx.ExecMethodThresholdBankSend:
			var req tx.ThresholdBankSendRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.ThresholdBankSendRequests = append(
				report.ThresholdBankSendRequests, ContractCallTx[tx.ThresholdBankSendRequest]{
					Tx:      txn,
					Msg:     msg,
					Payload: req,
				})
		case tx.ExecMethodExecutePending:
			var req tx.ExecutePendingRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.ExecutePendingRequests = append(report.ExecutePendingRequests, ContractCallTx[tx.ExecutePendingRequest]{
				Tx:      txn,
				Msg:     msg,
				Payload: req,
			})
		case tx.ExecMethodUpdateMinAmount:
			var req tx.UpdateMinAmountRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.UpdateMinAmountRequests = append(
				report.UpdateMinAmountRequests, ContractCallTx[tx.UpdateMinAmountRequest]{
					Tx:      txn,
					Msg:     msg,
					Payload: req,
				})
		case tx.ExecMethodUpdateMaxAmount:
			var req tx.UpdateMaxAmountRequest
			if err := json.Unmarshal(methodPayload, &req); err != nil {
				return err
			}
			report.UpdateMaxAmountRequests = append(
				report.UpdateMaxAmountRequests, ContractCallTx[tx.UpdateMaxAmountRequest]{
					Tx:      txn,
					Msg:     msg,
					Payload: req,
				})
		default:
			return errors.Errorf("exec method not found, method:%s", methodName)
		}
	}

	return nil
}

func sumCoinsEventsSiblingAttributeValues(
	events []tmtypes.Event,
	etype, siblingKey, siblingValue, nextAttrKey string,
) (sdk.Coins, error) {
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

func convertFromBase64ToStringAttributes(events []tmtypes.Event) ([]tmtypes.Event, error) {
	result := make([]tmtypes.Event, 0, len(events))
	for _, evt := range events {
		attrs := make([]tmtypes.EventAttribute, 0, len(evt.Attributes))
		for _, attr := range evt.Attributes {
			key, err := base64.StdEncoding.DecodeString(attr.Key)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to decode base64 key string, value:%s", attr.Key)
			}
			value, err := base64.StdEncoding.DecodeString(attr.Value)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to decode base64 value string, value:%s", attr.Value)
			}
			attrs = append(attrs, tmtypes.EventAttribute{
				Key:   string(key),
				Value: string(value),
				Index: attr.Index,
			})
		}
		result = append(result, tmtypes.Event{
			Type:       evt.Type,
			Attributes: attrs,
		})
	}

	return result, nil
}

func findEventsSiblingAttributeValues(
	events []tmtypes.Event,
	etype, siblingKey, siblingValue, nextAttrKey string,
) []string {
	values := make([]string, 0)
	for _, ev := range sdk.StringifyEvents(events) {
		if ev.Type == etype {
			values = append(values, findEventSiblingAttributeValues(ev, siblingKey, siblingValue, nextAttrKey)...)
		}
	}

	return values
}

// findEventSiblingAttributeValues find and returns all attribute values if sibling key with values is found in the
// same event.
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
