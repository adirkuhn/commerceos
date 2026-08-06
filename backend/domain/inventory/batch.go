package inventory

import "time"

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
	CreatedAt   time.Time
	UpdatedAt   time.Time
	// Quantity is transient -- populated by ListBatchesUseCase from the
	// StockMovement ledger, never persisted on the batch row itself (see
	// the type doc comment above).
	Quantity int64
	// IsGeneralLedger marks a synthesized row (ID/LotNumber/ExpiryDate all
	// zero-value) representing a warehouse's un-lotted stock rather than a
	// real batch -- see ListBatchesUseCase.
	IsGeneralLedger bool
}
