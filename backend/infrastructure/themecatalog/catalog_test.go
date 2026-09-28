package themecatalog_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gentheme "github.com/adirkuhn/commerceos/api/theme"
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
