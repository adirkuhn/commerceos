package storefront

import (
	"context"
	"time"
)

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

// WishlistRepository mirrors CartRepository's shape/conventions exactly
// (see its doc comment) -- same token-lookup, same whole-object Update.
type WishlistRepository interface {
	Create(ctx context.Context, w *Wishlist) error
	FindByToken(ctx context.Context, token string) (*Wishlist, error)
	Update(ctx context.Context, w *Wishlist) error
}

type MagicLinkTokenRepository interface {
	Create(ctx context.Context, t *MagicLinkToken) error
	// FindByToken returns nil, nil when no token matches -- same
	// "repo returns nil, use case owns the domain error" convention as
	// CartRepository.FindByToken.
	FindByToken(ctx context.Context, token string) (*MagicLinkToken, error)
	// MarkUsed is a single atomic UPDATE ... SET used_at = ? WHERE used_at
	// IS NULL -- never read-modify-write, so two concurrent redemption
	// attempts of the same emailed link can't both succeed (RowsAffected
	// == 0 on the loser).
	MarkUsed(ctx context.Context, id string, usedAt time.Time) error
}

type CustomerSessionRepository interface {
	Create(ctx context.Context, s *CustomerSession) error
	FindByToken(ctx context.Context, token string) (*CustomerSession, error)
	Delete(ctx context.Context, token string) error
}
