//go:build integrationtests

package integrationtests

import (
	"context"
	"fmt"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	integrationtests "github.com/CoreumFoundation/coreum/integration-tests"
	"github.com/CoreumFoundation/coreum/testutil/event"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
)

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

	// module used for the empty responses
	emptyTx := coreum.Transaction{
		Amount: sdk.Coin{
			Denom:  "",
			Amount: sdk.ZeroInt(),
		},
		Recipient:         "",
		EvidenceProviders: []string{},
	}

	bankClient := banktypes.NewQueryClient(chain.ClientContext)
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(""), chain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold:  2,
		AccessType: wasmtypes.AccessTypeUnspecified,
		Label:      "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(sdk.MustAccAddressFromBech32(contractAddr), coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	// generate the tx to be sent with the threshold
	coinsToSend := chain.NewCoin(sdk.NewInt(1000))
	txHash := "9752A1D96CA8C54400FD11DD19FD88FC6F386A9DD0E29DE92DDD1FD419389998"
	sendExecuteReq := coreum.ThresholdBankSendRequest{
		ID:        txHash,
		Amount:    coinsToSend,
		Recipient: txSendRecipient.String(),
	}

	t.Logf("Trying to execute send from the owner address which is not in the list of trusted addresses.")
	_, err = contractClient.ThresholdBankSend(ctx, owner, sendExecuteReq)
	requireT.True(coreum.IsUnauthorizedError(err))

	t.Logf("Executing send from the first trusted address.")
	txRes, err := contractClient.ThresholdBankSend(ctx, trustedAddress1, sendExecuteReq)
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

	pendingTx, err := contractClient.GetPendingTx(ctx, evidenceID)
	requireT.NoError(err)
	requireT.Equal(coreum.Transaction{
		Amount:            sendExecuteReq.Amount,
		Recipient:         sendExecuteReq.Recipient,
		EvidenceProviders: []string{trustedAddress1.String()},
	}, pendingTx)
	t.Logf("Pending tx: %+v", pendingTx)
	sentTx, err := contractClient.GetSentTx(ctx, txHash)
	requireT.NoError(err)
	requireT.Equal(emptyTx, sentTx)

	t.Logf("Trying to execute same send from the first trusted address.")
	_, err = contractClient.ThresholdBankSend(ctx, trustedAddress1, sendExecuteReq)
	requireT.True(coreum.IsEvidenceProvidedError(err))

	t.Logf("Executing send from the second trusted address with same hash but modified payload.")
	modifiedSendExecuteReq := sendExecuteReq
	modifiedSendExecuteReq.Amount = coinsToSend.Add(chain.NewCoin(sdk.NewInt(1)))
	txRes, err = contractClient.ThresholdBankSend(ctx, trustedAddress2, modifiedSendExecuteReq)
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

	pendingTx, err = contractClient.GetPendingTx(ctx, evidenceIDWithModifierPayload)
	requireT.NoError(err)
	requireT.Equal(coreum.Transaction{
		Amount:            modifiedSendExecuteReq.Amount,
		Recipient:         modifiedSendExecuteReq.Recipient,
		EvidenceProviders: []string{trustedAddress2.String()},
	}, pendingTx)
	t.Logf("Pending tx: %+v", pendingTx)
	sentTx, err = contractClient.GetSentTx(ctx, txHash)
	requireT.NoError(err)
	requireT.Equal(emptyTx, sentTx)

	t.Logf("Executing send from the third trusted address.")
	txRes, err = contractClient.ThresholdBankSend(ctx, trustedAddress3, sendExecuteReq)
	requireT.NoError(err)

	// balance of the contract is updated
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract.Sub(coinsToSend))
	// balance of the recipient is updated
	assertBankBalance(ctx, t, bankClient, txSendRecipient.String(), coinsToSend)

	action, err = event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "result")
	requireT.NoError(err)
	requireT.Equal(action, "sent")

	pendingTx, err = contractClient.GetPendingTx(ctx, evidenceID)
	requireT.NoError(err)
	requireT.Equal(emptyTx, pendingTx)
	sentTx, err = contractClient.GetSentTx(ctx, txHash)
	requireT.NoError(err)
	requireT.Equal(coreum.Transaction{
		Amount:            coinsToSend,
		Recipient:         txSendRecipient.String(),
		EvidenceProviders: []string{trustedAddress1.String(), trustedAddress3.String()},
	}, sentTx)
	t.Logf("Sent tx: %+v", sentTx)

	t.Logf("Trying to send the tx with the same ID (hash), but payload which has not been processed.")
	_, err = contractClient.ThresholdBankSend(ctx, trustedAddress2, modifiedSendExecuteReq)

	requireT.True(coreum.IsTransferSentError(err))
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
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(""), chain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
		},
		Threshold:  1,
		AccessType: wasmtypes.AccessTypeUnspecified,
		Label:      "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(sdk.MustAccAddressFromBech32(contractAddr), coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	contractBalanceRes, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: contractAddr,
			Denom:   coinToFundContract.Denom,
		})
	requireT.NoError(err)
	requireT.NotEqual(sdk.ZeroInt().String(), contractBalanceRes.Balance.String())

	t.Logf("Trying to withdraw from the trusted address which is not owner.")
	_, err = contractClient.Withdraw(
		ctx, trustedAddress1,
	)
	requireT.True(coreum.IsUnauthorizedError(err))

	ownerBalanceBeforeTxRes, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: owner.String(),
			Denom:   coinToFundContract.Denom,
		})
	requireT.NoError(err)

	t.Logf("Withdrawing with the owner.")
	_, err = contractClient.Withdraw(ctx, owner)
	requireT.NoError(err)

	// contract balance is zero now
	assertBankBalance(ctx, t, bankClient, contractAddr, chain.NewCoin(sdk.ZeroInt()))
	ownerBalanceAfterTxRes, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: owner.String(),
			Denom:   coinToFundContract.Denom,
		})
	requireT.NoError(err)

	requireT.True(ownerBalanceAfterTxRes.Balance.Amount.GT(ownerBalanceBeforeTxRes.Balance.Amount))
}

func TestWASMContractQueryPagination(t *testing.T) {
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
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(""), chain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold:  2,
		AccessType: wasmtypes.AccessTypeUnspecified,
		Label:      "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(sdk.MustAccAddressFromBech32(contractAddr), coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	t.Logf("Funding the smart contract to test pagination.")
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(sdk.MustAccAddressFromBech32(contractAddr), chain.NewCoin(sdk.NewInt(1000000000))),
	)

	transactionsCount := 100
	coinsToSendForPagination := chain.NewCoin(sdk.NewInt(1))
	sendExecuteReqBatch := make([]coreum.ThresholdBankSendRequest, 0, transactionsCount)
	t.Logf("Creating %d pending tansactions from first trusted address.", transactionsCount)
	for i := 0; i < transactionsCount; i++ {
		sendExecuteReqBatch = append(sendExecuteReqBatch, coreum.ThresholdBankSendRequest{
			ID:        fmt.Sprintf("hash1-tx%d", i),
			Amount:    coinsToSendForPagination,
			Recipient: txSendRecipient.String(),
		})
	}
	_, err = contractClient.ThresholdBankSend(ctx, trustedAddress1, sendExecuteReqBatch...)
	requireT.NoError(err)

	t.Logf("Quering pending transactions with default pagination.")
	pendingTxs, err := contractClient.GetPendingTxs(ctx, nil, nil)
	require.NoError(t, err)
	for _, tx := range pendingTxs {
		require.Equal(t, 1, len(tx.EvidenceProviders))
	}

	t.Logf("Quering pending transactions with pagination greater than max.")
	pendingTxs, err = contractClient.GetPendingTxs(ctx, nil, lo.ToPtr(uint32(10000)))
	require.NoError(t, err)
	require.Equal(t, 50, len(pendingTxs))

	t.Logf("Quering pending transactions with offet.")
	pendingTxs, err = contractClient.GetPendingTxs(ctx, lo.ToPtr(uint64(90)), nil)
	require.NoError(t, err)
	require.Equal(t, 10, len(pendingTxs))

	t.Logf("Confirming %d pending tansactions from second trusted address.", transactionsCount)
	_, err = contractClient.ThresholdBankSend(ctx, trustedAddress2, sendExecuteReqBatch...)
	requireT.NoError(err)

	t.Logf("Quering sent transactions with default pagination.")
	sentTxs, err := contractClient.GetSentTxs(ctx, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 50, len(sentTxs))
	for _, tx := range sentTxs {
		require.Equal(t, 2, len(tx.EvidenceProviders))
	}
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
