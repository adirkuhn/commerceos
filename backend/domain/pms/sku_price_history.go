package pms

import "time"

// PriceChangeReason says why a SKUPriceHistory interval was opened.
type PriceChangeReason string

const (
	// PriceChangeReasonCreated is the SKU's very first price, recorded by
	// CreateSKUUseCase.
	PriceChangeReasonCreated PriceChangeReason = "created"
	// PriceChangeReasonManualUpdate is a human editing price or sale price
	// via UpdateSKUUseCase.
	PriceChangeReasonManualUpdate PriceChangeReason = "manual_update"
	// PriceChangeReasonSaleExpired is the scheduler clearing a sale whose
	// SaleEndDate has passed, via ExpireSKUSalePriceUseCase.
	PriceChangeReasonSaleExpired PriceChangeReason = "sale_expired"
	// PriceChangeReasonPromotionApplied is a promotions.Promotion entering
	// its active window, via SetPromotionSalePriceUseCase.Apply.
	PriceChangeReasonPromotionApplied PriceChangeReason = "promotion_applied"
	// PriceChangeReasonPromotionEnded is a promotions.Promotion leaving its
	// active window, restoring whatever sale price existed before it took
	// over, via SetPromotionSalePriceUseCase.Revert.
	PriceChangeReasonPromotionEnded PriceChangeReason = "promotion_ended"
)

// SKUPriceHistory is one effective-dated interval of a SKU's regular and
// sale price -- an immutable audit trail, distinct from SKUCostPriceHistory
// (a periodic cost-price snapshot). A row is written only when price or
// sale price actually changes (see CreateSKUUseCase, UpdateSKUUseCase,
// ExpireSKUSalePriceUseCase), never edited except to close it out when
// superseded: ValidUntil starts nil (the interval is open / currently in
// effect) and is filled in by SKUPriceHistoryRepository.CloseOpen the
// moment a new interval opens. This lets CommerceOS reconstruct what a
// SKU's price actually was at any past point in time, independent of the
// SKU's current PriceMinorUnits/SalePriceMinorUnits.
type SKUPriceHistory struct {
	ID                  string
	TenantID            string
	SKUID               string
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	CurrencyCode        string
	ValidFrom           time.Time
	// ValidUntil is nil while this is the currently-effective interval.
	ValidUntil *time.Time
	Reason     PriceChangeReason
	CreatedAt  time.Time
}
