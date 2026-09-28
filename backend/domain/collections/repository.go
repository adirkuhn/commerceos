package collections

import "context"

// Repository is tenant-scoped via RLS; the transaction comes from ctx.
type Repository interface {
	// Create fills in c.ID; ErrSlugTaken when the slug is in use.
	Create(ctx context.Context, c *Collection) error
	// FindByID / FindBySlug return ErrCollectionNotFound when missing.
	FindByID(ctx context.Context, id string) (*Collection, error)
	FindBySlug(ctx context.Context, slug string) (*Collection, error)
	List(ctx context.Context, publishedOnly bool, limit, offset int) ([]Collection, int64, error)
	// Update persists the draft fields; ErrSlugTaken when the slug is in use.
	Update(ctx context.Context, c *Collection) error
	Delete(ctx context.Context, id string) error
	// ManualProducts returns a manual collection's product ids in order.
	ManualProducts(ctx context.Context, collectionID string) ([]string, error)
	// SetManualProducts replaces a manual collection's products and order.
	SetManualProducts(ctx context.Context, collectionID string, productIDs []string) error
	// ResolveProductIDs lists the active products a collection contains,
	// in its sort order, up to limit.
	ResolveProductIDs(ctx context.Context, c Collection, limit int) ([]string, error)
}
