package metric

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// Recorder is metrics recorder.
type Recorder struct {
	registry *prometheus.Registry

	coreumSenderBalanceGauge   prometheus.Gauge
	coreumContractBalanceGauge prometheus.Gauge

	xrplLatestLedgerIndexGauge prometheus.Gauge
	xrplLatestLedgerIndex      int64

	errorsCounter prometheus.Counter

	mu sync.Mutex
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

	coreumSenderBalanceGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coreum_sender_balance",
		Help: "Coreum sender balance",
	})
	if err := registry.Register(coreumSenderBalanceGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register coreum sender balance gauge")
	}

	coreumContractBalanceGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "coreum_contract_balance",
		Help: "Coreum contract balance",
	})

	if err := registry.Register(coreumContractBalanceGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register coreum contract balance gauge")
	}

	xrplLatestLedgerIndexGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "xrpl_latest_account_ledger_index",
		Help: "Latest observer XRPL account ledger index",
	})
	if err := registry.Register(xrplLatestLedgerIndexGauge); err != nil {
		return nil, errors.Wrapf(err, "failed to register xrpl latest ledger index gauge")
	}

	errorsCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "errors_total",
		Help: "Errors counter",
	})
	if err := registry.Register(errorsCounter); err != nil {
		return nil, errors.Wrapf(err, "failed to register errors сounter")
	}

	return &Recorder{
		registry:                   registry,
		coreumSenderBalanceGauge:   coreumSenderBalanceGauge,
		coreumContractBalanceGauge: coreumContractBalanceGauge,
		xrplLatestLedgerIndexGauge: xrplLatestLedgerIndexGauge,
		xrplLatestLedgerIndex:      0,

		errorsCounter: errorsCounter,
		mu:            sync.Mutex{},
	}, nil
}

// GetRegistry returns metrics registry.
func (r *Recorder) GetRegistry() *prometheus.Registry {
	return r.registry
}

// SetCoreumSenderBalance sets coreum sender balance metric.
func (r *Recorder) SetCoreumSenderBalance(v int64) {
	r.coreumSenderBalanceGauge.Set(float64(v))
}

// SetCoreumContractBalance sets coreum contract balance metric.
func (r *Recorder) SetCoreumContractBalance(v int64) {
	r.coreumContractBalanceGauge.Set(float64(v))
}

// SetXRPLLatestAccountLedgerIndex sets latest xrpl ledger index metric to v if v is greater than current value.
func (r *Recorder) SetXRPLLatestAccountLedgerIndex(v int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v < r.xrplLatestLedgerIndex {
		return
	}
	r.xrplLatestLedgerIndexGauge.Set(float64(v))
}

// IncrementError increments error metric.
func (r *Recorder) IncrementError() {
	r.errorsCounter.Inc()
}
