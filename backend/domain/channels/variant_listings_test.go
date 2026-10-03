package channels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func strp(s string) *string { return &s }

func TestListingsOfVariant_separateProductsMode(t *testing.T) {
	all := []channelsDomain.ProductChannelListing{
		{ID: "l-380", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "900", VariantID: strp("v-380")},
		{ID: "l-222", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "901", VariantID: strp("v-222")},
		{ID: "l-bol", ConnectionID: "bol", Provider: "bol", ExternalID: "2994721002222", VariantID: strp("v-222")},
	}

	assert.Equal(t, []channelsDomain.VariantListingRef{
		{ListingID: "l-222", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "901"},
		{ListingID: "l-bol", ConnectionID: "bol", Provider: "bol", ExternalID: "2994721002222"},
	}, channelsDomain.ListingsOfVariant(all, "v-222"))
}

func TestListingsOfVariant_variationCarriesItsParentProductID(t *testing.T) {
	all := []channelsDomain.ProductChannelListing{
		{ID: "l-parent", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "800"},
		{ID: "l-222", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "801", VariantID: strp("v-222"), IsVariation: true},
	}

	assert.Equal(t, []channelsDomain.VariantListingRef{
		{ListingID: "l-222", ConnectionID: "woo", Provider: "woocommerce", ExternalID: "801", ParentExternalID: "800"},
	}, channelsDomain.ListingsOfVariant(all, "v-222"))
}

func TestListingsOfVariant_noneForAnUnlistedVariant(t *testing.T) {
	assert.Empty(t, channelsDomain.ListingsOfVariant([]channelsDomain.ProductChannelListing{
		{ID: "l-parent", ConnectionID: "woo", ExternalID: "800"},
	}, "v-222"))
}
