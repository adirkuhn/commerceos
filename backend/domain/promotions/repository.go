package promotions

import "context"

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
}
