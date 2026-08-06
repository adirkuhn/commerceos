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
	RawKey    string
	SKUID     string
	CreatedAt time.Time
	UpdatedAt time.Time
}
