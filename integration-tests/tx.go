//go:build integrationtests

package integrationtests

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"github.com/CoreumFoundation/coreum/v4/app"
	"github.com/CoreumFoundation/coreum/v4/pkg/client"
	"github.com/CoreumFoundation/coreum/v4/pkg/config/constant"
	"github.com/CoreumFoundation/coreum/v4/testutil/integration"
	feemodeltypes "github.com/CoreumFoundation/coreum/v4/x/feemodel/types"
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

	txClientCtx := client.NewContext(getTestContextConfig(), app.ModuleBasics).
		WithGRPCClient(txGRPCClient)

	txFeemodelParamsRes, err := feemodeltypes.
		NewQueryClient(txClientCtx).
		Params(queryCtx, &feemodeltypes.QueryParamsRequest{})
	if err != nil {
		return TXChain{}, errors.WithStack(err)
	}
	txSettings.GasPrice = txFeemodelParamsRes.Params.Model.InitialGasPrice
	txSettings.CoinType = constant.CoinType

	setSDKConfig(txSettings.AddressPrefix)

	return TXChain{
		cfg: txCfg,
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
