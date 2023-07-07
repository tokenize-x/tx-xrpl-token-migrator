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
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/coreum"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// CoreumRecorder is metrics recorder required for the collector.
type CoreumRecorder interface {
	SetCoreumSenderBalance(v int64)
	SetCoreumContractBalance(v int64)
	SetCoreumPendingUnapprovedTransactionsCount(v int)
	SetCoreumPendingApprovedTransactionsCount(v int)
}

// ContractClient defines contract client interface.
type ContractClient interface {
	GetConfig(ctx context.Context) (coreum.Config, error)
	GetPendingTxs(ctx context.Context, offset *uint64, limit *uint32) ([]coreum.PendingTransaction, error)
}

// CoreumRecorderConfig represents CoreumRecorder config.
type CoreumRecorderConfig struct {
	ContractAddress  sdk.AccAddress
	ContractPageSize uint32
	SenderAddress    sdk.AccAddress
	Denom            string
	RepeatDelay      time.Duration
}

// DefaultCoreumRecorderConfig returns CoreumRecorder default config.
func DefaultCoreumRecorderConfig(contractAddress, senderAddress sdk.AccAddress, denom string) CoreumRecorderConfig {
	return CoreumRecorderConfig{
		ContractAddress:  contractAddress,
		ContractPageSize: 500,
		SenderAddress:    senderAddress,
		Denom:            denom,
		RepeatDelay:      30 * time.Second,
	}
}

// CoreumCollector is coreum metrics collector.
type CoreumCollector struct {
	cfg            CoreumRecorderConfig
	log            logger.Logger
	bankClient     banktypes.QueryClient
	metricRecorder CoreumRecorder
	contractClient ContractClient
}

// NewCoreumCollector returns a new instance of the CoreumCollector.
func NewCoreumCollector(
	cfg CoreumRecorderConfig,
	log logger.Logger,
	clientCtx client.Context,
	metricRecorder CoreumRecorder,
	contractClient ContractClient,
) *CoreumCollector {
	return &CoreumCollector{
		cfg:            cfg,
		log:            log,
		bankClient:     banktypes.NewQueryClient(clientCtx),
		metricRecorder: metricRecorder,
		contractClient: contractClient,
	}
}

// Start starts the metric collector.
func (c *CoreumCollector) Start(ctx context.Context) {
	c.startCollectingBalance(ctx, c.cfg.ContractAddress.String(), c.metricRecorder.SetCoreumContractBalance)
	c.startCollectingBalance(ctx, c.cfg.SenderAddress.String(), c.metricRecorder.SetCoreumSenderBalance)
	c.startCollectingPendingTransactions(ctx)
}

func (c *CoreumCollector) startCollectingBalance(ctx context.Context, accAddress string, setter func(int64)) {
	go func() {
		err := retry.Do(ctx, c.cfg.RepeatDelay, func() error {
			balanceRes, err := c.bankClient.Balance(ctx, &banktypes.QueryBalanceRequest{
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

func (c *CoreumCollector) startCollectingPendingTransactions(ctx context.Context) {
	go func() {
		err := retry.Do(ctx, c.cfg.RepeatDelay, func() error {
			offset := uint64(0)
			limit := c.cfg.ContractPageSize

			var (
				unapprovedTransactionsCount int
				approvedTransactionsCount   int
			)

			contractCfg, err := c.contractClient.GetConfig(ctx)
			if err != nil {
				c.log.Error("Error on getting contract config", zap.Error(err))
				return retry.Retryable(err)
			}

			for {
				pendingTxs, err := c.contractClient.GetPendingTxs(ctx, &offset, &limit)
				if err != nil {
					c.log.Error("Error on getting contract pending transactions", zap.Error(err))
					return retry.Retryable(errors.New("repeat metric collector"))
				}
				if len(pendingTxs) == 0 {
					break
				}

				for _, pendingTx := range pendingTxs {
					if len(pendingTx.EvidenceProviders) < contractCfg.Threshold {
						unapprovedTransactionsCount++
						continue
					}
					approvedTransactionsCount++
				}

				offset += uint64(c.cfg.ContractPageSize)
				limit += c.cfg.ContractPageSize
			}

			c.metricRecorder.SetCoreumPendingUnapprovedTransactionsCount(unapprovedTransactionsCount)
			c.metricRecorder.SetCoreumPendingApprovedTransactionsCount(approvedTransactionsCount)

			return retry.Retryable(errors.New("repeat metric collector"))
		})
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		// this panic is unexpected
		panic(errors.Wrap(err, "unexpected error in collect balance"))
	}()
}
