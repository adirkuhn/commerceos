package theme_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/theme"
)

func TestNewAsset_acceptsImagesOnly(t *testing.T) {
	a, err := theme.NewAsset("tenant-1", "https://cdn/x.png", "Logo.PNG", "image/png", 1234, 64, 32, " Our logo ", "user-1", now)
	require.NoError(t, err)
	assert.Equal(t, "Our logo", a.Alt)
	assert.Equal(t, 64, a.Width)

	for _, ct := range []string{"image/svg+xml", "text/html", "application/pdf", ""} {
		_, err := theme.NewAsset("tenant-1", "u", "f", ct, 10, 0, 0, "", "u", now)
		assert.ErrorIs(t, err, theme.ErrUnsupportedAssetType, ct)
	}
}

func TestNewAsset_limits(t *testing.T) {
	_, err := theme.NewAsset("tenant-1", "u", "f", "image/png", theme.MaxAssetBytes+1, 0, 0, "", "u", now)
	assert.ErrorIs(t, err, theme.ErrAssetTooLarge)

	_, err = theme.NewAsset("tenant-1", "u", "f", "image/png", 10, 0, 0, strings.Repeat("a", 251), "u", now)
	assert.ErrorIs(t, err, theme.ErrInvalidAssetAlt)
}

func TestAsset_SetAlt(t *testing.T) {
	a, _ := theme.NewAsset("tenant-1", "u", "f", "image/png", 10, 0, 0, "", "u", now)
	require.NoError(t, a.SetAlt(" hi "))
	assert.Equal(t, "hi", a.Alt)
	assert.ErrorIs(t, a.SetAlt(strings.Repeat("a", 251)), theme.ErrInvalidAssetAlt)
}

func TestReferencedAssetIDs_findsRefsInPropsStylesAndSettings(t *testing.T) {
	const a1 = "0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e"
	const a2 = "1b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e"
	const a3 = "2b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e"
	root := theme.Node{ID: "root", Type: theme.RootType, Children: []theme.Node{
		{ID: "s", Type: "section", Style: theme.NodeStyle{Mobile: map[string]string{"background_image": "asset:" + a1}}, Children: []theme.Node{
			{ID: "i", Type: "image", Props: map[string]any{"image": "asset:" + a2, "nested": []any{map[string]any{"x": "asset:" + a2}}, "text": "asset:not-a-uuid"}},
		}},
	}}
	s := theme.DefaultSettings()
	s.Brand.LogoAssetID = ptr(a3)

	assert.ElementsMatch(t, []string{a1, a2}, theme.ReferencedAssetIDs(root))
	assert.ElementsMatch(t, []string{a3}, s.ReferencedAssetIDs())
}

func ptr(s string) *string { return &s }
