package finder

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/CoreumFoundation/coreum/v4/pkg/config"
	"github.com/CoreumFoundation/coreum/v4/pkg/config/constant"
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

func TestBuildPendingTransaction(t *testing.T) {
	t.Parallel()

	setSDKConfig()

	cfg := Config{
		XRPLIssuer:                 convertStringToRippleAccount(t, "rcoreNywaoz2ZCQ8Lg2EbSLnGuRBmun6D"),
		XRPLCurrency:               convertStringToRippleCurrency(t, "434F524500000000000000000000000000000000"),
		ActivationDate:             time.Date(2000, 5, 1, 0, 0, 0, 0, time.UTC),
		XRPLHistoryScanStartLedger: 8000,
		XRPLMemoSuffix:             "=cored",
		CoreumDenom:                "ucore",
		CoreumDecimals:             6,
	}

	coreumAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	validXRPLTransaction := xrpl.Transaction{
		DeliveryAmount: rippledata.Amount{
			Currency: cfg.XRPLCurrency,
			Issuer:   cfg.XRPLIssuer,
			Value:    convertStringToRippleValue(t, "1.23456789", false),
		},
		Memos: []string{
			fmt.Sprintf("none-address%s", cfg.XRPLMemoSuffix),
			fmt.Sprintf("%s%s", coreumAddress, cfg.XRPLMemoSuffix),
			"any-string",
		},
		Hash:              "xrpl-tx-hash",
		TransactionType:   xrpl.TransactionTypePayment,
		TransactionResult: xrpl.TransactionResultSuccess,
		Validated:         true,
		Date:              time.Date(2000, 6, 1, 0, 0, 0, 0, time.UTC),
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
			name: "negative_tx_before_activation_date",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Date = time.Date(2000, 4, 1, 0, 0, 0, 0, time.UTC)
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
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
				tx.DeliveryAmount.Currency = convertStringToRippleCurrency(t, "IND")
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "invalid_issuer",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Issuer = convertStringToRippleAccount(t, rippledata.Account{}.String())
				return tx
			},
			wantMatches: false,
			want:        PendingCoreumSendTransaction{},
		},
		{
			name: "zero_amount",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Value = convertStringToRippleValue(t, "0.0000001", false)
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

			metricRecorder, err := metric.NewRecorder()
			require.NoError(t, err)
			finder := NewFinder(cfg, logger.NewZapLogger(zaptest.NewLogger(t), metricRecorder), nil)
			pendingTx, matches := finder.buildPendingTransaction(tt.xrplTxFunc(validXRPLTransaction))
			require.Equal(t, tt.want, pendingTx)
			require.Equal(t, tt.wantMatches, matches)
		})
	}
}

func TestFinder_convertXRPLAmountToCoreumCoin(t *testing.T) {
	const denom = "ucore"
	tests := []struct {
		name       string
		xrplAmount *rippledata.Value
		wantAmount sdk.Coin
	}{
		{
			name:       "no_truncation",
			xrplAmount: convertStringToRippleValue(t, "10.123456", false),
			wantAmount: sdk.NewCoin(denom, sdk.NewInt(10123456)),
		},
		{
			name:       "max_amount",
			xrplAmount: convertStringToRippleValue(t, "1000000000", false),
			wantAmount: sdk.NewCoin(denom, func() sdkmath.Int {
				v, _ := sdk.NewIntFromString("1000000000000000")
				return v
			}()),
		},
		{
			name:       "many_decimals",
			xrplAmount: convertStringToRippleValue(t, "0.100001000000001", false),
			wantAmount: sdk.NewInt64Coin(denom, 100001),
		},
		{
			name:       "many_decimals_to_zero",
			xrplAmount: convertStringToRippleValue(t, "0.000000000000001", false),
			wantAmount: sdk.NewInt64Coin(denom, 0),
		},
		{
			name:       "default_float_rounding",
			xrplAmount: convertStringToRippleValue(t, "0.001", false),
			wantAmount: sdk.NewInt64Coin(denom, 1000),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &Finder{
				cfg: Config{
					CoreumDenom:    denom,
					CoreumDecimals: 6,
				},
			}
			if got := f.convertXRPLAmountToCoreumCoin(tt.xrplAmount); !reflect.DeepEqual(got.String(), tt.wantAmount.String()) {
				t.Errorf("convertXRPLAmountToCoreumCoin() = %v, want %v", got.String(), tt.wantAmount.String())
			}
		})
	}
}

func convertStringToRippleCurrency(t *testing.T, s string) rippledata.Currency {
	currency, err := rippledata.NewCurrency(s)
	require.NoError(t, err)

	return currency
}

func convertStringToRippleAccount(t *testing.T, s string) rippledata.Account {
	acc, err := rippledata.NewAccountFromAddress(s)
	require.NoError(t, err)

	return *acc
}

//nolint:unparam // helper func
func convertStringToRippleValue(t *testing.T, s string, native bool) *rippledata.Value {
	v, err := rippledata.NewValue(s, native)
	require.NoError(t, err)

	return v
}
