package oms

import "context"

type ListSalesOrdersFilter struct {
	Limit  int
	Offset int
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
}

type SalesOrderLineItemRepository interface {
	Create(ctx context.Context, li *SalesOrderLineItem) error
	ListBySalesOrder(ctx context.Context, salesOrderID string) ([]SalesOrderLineItem, error)
}
