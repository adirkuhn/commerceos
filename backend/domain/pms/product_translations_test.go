package pms_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/pms"
)

func TestValidProductTranslations_trimsAndDropsEmptyLanguages(t *testing.T) {
	got, err := pms.ValidProductTranslations(map[locale.Locale]pms.ProductTexts{
		locale.English: {Name: " Cheese bread ", Description: "<p>Warm</p>"},
		locale.Dutch:   {Name: "  "},
	})
	require.NoError(t, err)
	assert.Equal(t, map[locale.Locale]pms.ProductTexts{locale.English: {Name: "Cheese bread", Description: "<p>Warm</p>"}}, got)
}

func TestValidProductTranslations_rejections(t *testing.T) {
	for name, tr := range map[string]map[locale.Locale]pms.ProductTexts{
		"unsupported language": {"de": {Name: "Käsebrot"}},
		"name too long":        {locale.English: {Name: strings.Repeat("a", 256)}},
		"description too long": {locale.English: {Description: strings.Repeat("a", 20001)}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := pms.ValidProductTranslations(tr)
			assert.ErrorIs(t, err, pms.ErrInvalidProductTranslation)
		})
	}
}

func TestProduct_In(t *testing.T) {
	p := pms.Product{Name: "Pão de queijo", Description: "Quentinho", Slug: "pao-de-queijo",
		Translations: map[locale.Locale]pms.ProductTexts{locale.English: {Name: "Cheese bread"}}}

	en := p.In(locale.English)
	assert.Equal(t, "Cheese bread", en.Name)
	assert.Equal(t, "Quentinho", en.Description, "untranslated description falls back")
	assert.Equal(t, "pao-de-queijo", en.Slug)
	assert.Equal(t, "Pão de queijo", p.In(locale.Dutch).Name)
	assert.Equal(t, "Pão de queijo", p.In("").Name)
}
