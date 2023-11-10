//go:build integrationtests

package integrationtests

import (
	"context"
	"fmt"
	"math"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdkmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	multisigtypes "github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdksigning "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/CoreumFoundation/coreum/v3/pkg/client"
	"github.com/CoreumFoundation/coreum/v3/testutil/event"
	integrationtests "github.com/CoreumFoundation/coreum/v3/testutil/integration"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
)

const txHash = "9752A1D96CA8C54400FD11DD19FD88FC6F386A9DD0E29DE92DDD1FD419389998"

// used for the empty responses.
var emptyTx = coreum.Transaction{
	Amount: sdk.Coin{
		Denom:  "",
		Amount: sdk.ZeroInt(),
	},
	Recipient:         "",
	EvidenceProviders: []string{},
}

func TestWASMContractThresholdBankSend(t *testing.T) {
	t.Parallel()

	ctx, chain := NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()
	trustedAddress2 := chain.GenAccount()
	trustedAddress3 := chain.GenAccount()

	txSendRecipient := chain.GenAccount()

	const threshold = 2
	trustedAddresses := []string{
		trustedAddress1.String(),
		trustedAddress2.String(),
		trustedAddress3.String(),
	}
	slices.Sort(trustedAddresses)

	minAmount := sdk.NewIntFromUint64(5)
	maxAmount := sdk.NewIntFromUint64(math.MaxUint64)

	requireT := require.New(t)
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress2, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress3, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), chain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner:            owner.String(),
		Admin:            owner.String(),
		TrustedAddresses: trustedAddresses,
		Threshold:        threshold,
		MinAmount:        minAmount,
		MaxAmount:        maxAmount,
		Label:            "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(contractAddr, coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	// validate contract config

	cfg, err := contractClient.GetContractConfig(ctx)
	requireT.NoError(err)
	requireT.Equal(owner.String(), cfg.Owner)
	requireT.Equal(trustedAddresses, cfg.TrustedAddresses)
	requireT.Equal(threshold, cfg.Threshold)
	requireT.Equal(minAmount.String(), cfg.MinAmount.String())
	requireT.Equal(maxAmount.String(), cfg.MaxAmount.String())

	// generate the tx to be sent with the threshold
	coinsToSend := chain.NewCoin(sdk.NewInt(1000))
	sendExecuteReq := coreum.ThresholdBankSendRequest{
		ID:        txHash,
		Amount:    coinsToSend,
		Recipient: txSendRecipient.String(),
	}

	t.Logf("Trying to execute send from the owner address which is not in the list of trusted addresses.")
	_, err = contractClient.ThresholdBankSend(ctx, owner, sendExecuteReq)
	requireT.True(coreum.IsUnauthorizedError(err))

	t.Logf("Trying to execute with low amount.")
	sendLowAmountExecuteReq := coreum.ThresholdBankSendRequest{
		ID:        txHash,
		Amount:    chain.NewCoin(sdk.NewInt(1)),
		Recipient: txSendRecipient.String(),
	}
	_, err = contractClient.ThresholdBankSend(ctx, trustedAddress1, sendLowAmountExecuteReq)
	requireT.True(coreum.IsLowAmountError(err))

	t.Logf("Executing send from the first trusted address.")
	txRes, err := contractClient.ThresholdBankSend(ctx, trustedAddress1, sendExecuteReq)
	requireT.NoError(err)

	// balance of the contract remains the same
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)
	// balance of the recipient remains the same
	assertBankBalance(ctx, t, bankClient, txSendRecipient, chain.NewCoin(sdk.ZeroInt()))

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
	assertBankBalance(ctx, t, bankClient, txSendRecipient, chain.NewCoin(sdk.ZeroInt()))

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
	assertBankBalance(ctx, t, bankClient, txSendRecipient, coinsToSend)

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

func TestWASMContractExecutePending(t *testing.T) {
	t.Parallel()

	ctx, chain := NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()
	trustedAddress2 := chain.GenAccount()
	trustedAddress3 := chain.GenAccount()

	txSendRecipient := chain.GenAccount()
	anyAddress := chain.GenAccount()

	const threshold = 2
	trustedAddresses := []string{
		trustedAddress1.String(),
		trustedAddress2.String(),
		trustedAddress3.String(),
	}
	slices.Sort(trustedAddresses)

	minAmount := sdk.NewIntFromUint64(5)
	maxAmount := sdk.NewIntFromUint64(100_000)

	requireT := require.New(t)
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress2, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress3, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(anyAddress, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), chain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner:            owner.String(),
		Admin:            owner.String(),
		TrustedAddresses: trustedAddresses,
		Threshold:        threshold,
		MinAmount:        minAmount,
		MaxAmount:        maxAmount,
		Label:            "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(contractAddr, coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	// validate contract config
	cfg, err := contractClient.GetContractConfig(ctx)
	requireT.NoError(err)
	requireT.Equal(owner.String(), cfg.Owner)
	requireT.Equal(trustedAddresses, cfg.TrustedAddresses)
	requireT.Equal(threshold, cfg.Threshold)
	requireT.Equal(minAmount.String(), cfg.MinAmount.String())
	requireT.Equal(maxAmount.String(), cfg.MaxAmount.String())

	// generate the tx with high amount
	coinsToSend := chain.NewCoin(sdk.NewInt(200_000))
	sendExecuteReq := coreum.ThresholdBankSendRequest{
		ID:        txHash,
		Amount:    coinsToSend,
		Recipient: txSendRecipient.String(),
	}

	t.Logf("Executing send from the first trusted address.")
	txRes, err := contractClient.ThresholdBankSend(ctx, trustedAddress1, sendExecuteReq)
	requireT.NoError(err)
	evidenceID, err := event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "evidence_id")
	requireT.NoError(err)

	t.Logf("Trying to execute pending not confirmed transaction from the any address and funds.")
	_, err = contractClient.ExecutePending(ctx, anyAddress, coinsToSend, evidenceID)
	requireT.True(coreum.IsTransactionNotConfirmedError(err))

	t.Logf("Executing send from the second trusted address.")
	_, err = contractClient.ThresholdBankSend(ctx, trustedAddress2, sendExecuteReq)
	requireT.NoError(err)
	t.Logf("Executing send from the third trusted address.")
	txRes, err = contractClient.ThresholdBankSend(ctx, trustedAddress3, sendExecuteReq)
	requireT.NoError(err)
	// balance of the contract remains the same
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)
	// balance of the recipient remains the same
	assertBankBalance(ctx, t, bankClient, txSendRecipient, chain.NewCoin(sdk.ZeroInt()))
	action, err := event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "result")
	requireT.NoError(err)
	requireT.Equal(action, "pending")

	pendingTx, err := contractClient.GetPendingTx(ctx, evidenceID)
	requireT.NoError(err)
	requireT.Equal(coreum.Transaction{
		Amount:    sendExecuteReq.Amount,
		Recipient: sendExecuteReq.Recipient,
		EvidenceProviders: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
	}, pendingTx)
	t.Logf("Pending tx: %+v", pendingTx)
	sentTx, err := contractClient.GetSentTx(ctx, txHash)
	requireT.NoError(err)
	requireT.Equal(emptyTx, sentTx)

	t.Logf("Trying to execute pending transaction from any address and with incorrect funds.")
	_, err = contractClient.ExecutePending(ctx, anyAddress, chain.NewCoin(sdk.NewInt(5_000)), evidenceID)
	requireT.True(coreum.IsFundsMismatchError(err))

	anyAddressBalanceBeforeRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: anyAddress.String(),
		Denom:   pendingTx.Amount.Denom,
	})
	requireT.NoError(err)

	t.Logf("Executing pending transaction from any address with funds.")
	txRes, err = contractClient.ExecutePending(ctx, anyAddress, pendingTx.Amount, evidenceID)
	requireT.NoError(err)

	anyAddressBalanceAfterRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: anyAddress.String(),
		Denom:   pendingTx.Amount.Denom,
	})
	requireT.NoError(err)

	// check that the balance of any address is decreased by the tx amount
	requireT.True(anyAddressBalanceAfterRes.Balance.Amount.Add(pendingTx.Amount.Amount).
		LT(anyAddressBalanceBeforeRes.Balance.Amount))

	// balance of the contract remains the same
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)
	// balance of the recipient is updated
	assertBankBalance(ctx, t, bankClient, txSendRecipient, coinsToSend)

	action, err = event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "result")
	requireT.NoError(err)
	requireT.Equal(action, "sent")

	pendingTx, err = contractClient.GetPendingTx(ctx, evidenceID)
	requireT.NoError(err)
	requireT.Equal(emptyTx, pendingTx)
	sentTx, err = contractClient.GetSentTx(ctx, txHash)
	requireT.NoError(err)
	requireT.Equal(coreum.Transaction{
		Amount:    coinsToSend,
		Recipient: txSendRecipient.String(),
		EvidenceProviders: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
	}, sentTx)
	t.Logf("Sent tx: %+v", sentTx)

	t.Logf("Trying to execute processed pending transaction from the any address with funds.")
	_, err = contractClient.ExecutePending(ctx, anyAddress, pendingTx.Amount, evidenceID)
	requireT.True(coreum.IsTransactionNotFoundError(err))
}

func TestWASMContractExecutePendingWithMultisig(t *testing.T) {
	t.Parallel()

	ctx, chain := NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()
	trustedAddress2 := chain.GenAccount()
	trustedAddress3 := chain.GenAccount()

	txSendRecipient := chain.GenAccount()

	const threshold = 2
	trustedAddresses := []string{
		trustedAddress1.String(),
		trustedAddress2.String(),
		trustedAddress3.String(),
	}
	slices.Sort(trustedAddresses)

	minAmount := sdk.NewIntFromUint64(5)
	maxAmount := sdk.NewIntFromUint64(100_000)

	requireT := require.New(t)

	multisigPublicKey, keyNamesSet, err := chain.GenMultisigAccount(3, 2)
	requireT.NoError(err)

	multisigAddress := sdk.AccAddress(multisigPublicKey.Address())
	signer1KeyName := keyNamesSet[0]
	signer2KeyName := keyNamesSet[1]

	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress2, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(multisigAddress, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, chain.ChainSettings.Denom), chain.ClientContext)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner:            owner.String(),
		Admin:            owner.String(),
		TrustedAddresses: trustedAddresses,
		Threshold:        threshold,
		MinAmount:        minAmount,
		MaxAmount:        maxAmount,
		Label:            "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(contractAddr, coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	// generate the tx with high amount
	coinsToSend := chain.NewCoin(sdk.NewInt(200_000))
	sendExecuteReq := coreum.ThresholdBankSendRequest{
		ID:        txHash,
		Amount:    coinsToSend,
		Recipient: txSendRecipient.String(),
	}

	t.Logf("Executing send from the first trusted address.")
	txRes, err := contractClient.ThresholdBankSend(ctx, trustedAddress1, sendExecuteReq)
	requireT.NoError(err)
	evidenceID, err := event.FindStringEventAttribute(txRes.Events, wasmtypes.ModuleName, "evidence_id")
	requireT.NoError(err)

	t.Logf("Executing send from the second trusted address.")
	_, err = contractClient.ThresholdBankSend(ctx, trustedAddress2, sendExecuteReq)
	requireT.NoError(err)

	pendingTx, err := contractClient.GetPendingTx(ctx, evidenceID)
	requireT.NoError(err)

	requireT.Equal(coreum.Transaction{
		Amount:    sendExecuteReq.Amount,
		Recipient: sendExecuteReq.Recipient,
		EvidenceProviders: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
		},
	}, pendingTx)

	// filter by id
	msgs, err := contractClient.BuildExecutePendingMessages(ctx, multisigAddress, []string{evidenceID})
	requireT.NoError(err)
	requireT.Equal(1, len(msgs))
	// get all
	msgs, err = contractClient.BuildExecutePendingMessages(ctx, multisigAddress, nil)
	requireT.NoError(err)
	requireT.Equal(1, len(msgs))

	fees, gas, err := contractClient.EstimateExecuteMessages(ctx, multisigAddress, msgs...)
	requireT.NoError(err)

	multisigAccInfo, err := client.GetAccountInfo(ctx, chain.ClientContext, multisigAddress)
	requireT.NoError(err)
	txf := chain.TxFactory().
		WithGas(gas).
		WithGasPrices(""). // reset to check fees
		WithFees(fees.String()).
		WithAccountNumber(multisigAccInfo.GetAccountNumber()).
		WithSequence(multisigAccInfo.GetSequence()).
		WithSignMode(sdksigning.SignMode_SIGN_MODE_LEGACY_AMINO_JSON)

	// sign and submit with the min threshold
	txBuilder, err := txf.BuildUnsignedTx(msgs...)
	requireT.NoError(err)
	err = client.Sign(txf, signer1KeyName, txBuilder, false)
	requireT.NoError(err)
	err = client.Sign(txf, signer2KeyName, txBuilder, false)
	requireT.NoError(err)
	multisigTx := createMulisignTx(requireT, txBuilder, multisigAccInfo.GetSequence(), multisigPublicKey)
	encodedTx, err := chain.ClientContext.TxConfig().TxEncoder()(multisigTx)
	requireT.NoError(err)

	multisigAddressBalanceBeforeRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: multisigAddress.String(),
		Denom:   pendingTx.Amount.Denom,
	})
	requireT.NoError(err)

	result, err := client.BroadcastRawTx(ctx, chain.ClientContext, encodedTx)
	requireT.NoError(err)
	t.Logf("Fully signed tx executed, txHash:%s", result.TxHash)

	multisigAddressBalanceAfterRes, err := bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: multisigAddress.String(),
		Denom:   pendingTx.Amount.Denom,
	})
	requireT.NoError(err)

	// check that the balance of multisig address is decreased by the tx amount
	requireT.True(multisigAddressBalanceAfterRes.Balance.Amount.Add(pendingTx.Amount.Amount).
		LT(multisigAddressBalanceBeforeRes.Balance.Amount))

	// balance of the contract remains the same
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)
	// balance of the recipient is updated
	assertBankBalance(ctx, t, bankClient, txSendRecipient, coinsToSend)
}

func TestWASMUpdateMinMaxAmounts(t *testing.T) {
	t.Parallel()

	ctx, chain := NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	anyAddress := chain.GenAccount()

	requireT := require.New(t)
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(anyAddress, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), chain.ClientContext)

	minAmount := sdk.NewIntFromUint64(1)
	maxAmount := sdk.NewIntFromUint64(10_000)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			anyAddress.String(),
		},
		Threshold: 1,
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		Label:     "bank_threshold_send",
	})
	requireT.NoError(err)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	t.Logf("Trying to change min amount from non-owner.")
	newMinAmount := sdk.NewIntFromUint64(5)
	_, err = contractClient.UpdateMinAmount(
		ctx, anyAddress, newMinAmount,
	)
	requireT.True(coreum.IsUnauthorizedError(err))

	t.Logf("Updating min amount from the owner.")
	_, err = contractClient.UpdateMinAmount(
		ctx, owner, newMinAmount,
	)
	requireT.NoError(err)

	cfg, err := contractClient.GetContractConfig(ctx)
	requireT.NoError(err)
	requireT.Equal(newMinAmount.String(), cfg.MinAmount.String())

	t.Logf("Trying to change max amount from non-owner.")
	newMaxAmount := sdk.NewIntFromUint64(100)
	_, err = contractClient.UpdateMaxAmount(
		ctx, anyAddress, newMaxAmount,
	)
	requireT.True(coreum.IsUnauthorizedError(err))

	t.Logf("Updating max amount from owner.")
	_, err = contractClient.UpdateMaxAmount(
		ctx, owner, newMaxAmount,
	)
	requireT.NoError(err)

	cfg, err = contractClient.GetContractConfig(ctx)
	requireT.NoError(err)
	requireT.Equal(newMaxAmount.String(), cfg.MaxAmount.String())
}

func TestWASMContractExecuteWithdraw(t *testing.T) {
	t.Parallel()

	ctx, chain := NewCoreumTestingContext(t)

	owner := chain.GenAccount()
	trustedAddress1 := chain.GenAccount()

	requireT := require.New(t)
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(owner, chain.NewCoin(sdk.NewInt(5000000000))),
		integrationtests.NewFundedAccount(trustedAddress1, chain.NewCoin(sdk.NewInt(5000000000))),
	)

	bankClient := banktypes.NewQueryClient(chain.ClientContext)
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), chain.ClientContext)

	minAmount := sdk.ZeroInt()
	maxAmount := sdk.NewIntFromUint64(math.MaxUint64)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
		},
		Threshold: 1,
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		Label:     "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(100_000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(contractAddr, coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)
	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	contractBalanceRes, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: contractAddr.String(),
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

	ctx, chain := NewCoreumTestingContext(t)

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
	contractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(nil, ""), chain.ClientContext)

	minAmount := sdk.ZeroInt()
	maxAmount := sdk.NewIntFromUint64(math.MaxUint64)

	t.Log("Deploying and instantiating the smart contract.")
	contractAddr, err := contractClient.DeployAndInstantiate(ctx, owner, coreum.DeployAndInstantiateConfig{
		Owner: owner.String(),
		Admin: owner.String(),
		TrustedAddresses: []string{
			trustedAddress1.String(),
			trustedAddress2.String(),
			trustedAddress3.String(),
		},
		Threshold: 2,
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		Label:     "bank_threshold_send",
	})
	requireT.NoError(err)

	coinToFundContract := chain.NewCoin(sdk.NewInt(10_000000))
	chain.Faucet.FundAccounts(ctx, t, integrationtests.NewFundedAccount(contractAddr, coinToFundContract))

	assertBankBalance(ctx, t, bankClient, contractAddr, coinToFundContract)

	requireT.NoError(contractClient.SetContractAddress(contractAddr))
	t.Logf("Contract deployed and instantiated, address:%s.", contractAddr)

	t.Logf("Funding the smart contract to test pagination.")
	chain.Faucet.FundAccounts(ctx, t,
		integrationtests.NewFundedAccount(contractAddr, chain.NewCoin(sdk.NewInt(1000000000))),
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
	require.Equal(t, 100, len(pendingTxs))

	t.Logf("Quering pending transactions with pagination greater than max.")
	pendingTxs, err = contractClient.GetPendingTxs(ctx, nil, lo.ToPtr(uint32(10000)))
	require.NoError(t, err)
	require.Equal(t, 100, len(pendingTxs))

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
	require.Equal(t, 100, len(sentTxs))
	for _, tx := range sentTxs {
		require.Equal(t, 2, len(tx.EvidenceProviders))
	}
}

func assertBankBalance(
	ctx context.Context,
	t *testing.T,
	bankClient banktypes.QueryClient,
	address sdk.AccAddress,
	expectedBalance sdk.Coin,
) {
	t.Helper()

	recipientBalance, err := bankClient.Balance(ctx,
		&banktypes.QueryBalanceRequest{
			Address: address.String(),
			Denom:   expectedBalance.Denom,
		})
	require.NoError(t, err)
	require.Equal(t, expectedBalance.Amount.String(), recipientBalance.Balance.Amount.String())
}

func createMulisignTx(requireT *require.Assertions, txBuilder sdkclient.TxBuilder, accSec uint64, multisigPublicKey *sdkmultisig.LegacyAminoPubKey) authsigning.Tx {
	signs, err := txBuilder.GetTx().GetSignaturesV2()
	requireT.NoError(err)

	multisigSig := multisigtypes.NewMultisig(len(multisigPublicKey.PubKeys))
	for _, sig := range signs {
		requireT.NoError(multisigtypes.AddSignatureV2(multisigSig, sig, multisigPublicKey.GetPubKeys()))
	}

	sigV2 := sdksigning.SignatureV2{
		PubKey:   multisigPublicKey,
		Data:     multisigSig,
		Sequence: accSec,
	}

	requireT.NoError(txBuilder.SetSignatures(sigV2))

	return txBuilder.GetTx()
}
