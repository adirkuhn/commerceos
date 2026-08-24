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
	// ListDueForActivation returns every not-yet-applied line whose parent
	// promotion is active and currently within its time window
	// (starts_at <= at < ends_at, or ends_at NULL) -- cross-tenant, for
	// application/scheduler.RunPromotionActivationUseCase's periodic sweep.
	ListDueForActivation(ctx context.Context, at time.Time) ([]PromotionSKU, error)
	// ListDueForReversal returns every currently-applied line whose parent
	// promotion has left its active window (ended, drafted back, or past
	// ends_at) -- cross-tenant, same sweep as ListDueForActivation.
	ListDueForReversal(ctx context.Context, at time.Time) ([]PromotionSKU, error)
	// MarkApplied records that lineID's discount is now live on its SKU,
	// snapshotting the SKU's pre-promotion sale price fields for later
	// reversal.
	MarkApplied(ctx context.Context, lineID string, previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time) error
	// MarkReverted clears lineID's Applied flag and snapshot fields once
	// its SKU has been restored.
	MarkReverted(ctx context.Context, lineID string) error
}
