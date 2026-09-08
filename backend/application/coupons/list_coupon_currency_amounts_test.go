package coupons_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
)

func TestListCouponCurrencyAmountsUseCase_Execute(t *testing.T) {
	repo := &fakeCouponCurrencyAmountRepo{byKey: map[string]couponsDomain.CouponCurrencyAmount{
		"coupon-1|USD": {CouponID: "coupon-1", CurrencyCode: "USD", AmountMinorUnits: 1200},
	}}
	uc := appcoupons.NewListCouponCurrencyAmountsUseCase(newMockGormDB(t, 1), repo)

	amounts, err := uc.Execute(context.Background(), "coupon-1")
	require.NoError(t, err)
	require.Len(t, amounts, 1)
	require.Equal(t, "USD", amounts[0].CurrencyCode)
}
