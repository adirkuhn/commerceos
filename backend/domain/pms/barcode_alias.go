package pms

import "time"

// BarcodeAlias maps an alternate barcode (differing from the primary unit
// barcode on SKU.Barcode) to a SKU, with a multiplier -- scanning it books
// UnitsPerScan units at once. Originally built (Phase 5) for case/box
// barcodes (UnitsPerScan > 1), then reused for a second case:
// UnitsPerScan == 1 aliases, where the SKU legitimately has more than one
// unit barcode (e.g. the same product repackaged by two suppliers, each
// printing its own EAN) -- same price, same SKU, just another code that
// resolves to it. Both cases share one table/lookup path.
type BarcodeAlias struct {
	ID       string
	TenantID string
	SKUID    string
	Code     string
	// UnitsPerScan >= 1: > 1 for a case/box multiplier, == 1 for a plain
	// alternate unit barcode.
	UnitsPerScan int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
