package pms

import pmsDomain "github.com/adirkuhn/commerceos/domain/pms"

// ProductDetail aggregates a product with its variants and each variant's
// SKUs — the shape the product editor needs in one round trip.
type ProductDetail struct {
	Product  pmsDomain.Product
	Variants []VariantDetail
	// LinkedSKUs names the SKUs this product's SKUs are a case or a
	// portion of (SKU.CaseOfSKUID / PortionedFromSKUID), keyed by SKU id.
	// Only GetProductUseCase fills it.
	LinkedSKUs map[string]LinkedSKU
}

// LinkedSKU identifies another product's SKU for display.
type LinkedSKU struct {
	ID          string
	ProductName string
	SKUCode     string
	Barcode     string
}

type VariantDetail struct {
	Variant pmsDomain.Variant
	SKUs    []pmsDomain.SKU
}
