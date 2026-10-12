package inventory

import (
	"context"
	"io"
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

// ExpiredBatchReview is the per-tenant read-model row returned by
// BatchRepository.ListPendingReview -- an Expired batch a user hasn't
// acted on yet (ReviewedAt still nil), enriched the same way ExpiringBatch
// is for the expiring-soon report.
type ExpiredBatchReview struct {
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
	SKUID    string
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
	// ListPendingReview returns every Expired, not-yet-reviewed batch for
	// the current tenant, soonest-expired first -- the expired-batch
	// review queue's data source.
	ListPendingReview(ctx context.Context) ([]ExpiredBatchReview, error)
	// MarkReviewed sets ReviewedAt to now, removing the batch from the
	// review queue without changing Status or ExpiryDate (the "Dismiss"
	// action; also called by the write-off action after its movement is
	// recorded).
	MarkReviewed(ctx context.Context, id string) error
	// ExtendExpiry pushes a batch's ExpiryDate out and flips Status back
	// to Active (the "Extend" review-queue action) -- ReviewedAt is left
	// nil so the batch reappears in the queue if it expires again later.
	ExtendExpiry(ctx context.Context, id string, newExpiryDate time.Time) error
}

// BatchQueueCounter counts the rows of BatchRepository's two batch work
// queues without loading them -- CountExpiringSoon is len(ListExpiringSoon)
// and CountPendingReview is len(ListPendingReview), same rules exactly.
// Backs the dashboard's "Today" work-queue counts.
type BatchQueueCounter interface {
	CountExpiringSoon(ctx context.Context, withinDays int) (int64, error)
	CountPendingReview(ctx context.Context) (int64, error)
}

// DailyStockFlow is one day's inbound/outbound totals for
// DashboardStockReportRepository.DailyFlow -- In is the sum of positive
// QuantityDelta that day, Out the sum (as a positive number) of negative
// QuantityDelta, both across every SKU/warehouse for the tenant.
type DailyStockFlow struct {
	Date time.Time
	In   int64
	Out  int64
}

// DashboardStockReportRepository backs the dashboard's stock figures.
// TotalOnHand is the literal tenant-wide ledger sum -- unlike
// application/inventory.computeStockLevel's per-SKU logic, it has no
// syncable/nearest-expiry-batch nuance, just every movement added up.
type DashboardStockReportRepository interface {
	TotalOnHand(ctx context.Context) (int64, error)
	// DailyFlow returns one row per day from `since` (inclusive) to today,
	// oldest first, including days with no movements at all (In/Out both 0).
	// Days are local days in timezone (settings.TenantSettings.ReportingTimezone).
	DailyFlow(ctx context.Context, since time.Time, timezone string) ([]DailyStockFlow, error)
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

// DiscardRecordRepository backs the discarded-items report -- append-only,
// same reasoning as StockMovementRepository (a discard is a fact that
// happened, never edited/deleted).
type DiscardRecordRepository interface {
	Create(ctx context.Context, d *DiscardRecord) error
	// List returns the tenant-wide discarded-items report, newest first,
	// narrowed by filter.
	List(ctx context.Context, filter DiscardRecordFilter) ([]DiscardRecordReport, int64, error)
}

// FileStorage is this domain's own copy of pms.FileStorage's shape (same
// three methods) -- domain packages don't import each other (see CLAUDE.md),
// so it's redeclared here rather than shared; infrastructure/storage.S3Storage
// satisfies both structurally with no adapter code needed.
type FileStorage interface {
	Save(ctx context.Context, tenantID, filename string, data io.Reader) (url string, err error)
	Delete(ctx context.Context, url string) error
}
