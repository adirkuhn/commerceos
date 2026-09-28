package channels

import (
	"errors"
	"net/url"
	"strings"
)

// maxSignInURLLength keeps the emailed link a sane size.
const maxSignInURLLength = 2000

var (
	ErrInvalidSignInURL = errors.New("sign-in page must be an absolute https URL without credentials or fragment")
	// ErrSignInURLRequiresInternal: shopper accounts only exist on
	// CommerceOS' own storefront channels.
	ErrSignInURLRequiresInternal = errors.New("only internal sales channels have a shopper sign-in page")
	// ErrSignInNotConfigured: a headless channel whose merchant hasn't set
	// the page that completes sign-in.
	ErrSignInNotConfigured = errors.New("this store has no sign-in page configured")
)

// ResolveSignInURL validates the page a headless storefront completes
// shopper sign-in on. Emailed sign-in links point there (with ?token=),
// so it is configured by the merchant -- never taken from the shopper's
// request, where an attacker could aim a victim's link at their own site.
// Blank clears it (nil).
func ResolveSignInURL(channelType ChannelType, raw string) (*string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	if channelType != ChannelTypeInternal {
		return nil, ErrSignInURLRequiresInternal
	}
	if len(s) > maxSignInURLLength {
		return nil, ErrInvalidSignInURL
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || strings.Contains(s, "#") {
		return nil, ErrInvalidSignInURL
	}
	switch {
	case u.Scheme == "https":
	case u.Scheme == "http" && isLocalHost(u.Hostname()):
	default:
		return nil, ErrInvalidSignInURL
	}
	return &s, nil
}

func isLocalHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// SignInLink is the emailed sign-in link for this channel: the hosted
// store's own login page for a themed storefront, else the merchant's
// configured page. themedStorefrontBase is where CommerceOS serves hosted
// storefronts (e.g. https://shops.example.com).
func (c SalesChannel) SignInLink(themedStorefrontBase, token string) (string, error) {
	page := ""
	switch {
	case c.HostsThemedStorefront():
		page = strings.TrimRight(themedStorefrontBase, "/") + "/store/" + url.PathEscape(c.ID) + "/login"
	case c.SignInURL != nil:
		page = *c.SignInURL
	default:
		return "", ErrSignInNotConfigured
	}
	u, err := url.Parse(page)
	if err != nil {
		return "", ErrSignInNotConfigured
	}
	q := u.Query()
	q.Set("token", token)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
