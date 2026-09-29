package channels

import "errors"

// StorefrontLocale is the language a channel's storefront speaks to
// shoppers: its built-in texts (buttons, checkout labels, messages) and how
// it formats prices and dates. BCP 47 tags, matching the storefront's own
// message catalogs.
type StorefrontLocale string

const (
	StorefrontLocaleEnglish    StorefrontLocale = "en"
	StorefrontLocalePortuguese StorefrontLocale = "pt-BR"
	StorefrontLocaleDutch      StorefrontLocale = "nl"
)

var ErrInvalidStorefrontLocale = errors.New("unsupported storefront language")

func (l StorefrontLocale) Valid() bool {
	switch l {
	case StorefrontLocaleEnglish, StorefrontLocalePortuguese, StorefrontLocaleDutch:
		return true
	}
	return false
}

// ResolveStorefrontLocale validates a requested language, defaulting "" to
// English.
func ResolveStorefrontLocale(requested StorefrontLocale) (StorefrontLocale, error) {
	if requested == "" {
		return StorefrontLocaleEnglish, nil
	}
	if !requested.Valid() {
		return "", ErrInvalidStorefrontLocale
	}
	return requested, nil
}
