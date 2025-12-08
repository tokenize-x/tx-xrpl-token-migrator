//go:build integrationtests

package integrationtests

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"
)

// Test constants matching defaultTestnetCfg from main.go.
const (
	xrplTestMemoSuffix = "/integration-test"
	xrplCORECurrency   = "434F524500000000000000000000000000000000"
	xrplXCORECurrency  = "58434F5245000000000000000000000000000000"
	xrplSOLOCurrency   = "534F4C4F00000000000000000000000000000000"
)

// testXRPLTokens matches the defaultTestnetCfg XRPL tokens configuration.
var testXRPLTokens = []tx.XRPLToken{
	{
		Currency:       xrplCORECurrency,
		Issuer:         "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA",
		ActivationDate: 946684800, // time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
		Multiplier:     "1.0",
	},
	{
		Currency:       xrplXCORECurrency,
		Issuer:         "rawnyFwFLkntQttzBgEFiASg5iB5ULdKpX",
		ActivationDate: 946684800,
		Multiplier:     "1.0",
	},
	{
		Currency:       xrplSOLOCurrency,
		Issuer:         "rHZwvHEs56GCmHupwjA4RY7oPA3EoAJWuN",
		ActivationDate: 946684800,
		Multiplier:     "1.25",
	},
}

var chains Chains

// flag variables.
var (
	txCfg   TXChainConfig
	xrplCfg XRPLChainConfig
)

// Chains struct holds chains required for the testing.
type Chains struct {
	TX   TXChain
	XRPL XRPLChain
	Log  logger.Logger
}

//nolint:lll // breaking down cli flags will make it less readable.
func init() {
	flag.StringVar(&txCfg.RPCAddress, "tx-rpc-address", "http://localhost:26657", "RPC address of cored node started by TX")
	flag.StringVar(&txCfg.GRPCAddress, "tx-grpc-address", "localhost:9090", "GRPC address of cored node started by TX")
	flag.StringVar(&txCfg.FundingMnemonic, "tx-funding-mnemonic", "sad hobby filter tray ordinary gap half web cat hard call mystery describe member round trend friend beyond such clap frozen segment fan mistake", "Funding TX account mnemonic required by tests")
	flag.StringVar(&txCfg.ContractPath, "tx-contract-path", "../../contract/artifacts/coreumbridge_xrpl.wasm", "Path to smart contract bytecode")
	flag.StringVar(&txCfg.PreviousContractPath, "tx-previous-contract-path", "../../bin/coreumbridge-xrpl-v1.1.0.wasm", "Path to previous smart contract bytecode")
	flag.StringVar(&xrplCfg.RPCAddress, "xrpl-rpc-address", "http://localhost:5005", "RPC address of xrpl node")
	flag.StringVar(&xrplCfg.FundingSeed, "xrpl-funding-seed", "snoPBrXtMeMyMHUVTgbuqAfg1SUTb", "Funding XRPL account seed required by tests")

	// accept testing flags
	testing.Init()
	// parse additional flags
	flag.Parse()

	log, err := zap.NewDevelopment()
	if err != nil {
		panic(errors.WithStack(err))
	}
	chains.Log = log

	txChain, err := NewTXChain(txCfg)
	if err != nil {
		panic(errors.Wrapf(err, "failed to init TX chain"))
	}
	chains.TX = txChain

	xrplChain, err := NewXRPLChain(xrplCfg, chains.Log)
	if err != nil {
		panic(errors.Wrapf(err, "failed to init XRPL chain"))
	}
	chains.XRPL = xrplChain
}

// NewTestingContext returns the configured TX and XRPL chains and new context for the integration tests.
func NewTestingContext(t *testing.T) (context.Context, Chains) {
	testCtx, testCtxCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(func() {
		require.NoError(t, testCtx.Err())
		testCtxCancel()
	})

	return testCtx, chains
}

// NewTXTestingContext returns the configured TX blockchain and new context for the integration tests.
func NewTXTestingContext(t *testing.T) (context.Context, TXChain) {
	testCtx, testChains := NewTestingContext(t)

	return testCtx, testChains.TX
}
