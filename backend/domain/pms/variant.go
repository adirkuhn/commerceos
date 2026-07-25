package pms

import "time"

type Variant struct {
	ID         string
	TenantID   string
	ProductID  string
	Name       string
	Position   int
	Attributes []VariantAttribute
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type VariantAttribute struct {
	AttributeDefinitionID string
	Value                 string
}
