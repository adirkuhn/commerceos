package themecatalog_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
	"github.com/adirkuhn/commerceos/infrastructure/themecatalog"
)

func load(t *testing.T) *theme.Catalog {
	t.Helper()
	spec, err := gentheme.GetSpec()
	require.NoError(t, err)
	c, err := themecatalog.FromSpec(spec)
	require.NoError(t, err)
	return c
}

func TestFromSpec_readsPlacementRules(t *testing.T) {
	c := load(t)

	section, ok := c.Rule("section")
	require.True(t, ok)
	assert.True(t, section.Container)
	assert.Equal(t, []string{theme.RootType}, section.AllowedParents)

	heading, ok := c.Rule("heading")
	require.True(t, ok)
	assert.False(t, heading.Container)

	_, ok = c.Rule("marquee")
	assert.False(t, ok)
}

func TestFromSpec_validatesPropsAgainstTheSchema(t *testing.T) {
	heading, _ := load(t).Rule("heading")

	assert.NoError(t, heading.ValidateProps(map[string]any{"text": "Hi", "level": float64(2)}))
	assert.Error(t, heading.ValidateProps(map[string]any{"text": "Hi"}), "level required")
	assert.Error(t, heading.ValidateProps(map[string]any{"text": "Hi", "level": float64(9)}))
	assert.Error(t, heading.ValidateProps(map[string]any{"text": "Hi", "level": float64(1), "onclick": "x"}), "no extra props")
	assert.Error(t, heading.ValidateProps(nil))
}

func TestFromSpec_richTextRejectsScriptLinks(t *testing.T) {
	text, _ := load(t).Rule("text")
	para := func(href string) map[string]any {
		return map[string]any{"content": []any{map[string]any{
			"type": "paragraph", "children": []any{map[string]any{"text": "go", "href": href}},
		}}}
	}

	assert.NoError(t, text.ValidateProps(para("https://example.com")))
	assert.NoError(t, text.ValidateProps(para("/pages/about")))
	assert.Error(t, text.ValidateProps(para("javascript:alert(1)")))
}

func TestFromSpec_containerWithNoPropsAcceptsEmpty(t *testing.T) {
	container, _ := load(t).Rule("container")
	assert.NoError(t, container.ValidateProps(nil))
}

// The domain's style vocabulary and the contract's StyleValues schema must
// list the same properties -- one is enforced on write, the other is what
// clients are told.
func TestStyleVocabularyMatchesContract(t *testing.T) {
	spec, err := gentheme.GetSpec()
	require.NoError(t, err)
	var contract []string
	for name := range spec.Components.Schemas["StyleValues"].Value.Properties {
		contract = append(contract, name)
	}
	domain := theme.StyleProperties()
	sort.Strings(contract)
	sort.Strings(domain)
	assert.Equal(t, domain, contract)
}

func TestFromSpec_newComponentsGuardLinksAndImages(t *testing.T) {
	c := load(t)
	button, _ := c.Rule("button")
	image, _ := c.Rule("image")
	grid, _ := c.Rule("product-grid")
	column, _ := c.Rule("column")
	columns, _ := c.Rule("columns")

	assert.NoError(t, button.ValidateProps(map[string]any{"label": "Shop", "link": "/collections/summer"}))
	assert.Error(t, button.ValidateProps(map[string]any{"label": "x", "link": "javascript:alert(1)"}))
	assert.Error(t, button.ValidateProps(map[string]any{"label": "", "link": "/"}))

	assert.NoError(t, image.ValidateProps(map[string]any{"image": "asset:0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e"}))
	assert.NoError(t, image.ValidateProps(map[string]any{"image": ""}))
	assert.Error(t, image.ValidateProps(map[string]any{"image": "https://evil.example/x.png"}), "images come from the media library only")

	assert.NoError(t, grid.ValidateProps(map[string]any{"source": "category", "category_id": "c1", "limit": float64(8)}))
	assert.Error(t, grid.ValidateProps(map[string]any{"source": "category", "limit": float64(100)}))

	assert.Equal(t, []string{"columns"}, column.AllowedParents)
	assert.Equal(t, []string{"column"}, columns.AllowedChildren)
}

// A new theme must be publishable as created: every preset document has
// to satisfy the contract (incl. required components like Add to cart).
func TestEveryPresetValidatesAgainstTheContract(t *testing.T) {
	c := load(t)
	for _, key := range theme.PresetKeys() {
		for _, lang := range []locale.Locale{locale.English, locale.Portuguese, locale.Dutch} {
			docs, err := theme.PresetDocuments(key, lang)
			require.NoError(t, err, key)
			for _, d := range docs {
				assert.NoError(t, theme.ValidateTree(d, c), "%s (%s): %s %s", key, lang, d.Kind, d.Key)
			}
		}
	}
}
