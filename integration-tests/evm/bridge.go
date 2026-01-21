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

// represents a completed bridge transaction.
type BridgeTransaction struct {
	TxHash             common.Hash
	Amount             *big.Int
	DestinationPayload string
	From               common.Address
}

// initiates a bridge transaction.
func Bridge(
	client *ethclient.Client,
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	bridge *bscabi.TxBridge,
	amount *big.Int,
	destinationPayload string,
) (*BridgeTransaction, error) {
	auth, err := getTransactOpts(client, privateKey, chainID)
	if err != nil {
		return nil, err
	}

	tx, err := bridge.Bridge(auth, amount, destinationPayload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to call bridge (from=%s, amount=%s)", auth.From.Hex(), amount.String())
	}

	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to wait for bridge tx")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return nil, errors.New("bridge transaction failed")
	}

	return &BridgeTransaction{
		TxHash:             receipt.TxHash,
		Amount:             amount,
		DestinationPayload: destinationPayload,
		From:               auth.From,
	}, nil
}

// mints tokens to user and bridges them in sequence. full flow for testing.
func MintAndBridge(
	client *ethclient.Client,
	adminPrivateKey *ecdsa.PrivateKey,
	userPrivateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	contracts *DeployedContracts,
	amount *big.Int,
	destinationPayload string,
) (*BridgeTransaction, error) {
	userAuth, err := getTransactOpts(client, userPrivateKey, chainID)
	if err != nil {
		return nil, err
	}
	userAddress := userAuth.From

	if err := MintTokens(client, adminPrivateKey, chainID, contracts.Token, userAddress, amount); err != nil {
		return nil, errors.Wrap(err, "failed to mint tokens")
	}

	// user bridges
	return Bridge(client, userPrivateKey, chainID, contracts.Bridge, amount, destinationPayload)
}

// retrieves BridgeInitiated events from the bridge contract.
func GetBridgeEvents(
	client *ethclient.Client,
	bridge *bscabi.TxBridge,
	fromBlock uint64,
	toBlock *uint64,
) ([]*bscabi.TxBridgeBridgeInitiated, error) {
	iter, err := bridge.FilterBridgeInitiated(&bind.FilterOpts{
		Start:   fromBlock,
		End:     toBlock,
		Context: context.Background(),
	}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter events")
	}
	defer iter.Close()

	var events []*bscabi.TxBridgeBridgeInitiated
	for iter.Next() {
		events = append(events, iter.Event)
	}

	if err := iter.Error(); err != nil {
		return nil, errors.Wrap(err, "error iterating events")
	}

	return events, nil
}

// constructs the destination payload from address and chain ID.
func BuildDestinationPayload(bech32Address, chainID string) string {
	return bech32Address + chainID
}
