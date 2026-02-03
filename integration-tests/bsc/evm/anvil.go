//go:build integrationtests

package evm

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os/exec"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
)

const (
	// default port for Anvil.
	DefaultAnvilPort = 8545

	// default chain ID for Anvil.
	DefaultAnvilChainID = 31337

	// block time for Anvil
	AnvilBlockTime = 1
)

// pre-funded test accounts (first 10 accounts with 10000 ETH each).
// these are deterministic and always the same in Anvil.
var (
	// private keys for pre-funded accounts.
	AnvilPrivateKeys = []string{
		"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		"59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
		"5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a",
		"7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
		"47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
	}

	// addresses corresponding to the private keys.
	AnvilAddresses = []string{
		"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		"0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
		"0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
		"0x90F79bf6EB2c4f870365E785982E1f101E93b906",
		"0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
	}
)

type AnvilConfig struct {
	Port      int
	ChainID   int64
	BlockTime int
}

func DefaultAnvilConfig() AnvilConfig {
	return AnvilConfig{
		Port:      DefaultAnvilPort,
		ChainID:   DefaultAnvilChainID,
		BlockTime: AnvilBlockTime,
	}
}

type Anvil struct {
	cfg    AnvilConfig
	cmd    *exec.Cmd
	rpcURL string
}

func StartAnvil(cfg AnvilConfig) (*Anvil, error) {
	rpcURL := fmt.Sprintf("http://localhost:%d", cfg.Port)

	args := []string{
		"--port", fmt.Sprintf("%d", cfg.Port),
		"--chain-id", fmt.Sprintf("%d", cfg.ChainID),
	}

	if cfg.BlockTime > 0 {
		args = append(args, "--block-time", fmt.Sprintf("%d", cfg.BlockTime))
	}

	cmd := exec.Command("anvil", args...)

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrap(err, "failed to start anvil")
	}

	anvil := &Anvil{
		cfg:    cfg,
		cmd:    cmd,
		rpcURL: rpcURL,
	}

	if err := anvil.waitReady(10 * time.Second); err != nil {
		anvil.Stop()
		return nil, errors.Wrap(err, "anvil failed to start")
	}

	return anvil, nil
}

// stops the Anvil instance.
func (a *Anvil) Stop() error {
	if a.cmd != nil && a.cmd.Process != nil {
		if err := a.cmd.Process.Kill(); err != nil {
			return errors.Wrap(err, "failed to kill anvil process")
		}
		// Wait for process to exit to avoid zombies
		_ = a.cmd.Wait()
	}
	return nil
}

// returns an ethclient connected to Anvil.
func (a *Anvil) Client() (*ethclient.Client, error) {
	return ethclient.Dial(a.rpcURL)
}

// returns the RPC URL for Anvil.
func (a *Anvil) RPCURL() string {
	return a.rpcURL
}

// returns the chain ID as *big.Int.
func (a *Anvil) ChainID() *big.Int {
	return big.NewInt(a.cfg.ChainID)
}

// waits for Anvil to be ready to accept connections.
func (a *Anvil) waitReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout waiting for anvil to be ready")
		case <-ticker.C:
			client, err := ethclient.Dial(a.rpcURL)
			if err != nil {
				continue
			}
			_, err = client.ChainID(ctx)
			client.Close()
			if err == nil {
				return nil
			}
		}
	}
}

// returns the private key for the given account index.
func GetPrivateKey(accountIndex int) (*ecdsa.PrivateKey, error) {
	if accountIndex < 0 || accountIndex >= len(AnvilPrivateKeys) {
		return nil, errors.Errorf("account index %d out of range (0-%d)", accountIndex, len(AnvilPrivateKeys)-1)
	}
	return crypto.HexToECDSA(AnvilPrivateKeys[accountIndex])
}

// returns the address for the given account index.
func GetAddress(accountIndex int) (string, error) {
	if accountIndex < 0 || accountIndex >= len(AnvilAddresses) {
		return "", errors.Errorf("account index %d out of range (0-%d)", accountIndex, len(AnvilAddresses)-1)
	}
	return AnvilAddresses[accountIndex], nil
}
