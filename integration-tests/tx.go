//go:build integrationtests

// Package integrationtests provides common utilities for integration tests.
package integrationtests

import (
	"context"
	"time"

	txapp "github.com/CoreumFoundation/coreum/v5/app"
	"github.com/CoreumFoundation/coreum/v5/pkg/client"
	"github.com/CoreumFoundation/coreum/v5/pkg/config"
	"github.com/CoreumFoundation/coreum/v5/pkg/config/constant"
	"github.com/CoreumFoundation/coreum/v5/testutil/integration"
	feemodeltypes "github.com/CoreumFoundation/coreum/v5/x/feemodel/types"
	"github.com/CosmWasm/wasmd/x/wasm"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/pkg/errors"
)

// TXChainConfig represents TX chain config.
type TXChainConfig struct {
	RPCAddress           string
	GRPCAddress          string
	FundingMnemonic      string
	ContractPath         string
	PreviousContractPath string
}

// TXChain is configured TX chain.
type TXChain struct {
	cfg     TXChainConfig
	TXChain integration.CoreumChain
}

// NewTXChain returns new instance of the TX blockchain.
func NewTXChain(cfg TXChainConfig) (TXChain, error) {
	queryCtx, queryCtxCancel := context.WithTimeout(
		context.Background(),
		getTestContextConfig().TimeoutConfig.RequestTimeout,
	)
	defer queryCtxCancel()

	txGRPCClient, err := integration.DialGRPCClient(cfg.GRPCAddress)
	if err != nil {
		return TXChain{}, errors.WithStack(err)
	}
	txSettings := integration.QueryChainSettings(queryCtx, txGRPCClient)

	txClientCtx := client.NewContext(getTestContextConfig(), auth.AppModuleBasic{}, wasm.AppModuleBasic{}).
		WithGRPCClient(txGRPCClient)

	txFeemodelParamsRes, err := feemodeltypes.
		NewQueryClient(txClientCtx).
		Params(queryCtx, &feemodeltypes.QueryParamsRequest{})
	if err != nil {
		return TXChain{}, errors.WithStack(err)
	}
	txSettings.GasPrice = txFeemodelParamsRes.Params.Model.InitialGasPrice
	txSettings.CoinType = constant.CoinType

	// Set the chosen network for the app (required by integration.NewChain)
	networkCfg, err := config.NetworkConfigByChainID(constant.ChainID(txSettings.ChainID))
	if err != nil {
		return TXChain{}, errors.WithStack(err)
	}
	txapp.ChosenNetwork = networkCfg

	setSDKConfig(txSettings.AddressPrefix)

	return TXChain{
		cfg: cfg,
		TXChain: integration.NewCoreumChain(integration.NewChain(
			txGRPCClient,
			nil,
			txSettings,
			cfg.FundingMnemonic),
			[]string{},
		),
	}, nil
}

// Config returns the chain config.
func (c TXChain) Config() TXChainConfig {
	return c.cfg
}

func getTestContextConfig() client.ContextConfig {
	cfg := client.DefaultContextConfig()
	cfg.TimeoutConfig.TxStatusPollInterval = 100 * time.Millisecond

	return cfg
}

func setSDKConfig(addressPrefix string) {
	config := sdk.GetConfig()

	// Set address & public key prefixes
	config.SetBech32PrefixForAccount(addressPrefix, addressPrefix+"pub")
	config.SetBech32PrefixForValidator(addressPrefix+"valoper", addressPrefix+"valoperpub")
	config.SetBech32PrefixForConsensusNode(addressPrefix+"valcons",
		addressPrefix+"valconspub")

	// Set BIP44 coin type corresponding to CORE
	config.SetCoinType(constant.CoinType)
}
