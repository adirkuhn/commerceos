package coupons_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
)

func existingCoupon() couponsDomain.Coupon {
	return couponsDomain.Coupon{ID: "coupon-1", Code: "OLD", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 500, Active: true}
}

func TestUpdateCoupon_success(t *testing.T) {
	repo := &fakeCouponRepo{byID: map[string]couponsDomain.Coupon{"coupon-1": existingCoupon()}}
	uc := appcoupons.NewUpdateCouponUseCase(newMockGormDB(t, 1), repo)

	coupon, err := uc.Execute(context.Background(), appcoupons.UpdateCouponInput{
		ID: "coupon-1", Code: "new", DiscountType: couponsDomain.DiscountTypeFixed, DiscountValue: 1000, Active: false,
	})

	require.NoError(t, err)
	require.Equal(t, "NEW", coupon.Code)
	require.Equal(t, couponsDomain.DiscountTypeFixed, coupon.DiscountType)
	require.False(t, coupon.Active)
}

func TestUpdateCoupon_propagatesNotFound(t *testing.T) {
	uc := appcoupons.NewUpdateCouponUseCase(newMockGormDB(t, 1), &fakeCouponRepo{})

	_, err := uc.Execute(context.Background(), appcoupons.UpdateCouponInput{
		ID: "missing", Code: "X", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000,
	})

	require.ErrorIs(t, err, couponsDomain.ErrCouponNotFound)
}

func TestUpdateCoupon_rejectsInvalidInputBeforeLookup(t *testing.T) {
	repo := &fakeCouponRepo{byID: map[string]couponsDomain.Coupon{"coupon-1": existingCoupon()}}
	uc := appcoupons.NewUpdateCouponUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), appcoupons.UpdateCouponInput{
		ID: "coupon-1", Code: "", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000,
	})

	require.ErrorIs(t, err, couponsDomain.ErrCouponCodeRequired)
}
