package storefront_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
)

func TestGetCart_returnsTheCartForItsToken(t *testing.T) {
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {ID: "cart-1", Token: "token-1", Items: []storefrontDomain.CartItem{{SKUCode: "SKU-A", Quantity: 2, PriceMinorUnits: 500}}},
	}}
	uc := appstorefront.NewGetCartUseCase(newMockGormDB(t, 1), cartRepo, nil, &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepo{})

	result, err := uc.Execute(context.Background(), "token-1")

	require.NoError(t, err)
	require.Equal(t, "cart-1", result.Cart.ID)
	require.Equal(t, int64(1000), result.Cart.SubtotalMinorUnits())
	require.Zero(t, result.DiscountMinorUnits)
}

func TestGetCart_returnsNotFoundForUnknownToken(t *testing.T) {
	uc := appstorefront.NewGetCartUseCase(newMockGormDB(t, 1), &fakeCartRepo{}, nil, &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepo{})

	_, err := uc.Execute(context.Background(), "unknown-token")

	require.ErrorIs(t, err, storefrontDomain.ErrCartNotFound)
}

func TestGetCart_computesLiveDiscountWhenCouponAttached(t *testing.T) {
	code := "SUMMER10"
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {
			ID: "cart-1", Token: "token-1", CouponCode: &code,
			Items: []storefrontDomain.CartItem{{SKUCode: "SKU-A", Quantity: 2, PriceMinorUnits: 500}},
		},
	}}
	couponRepo := &fakeCouponRepoForCheckout{byCode: map[string]couponsDomain.Coupon{"SUMMER10": redeemableCoupon()}}
	uc := appstorefront.NewGetCartUseCase(newMockGormDB(t, 2), cartRepo, couponRepo, &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepo{})

	result, err := uc.Execute(context.Background(), "token-1")

	require.NoError(t, err)
	require.Equal(t, int64(100), result.DiscountMinorUnits) // 1000 subtotal, 10% off
}

func TestGetCart_showsZeroDiscountWhenCouponNoLongerRedeemable(t *testing.T) {
	code := "SUMMER10"
	cartRepo := &fakeCartRepo{byToken: map[string]storefrontDomain.Cart{
		"token-1": {
			ID: "cart-1", Token: "token-1", CouponCode: &code,
			Items: []storefrontDomain.CartItem{{SKUCode: "SKU-A", Quantity: 2, PriceMinorUnits: 500}},
		},
	}}
	uc := appstorefront.NewGetCartUseCase(newMockGormDB(t, 2), cartRepo, &fakeCouponRepoForCheckout{}, &fakeCouponCurrencyAmountRepo{}, &fakeExchangeRateRepo{})

	result, err := uc.Execute(context.Background(), "token-1")

	require.NoError(t, err)
	require.Zero(t, result.DiscountMinorUnits)
}
