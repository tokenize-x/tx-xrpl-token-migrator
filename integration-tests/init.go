//go:build integrationtests

package integrationtests

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// Test constants matching defaultTestnetCfg from main.go.
const (
	xrplTestMemoSuffix = "/integration-test"
	xrplCORECurrency   = "434F524500000000000000000000000000000000"
	xrplXCORECurrency  = "58434F5245000000000000000000000000000000"
	xrplSOLOCurrency   = "534F4C4F00000000000000000000000000000000"
)

// testXRPLTokens matches the defaultTestnetCfg XRPL tokens configuration.
var testXRPLTokens = []coreum.XRPLToken{
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
	coreumCfg CoreumChainConfig
	xrplCfg   XRPLChainConfig
)

// Chains struct holds chains required for the testing.
type Chains struct {
	Coreum CoreumChain
	XRPL   XRPLChain
	Log    logger.Logger
}

//nolint:lll // breaking down cli flags will make it less readable.
func init() {
	flag.StringVar(&coreumCfg.RPCAddress, "coreum-rpc-address", "http://localhost:26657", "RPC address of cored node started by coreum")
	flag.StringVar(&coreumCfg.GRPCAddress, "coreum-grpc-address", "localhost:9090", "GRPC address of cored node started by coreum")
	flag.StringVar(&coreumCfg.FundingMnemonic, "coreum-funding-mnemonic", "sad hobby filter tray ordinary gap half web cat hard call mystery describe member round trend friend beyond such clap frozen segment fan mistake", "Funding coreum account mnemonic required by tests")
	flag.StringVar(&coreumCfg.ContractPath, "coreum-contract-path", "../../contract/artifacts/coreumbridge_xrpl.wasm", "Path to smart contract bytecode")
	flag.StringVar(&coreumCfg.PreviousContractPath, "coreum-previous-contract-path", "../../bin/coreumbridge-xrpl-v1.1.0.wasm", "Path to previous smart contract bytecode")
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

	coreumChain, err := NewCoreumChain(coreumCfg)
	if err != nil {
		panic(errors.Wrapf(err, "failed to init coreum chain"))
	}
	chains.Coreum = coreumChain

	xrplChain, err := NewXRPLChain(xrplCfg, chains.Log)
	if err != nil {
		panic(errors.Wrapf(err, "failed to init coreum chain"))
	}
	chains.XRPL = xrplChain
}

// NewTestingContext returns the configured coreum and xrpl chains and new context for the integration tests.
func NewTestingContext(t *testing.T) (context.Context, Chains) {
	testCtx, testCtxCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(func() {
		require.NoError(t, testCtx.Err())
		testCtxCancel()
	})

	return testCtx, chains
}

// NewCoreumTestingContext returns the configured coreum chain and new context for the integration tests.
func NewCoreumTestingContext(t *testing.T) (context.Context, CoreumChain) {
	testCtx, testChains := NewTestingContext(t)

	return testCtx, testChains.Coreum
}
