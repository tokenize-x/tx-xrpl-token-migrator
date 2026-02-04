package metric

import (
	"context"
	"net/http"
	"time"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
)

const instanceNameLabel = "instance"

// PusherConfig defines Pusher config.
type PusherConfig struct {
	URL                string
	JobName            string
	InstanceName       string
	RequestTimeout     time.Duration
	PushDelay          time.Duration
	Username, Password string
}

// DefaultPusherConfig returns default Pusher config.
func DefaultPusherConfig(url, username, password, instanceName string) PusherConfig {
	return PusherConfig{
		URL:            url,
		JobName:        "bridge",
		InstanceName:   instanceName,
		RequestTimeout: 10 * time.Second,
		PushDelay:      5 * time.Second,
		Username:       username,
		Password:       password,
	}
}

// Pusher is a metric pusher.
type Pusher struct {
	cfg    PusherConfig
	log    logger.Logger
	pusher *push.Pusher
}

// NewPusher returns a new instance of the Pusher.
func NewPusher(cfg PusherConfig, log logger.Logger, registry *prometheus.Registry) (*Pusher, error) {
	httpClient := &http.Client{
		Timeout: cfg.RequestTimeout,
	}

	pusher := push.New(cfg.URL, cfg.JobName).
		BasicAuth(cfg.Username, cfg.Password).
		Gatherer(registry).
		Client(httpClient).
		Grouping(instanceNameLabel, cfg.InstanceName)

	return &Pusher{
		cfg:    cfg,
		log:    log,
		pusher: pusher,
	}, nil
}

// PushMetrics pushes metrics to Prometheus.
func (p *Pusher) PushMetrics(ctx context.Context) error {
	err := retry.Do(ctx, p.cfg.PushDelay, func() error {
		if err := p.pusher.Add(); err != nil {
			p.log.Error("Failed to push metrics", zap.Error(err))
			return retry.Retryable(err)
		}

		return retry.Retryable(errors.New("repeat push"))
	})
	if err == nil || errors.Is(err, context.Canceled) {
		return err
	}
	panic(errors.Wrap(err, "unexpected error in push with retry"))
}
