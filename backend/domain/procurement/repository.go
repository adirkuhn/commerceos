package procurement

import (
	"context"
	"time"
)

type PurchaseOrderRepository interface {
	Create(ctx context.Context, po *PurchaseOrder) error
	FindByID(ctx context.Context, id string) (*PurchaseOrder, error)
	List(ctx context.Context) ([]PurchaseOrder, error)
	Update(ctx context.Context, po *PurchaseOrder) error
	Delete(ctx context.Context, id string) error
	// UpdateSuggestions writes only SuggestedSupplierName/SuggestedCurrencyCode
	// -- deliberately separate from Update so it can never be reached by the
	// header-save flow, which knows nothing about these fields.
	UpdateSuggestions(ctx context.Context, id string, supplierName, currencyCode string) error
	// MarkPotentialDuplicate writes only IsPotentialDuplicate/
	// DuplicateOfPurchaseOrderID -- same narrow-write rationale as
	// UpdateSuggestions, called by AttachDocumentToPurchaseOrderUseCase
	// when a newly-attached document's content hash matches one already on
	// a different PO.
	MarkPotentialDuplicate(ctx context.Context, id, duplicateOfPurchaseOrderID string) error
	// ClearPotentialDuplicate is MarkPotentialDuplicate's inverse -- same
	// narrow-write rationale, called by DeletePurchaseOrderUseCase when a
	// PO's only remaining duplicate match no longer exists.
	ClearPotentialDuplicate(ctx context.Context, id string) error
	// FindDuplicatesOf returns every PO currently pointing at id via
	// DuplicateOfPurchaseOrderID -- deleting a PO nulls that column (ON
	// DELETE SET NULL) but leaves IsPotentialDuplicate stuck true unless
	// something recomputes it; DeletePurchaseOrderUseCase uses this list to
	// do that (re-point to another surviving duplicate, or clear the flag).
	FindDuplicatesOf(ctx context.Context, id string) ([]PurchaseOrder, error)
}

// PurchaseOrderLineItemRepository is a sub-resource repository, same shape
// as pms.SKURepository nesting under a Variant.
type PurchaseOrderLineItemRepository interface {
	Create(ctx context.Context, li *PurchaseOrderLineItem) error
	FindByID(ctx context.Context, id string) (*PurchaseOrderLineItem, error)
	ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]PurchaseOrderLineItem, error)
	Update(ctx context.Context, li *PurchaseOrderLineItem) error
	Delete(ctx context.Context, id string) error
	// SumByPurchaseOrders returns quantity*unit_cost_minor_units summed per
	// PO, keyed by purchase_order_id -- used by ListPurchaseOrdersUseCase to
	// attach PurchaseOrder.TotalMinorUnits to every row in one query instead
	// of loading every PO's full line items. A PO with no rows in the
	// result has zero line items (sum is 0, not present in the map).
	SumByPurchaseOrders(ctx context.Context, purchaseOrderIDs []string) (map[string]int64, error)
	// ListPurchaseOrderIDsReceivedBySKU returns the distinct IDs of every
	// received/partially_received PO with a product line item for skuID --
	// used by procurement.ListReceivedCostLinesForSKUUseCase (see its doc
	// comment) to source pms's cost-price rollup from a PO's own line items
	// when it was marked received without ever creating an inventory.Batch
	// (e.g. the desktop Edit Purchase Order form's status field has no
	// stock-booking gate of its own).
	ListPurchaseOrderIDsReceivedBySKU(ctx context.Context, skuID string) ([]string, error)
}

type PurchaseOrderDocumentRepository interface {
	Create(ctx context.Context, doc *PurchaseOrderDocument) error
	FindByID(ctx context.Context, id string) (*PurchaseOrderDocument, error)
	ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]PurchaseOrderDocument, error)
	Delete(ctx context.Context, id string) error
	// FindDuplicateCandidates returns every existing document (on any PO in
	// the tenant, RLS-scoped) with the same contentHash -- tenant-wide, not
	// scoped to purchaseOrderID or supplier, since the same file can be
	// uploaded to two supplier-less POs before either has one assigned.
	// purchaseOrderID is still passed so callers can tell a same-PO
	// re-upload apart from a genuine cross-PO duplicate.
	FindDuplicateCandidates(ctx context.Context, contentHash, purchaseOrderID string) ([]PurchaseOrderDocument, error)
}

// DocumentParsingJobRepository mirrors channels.ImportJobRepository's
// pending/running/completed|failed lifecycle and cross-tenant scheduler
// access pattern.
type DocumentParsingJobRepository interface {
	Create(ctx context.Context, job *DocumentParsingJob) error
	FindByID(ctx context.Context, id string) (*DocumentParsingJob, error)
	// ListPending is a system-level, cross-tenant query -- cmd/scheduler
	// polls it under tenant.WithSuperAdmin, same as ImportJobRepository.
	ListPending(ctx context.Context) ([]DocumentParsingJob, error)
	MarkRunning(ctx context.Context, id string, startedAt time.Time) error
	MarkCompleted(ctx context.Context, id string, finishedAt time.Time) error
	MarkFailed(ctx context.Context, id string, errMsg string, finishedAt time.Time) error
	// RecoverOrphanedRunning requeues jobs stuck in "running" after a
	// scheduler crash/redeploy back to pending (up to maxRetries attempts),
	// falling back to failed beyond that -- see
	// channels.ImportJobRepository.RecoverOrphanedRunning for the full
	// rationale. Returns (requeued, failed).
	RecoverOrphanedRunning(ctx context.Context, maxRetries int, errMsg string, finishedAt time.Time) (requeued, failed int64, err error)
}

// SupplierProductAliasRepository stores learned raw-text -> SKU corrections
// per supplier (see SupplierProductAlias).
type SupplierProductAliasRepository interface {
	Upsert(ctx context.Context, alias *SupplierProductAlias) error
	FindBySupplierAndRawKey(ctx context.Context, supplierID, rawKey string) (*SupplierProductAlias, error)
}

type ReceivingRecordRepository interface {
	Create(ctx context.Context, r *ReceivingRecord) error
	FindByID(ctx context.Context, id string) (*ReceivingRecord, error)
	ListByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]ReceivingRecord, error)
	Update(ctx context.Context, r *ReceivingRecord) error
	// FindDuplicateCandidates returns existing records with the same
	// contentHash scoped to purchaseOrderID or (when supplierID is non-nil)
	// any other PO belonging to the same supplier -- used by
	// UploadInvoiceUseCase to flag IsPotentialDuplicate before Create.
	FindDuplicateCandidates(ctx context.Context, contentHash, purchaseOrderID string, supplierID *string) ([]ReceivingRecord, error)
}

type ReceivingDiscrepancyRepository interface {
	Create(ctx context.Context, d *ReceivingDiscrepancy) error
	FindByID(ctx context.Context, id string) (*ReceivingDiscrepancy, error)
	ListByReceivingRecord(ctx context.Context, receivingRecordID string) ([]ReceivingDiscrepancy, error)
	Update(ctx context.Context, d *ReceivingDiscrepancy) error
}

// ReceivingParsingJobRepository mirrors DocumentParsingJobRepository's
// pending/running/completed|failed lifecycle and cross-tenant scheduler
// access pattern.
type ReceivingParsingJobRepository interface {
	Create(ctx context.Context, job *ReceivingParsingJob) error
	FindByID(ctx context.Context, id string) (*ReceivingParsingJob, error)
	ListPending(ctx context.Context) ([]ReceivingParsingJob, error)
	MarkRunning(ctx context.Context, id string, startedAt time.Time) error
	MarkCompleted(ctx context.Context, id string, finishedAt time.Time) error
	MarkFailed(ctx context.Context, id string, errMsg string, finishedAt time.Time) error
	// RecoverOrphanedRunning mirrors DocumentParsingJobRepository's
	// identical method -- see its doc comment for the full rationale.
	RecoverOrphanedRunning(ctx context.Context, maxRetries int, errMsg string, finishedAt time.Time) (requeued, failed int64, err error)
}

type BookingRepository interface {
	Create(ctx context.Context, b *Booking) error
	FindByID(ctx context.Context, id string) (*Booking, error)
	// FindActiveByPurchaseOrder returns the PO's current in_progress or
	// paused booking, if any -- StartOrResumeBookingUseCase reuses it
	// instead of creating a duplicate. Nil (not an error) when none exists.
	FindActiveByPurchaseOrder(ctx context.Context, purchaseOrderID string) (*Booking, error)
	Update(ctx context.Context, b *Booking) error
	// ListCompleted returns every Completed booking for the tenant, most
	// recently completed first -- the discrepancies report's data source
	// (see application/procurement.ListBookingDiscrepanciesUseCase).
	ListCompleted(ctx context.Context) ([]Booking, error)
	// ListCompletedByPurchaseOrder returns every Completed booking (one per
	// delivery) of the PO, oldest first.
	ListCompletedByPurchaseOrder(ctx context.Context, purchaseOrderID string) ([]Booking, error)
}

type BookingScanRepository interface {
	Create(ctx context.Context, s *BookingScan) error
	ListByBooking(ctx context.Context, bookingID string) ([]BookingScan, error)
	// MarkReplayed records that CompleteBookingUseCase's replay applied
	// this row to real inventory -- batchID is the row's resolved real
	// batch (nil for general ledger), persisted alongside so a skipped
	// re-replay of this row can still resolve a LATER row's reference to
	// the same not-yet-real lot without recomputing anything.
	MarkReplayed(ctx context.Context, id string, replayedAt time.Time, batchID *string) error
	// Delete removes a staged row before it's ever been replayed to real
	// inventory -- see DeleteBookingScanUseCase, which is the only caller
	// and enforces that the row's booking is still in_progress before this
	// runs. Idempotent like every other Delete in this codebase; the use
	// case itself is what checks the row actually belongs to the booking.
	Delete(ctx context.Context, id string) error
}
