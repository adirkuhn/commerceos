package procurement

import "time"

// SupplierProductAlias is a learned mapping from a human correction made in
// the review UI: the next time this supplier's document contains the same
// raw text (RawKey), the matching pipeline links straight to SKUID instead
// of falling through to AI/needs_review again.
type SupplierProductAlias struct {
	ID         string
	TenantID   string
	SupplierID string
	// RawKey is the normalized (lowercased, trimmed) raw SKU or description
	// text extracted from the supplier's document.
	RawKey string
	SKUID  string
	// UnitsPerPurchasedItem is the learned "how many stock units does one
	// purchased item of this raw text represent" -- e.g. a supplier's
	// "Rapadurinha 36x25gr" line links to the individual 25g unit SKU with
	// this set to 36. Always >= 1 (1 is the ordinary case: one purchased
	// item is one stock unit). Written alongside SKUID whenever a human
	// links/relinks a line item (see LinkLineItemToSKUUseCase), so the same
	// supplier text carries its multiplier forward automatically next time,
	// same as the SKU link itself.
	UnitsPerPurchasedItem int64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}
