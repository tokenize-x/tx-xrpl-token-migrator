package finder

import (
	"fmt"
	"math/big"
	"sync"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/CoreumFoundation/coreum/pkg/config"
	"github.com/CoreumFoundation/coreum/pkg/config/constant"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/client/xrpl"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/logger"
	"github.com/CoreumFoundation/xrpl-bridge/relayer/metric"
)

var (
	once         = sync.Once{}
	setSDKConfig = func() {
		once.Do(func() {
			network, err := config.NetworkConfigByChainID(constant.ChainIDMain)
			if err != nil {
				panic(err)
			}
			network.SetSDKConfig()
		})
	}
)

func TestBuildPendingTransaction(t *testing.T) { //nolint:funlen // a lot of test cases
	t.Parallel()

	setSDKConfig()

	cfg := Config{
		XRPLIssuer:                 "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D",
		XRPLCurrency:               "434F524500000000000000000000000000000000",
		XRPLHistoryScanStartLedger: 8000,
		XRPLMemoSuffix:             "=cored",
		CoreumDenom:                "ucore",
		CoreumDecimals:             6,
	}

	coreumAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	validXRPLTransaction := xrpl.Transaction{
		DeliveryAmount: xrpl.DeliveredAmount{
			Currency: cfg.XRPLCurrency,
			Issuer:   cfg.XRPLIssuer,
			Value: func() *big.Float {
				v, _ := big.NewFloat(0).SetString("1.23456789")
				return v
			}(),
		},
		Memos:             []string{fmt.Sprintf("none-address%s", cfg.XRPLMemoSuffix), fmt.Sprintf("%s%s", coreumAddress, cfg.XRPLMemoSuffix), "any-string"},
		Hash:              "xrpl-tx-hash",
		TransactionType:   xrpl.TransactionTypePayment,
		TransactionResult: xrpl.TransactionResultSuccess,
		Validated:         true,
	}

	tests := []struct {
		name        string
		xrplTxFunc  func(xrpl.Transaction) xrpl.Transaction
		wantMatches bool
		want        PendingCoreumSendTransaction
	}{
		{
			name: "positive",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				return tx
			},
			wantMatches: true,
			want: PendingCoreumSendTransaction{
				CoreumDestination: coreumAddress,
				CoreumAmount:      sdk.NewInt64Coin(cfg.CoreumDenom, 1234567),
				XRPLTxHash:        "xrpl-tx-hash",
			},
		},
		{
			name: "negative_invalid_tx_validated",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Validated = false
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "negative_invalid_tx_result",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.TransactionType = "fail"
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "negative_invalid_tx_type",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.TransactionType = "AccountSet"
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "invalid_memo",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Memos = []string{"invalid-memo"}
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "invalid_address_prefx_memo",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Memos = []string{fmt.Sprintf("devcore17l2fxde2662s2p8pgmzu04jcvflnnlq4l30hff%s", cfg.XRPLMemoSuffix)}
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "invalid_currency",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Currency = "invalid"
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "invalid_issuer",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Issuer = "invalid"
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "zero_amount",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Value = func() *big.Float {
					v, _ := big.NewFloat(0).SetString("0.0000001")
					return v
				}()
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "negative_empty",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				return xrpl.Transaction{}
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			metricRecorder, err := metric.NewRecorder(metric.RecorderConfig{})
			require.NoError(t, err)
			finder := NewFinder(cfg, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), nil)
			pendingTx, matches := finder.buildPendingTransaction(tt.xrplTxFunc(validXRPLTransaction))
			require.Equal(t, tt.want, pendingTx)
			require.Equal(t, tt.wantMatches, matches)
		})
	}
}
