package pms

import "time"

// ScaleCodeAlias maps an additional 5-digit scale code to a SKU, alongside
// its primary SKU.ScaleCode -- for a tenant with more than one physical
// scale, each independently programmed with its own code for the same
// product. Same shape/purpose as BarcodeAlias, but for scale codes:
// DecodeWeightEmbeddedBarcode's extracted ScaleCode is looked up against
// SKU.ScaleCode first, then this table, so any of a SKU's scale codes
// resolves to the same product/pricing regardless of which physical scale
// printed the label.
type ScaleCodeAlias struct {
	ID        string
	TenantID  string
	SKUID     string
	Code      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
