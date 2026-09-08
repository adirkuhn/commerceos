package reviews

import "context"

type ListFilter struct {
	SalesChannelID string
	ProductID      string
	// Limit/Offset default to a repo-chosen page size when Limit is zero.
	Limit  int
	Offset int
}

type Repository interface {
	Create(ctx context.Context, r *Review) error
	// ListByProduct returns the given product's reviews for this channel,
	// newest first.
	ListByProduct(ctx context.Context, filter ListFilter) ([]Review, error)
	// SummaryByProduct aggregates the given product's reviews for this
	// channel -- computed in SQL rather than by loading every row, since a
	// popular product's review count can far exceed any page size.
	SummaryByProduct(ctx context.Context, salesChannelID, productID string) (Summary, error)
	// ListAll is the tenant-wide admin moderation view -- every channel,
	// every product, newest first.
	ListAll(ctx context.Context) ([]Review, error)
	Delete(ctx context.Context, id string) error
	// HasVerifiedPurchase reports whether email completed an order on
	// salesChannelID containing productID -- used at submission time to set
	// Review.Verified. A raw cross-table (sales_orders/sales_order_line_items
	// /skus/variants) query, not a domain repository call, since Review's
	// own bounded context has no business owning an oms/pms query -- see
	// infrastructure/repositories/review_repo.go.
	HasVerifiedPurchase(ctx context.Context, salesChannelID, productID, email string) (bool, error)
}
