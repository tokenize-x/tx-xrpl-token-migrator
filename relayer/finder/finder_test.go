package finder

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/CoreumFoundation/coreum/v5/pkg/config"
	"github.com/CoreumFoundation/coreum/v5/pkg/config/constant"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	rippledata "github.com/rubblelabs/ripple/data"
	"github.com/stretchr/testify/require"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/client/xrpl"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/logger"
	"github.com/tokenize-x/tx-xrpl-token-migrator/relayer/metric"
	"go.uber.org/zap/zaptest"
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
		Multiplier:                 "1.0",
		XRPLHistoryScanStartLedger: 8000,
		XRPLMemoSuffix:             "=cored",
		TXDenom:                    "ucore",
		TXDecimals:                 6,
	}

	txAddress := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address())

	validXRPLTransaction := xrpl.Transaction{
		DeliveryAmount: rippledata.Amount{
			Currency: cfg.XRPLCurrency,
			Issuer:   cfg.XRPLIssuer,
			Value:    convertStringToRippleValue(t, "1.23456789", false),
		},
		Memos: []string{
			"none-address" + cfg.XRPLMemoSuffix,
			fmt.Sprintf("%s%s", txAddress, cfg.XRPLMemoSuffix),
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
		want        PendingTXSendTransaction
	}{
		{
			name: "positive",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				return tx
			},
			wantMatches: true,
			want: PendingTXSendTransaction{
				TXDestination: txAddress,
				TXAmount:      sdk.NewInt64Coin(cfg.TXDenom, 1234567),
				XRPLTxHash:    "xrpl-tx-hash",
			},
		},
		{
			name: "negative_tx_before_activation_date",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Date = time.Date(2000, 4, 1, 0, 0, 0, 0, time.UTC)
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_invalid_tx_validated",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Validated = false
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_invalid_tx_result",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.TransactionType = "fail"
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_invalid_tx_type",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.TransactionType = "AccountSet"
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "invalid_memo",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Memos = []string{"invalid-memo"}
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "invalid_address_prefx_memo",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.Memos = []string{"devcore17l2fxde2662s2p8pgmzu04jcvflnnlq4l30hff" + cfg.XRPLMemoSuffix}
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "invalid_currency",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Currency = convertStringToRippleCurrency(t, "IND")
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "invalid_issuer",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Issuer = convertStringToRippleAccount(t, rippledata.Account{}.String())
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "zero_amount",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				tx.DeliveryAmount.Value = convertStringToRippleValue(t, "0.0000001", false)
				return tx
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
		{
			name: "negative_empty",
			xrplTxFunc: func(tx xrpl.Transaction) xrpl.Transaction {
				return xrpl.Transaction{}
			},
			wantMatches: false,
			want:        PendingTXSendTransaction{},
		},
	}
	for _, tt := range tests {
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

func TestFinder_convertXRPLAmountToTXCoin(t *testing.T) {
	const denom = "ucore"
	tests := []struct {
		name       string
		xrplAmount *rippledata.Value
		multiplier string
		wantAmount sdk.Coin
	}{
		{
			name:       "no_truncation",
			xrplAmount: convertStringToRippleValue(t, "10.123456", false),
			multiplier: "1.0",
			wantAmount: sdk.NewCoin(denom, sdkmath.NewInt(10123456)),
		},
		{
			name:       "max_amount",
			xrplAmount: convertStringToRippleValue(t, "1000000000", false),
			multiplier: "1.0",
			wantAmount: sdk.NewCoin(denom, func() sdkmath.Int {
				v, _ := sdkmath.NewIntFromString("1000000000000000")
				return v
			}()),
		},
		{
			name:       "many_decimals",
			xrplAmount: convertStringToRippleValue(t, "0.100001000000001", false),
			multiplier: "1.0",
			wantAmount: sdk.NewInt64Coin(denom, 100001),
		},
		{
			name:       "many_decimals_to_zero",
			xrplAmount: convertStringToRippleValue(t, "0.000000000000001", false),
			multiplier: "1.0",
			wantAmount: sdk.NewInt64Coin(denom, 0),
		},
		{
			name:       "default_float_rounding",
			xrplAmount: convertStringToRippleValue(t, "0.001", false),
			multiplier: "1.0",
			wantAmount: sdk.NewInt64Coin(denom, 1000),
		},
		{
			name:       "default_float_rounding_down",
			xrplAmount: convertStringToRippleValue(t, "1.111111111111111", false),
			multiplier: "1.0",
			wantAmount: sdk.NewInt64Coin(denom, 1111111),
		},
		{
			name:       "just_below_min_0.99",
			xrplAmount: convertStringToRippleValue(t, "1.0", false),
			multiplier: "0.99",
			wantAmount: sdk.NewInt64Coin(denom, 990000),
		},
		{
			name:       "just_above_min_1.01",
			xrplAmount: convertStringToRippleValue(t, "1.0", false),
			multiplier: "1.01",
			wantAmount: sdk.NewInt64Coin(denom, 1010000),
		},
		{
			name:       "just_below_max_1.99",
			xrplAmount: convertStringToRippleValue(t, "1.0", false),
			multiplier: "1.99",
			wantAmount: sdk.NewInt64Coin(denom, 1990000),
		},
		{
			name:       "just_above_max_2.01",
			xrplAmount: convertStringToRippleValue(t, "1.0", false),
			multiplier: "2.01",
			wantAmount: sdk.NewInt64Coin(denom, 2010000),
		},
		{
			name:       "multiplier_0.5",
			xrplAmount: convertStringToRippleValue(t, "2.0", false),
			multiplier: "0.5",
			wantAmount: sdk.NewInt64Coin(denom, 1000000),
		},
		{
			name:       "multiplier_1.5",
			xrplAmount: convertStringToRippleValue(t, "2.0", false),
			multiplier: "1.5",
			wantAmount: sdk.NewInt64Coin(denom, 3000000),
		},
		{
			name:       "multiplier_2.5",
			xrplAmount: convertStringToRippleValue(t, "2.0", false),
			multiplier: "2.5",
			wantAmount: sdk.NewInt64Coin(denom, 5000000),
		},
		{
			name:       "very_large_value_with_multiplier",
			xrplAmount: convertStringToRippleValue(t, "1000000000", false),
			multiplier: "2.5",
			wantAmount: sdk.NewCoin(denom, func() sdkmath.Int {
				v, _ := sdkmath.NewIntFromString("2500000000000000")
				return v
			}()),
		},
		{
			name:       "very_small_fraction_with_multiplier",
			xrplAmount: convertStringToRippleValue(t, "0.000001", false),
			multiplier: "0.5",
			wantAmount: sdk.NewInt64Coin(denom, 0), // 0.0000005 * 1e6 = 0.5
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Finder{
				cfg: Config{
					TXDenom:    denom,
					TXDecimals: 6,
					Multiplier: tt.multiplier,
				},
			}
			if got := f.convertXRPLAmountToTXCoin(tt.xrplAmount); !reflect.DeepEqual(got.String(), tt.wantAmount.String()) {
				t.Errorf("convertXRPLAmountToTXCoin() = %v, want %v", got.String(), tt.wantAmount.String())
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
