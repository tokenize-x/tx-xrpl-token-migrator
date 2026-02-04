//go:build integrationtests

package bsc

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	integrationtests "github.com/tokenize-x/tx-xrpl-token-migrator/integration-tests"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

var chains Chains

// flag variables.
var txCfg integrationtests.TXChainConfig

// Chains struct holds chains required for the BSC testing.
type Chains struct {
	TX  integrationtests.TXChain
	Log logger.Logger
}

//nolint:lll // breaking down cli flags will make it less readable.
func init() {
	flag.StringVar(&txCfg.RPCAddress, "tx-rpc-address", "http://localhost:26657", "RPC address of cored node started by TX")
	flag.StringVar(&txCfg.GRPCAddress, "tx-grpc-address", "localhost:9090", "GRPC address of cored node started by TX")
	flag.StringVar(&txCfg.FundingMnemonic, "tx-funding-mnemonic", "sad hobby filter tray ordinary gap half web cat hard call mystery describe member round trend friend beyond such clap frozen segment fan mistake", "Funding TX account mnemonic required by tests")
	flag.StringVar(&txCfg.ContractPath, "tx-contract-path", "../../contract/artifacts/coreumbridge_xrpl.wasm", "Path to smart contract bytecode")
	flag.StringVar(&txCfg.PreviousContractPath, "tx-previous-contract-path", "../../bin/coreumbridge-xrpl-v1.1.0.wasm", "Path to previous smart contract bytecode")

	// accept testing flags
	testing.Init()
	// parse additional flags
	flag.Parse()

	log, err := zap.NewDevelopment()
	if err != nil {
		panic(errors.WithStack(err))
	}
	chains.Log = log

	txChain, err := integrationtests.NewTXChain(txCfg)
	if err != nil {
		panic(errors.Wrapf(err, "failed to init TX chain"))
	}
	chains.TX = txChain
}

// NewTestingContext returns the configured TX chain and new context for the BSC integration tests.
func NewTestingContext(t *testing.T) (context.Context, Chains) {
	//nolint:usetesting // t.Context() cancelled before cleanup runs
	testCtx, testCtxCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(func() {
		require.NoError(t, testCtx.Err())
		testCtxCancel()
	})

	return testCtx, chains
}
