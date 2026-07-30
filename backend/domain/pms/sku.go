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
	Status    SKUStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
