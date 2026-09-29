package channels

import (
	"errors"
	"fmt"

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
