package notifications_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/notifications"
)

var languages = []locale.Locale{locale.English, locale.Portuguese, locale.Dutch}

func TestTypes_everyTypeIsTranslatedInEveryLanguage(t *testing.T) {
	sample := map[string]any{"error": "x", "reason": "x", "product": "x", "count": 3, "window_days": 30,
		"products": 1, "already_linked": 0, "skipped": 0, "customers": 2, "orders": 4, "store": "x", "hours": 30}
	for _, typ := range notifications.Types {
		for _, lang := range languages {
			text, ok := typ.Render(lang, sample)
			require.True(t, ok, "%s in %s", typ, lang)
			assert.NotEmpty(t, text.Title, "%s in %s", typ, lang)
			assert.NotContains(t, text.Title+text.Message, "{{", "%s in %s leaves a token", typ, lang)
		}
	}
}

func TestRender_interpolatesParams(t *testing.T) {
	text, ok := notifications.TypeBolNotForSale.Render(locale.Portuguese, map[string]any{"product": "Guaraná", "reason": "EAN unknown"})
	require.True(t, ok)
	assert.Equal(t, `Bol: "Guaraná" não está à venda`, text.Title)
	assert.Equal(t, "EAN unknown", text.Message)
}

func TestRender_pluralizesOnCount(t *testing.T) {
	one, _ := notifications.TypeStockCloseToExpiry.Render(locale.English, map[string]any{"count": 1, "window_days": 7})
	many, _ := notifications.TypeStockCloseToExpiry.Render(locale.English, map[string]any{"count": float64(4), "window_days": 30})

	assert.Equal(t, "1 product expires within 7 days", one.Message)
	assert.Equal(t, "4 products expire within 30 days", many.Message)
}

func TestRender_appendsOptionalPartsOnlyWhenTheirParamIsSet(t *testing.T) {
	base := map[string]any{"products": 5, "already_linked": 2, "skipped": 1}
	without, _ := notifications.TypeImportCompleted.Render(locale.English, base)
	assert.Equal(t, "Imported 5 products (2 already linked, 1 skipped)", without.Message)

	base["orders"] = 3
	with, _ := notifications.TypeImportCompleted.Render(locale.English, base)
	assert.Equal(t, "Imported 5 products (2 already linked, 1 skipped), 3 orders", with.Message)
}

func TestRender_unknownTypeIsNotOK(t *testing.T) {
	_, ok := notifications.Type("something.new").Render(locale.English, nil)
	assert.False(t, ok)
}

func TestRender_unsetLanguageIsEnglish(t *testing.T) {
	en, _ := notifications.TypeImportStarted.Render(locale.English, nil)
	unset, _ := notifications.TypeImportStarted.Render("", nil)
	assert.Equal(t, en, unset)
}

func TestEmailTexts_inEveryLanguage(t *testing.T) {
	for _, lang := range languages {
		e := notifications.EmailTextsFor(lang)
		assert.True(t, strings.HasPrefix(e.SubjectPrefix, "CommerceOS"), lang)
		assert.NotEmpty(t, e.OpenLink, lang)
		assert.NotEmpty(t, e.Footer, lang)
	}
	assert.NotEqual(t, notifications.EmailTextsFor(locale.English).Footer, notifications.EmailTextsFor(locale.Dutch).Footer)
}
