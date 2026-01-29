//go:build integrationtests

package evm

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"

	bscabi "github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc/abi"
)

// BridgeTransaction represents a completed bridge transaction.
type BridgeTransaction struct {
	TxHash    common.Hash
	Amount    *big.Int
	TxAddress string
	From      common.Address
}

// SendToTxChain initiates a bridge transaction.
func SendToTxChain(
	ctx context.Context,
	client *ethclient.Client,
	chainID *big.Int,
	ks *keystore.KeyStore,
	account accounts.Account,
	bridge *bscabi.TXBridge,
	amount *big.Int,
	txAddress string,
) (*BridgeTransaction, error) {
	auth, err := getTransactOpts(ctx, client, chainID, ks, account)
	if err != nil {
		return nil, err
	}

	tx, err := bridge.SendToTXChain(auth, amount, txAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call sendToTxChain (from=%s, amount=%s)", auth.From.Hex(), amount.String())
	}

	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wait for bridge tx")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("bridge transaction failed")
	}

	return &BridgeTransaction{
		TxHash:    receipt.TxHash,
		Amount:    amount,
		TxAddress: txAddress,
		From:      auth.From,
	}, nil
}

// MintAndSendToTxChain mints tokens to user and bridges them in sequence.
func MintAndSendToTxChain(
	ctx context.Context,
	client *ethclient.Client,
	chainID *big.Int,
	ks *keystore.KeyStore,
	adminAccount accounts.Account,
	userAccount accounts.Account,
	contracts *DeployedContracts,
	amount *big.Int,
	txAddress string,
) (*BridgeTransaction, error) {
	userAuth, err := getTransactOpts(ctx, client, chainID, ks, userAccount)
	if err != nil {
		return nil, err
	}
	userAddress := userAuth.From

	if err := MintTokens(ctx, client, chainID, ks, adminAccount, contracts.Token, userAddress, amount); err != nil {
		return nil, errors.Wrap(err, "failed to mint tokens")
	}

	return SendToTxChain(ctx, client, chainID, ks, userAccount, contracts.Bridge, amount, txAddress)
}

// GetBridgeEvents retrieves SentToTXChain events from the bridge contract.
func GetBridgeEvents(
	ctx context.Context,
	bridge *bscabi.TXBridge,
	fromBlock uint64,
	toBlock *uint64,
) ([]*bscabi.TXBridgeSentToTXChain, error) {
	iter, err := bridge.FilterSentToTXChain(&bind.FilterOpts{
		Start:   fromBlock,
		End:     toBlock,
		Context: ctx,
	}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter events")
	}
	defer iter.Close()

	var events []*bscabi.TXBridgeSentToTXChain
	for iter.Next() {
		events = append(events, iter.Event)
	}

	if err := iter.Error(); err != nil {
		return nil, errors.Wrap(err, "error iterating events")
	}

	return events, nil
}
