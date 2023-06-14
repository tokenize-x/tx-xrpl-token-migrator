//go:build integrationtests

package integrationtests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	integrationtests "github.com/CoreumFoundation/coreum/integration-tests"
	"github.com/CoreumFoundation/coreum/pkg/client"
	"github.com/CoreumFoundation/coreum/testutil/event"
)

type InstantiateRequest struct {
	Owner            string   `json:"owner"`
	TrustedAddresses []string `json:"trusted_addresses"` //nolint:tagliatelle //contract spec
	Threshold        int      `json:"threshold"`
}

type Request struct {
	ID        string   `json:"id"`
	Amount    sdk.Coin `json:"amount"`
	Recipient string   `json:"recipient"`
}

type PendingTxQueryRequest struct {
	EvidenceID string `json:"evidence_id"` //nolint:tagliatelle //contract spec
}

type PendingTxsQueryRequest struct {
	Offset *uint64 `json:"offset"`
	Limit  *uint32 `json:"limit"`
}

type SentTxQueryRequest struct {
	ID string `json:"id"`
}

type SentTxsQueryRequest struct {
	Offset *uint64 `json:"offset"`
	Limit  *uint32 `json:"limit"`
}

type QueryTxResponse struct {
	Amount            sdk.Coin `json:"amount"`
	Recipient         string   `json:"recipient"`
	EvidenceProviders []string `json:"evidence_providers"` //nolint:tagliatelle //contract spec
}

type PendingTxResponse struct {
	EvidenceID string `json:"evidence_id"` //nolint:tagliatelle //contract spec
	QueryTxResponse
}

type SentTxResponse struct {
	ID string `json:"id"`
	QueryTxResponse
}

type QueryTxsResponse[T any] struct {
	Transactions []T `json:"transactions"`
}

type Method string

const (
	// transactions.
	MethodThresholdBankSend Method = "threshold_bank_send"
	MethodWithdraw          Method = "withdraw"
	// query.
	MethodGetPendingTransaction  Method = "get_pending_transaction"
	MethodGetPendingTransactions Method = "get_pending_transactions"
	MethodGetSentTransaction     Method = "get_sent_transaction"
	MethodGetSentTransactions    Method = "get_sent_transactions"
)

type executeWasmReq struct {
	fundAmt sdk.Coin
	payload json.RawMessage
}

func TestWASMContractExecuteSend(t *testing.T) {
	t.Parallel()

	ctx, chain := integrationtests.NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()
	trustedAddress2 := chain.GenAccount()
	trustedAddress3 := chain.GenAccount()

	txSendRecipient := chain.GenAccount()

	requireT := require.New(t)
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress2, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress3, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	t.Logf("Deploying and instantiating the smart contract, initial funds:%s.", coinToFundContract.String())
	initialPayload, err := json.Marshal(InstantiateRequest{
		Owner: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold: 2,
	})
	requireT.NoError(err)

	contractByteCode := readContractByteCode(t)
	contractAddr, err := deployAndInstantiateWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(owner),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractByteCode,
		instantiateConfig{
			accessType: wasmtypes.AccessTypeUnspecified,
			payload:    initialPayload,
			label:      "bank_threshold_send",
			amount:     coinToFundContract,
		},
	)
	requireT.NoError(err)
	t.Logf("Contract instantiated, address:%s.", contractAddr)

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	// generate the tx to be sent with the threshold
	coinsToSend := chain.NewCoin(sdk.NewInt(1000))
	txHash := "9752A1D96CA8C54400FD11DD19FD88FC6F386A9DD0E29DE92DDD1FD419389998"
	wasmSendReqPayload, err := json.Marshal(map[Method]Request{
		MethodThresholdBankSend: {
			ID:        txHash,
			Amount:    coinsToSend,
			Recipient: txSendRecipient.String(),
		},
	})
	requireT.NoError(err)

	t.Logf("Trying to execute send from owner address which is not in the list of trusted.")
	_, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(owner),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReq{
			payload: wasmSendReqPayload,
		})
	requireT.ErrorContains(err, "Unauthorized")

	t.Logf("Executing send from first trusted address.")
	txRes, err := executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress1),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReq{
			payload: wasmSendReqPayload,
		})
	requireT.NoError(err)

	// balance of the contract remains the same
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)
	// balance of the recipient remains the same
	assertBankBalance(ctx, t, bankClient, txSendRecipient.String(), chain.NewCoin(sdk.ZeroInt()))

	action, err := event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "result")
	requireT.NoError(err)
	requireT.Equal(action, "pending")

	evidenceID, err := event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "evidence_id")
	requireT.NoError(err)

	pendingTx := queryPendingTx(ctx, t, chain.ClientContext, contractAddr, evidenceID)
	require.Equal(t, QueryTxResponse{
		Amount:            coinsToSend,
		Recipient:         txSendRecipient.String(),
		EvidenceProviders: []string{trustedAddress1.String()},
	}, pendingTx)
	t.Logf("Pending tx: %+v", pendingTx)
	sentTx := querySentTx(ctx, t, chain.ClientContext, contractAddr, txHash)
	require.Equal(t, QueryTxResponse{
		Amount: sdk.Coin{
			Denom:  "",
			Amount: sdk.ZeroInt(),
		},
		Recipient:         "",
		EvidenceProviders: []string{},
	}, sentTx)

	t.Logf("Trying to execute same send from first trusted address.")
	_, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress1),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReq{
			payload: wasmSendReqPayload,
		})
	requireT.ErrorContains(err, "Sender already provided the evidence")

	t.Logf("Executing send from second trusted address with same hash but malicious payload.")
	wasmSendReqMaliciousPayload, err := json.Marshal(map[Method]Request{
		MethodThresholdBankSend: {
			ID:        txHash,
			Amount:    sdk.NewCoin(coinsToSend.Denom, coinsToSend.Amount.Add(sdk.NewInt(1))),
			Recipient: txSendRecipient.String(),
		},
	})
	requireT.NoError(err)

	txRes, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress2),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReq{
			payload: wasmSendReqMaliciousPayload,
		},
	)
	requireT.NoError(err)

	// balance of the contract remains the same
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)
	// balance of the recipient remains the same
	assertBankBalance(ctx, t, bankClient, txSendRecipient.String(), chain.NewCoin(sdk.ZeroInt()))

	action, err = event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "result")
	requireT.NoError(err)
	requireT.Equal(action, "pending")

	evidenceIDWithModifierPayload, err := event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "evidence_id")
	requireT.NoError(err)
	requireT.NotEqual(evidenceID, evidenceIDWithModifierPayload)

	pendingTx = queryPendingTx(ctx, t, chain.ClientContext, contractAddr, evidenceIDWithModifierPayload)
	require.Equal(t, QueryTxResponse{
		Amount:            coinsToSend.AddAmount(sdk.NewInt(1)),
		Recipient:         txSendRecipient.String(),
		EvidenceProviders: []string{trustedAddress2.String()},
	}, pendingTx)
	t.Logf("Pending tx: %+v", pendingTx)
	sentTx = querySentTx(ctx, t, chain.ClientContext, contractAddr, txHash)
	require.Equal(t, QueryTxResponse{
		Amount: sdk.Coin{
			Denom:  "",
			Amount: sdk.ZeroInt(),
		},
		Recipient:         "",
		EvidenceProviders: []string{},
	}, sentTx)

	t.Logf("Executing send from third trusted address.")
	txRes, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress3),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReq{
			payload: wasmSendReqPayload,
		},
	)
	requireT.NoError(err)

	// balance of the contract is updated
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract.Sub(coinsToSend))
	// balance of the recipient is updated
	assertBankBalance(ctx, t, bankClient, txSendRecipient.String(), coinsToSend)

	action, err = event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "result")
	requireT.NoError(err)
	requireT.Equal(action, "sent")

	pendingTx = queryPendingTx(ctx, t, chain.ClientContext, contractAddr, evidenceID)
	require.Equal(t, QueryTxResponse{
		Amount: sdk.Coin{
			Denom:  "",
			Amount: sdk.ZeroInt(),
		},
		Recipient:         "",
		EvidenceProviders: []string{},
	}, pendingTx)
	sentTx = querySentTx(ctx, t, chain.ClientContext, contractAddr, txHash)
	require.Equal(t, QueryTxResponse{
		Amount:            coinsToSend,
		Recipient:         txSendRecipient.String(),
		EvidenceProviders: []string{trustedAddress1.String(), trustedAddress3.String()},
	}, sentTx)
	t.Logf("Sent tx: %+v", sentTx)

	t.Logf("Trying to send the tx with the same ID (hash), but payload which has not been processed.")
	_, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress3),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReq{
			payload: wasmSendReqMaliciousPayload,
		},
	)
	requireT.ErrorContains(err, "Transfer already sent")
}

func TestWASMContractExecuteWithdraw(t *testing.T) {
	t.Parallel()

	ctx, chain := integrationtests.NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()

	requireT := require.New(t)
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)

	coinToFundContract := chain.NewCoin(sdk.NewInt(2_000_000))
	t.Logf("Deploying and instantiating the smart contract, initial funds:%s.", coinToFundContract.String())
	initialPayload, err := json.Marshal(InstantiateRequest{
		Owner: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
		},
		Threshold: 1,
	})
	requireT.NoError(err)

	contractByteCode := readContractByteCode(t)
	contractAddr, err := deployAndInstantiateWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(owner),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractByteCode,
		instantiateConfig{
			accessType: wasmtypes.AccessTypeUnspecified,
			payload:    initialPayload,
			label:      "bank_threshold_send",
			amount:     coinToFundContract,
		},
	)
	requireT.NoError(err)
	t.Logf("Contract instantiated, address:%s.", contractAddr)

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	contractBalanceRes, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: contractAddr,
			Denom:   coinToFundContract.Denom,
		})
	requireT.NoError(err)
	requireT.NotEqual(sdk.ZeroInt().String(), contractBalanceRes.Balance.String())

	wasmWithdrawReqPayload, err := json.Marshal(map[Method]struct{}{
		MethodWithdraw: {},
	})
	requireT.NoError(err)
	t.Logf("Trying to withdraw from the trusted address which is not owner.")
	_, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress1),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReq{
			payload: wasmWithdrawReqPayload,
		},
	)
	requireT.ErrorContains(err, "Unauthorized")

	ownerBalanceBeforeTxRes, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: owner.String(),
			Denom:   coinToFundContract.Denom,
		})
	requireT.NoError(err)

	// we use fixed fees to be precise in the final owner balance
	txFees := chain.NewCoin(sdk.NewInt(100_000))
	t.Logf("Withdrawing with the owner.")
	_, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(owner),
		chain.TxFactory().
			WithGasPrices("").
			WithGas(200_000).
			WithFees(txFees.String()),
		contractAddr,
		executeWasmReq{
			payload: wasmWithdrawReqPayload,
		},
	)
	requireT.NoError(err)

	// contract balance is zero now
	assertBankBalance(ctx, t, bankClient, contractAddr, chain.NewCoin(sdk.ZeroInt()))
	ownerBalanceAfterTxRes, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: owner.String(),
			Denom:   coinToFundContract.Denom,
		})
	requireT.NoError(err)

	requireT.Equal(ownerBalanceAfterTxRes.Balance.Sub(*ownerBalanceBeforeTxRes.Balance).Add(txFees).String(), coinToFundContract.String())
}

func TestWASMContractQueryPagination(t *testing.T) {
	t.Parallel()

	ctx, chain := integrationtests.NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()
	trustedAddress2 := chain.GenAccount()

	txSendRecipient := chain.GenAccount()

	requireT := require.New(t)
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress2, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	t.Logf("Deploying and instantiating the smart contract, initial funds:%s.", coinToFundContract.String())
	initialPayload, err := json.Marshal(InstantiateRequest{
		Owner: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
		},
		Threshold: 2,
	})
	requireT.NoError(err)

	contractByteCode := readContractByteCode(t)
	contractAddr, err := deployAndInstantiateWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(owner),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractByteCode,
		instantiateConfig{
			accessType: wasmtypes.AccessTypeUnspecified,
			payload:    initialPayload,
			label:      "bank_threshold_send",
			amount:     coinToFundContract,
		},
	)
	requireT.NoError(err)
	t.Logf("Contract instantiated, address:%s.", contractAddr)

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	t.Logf("Funding the smart contract to test pagination.")
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(sdk.MustAccAddressFromBech32(contractAddr), chain.NewCoin(sdk.NewInt(1000000000))),
	)

	transactionsCount := 100
	coinsToSendForPagination := chain.NewCoin(sdk.NewInt(1))
	executeWasmReqs := make([]executeWasmReq, 0, transactionsCount)
	t.Logf("Creating %d pending tansactions from first trusted address.", transactionsCount)
	for i := 0; i < transactionsCount; i++ {
		wasmSendReqPayloadPagination, err := json.Marshal(map[Method]Request{
			MethodThresholdBankSend: {
				ID:        fmt.Sprintf("hash%d", i),
				Amount:    coinsToSendForPagination,
				Recipient: txSendRecipient.String(),
			},
		})
		requireT.NoError(err)
		executeWasmReqs = append(executeWasmReqs, executeWasmReq{
			payload: wasmSendReqPayloadPagination,
		})
	}
	_, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress1),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReqs...,
	)
	requireT.NoError(err)

	t.Logf("Queries pending transactions with default pagination.")
	queryReq, err := json.Marshal(map[Method]PendingTxsQueryRequest{
		MethodGetPendingTransactions: {}, // default paging
	})
	require.NoError(t, err)
	queryOut, err := queryWASMContract(ctx, chain.ClientContext, contractAddr, queryReq)
	require.NoError(t, err)
	var pendingTxsRes QueryTxsResponse[PendingTxResponse]
	require.NoError(t, json.Unmarshal(queryOut, &pendingTxsRes))
	require.Equal(t, 50, len(pendingTxsRes.Transactions))
	for _, tx := range pendingTxsRes.Transactions {
		require.Equal(t, 1, len(tx.EvidenceProviders))
	}

	t.Logf("Queries pending transactions with pagination greater than max.")
	queryReq, err = json.Marshal(map[Method]PendingTxsQueryRequest{
		MethodGetPendingTransactions: {
			Limit: lo.ToPtr(uint32(1000)),
		},
	})
	require.NoError(t, err)
	queryOut, err = queryWASMContract(ctx, chain.ClientContext, contractAddr, queryReq)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryOut, &pendingTxsRes))
	require.Equal(t, 50, len(pendingTxsRes.Transactions))

	t.Logf("Queries pending transactions with offet.")
	queryReq, err = json.Marshal(map[Method]PendingTxsQueryRequest{
		MethodGetPendingTransactions: {
			Offset: lo.ToPtr(uint64(90)),
		},
	})
	require.NoError(t, err)
	queryOut, err = queryWASMContract(ctx, chain.ClientContext, contractAddr, queryReq)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryOut, &pendingTxsRes))
	require.Equal(t, 10, len(pendingTxsRes.Transactions))

	executeWasmReqs = make([]executeWasmReq, 0, transactionsCount)
	t.Logf("Confirming %d pending tansactions from second trusted address.", transactionsCount)
	for i := 0; i < transactionsCount; i++ {
		wasmSendReqPayloadPagination, err := json.Marshal(map[Method]Request{
			MethodThresholdBankSend: {
				ID:        fmt.Sprintf("hash%d", i),
				Amount:    coinsToSendForPagination,
				Recipient: txSendRecipient.String(),
			},
		})
		requireT.NoError(err)
		executeWasmReqs = append(executeWasmReqs, executeWasmReq{
			payload: wasmSendReqPayloadPagination,
		})
	}
	_, err = executeWASMContract(
		ctx,
		chain.ClientContext.WithFromAddress(trustedAddress2),
		chain.TxFactory().WithSimulateAndExecute(true),
		contractAddr,
		executeWasmReqs...,
	)
	requireT.NoError(err)

	t.Logf("Queries sent transactions with default pagination.")
	queryReq, err = json.Marshal(map[Method]SentTxsQueryRequest{
		MethodGetSentTransactions: {}, // default paging
	})
	require.NoError(t, err)
	var sentTxsRes QueryTxsResponse[PendingTxResponse]
	queryOut, err = queryWASMContract(ctx, chain.ClientContext, contractAddr, queryReq)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(queryOut, &sentTxsRes))
	require.Equal(t, 50, len(sentTxsRes.Transactions))
	for _, tx := range sentTxsRes.Transactions {
		require.Equal(t, 2, len(tx.EvidenceProviders))
	}
}

func readContractByteCode(t *testing.T) []byte {
	const contractBytesPath = "../contract/artifacts/threshold_bank_send.wasm"
	bytes, err := os.ReadFile(contractBytesPath)
	require.NoError(t, err, fmt.Sprintf("Not found compile contract on the path:%s", contractBytesPath))
	return bytes
}

func queryPendingTx(
	ctx context.Context,
	t *testing.T,
	clientCtx client.Context,
	contractAddr string,
	evidenceID string,
) QueryTxResponse {
	t.Helper()

	queryReq, err := json.Marshal(map[Method]PendingTxQueryRequest{
		MethodGetPendingTransaction: {
			EvidenceID: evidenceID,
		},
	})
	require.NoError(t, err)
	queryOut, err := queryWASMContract(ctx, clientCtx, contractAddr, queryReq)
	require.NoError(t, err)
	var res QueryTxResponse
	require.NoError(t, json.Unmarshal(queryOut, &res))
	return res
}

func querySentTx(
	ctx context.Context,
	t *testing.T,
	clientCtx client.Context,
	contractAddr string,
	id string,
) QueryTxResponse {
	t.Helper()

	queryReq, err := json.Marshal(map[Method]SentTxQueryRequest{
		MethodGetSentTransaction: {
			ID: id,
		},
	})
	require.NoError(t, err)
	queryOut, err := queryWASMContract(ctx, clientCtx, contractAddr, queryReq)
	require.NoError(t, err)
	var res QueryTxResponse
	require.NoError(t, json.Unmarshal(queryOut, &res))
	return res
}

func assertBankBalance(
	ctx context.Context,
	t *testing.T,
	bankClient banktypes.QueryClient,
	address string,
	expectedBalance sdk.Coin,
) {
	t.Helper()

	recipientBalance, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: address,
			Denom:   expectedBalance.Denom,
		})
	require.NoError(t, err)
	require.Equal(t, expectedBalance.Amount.String(), recipientBalance.Balance.Amount.String())
}

// --------------------------- Client ---------------------------

var gasMultiplier = 1.5

// instantiateConfig contains params specific to contract instantiation.
type instantiateConfig struct {
	admin      sdk.AccAddress
	accessType wasmtypes.AccessType
	payload    json.RawMessage
	amount     sdk.Coin
	label      string
	CodeID     uint64
}

// deployAndInstantiateWASMContract deploys, instantiateWASMContract the wasm contract and returns its address.
func deployAndInstantiateWASMContract(ctx context.Context, clientCtx client.Context, txf client.Factory, wasmData []byte, initConfig instantiateConfig) (string, error) {
	codeID, err := deployWASMContract(ctx, clientCtx, txf, wasmData)
	if err != nil {
		return "", err
	}

	initConfig.CodeID = codeID
	contractAddr, err := instantiateWASMContract(ctx, clientCtx, txf, initConfig)
	if err != nil {
		return "", err
	}

	return contractAddr, nil
}

// executeWASMContract executes the wasm contract with the payload and optionally funding amount.
func executeWASMContract(
	ctx context.Context,
	clientCtx client.Context,
	txf client.Factory,
	contractAddr string,
	requests ...executeWasmReq,
) (*sdk.TxResponse, error) {
	msgs := make([]sdk.Msg, 0, len(requests))
	for _, req := range requests {
		funds := sdk.NewCoins()
		if !req.fundAmt.Amount.IsNil() {
			funds = funds.Add(req.fundAmt)
		}
		msg := &wasmtypes.MsgExecuteContract{
			Sender:   clientCtx.FromAddress().String(),
			Contract: contractAddr,
			Msg:      wasmtypes.RawContractMessage(req.payload),
			Funds:    funds,
		}
		msgs = append(msgs, msg)
	}
	txf = txf.WithGasAdjustment(gasMultiplier)
	res, err := client.BroadcastTx(ctx, clientCtx, txf, msgs...)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// queryWASMContract queries the contract with the requested payload.
func queryWASMContract(ctx context.Context, clientCtx client.Context, contractAddr string, payload json.RawMessage) (json.RawMessage, error) {
	query := &wasmtypes.QuerySmartContractStateRequest{
		Address:   contractAddr,
		QueryData: wasmtypes.RawContractMessage(payload),
	}

	wasmClient := wasmtypes.NewQueryClient(clientCtx)
	resp, err := wasmClient.SmartContractState(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "WASMQueryClient returned an error after smart contract state queryWASMContract")
	}

	return json.RawMessage(resp.Data), nil
}

// deploys the wasm contract and returns its codeID.
func deployWASMContract(ctx context.Context, clientCtx client.Context, txf client.Factory, wasmData []byte) (uint64, error) {
	msgStoreCode := &wasmtypes.MsgStoreCode{
		Sender:       clientCtx.FromAddress().String(),
		WASMByteCode: wasmData,
	}

	txf = txf.
		WithGasAdjustment(gasMultiplier)

	res, err := client.BroadcastTx(ctx, clientCtx, txf, msgStoreCode)
	if err != nil {
		return 0, err
	}

	codeID, err := event.FindUint64EventAttribute(res.Events, wasmtypes.EventTypeStoreCode, wasmtypes.AttributeKeyCodeID)
	if err != nil {
		return 0, err
	}

	return codeID, nil
}

// instantiates the contract and returns the contract address.
func instantiateWASMContract(ctx context.Context, clientCtx client.Context, txf client.Factory, req instantiateConfig) (string, error) {
	funds := sdk.NewCoins()
	if amount := req.amount; !amount.Amount.IsNil() {
		funds = funds.Add(amount)
	}
	msg := &wasmtypes.MsgInstantiateContract{
		Sender: clientCtx.FromAddress().String(),
		Admin:  req.admin.String(),
		CodeID: req.CodeID,
		Label:  req.label,
		Msg:    wasmtypes.RawContractMessage(req.payload),
		Funds:  funds,
	}

	txf = txf.
		WithGasAdjustment(gasMultiplier)

	res, err := client.BroadcastTx(ctx, clientCtx, txf, msg)
	if err != nil {
		return "", err
	}

	contractAddr, err := event.FindStringEventAttribute(res.Events, wasmtypes.EventTypeInstantiate, wasmtypes.AttributeKeyContractAddr)
	if err != nil {
		return "", err
	}

	return contractAddr, nil
}
