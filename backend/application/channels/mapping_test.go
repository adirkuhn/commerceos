package channels

import (
	"testing"

	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// TestWooImagesFromMedia_fallsBackToProductNameWhenAltTextEmpty is the fix
// for a Woo product image shipping with a blank alt attribute: a media row
// with no AltText set (the common case -- nothing prompts a merchant to
// fill it in on upload) used to send Alt: "" straight to Woo. Falling back
// to the product's own name keeps every synced image accessible/SEO-sane
// without forcing a manual edit on each one.
func TestWooImagesFromMedia_fallsBackToProductNameWhenAltTextEmpty(t *testing.T) {
	media := []pmsDomain.Media{
		{ID: "media-1", URL: "https://cdn.example.com/front.jpg", AltText: ""},
	}

	images := wooImagesFromMedia(media, "Carne Seca")

	require.Equal(t, []integrationsDomain.WooProductImage{
		{URL: "https://cdn.example.com/front.jpg", Alt: "Carne Seca"},
	}, images)
}

// TestWooImagesFromMedia_keepsExplicitAltText proves the fallback only
// fires on empty AltText -- an explicitly-set title is never overridden by
// the product name.
func TestWooImagesFromMedia_keepsExplicitAltText(t *testing.T) {
	media := []pmsDomain.Media{
		{ID: "media-1", URL: "https://cdn.example.com/front.jpg", AltText: "Front of the pack"},
	}

	images := wooImagesFromMedia(media, "Carne Seca")

	require.Equal(t, []integrationsDomain.WooProductImage{
		{URL: "https://cdn.example.com/front.jpg", Alt: "Front of the pack"},
	}, images)
}

// TestWooImagesFromMedia_appliesFallbackPerImage proves each image in a
// multi-image gallery gets its own alt resolved independently -- one
// missing title among several set ones doesn't fall back for the whole
// gallery, nor does the product name only apply to the first image.
func TestWooImagesFromMedia_appliesFallbackPerImage(t *testing.T) {
	media := []pmsDomain.Media{
		{ID: "media-1", URL: "https://cdn.example.com/front.jpg", AltText: "Front"},
		{ID: "media-2", URL: "https://cdn.example.com/back.jpg", AltText: ""},
	}

	images := wooImagesFromMedia(media, "Carne Seca")

	require.Equal(t, []integrationsDomain.WooProductImage{
		{URL: "https://cdn.example.com/front.jpg", Alt: "Front"},
		{URL: "https://cdn.example.com/back.jpg", Alt: "Carne Seca"},
	}, images)
}
