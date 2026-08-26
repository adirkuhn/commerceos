package shipping

import (
	"context"
	"time"
)

// MyParcelFulfilmentOrder tracks one sales order's export via MyParcel's
// Orders/Fulfilment API (ExportMyParcelOrderUseCase) -- one row per
// (connection, sales order), so re-clicking "Create label" is idempotent
// (see FindByConnectionAndSalesOrder) rather than creating a duplicate
// MyParcel order on every retry.
type MyParcelFulfilmentOrder struct {
	ID           string
	TenantID     string
	ConnectionID string
	SalesOrderID string
	MyParcelUUID string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type MyParcelFulfilmentOrderRepository interface {
	Create(ctx context.Context, o *MyParcelFulfilmentOrder) error
	// FindByConnectionAndSalesOrder returns nil (not an error) when this
	// order hasn't been exported yet.
	FindByConnectionAndSalesOrder(ctx context.Context, connectionID, salesOrderID string) (*MyParcelFulfilmentOrder, error)
}
