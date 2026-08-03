package pms

import "time"

type Variant struct {
	ID         string
	TenantID   string
	ProductID  string
	Name       string
	Position   int
	Attributes []VariantAttribute
	// FeaturedMediaID points at a Media row belonging to this variant's
	// product — which of the product's images this variant displays. Not
	// ownership (see Media.ProductID/VariantID for that); just a selection.
	FeaturedMediaID *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type VariantAttribute struct {
	AttributeDefinitionID string
	Value                 string
}
