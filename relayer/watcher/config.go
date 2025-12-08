package watcher

import (
	"context"
	"sync"
	"time"

	"cosmossdk.io/errors"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/tx"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"go.uber.org/zap"
)

// ErrConfigChanged is returned when a configuration change is detected.
var ErrConfigChanged = errors.New("watcher", 1, "contract configuration has changed")

// ConfigWatcher watches for configuration changes in the smart contract.
type ConfigWatcher struct {
	cfg            Config
	log            logger.Logger
	contractClient *tx.ContractClient

	mu             sync.RWMutex
	currentVersion uint64
}

// NewConfigWatcher creates a new ConfigWatcher instance.
func NewConfigWatcher(
	cfg Config,
	log logger.Logger,
	contractClient *tx.ContractClient,
) *ConfigWatcher {
	return &ConfigWatcher{
		cfg:            cfg,
		log:            log,
		contractClient: contractClient,
	}
}

// Initialize loads the initial configuration version.
func (w *ConfigWatcher) Initialize(ctx context.Context) error {
	config, err := w.contractClient.GetContractConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to query initial contract config version")
	}

	w.mu.Lock()
	w.currentVersion = config.Version
	w.mu.Unlock()

	w.log.Info(
		"Initialized config watcher",
		zap.Uint64("version", config.Version),
	)

	return nil
}

// Watch polls for configuration changes and returns ErrConfigChanged when detected.
// This function blocks until a config change is detected or context is cancelled.
func (w *ConfigWatcher) Watch(ctx context.Context) error {
	w.log.Info("Starting config watcher")

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Config watcher stopped")
			return ctx.Err()

		case <-ticker.C:
			config, err := w.contractClient.GetContractConfig(ctx)
			if err != nil {
				w.log.Error("Failed to query contract config version", zap.Error(err))
				continue
			}

			w.mu.RLock()
			currentVersion := w.currentVersion
			w.mu.RUnlock()

			if config.Version != currentVersion {
				w.log.Info(
					"Detected contract config change",
					zap.Uint64("old_version", currentVersion),
					zap.Uint64("new_version", config.Version),
				)
				return ErrConfigChanged
			}
		}
	}
}

// GetCurrentVersion returns the current configuration version.
func (w *ConfigWatcher) GetCurrentVersion() uint64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.currentVersion
}
