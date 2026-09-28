package channels

import "errors"

// StorefrontMode is how an internal channel faces shoppers: headless means
// the merchant brings their own frontend and uses only the /storefront/v1
// REST API; theme means CommerceOS also hosts a storefront rendered from the
// channel's live theme (see docs/storefront-architecture.md). The shopper
// API works in both modes.
type StorefrontMode string

const (
	StorefrontModeHeadless StorefrontMode = "headless"
	StorefrontModeTheme    StorefrontMode = "theme"
)

var (
	ErrInvalidStorefrontMode = errors.New("invalid storefront mode")
	// ErrThemedStorefrontRequiresInternal guards against a WooCommerce/Bol/
	// WhatsApp channel claiming a hosted storefront -- their shop front
	// lives on the external platform.
	ErrThemedStorefrontRequiresInternal = errors.New("only internal sales channels can host a themed storefront")
)

func (m StorefrontMode) Valid() bool {
	return m == StorefrontModeHeadless || m == StorefrontModeTheme
}

// ResolveStorefrontMode validates a requested mode for a channel type,
// defaulting "" to headless.
func ResolveStorefrontMode(channelType ChannelType, requested StorefrontMode) (StorefrontMode, error) {
	if requested == "" {
		return StorefrontModeHeadless, nil
	}
	if !requested.Valid() {
		return "", ErrInvalidStorefrontMode
	}
	if requested == StorefrontModeTheme && channelType != ChannelTypeInternal {
		return "", ErrThemedStorefrontRequiresInternal
	}
	return requested, nil
}

// HostsThemedStorefront reports whether CommerceOS renders a storefront for
// this channel from its theme.
func (c SalesChannel) HostsThemedStorefront() bool {
	return c.Type == ChannelTypeInternal && c.StorefrontMode == StorefrontModeTheme
}
