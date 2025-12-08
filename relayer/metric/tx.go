package metric

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/coreum/v4/pkg/client"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

// TXRecorder is metrics recorder required for the collector.
type TXRecorder interface {
	SetTXSenderBalance(v int64)
	SetTXContractBalance(v int64)
	SetTXPendingUnapprovedTransactionsCount(v int)
	SetTXPendingApprovedTransactionsCount(v int)
}

// ContractClient defines contract client interface.
type ContractClient interface {
	GetAllPendingTransactions(ctx context.Context) ([]tx.PendingTransaction, []tx.PendingTransaction, error)
}

// TXRecorderConfig represents TXRecorder config.
type TXRecorderConfig struct {
	ContractAddress sdk.AccAddress
	SenderAddress   sdk.AccAddress
	Denom           string
	RepeatDelay     time.Duration
}

// DefaultTXRecorderConfig returns TXRecorder default config.
func DefaultTXRecorderConfig(contractAddress, senderAddress sdk.AccAddress, denom string) TXRecorderConfig {
	return TXRecorderConfig{
		ContractAddress: contractAddress,
		SenderAddress:   senderAddress,
		Denom:           denom,
		RepeatDelay:     30 * time.Second,
	}
}

// TXCollector is TX metrics collector.
type TXCollector struct {
	cfg            TXRecorderConfig
	log            logger.Logger
	bankClient     banktypes.QueryClient
	metricRecorder TXRecorder
	contractClient ContractClient
}

// NewTXCollector returns a new instance of the TXCollector.
func NewTXCollector(
	cfg TXRecorderConfig,
	log logger.Logger,
	clientCtx client.Context,
	metricRecorder TXRecorder,
	contractClient ContractClient,
) *TXCollector {
	return &TXCollector{
		cfg:            cfg,
		log:            log,
		bankClient:     banktypes.NewQueryClient(clientCtx),
		metricRecorder: metricRecorder,
		contractClient: contractClient,
	}
}

// CollectContractBalance collects contract balance metrics.
func (c *TXCollector) CollectContractBalance(ctx context.Context) error {
	return c.collectBalance(ctx, c.cfg.ContractAddress.String(), c.metricRecorder.SetTXContractBalance)
}

// CollectSenderBalance collects sender balance metrics.
func (c *TXCollector) CollectSenderBalance(ctx context.Context) error {
	return c.collectBalance(ctx, c.cfg.SenderAddress.String(), c.metricRecorder.SetTXSenderBalance)
}

// CollectPendingTransactions collects pending transactions metrics.
func (c *TXCollector) CollectPendingTransactions(ctx context.Context) error {
	return c.collectPendingTransactions(ctx)
}

func (c *TXCollector) collectBalance(ctx context.Context, accAddress string, setter func(int64)) error {
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
		return err
	}
	// this panic is unexpected
	panic(errors.Wrap(err, "unexpected error in collect balance"))
}

func (c *TXCollector) collectPendingTransactions(ctx context.Context) error {
	err := retry.Do(ctx, c.cfg.RepeatDelay, func() error {
		unapprovedTransactions, approvedTransactions, err := c.contractClient.GetAllPendingTransactions(ctx)
		if err != nil {
			c.log.Error("Error on getting contract pending transactions", zap.Error(err))
			return retry.Retryable(errors.New("repeat metric collector"))
		}

		c.metricRecorder.SetTXPendingUnapprovedTransactionsCount(len(unapprovedTransactions))
		c.metricRecorder.SetTXPendingApprovedTransactionsCount(len(approvedTransactions))

		return retry.Retryable(errors.New("repeat metric collector"))
	})
	if err == nil || errors.Is(err, context.Canceled) {
		return err
	}
	// this panic is unexpected
	panic(errors.Wrap(err, "unexpected error in collect pending transactions"))
}
