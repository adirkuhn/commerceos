package procurement

import "time"

type ReceivingRecordStatus string

const (
	// ReceivingRecordStatusPendingReview is the only starting state -- every
	// uploaded invoice needs a human to look at the diff and book the
	// stock, even a clean one with zero discrepancies (per the plan: "this
	// should also be moved under a special tab to be checked by a human").
	ReceivingRecordStatusPendingReview ReceivingRecordStatus = "pending_review"
	// ReceivingRecordStatusApplied is set once a human explicitly marks the
	// receiving as done (see CompleteReceivingUseCase) -- stock booking
	// happens per line item before that via ApplyReceivingLineUseCase, this
	// status change doesn't itself move any stock.
	ReceivingRecordStatusApplied ReceivingRecordStatus = "applied"
)

// ReceivingRecord is one supplier invoice uploaded against a PurchaseOrder
// -- same upload->parse->review shape as PurchaseOrderDocument/
// DocumentParsingJob in Phase 3, but the parsing job diffs extracted lines
// against the PO's own line items instead of matching them to the product
// catalog.
type ReceivingRecord struct {
	ID              string
	TenantID        string
	PurchaseOrderID string
	InvoiceFileURL  string
	InvoiceFileName string
	ContentType     string
	Status          ReceivingRecordStatus
	// ContentHash is the sha256 (hex) of the uploaded file's bytes --
	// UploadInvoiceUseCase computes it once at upload time to check for
	// duplicates, see IsPotentialDuplicate.
	ContentHash string
	// IsPotentialDuplicate is set at creation when another invoice with the
	// same ContentHash already exists for this PO or another PO of the
	// same supplier -- a hint for the reviewer, never blocks the upload,
	// and never recomputed after the fact (a later dedup is the new row's
	// problem to flag, not the original's).
	IsPotentialDuplicate bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ReceivingParsingJobStatus string

const (
	ReceivingParsingJobPending   ReceivingParsingJobStatus = "pending"
	ReceivingParsingJobRunning   ReceivingParsingJobStatus = "running"
	ReceivingParsingJobCompleted ReceivingParsingJobStatus = "completed"
	ReceivingParsingJobFailed    ReceivingParsingJobStatus = "failed"
)

// ReceivingParsingJob mirrors DocumentParsingJob's lifecycle exactly, kept
// as its own table/type rather than generalizing DocumentParsingJob to
// cover both cases -- the two jobs do fundamentally different work on
// completion (link to catalog vs. diff against a PO), and a shared table
// would need a discriminator column plus branching everywhere it's read.
type ReceivingParsingJob struct {
	ID                string
	TenantID          string
	ReceivingRecordID string
	Status            ReceivingParsingJobStatus
	ErrorMessage      string
	CreatedAt         time.Time
	StartedAt         *time.Time
	FinishedAt        *time.Time
	// RetryCount tracks automatic requeue-on-restart attempts -- see
	// ReceivingParsingJobRepository.RecoverOrphanedRunning.
	RetryCount int
}

type DiscrepancyType string

const (
	// DiscrepancyTypeQuantityMismatch: matched a PO line item, but the
	// invoice's quantity differs from what was ordered.
	DiscrepancyTypeQuantityMismatch DiscrepancyType = "quantity_mismatch"
	// DiscrepancyTypeMissingItem: a PO line item has no corresponding line
	// on the invoice at all -- the supplier didn't ship it (or it's on a
	// later partial shipment).
	DiscrepancyTypeMissingItem DiscrepancyType = "missing_item"
	// DiscrepancyTypeUnexpectedItem: the invoice has a line that doesn't
	// correspond to anything on the PO.
	DiscrepancyTypeUnexpectedItem DiscrepancyType = "unexpected_item"
	// DiscrepancyTypeExpiryMismatch: the invoice declared an expiry date
	// that's already passed or is suspiciously close -- flagged since
	// there's no PO-side expiry to diff against, this is a data-quality
	// flag for the human, not a mismatch against an expectation.
	DiscrepancyTypeExpiryMismatch DiscrepancyType = "expiry_mismatch"
)

// ReceivingDiscrepancy is one line of the diff between a ReceivingRecord's
// invoice and its PurchaseOrder's line items -- purely informational (see
// package doc comment on ApplyReceivingLineUseCase for why resolving a
// discrepancy isn't itself a persisted state: booking stock via the
// inventory ledger is the thing that matters, not a checkbox on this row).
type ReceivingDiscrepancy struct {
	ID                string
	TenantID          string
	ReceivingRecordID string
	// LineItemID is nil for unexpected_item (no corresponding PO line to
	// point at).
	LineItemID  *string
	Type        DiscrepancyType
	ExpectedQty int64
	ReceivedQty int64
	Details     string
	CreatedAt   time.Time
}
