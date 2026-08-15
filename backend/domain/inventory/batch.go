package inventory

import "time"

// Batch status values. A batch starts Active and is flipped to Expired by
// the scheduler's auto-expire job (see application/inventory.ExpireBatchUseCase)
// once its ExpiryDate is more than tenant_settings.batch_expiry_threshold_days
// in the past. This is a distinct, persisted signal from the live
// ExpiryDate-vs-now() check computeStockLevel does on every read — Status
// only drives the expired-batch review queue, never the sync-quantity math.
const (
	BatchStatusActive  = "active"
	BatchStatusExpired = "expired"
)

// Batch is an optional finer-grained tracking unit for a SKU's stock —
// a lot received at a given time, in a given warehouse, with an optional
// expiry. Its current quantity is never stored on the batch itself; it's
// derived by summing the StockMovement ledger entries that reference it
// (see StockMovementRepository.ListByBatch / application layer). Movements
// against a SKU don't have to reference a batch at all — plain, un-lotted
// stock (e.g. a bulk import with no batch/expiry info) is a StockMovement
// with BatchID nil.
type Batch struct {
	ID          string
	TenantID    string
	SKUID       string
	WarehouseID string
	LotNumber   string
	ExpiryDate  *time.Time
	ReceivedAt  time.Time
	// Status is one of the BatchStatus* constants above.
	Status string
	// ReviewedAt is set once a user has acted on this batch from the
	// expired-batch review queue (write-off, extend, or dismiss) --
	// ListPendingReview only returns rows where this is still nil, even
	// after Status flips back to Active or Status stays Expired forever
	// (the dismiss case). See application/inventory's
	// WriteOffExpiredBatchUseCase / ExtendBatchExpiryUseCase /
	// DismissExpiredBatchUseCase.
	ReviewedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	// Quantity is transient -- populated by ListBatchesUseCase from the
	// StockMovement ledger, never persisted on the batch row itself (see
	// the type doc comment above).
	Quantity int64
	// IsGeneralLedger marks a synthesized row (ID/LotNumber/ExpiryDate all
	// zero-value) representing a warehouse's un-lotted stock rather than a
	// real batch -- see ListBatchesUseCase.
	IsGeneralLedger bool
}
