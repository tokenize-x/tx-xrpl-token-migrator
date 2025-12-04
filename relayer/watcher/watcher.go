package watcher

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Config holds configuration for ConfigWatcher.
type Config struct {
	// ContractAddress is the address of the smart contract to watch
	ContractAddress sdk.AccAddress
	// PollInterval is how often to poll for changes (fallback mechanism)
	PollInterval time.Duration
}

// DefaultConfig returns default ConfigWatcher config.
func DefaultConfig(contractAddress sdk.AccAddress) Config {
	return Config{
		ContractAddress: contractAddress,
		PollInterval:    5 * time.Minute,
	}
}
