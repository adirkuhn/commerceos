package coupons_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

func TestResolveCouponFixedAmount(t *testing.T) {
	coupon := couponsDomain.Coupon{ID: "coupon-1", DiscountType: couponsDomain.DiscountTypeFixed, DiscountValue: 1000}

	t.Run("empty channel currency passes the coupon's own value through", func(t *testing.T) {
		amount, err := appcoupons.ResolveCouponFixedAmount(context.Background(), coupon, "", &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepoCoupons{})
		require.NoError(t, err)
		require.Equal(t, int64(1000), amount)
	})

	t.Run("a manual override wins over fx conversion", func(t *testing.T) {
		overrideRepo := &fakeCouponCurrencyAmountRepo{byKey: map[string]couponsDomain.CouponCurrencyAmount{
			"coupon-1|USD": {CouponID: "coupon-1", CurrencyCode: "USD", AmountMinorUnits: 1200},
		}}
		fxRepo := &fakeExchangeRateRepoCoupons{byCurrency: map[string]fxDomain.ExchangeRate{"USD": {CurrencyCode: "USD", RateBasisPoints: 20000}}}

		amount, err := appcoupons.ResolveCouponFixedAmount(context.Background(), coupon, "USD", overrideRepo, fxRepo)
		require.NoError(t, err)
		require.Equal(t, int64(1200), amount)
	})

	t.Run("no override, converts via the configured fx rate", func(t *testing.T) {
		fxRepo := &fakeExchangeRateRepoCoupons{byCurrency: map[string]fxDomain.ExchangeRate{"USD": {CurrencyCode: "USD", RateBasisPoints: 10876}}}

		amount, err := appcoupons.ResolveCouponFixedAmount(context.Background(), coupon, "USD", &fakeCouponCurrencyAmountRepo{}, fxRepo)
		require.NoError(t, err)
		require.Equal(t, int64(1088), amount)
	})

	t.Run("no override and no fx rate is a hard error", func(t *testing.T) {
		_, err := appcoupons.ResolveCouponFixedAmount(context.Background(), coupon, "USD", &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepoCoupons{})
		require.ErrorIs(t, err, couponsDomain.ErrExchangeRateRequired)
	})
}
