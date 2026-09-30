package theme_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestValidDocumentTranslations_trimsAndDropsEmptyLanguages(t *testing.T) {
	got, err := theme.ValidDocumentTranslations(theme.KindPage, map[locale.Locale]theme.DocumentTexts{
		locale.English: {Title: "  About us ", SEO: theme.SEO{Title: "About", Description: "Who we are"}},
		locale.Dutch:   {},
	})

	require.NoError(t, err)
	assert.Equal(t, map[locale.Locale]theme.DocumentTexts{locale.English: {Title: "About us", SEO: theme.SEO{Title: "About", Description: "Who we are"}}}, got)
}

func TestValidDocumentTranslations_rejections(t *testing.T) {
	cases := map[string]struct {
		kind theme.DocumentKind
		tr   map[locale.Locale]theme.DocumentTexts
	}{
		"unsupported language":     {theme.KindPage, map[locale.Locale]theme.DocumentTexts{"de": {Title: "Über uns"}}},
		"title on a template":      {theme.KindTemplate, map[locale.Locale]theme.DocumentTexts{locale.English: {Title: "Home"}}},
		"title too long":           {theme.KindPage, map[locale.Locale]theme.DocumentTexts{locale.English: {Title: strings.Repeat("a", 121)}}},
		"seo title too long":       {theme.KindTemplate, map[locale.Locale]theme.DocumentTexts{locale.English: {SEO: theme.SEO{Title: strings.Repeat("a", 71)}}}},
		"seo description too long": {theme.KindTemplate, map[locale.Locale]theme.DocumentTexts{locale.English: {SEO: theme.SEO{Description: strings.Repeat("a", 321)}}}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := theme.ValidDocumentTranslations(c.kind, c.tr)
			assert.ErrorIs(t, err, theme.ErrInvalidDocumentTranslation)
		})
	}
}
