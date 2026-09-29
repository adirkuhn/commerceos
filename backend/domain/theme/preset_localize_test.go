package theme_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

// Every text a preset writes must be translated into every language, so a
// new theme starts in its store's language.
func TestPresetTexts_areTranslatedIntoEveryLanguage(t *testing.T) {
	for _, key := range theme.PresetKeys() {
		docs, err := theme.PresetDocuments(key, locale.English)
		require.NoError(t, err)
		for _, text := range theme.PresetTexts(docs) {
			for _, l := range []locale.Locale{locale.Portuguese, locale.Dutch} {
				assert.True(t, theme.HasPresetTranslation(l, text), "%s: %q has no %s translation", key, text, l)
			}
		}
	}
}

func TestPresetDocuments_startInTheStoreLanguage(t *testing.T) {
	docs, err := theme.PresetDocuments(theme.PresetMercado, locale.Portuguese)
	require.NoError(t, err)
	var home, header theme.Document
	for _, d := range docs {
		switch d.Key {
		case "home":
			home = d
		case "header":
			header = d
		}
	}
	title := find(home.Root, "hero")[0].Props["title"]
	assert.Equal(t, "Grande promoção: até 60% off", title)
	menu := find(header.Root, "navigation-menu")[1].Props["items"].([]any)
	assert.Equal(t, "Mercearia", menu[0].(map[string]any)["label"])

	en, _ := theme.PresetDocuments(theme.PresetMercado, locale.English)
	for _, d := range en {
		if d.Key == "home" {
			assert.Equal(t, "Big sale: up to 60% off", find(d.Root, "hero")[0].Props["title"], "English is the source")
		}
	}
}
