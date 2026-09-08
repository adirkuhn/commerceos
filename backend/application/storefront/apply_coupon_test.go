package storefront_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

type fakeCouponRepoForCheckout struct {
	couponsDomain.Repository
	byCode        map[string]couponsDomain.Coupon
	incrementedID string
}

func (f *fakeCouponRepoForCheckout) FindByCode(_ context.Context, code string) (*couponsDomain.Coupon, error) {
	if c, ok := f.byCode[code]; ok {
		cp := c
		return &cp, nil
	}
	return nil, nil
}

func (f *fakeCouponRepoForCheckout) IncrementTimesUsed(_ context.Context, id string) error {
	f.incrementedID = id
	return nil
}

func redeemableCoupon() couponsDomain.Coupon {
	return couponsDomain.Coupon{ID: "coupon-1", Code: "SUMMER10", DiscountType: couponsDomain.DiscountTypePercent, DiscountValue: 1000, Active: true}
}

func TestApplyCoupon_attachesARedeemableCode(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	couponRepo := &fakeCouponRepoForCheckout{byCode: map[string]couponsDomain.Coupon{"SUMMER10": redeemableCoupon()}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, couponRepo)

	result, err := uc.Execute(context.Background(), "token-1", "summer10")

	require.NoError(t, err)
	require.NotNil(t, result.Cart.CouponCode)
	require.Equal(t, "SUMMER10", *result.Cart.CouponCode)
	require.Equal(t, int64(100), result.DiscountMinorUnits)
	require.NotNil(t, cartRepo.updated)
}

func TestApplyCoupon_removesTheCouponWhenCodeIsEmpty(t *testing.T) {
	cart := openCartWithOneItem()
	code := "SUMMER10"
	cart.CouponCode = &code
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": cart}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, &fakeCouponRepoForCheckout{})

	result, err := uc.Execute(context.Background(), "token-1", "")

	require.NoError(t, err)
	require.Nil(t, result.Cart.CouponCode)
	require.Zero(t, result.DiscountMinorUnits)
}

func TestApplyCoupon_rejectsUnknownCode(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, &fakeCouponRepoForCheckout{})

	_, err := uc.Execute(context.Background(), "token-1", "NOPE")

	require.ErrorIs(t, err, couponsDomain.ErrCouponNotRedeemable)
}

func TestApplyCoupon_rejectsExpiredCode(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	expired := redeemableCoupon()
	expired.ExpiresAt = &past
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	couponRepo := &fakeCouponRepoForCheckout{byCode: map[string]couponsDomain.Coupon{"SUMMER10": expired}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, couponRepo)

	_, err := uc.Execute(context.Background(), "token-1", "SUMMER10")

	require.ErrorIs(t, err, couponsDomain.ErrCouponNotRedeemable)
}

func TestApplyCoupon_rejectsWhenCartSubtotalBelowMinimum(t *testing.T) {
	min := int64(100000)
	tooHigh := redeemableCoupon()
	tooHigh.MinSubtotalMinorUnits = &min
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": openCartWithOneItem()}}
	couponRepo := &fakeCouponRepoForCheckout{byCode: map[string]couponsDomain.Coupon{"SUMMER10": tooHigh}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, couponRepo)

	_, err := uc.Execute(context.Background(), "token-1", "SUMMER10")

	require.ErrorIs(t, err, couponsDomain.ErrCouponMinSubtotalNotMet)
}

func TestApplyCoupon_rejectsAlreadyCheckedOutCart(t *testing.T) {
	cart := openCartWithOneItem()
	cart.CheckedOutAt = timePtr()
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{"token-1": cart}}
	uc := appstorefront.NewApplyCouponUseCase(newMockGormDB(t, 1), cartRepo, &fakeCouponRepoForCheckout{})

	_, err := uc.Execute(context.Background(), "token-1", "SUMMER10")

	require.ErrorIs(t, err, storefrontDomain.ErrCartAlreadyCheckedOut)
}
