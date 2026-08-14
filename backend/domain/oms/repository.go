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
	// note, StockDecremented) -- never line items, which are immutable
	// once created (see UpdateSalesOrderUseCase's doc).
	Update(ctx context.Context, id string, fields map[string]any) error
	Delete(ctx context.Context, id string) error
}

type SalesOrderLineItemRepository interface {
	Create(ctx context.Context, li *SalesOrderLineItem) error
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]SalesOrderLineItem, error)
	DeleteBySalesOrder(ctx context.Context, salesOrderID string) error
}

// SalesOrderLineItemStockMovement links a line item to the stock movement
// it caused -- see the migration's doc comment for why this is a dedicated
// link table rather than a column on stock_movements.
type SalesOrderLineItemStockMovement struct {
	ID                   string
	TenantID             string
	SalesOrderLineItemID string
	StockMovementID      string
	CreatedAt            time.Time
}

type SalesOrderLineItemStockMovementRepository interface {
	Create(ctx context.Context, l *SalesOrderLineItemStockMovement) error
}
