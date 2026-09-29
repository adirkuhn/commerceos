package channels

import (
	"errors"
	"fmt"
	"slices"

	"github.com/adirkuhn/commerceos/domain/locale"
)

// StorefrontLocale is the language a channel's storefront speaks to
// shoppers: its built-in texts (buttons, checkout labels, messages) and how
// it formats prices and dates. See package locale for the languages.
type StorefrontLocale = locale.Locale

const (
	StorefrontLocaleEnglish    = locale.English
	StorefrontLocalePortuguese = locale.Portuguese
	StorefrontLocaleDutch      = locale.Dutch
)

var ErrInvalidStorefrontLocale = fmt.Errorf("storefront %w", locale.ErrUnsupported)

// ResolveStorefrontLocale validates a requested language, defaulting "" to
// English.
func ResolveStorefrontLocale(requested StorefrontLocale) (StorefrontLocale, error) {
	l, err := locale.Resolve(requested)
	if errors.Is(err, locale.ErrUnsupported) {
		return "", ErrInvalidStorefrontLocale
	}
	return l, err
}

// ResolveStorefrontLocales is every language a storefront offers: its
// default first, then the others in the order given, each once. Shoppers
// can switch between them; the default is what a plain store URL speaks.
func ResolveStorefrontLocales(defaultLocale StorefrontLocale, others []StorefrontLocale) ([]StorefrontLocale, error) {
	if err := ValidateStorefrontLocales(others); err != nil {
		return nil, err
	}
	return offered(defaultLocale, others), nil
}

// ValidateStorefrontLocales checks every language is one CommerceOS speaks.
func ValidateStorefrontLocales(locales []StorefrontLocale) error {
	for _, l := range locales {
		if !l.Valid() {
			return ErrInvalidStorefrontLocale
		}
	}
	return nil
}

func offered(defaultLocale StorefrontLocale, others []StorefrontLocale) []StorefrontLocale {
	all := []StorefrontLocale{defaultLocale}
	for _, l := range others {
		if !slices.Contains(all, l) {
			all = append(all, l)
		}
	}
	return all
}

// OfferedStorefrontLocales is ResolveStorefrontLocales for a saved channel
// (whose languages were validated when saved).
func (c SalesChannel) OfferedStorefrontLocales() []StorefrontLocale {
	return offered(c.StorefrontLocale.OrDefault(), c.StorefrontLocales)
}
