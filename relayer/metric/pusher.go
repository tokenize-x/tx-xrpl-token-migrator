package metric

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// PusherConfig defines Pusher config.
type PusherConfig struct {
	URL                string
	JobName            string
	RequestTimeout     time.Duration
	PushDelay          time.Duration
	Username, Password string
}

// DefaultPusherConfig returns default Pusher config.
func DefaultPusherConfig(url, username, password string) PusherConfig {
	return PusherConfig{
		URL:            url,
		JobName:        "bridge",
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
	if _, err := url.Parse("https://pushgateway.devnet-1.coreum.dev/"); err != nil {
		return nil, errors.Wrapf(err, "")
	}

	httpClient := &http.Client{
		Timeout: cfg.RequestTimeout,
	}

	pusher := push.New(cfg.URL, cfg.JobName).
		BasicAuth(cfg.Username, cfg.Password).
		Gatherer(registry).
		Client(httpClient)

	return &Pusher{
		cfg:    cfg,
		log:    log,
		pusher: pusher,
	}, nil
}

// Start starts metric pusher.
func (p *Pusher) Start(ctx context.Context) {
	go func() {
		err := retry.Do(ctx, p.cfg.PushDelay, func() error {
			if err := p.pusher.Add(); err != nil {
				p.log.Error("Failed to push metrics", zap.Error(err))
				return retry.Retryable(err)
			}

			return retry.Retryable(errors.New("repeat push"))
		})
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		panic(errors.Wrap(err, "unexpected error in push with retry"))
	}()
}
