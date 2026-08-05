package inventory

import (
	"context"
	"time"
)

// ExpiringBatch is a read-model row for the tenant-wide "expiring soon"
// report — a Batch enriched with its SKU code, product name, warehouse
// name, and current remaining quantity. This join lives entirely at the
// infrastructure layer for this one reporting query; it isn't a call into
// another domain's repository, and nothing here is written back.
type ExpiringBatch struct {
	BatchID       string
	SKUID         string
	SKUCode       string
	ProductName   string
	WarehouseID   string
	WarehouseName string
	LotNumber     string
	ExpiryDate    time.Time
	Quantity      int64
}

type WarehouseRepository interface {
	Create(ctx context.Context, w *Warehouse) error
	FindByID(ctx context.Context, id string) (*Warehouse, error)
	List(ctx context.Context) ([]Warehouse, error)
	Update(ctx context.Context, w *Warehouse) error
	Delete(ctx context.Context, id string) error
}

type BatchRepository interface {
	Create(ctx context.Context, b *Batch) error
	FindByID(ctx context.Context, id string) (*Batch, error)
	ListBySKU(ctx context.Context, skuID string) ([]Batch, error)
	Update(ctx context.Context, b *Batch) error
	// Delete removes a batch — callers must ensure no StockMovement
	// references it first (see application/inventory's DeleteBatchUseCase);
	// the DB's ON DELETE RESTRICT FK would reject it anyway, but the app
	// layer check gives a clean ErrBatchHasMovements instead of a raw
	// constraint-violation error.
	Delete(ctx context.Context, id string) error
	// ListExpiringSoon returns every batch (across all SKUs) expiring
	// within the given number of days that still has stock remaining
	// (Quantity > 0), soonest-expiring first.
	ListExpiringSoon(ctx context.Context, withinDays int) ([]ExpiringBatch, error)
}

// StockMovementRepository is append-only by design — a ledger entry is
// never updated or deleted; corrections happen via a new adjustment entry.
type StockMovementRepository interface {
	Create(ctx context.Context, m *StockMovement) error
	ListBySKU(ctx context.Context, skuID string) ([]StockMovement, error)
	ListByBatch(ctx context.Context, batchID string) ([]StockMovement, error)
}
