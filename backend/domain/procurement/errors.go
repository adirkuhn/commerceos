package procurement

import "errors"

var (
	ErrPurchaseOrderNotFound = errors.New("purchase order not found")
	ErrLineItemNotFound      = errors.New("purchase order line item not found")
	// ErrInvalidQuantity guards PurchaseOrderLineItem.Quantity: must be positive.
	ErrInvalidQuantity       = errors.New("quantity must be positive")
	ErrDocumentNotFound      = errors.New("purchase order document not found")
	ErrParsingJobNotFound    = errors.New("document parsing job not found")
	ErrUnsupportedFileType   = errors.New("unsupported file type: must be pdf, csv, or xlsx")
	ErrLineItemAlreadyLinked = errors.New("line item is already linked to a sku")
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
)
