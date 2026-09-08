package coupons

import "context"

// CouponCurrencyAmount is a manual per-currency override for a Fixed-type
// Coupon's DiscountValue -- coupons are tenant-wide (not channel-scoped,
// see Coupon's own doc comment), so a single code can be redeemed on
// channels running in different currencies. Absent an override, the
// checkout flow converts DiscountValue via an fx.ExchangeRate instead (see
// application/storefront's resolveCouponDiscount). Meaningless for
// DiscountTypePercent, whose DiscountValue is already currency-agnostic
// (basis points off the subtotal).
type CouponCurrencyAmount struct {
	ID               string
	TenantID         string
	CouponID         string
	CurrencyCode     string
	AmountMinorUnits int64
}

type CouponCurrencyAmountRepository interface {
	// FindByCouponAndCurrency returns nil, nil (not a typed not-found
	// error) when this coupon has no override for currencyCode -- the
	// common case, not a fault.
	FindByCouponAndCurrency(ctx context.Context, couponID, currencyCode string) (*CouponCurrencyAmount, error)
	ListByCoupon(ctx context.Context, couponID string) ([]CouponCurrencyAmount, error)
	// Upsert creates or replaces the override for (CouponID, CurrencyCode)
	// -- unique per pair, never a history of past overrides.
	Upsert(ctx context.Context, a *CouponCurrencyAmount) error
	Delete(ctx context.Context, couponID, currencyCode string) error
}
