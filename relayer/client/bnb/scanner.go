package bnb

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bnb/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"
)

// configuration for the BNB event scanner.
type ScannerConfig struct {
	RPCURL        string
	BridgeAddress common.Address
	StartBlock    uint64
	PollInterval  time.Duration
	Confirmations uint64
	ChainSuffix   string // chain suffix to strip from txchainAddress (e.g., "/coreum-testnet-1/v1")
}

// Scanner is the BNB bridge event scanner.
type Scanner struct {
	cfg      ScannerConfig
	log      logger.Logger
	client   *ethclient.Client
	filterer *abi.TxBridgeFilterer
}

func NewScanner(cfg ScannerConfig, log logger.Logger) (*Scanner, error) {
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to BNB RPC")
	}

	filterer, err := abi.NewTxBridgeFilterer(cfg.BridgeAddress, client)
	if err != nil {
		client.Close()
		return nil, errors.Wrap(err, "failed to create TxBridge filterer")
	}

	return &Scanner{
		cfg:      cfg,
		log:      log,
		client:   client,
		filterer: filterer,
	}, nil
}

// starts scanning for BridgeInitiated events and sends them to the channel.
func (s *Scanner) Subscribe(ctx context.Context, ch chan<- *abi.TxBridgeBridgeInitiated) error {
	currentBlock, err := s.client.BlockNumber(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get current block")
	}

	safeBlock := currentBlock - s.cfg.Confirmations

	go s.scanHistorical(ctx, s.cfg.StartBlock, safeBlock, ch)
	go s.scanRecent(ctx, safeBlock, ch)

	return nil
}

func (s *Scanner) scanHistorical(ctx context.Context, from, to uint64, ch chan<- *abi.TxBridgeBridgeInitiated) {
	if from >= to {
		return
	}

	s.log.Info("starting BNB historical scan", zap.Uint64("from", from), zap.Uint64("to", to))

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
			s.log.Error("BNB historical scan error", zap.Error(err))
			time.Sleep(s.cfg.PollInterval)
			start -= batchSize // retry the same batch
			continue
		}
	}

	s.log.Info("BNB historical scan completed")
}

func (s *Scanner) scanRecent(ctx context.Context, from uint64, ch chan<- *abi.TxBridgeBridgeInitiated) {
	lastBlock := from
	s.log.Info("starting BNB recent block polling", zap.Uint64("from", from))

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.cfg.PollInterval):
		}

		currentBlock, err := s.client.BlockNumber(ctx)
		if err != nil {
			s.log.Error("failed to get BNB block number", zap.Error(err))
			continue
		}

		safeBlock := currentBlock - s.cfg.Confirmations
		if safeBlock <= lastBlock {
			continue
		}

		count, err := s.scanRangeWithCount(ctx, lastBlock+1, safeBlock, ch)
		if err != nil {
			s.log.Error("BNB recent scan error", zap.Error(err))
			continue
		}

		s.log.Info("polled BNB blocks", zap.Uint64("from", lastBlock+1), zap.Uint64("to", safeBlock), zap.Int("events", count))
		lastBlock = safeBlock
	}
}

func (s *Scanner) scanRange(ctx context.Context, from, to uint64, ch chan<- *abi.TxBridgeBridgeInitiated) error {
	_, err := s.scanRangeWithCount(ctx, from, to, ch)
	return err
}

func (s *Scanner) scanRangeWithCount(ctx context.Context, from, to uint64, ch chan<- *abi.TxBridgeBridgeInitiated) (int, error) {
	iter, err := s.filterer.FilterBridgeInitiated(&bind.FilterOpts{
		Start:   from,
		End:     &to,
		Context: ctx,
	}, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to filter BridgeInitiated events")
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

func (s *Scanner) GetCurrentBlock(ctx context.Context) (uint64, error) {
	return s.client.BlockNumber(ctx)
}

func (s *Scanner) Close() {
	s.client.Close()
}
