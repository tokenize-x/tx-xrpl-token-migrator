package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/http"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/xrpl"
)

// Build options.
var (
	BuildDate    = ""
	BuildVersion = ""
)

// temporary constants.
const (
	mainnetXRPLRPCURL                   = "https://s2.ripple.com:51234/"
	mainnetXRPLWSURL                    = "wss://s2.ripple.com/"
	mainnetXRPLCoreAccount              = "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D"
	mainnetXRPLInitialBridgeLedgerIndex = 80175264
)

type servicesConfig struct {
	XRPLRPCURL string
	XRPLWSURL  string
}

type services struct {
	XRPLTxScanner *xrpl.TxScanner
}

func newServices(cfg servicesConfig) *services {
	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	rpcClientConfig := xrpl.DefaultRPCClientConfig(cfg.XRPLRPCURL)
	rpcClient := xrpl.NewRPCClient(rpcClientConfig, httpClient)

	xrplTxScanner := xrpl.NewTxScanner(xrpl.DefaultTxScannerConfig(), rpcClient)
	return &services{
		XRPLTxScanner: xrplTxScanner,
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
	)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		<-c
		cancel()
	}()

	log := logger.New(logger.ConfigureWithCLI(logger.ServiceDefaultConfig))
	log.Info(fmt.Sprintf("Build date: %s, version: %s", BuildDate, BuildVersion))
	ctx = logger.WithLogger(ctx, log)
	services := newServices(servicesConfig{
		XRPLRPCURL: mainnetXRPLRPCURL,
		XRPLWSURL:  mainnetXRPLWSURL,
	})
	rootCmd := RootCmd(services)
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Error("Error executing root cmd.", zap.Error(err))
		cancel()
		os.Exit(1) //nolint:gocritic // we cancel the context manually.
	}
}

// RootCmd returns the root cmd.
func RootCmd(services *services) *cobra.Command {
	cmd := &cobra.Command{
		Short: "XRPL relayer.",
	}

	cmd.AddCommand(StartCmd(services))

	return cmd
}

// StartCmd returns the start cmd.
func StartCmd(services *services) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start xrpl to coreum relayer.",
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.Get(cmd.Context())
			log.Info("Starting xrpl relayer.")

			ch := make(chan xrpl.Transaction)
			if err := services.XRPLTxScanner.Subscribe(
				cmd.Context(),
				mainnetXRPLCoreAccount,
				mainnetXRPLInitialBridgeLedgerIndex,
				30_000, // about a day
				ch,
			); err != nil {
				return err
			}

			for tx := range ch {
				log.Info("Received transaction from scanner.", zap.Any("transaction", tx))
			}

			return nil
		},
	}

	return cmd
}
