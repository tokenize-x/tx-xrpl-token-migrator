//go:build integrationtests

package integrationtests

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"testing"

	"github.com/CoreumFoundation/coreum-tools/pkg/http"
	txconfig "github.com/CoreumFoundation/coreum/v5/pkg/config"
	txkeyring "github.com/CoreumFoundation/coreum/v5/pkg/keyring"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/xrpl"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

const (
	// XRPCurrencyCode is XRP toke currency code on XRPL chain.
	XRPCurrencyCode = "XRP"

	ecdsaKeyType         = rippledata.ECDSA
	faucetKeyringKeyName = "faucet"
)

// ********** XRPLChain **********

// XRPLChainConfig is a config required for the XRPL chain to be created.
type XRPLChainConfig struct {
	RPCAddress  string
	FundingSeed string
}

// XRPLChain is XRPL chain for the testing.
type XRPLChain struct {
	cfg       XRPLChainConfig
	signer    *xrpl.KeyringTxSigner
	rpcClient *xrpl.RPCClient
	fundMu    *sync.Mutex
}

// NewXRPLChain returns the new instance of the XRPL chain.
func NewXRPLChain(cfg XRPLChainConfig, log logger.Logger) (XRPLChain, error) {
	kr := createInMemoryKeyring()
	faucetPrivateKey, err := extractPrivateKeyFromSeed(cfg.FundingSeed)
	if err != nil {
		return XRPLChain{}, err
	}
	if err := kr.ImportPrivKeyHex(faucetKeyringKeyName, faucetPrivateKey, string(hd.Secp256k1Type)); err != nil {
		return XRPLChain{}, errors.Wrapf(err, "failed to import private key to keyring")
	}

	rpcClient := xrpl.NewRPCClient(
		xrpl.DefaultRPCClientConfig(cfg.RPCAddress),
		log,
		http.NewRetryableClient(http.DefaultClientConfig()),
	)

	signer := xrpl.NewKeyringTxSigner(kr)

	return XRPLChain{
		cfg:       cfg,
		signer:    signer,
		rpcClient: rpcClient,
		fundMu:    &sync.Mutex{},
	}, nil
}

// Config returns the chain config.
func (c XRPLChain) Config() XRPLChainConfig {
	return c.cfg
}

// RPCClient returns the XRPL RPC client.
func (c XRPLChain) RPCClient() *xrpl.RPCClient {
	return c.rpcClient
}

// GenAccount generates the active signer with the initial provided amount.
func (c XRPLChain) GenAccount(ctx context.Context, t *testing.T, amount float64) rippledata.Account {
	t.Helper()

	acc := c.GenEmptyAccount(t)
	c.CreateAccount(ctx, t, acc, amount)

	return acc
}

// GenEmptyAccount generates the signer but doesn't activate it.
func (c XRPLChain) GenEmptyAccount(t *testing.T) rippledata.Account {
	t.Helper()

	const signerKeyName = "signer"
	kr := createInMemoryKeyring()
	_, mnemonic, err := kr.NewMnemonic(
		signerKeyName,
		keyring.English,
		xrpl.XRPLHDPath,
		"",
		hd.Secp256k1,
	)
	require.NoError(t, err)
	acc, err := xrpl.NewKeyringTxSigner(kr).Account(signerKeyName)
	require.NoError(t, err)

	// reimport with the key as signer address
	_, err = c.signer.GetKeyring().NewAccount(
		acc.String(),
		mnemonic,
		"",
		xrpl.XRPLHDPath,
		hd.Secp256k1,
	)
	require.NoError(t, err)

	return acc
}

// CreateAccount funds the provided account with the amount/reserve to activate the account.
func (c XRPLChain) CreateAccount(ctx context.Context, t *testing.T, acc rippledata.Account, amount float64) {
	t.Helper()
	// amount to activate the account and some tokens on top
	c.FundAccount(ctx, t, acc, amount+xrpl.ReserveToActivateAccount)
}

// FundAccount funds the provided account with the provided amount.
func (c XRPLChain) FundAccount(ctx context.Context, t *testing.T, acc rippledata.Account, amount float64) {
	t.Helper()

	c.fundMu.Lock()
	defer c.fundMu.Unlock()

	xrpAmount, err := rippledata.NewAmount(fmt.Sprintf("%f%s", amount, XRPCurrencyCode))
	require.NoError(t, err)
	fundXrpTx := rippledata.Payment{
		Destination: acc,
		Amount:      *xrpAmount,
		TxBase: rippledata.TxBase{
			TransactionType: rippledata.PAYMENT,
		},
	}

	fundingAcc, err := c.signer.Account(faucetKeyringKeyName)
	require.NoError(t, err)
	c.AutoFillTx(ctx, t, &fundXrpTx, fundingAcc)
	require.NoError(t, c.signer.Sign(&fundXrpTx, faucetKeyringKeyName))

	t.Logf("Funding account, account address: %s, amount: %f", acc, amount)
	require.NoError(t, c.RPCClient().SubmitAndAwaitSuccess(ctx, &fundXrpTx))
	t.Logf("The account %s is funded", acc)
}

// AutoFillSignAndSubmitTx autofills the transaction and submits it.
func (c XRPLChain) AutoFillSignAndSubmitTx(
	ctx context.Context, t *testing.T, txn rippledata.Transaction, acc rippledata.Account,
) error {
	t.Helper()

	c.AutoFillTx(ctx, t, txn, acc)
	return c.SignAndSubmitTx(ctx, t, txn, acc)
}

// SignAndSubmitTx signs the transaction from the signer and submits it.
func (c XRPLChain) SignAndSubmitTx(
	ctx context.Context, t *testing.T, txn rippledata.Transaction, acc rippledata.Account,
) error {
	t.Helper()

	require.NoError(t, c.signer.Sign(txn, acc.String()))
	return c.RPCClient().SubmitAndAwaitSuccess(ctx, txn)
}

// AutoFillTx add seq number and fee for the transaction.
func (c XRPLChain) AutoFillTx(
	ctx context.Context,
	t *testing.T,
	txn rippledata.Transaction,
	sender rippledata.Account,
) {
	t.Helper()
	require.NoError(t, c.rpcClient.AutoFillTx(ctx, txn, sender, 1))
}

func extractPrivateKeyFromSeed(seedPhrase string) (string, error) {
	seed, err := rippledata.NewSeedFromAddress(seedPhrase)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create rippledata seed from seed phrase")
	}
	key := seed.Key(ecdsaKeyType)
	return hex.EncodeToString(key.Private(lo.ToPtr(uint32(0)))), nil
}

func createInMemoryKeyring() keyring.Keyring {
	encodingConfig := txconfig.NewEncodingConfig(auth.AppModuleBasic{}, wasm.AppModuleBasic{})
	return txkeyring.NewConcurrentSafeKeyring(keyring.NewInMemory(encodingConfig.Codec))
}
