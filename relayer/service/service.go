package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	"github.com/CoreumFoundation/coreum-tools/pkg/http"
	"github.com/CoreumFoundation/coreum-tools/pkg/parallel"
	"github.com/CoreumFoundation/coreum/v5/pkg/client"
	"github.com/CoreumFoundation/coreum/v5/pkg/config"
	"github.com/CoreumFoundation/coreum/v5/pkg/config/constant"
	"github.com/CosmWasm/wasmd/x/wasm"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/audit"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/xrpl"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/executor"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/finder"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/metric"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/watcher"
)

// XRPLTokenConfig is XRPL token config.
type XRPLTokenConfig struct {
	XRPLCurrency   string
	XRPLIssuer     string
	ActivationDate time.Time
	Multiplier     string
}

// Config is services config.
type Config struct {
	XRPLScannerDisabled           bool
	XRPLRPCURL                    string
	XRPLHistoryScanStartLedger    int64
	XRPLRecentScanIndexesBack     int64
	XRPLRecentScanSkipLastIndexes int64
	XRPLMemoSuffix                string

	BSCScannerDisabled bool
	BSCScanner         bsc.ScannerConfig

	TXChainID         string
	TXRPCURL          string
	TXGRPCURL         string
	TXSenderAddress   string
	TXContractAddress string

	PrometheusURL          string
	PrometheusInstanceName string
	PrometheusUsername     string
	PrometheusPassword     string

	AuditStartDate            time.Time
	ConfigWatcherPollInterval time.Duration
}

// Services is the struct which aggregates application service.
type Services struct {
	Config            Config
	Logger            logger.Logger
	XRPLTxScanner     *xrpl.TxScanner
	TXContractClient  *tx.ContractClient
	ConfigWatcher     *watcher.ConfigWatcher
	Finders           []*finder.Finder
	Executor          *executor.Executor
	MetricRecorder    *metric.Recorder
	MetricPusher      *metric.Pusher
	TXMetricCollector *metric.TXCollector
	Auditor           *audit.Auditor
}

// buildTXClientContext builds and returns a TX client context with RPC and gRPC connections.
func buildTXClientContext(
	cfg Config,
	kr keyring.Keyring,
) (client.Context, error) {
	network, err := config.NetworkConfigByChainID(constant.ChainID(cfg.TXChainID))
	if err != nil {
		return client.Context{}, err
	}

	txClientCtx := client.NewContext(client.DefaultContextConfig(), auth.AppModuleBasic{}, wasm.AppModuleBasic{}).
		WithChainID(string(network.ChainID())).
		WithKeyring(kr)

	if cfg.TXRPCURL != "" {
		txRPCClient, err := cosmosclient.NewClientFromNode(cfg.TXRPCURL)
		if err != nil {
			return client.Context{}, errors.Wrapf(err, "faild to create TX RPC client")
		}
		txClientCtx = txClientCtx.WithClient(txRPCClient)
	}

	if cfg.TXGRPCURL != "" {
		txGRPCClient, err := getGRPCClientConn(cfg.TXGRPCURL)
		if err != nil {
			return client.Context{}, err
		}
		txClientCtx = txClientCtx.WithGRPCClient(txGRPCClient)
	}

	return txClientCtx, nil
}

// NewDeployContractClient creates a contract client and logger for deploy operations.
// This function does not require a contract address since it's used to deploy new contracts.
func NewDeployContractClient(
	cfg Config,
	kr keyring.Keyring,
	zapLogger *zap.Logger,
) (*tx.ContractClient, logger.Logger, error) {
	metricRecorder, err := metric.NewRecorder()
	if err != nil {
		return nil, nil, err
	}

	log := logger.NewZapLogger(zapLogger, metricRecorder)

	network, err := config.NetworkConfigByChainID(constant.ChainID(cfg.TXChainID))
	if err != nil {
		return nil, nil, err
	}

	txClientCtx, err := buildTXClientContext(cfg, kr)
	if err != nil {
		return nil, nil, err
	}

	// For deploy operations, contract address is nil
	deployClient := tx.NewContractClient(
		tx.DefaultContractClientConfig(
			nil,
			network.Denom(),
		),
		txClientCtx,
	)

	return deployClient, log, nil
}

// NewContractClient creates a contract client and logger for contract operations.
// Contract address is required for operations on existing contracts.
func NewContractClient(
	ctx context.Context,
	cfg Config,
	kr keyring.Keyring,
	zapLogger *zap.Logger,
) (*tx.ContractClient, logger.Logger, error) {
	metricRecorder, err := metric.NewRecorder()
	if err != nil {
		return nil, nil, err
	}

	log := logger.NewZapLogger(zapLogger, metricRecorder)

	network, err := config.NetworkConfigByChainID(constant.ChainID(cfg.TXChainID))
	if err != nil {
		return nil, nil, err
	}

	if cfg.TXContractAddress == "" {
		return nil, nil, errors.New("contract address is required")
	}

	contractAddress, err := sdk.AccAddressFromBech32(cfg.TXContractAddress)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "invalid contract address")
	}

	txClientCtx, err := buildTXClientContext(cfg, kr)
	if err != nil {
		return nil, nil, err
	}

	txContractClient := tx.NewContractClient(
		tx.DefaultContractClientConfig(
			contractAddress,
			network.Denom(),
		),
		txClientCtx,
	)

	return txContractClient, log, nil
}

// NewAuditor creates an auditor and logger for audit operations.
func NewAuditor(
	ctx context.Context,
	cfg Config,
	kr keyring.Keyring,
	zapLogger *zap.Logger,
) (*audit.Auditor, logger.Logger, error) {
	metricRecorder, err := metric.NewRecorder()
	if err != nil {
		return nil, nil, err
	}

	log := logger.NewZapLogger(zapLogger, metricRecorder)
	httpClient := http.NewRetryableClient(http.DefaultClientConfig())
	xrplRPCClient := xrpl.NewRPCClient(xrpl.DefaultRPCClientConfig(cfg.XRPLRPCURL), log, httpClient)

	network, err := config.NetworkConfigByChainID(constant.ChainID(cfg.TXChainID))
	if err != nil {
		return nil, nil, err
	}

	var contractAddress sdk.AccAddress
	if cfg.TXContractAddress != "" {
		contractAddress, err = sdk.AccAddressFromBech32(cfg.TXContractAddress)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "invalid contract address")
		}
	} else {
		return nil, nil, errors.New("contract address is required")
	}

	txClientCtx, err := buildTXClientContext(cfg, kr)
	if err != nil {
		return nil, nil, err
	}

	txContractClient := tx.NewContractClient(
		tx.DefaultContractClientConfig(
			contractAddress,
			network.Denom(),
		),
		txClientCtx,
	)

	// Query contract for token configuration
	contractCfg, err := txContractClient.GetContractConfig(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to query contract config")
	}

	// Create audit config from token configuration
	auditTokensCfg := make([]audit.XRPLTokenConfig, 0, len(contractCfg.XRPLTokens))
	for _, token := range contractCfg.XRPLTokens {
		xrplIssuer, err := rippledata.NewAccountFromAddress(token.Issuer)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert XRPLIssuer string to type, value:%s", token.Issuer)
		}

		xrplCurrency, err := rippledata.NewCurrency(token.Currency)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert XRPLCurrency string to type, value:%s", token.Currency)
		}

		auditTokensCfg = append(auditTokensCfg, audit.XRPLTokenConfig{
			XRPLIssuer:   *xrplIssuer,
			XRPLCurrency: xrplCurrency,
			Multiplier:   token.Multiplier,
		})
	}

	txChainClient := tx.NewChainClient(tx.DefaultChainClientConfig(), log, txClientCtx)

	auditor := audit.NewAuditor(audit.AuditorConfig{
		ContractAddress: contractAddress.String(),
		XRPLMemoSuffix:  cfg.XRPLMemoSuffix,
		TXDenom:         network.Denom(),
		TXDecimals:      6,
		XRPLTokens:      auditTokensCfg,
		StartDate:       cfg.AuditStartDate,
	}, log, txChainClient, xrplRPCClient)

	return auditor, log, nil
}

// NewServices returns new instance on the services.
//
//nolint:funlen // step-by step initialization
func NewServices(
	ctx context.Context, cfg Config, kr keyring.Keyring, useInMemoryKr bool, zapLogger *zap.Logger,
) (*Services, error) {
	metricRecorder, err := metric.NewRecorder()
	if err != nil {
		return nil, err
	}

	log := logger.NewZapLogger(zapLogger, metricRecorder)
	httpClient := http.NewRetryableClient(http.DefaultClientConfig())

	if cfg.XRPLScannerDisabled && cfg.BSCScannerDisabled {
		return nil, errors.New("at least one scanner (XRPL or BSC) must be enabled")
	}

	var xrplTxScanner *xrpl.TxScanner
	if !cfg.XRPLScannerDisabled {
		if cfg.XRPLRPCURL == "" {
			return nil, errors.New("xrpl-rpc-url is required when XRPL scanner is enabled")
		}
		xrplRPCClient := xrpl.NewRPCClient(xrpl.DefaultRPCClientConfig(cfg.XRPLRPCURL), log, httpClient)
		scannerCfg := xrpl.DefaultTxScannerConfig()
		scannerCfg.RecentScanSkipLastIndexes = cfg.XRPLRecentScanSkipLastIndexes
		xrplTxScanner = xrpl.NewTxScanner(scannerCfg, log, xrplRPCClient, metricRecorder)
		log.Info("XRPL scanner enabled", zap.String("rpcURL", cfg.XRPLRPCURL))
	} else {
		log.Info("XRPL scanner disabled via flag")
	}

	var bscScanner *bsc.Scanner
	if !cfg.BSCScannerDisabled {
		if cfg.BSCScanner.RPCURL == "" {
			return nil, errors.New("bsc-rpc-url is required when BSC scanner is enabled")
		}
		ethClient, err := ethclient.Dial(cfg.BSCScanner.RPCURL)
		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to BSC RPC")
		}
		bscScanner, err = bsc.NewScanner(cfg.BSCScanner, log, ethClient, metricRecorder)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create BSC scanner")
		}
		log.Info("BSC scanner enabled",
			zap.String("rpcURL", cfg.BSCScanner.RPCURL),
			zap.String("bridgeAddress", cfg.BSCScanner.BridgeAddress.Hex()),
		)
	} else {
		log.Info("BSC scanner disabled via flag")
	}

	network, err := config.NetworkConfigByChainID(constant.ChainID(cfg.TXChainID))
	if err != nil {
		return nil, err
	}

	var senderAddress sdk.AccAddress
	if cfg.TXSenderAddress != "" {
		senderAddress, err = sdk.AccAddressFromBech32(cfg.TXSenderAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid sender address")
		}
	}

	if useInMemoryKr {
		keyInfo, err := kr.KeyByAddress(senderAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get key from keyring for address:%s", cfg.TXSenderAddress)
		}
		pass := uuid.NewString()
		armor, err := kr.ExportPrivKeyArmor(keyInfo.Name, pass)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to export key")
		}
		kr = keyring.NewInMemory(config.NewEncodingConfig(auth.AppModuleBasic{}, wasm.AppModuleBasic{}).Codec)
		if err := kr.ImportPrivKey(keyInfo.Name, armor, pass); err != nil {
			return nil, errors.Wrapf(err, "failed to import key")
		}
	}

	var contractAddress sdk.AccAddress
	if cfg.TXContractAddress != "" {
		contractAddress, err = sdk.AccAddressFromBech32(cfg.TXContractAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid contract address")
		}
	} else {
		return nil, errors.New("contract address is required")
	}

	txClientCtx, err := buildTXClientContext(cfg, kr)
	if err != nil {
		return nil, err
	}

	txContractClient := tx.NewContractClient(
		tx.DefaultContractClientConfig(
			contractAddress,
			network.Denom(),
		),
		txClientCtx,
	)

	// Query contract for initial configuration
	contractCfg, err := txContractClient.GetContractConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query initial contract config")
	}

	var allFinders []executor.Finder
	var txFinders []*finder.Finder

	// XRPL finders
	if xrplTxScanner != nil {
		txFinders, _, err = createFindersAndAuditConfig(
			contractCfg.XRPLTokens,
			cfg.XRPLHistoryScanStartLedger,
			cfg.XRPLRecentScanIndexesBack,
			cfg.XRPLMemoSuffix,
			network.Denom(),
			6,
			log,
			xrplTxScanner,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create XRPL finders")
		}
		allFinders = lo.Map(txFinders, func(f *finder.Finder, _ int) executor.Finder {
			return f
		})
	}

	// BSC finder
	if bscScanner != nil {
		bscFinder := finder.NewBSCFinder(
			finder.BSCFinderConfig{
				TXDenom:    network.Denom(),
				TXDecimals: 6,
			},
			log,
			bscScanner,
		)
		allFinders = append(allFinders, bscFinder)
	}

	// Create executor with all finders
	txExecutor := executor.NewExecutor(
		executor.DefaultConfig(senderAddress),
		log,
		txContractClient,
		allFinders,
	)

	// Create ConfigWatcher for dynamic token configuration management
	pollInterval := cfg.ConfigWatcherPollInterval
	if pollInterval == 0 {
		pollInterval = 5 * time.Minute // Default poll interval
	}
	configWatcher := watcher.NewConfigWatcher(
		watcher.Config{
			ContractAddress: contractAddress,
			PollInterval:    pollInterval,
		},
		log,
		txContractClient,
	)

	// Initialize ConfigWatcher with current version
	if err := configWatcher.Initialize(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to initialize config watcher")
	}

	var metricPusher *metric.Pusher
	if cfg.PrometheusURL != "" {
		metricPusher, err = metric.NewPusher(
			metric.DefaultPusherConfig(
				cfg.PrometheusURL,
				cfg.PrometheusUsername,
				cfg.PrometheusPassword,
				cfg.PrometheusInstanceName,
			),
			log,
			metricRecorder.GetRegistry(),
		)
		if err != nil {
			return nil, err
		}
	}

	txMetricCollector := metric.NewTXCollector(
		metric.DefaultTXRecorderConfig(contractAddress, senderAddress, network.Denom()),
		log,
		txClientCtx,
		metricRecorder,
		txContractClient,
	)

	// Create auditor using NewAuditor function
	auditor, _, err := NewAuditor(ctx, cfg, kr, zapLogger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create auditor")
	}

	return &Services{
		Config:            cfg,
		Logger:            log,
		XRPLTxScanner:     xrplTxScanner,
		TXContractClient:  txContractClient,
		ConfigWatcher:     configWatcher,
		Finders:           txFinders,
		Executor:          txExecutor,
		MetricRecorder:    metricRecorder,
		MetricPusher:      metricPusher,
		TXMetricCollector: txMetricCollector,
		Auditor:           auditor,
	}, nil
}

// RunExecutorWithAutoRestart runs the executor and restarts it when config changes are detected.
// It continuously runs the executor and watcher in parallel, and automatically restarts
// services when a configuration change is detected.
func RunExecutorWithAutoRestart(ctx context.Context, cfg Config, kr keyring.Keyring, zapLogger *zap.Logger) error {
	for {
		// Check if context is already canceled before creating services
		if err := ctx.Err(); err != nil {
			return err
		}

		// Create new services for this iteration
		services, err := NewServices(ctx, cfg, kr, true, zapLogger)
		if err != nil {
			return errors.Wrap(err, "failed to create services")
		}

		services.Logger.Info("Starting relayer", zap.String("contract-address", cfg.TXContractAddress))

		// Start metric collectors and pushers in parallel with executor and watcher
		// parallel.Run manages its own context lifecycle - when it returns, all tasks are already stopped
		err = parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
			spawn("executor", parallel.Fail, services.Executor.Start)
			// Watcher uses Exit mode to initiate graceful shutdown when config changes
			spawn("watcher", parallel.Exit, services.ConfigWatcher.Watch)
			// Spawn metric collector tasks - use Fail mode as they should never return unless context is closed
			spawn("collect-contract-balance", parallel.Fail, services.TXMetricCollector.CollectContractBalance)
			spawn("collect-sender-balance", parallel.Fail, services.TXMetricCollector.CollectSenderBalance)
			spawn("collect-pending-transactions", parallel.Fail, services.TXMetricCollector.CollectPendingTransactions)
			// Spawn metric pusher - use Fail mode as it should never return unless context is closed
			if services.MetricPusher != nil {
				spawn("metric-pusher", parallel.Fail, services.MetricPusher.PushMetrics)
			}
			return nil
		}, parallel.WithGroupLogger(parallel.NewZapLogger(zapLogger)))

		// Handle the result
		// Config change detected - restart services
		// parallel.Run has already canceled all tasks when it returns
		if err != nil && errors.Is(err, watcher.ErrConfigChanged) {
			services.Logger.Info("Config changed, restarting services")
			// Services will be recreated on next iteration
			continue
		}

		// Context canceled - graceful shutdown
		if err != nil && errors.Is(err, context.Canceled) {
			services.Logger.Info("Context canceled, shutting down")
			return nil
		}

		// Other error - return it
		if err != nil {
			services.Logger.Info("Service stopped", zap.Error(err))
			return err
		}

		// Normal exit (shouldn't happen in practice, but handle it)
		return nil
	}
}

// createFindersAndAuditConfig creates finders and audit config from token configuration.
func createFindersAndAuditConfig(
	tokens []tx.XRPLToken,
	xrplHistoryScanStartLedger int64,
	xrplRecentScanIndexesBack int64,
	xrplMemoSuffix string,
	txDenom string,
	txDecimals int,
	log logger.Logger,
	txScanner *xrpl.TxScanner,
) ([]*finder.Finder, []audit.XRPLTokenConfig, error) {
	finders := make([]*finder.Finder, 0, len(tokens))
	auditCfg := make([]audit.XRPLTokenConfig, 0, len(tokens))

	for _, token := range tokens {
		xrplIssuer, err := rippledata.NewAccountFromAddress(token.Issuer)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert XRPLIssuer string to type, value:%s", token.Issuer)
		}

		xrplCurrency, err := rippledata.NewCurrency(token.Currency)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert XRPLCurrency string to type, value:%s", token.Currency)
		}

		txFinder := finder.NewFinder(finder.Config{
			XRPLIssuer:                 *xrplIssuer,
			XRPLCurrency:               xrplCurrency,
			ActivationDate:             time.Unix(int64(token.ActivationDate), 0),
			Multiplier:                 token.Multiplier,
			XRPLHistoryScanStartLedger: xrplHistoryScanStartLedger,
			XRPLRecentScanIndexesBack:  xrplRecentScanIndexesBack,
			XRPLMemoSuffix:             xrplMemoSuffix,
			TXDenom:                    txDenom,
			TXDecimals:                 txDecimals,
		}, log, txScanner)
		finders = append(finders, txFinder)

		auditCfg = append(auditCfg, audit.XRPLTokenConfig{
			XRPLIssuer:   *xrplIssuer,
			XRPLCurrency: xrplCurrency,
			Multiplier:   token.Multiplier,
		})
	}

	return finders, auditCfg, nil
}

func getGRPCClientConn(grpcURL string) (*grpc.ClientConn, error) {
	parsedURL, err := url.Parse(grpcURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse grpc URL")
	}

	encodingConfig := config.NewEncodingConfig(auth.AppModuleBasic{}, wasm.AppModuleBasic{})
	pc, ok := encodingConfig.Codec.(codec.GRPCCodecProvider)
	if !ok {
		return nil, errors.New("failed to cast codec to codec.GRPCCodecProvider)")
	}

	host := parsedURL.Host

	// https - tls grpc
	if parsedURL.Scheme == "https" {
		grpcClient, err := grpc.NewClient(
			host,
			grpc.WithDefaultCallOptions(grpc.ForceCodec(pc.GRPCCodec())),
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create grpc client")
		}
		return grpcClient, nil
	}

	// handling of host:port URL without the protocol
	if host == "" {
		host = fmt.Sprintf("%s:%s", parsedURL.Scheme, parsedURL.Opaque)
	}
	// http - insecure
	grpcClient, err := grpc.NewClient(
		host,
		grpc.WithDefaultCallOptions(grpc.ForceCodec(pc.GRPCCodec())),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create grpc client")
	}

	return grpcClient, nil
}
