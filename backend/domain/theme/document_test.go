package theme_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestDocumentKey_templatesAndSectionGroupsAreAFixedSet(t *testing.T) {
	for _, key := range theme.TemplateKeys() {
		assert.NoError(t, theme.ValidateDocumentKey(theme.KindTemplate, key), key)
	}
	assert.ErrorIs(t, theme.ValidateDocumentKey(theme.KindTemplate, "about"), theme.ErrInvalidDocumentKey)

	assert.NoError(t, theme.ValidateDocumentKey(theme.KindSectionGroup, "header"))
	assert.NoError(t, theme.ValidateDocumentKey(theme.KindSectionGroup, "footer"))
	assert.ErrorIs(t, theme.ValidateDocumentKey(theme.KindSectionGroup, "sidebar"), theme.ErrInvalidDocumentKey)
}

func TestDocumentKey_pagesAreSlugs(t *testing.T) {
	for _, ok := range []string{"about", "shipping-and-returns", "faq-2026"} {
		assert.NoError(t, theme.ValidateDocumentKey(theme.KindPage, ok), ok)
	}
	for _, bad := range []string{"", "About", "a b", "-lead", "trail-", "a--b", "../etc", strings.Repeat("a", 65)} {
		assert.ErrorIs(t, theme.ValidateDocumentKey(theme.KindPage, bad), theme.ErrInvalidDocumentKey, bad)
	}
}

func TestDocumentKind_unknownIsRejected(t *testing.T) {
	assert.ErrorIs(t, theme.ValidateDocumentKey(theme.DocumentKind("widget"), "x"), theme.ErrInvalidDocumentKind)
}

func TestNewPage_requiresTitle(t *testing.T) {
	_, err := theme.NewPage("about", "  ")
	assert.ErrorIs(t, err, theme.ErrInvalidDocumentTitle)

	doc, err := theme.NewPage("about", " About us ")
	require.NoError(t, err)
	assert.Equal(t, theme.KindPage, doc.Kind)
	assert.Equal(t, "About us", doc.Title)
	assert.Equal(t, theme.RootType, doc.Root.Type)
	assert.Empty(t, doc.Root.Children)
}

func TestDataContextsFor(t *testing.T) {
	assert.True(t, theme.ContextAvailable(theme.KindTemplate, "product", theme.ContextProduct))
	assert.False(t, theme.ContextAvailable(theme.KindTemplate, "home", theme.ContextProduct))
	assert.True(t, theme.ContextAvailable(theme.KindTemplate, "collection", theme.ContextCollection))
	assert.True(t, theme.ContextAvailable(theme.KindTemplate, "search", theme.ContextCollection))
	assert.True(t, theme.ContextAvailable(theme.KindTemplate, "order-confirmation", theme.ContextOrder))
	// cart and customer are per-shopper session data, available everywhere
	// (e.g. a cart icon in the header, "Hi, Anna" on the home page).
	assert.True(t, theme.ContextAvailable(theme.KindSectionGroup, "header", theme.ContextCart))
	assert.True(t, theme.ContextAvailable(theme.KindPage, "about", theme.ContextCustomer))
	assert.False(t, theme.ContextAvailable(theme.KindPage, "about", theme.ContextProduct))
	assert.True(t, theme.ContextAvailable(theme.KindPage, "about", theme.ContextNone))
}
