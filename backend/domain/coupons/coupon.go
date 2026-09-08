// Package coupons holds cart-level discount codes -- deliberately its own
// bounded context rather than folded into domain/promotions, whose own
// doc comment is explicit that coupons are out of scope for that package
// (it only ever drives pms.SKU.SalePriceMinorUnits on a schedule, never a
// code a shopper types in). A Coupon is a tenant-wide code (not scoped to
// a sales channel or product) applied once per cart at checkout.
package coupons

import "time"

type DiscountType string

const (
	DiscountTypePercent DiscountType = "percent"
	DiscountTypeFixed   DiscountType = "fixed"
)

func (t DiscountType) Valid() bool {
	switch t {
	case DiscountTypePercent, DiscountTypeFixed:
		return true
	default:
		return false
	}
}

// Coupon is a tenant-wide discount code. DiscountValue's unit depends on
// DiscountType: basis points (0-10000) off the cart subtotal for Percent,
// or a flat minor-units amount for Fixed -- same "integer, never a float"
// convention as tax.TaxRate.RateBasisPoints and
// channels.SalesChannel.PriceAdjustmentBasisPoints.
type Coupon struct {
	ID            string
	TenantID      string
	Code          string
	DiscountType  DiscountType
	DiscountValue int
	// MinSubtotalMinorUnits, nil meaning no minimum, is the cart subtotal
	// (before this discount) required for the code to apply at all -- see
	// MeetsMinSubtotal.
	MinSubtotalMinorUnits *int64
	// UsageLimit, nil meaning unlimited, caps how many times this code can
	// ever be redeemed across every cart/customer -- see IsRedeemable.
	// TimesUsed increments once per successful checkout that used it (see
	// application/storefront.CheckoutCartUseCase).
	UsageLimit *int
	TimesUsed  int
	// ExpiresAt, nil meaning it never expires.
	ExpiresAt *time.Time
	Active    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsRedeemable reports whether this code can still be used right now --
// active, not expired, and (if UsageLimit is set) not yet exhausted. Does
// NOT check MinSubtotalMinorUnits -- that depends on a specific cart's
// subtotal, checked separately via MeetsMinSubtotal, since "redeemable at
// all" and "applies to this cart" are different questions with different
// error messages for the shopper.
func (c Coupon) IsRedeemable(now time.Time) bool {
	if !c.Active {
		return false
	}
	if c.ExpiresAt != nil && now.After(*c.ExpiresAt) {
		return false
	}
	if c.UsageLimit != nil && c.TimesUsed >= *c.UsageLimit {
		return false
	}
	return true
}

// MeetsMinSubtotal reports whether subtotalMinorUnits qualifies for this
// code at all -- see MinSubtotalMinorUnits's doc comment.
func (c Coupon) MeetsMinSubtotal(subtotalMinorUnits int64) bool {
	return c.MinSubtotalMinorUnits == nil || subtotalMinorUnits >= *c.MinSubtotalMinorUnits
}

// DiscountForSubtotal computes the discount amount for subtotalMinorUnits
// -- callers check IsRedeemable/MeetsMinSubtotal first (this never checks
// either itself, so it can't be used to silently apply an expired or
// under-minimum code). Never returns more than subtotalMinorUnits itself
// -- a fixed-amount coupon larger than the cart discounts it to exactly 0,
// never negative.
func (c Coupon) DiscountForSubtotal(subtotalMinorUnits int64) int64 {
	var discount int64
	if c.DiscountType == DiscountTypePercent {
		discount = subtotalMinorUnits * int64(c.DiscountValue) / 10000
	} else {
		discount = int64(c.DiscountValue)
	}
	if discount > subtotalMinorUnits {
		discount = subtotalMinorUnits
	}
	if discount < 0 {
		discount = 0
	}
	return discount
}
