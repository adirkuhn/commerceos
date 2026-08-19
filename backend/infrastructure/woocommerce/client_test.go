package woocommerce

import (
	"testing"

	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestToWcImages_mapsURLAndAltInOrder(t *testing.T) {
	images := []integrationsDomain.WooProductImage{
		{URL: "https://cdn.example.com/a.jpg", Alt: "Front"},
		{URL: "https://cdn.example.com/b.jpg", Alt: "Back"},
	}

	out := toWcImages(images)

	require.Len(t, out, 2)
	require.Equal(t, "https://cdn.example.com/a.jpg", out[0].Source)
	require.Equal(t, "Front", out[0].Alt)
	require.Equal(t, "https://cdn.example.com/b.jpg", out[1].Source)
	require.Equal(t, "Back", out[1].Alt)
}

// TestToWcImages_nilForNoImages proves an empty gallery sends no images
// field at all (via omitempty), not an empty array that could clear an
// existing gallery on Woo's side for an UpdateProduct call whose media just
// hasn't loaded.
func TestToWcImages_nilForNoImages(t *testing.T) {
	require.Nil(t, toWcImages(nil))
	require.Nil(t, toWcImages([]integrationsDomain.WooProductImage{}))
}
