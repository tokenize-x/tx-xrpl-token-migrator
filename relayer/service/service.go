package service

import (
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

	XRPLTokens []XRPLTokenConfig

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

	AuditStartDate time.Time
}

// Services is the struct which aggregates application service.
type Services struct {
	Config                Config
	Logger                logger.Logger
	XRPLTxScanner         *xrpl.TxScanner
	CoreumContractClient  *coreum.ContractClient
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
func NewServices(cfg Config, kr keyring.Keyring, useInMemoryKr bool, zapLogger *zap.Logger) (*Services, error) {
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

	txFinders := make([]*finder.Finder, 0, len(cfg.XRPLTokens))
	auditTokensCfg := make([]audit.XRPLTokenConfig, 0, len(cfg.XRPLTokens))
	for _, tokenCfg := range cfg.XRPLTokens {
		xrplIssuer, err := rippledata.NewAccountFromAddress(tokenCfg.XRPLIssuer)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert XRPLIssuer string to type, value:%s", tokenCfg.XRPLIssuer)
		}
		xrplCurrency, err := rippledata.NewCurrency(tokenCfg.XRPLCurrency)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to convert XRPLCurrency string to type, value:%s", tokenCfg.XRPLCurrency)
		}
		txFinder := finder.NewFinder(finder.Config{
			XRPLIssuer:                 *xrplIssuer,
			XRPLCurrency:               xrplCurrency,
			ActivationDate:             tokenCfg.ActivationDate,
			Multiplier:                 tokenCfg.Multiplier,
			XRPLHistoryScanStartLedger: cfg.XRPLHistoryScanStartLedger,
			XRPLRecentScanIndexesBack:  cfg.XRPLRecentScanIndexesBack,
			XRPLMemoSuffix:             cfg.XRPLMemoSuffix,
			CoreumDenom:                network.Denom(),
			CoreumDecimals:             6,
		}, log, xrplTxScanner)
		txFinders = append(txFinders, txFinder)

		auditTokensCfg = append(auditTokensCfg, audit.XRPLTokenConfig{
			XRPLIssuer:   *xrplIssuer,
			XRPLCurrency: xrplCurrency,
			Multiplier:   tokenCfg.Multiplier,
		})
	}

	txExecutor := executor.NewExecutor(
		executor.DefaultConfig(senderAddress),
		log,
		coreumContractClient,
		lo.Map(txFinders, func(finder *finder.Finder, _ int) executor.Finder {
			return finder
		}),
	)

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
		Finders:               txFinders,
		Executor:              txExecutor,
		MetricRecorder:        metricRecorder,
		MetricPusher:          metricPusher,
		CoreumMetricCollector: coreumMetricCollector,
		Auditor:               auditor,
	}, nil
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
