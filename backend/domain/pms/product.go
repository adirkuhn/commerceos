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
	Tags []string
	// SyncVariantsAsWooVariations only matters when HasVariants is true, and
	// only for channels that support it (WooCommerce today). false (the
	// default): each Variant pushes as its own separate simple Woo product.
	// true: all Variants push as ONE Woo "variable" product with one Woo
	// variation per Variant (native Woo variable-product support), letting
	// a shopper pick e.g. a specific weight from a dropdown on one listing
	// instead of seeing N separate products. Changing this on an
	// already-pushed product is a real migration, not just a field update
	// -- see application/channels.SyncProductToChannelsUseCase, which
	// deletes the old external representation and recreates it in the new
	// shape when it detects this flag no longer matches what's actually on
	// the channel.
	SyncVariantsAsWooVariations bool
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}
