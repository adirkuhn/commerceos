package inventory

import "context"

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
}

// StockMovementRepository is append-only by design — a ledger entry is
// never updated or deleted; corrections happen via a new adjustment entry.
type StockMovementRepository interface {
	Create(ctx context.Context, m *StockMovement) error
	ListBySKU(ctx context.Context, skuID string) ([]StockMovement, error)
	ListByBatch(ctx context.Context, batchID string) ([]StockMovement, error)
}
