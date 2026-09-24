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
// discounted"). A scheduled job (application/scheduler.RunPromotionActivationUseCase)
// drives each covered SKU's SalePriceMinorUnits while the promotion is in
// its active window, reusing the same field/mechanism a manually-set sale
// price uses -- see PromotionSKU's Applied/Previous* fields for how that's
// tracked and reversed.
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
	// Applied is true while this line's discount is currently live on the
	// SKU's SalePriceMinorUnits -- set by ApplyPromotionSKUUseCase, cleared
	// by RevertPromotionSKUUseCase once the parent promotion leaves its
	// active window. A promotion always wins over a pre-existing manual
	// sale price while active (see CreatePromotionUseCase's warning on
	// create, which is the only place a conflict is surfaced -- there's no
	// runtime precedence check here).
	Applied bool
	// PreviousSalePriceMinorUnits/PreviousSaleEndDate snapshot the SKU's
	// sale price fields at the moment this line was applied, nil if it had
	// none -- restored verbatim by RevertPromotionSKUUseCase so ending a
	// promotion never fabricates or loses an unrelated manual sale.
	PreviousSalePriceMinorUnits *int64
	PreviousSaleEndDate         *time.Time
	// BatchID makes this a clearance line: it sells off one near-expiry
	// inventory batch and ends on its own once that batch is finished (see
	// ClearanceBatch.IsFinished). Nil for an ordinary line, which only ends
	// with its parent promotion's window. The discount still lands on the
	// whole SKU's sale price -- channels and the POS sell SKUs, not batches.
	BatchID *string
	// FinishedAt is set once a clearance line's batch is finished. A
	// finished line is never (re)applied and, if still applied, is due for
	// reversal regardless of its promotion's window -- see
	// PromotionSKURepository.ListDueForActivation/ListDueForReversal.
	FinishedAt *time.Time
	CreatedAt  time.Time
}
