// Package storefront holds the internal-storefront checkout domain: a
// guest shopping cart on one of a tenant's ChannelTypeInternal sales
// channels (see domain/channels.ChannelType), and everything needed to
// turn it into an oms.SalesOrder. Distinct from domain/whatsapp's
// Conversation/CartItem -- that one is a multi-turn chat state machine
// keyed by phone number; this one is a stateless-per-request HTTP cart
// keyed by an opaque token, and (unlike WhatsApp, which has at most one
// channel per tenant) always belongs to one specific SalesChannel, since a
// tenant can run more than one internal storefront (e.g. B2C and B2B) with
// different catalogs and pricing.
package storefront

import "time"

// CartItem is one line of an in-progress cart. PriceMinorUnits/CurrencyCode
// are snapshotted at add-time (the price the shopper was shown), not
// re-read at checkout -- same reasoning as whatsapp.CartItem: a price
// change mid-session must never silently change what they agreed to pay.
type CartItem struct {
	SKUID       string
	SKUCode     string
	ProductName string
	Quantity    int64
	// PriceMinorUnits/CurrencyCode/TaxClassID are all snapshotted at
	// add-time (see the type's own doc comment) -- TaxClassID mirrors
	// pms.SKU.TaxClassID as of that moment, nil meaning "use the tenant's
	// default class", same convention as the SKU field itself.
	PriceMinorUnits int64
	CurrencyCode    string
	TaxClassID      *string
}

// Cart is one shopper's in-progress guest cart on one internal sales
// channel. Token is the bearer capability the storefront frontend holds
// (in a cookie or local storage) and presents on every call -- there is no
// customer login yet (see the roadmap's later "storefront customer
// accounts" phase), so Token, not a session, is what proves "this request
// owns this cart".
type Cart struct {
	ID             string
	TenantID       string
	SalesChannelID string
	Token          string
	Items          []CartItem
	// CheckedOutAt is nil while the cart is still open. Set once by
	// CheckoutCartUseCase the moment its SalesOrder is created --
	// CheckoutCartUseCase refuses to run again against a cart that already
	// has this set, so a retried/double-submitted checkout request can
	// never place a second order for the same cart.
	CheckedOutAt *time.Time
	// CouponCode is nil when no coupon is attached -- identity only, never
	// a stored discount amount (see SubtotalMinorUnits's own doc comment
	// on why a Cart caches nothing derivable from its state): the discount
	// is always computed live from this code plus the cart's current
	// subtotal, in application/storefront.
	CouponCode *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// IsCheckedOut reports whether this cart has already been turned into an
// order -- see CheckedOutAt's doc comment.
func (c Cart) IsCheckedOut() bool {
	return c.CheckedOutAt != nil
}

// SubtotalMinorUnits sums every line's Quantity*PriceMinorUnits. Callers
// needing tax/shipping/total add those separately -- a Cart itself never
// carries a currency-total field of its own to avoid it silently going
// stale as items change (same reasoning SalesOrder avoids caching anything
// derivable from its own line items).
func (c Cart) SubtotalMinorUnits() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.Quantity * item.PriceMinorUnits
	}
	return total
}
