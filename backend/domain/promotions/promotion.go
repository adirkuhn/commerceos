package promotions

import "time"

// PromotionType is the discount mechanism -- both a Promotion's overall
// classification and, per line, what PromotionSKU.DiscountValue means. V1
// supports only the two simplest mechanisms; BOGO/bundle/quantity/category/
// customer-segment/coupon are deliberately out of scope until a later slice
// (see the roadmap's Phase 0 promotion-history item).
type PromotionType string

const (
	// PromotionTypePercentage means DiscountValue is basis points (1 bp =
	// 0.01%), e.g. 1000 = 10% off -- same convention as tax.TaxRate's
	// RateBasisPoints, so a value that ends up multiplied into money never
	// carries float rounding error.
	PromotionTypePercentage PromotionType = "percentage"
	// PromotionTypeFixedAmount means DiscountValue is minor currency units
	// (e.g. cents), same convention as pms.SKU.PriceMinorUnits.
	PromotionTypeFixedAmount PromotionType = "fixed_amount"
)

type PromotionStatus string

const (
	PromotionStatusDraft  PromotionStatus = "draft"
	PromotionStatusActive PromotionStatus = "active"
	PromotionStatusEnded  PromotionStatus = "ended"
)

// Promotion is a named, time-boxed discount campaign -- tracked separately
// from pms.SKU's own PriceMinorUnits/SalePriceMinorUnits (see the Phase 0
// roadmap's distinction between "what a SKU costs" and "why/how it's
// discounted"). This slice only defines and manages promotions; nothing
// here yet mutates a SKU's effective price -- that's a separate, later
// decision (scheduler-driven activation vs. computed at checkout).
type Promotion struct {
	ID       string
	TenantID string
	Name     string
	Type     PromotionType
	StartsAt time.Time
	// EndsAt is nil for an open-ended promotion (no planned end date).
	EndsAt    *time.Time
	Status    PromotionStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PromotionSKU is one SKU's discount within a Promotion -- DiscountType is
// independent of the parent Promotion's own Type so a promotion can (in
// principle) mix mechanisms across its lines, e.g. 10% off SKU A and a
// fixed €2 off SKU B under the same campaign.
type PromotionSKU struct {
	ID            string
	TenantID      string
	PromotionID   string
	SKUID         string
	DiscountType  PromotionType
	DiscountValue int64
	CreatedAt     time.Time
}
