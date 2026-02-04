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
	"go.uber.org/zap/zaptest"

	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/bsc/abi"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/metric"
)

func TestBSCFinderBuildPendingTransaction(t *testing.T) {
	t.Parallel()

	setSDKConfig()

	cfg := BSCFinderConfig{
		TXDenom:    "ucore",
		TXDecimals: 6,
	}

	txAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	// Amount in 6 decimals (1.5 tokens = 1500000)
	validEvent := &abi.TXBridgeSentToTXChain{
		From:      common.HexToAddress("0x1234567890123456789012345678901234567890"),
		TxAddress: txAddress.String(),
		Amount:    big.NewInt(1500000), // 1.5 tokens in 6 decimals
		Timestamp: big.NewInt(1234567890),
		Raw: types.Log{
			TxHash: common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"),
		},
	}

	tests := []struct {
		name        string
		eventFunc   func(*abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain
		wantMatches bool
		want        PendingTXSendTransaction
	}{
		{
			name: "positive_valid_event",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
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
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				e.TxAddress = "invalid_address"
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_wrong_prefix_address",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				// devcore prefix instead of core
				e.TxAddress = "devcore17l2fxde2662s2p8pgmzu04jcvflnnlq4l30hff"
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_zero_amount",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				e.Amount = big.NewInt(0)
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_nil_amount",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				e.Amount = nil
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_amount",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				e.Amount = big.NewInt(-1000000)
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_invalid_checksum",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				// Valid format but wrong checksum (changed last char)
				e.TxAddress = "core1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqm9l4xz"
				return e
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "positive_minimum_amount",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				e.Amount = big.NewInt(1) // 1 unit (0.000001 tokens)
				return e
			},
			wantMatches: true,
			want: PendingTXSendTransaction{
				TXDestination: txAddress,
				TXAmount:      sdk.NewCoin(cfg.TXDenom, sdkmath.NewInt(1)),
				XRPLTxHash:    "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},
		{
			name: "positive_large_amount",
			eventFunc: func(e *abi.TXBridgeSentToTXChain) *abi.TXBridgeSentToTXChain {
				e.Amount = big.NewInt(1000000000000) // 1 million tokens
				return e
			},
			wantMatches: true,
			want: PendingTXSendTransaction{
				TXDestination: txAddress,
				TXAmount:      sdk.NewCoin(cfg.TXDenom, sdkmath.NewInt(1000000000000)),
				XRPLTxHash:    "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			metricRecorder, err := metric.NewRecorder()
			require.NoError(t, err)

			finder := NewBSCFinder(cfg, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), nil)

			// Create a copy of the event
			eventCopy := *validEvent
			pendingTx, matches := finder.buildPendingTransaction(tt.eventFunc(&eventCopy))

			require.Equal(t, tt.wantMatches, matches)
			require.Equal(t, tt.want, pendingTx)
		})
	}
}
