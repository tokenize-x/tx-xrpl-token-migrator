package main

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	txclient "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bnb"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/finder"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/service"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/logger"
	"github.com/CoreumFoundation/coreum-tools/pkg/run"
	"github.com/CoreumFoundation/coreum/v5/pkg/config"
	"github.com/CoreumFoundation/coreum/v5/pkg/config/constant"
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

	flagBNBRPCURL        = "bnb-rpc-url"
	flagBNBBridgeAddress = "bnb-bridge-address"
	flagBNBStartBlock    = "bnb-start-block"
	flagBNBChainID       = "bnb-chain-id"
	flagBNBPollInterval  = "bnb-poll-interval"
	flagBNBConfirmations = "bnb-confirmations"

	flagTXChainID             = "tx-chain-id"
	flagTXRPCURL              = "tx-rpc-url"
	flagTXGRPCURL             = "tx-grpc-url"
	flagTXSenderAddress       = "tx-sender-address"
	flagTXContractAddress     = "tx-contract-address"
	flagTXContractEvidenceIDs = "tx-contract-evidence-ids"
	flagTXTrustedAddress      = "tx-contract-trusted-addresses"

	flagTXContractTrustedAddresses = "tx-contract-trusted-addresses"
	flagTXContractOwnerAddress     = "tx-contract-owner-address"
	flagTXContractThreshold        = "tx-contract-threshold"
	flagTXContractMinAmount        = "tx-contract-min-amount"
	flagTXContractMaxAmount        = "tx-contract-max-amount"

	flagPrometheusURL          = "prometheus-url"
	flagPrometheusInstanceName = "prometheus-instance-name"
	flagPrometheusUsername     = "prometheus-username"
	flagPrometheusPassword     = "prometheus-password"

	flagAuditStartDate = "audit-start-date"
)

const defaultHome = ".tx-xrpl-token-migrator"

var (
	defaultTestnetCfg = service.Config{
		XRPLHistoryScanStartLedger:    20_000,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 20,

		XRPLMemoSuffix: "/coreum-testnet-1/v1",

		TXChainID: string(constant.ChainIDTest),

		AuditStartDate: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
	}

	defaultMainnnetCfg = service.Config{
		XRPLHistoryScanStartLedger:    81400000,
		XRPLRecentScanIndexesBack:     30_000,
		XRPLRecentScanSkipLastIndexes: 20,

		XRPLMemoSuffix: "/coreum-mainnet-1/v1",

		TXChainID: string(constant.ChainIDMain),

		AuditStartDate: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
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

	encodingConfig := config.NewEncodingConfig(auth.AppModuleBasic{}, wasm.AppModuleBasic{})
	clientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin)
	ctx = context.WithValue(ctx, client.ClientContextKey, &clientCtx)
	cmd := &cobra.Command{
		Short: "XRPL to TX relayer.",
	}
	cmd.SetContext(ctx)

	cmd.AddCommand(VersionCmd(ctx))
	cmd.AddCommand(TestBNBCmd(ctx))
	cmd.AddCommand(StartCmd(ctx))
	cmd.AddCommand(DeployAndInstantiateCmd(ctx))
	cmd.AddCommand(DeployCmd(ctx))
	cmd.AddCommand(GetContractConfigCmd(ctx))
	cmd.AddCommand(GetPendingUnapprovedTransactionsCmd(ctx))
	cmd.AddCommand(GetPendingApprovedTransactionsCmd(ctx))
	cmd.AddCommand(BuildExecutePendingApprovedTransactionsCmd(ctx))
	cmd.AddCommand(BuildMigrateContractTransactionCmd(ctx))
	cmd.AddCommand(BuildUpdateTrustedAddressesTransactionCmd(ctx))
	cmd.AddCommand(BuildAddXRPLTokensTransactionCmd(ctx))
	cmd.AddCommand(AuditCmd(ctx))

	cmd.AddCommand(keys.Commands())

	cmd.PersistentFlags().String(flagTXChainID, string(constant.ChainIDMain), "")

	return cmd, nil
}

// VersionCmd returns the version cmd.
func VersionCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the relayer version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Get(ctx).Info("version:" + BuildVersion)
			return nil
		},
	}

	return cmd
}

// TestBNBCmd returns a test command for BNB scanner/finder only.
// This is for development/debugging - it doesn't require prometheus or TX chain.
func TestBNBCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test-bnb",
		Short: "Test BNB scanner and finder (development only).",
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.Get(ctx)

			rpcURL, _ := cmd.Flags().GetString(flagBNBRPCURL)
			if rpcURL == "" {
				return errors.Errorf("flag %s is required", flagBNBRPCURL)
			}

			bridgeAddrStr, _ := cmd.Flags().GetString(flagBNBBridgeAddress)
			if bridgeAddrStr == "" {
				return errors.Errorf("flag %s is required", flagBNBBridgeAddress)
			}
			if !common.IsHexAddress(bridgeAddrStr) {
				return errors.Errorf("invalid bridge address: %s", bridgeAddrStr)
			}

			startBlock, _ := cmd.Flags().GetUint64(flagBNBStartBlock)
			chainID, _ := cmd.Flags().GetString(flagBNBChainID)
			if chainID == "" {
				return errors.Errorf("flag %s is required", flagBNBChainID)
			}
			pollInterval, _ := cmd.Flags().GetDuration(flagBNBPollInterval)
			confirmations, _ := cmd.Flags().GetUint64(flagBNBConfirmations)

			txDenom, _ := cmd.Flags().GetString("tx-denom")
			if txDenom == "" {
				txDenom = "ucore" // default
			}
			txDecimals, _ := cmd.Flags().GetInt("tx-decimals")
			if txDecimals == 0 {
				txDecimals = 6 // default
			}

			log.Info("Starting BNB scanner test",
				zap.String("rpcURL", rpcURL),
				zap.String("bridgeAddress", bridgeAddrStr),
				zap.Uint64("startBlock", startBlock),
				zap.String("chainID", chainID),
				zap.Duration("pollInterval", pollInterval),
				zap.Uint64("confirmations", confirmations),
			)

			// Create scanner
			scannerCfg := bnb.ScannerConfig{
				RPCURL:        rpcURL,
				BridgeAddress: common.HexToAddress(bridgeAddrStr),
				StartBlock:    startBlock,
				PollInterval:  pollInterval,
				Confirmations: confirmations,
				ChainID:       chainID,
			}

			scanner, err := bnb.NewScanner(scannerCfg, log)
			if err != nil {
				return errors.Wrap(err, "failed to create BNB scanner")
			}

			// Create finder
			finderCfg := finder.BNBFinderConfig{
				ChainID:    chainID,
				TXDenom:    txDenom,
				TXDecimals: txDecimals,
			}
			bnbFinder := finder.NewBNBFinder(finderCfg, log, scanner)

			// Subscribe and log events
			pendingTxCh := make(chan finder.PendingTXSendTransaction)
			if err := bnbFinder.SubscribeTXSendTransactions(ctx, pendingTxCh); err != nil {
				return errors.Wrap(err, "failed to subscribe to BNB events")
			}

			log.Info("BNB scanner started, waiting for events...")

			for {
				select {
				case <-ctx.Done():
					log.Info("Context cancelled, stopping test")
					return nil
				case pendingTx := <-pendingTxCh:
					log.Info("Received PendingTXSendTransaction",
						zap.String("destination", pendingTx.TXDestination.String()),
						zap.String("amount", pendingTx.TXAmount.String()),
						zap.String("txHash", pendingTx.XRPLTxHash),
					)
				}
			}
		},
	}

	addBNBFlags(cmd)
	cmd.PersistentFlags().String("tx-denom", "ucore", "TX chain denom for converted amounts")
	cmd.PersistentFlags().Int("tx-decimals", 6, "TX chain decimals")

	return cmd
}

// StartCmd returns the start cmd.
func StartCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start xrpl to TX relayer.",
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

			// Run the executor with auto-restart on config changes
			return service.RunExecutorWithAutoRestart(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
		},
	}

	addXRPLFlags(cmd)
	addBNBFlags(cmd)
	addTXFlags(cmd)
	addKeyringFlags(cmd)
	addPrometheusFlags(cmd)

	return cmd
}

// DeployAndInstantiateCmd returns the deploy and instantiate cmd.
func DeployAndInstantiateCmd(ctx context.Context) *cobra.Command { //nolint:funlen // long logic of flags reading
	cmd := &cobra.Command{
		Use:   "deploy-and-instantiate",
		Short: "Deploy and instantiate contract to TX chain.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}

			trustedAddresses, err := cmd.Flags().GetStringSlice(flagTXContractTrustedAddresses)
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

			ownerAddress, err := cmd.Flags().GetString(flagTXContractOwnerAddress)
			if err != nil {
				return err
			}
			if err := validateAccAddress(ownerAddress); err != nil {
				return err
			}

			threshold, err := cmd.Flags().GetInt(flagTXContractThreshold)
			if err != nil {
				return err
			}
			if threshold <= 0 {
				return errors.New("threshold must be greater than zero")
			}

			minAmountString, err := cmd.Flags().GetString(flagTXContractMinAmount)
			if err != nil {
				return err
			}
			minAmount, ok := sdkmath.NewIntFromString(minAmountString)
			if !ok || !minAmount.IsPositive() {
				return errors.Errorf("%s must be greater than zero", flagTXContractMinAmount)
			}

			maxAmountString, err := cmd.Flags().GetString(flagTXContractMaxAmount)
			if err != nil {
				return err
			}
			maxAmount, ok := sdkmath.NewIntFromString(maxAmountString)
			if !ok || maxAmount.LT(minAmount) {
				return errors.Errorf(
					"%s must be greater or equal than %s",
					flagTXContractMaxAmount, flagTXContractMinAmount,
				)
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			deployClient, log, err := service.NewDeployContractClient(cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}

			deployCfg := tx.DeployAndInstantiateConfig{
				Owner:            ownerAddress,
				Admin:            ownerAddress,
				TrustedAddresses: trustedAddresses,
				Threshold:        uint32(threshold),
				MinAmount:        minAmount,
				MaxAmount:        maxAmount,
				Label:            "bank_threshold_send",
			}
			log.Info("Deploying contract.", zap.Any("config", deployCfg))

			senderAddress, err := sdk.AccAddressFromBech32(cfg.TXSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid sender address")
			}

			contractAddress, err := deployClient.DeployAndInstantiate(
				ctx,
				senderAddress,
				deployCfg,
			)
			if err != nil {
				return err
			}
			log.Info("Contract deployed", zap.String("address", contractAddress.String()))

			return nil
		},
	}

	addTXFlags(cmd)
	addKeyringFlags(cmd)

	cmd.PersistentFlags().StringSlice(flagTXContractTrustedAddresses, nil, "")
	cmd.PersistentFlags().String(flagTXContractOwnerAddress, "", "")
	cmd.PersistentFlags().Int(flagTXContractThreshold, 0, "")
	cmd.PersistentFlags().String(flagTXContractMinAmount, "", "")
	cmd.PersistentFlags().String(flagTXContractMaxAmount, "", "")

	return cmd
}

// DeployCmd returns the deployment cmd.
func DeployCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy contract to TX blockchain.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}

			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			deployClient, log, err := service.NewDeployContractClient(cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}

			log.Info("Deploying contract.")

			senderAddress, err := sdk.AccAddressFromBech32(cfg.TXSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid sender address")
			}

			codeID, err := deployClient.Deploy(
				ctx,
				senderAddress,
			)
			if err != nil {
				return err
			}
			log.Info("Contract deployed", zap.Uint64("codeID", codeID))

			return nil
		},
	}

	addTXFlags(cmd)
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
			contractClient, log, err := service.NewContractClient(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}
			contractCfg, err := contractClient.GetContractConfig(ctx)
			if err != nil {
				return err
			}

			log.Info("Contract config:", zap.Any("config", contractCfg))

			return nil
		},
	}

	addTXFlags(cmd)

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
			contractClient, log, err := service.NewContractClient(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}
			unapprovedTransactions, _, err := contractClient.GetAllPendingTransactions(ctx)
			if err != nil {
				return err
			}
			log.Info("Unapproved pending transactions:",
				zap.Int("total", len(unapprovedTransactions)),
				zap.Any("txs", unapprovedTransactions),
			)

			return nil
		},
	}

	addTXFlags(cmd)

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
			contractClient, log, err := service.NewContractClient(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}
			_, approvedTransactions, err := contractClient.GetAllPendingTransactions(ctx)
			if err != nil {
				return err
			}
			evidenceIDs := lo.Map(approvedTransactions, func(txn tx.PendingTransaction, _ int) string {
				return txn.EvidenceID
			})
			log.Info("Approved pending transactions:",
				zap.Int("total", len(evidenceIDs)),
				zap.Any("evidenceIDs", evidenceIDs),
			)

			return nil
		},
	}

	addTXFlags(cmd)

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
			contractClient, _, err := service.NewContractClient(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}
			senderAddress, err := sdk.AccAddressFromBech32(cfg.TXSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid signer address")
			}

			evidenceIDs, err := cmd.Flags().GetStringSlice(flagTXContractEvidenceIDs)
			if err != nil {
				return err
			}

			msgs, err := contractClient.BuildExecutePendingMessages(ctx, senderAddress, evidenceIDs)
			if err != nil {
				return err
			}

			fees, gas, err := contractClient.EstimateExecuteMessages(ctx, senderAddress, msgs...)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.
				WithChainID(cfg.TXChainID).
				WithGenerateOnly(true)

			txf, err := txclient.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrapf(err, "failed to create tx factory")
			}

			txf = txf.WithFees(fees.String()).
				WithGas(gas)

			return txclient.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msgs...)
		},
	}

	addTXFlags(cmd)
	cmd.PersistentFlags().StringSlice(flagTXContractEvidenceIDs, nil, "")

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
			auditor, log, err := service.NewAuditor(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}

			discrepancies, err := auditor.Audit(ctx)
			if err != nil {
				return err
			}

			if len(discrepancies) > 0 {
				for _, discrepancy := range discrepancies {
					fields := []zap.Field{
						zap.String("type", string(discrepancy.Type)),
						zap.String("description", discrepancy.Description),
					}
					if discrepancy.TXTx != nil {
						fields = append(fields, zap.String("txTxHash", discrepancy.TXTx.TxHash))
					}
					if discrepancy.XRPLTx.Hash != "" {
						fields = append(fields, zap.String("xrplTxHash", discrepancy.XRPLTx.Hash))
					}
					log.Info("Found discrepancy", fields...)
				}
				log.Warn("!!! The audit is failed !!!", zap.Int("discrepanciesCount", len(discrepancies)))

				return nil
			}

			log.Info("The audit is succeed. No discrepancies found.")

			return nil
		},
	}

	addTXFlags(cmd)
	addXRPLFlags(cmd)
	cmd.PersistentFlags().String(flagAuditStartDate, "", "Audit stat date, e.g. 2006-01-02")

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
			contractClient, _, err := service.NewContractClient(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}
			senderAddress, err := sdk.AccAddressFromBech32(cfg.TXSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid signer address")
			}

			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errors.Wrapf(err, "failed to parse codeID")
			}

			msg := contractClient.BuildMigrateContractMessage(senderAddress, codeID)

			fees, gas, err := contractClient.EstimateExecuteMessages(ctx, senderAddress, msg)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.
				WithChainID(cfg.TXChainID).
				WithGenerateOnly(true)

			txf, err := txclient.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrapf(err, "failed to create tx factory")
			}

			txf = txf.WithFees(fees.String()).
				WithGas(gas)

			return txclient.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	addTXFlags(cmd)

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
			contractClient, _, err := service.NewContractClient(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}
			senderAddress, err := sdk.AccAddressFromBech32(cfg.TXSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid signer address")
			}

			trustedAddressesStr, err := cmd.Flags().GetStringSlice(flagTXTrustedAddress)
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

			msg, err := contractClient.BuildUpdateTrustedAddressesTransaction(senderAddress, trustedAddresses)
			if err != nil {
				return err
			}

			fees, gas, err := contractClient.EstimateExecuteMessages(ctx, senderAddress, msg)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.
				WithChainID(cfg.TXChainID).
				WithGenerateOnly(true)

			txf, err := txclient.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrapf(err, "failed to create tx factory")
			}

			txf = txf.WithFees(fees.String()).
				WithGas(gas)

			return txclient.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	cmd.PersistentFlags().StringSlice(flagTXTrustedAddress, nil, "")
	addTXFlags(cmd)

	return cmd
}

// BuildAddXRPLTokensTransactionCmd builds transaction for the add_xrpl_tokens contract method.
func BuildAddXRPLTokensTransactionCmd(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-add-xrpl-tokens",
		Short: "Builds transaction for the add_xrpl_tokens method",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readServicesConfig(cmd)
			if err != nil {
				return err
			}
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			contractClient, _, err := service.NewContractClient(ctx, cfg, clientCtx.Keyring, logger.Get(ctx))
			if err != nil {
				return err
			}
			senderAddress, err := sdk.AccAddressFromBech32(cfg.TXSenderAddress)
			if err != nil {
				return errors.Wrapf(err, "invalid signer address")
			}

			xrplTokensStr, err := cmd.Flags().GetStringSlice(flagXRPLToken)
			if err != nil {
				return err
			}

			xrplTokens := make([]tx.XRPLToken, 0, len(xrplTokensStr))
			for _, tokenStr := range xrplTokensStr {
				parts := strings.Split(tokenStr, "/")
				if len(parts) != 4 {
					return errors.Errorf(
						"invalid %s value: %s, expected format: issuer/currency/activation_unix_timestamp/multiplier",
						flagXRPLToken,
						tokenStr,
					)
				}
				activationUnixTimestamp, err := strconv.ParseUint(parts[2], 10, 64)
				if err != nil {
					return errors.Wrapf(err, "failed to parse activation_unix_timestamp: %s", parts[2])
				}
				xrplTokens = append(xrplTokens, tx.XRPLToken{
					Issuer:         parts[0],
					Currency:       parts[1],
					ActivationDate: activationUnixTimestamp,
					Multiplier:     parts[3],
				})
			}

			msg, err := contractClient.BuildAddXRPLTokensTransaction(senderAddress, xrplTokens)
			if err != nil {
				return err
			}

			fees, gas, err := contractClient.EstimateExecuteMessages(ctx, senderAddress, msg)
			if err != nil {
				return err
			}

			clientCtx = clientCtx.
				WithChainID(cfg.TXChainID).
				WithGenerateOnly(true)

			txf, err := txclient.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return errors.Wrapf(err, "failed to create tx factory")
			}

			txf = txf.WithFees(fees.String()).
				WithGas(gas)

			return txclient.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	cmd.PersistentFlags().StringSlice(
		flagXRPLToken,
		nil,
		"XRPL tokens in format: issuer/currency/activation_unix_timestamp/multiplier",
	)
	addTXFlags(cmd)

	return cmd
}

func preProcessFlags() error {
	flagSet := pflag.NewFlagSet("pre-process", pflag.ExitOnError)
	flagSet.ParseErrorsWhitelist.UnknownFlags = true

	chainID := flagSet.String(flagTXChainID, string(constant.ChainIDMain), "")
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	if chainID == nil || *chainID == "" {
		return errors.Errorf("flag %s is required", flagTXChainID)
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

func addTXFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagTXRPCURL, "", "")
	cmd.PersistentFlags().String(flagTXGRPCURL, "", "")
	cmd.PersistentFlags().String(flagTXSenderAddress, "", "")
	cmd.PersistentFlags().String(flagTXContractAddress, "", "")
}

func addXRPLFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagXRPLRPCURL, "", "")
	cmd.PersistentFlags().Int64(flagXRPLHistoryScanStartLedger, 0, "")
	cmd.PersistentFlags().Int64(flagXRPLRecentScanIndexesBack, 0, "")
	cmd.PersistentFlags().String(flagXRPLMemoSuffix, "", "")
}

func addBNBFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String(flagBNBRPCURL, "", "BNB/EVM RPC URL for bridge event scanning")
	cmd.PersistentFlags().String(flagBNBBridgeAddress, "", "BNB bridge contract address")
	cmd.PersistentFlags().Uint64(flagBNBStartBlock, 0, "BNB block number to start scanning from")
	cmd.PersistentFlags().String(flagBNBChainID, "", "ChainID suffix to strip from destinationPayload (e.g., /coreum-testnet-1/v1)")
	cmd.PersistentFlags().Duration(flagBNBPollInterval, 3*time.Second, "BNB block polling interval (e.g., 3s, 5s)")
	cmd.PersistentFlags().Uint64(flagBNBConfirmations, 5, "BNB block confirmations before processing (reorg protection)")
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
	chainID, err := cmd.Flags().GetString(flagTXChainID)
	if err != nil {
		return service.Config{}, err
	}
	var cfg service.Config
	switch constant.ChainID(chainID) {
	case constant.ChainIDDev:
		cfg = defaultTestnetCfg
		cfg.TXChainID = string(constant.ChainIDDev)
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
		flagXRPLMemoSuffix: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.XRPLMemoSuffix)
		},
		flagTXRPCURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.TXRPCURL)
		},
		flagTXGRPCURL: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.TXGRPCURL)
		},
		flagTXSenderAddress: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.TXSenderAddress)
		},
		flagTXContractAddress: func(flag string) error {
			return setStringIfNotEmpty(cmd, flag, &cfg.TXContractAddress)
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
		flagAuditStartDate: func(flag string) error {
			return setDateIfNotEmpty(flag, cmd, &cfg.AuditStartDate)
		},
	}

	for flagName, setter := range setters {
		if err := setter(flagName); err != nil {
			return service.Config{}, err
		}
	}

	// Read BNB config (handled separately due to address parsing)
	if err := readBNBConfig(cmd, &cfg.BNBScanner); err != nil {
		return service.Config{}, err
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

func setDateIfNotEmpty(flag string, cmd *cobra.Command, v *time.Time) error {
	var dateStr string
	if err := setStringIfNotEmpty(cmd, flag, &dateStr); err != nil {
		return err
	}
	if dateStr == "" {
		return nil
	}
	val, err := time.Parse(time.DateOnly, dateStr)
	if err != nil {
		return err
	}
	*v = val

	return nil
}

func readBNBConfig(cmd *cobra.Command, cfg *bnb.ScannerConfig) error {
	if cmd.Flags().Lookup(flagBNBRPCURL) == nil {
		return nil
	}

	rpcURL, err := cmd.Flags().GetString(flagBNBRPCURL)
	if err != nil {
		return err
	}
	if rpcURL == "" {
		return nil // BNB not configured
	}

	bridgeAddrStr, err := cmd.Flags().GetString(flagBNBBridgeAddress)
	if err != nil {
		return err
	}
	if bridgeAddrStr == "" {
		return errors.Errorf("flag %s is required when %s is set", flagBNBBridgeAddress, flagBNBRPCURL)
	}
	if !common.IsHexAddress(bridgeAddrStr) {
		return errors.Errorf("invalid bridge address: %s", bridgeAddrStr)
	}

	startBlock, err := cmd.Flags().GetUint64(flagBNBStartBlock)
	if err != nil {
		return err
	}

	chainID, err := cmd.Flags().GetString(flagBNBChainID)
	if err != nil {
		return err
	}
	if chainID == "" {
		return errors.Errorf("flag %s is required when %s is set", flagBNBChainID, flagBNBRPCURL)
	}

	pollInterval, err := cmd.Flags().GetDuration(flagBNBPollInterval)
	if err != nil {
		return err
	}

	confirmations, err := cmd.Flags().GetUint64(flagBNBConfirmations)
	if err != nil {
		return err
	}

	cfg.RPCURL = rpcURL
	cfg.BridgeAddress = common.HexToAddress(bridgeAddrStr)
	cfg.StartBlock = startBlock
	cfg.ChainID = chainID
	cfg.PollInterval = pollInterval
	cfg.Confirmations = confirmations

	return nil
}
