package service

import (
	"crypto/tls"
	"net/url"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/CoreumFoundation/coreum/app"
	"github.com/CoreumFoundation/coreum/pkg/client"
	"github.com/CoreumFoundation/coreum/pkg/config"
	"github.com/CoreumFoundation/coreum/pkg/config/constant"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/http"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/xrpl"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/executor"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/finder"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/metric"
)

// Config is services config.
type Config struct {
	XRPLRPCURL                 string
	XRPLHistoryScanStartLedger int64
	XRPLRecentScanIndexesBack  int64
	XRPLAccount                string
	XRPLCurrency               string
	XRPLIssuer                 string
	XRPLMemoSuffix             string

	CoreumChainID         string
	CoreumGRPCURL         string
	CoreumMnemonic        string
	CoreumContractAddress string
}

// Services is the struct which aggregates application service.
type Services struct {
	Logger                logger.Logger
	XRPLTxScanner         *xrpl.TxScanner
	CoreumSenderAddress   sdk.AccAddress
	CoreumContractClient  *coreum.ContractClient
	Finder                *finder.Finder
	Executor              *executor.Executor
	MetricRecorder        *metric.Recorder
	MetricServer          *metric.Server
	CoreumMetricCollector *metric.CoreumCollector
}

// NewServices returns new instance on the services.
func NewServices(cfg Config, zapLogger *zap.Logger, setSDKConfig bool) (*Services, error) {
	metricRecorder, err := metric.NewRecorder()
	if err != nil {
		return nil, err
	}

	log := logger.NewZapLogger(zapLogger, metricRecorder)

	httpClient := http.NewRetryableClient(http.DefaultClientConfig())
	rpcClientConfig := xrpl.DefaultRPCClientConfig(cfg.XRPLRPCURL)

	rpcClient := xrpl.NewRPCClient(rpcClientConfig, log, httpClient)

	xrplTxScanner := xrpl.NewTxScanner(xrpl.DefaultTxScannerConfig(), log, rpcClient, metricRecorder)

	network, err := config.NetworkConfigByChainID(constant.ChainID(cfg.CoreumChainID))
	if err != nil {
		return nil, err
	}
	if setSDKConfig {
		network.SetSDKConfig()
	}

	coreumGRPCClient, err := getGRPCClientConn(cfg.CoreumGRPCURL)
	if err != nil {
		return nil, err
	}

	kr := keyring.NewInMemory()
	coreumSenderAddress, err := importMnemonic(kr, constant.CoinType, cfg.CoreumMnemonic)
	if err != nil {
		return nil, err
	}

	clientCtx := client.NewContext(client.DefaultContextConfig(), app.ModuleBasics).
		WithGRPCClient(coreumGRPCClient).
		WithChainID(string(network.ChainID())).
		WithKeyring(kr)

	coreumContractClient := coreum.NewContractClient(coreum.DefaultContractClientConfig(cfg.CoreumContractAddress), clientCtx)

	txFinder := finder.NewFinder(finder.Config{
		XRPLIssuer:                 cfg.XRPLIssuer,
		XRPLCurrency:               cfg.XRPLCurrency,
		XRPLHistoryScanStartLedger: cfg.XRPLHistoryScanStartLedger,
		XRPLRecentScanIndexesBack:  cfg.XRPLRecentScanIndexesBack,
		XRPLMemoSuffix:             cfg.XRPLMemoSuffix,
		CoreumDenom:                network.Denom(),
		CoreumDecimals:             6,
	}, log, xrplTxScanner)

	txExecutor := executor.NewExecutor(executor.DefaultConfig(coreumSenderAddress), log, coreumContractClient, txFinder)

	metricServer := metric.NewServer(metric.DefaultServerConfig(), log, metricRecorder.GetRegistry())

	coreumMetricCollector := metric.NewCoreumCollector(
		metric.DefaultCoreumRecorderConfig(cfg.CoreumContractAddress, coreumSenderAddress.String(), network.Denom()),
		log,
		clientCtx,
		metricRecorder,
	)

	return &Services{
		Logger:                log,
		XRPLTxScanner:         xrplTxScanner,
		CoreumSenderAddress:   coreumSenderAddress,
		CoreumContractClient:  coreumContractClient,
		Finder:                txFinder,
		Executor:              txExecutor,
		MetricRecorder:        metricRecorder,
		MetricServer:          metricServer,
		CoreumMetricCollector: coreumMetricCollector,
	}, nil
}

func importMnemonic(kr keyring.Keyring, coinType uint32, mnemonic string) (sdk.AccAddress, error) {
	keyInfo, err := kr.NewAccount(
		uuid.New().String(),
		mnemonic,
		"",
		hd.CreateHDPath(coinType, 0, 0).String(),
		hd.Secp256k1,
	)
	if err != nil {
		return nil, errors.Wrap(err, "can't import mnemonic to keyring")
	}

	return keyInfo.GetAddress(), nil
}

func getGRPCClientConn(grpcURL string) (*grpc.ClientConn, error) {
	parsedURL, err := url.Parse(grpcURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed parse grpc URL")
	}

	// tls grpc
	if parsedURL.Scheme == "https" {
		grpcClient, err := grpc.Dial(parsedURL.Host, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		if err != nil {
			return nil, errors.Wrap(err, "failed to dial grpc")
		}
		return grpcClient, nil
	}

	grpcClient, err := grpc.Dial(parsedURL.Host, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial grpc")
	}

	return grpcClient, nil
}
