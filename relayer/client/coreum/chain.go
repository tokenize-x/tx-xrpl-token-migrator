package coreum

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	ctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/gammazero/workerpool"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum/v3/pkg/client"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// ChainClientConfig represent the ChainClient config.
type ChainClientConfig struct {
	EventsPageSize int
	WorkerPoolSize int
	RequestTimeout time.Duration
}

// DefaultChainClientConfig returns default ChainClient config.
func DefaultChainClientConfig() ChainClientConfig {
	return ChainClientConfig{
		EventsPageSize: 100,
		WorkerPoolSize: 10,
		RequestTimeout: time.Minute,
	}
}

// ChainClient is the coreum chain client.
type ChainClient struct {
	cfg        ChainClientConfig
	log        logger.Logger
	clientCtx  client.Context
	workerPool *workerpool.WorkerPool
}

// NewChainClient returns new instance of ChainClient.
func NewChainClient(cfg ChainClientConfig, log logger.Logger, clientCtx client.Context) *ChainClient {
	return &ChainClient{
		cfg:        cfg,
		log:        log,
		clientCtx:  clientCtx,
		workerPool: workerpool.New(cfg.WorkerPoolSize),
	}
}

// GetSpendingTransactions returns all txs which spends coins from and address.
func (c *ChainClient) GetSpendingTransactions(ctx context.Context, fromAddress string) ([]*sdk.TxResponse, error) {
	return c.queryTransactionsByEvents(ctx, fmt.Sprintf("coin_spent.spender='%s'", fromAddress))
}

func (c *ChainClient) queryTransactionsByEvents(ctx context.Context, event string) ([]*sdk.TxResponse, error) {
	c.log.Info("Fetching coreum transactions.", zap.String("event", event))

	tmEvents := []string{event}
	// call fast to get total pages
	reqCtx, reqCtxCancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
	defer reqCtxCancel()
	res, err := queryTxsByEvents(reqCtx, c.clientCtx, tmEvents, 1, 1, "asc")
	if err != nil {
		return nil, err
	}
	// compute pages length
	pagesTotal := int(res.PageTotal) / c.cfg.EventsPageSize
	remainder := int(res.PageTotal) % c.cfg.EventsPageSize
	if remainder != 0 {
		pagesTotal++
	}

	txs := make([]*sdk.TxResponse, 0)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	var fetchError error
	wg.Add(pagesTotal)
	for page := 1; page <= pagesTotal; page++ {
		pageToFetch := page
		c.workerPool.Submit(func() {
			defer wg.Done()
			c.log.Debug("Fetching page", zap.String("Page", fmt.Sprintf("%d/%d", pageToFetch, pagesTotal)))
			defer c.log.Debug("Fetchedpage ", zap.String("Page", fmt.Sprintf("%d/%d", pageToFetch, pagesTotal)))
			reqCtx, reqCtxCancel := context.WithTimeout(ctx, c.cfg.RequestTimeout)
			defer reqCtxCancel()
			res, err = queryTxsByEvents(reqCtx, c.clientCtx, tmEvents, pageToFetch, c.cfg.EventsPageSize, "asc")
			if err != nil {
				fetchError = err
				c.log.Error(
					"Failed to fetch page",
					zap.String("Page", fmt.Sprintf("%d/%d", pageToFetch, pagesTotal)),
					zap.Error(err),
				)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, tx := range txs {
				// keep success transactions only
				if tx.Code != 0 {
					continue
				}
			}
			txs = append(txs, res.Txs...)
		})
	}
	wg.Wait()
	if fetchError != nil {
		return nil, fetchError
	}

	c.log.Debug("Found coreum transactions.", zap.Int("count", len(txs)))

	return txs, nil
}

func queryTxsByEvents(
	ctx context.Context,
	clientCtx client.Context,
	events []string,
	page, limit int,
	orderBy string,
) (*sdk.SearchTxsResult, error) {
	query := strings.Join(events, " AND ")

	node := clientCtx.RPCClient()
	if node == nil {
		return nil, errors.Errorf("the RPC node is not initialized")
	}

	resTxs, err := node.TxSearch(ctx, query, true, &page, &limit, orderBy)
	if err != nil {
		return nil, err
	}

	txs, err := formatTxResults(clientCtx.TxConfig(), resTxs.Txs)
	if err != nil {
		return nil, err
	}

	result := sdk.NewSearchTxsResult(uint64(resTxs.TotalCount), uint64(len(txs)), uint64(page), uint64(limit), txs)

	return result, nil
}

func formatTxResults(txConfig sdkclient.TxConfig, resTxs []*ctypes.ResultTx) ([]*sdk.TxResponse, error) {
	var err error
	out := make([]*sdk.TxResponse, len(resTxs))
	for i := range resTxs {
		out[i], err = mkTxResult(txConfig, resTxs[i])
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

func mkTxResult(txConfig sdkclient.TxConfig, resTx *ctypes.ResultTx) (*sdk.TxResponse, error) {
	txb, err := txConfig.TxDecoder()(resTx.Tx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode tx")
	}
	type intoAny interface {
		AsAny() *codectypes.Any
	}
	p, ok := txb.(intoAny)
	if !ok {
		return nil, errors.Errorf("expecting a type implementing intoAny, got: %T", txb)
	}
	return sdk.NewResponseResultTx(resTx, p.AsAny(), ""), nil
}
