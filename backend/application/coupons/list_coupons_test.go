package coupons_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appcoupons "github.com/adirkuhn/commerceos/application/coupons"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
)

func TestListCoupons_returnsAll(t *testing.T) {
	repo := &fakeCouponRepo{byID: map[string]couponsDomain.Coupon{"coupon-1": existingCoupon()}}
	uc := appcoupons.NewListCouponsUseCase(newMockGormDB(t, 1), repo)

	coupons, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, coupons, 1)
}
