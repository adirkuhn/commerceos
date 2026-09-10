package theme

import "time"

// EditSession is a short-lived, admin-minted bearer credential that lets
// storefront-frontend's embedded visual (Puck) editor read/write one
// internal channel's theme pages -- without ever holding the merchant's
// own JWT. Distinct from, and never interchangeable with, a storefront
// API key (apikeys.ResolvedAPIKey): that grants only public/anonymous-
// shopper privilege (see application/apikeys.ResolveAPIKeyUseCase's own
// doc comment on the Stripe-publishable-key trust model it deliberately
// stays at), while this grants edit access to one specific channel's
// theme and must never be reachable by an anonymous shopper. Minted by
// application/theme.CreateEditSessionUseCase (admin-JWT-gated), consumed
// by ResolveEditSessionUseCase (called from storefront-frontend's Puck
// editor, no JWT involved).
type EditSession struct {
	ID             string
	TenantID       string
	SalesChannelID string
	Token          string
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

// IsExpired mirrors storefront.CustomerSession.IsExpired's convention --
// no "used" flag: unlike a magic-link token, one edit session is meant to
// authorize many reads/writes across an entire editing visit, not a
// single redemption.
func (s EditSession) IsExpired(now time.Time) bool {
	return !now.Before(s.ExpiresAt)
}
