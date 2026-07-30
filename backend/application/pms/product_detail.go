package pms

import pmsDomain "github.com/adirkuhn/commerceos/domain/pms"

// ProductDetail aggregates a product with its variants and each variant's
// SKUs — the shape the product editor needs in one round trip.
type ProductDetail struct {
	Product  pmsDomain.Product
	Variants []VariantDetail
}

type VariantDetail struct {
	Variant pmsDomain.Variant
	SKUs    []pmsDomain.SKU
}
