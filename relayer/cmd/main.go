package main

import (
	"context"
	"fmt"
	"strings"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
	"github.com/CoreumFoundation/coreum-tools/pkg/run"
	"github.com/CoreumFoundation/coreum/pkg/config/constant"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/service"
)

// Build options.
var (
	BuildVersion = ""
)

const (
	flagXRPLRPCURL                 = "xrpl-rpc-url"
	flagXRPLHistoryScanStartLedger = "xrpl-history-scan-start-ledger"
	flagXRPLRecentScanIndexesBack  = "xrpl-recent-scan-indexes-back"
	flagXRPLAccount                = "xrpl-account"
	flagXRPLCurrency               = "xrpl-currency"
	flagXRPLIssuer                 = "xrpl-issuer"
	flagXRPLMemoSuffix             = "xrpl-memo-suffix"

	flagCoreumChainID         = "coreum-chain-id"
	flagCoreumGRPCURL         = "coreum-grpc-url"
	flagCoreumMnemonic        = "coreum-mnemonic"
	flagCoreumContractAddress = "coreum-contract-address"

	flagCoreumContractTrustedAddresses = "coreum-contract-trusted-addresses"
	flagCoreumContractOwnerAddress     = "coreum-contract-owner-address"
	flagCoreumContractThreshold        = "coreum-contract-threshold"
)

// temporary constants.
var (
	defaultTestnetCfg = service.Config{
		XRPLRPCURL:                 "https://s.altnet.rippletest.net:51234/",
		XRPLHistoryScanStartLedger: 38500000,
		XRPLRecentScanIndexesBack:  30_000,
		XRPLAccount:                "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA",
		XRPLCurrency:               "434F524500000000000000000000000000000000",
		XRPLIssuer:                 "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA",
		XRPLMemoSuffix:             "/coreum-testnet-1",

		CoreumChainID: string(constant.ChainIDTest),
		CoreumGRPCURL: "https://full-node.testnet-1.coreum.dev:9090",

		CoreumContractAddress: "testcore1wt8hmu6yzrdaq030cp7pxa6asdrtc7ltvlzpat99dgust8g6w73qkqy8s5",
	}

	defaultMainnnetCfg = service.Config{
		XRPLRPCURL:                 "https://s1.ripple.com:51234/",
		XRPLHistoryScanStartLedger: 80590000,
		XRPLRecentScanIndexesBack:  30_000,
		XRPLAccount:                "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
		XRPLCurrency:               "434F524500000000000000000000000000000000",
		XRPLIssuer:                 "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
		XRPLMemoSuffix:             "/coreum-mainnet-1",

		CoreumGRPCURL: "https://full-node.mainnet-1.coreum.dev:9090",
		CoreumChainID: string(constant.ChainIDMain),
	}
)

func main() {
	run.Tool("bridge", func(ctx context.Context) error {
		log := logger.Get(ctx)
		log.Info(fmt.Sprintf("Build version: %s", BuildVersion))
		rootCmd := RootCmd(ctx)
		if err := rootCmd.Execute(); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("Error executing root cmd.", zap.Error(err))
			return err
		}

		return nil
	})
}

// RootCmd returns the root cmd.
func RootCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Short: "XRPL relayer.",
	}

	cmd.AddCommand(VersionCmd(ctx))
	cmd.AddCommand(StartCmd(ctx))
	cmd.AddCommand(DeployCmd(ctx))

	return cmd
}

// VersionCmd returns the version cmd.
func VersionCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the relayer version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Get(ctx).Info(fmt.Sprintf("version:%s", BuildVersion))
			return nil
		},
	}

	addDefaultFlags(cmd)

	return cmd
}

// StartCmd returns the start cmd.
func StartCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start xrpl to coreum relayer.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readDefaultConfig(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, logger.Get(ctx), true)
			if err != nil {
				return err
			}
			if err := validateAccAddress(cfg.CoreumContractAddress); err != nil {
				return errors.Wrapf(err, "invalid contract address")
			}

			services.Logger.Info("Starting relayer.", zap.String("contract-address", cfg.CoreumContractAddress))
			services.CoreumMetricCollector.Start(ctx)
			services.MetricServer.Start(ctx)

			return services.Executor.Start(ctx)
		},
	}

	addDefaultFlags(cmd)

	return cmd
}

// DeployCmd returns the deployment cmd.
func DeployCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy contract to coreum chain.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readDefaultConfig(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, logger.Get(ctx), true)
			if err != nil {
				return err
			}

			trustedAddressesString, err := cmd.Flags().GetString(flagCoreumContractTrustedAddresses)
			if err != nil {
				return err
			}
			if len(trustedAddressesString) == 0 {
				return errors.New("at least one trusted address must be specified")
			}

			trustedAddresses := strings.Split(trustedAddressesString, ",")
			for _, address := range trustedAddresses {
				if err := validateAccAddress(address); err != nil {
					return err
				}
			}

			ownerAddress, err := cmd.Flags().GetString(flagCoreumContractOwnerAddress)
			if err != nil {
				return err
			}
			if err := validateAccAddress(ownerAddress); err != nil {
				return err
			}

			threshold, err := cmd.Flags().GetInt(flagCoreumContractThreshold)
			if err != nil {
				return err
			}
			if threshold <= 0 {
				return errors.New("threshold must be greater than zero")
			}

			deployCfg := coreum.DeployAndInstantiateConfig{
				Owner:            ownerAddress,
				Admin:            ownerAddress,
				TrustedAddresses: trustedAddresses,
				Threshold:        threshold,
				AccessType:       wasmtypes.AccessTypeUnspecified,
				Label:            "bank_threshold_send",
			}
			services.Logger.Info("Deploying contract.", zap.Any("config", deployCfg))

			contractAddress, err := services.CoreumContractClient.DeployAndInstantiate(
				ctx,
				services.CoreumSenderAddress,
				deployCfg,
			)
			if err != nil {
				return err
			}
			services.Logger.Info("Contract deployed", zap.String("address", contractAddress))

			return nil
		},
	}

	addDefaultFlags(cmd)

	cmd.PersistentFlags().StringArray(flagCoreumContractTrustedAddresses, nil, "")
	cmd.PersistentFlags().String(flagCoreumContractOwnerAddress, "", "")
	cmd.PersistentFlags().Int(flagCoreumContractThreshold, 0, "")

	return cmd
}

func validateAccAddress(address string) error {
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return errors.Wrapf(err, "invalid account address:%s", address)
	}
	return nil
}

func addDefaultFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagXRPLRPCURL, "", "")
	cmd.PersistentFlags().Int64(flagXRPLHistoryScanStartLedger, 0, "")
	cmd.PersistentFlags().Int64(flagXRPLRecentScanIndexesBack, 0, "")
	cmd.PersistentFlags().String(flagXRPLAccount, "", "")
	cmd.PersistentFlags().String(flagXRPLCurrency, "", "")
	cmd.PersistentFlags().String(flagXRPLIssuer, "", "")
	cmd.PersistentFlags().String(flagXRPLMemoSuffix, "", "")

	cmd.PersistentFlags().String(flagCoreumChainID, "", "")
	cmd.PersistentFlags().String(flagCoreumGRPCURL, "", "")
	cmd.PersistentFlags().String(flagCoreumMnemonic, "", "")
	cmd.PersistentFlags().String(flagCoreumContractAddress, "", "")
}

func readDefaultConfig(cmd *cobra.Command) (service.Config, error) {
	chainID, err := cmd.Flags().GetString(flagCoreumChainID)
	if err != nil {
		return service.Config{}, err
	}
	var config service.Config
	switch constant.ChainID(chainID) {
	case constant.ChainIDTest:
		config = defaultTestnetCfg
	case constant.ChainIDMain:
		config = defaultMainnnetCfg
	default:
		return service.Config{}, errors.Errorf("unspported chain id: %s", chainID)
	}

	setters := map[string]func(string) error{
		flagXRPLRPCURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.XRPLRPCURL = v
			})
		},

		flagXRPLHistoryScanStartLedger: func(flag string) error {
			return setStringInt64IfNotZero(cmd, flag, func(v int64) {
				config.XRPLHistoryScanStartLedger = v
			})
		},
		flagXRPLRecentScanIndexesBack: func(flag string) error {
			return setStringInt64IfNotZero(cmd, flag, func(v int64) {
				config.XRPLRecentScanIndexesBack = v
			})
		},
		flagXRPLAccount: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.XRPLAccount = v
			})
		},
		flagXRPLCurrency: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.XRPLCurrency = v
			})
		},
		flagXRPLIssuer: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.XRPLIssuer = v
			})
		},
		flagXRPLMemoSuffix: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.XRPLMemoSuffix = v
			})
		},

		flagCoreumGRPCURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.CoreumGRPCURL = v
			})
		},
		flagCoreumMnemonic: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.CoreumMnemonic = v
			})
		},
		flagCoreumContractAddress: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, func(v string) {
				config.CoreumContractAddress = v
			})
		},
	}

	for flagName, setter := range setters {
		if err := setter(flagName); err != nil {
			return service.Config{}, err
		}
	}

	return config, nil
}

func setStringIfNotEmpty(cmd *cobra.Command, flagName string, setter func(v string)) error {
	val, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return err
	}
	if val == "" {
		return nil
	}
	setter(val)
	return nil
}

func setStringInt64IfNotZero(cmd *cobra.Command, flagName string, setter func(v int64)) error {
	val, err := cmd.Flags().GetInt64(flagName)
	if err != nil {
		return err
	}
	if val == 0 {
		return nil
	}
	setter(val)
	return nil
}
