//go:build integrationtests

package evm

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	bridge *bscabi.TxBridge,
	amount *big.Int,
	txAddress string,
) (*BridgeTransaction, error) {
	auth, err := getTransactOpts(ctx, client, privateKey, chainID)
	if err != nil {
		return nil, err
	}

	tx, err := bridge.SendToTxChain(auth, amount, txAddress)
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
	adminPrivateKey *ecdsa.PrivateKey,
	userPrivateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	contracts *DeployedContracts,
	amount *big.Int,
	txAddress string,
) (*BridgeTransaction, error) {
	userAuth, err := getTransactOpts(ctx, client, userPrivateKey, chainID)
	if err != nil {
		return nil, err
	}
	userAddress := userAuth.From

	if err := MintTokens(ctx, client, adminPrivateKey, chainID, contracts.Token, userAddress, amount); err != nil {
		return nil, errors.Wrap(err, "failed to mint tokens")
	}

	return SendToTxChain(ctx, client, userPrivateKey, chainID, contracts.Bridge, amount, txAddress)
}

// GetBridgeEvents retrieves SentToTxChain events from the bridge contract.
func GetBridgeEvents(
	ctx context.Context,
	bridge *bscabi.TxBridge,
	fromBlock uint64,
	toBlock *uint64,
) ([]*bscabi.TxBridgeSentToTxChain, error) {
	iter, err := bridge.FilterSentToTxChain(&bind.FilterOpts{
		Start:   fromBlock,
		End:     toBlock,
		Context: ctx,
	}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter events")
	}
	defer iter.Close()

	var events []*bscabi.TxBridgeSentToTxChain
	for iter.Next() {
		events = append(events, iter.Event)
	}

	if err := iter.Error(); err != nil {
		return nil, errors.Wrap(err, "error iterating events")
	}

	return events, nil
}
