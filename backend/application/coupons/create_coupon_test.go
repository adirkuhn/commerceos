package coupons_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
)

func TestCreateCoupon_success(t *testing.T) {
	repo := &fakeCouponRepo{}
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 1), repo)

	coupon, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "summer10", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000,
	})

	require.NoError(t, err)
	require.Equal(t, "SUMMER10", coupon.Code)
	require.True(t, coupon.Active)
	require.NotNil(t, repo.created)
}

func TestCreateCoupon_rejectsEmptyCode(t *testing.T) {
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 0), &fakeCouponRepo{})

	_, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "  ", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000,
	})

	require.ErrorIs(t, err, couponsDomain.ErrCouponCodeRequired)
}

func TestCreateCoupon_rejectsInvalidDiscountType(t *testing.T) {
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 0), &fakeCouponRepo{})

	_, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "X", DiscountType: "bogus", DiscountValue: 1000,
	})

	require.ErrorIs(t, err, couponsDomain.ErrInvalidDiscountType)
}

func TestCreateCoupon_rejectsPercentOver100(t *testing.T) {
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 0), &fakeCouponRepo{})

	_, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "X", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 10001,
	})

	require.ErrorIs(t, err, couponsDomain.ErrInvalidDiscountValue)
}

func TestCreateCoupon_rejectsNegativeFixedAmount(t *testing.T) {
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 0), &fakeCouponRepo{})

	_, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "X", DiscountType: couponsDomain.DiscountTypeFixed, DiscountValue: -1,
	})

	require.ErrorIs(t, err, couponsDomain.ErrInvalidDiscountValue)
}

func TestCreateCoupon_rejectsNegativeMinSubtotal(t *testing.T) {
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 0), &fakeCouponRepo{})
	min := int64(-1)

	_, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "X", DiscountType: couponsDomain.DiscountTypeFixed, DiscountValue: 100, MinSubtotalMinorUnits: &min,
	})

	require.ErrorIs(t, err, couponsDomain.ErrInvalidMinSubtotal)
}

func TestCreateCoupon_rejectsZeroOrNegativeUsageLimit(t *testing.T) {
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 0), &fakeCouponRepo{})
	limit := 0

	_, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "X", DiscountType: couponsDomain.DiscountTypeFixed, DiscountValue: 100, UsageLimit: &limit,
	})

	require.ErrorIs(t, err, couponsDomain.ErrInvalidUsageLimit)
}

func TestCreateCoupon_propagatesDuplicateCode(t *testing.T) {
	repo := &fakeCouponRepo{createErr: couponsDomain.ErrCouponCodeTaken}
	uc := appcoupons.NewCreateCouponUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), appcoupons.CreateCouponInput{
		Code: "X", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000,
	})

	require.ErrorIs(t, err, couponsDomain.ErrCouponCodeTaken)
}
