package pms

import (
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"strings"
	"unicode/utf8"

	"github.com/adirkuhn/commerceos/domain/locale"
)

// ErrInvalidProductTranslation guards SetProductTranslationsUseCase's input.
var ErrInvalidProductTranslation = errs.New("PRODUCT_TRANSLATION_INVALID", "invalid product translation")

const (
	maxTranslatedNameLength        = 255
	maxTranslatedDescriptionLength = 20000
)

// ProductTexts is a product's shopper-facing texts in one language. The
// product's own Name/Description are the company language; a blank
// translated text shows that one.
type ProductTexts struct {
	Name        string
	Description string
}

// ValidProductTranslations trims the texts, drops languages with none, and
// checks the languages and lengths.
func ValidProductTranslations(tr map[locale.Locale]ProductTexts) (map[locale.Locale]ProductTexts, error) {
	out := map[locale.Locale]ProductTexts{}
	for l, t := range tr {
		if !l.Valid() {
			return nil, fmt.Errorf("%w: language %q is not supported", ErrInvalidProductTranslation, l)
		}
		t = ProductTexts{Name: strings.TrimSpace(t.Name), Description: strings.TrimSpace(t.Description)}
		if utf8.RuneCountInString(t.Name) > maxTranslatedNameLength {
			return nil, fmt.Errorf("%w: name longer than %d characters", ErrInvalidProductTranslation, maxTranslatedNameLength)
		}
		if utf8.RuneCountInString(t.Description) > maxTranslatedDescriptionLength {
			return nil, fmt.Errorf("%w: description longer than %d characters", ErrInvalidProductTranslation, maxTranslatedDescriptionLength)
		}
		if t != (ProductTexts{}) {
			out[l] = t
		}
	}
	return out, nil
}

// In is the product as shoppers reading l see it: its translated name and
// description over the default ones.
func (p Product) In(l locale.Locale) Product {
	t, ok := p.Translations[l]
	if !ok {
		return p
	}
	if t.Name != "" {
		p.Name = t.Name
	}
	if t.Description != "" {
		p.Description = t.Description
	}
	return p
}
