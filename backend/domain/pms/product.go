package pms

import (
	"time"

	"github.com/adirkuhn/commerceos/domain/locale"
)

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
	// Translations are the name/description in the store's other
	// languages (see ProductTexts); set only by SetProductTranslationsUseCase.
	Translations map[locale.Locale]ProductTexts
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsSyncableToChannels is false for Draft/Archived -- a channel push (create
// or update, any provider) must never reach an external store unless the
// product is Active. See application/channels.SyncProductToChannelsUseCase,
// which also gates a listing that was already pushed while Active and then
// left it, blocking every further sync until the product returns here.
func (p Product) IsSyncableToChannels() bool {
	return p.Status == ProductStatusActive
}
