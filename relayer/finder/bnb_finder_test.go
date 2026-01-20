package finder

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bnb/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/metric"
	"go.uber.org/zap/zaptest"
)

func TestConvertBNBAmountToTXCoin(t *testing.T) {
	const denom = "ucore"
	tests := []struct {
		name       string
		weiAmount  *big.Int
		txDecimals int
		wantAmount sdk.Coin
	}{
		{
			name:       "1 token (18 decimals to 6 decimals)",
			weiAmount:  big.NewInt(1000000000000000000), // 1e18
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.NewInt(1000000)), // 1e6
		},
		{
			name:       "1.5 tokens",
			weiAmount:  big.NewInt(1500000000000000000), // 1.5e18
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.NewInt(1500000)), // 1.5e6
		},
		{
			name:       "0.000001 tokens (minimum TX unit)",
			weiAmount:  big.NewInt(1000000000000), // 1e12 wei = 0.000001 tokens
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.NewInt(1)), // 1 ucore
		},
		{
			name:       "below minimum (truncated to 0)",
			weiAmount:  big.NewInt(999999999999), // just below 1e12
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.NewInt(0)),
		},
		{
			name:       "large amount 1 million tokens",
			weiAmount:  new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1000000000000000000)),
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.NewInt(1000000000000)), // 1e12
		},
		{
			name:       "nil amount",
			weiAmount:  nil,
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.ZeroInt()),
		},
		{
			name:       "zero amount",
			weiAmount:  big.NewInt(0),
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.ZeroInt()),
		},
		{
			name:       "negative amount",
			weiAmount:  big.NewInt(-1000000000000000000),
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.ZeroInt()),
		},
		{
			name:       "fractional amount with truncation",
			weiAmount:  big.NewInt(1234567890123456789), // ~1.234567890123456789 tokens
			txDecimals: 6,
			wantAmount: sdk.NewCoin(denom, sdkmath.NewInt(1234567)), // truncated to 6 decimals
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertBNBAmountToTXCoin(tt.weiAmount, denom, tt.txDecimals)
			require.Equal(t, tt.wantAmount.String(), got.String())
		})
	}
}

func TestBNBFinderBuildPendingTransaction(t *testing.T) {
	t.Parallel()

	setSDKConfig()

	cfg := BNBFinderConfig{
		ChainSuffix: "/coreum-mainnet-1/v1",
		TXDenom:     "ucore",
		TXDecimals:  6,
	}

	txAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	validEvent := &abi.TxBridgeBridgeInitiated{
		From:               common.HexToAddress("0x1234567890123456789012345678901234567890"),
		DestinationPayload: txAddress.String() + cfg.ChainSuffix,
		Amount:             big.NewInt(1500000000000000000), // 1.5 tokens
		Timestamp:          big.NewInt(1234567890),
		Raw: types.Log{
			TxHash: common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		},
	}

	tests := []struct {
		name        string
		eventFunc   func(*abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated
		wantMatches bool
		want        PendingTXSendTransaction
	}{
		{
			name: "positive_valid_event",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				return e
			},
			wantMatches: true,
			want: PendingTXSendTransaction{
				TXDestination: txAddress,
				TXAmount:      sdk.NewCoin(cfg.TXDenom, sdkmath.NewInt(1500000)),
				XRPLTxHash:    "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},
		{
			name: "negative_invalid_bech32_address",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				e.DestinationPayload = "invalid_address" + cfg.ChainSuffix
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_wrong_prefix_address",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				// devcore prefix instead of core
				e.DestinationPayload = "devcore17l2fxde2662s2p8pgmzu04jcvflnnlq4l30hff" + cfg.ChainSuffix
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_zero_amount",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				e.Amount = big.NewInt(0)
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_nil_amount",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				e.Amount = nil
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_amount_below_min_unit",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				e.Amount = big.NewInt(999999999999) // below 1e12 (minimum 1 ucore)
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_missing_chain_suffix",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				e.DestinationPayload = txAddress.String() // no suffix
				return e
			},
			wantMatches: true, // TrimSuffix still works, address is valid
			want: PendingTXSendTransaction{
				TXDestination: txAddress,
				TXAmount:      sdk.NewCoin(cfg.TXDenom, sdkmath.NewInt(1500000)),
				XRPLTxHash:    "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},
		{
			name: "negative_invalid_checksum",
			eventFunc: func(e *abi.TxBridgeBridgeInitiated) *abi.TxBridgeBridgeInitiated {
				// Valid format but wrong checksum (changed last char)
				e.DestinationPayload = "core1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqm9l4xz" + cfg.ChainSuffix
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metricRecorder, err := metric.NewRecorder()
			require.NoError(t, err)

			finder := NewBNBFinder(cfg, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), nil)

			// Create a copy of the event
			eventCopy := *validEvent
			pendingTx, matches := finder.buildPendingTransaction(tt.eventFunc(&eventCopy))

			require.Equal(t, tt.wantMatches, matches)
			require.Equal(t, tt.want, pendingTx)
		})
	}
}

func TestExtractAddressFromDestinationPayload(t *testing.T) {
	tests := []struct {
		name               string
		destinationPayload string
		chainIDSuffix      string
		expectedAddress    string
	}{
		{
			name:               "valid_with_suffix",
			destinationPayload: "core1abc123xyz/coreum-mainnet-1",
			chainIDSuffix:      "/coreum-mainnet-1",
			expectedAddress:    "core1abc123xyz",
		},
		{
			name:               "no_suffix_in_address",
			destinationPayload: "core1abc123xyz",
			chainIDSuffix:      "/coreum-mainnet-1",
			expectedAddress:    "core1abc123xyz",
		},
		{
			name:               "empty_suffix",
			destinationPayload: "core1abc123xyz/coreum-mainnet-1",
			chainIDSuffix:      "",
			expectedAddress:    "core1abc123xyz/coreum-mainnet-1",
		},
		{
			name:               "testnet_suffix",
			destinationPayload: "testcore1abc123xyz/coreum-testnet-1",
			chainIDSuffix:      "/coreum-testnet-1",
			expectedAddress:    "testcore1abc123xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAddressFromDestinationPayload(tt.destinationPayload, tt.chainIDSuffix)
			require.Equal(t, tt.expectedAddress, got)
		})
	}
}
