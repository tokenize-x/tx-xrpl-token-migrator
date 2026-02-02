package metric

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	prometheusdto "github.com/prometheus/client_model/go"
)

// Recorder is metrics recorder.
type Recorder struct {
	registry *prometheus.Registry

	txSenderBalanceGauge                 prometheus.Gauge
	txContractBalanceGauge               prometheus.Gauge
	txPendingUnapprovedTransactionsCount prometheus.Gauge
	txPendingApprovedTransactionsCount   prometheus.Gauge

	xrplLatestLedgerIndexGauge prometheus.Gauge
	xrplLatestLedgerIndex      int64
	xrplLatestLedgerIndexMu    sync.Mutex

	bscLatestProcessedBlockGauge prometheus.Gauge
	bscLatestProcessedBlock      uint64
	bscLatestProcessedBlockMu    sync.Mutex
	bscChainHeadBlockGauge       prometheus.Gauge

	errorsCounter prometheus.Counter
}

// NewRecorder returns a new instance of the Recorder.
func NewRecorder() (*Recorder, error) {
	registry := prometheus.NewRegistry()

	startTimeGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "start_time",
		Help: "Start time of the application",
	})
	startTimeGauge.Set(float64(time.Now().Unix()))
	if err := registry.Register(startTimeGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register start time metric")
	}

	txSenderBalanceGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tx_sender_balance",
		Help: "TX sender balance",
	})
	if err := registry.Register(txSenderBalanceGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register tx sender balance gauge")
	}

	txContractBalanceGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tx_contract_balance",
		Help: "TX contract balance",
	})

	if err := registry.Register(txContractBalanceGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register tx contract balance gauge")
	}

	xrplLatestLedgerIndexGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xrpl_latest_account_ledger_index",
		Help: "Latest observer XRPL account ledger index",
	})
	if err := registry.Register(xrplLatestLedgerIndexGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register xrpl latest ledger index gauge")
	}

	bscLatestProcessedBlockGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bsc_latest_processed_block",
		Help: "Latest processed BSC block number",
	})
	if err := registry.Register(bscLatestProcessedBlockGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register bsc latest processed block gauge")
	}

	bscChainHeadBlockGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "bsc_chain_head_block",
		Help: "Current BSC chain head block number",
	})
	if err := registry.Register(bscChainHeadBlockGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register bsc chain head block gauge")
	}

	errorsCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "errors_total",
		Help: "Errors counter",
	})
	if err := registry.Register(errorsCounter); err != nil {
		return nil, errors.Wrapf(err, "failed to register errors сounter")
	}

	//nolint:promlinter // the name is expected
	txPendingUnapprovedTransactionsTotal := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tx_pending_unapproved_transactions_count",
		Help: "TX pending unapproved transactions count",
	})
	if err := registry.Register(txPendingUnapprovedTransactionsTotal); err != nil {
		return nil, errors.Wrapf(err, "failed to register xrpl TX pending unapproved transactions count gauge")
	}

	//nolint:promlinter // the name is expected
	txPendingApprovedTransactionsTotal := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tx_pending_approved_transactions_count",
		Help: "TX pending approved transactions count",
	})
	if err := registry.Register(txPendingApprovedTransactionsTotal); err != nil {
		return nil, errors.Wrapf(err, "failed to register xrpl TX pending approved transactions count gauge")
	}

	return &Recorder{
		registry:                             registry,
		txSenderBalanceGauge:                 txSenderBalanceGauge,
		txContractBalanceGauge:               txContractBalanceGauge,
		txPendingUnapprovedTransactionsCount: txPendingUnapprovedTransactionsTotal,
		txPendingApprovedTransactionsCount:   txPendingApprovedTransactionsTotal,
		xrplLatestLedgerIndexGauge:           xrplLatestLedgerIndexGauge,
		xrplLatestLedgerIndex:                0,
		xrplLatestLedgerIndexMu:              sync.Mutex{},
		bscLatestProcessedBlockGauge:         bscLatestProcessedBlockGauge,
		bscLatestProcessedBlock:              0,
		bscLatestProcessedBlockMu:            sync.Mutex{},
		bscChainHeadBlockGauge:               bscChainHeadBlockGauge,
		errorsCounter:                        errorsCounter,
	}, nil
}

// GetRegistry returns metrics registry.
func (r *Recorder) GetRegistry() *prometheus.Registry {
	return r.registry
}

// SetTXSenderBalance sets TX sender balance metric.
func (r *Recorder) SetTXSenderBalance(v int64) {
	r.txSenderBalanceGauge.Set(float64(v))
}

// SetTXContractBalance sets TX contract balance metric.
func (r *Recorder) SetTXContractBalance(v int64) {
	r.txContractBalanceGauge.Set(float64(v))
}

// SetXRPLLatestAccountLedgerIndex sets latest xrpl ledger index metric to v if v is greater than current value.
func (r *Recorder) SetXRPLLatestAccountLedgerIndex(v int64) {
	r.xrplLatestLedgerIndexMu.Lock()
	defer r.xrplLatestLedgerIndexMu.Unlock()
	if v < r.xrplLatestLedgerIndex {
		return
	}
	r.xrplLatestLedgerIndexGauge.Set(float64(v))
}

func (r *Recorder) SetBSCLatestProcessedBlock(v uint64) {
	r.bscLatestProcessedBlockMu.Lock()
	defer r.bscLatestProcessedBlockMu.Unlock()
	if v < r.bscLatestProcessedBlock {
		return
	}
	r.bscLatestProcessedBlock = v
	r.bscLatestProcessedBlockGauge.Set(float64(v))
}

func (r *Recorder) SetBSCChainHeadBlock(v uint64) {
	r.bscChainHeadBlockGauge.Set(float64(v))
}

// IncrementError increments error metric.
func (r *Recorder) IncrementError() {
	r.errorsCounter.Inc()
}

// GetTotalErrors returns current errors counter value.
func (r *Recorder) GetTotalErrors() (float64, error) {
	metric := &prometheusdto.Metric{}
	if err := r.errorsCounter.Write(metric); err != nil {
		return 0, err
	}

	return metric.GetCounter().GetValue(), nil
}

// SetTXPendingUnapprovedTransactionsCount sets TX contract pending unapproved transactions count.
func (r *Recorder) SetTXPendingUnapprovedTransactionsCount(v int) {
	r.txPendingUnapprovedTransactionsCount.Set(float64(v))
}

// SetTXPendingApprovedTransactionsCount sets TX contract pending approved transactions count.
func (r *Recorder) SetTXPendingApprovedTransactionsCount(v int) {
	r.txPendingApprovedTransactionsCount.Set(float64(v))
}
