package coupons

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrCouponNotFound = errs.New("COUPON_NOT_FOUND", "coupon not found")
	// ErrCouponCodeRequired guards Create/UpdateCouponUseCase: an empty
	// code has nothing for a shopper to type in.
	ErrCouponCodeRequired = errs.New("COUPON_CODE_REQUIRED", "coupon code is required")
	// ErrCouponCodeTaken maps the (tenant_id, code) unique constraint --
	// codes are case-insensitive-unique per tenant (always normalized to
	// uppercase before comparison/storage, see application/coupons).
	ErrCouponCodeTaken      = errs.New("COUPON_CODE_TAKEN", "a coupon with this code already exists")
	ErrInvalidDiscountType  = errs.New("COUPON_INVALID_DISCOUNT_TYPE", "invalid discount type")
	ErrInvalidDiscountValue = errs.New("COUPON_INVALID_DISCOUNT_VALUE", "discount value out of range for this discount type")
	ErrInvalidMinSubtotal   = errs.New("COUPON_INVALID_MIN_SUBTOTAL", "minimum subtotal must be zero or positive")
	ErrInvalidUsageLimit    = errs.New("COUPON_INVALID_USAGE_LIMIT", "usage limit must be a positive number")
	// ErrCouponNotRedeemable is returned when a coupon exists but
	// IsRedeemable is false (inactive, expired, or usage limit reached) --
	// deliberately the same message regardless of which, so a shopper
	// can't use the error to probe a code's configuration.
	ErrCouponNotRedeemable = errs.New("COUPON_NOT_REDEEMABLE", "this coupon code is no longer valid")
	// ErrCouponMinSubtotalNotMet is returned separately from
	// ErrCouponNotRedeemable (see Coupon.MeetsMinSubtotal's doc comment)
	// since a shopper can fix this one by adding more to their cart.
	ErrCouponMinSubtotalNotMet = errs.New("COUPON_MIN_SUBTOTAL_NOT_MET", "cart subtotal does not meet this coupon's minimum")
	// ErrExchangeRateRequired is returned by ResolveCouponFixedAmount when
	// a Fixed coupon needs converting into a channel currency that has
	// neither an explicit CouponCurrencyAmount override nor a configured
	// fx.ExchangeRate.
	ErrExchangeRateRequired = errs.New("COUPON_EXCHANGE_RATE_REQUIRED", "no exchange rate or amount override configured for this currency")
)
