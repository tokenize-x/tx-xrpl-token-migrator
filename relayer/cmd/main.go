package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
	"github.com/CoreumFoundation/coreum-tools/pkg/run"
	coruemapp "github.com/CoreumFoundation/coreum/v4/app"
	"github.com/CoreumFoundation/coreum/v4/pkg/config"
	"github.com/CoreumFoundation/coreum/v4/pkg/config/constant"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/service"
)

// Build options.
var (
	BuildVersion = ""
)

const (
	flagXRPLRPCURL                    = "xrpl-rpc-url"
	flagXRPLHistoryScanStartLedger    = "xrpl-history-scan-start-ledger"
	flagXRPLRecentScanIndexesBack     = "xrpl-recent-scan-indexes-back"
	flagXRPLRecentScanSkipLastIndexes = "xrpl-recent-scan-skip-last-indexes"
	flagXRPLToken                     = "xrpl-token"
	flagXRPLMemoSuffix                = "xrpl-memo-suffix"

	flagCoreumChainID             = "coreum-chain-id"
	flagCoreumRPCURL              = "coreum-rpc-url"
	flagCoreumGRPCURL             = "coreum-grpc-url"
	flagCoreumSenderAddress       = "coreum-sender-address"
	flagCoreumContractAddress     = "coreum-contract-address"
	flagCoreumContractEvidenceIDs = "coreum-contract-evidence-ids"
	flagCoreumTrustedAddress      = "coreum-contract-trusted-addresses"

	flagCoreumContractTrustedAddresses = "coreum-contract-trusted-addresses"
	flagCoreumContractOwnerAddress     = "coreum-contract-owner-address"
	flagCoreumContractThreshold        = "coreum-contract-threshold"
	flagCoreumContractMinAmount        = "coreum-contract-min-amount"
	flagCoreumContractMaxAmount        = "coreum-contract-max-amount"

	flagPrometheusURL          = "prometheus-url"
	flagPrometheusInstanceName = "prometheus-instance-name"
	flagPrometheusUsername     = "prometheus-username"
	flagPrometheusPassword     = "prometheus-password"
)

const defaultHome = ".xrpl-bridge"

var (
	defaultTestnetCfg = service.Config{
		XRPLHistoryScanStartLedger:    20_000,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 20,
		XRPLTokens: []service.XRPLTokenConfig{
			{
				XRPLCurrency: "434F524500000000000000000000000000000000",
				XRPLIssuer:   "raSEP47QAwU6jsZU493znUD2iGNHDQEyvA",
			},
		},
		XRPLMemoSuffix: "/coreum-testnet-1/v1",

		CoreumChainID: string(constant.ChainIDTest),
	}

	defaultMainnnetCfg = service.Config{
		XRPLHistoryScanStartLedger:    81400000,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 20,
		XRPLTokens: []service.XRPLTokenConfig{
			{
				XRPLCurrency: "434F524500000000000000000000000000000000",
				XRPLIssuer:   "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
			},
			{
				XRPLCurrency: "58434F5245000000000000000000000000000000",
				XRPLIssuer:   "r3dVizzUAS3U29WKaaSALqkieytA2LCoRe",
			},
		},
		XRPLMemoSuffix: "/coreum-mainnet-1/v1",

		CoreumChainID: string(constant.ChainIDMain),
	}
)

func main() {
	run.Tool("relayer", func(ctx context.Context) error {
		log := logger.Get(ctx)
		rootCmd, err := RootCmd(ctx)
		if err != nil {
			return err
		}
		if err := rootCmd.Execute(); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("Error executing root cmd.", zap.Error(err))
			return err
		}

		return nil
	})
}

// RootCmd returns the root cmd.
func RootCmd(ctx context.Context) (*cobra.Command, error) {
	if err := preProcessFlags(); err != nil {
		return nil, err
	}

	encodingConfig := config.NewEncodingConfig(coruemapp.ModuleBasics)
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin)
	ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
	cmd := &cobra.Command{
		Short: "XRPL to coreum relayer.",
	}
	cmd.SetContext(ctx)

	cmd.AddCommand(VersionCmd(ctx))
	cmd.AddCommand(StartCmd(ctx))
	cmd.AddCommand(DeployAndInstantiateCmd(ctx))
	cmd.AddCommand(DeployCmd(ctx))
	cmd.AddCommand(GetContractConfigCmd(ctx))
	cmd.AddCommand(GetPendingUnapprovedTransactionsCmd(ctx))
	cmd.AddCommand(GetPendingApprovedTransactionsCmd(ctx))
	cmd.AddCommand(BuildExecutePendingApprovedTransactionsCmd(ctx))
	cmd.AddCommand(BuildMigrateContractTransactionCmd(ctx))
	cmd.AddCommand(BuildUpdateTrustedAddressesTransactionCmd(ctx))
	cmd.AddCommand(AuditCmd(ctx))

	cmd.AddCommand(keys.Commands(defaultHome))

	cmd.PersistentFlags().String(flagCoreumChainID, string(constant.ChainIDMain), "")

	return cmd, nil
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

	return cmd
}

// StartCmd returns the start cmd.
func StartCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start xrpl to coreum relayer.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			if cfg.PrometheusURL == "" {
				return errors.Errorf("flag %s is required", flagPrometheusURL)
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, true, logger.Get(ctx))
			if err != nil {
				return err
			}

			services.Logger.Info("Starting relayer.", zap.String("contract-address", cfg.CoreumContractAddress))
			services.CoreumMetricCollector.Start(ctx)
			services.MetricPusher.Start(ctx)

			return services.Executor.Start(ctx)
		},
	}

	addXRPLFlags(cmd)
	addCoreumFlags(cmd)
	addKeyringFlags(cmd)
	addPrometheusFlags(cmd)

	return cmd
}

// DeployAndInstantiateCmd returns the deploy and instantiate cmd.
func DeployAndInstantiateCmd(ctx context.Context) *cobra.Command { //nolint:funlen // long logic of flags reading
	cmd := &cobra.Command{
		Use:   "deploy-and-instantiate",
		Short: "Deploy and instantiate contract to coreum chain.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}

			trustedAddresses, err := cmd.Flags().GetStringSlice(flagCoreumContractTrustedAddresses)
			if err != nil {
				return err
			}

			if len(trustedAddresses) == 0 {
				return errors.New("at least one trusted address must be specified")
			}

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

			minAmountString, err := cmd.Flags().GetString(flagCoreumContractMinAmount)
			if err != nil {
				return err
			}
			minAmount, ok := sdk.NewIntFromString(minAmountString)
			if !ok || !minAmount.IsPositive() {
				return errors.Errorf("%s must be greater than zero", flagCoreumContractMinAmount)
			}

			maxAmountString, err := cmd.Flags().GetString(flagCoreumContractMaxAmount)
			if err != nil {
				return err
			}
			maxAmount, ok := sdk.NewIntFromString(maxAmountString)
			if !ok || maxAmount.LT(minAmount) {
				return errors.Errorf(
					"%s must be greater or equal than %s",
					flagCoreumContractMaxAmount, flagCoreumContractMinAmount,
				)
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}
			deployCfg := coreum.DeployAndInstantiateConfig{
				Owner:            ownerAddress,
				Admin:            ownerAddress,
				TrustedAddresses: trustedAddresses,
				Threshold:        threshold,
				MinAmount:        minAmount,
				MaxAmount:        maxAmount,
				Label:            "bank_threshold_send",
			}
			services.Logger.Info("Deploying contract.", zap.Any("config", deployCfg))

			senderAddress, err := sdk.AccAddressFromBech32(cfg.CoreumSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid sender address")
			}

			contractAddress, err := services.CoreumContractClient.DeployAndInstantiate(
				ctx,
				senderAddress,
				deployCfg,
			)
			if err != nil {
				return err
			}
			services.Logger.Info("Contract deployed", zap.String("address", contractAddress.String()))

			return nil
		},
	}

	addCoreumFlags(cmd)
	addKeyringFlags(cmd)

	cmd.PersistentFlags().StringSlice(flagCoreumContractTrustedAddresses, nil, "")
	cmd.PersistentFlags().String(flagCoreumContractOwnerAddress, "", "")
	cmd.PersistentFlags().Int(flagCoreumContractThreshold, 0, "")
	cmd.PersistentFlags().String(flagCoreumContractMinAmount, "", "")
	cmd.PersistentFlags().String(flagCoreumContractMaxAmount, "", "")

	return cmd
}

// DeployCmd returns the deployment cmd.
func DeployCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy contract to coreum chain.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}

			services.Logger.Info("Deploying contract.")

			senderAddress, err := sdk.AccAddressFromBech32(cfg.CoreumSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid sender address")
			}

			codeID, err := services.CoreumContractClient.Deploy(
				ctx,
				senderAddress,
			)
			if err != nil {
				return err
			}
			services.Logger.Info("Contract deployed", zap.Uint64("codeID", codeID))

			return nil
		},
	}

	addCoreumFlags(cmd)
	addKeyringFlags(cmd)

	return cmd
}

// GetContractConfigCmd prints contract config.
func GetContractConfigCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-contract-config",
		Short: "Print contract config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}
			contractCfg, err := services.CoreumContractClient.GetContractConfig(ctx)
			if err != nil {
				return err
			}

			services.Logger.Info("Contract config:", zap.Any("config", contractCfg))

			return nil
		},
	}

	addCoreumFlags(cmd)

	return cmd
}

// GetPendingUnapprovedTransactionsCmd prints pending unapproved transactions.
func GetPendingUnapprovedTransactionsCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-pending-unapproved-transactions",
		Short: "Print pending unapproved transactions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}
			unapprovedTransactions, _, err := services.CoreumContractClient.GetAllPendingTransactions(ctx)
			if err != nil {
				return err
			}
			services.Logger.Info("Unapproved pending transactions:",
				zap.Int("total", len(unapprovedTransactions)),
				zap.Any("txs", unapprovedTransactions),
			)

			return nil
		},
	}

	addCoreumFlags(cmd)

	return cmd
}

// GetPendingApprovedTransactionsCmd prints pending approved transactions.
func GetPendingApprovedTransactionsCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-pending-approved-transactions",
		Short: "Print pending approved transactions.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}
			_, approvedTransactions, err := services.CoreumContractClient.GetAllPendingTransactions(ctx)
			if err != nil {
				return err
			}
			evidenceIDs := lo.Map(approvedTransactions, func(tx coreum.PendingTransaction, _ int) string {
				return tx.EvidenceID
			})
			services.Logger.Info("Approved pending transactions:",
				zap.Int("total", len(evidenceIDs)),
				zap.Any("evidenceIDs", evidenceIDs),
			)

			return nil
		},
	}

	addCoreumFlags(cmd)

	return cmd
}

// BuildExecutePendingApprovedTransactionsCmd builds transaction for pending approved transactions execution.
func BuildExecutePendingApprovedTransactionsCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-execute-pending-approved-transaction",
		Short: "Build transaction for pending approved transactions execution.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}
			senderAddress, err := sdk.AccAddressFromBech32(cfg.CoreumSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid signer address")
			}

			evidenceIDs, err := cmd.Flags().GetStringSlice(flagCoreumContractEvidenceIDs)
			if err != nil {
				return err
			}

			msgs, err := services.CoreumContractClient.BuildExecutePendingMessages(ctx, senderAddress, evidenceIDs)
			if err != nil {
				return err
			}

			fees, gas, err := services.CoreumContractClient.EstimateExecuteMessages(ctx, senderAddress, msgs...)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.
				WithChainID(cfg.CoreumChainID).
				WithGenerateOnly(true)

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrapf(err, "failed to create tx factory")
			}

			txf = txf.WithFees(fees.String()).
				WithGas(gas)

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msgs...)
		},
	}

	addCoreumFlags(cmd)
	cmd.PersistentFlags().StringSlice(flagCoreumContractEvidenceIDs, nil, "")

	return cmd
}

// AuditCmd prints audit report.
func AuditCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Print audit report.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}

			discrepancies, err := services.Auditor.Audit(ctx)
			if err != nil {
				return err
			}

			if len(discrepancies) > 0 {
				for _, discrepancy := range discrepancies {
					fields := []zap.Field{
						zap.String("type", string(discrepancy.Type)),
						zap.String("description", discrepancy.Description),
					}
					if discrepancy.CoreumTx != nil {
						fields = append(fields, zap.String("coreumTxHash", discrepancy.CoreumTx.TxHash))
					}
					if discrepancy.XRPLTx.Hash != "" {
						fields = append(fields, zap.String("xrplTxHash", discrepancy.XRPLTx.Hash))
					}
					services.Logger.Info("Found discrepancy", fields...)
				}
				services.Logger.Warn("!!! The audit is failed !!!", zap.Int("discrepanciesCount", len(discrepancies)))

				return nil
			}

			services.Logger.Info("The audit is succeed. No discrepancies found.")

			return nil
		},
	}

	addCoreumFlags(cmd)
	addXRPLFlags(cmd)

	return cmd
}

// BuildMigrateContractTransactionCmd builds transaction for the contract migration.
func BuildMigrateContractTransactionCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-migrate-contract-transaction [codeID]",
		Short: "Builds transaction for the contract migration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}
			senderAddress, err := sdk.AccAddressFromBech32(cfg.CoreumSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid signer address")
			}

			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errors.Wrapf(err, "failed to parse codeID")
			}

			msg := services.CoreumContractClient.BuildMigrateContractMessage(senderAddress, codeID)

			fees, gas, err := services.CoreumContractClient.EstimateExecuteMessages(ctx, senderAddress, msg)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.
				WithChainID(cfg.CoreumChainID).
				WithGenerateOnly(true)

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrapf(err, "failed to create tx factory")
			}

			txf = txf.WithFees(fees.String()).
				WithGas(gas)

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	addCoreumFlags(cmd)

	return cmd
}

// BuildUpdateTrustedAddressesTransactionCmd builds transaction for the update_trusted_addresses contract method.
func BuildUpdateTrustedAddressesTransactionCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-update-trusted-addresses",
		Short: "Builds transaction for the update_trusted_addresses method",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			services, err := service.NewServices(cfg, clientCtx.Keyring, false, logger.Get(ctx))
			if err != nil {
				return err
			}
			senderAddress, err := sdk.AccAddressFromBech32(cfg.CoreumSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid signer address")
			}

			trustedAddressesStr, err := cmd.Flags().GetStringSlice(flagCoreumTrustedAddress)
			if err != nil {
				return err
			}

			trustedAddresses := make([]sdk.AccAddress, 0, len(trustedAddressesStr))
			for _, addrStr := range trustedAddressesStr {
				addr, err := sdk.AccAddressFromBech32(addrStr)
				if err != nil {
					return errors.Wrapf(err, "invalid address %s", addrStr)
				}
				trustedAddresses = append(trustedAddresses, addr)
			}

			msg, err := services.CoreumContractClient.BuildUpdateTrustedAddressesTransaction(senderAddress, trustedAddresses)
			if err != nil {
				return err
			}

			fees, gas, err := services.CoreumContractClient.EstimateExecuteMessages(ctx, senderAddress, msg)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.
				WithChainID(cfg.CoreumChainID).
				WithGenerateOnly(true)

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrapf(err, "failed to create tx factory")
			}

			txf = txf.WithFees(fees.String()).
				WithGas(gas)

			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	cmd.PersistentFlags().StringSlice(flagCoreumTrustedAddress, nil, "")
	addCoreumFlags(cmd)

	return cmd
}

func preProcessFlags() error {
	flagSet := pflag.NewFlagSet("pre-process", pflag.ExitOnError)
	flagSet.ParseErrorsWhitelist.UnknownFlags = true

	chainID := flagSet.String(flagCoreumChainID, string(constant.ChainIDMain), "")
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	if chainID == nil || *chainID == "" {
		return errors.Errorf("flag %s is required", flagCoreumChainID)
	}

	network, err := config.NetworkConfigByChainID(constant.ChainID(*chainID))
	if err != nil {
		return err
	}
	network.SetSDKConfig()

	return nil
}

func addKeyringFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(
		flags.FlagKeyringBackend,
		flags.DefaultKeyringBackend,
		"Select keyring's backend (os|file|kwallet|pass|test)",
	)
	cmd.PersistentFlags().String(flags.FlagHome, defaultHome, "The application home directory")
	cmd.PersistentFlags().String(
		flags.FlagKeyringDir,
		"",
		"The client Keyring directory; if omitted, the default 'home' directory will be used",
	)
}

func addCoreumFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagCoreumRPCURL, "", "")
	cmd.PersistentFlags().String(flagCoreumGRPCURL, "", "")
	cmd.PersistentFlags().String(flagCoreumSenderAddress, "", "")
	cmd.PersistentFlags().String(flagCoreumContractAddress, "", "")
}

func addXRPLFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagXRPLRPCURL, "", "")
	cmd.PersistentFlags().Int64(flagXRPLHistoryScanStartLedger, 0, "")
	cmd.PersistentFlags().Int64(flagXRPLRecentScanIndexesBack, 0, "")
	cmd.PersistentFlags().StringSlice(flagXRPLToken, []string{}, "isser/currency")
	cmd.PersistentFlags().String(flagXRPLMemoSuffix, "", "")
}

func addPrometheusFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagPrometheusURL, "", "Prometheus URL for metrics publishing")
	cmd.PersistentFlags().String(flagPrometheusInstanceName, "", "Instance name label for prometheus")
	cmd.PersistentFlags().String(flagPrometheusUsername, "", "Prometheus username for metrics publishing")
	cmd.PersistentFlags().String(flagPrometheusPassword, "", "Prometheus password for metrics publishing")
}

func validateAccAddress(address string) error {
	if _, err := sdk.AccAddressFromBech32(address); err != nil {
		return errors.Wrapf(err, "invalid account address:%s", address)
	}
	return nil
}

func readServicesConfig(cmd *cobra.Command) (service.Config, error) {
	chainID, err := cmd.Flags().GetString(flagCoreumChainID)
	if err != nil {
		return service.Config{}, err
	}
	var cfg service.Config
	switch constant.ChainID(chainID) {
	case constant.ChainIDDev:
		cfg = defaultTestnetCfg
		cfg.CoreumChainID = string(constant.ChainIDDev)
	case constant.ChainIDTest:
		cfg = defaultTestnetCfg
	case constant.ChainIDMain:
		cfg = defaultMainnnetCfg
	default:
		return service.Config{}, errors.Errorf("unspported chain id: %s", chainID)
	}

	setters := map[string]func(string) error{
		flagXRPLRPCURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.XRPLRPCURL)
		},
		flagXRPLHistoryScanStartLedger: func(flag string) error {
			return setStringInt64IfNotZero(cmd, flag, &cfg.XRPLHistoryScanStartLedger)
		},
		flagXRPLRecentScanIndexesBack: func(flag string) error {
			return setStringInt64IfNotZero(cmd, flag, &cfg.XRPLRecentScanIndexesBack)
		},
		flagXRPLRecentScanSkipLastIndexes: func(flag string) error {
			return setStringInt64IfNotZero(cmd, flag, &cfg.XRPLRecentScanSkipLastIndexes)
		},
		flagXRPLToken: func(flagName string) error {
			if cmd.Flags().Lookup(flagName) == nil {
				return nil
			}
			val, err := cmd.Flags().GetStringSlice(flagName)
			if err != nil {
				return err
			}
			if len(val) == 0 {
				return nil
			}
			xrplTokensCfg := make([]service.XRPLTokenConfig, 0, len(val))
			for _, v := range val {
				parts := strings.Split(v, "/")
				if len(parts) != 2 {
					return errors.Errorf("invalid %s value: %s, expected isser/currency", flagName, v)
				}
				xrplTokensCfg = append(xrplTokensCfg, service.XRPLTokenConfig{
					XRPLIssuer:   parts[0],
					XRPLCurrency: parts[1],
				})
			}
			cfg.XRPLTokens = xrplTokensCfg
			return nil
		},
		flagXRPLMemoSuffix: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.XRPLMemoSuffix)
		},
		flagCoreumRPCURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.CoreumRPCURL)
		},
		flagCoreumGRPCURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.CoreumGRPCURL)
		},
		flagCoreumSenderAddress: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.CoreumSenderAddress)
		},
		flagCoreumContractAddress: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.CoreumContractAddress)
		},

		flagPrometheusURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.PrometheusURL)
		},
		flagPrometheusInstanceName: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.PrometheusInstanceName)
		},
		flagPrometheusUsername: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.PrometheusUsername)
		},
		flagPrometheusPassword: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.PrometheusPassword)
		},
	}

	for flagName, setter := range setters {
		if err := setter(flagName); err != nil {
			return service.Config{}, err
		}
	}

	return cfg, nil
}

func setStringIfNotEmpty(cmd *cobra.Command, flagName string, v *string) error {
	if cmd.Flags().Lookup(flagName) == nil {
		return nil
	}
	val, err := cmd.Flags().GetString(flagName)
	if err != nil {
		return err
	}
	if val == "" {
		return nil
	}
	*v = val
	return nil
}

func setStringInt64IfNotZero(cmd *cobra.Command, flagName string, v *int64) error {
	if cmd.Flags().Lookup(flagName) == nil {
		return nil
	}
	val, err := cmd.Flags().GetInt64(flagName)
	if err != nil {
		return err
	}
	if val == 0 {
		return nil
	}
	*v = val
	return nil
}
