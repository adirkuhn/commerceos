package promotions

import (
	"context"
	"time"
)

type PromotionRepository interface {
	Create(ctx context.Context, p *Promotion) error
	FindByID(ctx context.Context, id string) (*Promotion, error)
	// List returns every promotion for the tenant, in no particular order --
	// small, tenant-scoped set, callers sort/filter client-side if needed.
	List(ctx context.Context) ([]Promotion, error)
	Update(ctx context.Context, p *Promotion) error
	// Delete cascades to the promotion's PromotionSKU lines at the DB level
	// (ON DELETE CASCADE) -- callers don't need a separate line cleanup.
	Delete(ctx context.Context, id string) error
}

type PromotionSKURepository interface {
	Create(ctx context.Context, line *PromotionSKU) error
	ListByPromotion(ctx context.Context, promotionID string) ([]PromotionSKU, error)
	// DeleteByPromotion removes every line for promotionID -- used by
	// UpdatePromotionUseCase to replace a promotion's whole line set
	// wholesale rather than diffing individual lines.
	DeleteByPromotion(ctx context.Context, promotionID string) error
	// ListDueForActivation returns every not-yet-applied, not-finished line
	// whose parent promotion is active and currently within its time window
	// (starts_at <= at < ends_at, or ends_at NULL) -- cross-tenant, for
	// application/scheduler.RunPromotionActivationUseCase's periodic sweep.
	ListDueForActivation(ctx context.Context, at time.Time) ([]PromotionSKU, error)
	// ListDueForReversal returns every currently-applied line that is either
	// finished (see PromotionSKU.FinishedAt) or whose parent promotion has
	// left its active window (ended, drafted back, or past ends_at) --
	// cross-tenant, same sweep as ListDueForActivation.
	ListDueForReversal(ctx context.Context, at time.Time) ([]PromotionSKU, error)
	// MarkApplied records that lineID's discount is now live on its SKU,
	// snapshotting the SKU's pre-promotion sale price fields for later
	// reversal.
	MarkApplied(ctx context.Context, lineID string, previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time) error
	// MarkReverted clears lineID's Applied flag and snapshot fields once
	// its SKU has been restored.
	MarkReverted(ctx context.Context, lineID string) error
}

// ClearanceLineRepository is the clearance-specific side of promotion_skus
// (lines with a BatchID) -- kept apart from PromotionSKURepository so the
// many callers that never deal with clearance don't have to stub it (ISP).
type ClearanceLineRepository interface {
	// ListOpen returns every clearance line not yet finished -- cross-tenant
	// (called under tenant.WithSuperAdmin) for the periodic sweep, which is
	// the only thing that notices a batch passing its expiry date or being
	// deleted, since neither raises a stock event.
	ListOpen(ctx context.Context) ([]PromotionSKU, error)
	// ListOpenBySKU is ListOpen narrowed to one SKU for the current tenant --
	// what the inventory.stock.changed consumer checks.
	ListOpenBySKU(ctx context.Context, skuID string) ([]PromotionSKU, error)
	// MarkFinished sets lineID's FinishedAt.
	MarkFinished(ctx context.Context, lineID string, at time.Time) error
	// CountUnfinished returns how many of promotionID's lines are not
	// finished -- ordinary lines always count, since they never finish.
	CountUnfinished(ctx context.Context, promotionID string) (int64, error)
}
