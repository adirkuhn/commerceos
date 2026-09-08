package storefront

import "time"

// Wishlist is one shopper's saved-for-later product list on one internal
// sales channel -- same token-addressed, no-login-required identity model
// as Cart (see Cart's own doc comment), but holds only ProductIDs, not
// snapshotted lines: unlike a cart, a wishlist makes no promise about
// price, so it's always resolved against the catalog live at display time
// rather than carrying stale data forward.
type Wishlist struct {
	ID             string
	TenantID       string
	SalesChannelID string
	Token          string
	ProductIDs     []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Contains reports whether productID is already saved -- used to toggle
// add/remove from one request rather than requiring the caller to know
// which action to take.
func (w Wishlist) Contains(productID string) bool {
	for _, id := range w.ProductIDs {
		if id == productID {
			return true
		}
	}
	return false
}
