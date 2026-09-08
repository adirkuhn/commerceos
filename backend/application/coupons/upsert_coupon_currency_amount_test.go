package coupons_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

func TestUpsertCouponCurrencyAmountUseCase_Execute(t *testing.T) {
	t.Run("rejects an invalid currency code", func(t *testing.T) {
		uc := appcoupons.NewUpsertCouponCurrencyAmountUseCase(newMockGormDB(t, 0), &fakeCouponRepo{}, &fakeCouponCurrencyAmountRepo{})

		_, err := uc.Execute(context.Background(), appcoupons.UpsertCouponCurrencyAmountInput{CouponID: "coupon-1", CurrencyCode: "usd", AmountMinorUnits: 1000})
		require.ErrorIs(t, err, fxDomain.ErrInvalidCurrencyCode)
	})

	t.Run("rejects a negative amount", func(t *testing.T) {
		uc := appcoupons.NewUpsertCouponCurrencyAmountUseCase(newMockGormDB(t, 0), &fakeCouponRepo{}, &fakeCouponCurrencyAmountRepo{})

		_, err := uc.Execute(context.Background(), appcoupons.UpsertCouponCurrencyAmountInput{CouponID: "coupon-1", CurrencyCode: "USD", AmountMinorUnits: -1})
		require.ErrorIs(t, err, couponsDomain.ErrInvalidDiscountValue)
	})

	t.Run("upserts a valid override", func(t *testing.T) {
		couponRepo := &fakeCouponRepo{byID: map[string]couponsDomain.Coupon{"coupon-1": {ID: "coupon-1", DiscountType: couponsDomain.DiscountTypeFixed}}}
		repo := &fakeCouponCurrencyAmountRepo{}
		uc := appcoupons.NewUpsertCouponCurrencyAmountUseCase(newMockGormDB(t, 1), couponRepo, repo)

		amount, err := uc.Execute(context.Background(), appcoupons.UpsertCouponCurrencyAmountInput{CouponID: "coupon-1", CurrencyCode: "USD", AmountMinorUnits: 1200})
		require.NoError(t, err)
		require.Equal(t, "USD", amount.CurrencyCode)
		require.Equal(t, int64(1200), amount.AmountMinorUnits)
		require.Equal(t, "coupon-1", repo.upserted.CouponID)
	})

	t.Run("propagates coupon not found", func(t *testing.T) {
		uc := appcoupons.NewUpsertCouponCurrencyAmountUseCase(newMockGormDB(t, 1), &fakeCouponRepo{}, &fakeCouponCurrencyAmountRepo{})

		_, err := uc.Execute(context.Background(), appcoupons.UpsertCouponCurrencyAmountInput{CouponID: "missing", CurrencyCode: "USD", AmountMinorUnits: 1000})
		require.ErrorIs(t, err, couponsDomain.ErrCouponNotFound)
	})
}
