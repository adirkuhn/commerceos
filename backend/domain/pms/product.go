package pms

import "time"

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusArchived ProductStatus = "archived"
)

type Product struct {
	ID          string
	TenantID    string
	CategoryID  *string
	Name        string
	Slug        string
	Description string
	Status      ProductStatus
	HasVariants bool
	// Tags is free-form labels a tenant assigns to a product -- used by
	// SalesChannel's "by tag" enrollment rule (see domain/channels) to
	// decide which products auto-sync to a channel.
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}
