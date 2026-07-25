package pms

import "time"

type SKUStatus string

const (
	SKUStatusActive   SKUStatus = "active"
	SKUStatusInactive SKUStatus = "inactive"
)

type SKU struct {
	ID             string
	TenantID       string
	VariantID      string
	SKUCode        string
	Barcode        string
	// PriceMinorUnits is the price in the smallest currency unit (e.g. cents for USD).
	PriceMinorUnits int64
	CurrencyCode    string
	Status          SKUStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
