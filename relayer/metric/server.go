package metric

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/CoreumFoundation/coreum-tools/pkg/retry"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
)

// ServerConfig is Server config.
type ServerConfig struct {
	Port                          int
	ShutdownTimeout               time.Duration
	ListenAndServerRestartTimeout time.Duration
}

// DefaultServerConfig returns Server default config.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:                          8222,
		ShutdownTimeout:               5 * time.Second,
		ListenAndServerRestartTimeout: 5 * time.Second,
	}
}

// Server is a metric server.
type Server struct {
	cfg      ServerConfig
	log      logger.Logger
	registry *prometheus.Registry
}

// NewServer returns a new instance of the Server.
func NewServer(cfg ServerConfig, log logger.Logger, registry *prometheus.Registry) *Server {
	return &Server{
		cfg:      cfg,
		log:      log,
		registry: registry,
	}
}

// Start starts web-server which exposes the metrics.
func (s *Server) Start(ctx context.Context) {
	s.log.Info("Starting metrics web server", zap.Int("port", s.cfg.Port))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{}),
	}
	go func() {
		go func() {
			err := retry.Do(ctx, s.cfg.ListenAndServerRestartTimeout, func() error {
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					s.log.Error("Unexpected stop of metrics server serve.", zap.Error(err))
					return retry.Retryable(errors.New("restarting listen and serve"))
				}
				return nil
			})
			// unexpected error
			if err != nil && !errors.Is(err, context.Canceled) {
				panic(err)
			}
		}()

		<-ctx.Done()
		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer shutdownCtxCancel()
		if err := server.Shutdown(shutdownCtx); err != nil { //nolint:contextcheck // root ctx is canceled
			s.log.Error("Failed to gracefully shutdown the metrics web server", zap.Error(err))
		}
	}()
}
