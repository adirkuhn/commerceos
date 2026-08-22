package procurement

import "errors"

var (
	ErrPurchaseOrderNotFound = errors.New("purchase order not found")
	ErrLineItemNotFound      = errors.New("purchase order line item not found")
	// ErrInvalidQuantity guards PurchaseOrderLineItem.Quantity: must be positive.
	ErrInvalidQuantity = errors.New("quantity must be positive")
	// ErrInvalidUnitsPerPurchasedItem guards
	// PurchaseOrderLineItem.UnitsPerPurchasedItem: must be positive when
	// explicitly provided (zero/unset defaults to 1 instead of erroring).
	ErrInvalidUnitsPerPurchasedItem = errors.New("units per purchased item must be positive")
	ErrDocumentNotFound             = errors.New("purchase order document not found")
	ErrParsingJobNotFound           = errors.New("document parsing job not found")
	ErrUnsupportedFileType          = errors.New("unsupported file type: must be pdf, csv, or xlsx")
	ErrLineItemAlreadyLinked        = errors.New("line item is already linked to a sku")
	// ErrMissingRequiredColumns is returned by the deterministic CSV/XLSX
	// parser when it can't find a quantity column and a description-or-sku
	// column in the header row.
	ErrMissingRequiredColumns = errors.New("document is missing required columns: need quantity and description or sku")
	// ErrPDFHasNoExtractableText means the PDF is likely a scanned image
	// with no text layer -- the AI call is skipped rather than wasted on
	// an empty document.
	ErrPDFHasNoExtractableText = errors.New("pdf has no extractable text (may be a scanned image)")
	// ErrAIParsingNotConfigured means a PDF needs the AI path but no
	// AIDocumentParser adapter was wired up (no API key configured).
	ErrAIParsingNotConfigured = errors.New("AI document parsing is not configured")
	// ErrNoLineItemsExtracted means a document was successfully read but
	// zero line items came out of it -- for the deterministic CSV/XLSX
	// path, a quantity column was found but every data row failed to parse
	// against it (distinct from ErrMissingRequiredColumns, no column found
	// at all); for the AI/PDF path, the model returned an empty
	// line_items array. Either way this always surfaces as a failed job
	// instead of silently succeeding with zero line items.
	ErrNoLineItemsExtracted    = errors.New("no line items could be extracted from the document")
	ErrReceivingRecordNotFound = errors.New("receiving record not found")
	ErrBookingNotFound         = errors.New("booking not found")
	// ErrBookingNotActive guards actions (scan, pause, complete) against a
	// booking that's already completed.
	ErrBookingNotActive = errors.New("booking is not active")
	// ErrWarehouseRequired guards the first action that actually writes
	// stock against a PO (StartOrResumeBookingUseCase, or the receiving
	// review's "Receive stock" action) when the PO has no warehouse
	// assigned yet and none was supplied to assign.
	ErrWarehouseRequired = errors.New("a warehouse is required")
	// ErrScanCodeOrSKURequired guards ScanBookingItemUseCase's two input
	// paths: a scanned barcode (Code) or a directly-known SKU (SKUID, e.g.
	// tapping a booking line item row) -- exactly one must be provided.
	ErrScanCodeOrSKURequired = errors.New("either a scanned code or a sku_id is required")
	// ErrInvalidBookingScanActionType guards
	// ScanBookingItemInput.ActionType against anything other than the four
	// BookingScanActionType values.
	ErrInvalidBookingScanActionType = errors.New("invalid booking scan action type")
	// ErrPendingBatchNotFound is returned during CompleteBookingUseCase's
	// replay when a correction/damaged/lost row references a batch (by
	// SKUID+LotNumber) that no earlier receive row in the same booking ever
	// staged -- there's nothing to resolve it against.
	ErrPendingBatchNotFound = errors.New("no batch found for this correction -- the receive that would have created it is missing")
	// ErrBookingScanNotFound guards DeleteBookingScanUseCase against a
	// scanId that doesn't belong to the given booking (wrong booking,
	// already deleted, or never existed).
	ErrBookingScanNotFound = errors.New("booking scan not found")
	ErrDiscrepancyNotFound = errors.New("receiving discrepancy not found")
	// ErrDiscrepancyMismatch guards LinkReceivingDiscrepanciesUseCase (wrong
	// receiving_record_id pairing, or either side isn't the expected
	// unexpected_item/missing_item type) and FlagDamagedLineItemUseCase (the
	// line item doesn't belong to the receiving record's purchase order).
	ErrDiscrepancyMismatch            = errors.New("receiving discrepancies do not match for this action")
	ErrDiscrepancyAlreadyAcknowledged = errors.New("receiving discrepancy is already acknowledged")
	// ErrInvalidDamagedQuantity guards FlagDamagedLineItemUseCase: must be
	// positive.
	ErrInvalidDamagedQuantity = errors.New("damaged quantity must be positive")
)
