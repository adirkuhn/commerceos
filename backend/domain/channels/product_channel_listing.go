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
	// SyncStatusMissing: the channel no longer has the product this
	// listing points at (e.g. deleted in Woo). Nothing is pushed to it;
	// application/channels.HandleMissingListingUseCase either creates the
	// product again (still enrolled and active) or leaves it marked.
	SyncStatusMissing SyncStatus = "missing"
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
	// NotForSaleNote is Bol's own reason(s) the offer isn't for sale, from
	// GET /offers/{id}/not-for-sale-reasons -- checked after every push.
	// Empty means either "for sale" or "not a Bol listing / never
	// checked." Distinct from LastSyncError: the push itself can succeed
	// (no LastSyncError) while Bol still won't list the offer for sale
	// (this set) -- e.g. a still-missing mandatory content attribute Bol
	// only reports here, not as a push failure. Bol-specific today; empty
	// for every other provider.
	NotForSaleNote string
	// GatedAt is set the moment the product's status left Active while this
	// listing existed -- at that instant its current fields (and, for Woo,
	// the new draft/archived status) were pushed one last time, and every
	// further field or stock sync is skipped entirely (no channel API call
	// at all) until the product returns to Active, which clears this and
	// triggers one full catch-up push. nil means normal: sync freely. See
	// application/channels.SyncProductToChannelsUseCase.
	GatedAt   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsGated reports whether this listing is currently blocked from any
// further channel sync -- see GatedAt's doc comment.
func (l ProductChannelListing) IsGated() bool {
	return l.GatedAt != nil
}
