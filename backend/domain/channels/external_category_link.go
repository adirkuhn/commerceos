package channels

import "time"

// ExternalCategoryLink maps one pms.Category to its equivalent category on
// one specific channel connection -- e.g. a Woo category numeric ID.
// Deliberately connection-scoped (not provider-scoped like
// tax.ExternalTaxClassLink): Woo category IDs are per-store, so a tenant
// with two Woo connections needs independent mappings, not one shared by
// provider alone.
//
// A link is created two ways: automatically, when
// ResyncProductToChannelsUseCase (or the general product-field sync) needs
// a category for a product whose category has no link yet and finds a Woo
// category with a matching slug (see ResolveCategoryLinkUseCase); or
// manually, via the Settings UI, when no slug match exists or a merchant
// wants to repoint one.
type ExternalCategoryLink struct {
	ID           string
	TenantID     string
	ConnectionID string
	Provider     string // "woocommerce" today, kept generic for future channels
	ExternalID   string // the category's id on the channel's own system
	CategoryID   string // pms.Category.ID
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
