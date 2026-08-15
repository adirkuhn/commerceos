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

// BatchToExpire is the cross-tenant read-model row returned by
// BatchRepository.ListToExpire — just enough to dispatch the expire under
// the right tenant identity (see cmd/scheduler.runExpiredBatches).
type BatchToExpire struct {
	BatchID  string
	TenantID string
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
	// ListBySKUs is the bulk form of ListBySKU -- used by
	// application/inventory.ListStockLevelsUseCase to avoid one query per
	// SKU when computing a page of stock levels.
	ListBySKUs(ctx context.Context, skuIDs []string) ([]Batch, error)
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
	// ListToExpire is a cross-tenant, RLS-bypassing query (called under
	// tenant.WithSuperAdmin) for the scheduler's auto-expire job: every
	// Active batch whose ExpiryDate is more than that batch's own
	// tenant's tenant_settings.batch_expiry_threshold_days in the past.
	// A tenant with threshold 0 (disabled, the default) never appears.
	ListToExpire(ctx context.Context) ([]BatchToExpire, error)
	// MarkExpired flips a single batch's Status to BatchStatusExpired.
	MarkExpired(ctx context.Context, id string) error
}

// StockMovementRepository is append-only by design — a ledger entry is
// never updated or deleted; corrections happen via a new adjustment entry.
type StockMovementRepository interface {
	Create(ctx context.Context, m *StockMovement) error
	ListBySKU(ctx context.Context, skuID string) ([]StockMovement, error)
	ListByBatch(ctx context.Context, batchID string) ([]StockMovement, error)
	// ListBySKUs is the bulk form of ListBySKU -- see BatchRepository.ListBySKUs.
	ListBySKUs(ctx context.Context, skuIDs []string) ([]StockMovement, error)
	// ListDistinctSKUIDs returns a page of SKU IDs that have at least one
	// movement (changedSince nil) or at least one movement after
	// changedSince (non-nil) -- the "which SKUs changed" half of
	// ListStockLevelsUseCase's incremental-sync support, ordered by SKU ID
	// for stable pagination. total is the full matching count, for paging.
	ListDistinctSKUIDs(ctx context.Context, changedSince *time.Time, limit, offset int) (skuIDs []string, total int64, err error)
}
