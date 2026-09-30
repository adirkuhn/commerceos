// Package theme is the storefront theme system: themes with immutable,
// published versions and one mutable draft each, the documents (templates,
// pages, header/footer) inside a version, and the rules a document's node
// tree must follow. See docs/storefront-architecture.md.
package theme

import (
	"strings"
	"time"
	"unicode/utf8"
)

const maxNameLength = 120

// Presets decide which documents and settings a new theme's first draft
// starts with (see preset.go).
const (
	// PresetDefault is a general-purpose store.
	PresetDefault = "default"
	// PresetMercado is a grocery / market layout: category menu, rows of
	// products with discount badges and add to cart.
	PresetMercado = "mercado"
	// PresetFresh is an online supermarket: hero, promo cards, category
	// tiles, product rows and service promises.
	PresetFresh = "fresh"
)

// Theme is one design of a sales channel's storefront. A channel can keep
// several (a theme library); at most one is live. PublishedVersionID is the
// version shoppers see while IsLive -- and the version a non-live theme
// would go live with.
type Theme struct {
	ID                 string
	TenantID           string
	SalesChannelID     string
	Name               string
	PresetKey          string
	PublishedVersionID *string
	IsLive             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func NewTheme(tenantID, salesChannelID, name, presetKey string, now time.Time) (Theme, error) {
	name, err := validName(name)
	if err != nil {
		return Theme{}, err
	}
	if _, ok := presetRegistry[presetKey]; !ok {
		return Theme{}, ErrUnknownPreset
	}
	return Theme{
		TenantID: tenantID, SalesChannelID: salesChannelID, Name: name, PresetKey: presetKey,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (t *Theme) Rename(name string, now time.Time) error {
	name, err := validName(name)
	if err != nil {
		return err
	}
	t.Name = name
	t.UpdatedAt = now
	return nil
}

func (t Theme) CanDelete() error {
	if t.IsLive {
		return ErrThemeIsLive
	}
	return nil
}

func validName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNameLength {
		return "", ErrInvalidThemeName
	}
	return name, nil
}
