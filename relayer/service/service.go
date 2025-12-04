package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"time"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/CoreumFoundation/coreum-tools/pkg/http"
	"github.com/CoreumFoundation/coreum-tools/pkg/parallel"
	"github.com/CoreumFoundation/coreum/v4/app"
	"github.com/CoreumFoundation/coreum/v4/pkg/client"
	"github.com/CoreumFoundation/coreum/v4/pkg/config"
	"github.com/CoreumFoundation/coreum/v4/pkg/config/constant"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/audit"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/xrpl"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/executor"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/finder"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/metric"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/watcher"
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
	XRPLRPCURL                    string
	XRPLHistoryScanStartLedger    int64
	XRPLRecentScanIndexesBack     int64
	XRPLRecentScanSkipLastIndexes int64

	XRPLMemoSuffix string

	CoreumChainID         string
	CoreumRPCURL          string
	CoreumGRPCURL         string
	CoreumSenderAddress   string
	CoreumContractAddress string

	PrometheusURL          string
	PrometheusInstanceName string
	PrometheusUsername     string
	PrometheusPassword     string

	AuditStartDate            time.Time
	ConfigWatcherPollInterval time.Duration
}

// Services is the struct which aggregates application service.
type Services struct {
	Config                Config
	Logger                logger.Logger
	XRPLTxScanner         *xrpl.TxScanner
	CoreumContractClient  *coreum.ContractClient
	ConfigWatcher         *watcher.ConfigWatcher
	Finders               []*finder.Finder
	Executor              *executor.Executor
	MetricRecorder        *metric.Recorder
	MetricPusher          *metric.Pusher
	CoreumMetricCollector *metric.CoreumCollector
	Auditor               *audit.Auditor
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
	xrplRPCClient := xrpl.NewRPCClient(xrpl.DefaultRPCClientConfig(cfg.XRPLRPCURL), log, httpClient)

	scannerCfg := xrpl.DefaultTxScannerConfig()
	scannerCfg.RecentScanSkipLastIndexes = cfg.XRPLRecentScanSkipLastIndexes
	xrplTxScanner := xrpl.NewTxScanner(scannerCfg, log, xrplRPCClient, metricRecorder)

	network, err := config.NetworkConfigByChainID(constant.ChainID(cfg.CoreumChainID))
	if err != nil {
		return nil, err
	}

	var senderAddress sdk.AccAddress
	if cfg.CoreumSenderAddress != "" {
		senderAddress, err = sdk.AccAddressFromBech32(cfg.CoreumSenderAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid sender address")
		}
	}

	if useInMemoryKr {
		keyInfo, err := kr.KeyByAddress(senderAddress)
		if err != nil {
			return nil, errors.Wrapf(err, fmt.Sprintf("failed to get key from keyring for address:%s", cfg.CoreumSenderAddress))
		}
		pass := uuid.NewString()
		armor, err := kr.ExportPrivKeyArmor(keyInfo.Name, pass)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to export key")
		}
		kr = keyring.NewInMemory(config.NewEncodingConfig(app.ModuleBasics).Codec)
		if err := kr.ImportPrivKey(keyInfo.Name, armor, pass); err != nil {
			return nil, errors.Wrapf(err, "failed to import key")
		}
	}

	var contractAddress sdk.AccAddress
	if cfg.CoreumContractAddress != "" {
		contractAddress, err = sdk.AccAddressFromBech32(cfg.CoreumContractAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid contract address")
		}
	} else {
		// TODO: to be revised in the next PR
		return nil, errors.New("contract address is required")
	}
	coreumClientCtx := client.NewContext(client.DefaultContextConfig(), app.ModuleBasics).
		WithChainID(string(network.ChainID())).
		WithKeyring(kr)

	if cfg.CoreumRPCURL != "" {
		coreumRPCClient, err := cosmosclient.NewClientFromNode(cfg.CoreumRPCURL)
		if err != nil {
			return nil, errors.Wrapf(err, "faild to create coreum RPC client")
		}
		coreumClientCtx = coreumClientCtx.WithClient(coreumRPCClient)
	}

	if cfg.CoreumGRPCURL != "" {
		coreumGRPCClient, err := getGRPCClientConn(cfg.CoreumGRPCURL)
		if err != nil {
			return nil, err
		}
		coreumClientCtx = coreumClientCtx.WithGRPCClient(coreumGRPCClient)
	}

	coreumContractClient := coreum.NewContractClient(
		coreum.DefaultContractClientConfig(
			contractAddress,
			network.Denom(),
		),
		coreumClientCtx,
	)

	// Query contract for initial configuration
	config, err := coreumContractClient.GetContractConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query initial contract config")
	}

	// Create finders and audit config from token configuration
	txFinders, auditTokensCfg, err := createFindersAndAuditConfig(
		config.XRPLTokens,
		cfg.XRPLHistoryScanStartLedger,
		cfg.XRPLRecentScanIndexesBack,
		cfg.XRPLMemoSuffix,
		network.Denom(),
		6,
		log,
		xrplTxScanner,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create finders and audit config")
	}

	// Create executor with the initial finders
	txExecutor := executor.NewExecutor(
		executor.DefaultConfig(senderAddress),
		log,
		coreumContractClient,
		lo.Map(txFinders, func(f *finder.Finder, _ int) executor.Finder {
			return f
		}),
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
		coreumContractClient,
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

	coreumMetricCollector := metric.NewCoreumCollector(
		metric.DefaultCoreumRecorderConfig(contractAddress, senderAddress, network.Denom()),
		log,
		coreumClientCtx,
		metricRecorder,
		coreumContractClient,
	)

	coreumChainClient := coreum.NewChainClient(coreum.DefaultChainClientConfig(), log, coreumClientCtx)

	auditor := audit.NewAuditor(audit.AuditorConfig{
		ContractAddress: contractAddress.String(),
		XRPLMemoSuffix:  cfg.XRPLMemoSuffix,
		CoreumDenom:     network.Denom(),
		CoreumDecimals:  6,
		XRPLTokens:      auditTokensCfg,
		StartDate:       cfg.AuditStartDate,
	}, log, coreumChainClient, xrplRPCClient)

	return &Services{
		Config:                cfg,
		Logger:                log,
		XRPLTxScanner:         xrplTxScanner,
		CoreumContractClient:  coreumContractClient,
		ConfigWatcher:         configWatcher,
		Finders:               txFinders,
		Executor:              txExecutor,
		MetricRecorder:        metricRecorder,
		MetricPusher:          metricPusher,
		CoreumMetricCollector: coreumMetricCollector,
		Auditor:               auditor,
	}, nil
}

// RunExecutorWithAutoRestart runs the executor and restarts it when config changes are detected.
// It continuously runs the executor and watcher in parallel, and automatically restarts
// services when a configuration change is detected.
func RunExecutorWithAutoRestart(ctx context.Context, cfg Config, kr keyring.Keyring, zapLogger *zap.Logger) error {
	for {
		// Check if context is already cancelled before creating services
		if err := ctx.Err(); err != nil {
			return err
		}

		// Create new services for this iteration
		services, err := NewServices(ctx, cfg, kr, true, zapLogger)
		if err != nil {
			return errors.Wrap(err, "failed to create services")
		}

		services.Logger.Info("Starting relayer", zap.String("contract-address", cfg.CoreumContractAddress))

		// Start metric collectors and pushers in parallel with executor and watcher
		// parallel.Run manages its own context lifecycle - when it returns, all tasks are already stopped
		err = parallel.Run(ctx, func(ctx context.Context, spawn parallel.SpawnFn) error {
			spawn("executor", parallel.Fail, services.Executor.Start)
			// Watcher uses Exit mode to initiate graceful shutdown when config changes
			spawn("watcher", parallel.Exit, services.ConfigWatcher.Watch)
			// Spawn metric collector tasks - use Fail mode as they should never return unless context is closed
			spawn("collect-contract-balance", parallel.Fail, services.CoreumMetricCollector.CollectContractBalance)
			spawn("collect-sender-balance", parallel.Fail, services.CoreumMetricCollector.CollectSenderBalance)
			spawn("collect-pending-transactions", parallel.Fail, services.CoreumMetricCollector.CollectPendingTransactions)
			// Spawn metric pusher - use Fail mode as it should never return unless context is closed
			if services.MetricPusher != nil {
				spawn("metric-pusher", parallel.Fail, services.MetricPusher.PushMetrics)
			}
			return nil
		}, parallel.WithGroupLogger(parallel.NewZapLogger(zapLogger)))

		// Handle the result
		// Config change detected - restart services
		// parallel.Run has already cancelled all tasks when it returns
		if err != nil && errors.Is(err, watcher.ErrConfigChanged) {
			services.Logger.Info("Config changed, restarting services")
			// Services will be recreated on next iteration
			continue
		}

		// Context cancelled - graceful shutdown
		if err != nil && errors.Is(err, context.Canceled) {
			services.Logger.Info("Context cancelled, shutting down")
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
	tokens []coreum.XRPLToken,
	xrplHistoryScanStartLedger int64,
	xrplRecentScanIndexesBack int64,
	xrplMemoSuffix string,
	coreumDenom string,
	coreumDecimals int,
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
			CoreumDenom:                coreumDenom,
			CoreumDecimals:             coreumDecimals,
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

	encodingConfig := config.NewEncodingConfig(app.ModuleBasics)
	pc, ok := encodingConfig.Codec.(codec.GRPCCodecProvider)
	if !ok {
		return nil, errors.New("failed to cast codec to codec.GRPCCodecProvider)")
	}

	host := parsedURL.Host

	// https - tls grpc
	if parsedURL.Scheme == "https" {
		grpcClient, err := grpc.Dial(
			host,
			grpc.WithDefaultCallOptions(grpc.ForceCodec(pc.GRPCCodec())),
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to dial grpc")
		}
		return grpcClient, nil
	}

	// handling of host:port URL without the protocol
	if host == "" {
		host = fmt.Sprintf("%s:%s", parsedURL.Scheme, parsedURL.Opaque)
	}
	// http - insecure
	grpcClient, err := grpc.Dial(
		host,
		grpc.WithDefaultCallOptions(grpc.ForceCodec(pc.GRPCCodec())),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial grpc")
	}

	return grpcClient, nil
}
