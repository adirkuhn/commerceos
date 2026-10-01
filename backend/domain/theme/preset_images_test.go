package theme_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestBindPresetImages_pointsPlaceholdersAtLibraryAssets(t *testing.T) {
	docs := []theme.Document{{Kind: theme.KindTemplate, Key: "home", Root: theme.Node{ID: "root", Type: theme.RootType, Children: []theme.Node{
		{ID: "hero", Type: "hero", Props: map[string]any{"image": "preset-image:hero", "title": "Hi"}},
		{ID: "card", Type: "column", Style: theme.NodeStyle{Base: map[string]string{"background_image": "preset-image:banner", "padding_x": "lg"}}},
		{ID: "gone", Type: "image", Props: map[string]any{"image": "preset-image:missing"}},
	}}}}
	settings := theme.DefaultSettings()

	theme.BindPresetImages(docs, &settings, "logo", map[string]string{"hero": "11111111-1111-4111-8111-111111111111", "banner": "22222222-2222-4222-8222-222222222222", "logo": "33333333-3333-4333-8333-333333333333"})

	kids := docs[0].Root.Children
	assert.Equal(t, "asset:11111111-1111-4111-8111-111111111111", kids[0].Props["image"])
	assert.Equal(t, "Hi", kids[0].Props["title"])
	assert.Equal(t, "asset:22222222-2222-4222-8222-222222222222", kids[1].Style.Base["background_image"])
	assert.Equal(t, "lg", kids[1].Style.Base["padding_x"])
	assert.Equal(t, "", kids[2].Props["image"], "an image that couldn't be added leaves the block empty")
	require.NotNil(t, settings.Brand.LogoAssetID)
	assert.Equal(t, "33333333-3333-4333-8333-333333333333", *settings.Brand.LogoAssetID)
}

func TestBindPresetImages_withoutImagesLeavesNoPlaceholders(t *testing.T) {
	docs, err := theme.PresetDocuments(theme.PresetFresh, locale.English)
	require.NoError(t, err)
	settings, _ := theme.PresetSettings(theme.PresetFresh)

	theme.BindPresetImages(docs, &settings, "", nil)

	for _, d := range docs {
		assert.NotContains(t, fmt.Sprintf("%v", d.Root), "preset-image:", d.Key)
	}
	assert.Nil(t, settings.Brand.LogoAssetID)
}

func TestFreshPreset_usesTheTemplatesImages(t *testing.T) {
	images := theme.PresetImages(theme.PresetFresh)
	names := map[string]bool{}
	for _, img := range images {
		names[img.Name] = true
		assert.NotEmpty(t, img.Filename)
	}
	for _, want := range []string{"hero", "offer-produce", "offer-bakery", "promo-coffee", "logo"} {
		assert.True(t, names[want], want)
	}
	assert.Equal(t, "logo", theme.PresetLogo(theme.PresetFresh))
	assert.Empty(t, theme.PresetImages(theme.PresetDefault))
}
