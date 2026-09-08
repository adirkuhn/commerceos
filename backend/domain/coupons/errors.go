package coupons

import "errors"

var (
	ErrCouponNotFound = errors.New("coupon not found")
	// ErrCouponCodeRequired guards Create/UpdateCouponUseCase: an empty
	// code has nothing for a shopper to type in.
	ErrCouponCodeRequired = errors.New("coupon code is required")
	// ErrCouponCodeTaken maps the (tenant_id, code) unique constraint --
	// codes are case-insensitive-unique per tenant (always normalized to
	// uppercase before comparison/storage, see application/coupons).
	ErrCouponCodeTaken      = errors.New("a coupon with this code already exists")
	ErrInvalidDiscountType  = errors.New("invalid discount type")
	ErrInvalidDiscountValue = errors.New("discount value out of range for this discount type")
	ErrInvalidMinSubtotal   = errors.New("minimum subtotal must be zero or positive")
	ErrInvalidUsageLimit    = errors.New("usage limit must be a positive number")
	// ErrCouponNotRedeemable is returned when a coupon exists but
	// IsRedeemable is false (inactive, expired, or usage limit reached) --
	// deliberately the same message regardless of which, so a shopper
	// can't use the error to probe a code's configuration.
	ErrCouponNotRedeemable = errors.New("this coupon code is no longer valid")
	// ErrCouponMinSubtotalNotMet is returned separately from
	// ErrCouponNotRedeemable (see Coupon.MeetsMinSubtotal's doc comment)
	// since a shopper can fix this one by adding more to their cart.
	ErrCouponMinSubtotalNotMet = errors.New("cart subtotal does not meet this coupon's minimum")
)
