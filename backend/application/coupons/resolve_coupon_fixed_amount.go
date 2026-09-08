package coupons

import (
	"context"

	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
)

// ResolveCouponFixedAmount computes a Fixed-type coupon's discount in
// channelCurrency ("" meaning the tenant's default currency -- the
// pre-multi-currency behavior, always coupon.DiscountValue unchanged).
// Only meaningful for DiscountTypeFixed -- callers never call this for
// DiscountTypePercent, whose DiscountValue (basis points) is already
// currency-agnostic. Must be called from inside an existing
// tenant.WithTenant block, same convention as
// application/pms.ResolveSKUPrice.
//
// Resolution order: (1) "" needs no conversion; (2) a manual
// CouponCurrencyAmount override for channelCurrency wins if one exists;
// (3) otherwise fx.ExchangeRate converts coupon.DiscountValue --
// ErrExchangeRateRequired if none is configured.
func ResolveCouponFixedAmount(ctx context.Context, coupon couponsDomain.Coupon, channelCurrency string, overrideRepo couponsDomain.CouponCurrencyAmountRepository, fxRepo fxDomain.ExchangeRateRepository) (int64, error) {
	if channelCurrency == "" {
		return int64(coupon.DiscountValue), nil
	}

	override, err := overrideRepo.FindByCouponAndCurrency(ctx, coupon.ID, channelCurrency)
	if err != nil {
		return 0, err
	}
	if override != nil {
		return override.AmountMinorUnits, nil
	}

	rate, err := fxRepo.FindByCurrency(ctx, channelCurrency)
	if err != nil {
		return 0, err
	}
	if rate == nil {
		return 0, couponsDomain.ErrExchangeRateRequired
	}
	return rate.Convert(int64(coupon.DiscountValue)), nil
}

// EffectiveCoupon returns coupon with DiscountValue resolved into
// channelCurrency -- a no-op copy for DiscountTypePercent (already
// currency-agnostic) or "" (the tenant's default currency). Callers then
// call DiscountForSubtotal on the result unchanged, so the clamp-to-subtotal
// logic never needs duplicating here.
func EffectiveCoupon(ctx context.Context, coupon couponsDomain.Coupon, channelCurrency string, overrideRepo couponsDomain.CouponCurrencyAmountRepository, fxRepo fxDomain.ExchangeRateRepository) (couponsDomain.Coupon, error) {
	if coupon.DiscountType != couponsDomain.DiscountTypeFixed {
		return coupon, nil
	}
	amount, err := ResolveCouponFixedAmount(ctx, coupon, channelCurrency, overrideRepo, fxRepo)
	if err != nil {
		return coupon, err
	}
	coupon.DiscountValue = int(amount)
	return coupon, nil
}
