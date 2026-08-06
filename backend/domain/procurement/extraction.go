package procurement

import "time"

// ExtractedLineItem is one row parsed out of a supplier document, before
// matching against the product catalog (Phase 3) or diffing against a PO's
// line items (Phase 4 receiving). Produced by either the deterministic
// CSV/XLSX parser or an AIDocumentParser adapter -- downstream code treats
// both sources identically.
type ExtractedLineItem struct {
	RawSKU             string
	RawDescription     string
	Quantity           int64
	UnitCostMinorUnits int64
	// ExpiryDate is nil unless the document printed one (common on supplier
	// invoices, rare on purchase orders) -- only consulted by the Phase 4
	// receiving diff, ignored by Phase 3's catalog-matching pipeline.
	ExpiryDate *time.Time
}
