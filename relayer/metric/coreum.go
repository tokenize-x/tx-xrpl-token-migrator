package metric

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/coreum/pkg/client"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// CoreumRecorder is metrics recorder required for the collector.
type CoreumRecorder interface {
	SetCoreumSenderBalance(v int64)
	SetCoreumContractBalance(v int64)
}

// CoreumRecorderConfig represents CoreumRecorder config.
type CoreumRecorderConfig struct {
	ContractAddress sdk.AccAddress
	SenderAddress   sdk.AccAddress
	Denom           string

	RequestTimeout time.Duration
	RepeatDelay    time.Duration
}

// DefaultCoreumRecorderConfig returns CoreumRecorder default config.
func DefaultCoreumRecorderConfig(contractAddress, senderAddress sdk.AccAddress, denom string) CoreumRecorderConfig {
	return CoreumRecorderConfig{
		ContractAddress: contractAddress,
		SenderAddress:   senderAddress,
		Denom:           denom,

		RequestTimeout: 10 * time.Second,
		RepeatDelay:    30 * time.Second,
	}
}

// CoreumCollector is coreum metrics collector.
type CoreumCollector struct {
	cfg            CoreumRecorderConfig
	log            logger.Logger
	bankClient     banktypes.QueryClient
	metricRecorder CoreumRecorder
}

// NewCoreumCollector returns a new instance of the CoreumCollector.
func NewCoreumCollector(
	cfg CoreumRecorderConfig,
	log logger.Logger,
	clientCtx client.Context,
	metricRecorder CoreumRecorder,
) *CoreumCollector {
	return &CoreumCollector{
		cfg:            cfg,
		log:            log,
		bankClient:     banktypes.NewQueryClient(clientCtx),
		metricRecorder: metricRecorder,
	}
}

// Start starts the metric collector.
func (c *CoreumCollector) Start(ctx context.Context) {
	c.startCollectingBalance(ctx, c.cfg.ContractAddress.String(), c.metricRecorder.SetCoreumContractBalance)
	c.startCollectingBalance(ctx, c.cfg.SenderAddress.String(), c.metricRecorder.SetCoreumSenderBalance)
}

func (c *CoreumCollector) startCollectingBalance(ctx context.Context, accAddress string, setter func(int64)) {
	go func() {
		err := retry.Do(ctx, c.cfg.RepeatDelay, func() error {
			requestCtx, requestCancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
			defer requestCancel()
			balanceRes, err := c.bankClient.Balance(requestCtx, &banktypes.QueryBalanceRequest{
				Address: accAddress,
				Denom:   c.cfg.Denom,
			})
			if err != nil {
				c.log.Error(
					"Error on getting account balance",
					zap.String("account", accAddress),
					zap.Error(err),
				)
				return retry.Retryable(errors.New("repeat metric collector"))
			}
			setter(balanceRes.Balance.Amount.Int64())

			return retry.Retryable(errors.New("repeat metric collector"))
		})
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		// this panic is unexpected
		panic(errors.Wrap(err, "unexpected error in collect balance"))
	}()
}
