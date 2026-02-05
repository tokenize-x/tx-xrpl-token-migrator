//go:build integrationtests

package bsc

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/tokenize-x/tx-xrpl-token-migrator/integration-tests/bsc/evm"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

const (
	// BSCCurrencyCode is BSC toke currency code on BSC chain.
	// BSCCurrencyCode = "BSC"

	veryLightScryptN = 2
	veryLightScryptP = 1
)

// ********** Binance Smart Chain **********

// BinanceSmartChainConfig is a config required for the BSC chain to be created.
type BinanceSmartChainConfig struct {
	RPCAddress  string
	FundingSeed string
	ChainID     int64
}

// BinanceSmartChain is BSC chain for the testing.
type BinanceSmartChain struct {
	cfg       BinanceSmartChainConfig
	rpcClient *ethclient.Client
	faucet    accounts.Account
	keystore  *keystore.KeyStore
	fundMu    *sync.Mutex
}

// NewBinanceSmartChain returns the new instance of the BSC chain.
func NewBinanceSmartChain(cfg BinanceSmartChainConfig, log logger.Logger) (BinanceSmartChain, error) {
	ks, err := createTemporaryKeyStore()
	if err != nil {
		return BinanceSmartChain{}, errors.Wrapf(err, "failed to create temporary keystore")
	}
	faucetPrivateKey, err := extractPrivateKeyFromSeed(cfg.FundingSeed)
	if err != nil {
		return BinanceSmartChain{}, errors.Wrapf(err, "failed to extract private key from seed phrase")
	}
	faucetAccount, err := ks.ImportECDSA(faucetPrivateKey, keyring.DefaultBIP39Passphrase)
	if err != nil {
		return BinanceSmartChain{}, errors.Wrapf(err, "failed to import private key to keyring")
	}
	if err = ks.Unlock(faucetAccount, keyring.DefaultBIP39Passphrase); err != nil {
		return BinanceSmartChain{}, errors.Wrapf(err, "failed to unlock account")
	}

	client, err := ethclient.Dial(cfg.RPCAddress)
	if err != nil {
		return BinanceSmartChain{}, errors.Wrap(err, "failed to connect to BSC RPC")
	}

	return BinanceSmartChain{
		cfg:       cfg,
		rpcClient: client,
		faucet:    faucetAccount,
		keystore:  ks,
		fundMu:    &sync.Mutex{},
	}, nil
}

// Config returns the chain config.
func (c BinanceSmartChain) Config() BinanceSmartChainConfig {
	return c.cfg
}

// RPCClient returns the BSC RPC client.
func (c BinanceSmartChain) RPCClient() *ethclient.Client {
	return c.rpcClient
}

// KeyStore returns the BSC key store.
func (c BinanceSmartChain) KeyStore() *keystore.KeyStore {
	return c.keystore
}

// ChainID returns the BSC chain ID.
func (c BinanceSmartChain) ChainID() *big.Int {
	return big.NewInt(c.cfg.ChainID)
}

// GenAccount generates the signer.
func (c BinanceSmartChain) GenAccount(t *testing.T) accounts.Account {
	t.Helper()

	acc, err := c.keystore.NewAccount(keyring.DefaultBIP39Passphrase)
	require.NoError(t, err)
	require.NoError(t, c.keystore.Unlock(acc, keyring.DefaultBIP39Passphrase))
	return acc
}

// FundAccount funds the provided account with the provided amount.
func (c BinanceSmartChain) FundAccount(
	ctx context.Context, t *testing.T, client *ethclient.Client, acc common.Address, amount *big.Int,
) {
	t.Helper()

	c.fundMu.Lock()
	defer c.fundMu.Unlock()

	t.Logf("Funding account, account address: %s, amount: %s", acc.Hex(), amount.String())
	_, err := evm.TransferFunds(ctx, client, c.ChainID(), c.KeyStore(), c.faucet, acc, amount)
	require.NoError(t, err)
	t.Logf("The account %s is funded", acc.Hex())
}

func extractPrivateKeyFromSeed(seedPhrase string) (*ecdsa.PrivateKey, error) {
	seed := bip39.NewSeed(seedPhrase, keyring.DefaultBIP39Passphrase)
	masterPriv, ch := hd.ComputeMastersFromSeed(seed)
	hdPath := hd.CreateHDPath(60, 0, 0).String()
	derivedKey, err := hd.DerivePrivateKeyForPath(masterPriv, ch, hdPath)
	if err != nil {
		return nil, err
	}
	return crypto.ToECDSA(derivedKey)
}

func createTemporaryKeyStore() (*keystore.KeyStore, error) {
	d, err := os.MkdirTemp("", "bsc")
	if err != nil {
		return nil, err
	}
	return keystore.NewKeyStore(d, veryLightScryptN, veryLightScryptP), nil
}
