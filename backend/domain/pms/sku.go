package pms

import "time"

type SKUStatus string

const (
	SKUStatusActive   SKUStatus = "active"
	SKUStatusInactive SKUStatus = "inactive"
)

type SKU struct {
	ID        string
	TenantID  string
	VariantID string
	SKUCode   string
	Barcode   string
	// PriceMinorUnits is the price in the smallest currency unit (e.g. cents for USD).
	PriceMinorUnits int64
	CurrencyCode    string
	// WeightGrams and the *MM dimensions are nil when unknown — normalized
	// to fixed SI units (like PriceMinorUnits normalizes money) so values
	// from any future channel are comparable without ambiguity.
	WeightGrams *int64
	LengthMM    *int64
	WidthMM     *int64
	HeightMM    *int64
	// IsVirtual marks a digital/non-shippable SKU — never has weight or
	// dimensions set (see ErrVirtualSKUCannotHaveDimensions).
	IsVirtual bool
	// TracksExpiry marks a SKU whose inventory batches carry a
	// meaningful expiry date (e.g. food, cosmetics) — the frontend uses
	// this to decide whether to require/show an expiry date when
	// creating a batch. Purely advisory: a batch can still be created
	// with no expiry regardless of this flag (e.g. general-ledger stock,
	// or a lot whose expiry just isn't known yet).
	TracksExpiry bool
	Status       SKUStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
