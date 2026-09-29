package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestPresets_haveValidSettingsAndAPageForEveryTemplate(t *testing.T) {
	require.ElementsMatch(t, []string{theme.PresetDefault, theme.PresetMercado}, theme.PresetKeys())
	for _, key := range theme.PresetKeys() {
		s, err := theme.PresetSettings(key)
		require.NoError(t, err, key)
		assert.NoError(t, s.Validate(), key)
		docs, err := theme.PresetDocuments(key, locale.English)
		require.NoError(t, err, key)
		keys := map[string]bool{}
		for _, d := range docs {
			keys[string(d.Kind)+"/"+d.Key] = true
		}
		for _, k := range []string{"template/home", "template/product", "template/collection", "template/cart", "template/checkout", "section_group/header", "section_group/footer"} {
			assert.True(t, keys[k], "%s: %s", key, k)
		}
	}
	_, err := theme.PresetSettings("nope")
	assert.ErrorIs(t, err, theme.ErrUnknownPreset)
}

// Mercado is a grocery layout: a category menu, rows of products on the
// home page with prices, discount badges and add to cart.
func TestMercadoPreset_isAGroceryStorefront(t *testing.T) {
	docs, err := theme.PresetDocuments(theme.PresetMercado, locale.English)
	require.NoError(t, err)
	byKey := map[string]theme.Document{}
	for _, d := range docs {
		byKey[d.Key] = d
	}
	home := byKey["home"].Root
	grids := find(home, "product-grid")
	require.GreaterOrEqual(t, len(grids), 4, "several product rows")
	for _, g := range grids {
		assert.Equal(t, true, g.Props["show_add_to_cart"])
		assert.Equal(t, true, g.Props["show_discount"])
	}
	header := byKey["header"].Root
	assert.NotEmpty(t, find(header, "search-box"))
	assert.NotEmpty(t, find(header, "cart-link"))
	assert.GreaterOrEqual(t, len(find(header, "navigation-menu")), 1)
	assert.NotEmpty(t, find(byKey["footer"].Root, "social-links"))
	s, _ := theme.PresetSettings(theme.PresetMercado)
	assert.NotEqual(t, theme.DefaultSettings().Colors, s.Colors)
}

func find(n theme.Node, typ string) []theme.Node {
	var out []theme.Node
	if n.Type == typ {
		out = append(out, n)
	}
	for _, c := range n.Children {
		out = append(out, find(c, typ)...)
	}
	return out
}

// Every preset header carries a language switcher; it shows only once the
// store offers more than one language.
func TestPresets_headerHasALanguageSwitcher(t *testing.T) {
	for _, key := range theme.PresetKeys() {
		docs, err := theme.PresetDocuments(key, locale.English)
		require.NoError(t, err)
		for _, d := range docs {
			if d.Kind == theme.KindSectionGroup && d.Key == "header" {
				assert.NotEmpty(t, find(d.Root, "language-switcher"), key)
			}
		}
	}
}
