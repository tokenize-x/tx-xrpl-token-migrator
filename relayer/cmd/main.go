package main

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
	"github.com/CoreumFoundation/coreum-tools/pkg/run"
	"github.com/CoreumFoundation/coreum/pkg/config/constant"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/service"
)

// Build options.
var (
	BuildDate    = ""
	BuildVersion = ""
)

// temporary constants.
const (
	testnetXRPLRPCURL                 = "https://s.altnet.rippletest.net:51234/"
	testnetXRPLHistoryScanStartLedger = 0
	testnetXRPLRecentScanIndexesBack  = 30_000
	testnetXRPLAccount                = "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA"
	testnetXRPLCurrency               = "434F524500000000000000000000000000000000"
	testnetXRPLIssuer                 = "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA"
	testnetXRPLMemoSuffix             = "+coreum"

	testnetCoreumGRPCURL      = "full-node.testnet-1.coreum.dev:9090"
	testnetCoreumGRPCIsSecure = true
	testnetCoreumChainID      = constant.ChainIDTest

	testnetCoreumMnemonic        = "witness crouch lecture dish there because prevent garlic position illness poverty oven filter tongue choose hole valid quote tattoo physical cliff breeze insane leave"
	testnetCoreumContractAddress = "testcore1z8ed6e8j9ega7z0enadeeaz5f47zzge85ra33gzn75pgtyxyn5xqgyy9uu"
)

func main() {
	run.Service("bridge", func(ctx context.Context) error {
		services, err := service.NewServices(service.Config{
			XRPLRPCURL:                 testnetXRPLRPCURL,
			XRPLHistoryScanStartLedger: testnetXRPLHistoryScanStartLedger,
			XRPLRecentScanIndexesBack:  testnetXRPLRecentScanIndexesBack,
			XRPLAccount:                testnetXRPLAccount,
			XRPLCurrency:               testnetXRPLCurrency,
			XRPLIssuer:                 testnetXRPLIssuer,
			XRPLMemoSuffix:             testnetXRPLMemoSuffix,

			CoreumGRPCURL:         testnetCoreumGRPCURL,
			CoreumGRPCIsSecure:    testnetCoreumGRPCIsSecure,
			CoreumChainID:         string(testnetCoreumChainID),
			CoreumMnemonic:        testnetCoreumMnemonic,
			CoreumContractAddress: testnetCoreumContractAddress,
		}, true)
		if err != nil {
			return err
		}
		log := logger.Get(ctx)
		log.Info(fmt.Sprintf("Build date: %s, version: %s", BuildDate, BuildVersion))
		rootCmd := RootCmd(ctx, services)
		if err := rootCmd.Execute(); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("Error executing root cmd.", zap.Error(err))
			return err
		}

		return nil
	})
}

// RootCmd returns the root cmd.
func RootCmd(ctx context.Context, services *service.Services) *cobra.Command {
	cmd := &cobra.Command{
		Short: "XRPL relayer.",
	}

	cmd.AddCommand(StartCmd(ctx, services))

	return cmd
}

// StartCmd returns the start cmd.
func StartCmd(ctx context.Context, services *service.Services) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start xrpl to coreum relayer.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.Get(ctx)
			log.Info("Starting xrpl relayer.")
			return services.Executor.Start(ctx)
		},
	}

	return cmd
}
