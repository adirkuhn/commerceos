package procurement

import "time"

type PurchaseOrderStatus string

const (
	PurchaseOrderStatusDraft             PurchaseOrderStatus = "draft"
	PurchaseOrderStatusSent              PurchaseOrderStatus = "sent"
	PurchaseOrderStatusPartiallyReceived PurchaseOrderStatus = "partially_received"
	PurchaseOrderStatusReceived          PurchaseOrderStatus = "received"
	PurchaseOrderStatusCancelled         PurchaseOrderStatus = "cancelled"
	// PurchaseOrderStatusNeedsReview is set by the document-parsing pipeline
	// (see match_extracted_line_items.go) when a line item can't be linked
	// to a product automatically.
	PurchaseOrderStatusNeedsReview PurchaseOrderStatus = "needs_review"
)

type PurchaseOrder struct {
	ID       string
	TenantID string
	// WarehouseID is nil until stock is actually received against this PO
	// -- assigned by StartOrResumeBookingUseCase (scanner app) or the
	// receiving review's "Receive stock" action (desktop), whichever
	// happens first, and never asked for at order time. A PO can be
	// created, sent, and fully line-itemed with no warehouse decided yet.
	WarehouseID *string
	// SupplierID is nil when a PO was created from an uploaded document
	// without picking a supplier up front -- always needs_review until a
	// human links or creates one (see
	// CreateSupplierAndAssignToPurchaseOrderUseCase and
	// recomputePurchaseOrderReviewStatus).
	SupplierID   *string
	Status       PurchaseOrderStatus
	CurrencyCode string
	ExpectedDate *time.Time
	Notes        string
	// SuggestedSupplierName/SuggestedCurrencyCode are informational only --
	// set by RunDocumentParsingJobUseCase from a PDF's AI-extracted
	// supplier name and a deterministic currency-code guess, for the
	// review UI to pre-fill rather than the human typing from scratch.
	// Empty means no suggestion; neither ever overwrites SupplierID or
	// CurrencyCode directly.
	SuggestedSupplierName string
	SuggestedCurrencyCode string
	// IsPotentialDuplicate/DuplicateOfPurchaseOrderID are set when a
	// document attached to this PO has the same content hash as a
	// document already attached to a *different* PO (see
	// AttachDocumentToPurchaseOrderUseCase) -- tenant-wide, not scoped to
	// supplier, since the same file can land on two supplier-less POs
	// before either has one assigned.
	IsPotentialDuplicate       bool
	DuplicateOfPurchaseOrderID *string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type PurchaseOrderLineItem struct {
	ID              string
	TenantID        string
	PurchaseOrderID string
	// SKUID is nil when the line item hasn't been linked to a product yet
	// -- set immediately for manually-created line items, left nil by the
	// document-parsing pipeline when it can't confidently match (see
	// PurchaseOrderStatusNeedsReview).
	SKUID *string
	// RawSKU is the SKU/barcode text as printed on the source document, if
	// any -- kept separate from RawDescription so a manual link made in the
	// review UI can write an accurate SupplierProductAlias.RawKey (see
	// alias.go) for next time.
	RawSKU         string
	RawDescription string
	Quantity       int64
	// UnitCostMinorUnits is the smallest-currency-unit cost per unit, same
	// convention as pms.SKU.PriceMinorUnits.
	UnitCostMinorUnits int64
	// SKUAutoLinkedFromAlias is true when SKUID came from a learned
	// SupplierProductAlias guess rather than an exact-SKU match or an
	// explicit human link via the review UI. Exact-SKU matches never set
	// this (the code on the document is unambiguous, and matching is
	// supplier-independent so a supplier change can't invalidate it).
	// LinkLineItemToSKUUseCase clears it the moment a human confirms/
	// overrides the link. A supplier change on the PO clears SKUID and
	// re-runs the alias lookup against the new supplier for any item still
	// flagged true (see applyLearnedAliasesToUnmatchedLineItems).
	SKUAutoLinkedFromAlias bool
	// LineType classifies this row (see LineItemType) -- charge-type rows
	// (shipping/discount/fee/credit_note/other_charge) never need a SKU
	// match. Set from the AI's classification at parse time (or left ""/
	// product for the CSV/XLSX path and manually-added items), correctable
	// via the review UI same as SKUID.
	LineType LineItemType
	// LineTypeConfirmed is false when LineType came from the AI's guess,
	// true once a human has explicitly picked/confirmed it (manually-added
	// line items are confirmed immediately -- there's no ambiguity to
	// review when the human just typed it in themselves).
	LineTypeConfirmed bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
