//go:build integrationtests
// +build integrationtests

package integrationtests

import (
	"context"
	"flag"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/CoreumFoundation/coreum/v3/app"
	"github.com/CoreumFoundation/coreum/v3/pkg/client"
	"github.com/CoreumFoundation/coreum/v3/pkg/config"
	"github.com/CoreumFoundation/coreum/v3/pkg/config/constant"
	"github.com/CoreumFoundation/coreum/v3/testutil/integration"
	feemodeltypes "github.com/CoreumFoundation/coreum/v3/x/feemodel/types"
)

var coreumChain CoreumChain

// flag variables.
var (
	coreumCfg CoreumChainConfig
)

func init() {
	flag.StringVar(
		&coreumCfg.GRPCAddress,
		"coreum-grpc-address",
		"localhost:9090",
		"GRPC address of cored node started by coreum",
	)
	flag.StringVar(
		&coreumCfg.FundingMnemonic,
		"coreum-funding-mnemonic",
		//nolint:lll // one line mnemonic
		"sad hobby filter tray ordinary gap half web cat hard call mystery describe member round trend friend beyond such clap frozen segment fan mistake",
		"Funding coreum account mnemonic required by tests")

	// accept testing flags
	testing.Init()
	// parse additional flags
	flag.Parse()

	var err error
	coreumChain, err = NewCoreumChain(coreumCfg)
	if err != nil {
		panic(errors.Wrapf(err, "failed to init coreum chain"))
	}
}

// CoreumChainConfig represents coreum chain config.
type CoreumChainConfig struct {
	GRPCAddress     string
	FundingMnemonic string
}

// CoreumChain is configured coreum chain.
type CoreumChain struct {
	cfg CoreumChainConfig
	integration.CoreumChain
}

// NewCoreumChain returns new instance of the coreum chain.
func NewCoreumChain(cfg CoreumChainConfig) (CoreumChain, error) {
	queryCtx, queryCtxCancel := context.WithTimeout(
		context.Background(),
		getTestContextConfig().TimeoutConfig.RequestTimeout,
	)
	defer queryCtxCancel()

	coreumGRPCClient := integration.DialGRPCClient(cfg.GRPCAddress)
	coreumSettings := integration.QueryChainSettings(queryCtx, coreumGRPCClient)

	coreumClientCtx := client.NewContext(getTestContextConfig(), app.ModuleBasics).
		WithGRPCClient(coreumGRPCClient)

	coreumFeemodelParamsRes, err := feemodeltypes.NewQueryClient(coreumClientCtx).
		Params(queryCtx, &feemodeltypes.QueryParamsRequest{})
	if err != nil {
		return CoreumChain{}, errors.WithStack(err)
	}
	coreumSettings.GasPrice = coreumFeemodelParamsRes.Params.Model.InitialGasPrice
	coreumSettings.CoinType = constant.CoinType

	config.SetSDKConfig(coreumSettings.AddressPrefix, constant.CoinType)

	return CoreumChain{
		cfg: coreumCfg,
		CoreumChain: integration.NewCoreumChain(integration.NewChain(
			coreumGRPCClient,
			nil,
			coreumSettings,
			cfg.FundingMnemonic),
			[]string{},
		),
	}, nil
}

// NewCoreumTestingContext returns the configured coreum chain and new context for the integration tests.
func NewCoreumTestingContext(t *testing.T) (context.Context, CoreumChain) {
	testCtx, testCtxCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(func() {
		require.NoError(t, testCtx.Err())
		testCtxCancel()
	})

	return testCtx, coreumChain
}

func getTestContextConfig() client.ContextConfig {
	cfg := client.DefaultContextConfig()
	cfg.TimeoutConfig.TxStatusPollInterval = 100 * time.Millisecond

	return cfg
}
