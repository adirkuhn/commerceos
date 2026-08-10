// Package channels holds the cross-cutting concept of a CommerceOS product
// being listed on an external sales channel (WooCommerce today, others
// later). Deliberately its own bounded context rather than living inside
// pms or integrations — it references both by ID only, never by importing
// their domain types, so pms and integrations stay decoupled from each
// other and from any specific channel provider.
package channels

import "time"

type SyncStatus string

const (
	SyncStatusPending SyncStatus = "pending"
	SyncStatusSynced  SyncStatus = "synced"
	SyncStatusError   SyncStatus = "error"
)

type ProductChannelListing struct {
	ID           string
	TenantID     string
	ProductID    string // pms.Product.ID
	ConnectionID string // integrations.Connection.ID
	Provider     string // denormalized from the connection, e.g. "woocommerce"
	ExternalID   string // the product's id on the channel's own system
	// VariantID is nil for the "product-level" row: either a legacy
	// single-SKU simple product, or (when
	// pms.Product.SyncVariantsAsWooVariations is true) the PARENT variable
	// product. Set to a specific pms.Variant.ID for a per-variant row --
	// either its own standalone simple Woo product (separate-products
	// mode) or a variation nested under the sibling product-level row's
	// ExternalID as parent (variations mode, see IsVariation). A product's
	// listings for one connection are always either: one row (VariantID
	// nil, single-SKU product) — or N rows all with VariantID set and
	// IsVariation=false (separate-products mode) — or one row with
	// VariantID nil (the parent) plus N rows with VariantID set and
	// IsVariation=true (variations mode). Never a mix of the last two;
	// SyncProductToChannelsUseCase enforces that by fully tearing down and
	// recreating the listing set whenever the product's
	// SyncVariantsAsWooVariations flag no longer matches what's actually
	// on the channel.
	VariantID *string
	// IsVariation is only meaningful when VariantID is set: true means
	// ExternalID is a Woo variation ID nested under the sibling
	// product-level row's ExternalID (the parent variable product) --
	// false means ExternalID is its own standalone simple Woo product ID.
	IsVariation   bool
	SyncStatus    SyncStatus
	LastSyncedAt  *time.Time
	LastSyncError string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
