package oms

import (
	"context"
	"time"
)

// SalesOrderSortField is the allow-list of columns List can sort by --
// kept as a closed type (rather than a raw string) so an invalid/unknown
// value can never reach the repository's ORDER BY clause.
type SalesOrderSortField string

const (
	SalesOrderSortOccurredAt SalesOrderSortField = "occurred_at"
	SalesOrderSortCustomer   SalesOrderSortField = "customer"
	SalesOrderSortSource     SalesOrderSortField = "source"
	SalesOrderSortStatus     SalesOrderSortField = "status"
	SalesOrderSortTotal      SalesOrderSortField = "total"
)

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type ListSalesOrdersFilter struct {
	Limit  int
	Offset int
	// SortBy/SortDir empty means the repository's default (occurred_at,
	// desc -- newest first).
	SortBy  SalesOrderSortField
	SortDir SortDirection
	// NeedsStockReview/NeedsSKUReview nil means "don't filter on this" --
	// set to a non-nil true to power the "Stock Issues"/"Unmatched SKUs"
	// admin pages, which list only flagged orders.
	NeedsStockReview *bool
	NeedsSKUReview   *bool
	// Source nil means "don't filter on this" -- used by
	// cmd/repair-stock-review-flags to scan only WooCommerce-sourced orders.
	Source *SalesOrderSource
}

type SalesOrderRepository interface {
	Create(ctx context.Context, o *SalesOrder) error
	FindByID(ctx context.Context, id string) (*SalesOrder, error)
	// FindByExternalID looks up an already-received sale by its source's
	// own reference -- the idempotency check ReceivePOSSalesOrderUseCase
	// runs before creating anything. Nil (not an error) when none exists.
	FindByExternalID(ctx context.Context, source SalesOrderSource, externalOrderID string) (*SalesOrder, error)
	// List returns every sales order for the current tenant (RLS-scoped),
	// newest first, paginated.
	List(ctx context.Context, filter ListSalesOrdersFilter) ([]SalesOrder, int64, error)
	// ListByConnection returns every order stamped with this connection's
	// ID (see SalesOrder.ConnectionID) -- used by "delete all imported
	// products" to find that connection's own orders. Unpaginated, same
	// convention as channels.ProductChannelListingRepository.ListByConnection.
	ListByConnection(ctx context.Context, connectionID string) ([]SalesOrder, error)
	// Update patches header fields (status, customer, billing/shipping,
	// note, NeedsStockReview, NeedsSKUReview) -- never line items, which
	// are immutable once created (see UpdateSalesOrderUseCase's doc), and
	// never StockDecremented, which must only ever change through
	// TrySetStockDecremented's conditional write.
	Update(ctx context.Context, id string, fields map[string]any) error
	// TrySetStockDecremented atomically flips StockDecremented from
	// `from` to `to`, only if the row's current value is still `from` --
	// see the implementation's doc for why this can't be a plain field in
	// Update's fields map (the double-decrement race it closes).
	TrySetStockDecremented(ctx context.Context, id string, from, to bool) (bool, error)
	Delete(ctx context.Context, id string) error
}

type SalesOrderLineItemRepository interface {
	Create(ctx context.Context, li *SalesOrderLineItem) error
	FindByID(ctx context.Context, id string) (*SalesOrderLineItem, error)
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]SalesOrderLineItem, error)
	DeleteBySalesOrder(ctx context.Context, salesOrderID string) error
	// Update patches a line item's mutable fields -- today only SKUID
	// (LinkLineItemSKUUseCase) and StockStatus (RetryStockDecrementUseCase,
	// ImportWooOrdersUseCase, UpdateSalesOrderUseCase). Everything else
	// about a line item is immutable once created (see SalesOrderRepository
	// .Update's doc).
	Update(ctx context.Context, id string, fields map[string]any) error
}

// SalesOrderLineItemStockMovement links a line item to the stock movement
// it caused -- see the migration's doc comment for why this is a dedicated
// link table rather than a column on stock_movements.
type SalesOrderLineItemStockMovement struct {
	ID                   string
	TenantID             string
	SalesOrderLineItemID string
	StockMovementID      string
	// BatchID and QuantityDelta mirror the linked StockMovement's own
	// fields at the time it was created -- denormalized so a restock can
	// reverse exactly what was actually decremented (which real batch, how
	// much) without a second lookup into stock_movements. nil BatchID means
	// the general/un-lotted ledger, same convention as inventory.Batch.
	BatchID       *string
	QuantityDelta int64
	CreatedAt     time.Time
}

type SalesOrderLineItemStockMovementRepository interface {
	Create(ctx context.Context, l *SalesOrderLineItemStockMovement) error
	// ListByLineItem returns every movement linked to lineItemID, in the
	// order they were created -- used to reverse a decrement batch-by-batch
	// when an order restocks (see application/oms's restock helpers).
	ListByLineItem(ctx context.Context, lineItemID string) ([]SalesOrderLineItemStockMovement, error)
	// ExistsForLineItem reports whether lineItemID already has a recorded
	// stock movement linked to it -- used by
	// cmd/repair-stock-review-flags to distinguish a line item whose
	// decrement genuinely never ran (the bug this repairs) from one that
	// already succeeded.
	ExistsForLineItem(ctx context.Context, lineItemID string) (bool, error)
}
