package collections_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/collections"
	"github.com/adirkuhn/commerceos/domain/locale"
)

func TestNew_keepsCleanTranslations(t *testing.T) {
	d := valid()
	d.Translations = map[locale.Locale]collections.Texts{
		locale.English: {Name: " Summer ", SEOTitle: "Summer deals"},
		locale.Dutch:   {},
	}
	c, err := collections.New("t1", d, time.Now())
	require.NoError(t, err)
	assert.Equal(t, map[locale.Locale]collections.Texts{locale.English: {Name: "Summer", SEOTitle: "Summer deals"}}, c.Translations)
}

func TestNew_rejectsBadTranslations(t *testing.T) {
	for name, tr := range map[string]map[locale.Locale]collections.Texts{
		"unsupported language": {"de": {Name: "Sommer"}},
		"name too long":        {locale.English: {Name: strings.Repeat("a", 121)}},
		"seo title too long":   {locale.English: {SEOTitle: strings.Repeat("a", 71)}},
		"description too long": {locale.English: {Description: strings.Repeat("a", 5001)}},
	} {
		t.Run(name, func(t *testing.T) {
			d := valid()
			d.Translations = tr
			_, err := collections.New("t1", d, time.Now())
			assert.ErrorIs(t, err, collections.ErrInvalidCollection)
		})
	}
}

func TestCollection_In(t *testing.T) {
	c := collections.Collection{Draft: collections.Draft{Name: "Verão", Description: "Ofertas", SEOTitle: "Verão", Slug: "verao",
		Translations: map[locale.Locale]collections.Texts{locale.English: {Name: "Summer"}}}}
	en := c.In(locale.English)
	assert.Equal(t, "Summer", en.Name)
	assert.Equal(t, "Ofertas", en.Description)
	assert.Equal(t, "Verão", c.In(locale.Dutch).Name)
}
