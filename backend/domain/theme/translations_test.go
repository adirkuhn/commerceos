package theme_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func translatableCatalog(t *testing.T) *theme.Catalog {
	t.Helper()
	c, err := theme.NewCatalog([]theme.ComponentRule{{
		Type: "heading", Translatable: []string{"text"},
		ValidateProps: func(p map[string]any) error {
			if s, ok := p["text"].(string); !ok || len(s) > 10 {
				return fmt.Errorf("text must be a string of at most 10 characters")
			}
			return nil
		},
	}})
	require.NoError(t, err)
	return c
}

func translated(tr map[locale.Locale]map[string]any) theme.Document {
	return doc(theme.KindPage, "about", theme.Node{ID: "h", Type: "heading", Props: map[string]any{"text": "Olá", "level": 1}, Translations: tr})
}

func TestValidateTree_acceptsTranslationsOfTranslatableProps(t *testing.T) {
	d := translated(map[locale.Locale]map[string]any{locale.English: {"text": "Hello"}, locale.Dutch: {"text": "Hallo"}})
	assert.NoError(t, theme.ValidateTree(d, translatableCatalog(t)))
}

func TestValidateTree_rejectsBadTranslations(t *testing.T) {
	cases := map[string]map[locale.Locale]map[string]any{
		"unsupported language":     {"de": {"text": "Hallo"}},
		"prop not translatable":    {locale.English: {"level": 2}},
		"invalid translated value": {locale.English: {"text": "far too long for a heading"}},
	}
	for name, tr := range cases {
		t.Run(name, func(t *testing.T) {
			err := theme.ValidateTree(translated(tr), translatableCatalog(t))
			var docErr *theme.DocumentError
			require.ErrorAs(t, err, &docErr)
			assert.Equal(t, "h", docErr.NodeID)
		})
	}
}

func TestNode_PropsIn(t *testing.T) {
	n := theme.Node{Props: map[string]any{"text": "Olá", "level": 1}, Translations: map[locale.Locale]map[string]any{locale.English: {"text": "Hello"}}}

	assert.Equal(t, map[string]any{"text": "Hello", "level": 1}, n.PropsIn(locale.English))
	assert.Equal(t, map[string]any{"text": "Olá", "level": 1}, n.PropsIn(locale.Dutch), "untranslated falls back to the default language")
}
