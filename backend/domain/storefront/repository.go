package storefront

import "context"

// CartRepository is scoped to the current tenant via RLS (set by
// tenant.WithTenant), same convention as every other repository in this
// codebase.
type CartRepository interface {
	// Create fills in c.ID, c.CreatedAt and c.UpdatedAt.
	Create(ctx context.Context, c *Cart) error
	// FindByToken looks up a cart by its bearer token -- the only way a
	// storefront request identifies its own cart. Nil (not an error) when
	// none exists, same convention as oms.SalesOrderRepository.FindByExternalID.
	FindByToken(ctx context.Context, token string) (*Cart, error)
	// Update replaces Items/CheckedOutAt/UpdatedAt wholesale -- a cart's
	// line set is always rewritten in full, never patched incrementally
	// (same convention as application/promotions.UpdatePromotionUseCase's
	// "replace the whole line set" doc comment).
	Update(ctx context.Context, c *Cart) error
}
