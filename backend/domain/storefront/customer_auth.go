package storefront

import "time"

// MagicLinkToken is a single-use, short-lived credential emailed to a
// shopper who requested to sign in -- no password ever exists for a
// storefront customer (see RequestMagicLinkUseCase/VerifyMagicLinkUseCase).
// Bound to the channel it was requested from, same channel-isolation rule
// LookupOrderUseCase already enforces for guest lookup: signing in on a
// B2C channel must never also authenticate a B2B one.
type MagicLinkToken struct {
	ID             string
	TenantID       string
	SalesChannelID string
	CustomerID     string
	Token          string
	ExpiresAt      time.Time
	// UsedAt nil means unredeemed. Set exactly once, by
	// VerifyMagicLinkUseCase, to prevent a captured/forwarded email link
	// being replayed after the real recipient already signed in with it.
	UsedAt    *time.Time
	CreatedAt time.Time
}

// IsRedeemable reports whether this token can still be exchanged for a
// session -- unused and not yet expired.
func (t MagicLinkToken) IsRedeemable(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}

// CustomerSession is a signed-in shopper's bearer credential -- same
// opaque-token-in-path identity model as Cart/Wishlist (see Cart's own
// doc comment), scoped to the channel it was minted on.
type CustomerSession struct {
	ID             string
	TenantID       string
	SalesChannelID string
	CustomerID     string
	Token          string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

func (s CustomerSession) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}
