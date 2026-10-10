// Package locale is the set of languages CommerceOS speaks: to shoppers
// (a channel's storefront) and to merchants (the admin and theme editor).
// BCP 47 tags, matching the frontends' translation files.
package locale

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

type Locale string

const (
	English    Locale = "en"
	Portuguese Locale = "pt-BR"
	Dutch      Locale = "nl"
)

var ErrUnsupported = errs.New("LANGUAGE_UNSUPPORTED", "unsupported language")

func (l Locale) Valid() bool {
	switch l {
	case English, Portuguese, Dutch:
		return true
	}
	return false
}

// Resolve validates a requested language, defaulting "" to English.
func Resolve(requested Locale) (Locale, error) {
	if requested == "" {
		return English, nil
	}
	if !requested.Valid() {
		return "", ErrUnsupported
	}
	return requested, nil
}

// OrDefault is the language itself, or English when unset.
func (l Locale) OrDefault() Locale {
	if l == "" {
		return English
	}
	return l
}
