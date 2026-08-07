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
	// MarkOrphanedRunningAsFailed recovers jobs stuck in "running" after a
	// scheduler crash/redeploy -- see ImportJobRepository's identical method
	// for the full rationale. Returns the count recovered.
	MarkOrphanedRunningAsFailed(ctx context.Context, errMsg string, finishedAt time.Time) (int64, error)
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
	ListByReceivingRecord(ctx context.Context, receivingRecordID string) ([]ReceivingDiscrepancy, error)
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
	MarkOrphanedRunningAsFailed(ctx context.Context, errMsg string, finishedAt time.Time) (int64, error)
}

type BookingRepository interface {
	Create(ctx context.Context, b *Booking) error
	FindByID(ctx context.Context, id string) (*Booking, error)
	// FindActiveByPurchaseOrder returns the PO's current in_progress or
	// paused booking, if any -- StartOrResumeBookingUseCase reuses it
	// instead of creating a duplicate. Nil (not an error) when none exists.
	FindActiveByPurchaseOrder(ctx context.Context, purchaseOrderID string) (*Booking, error)
	Update(ctx context.Context, b *Booking) error
}

type BookingScanRepository interface {
	Create(ctx context.Context, s *BookingScan) error
	ListByBooking(ctx context.Context, bookingID string) ([]BookingScan, error)
}
