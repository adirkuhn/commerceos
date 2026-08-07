package procurement

import "time"

// LineItemType classifies what a parsed row actually represents. Rows like
// shipping/discount/fees are not products -- they must never be forced
// through SKU matching, but their cost still belongs on the document.
type LineItemType string

const (
	LineItemTypeProduct     LineItemType = "product"
	LineItemTypeShipping    LineItemType = "shipping"
	LineItemTypeDiscount    LineItemType = "discount"
	LineItemTypeCreditNote  LineItemType = "credit_note"
	LineItemTypeFee         LineItemType = "fee"
	LineItemTypeOtherCharge LineItemType = "other_charge"
)

// IsProduct reports whether this type represents an actual product line --
// the only kind that needs a SKU match. Empty is treated as product: the
// deterministic CSV/XLSX parser has no classification signal and always
// leaves LineType unset, so this preserves its existing all-rows-are-
// products behavior.
func (t LineItemType) IsProduct() bool {
	return t == "" || t == LineItemTypeProduct
}

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
	// LineType is the AI's classification (see LineItemType) -- always
	// empty from the deterministic CSV/XLSX parser, which has no signal to
	// classify from and treats every row as a product per IsProduct.
	LineType LineItemType
	// ExpiryDate is nil unless the document printed one (common on supplier
	// invoices, rare on purchase orders) -- only consulted by the Phase 4
	// receiving diff, ignored by Phase 3's catalog-matching pipeline.
	ExpiryDate *time.Time
}
