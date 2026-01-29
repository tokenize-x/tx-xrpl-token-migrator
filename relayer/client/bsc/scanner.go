// Package bsc provides BSC blockchain client and event scanning functionality.
package bsc

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

// ScannerConfig is configuration for the BSC event scanner.
type ScannerConfig struct {
	RPCURL        string
	BridgeAddress common.Address
	StartBlock    uint64
	PollInterval  time.Duration
	Confirmations uint64
}

// Scanner is the BSC bridge event scanner.
type Scanner struct {
	cfg      ScannerConfig
	log      logger.Logger
	client   *ethclient.Client
	filterer *abi.TXBridgeFilterer
}

// NewScanner creates a new BSC event scanner.
func NewScanner(cfg ScannerConfig, log logger.Logger, client *ethclient.Client) (*Scanner, error) {
	filterer, err := abi.NewTXBridgeFilterer(cfg.BridgeAddress, client)
	if err != nil {
		client.Close()
		return nil, errors.Wrap(err, "failed to create TXBridge filterer")
	}

	return &Scanner{
		cfg:      cfg,
		log:      log,
		client:   client,
		filterer: filterer,
	}, nil
}

// Subscribe starts scanning for SentToTXChain events and sends them to the channel.
func (s *Scanner) Subscribe(ctx context.Context, ch chan<- *abi.TXBridgeSentToTXChain) error {
	currentBlock, err := s.client.BlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get current block")
	}

	safeBlock := currentBlock - s.cfg.Confirmations

	go s.scanHistorical(ctx, s.cfg.StartBlock, safeBlock, ch)
	go s.scanRecent(ctx, safeBlock, ch)

	return nil
}

// GetCurrentBlock returns the current block number from BSC.
func (s *Scanner) GetCurrentBlock(ctx context.Context) (uint64, error) {
	return s.client.BlockNumber(ctx)
}

// Close closes the BSC client connection.
func (s *Scanner) Close() {
	s.client.Close()
}

func (s *Scanner) scanHistorical(ctx context.Context, from, to uint64, ch chan<- *abi.TXBridgeSentToTXChain) {
	if from >= to {
		return
	}

	s.log.Info("starting BSC historical scan",
		zap.Uint64("from", from), zap.Uint64("to", to))

	batchSize := uint64(10000)
	for start := from; start < to; start += batchSize {
		select {
		case <-ctx.Done():
			return
		default:
		}

		end := start + batchSize - 1
		if end > to {
			end = to
		}

		if err := s.scanRange(ctx, start, end, ch); err != nil {
			s.log.Error("BSC historical scan error", zap.Error(err))
			time.Sleep(s.cfg.PollInterval)
			start -= batchSize // retry the same batch
			continue
		}
	}

	s.log.Info("BSC historical scan completed")
}

func (s *Scanner) scanRecent(ctx context.Context, from uint64, ch chan<- *abi.TXBridgeSentToTXChain) {
	lastBlock := from
	s.log.Info("starting BSC recent block polling", zap.Uint64("from", from))

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.cfg.PollInterval):
		}

		currentBlock, err := s.client.BlockNumber(ctx)
		if err != nil {
			s.log.Error("failed to get BSC block number", zap.Error(err))
			continue
		}

		safeBlock := currentBlock - s.cfg.Confirmations
		if safeBlock <= lastBlock {
			continue
		}

		count, err := s.scanRangeWithCount(ctx, lastBlock+1, safeBlock, ch)
		if err != nil {
			s.log.Error("BSC recent scan error", zap.Error(err))
			continue
		}

		s.log.Info("polled BSC blocks",
			zap.Uint64("from", lastBlock+1), zap.Uint64("to", safeBlock), zap.Int("events", count))
		lastBlock = safeBlock
	}
}

func (s *Scanner) scanRange(ctx context.Context, from, to uint64, ch chan<- *abi.TXBridgeSentToTXChain) error {
	_, err := s.scanRangeWithCount(ctx, from, to, ch)
	return err
}

func (s *Scanner) scanRangeWithCount(
	ctx context.Context, from, to uint64, ch chan<- *abi.TXBridgeSentToTXChain,
) (int, error) {
	iter, err := s.filterer.FilterSentToTXChain(&bind.FilterOpts{
		Start:   from,
		End:     &to,
		Context: ctx,
	}, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to filter SentToTXChain events")
	}
	defer iter.Close()

	count := 0
	for iter.Next() {
		select {
		case <-ctx.Done():
			return count, ctx.Err()
		case ch <- iter.Event:
			count++
		}
	}

	return count, iter.Error()
}
