package pms

import "time"

// BarcodeAlias maps a case/box barcode (which differs from the unit
// barcode printed on SKU.Barcode) to a SKU, with a multiplier -- scanning a
// box barcode books UnitsPerScan units at once instead of one. Confirmed
// during Phase 5 planning that no such concept existed before the scanner
// app needed it.
type BarcodeAlias struct {
	ID       string
	TenantID string
	SKUID    string
	Code     string
	// UnitsPerScan is always >= 1 -- a plain unit barcode would just be
	// SKU.Barcode itself, not an alias row, so every alias represents a
	// multi-unit container by definition.
	UnitsPerScan int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
