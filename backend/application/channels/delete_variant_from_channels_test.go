package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/domain/events"
)

func TestDeleteVariantFromChannels_deletesAStandaloneWooProduct(t *testing.T) {
	client := &stubWooClient{}
	uc := appchannels.NewDeleteVariantFromChannelsUseCase(&fakeWooClients{client: client})

	err := uc.Execute(context.Background(), []events.VariantUnlistedListing{
		{ConnectionID: "woo", Provider: "woocommerce", ExternalID: "901"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"901"}, client.deleteProductCalled)
	assert.Empty(t, client.deleteVariationCalled)
}

func TestDeleteVariantFromChannels_deletesAWooVariationUnderItsParent(t *testing.T) {
	client := &stubWooClient{}
	uc := appchannels.NewDeleteVariantFromChannelsUseCase(&fakeWooClients{client: client})

	err := uc.Execute(context.Background(), []events.VariantUnlistedListing{
		{ConnectionID: "woo", Provider: "woocommerce", ExternalID: "801", ParentExternalID: "800"},
	})

	require.NoError(t, err)
	require.Len(t, client.deleteVariationCalled, 1)
	assert.Equal(t, "800", client.deleteVariationCalled[0].ParentExternalID)
	assert.Equal(t, "801", client.deleteVariationCalled[0].VariationExternalID)
	assert.Empty(t, client.deleteProductCalled)
}

func TestDeleteVariantFromChannels_skipsProvidersWithoutAnAdapter(t *testing.T) {
	clients := &fakeWooClients{client: &stubWooClient{}}
	uc := appchannels.NewDeleteVariantFromChannelsUseCase(clients)

	require.NoError(t, uc.Execute(context.Background(), []events.VariantUnlistedListing{
		{ConnectionID: "bol", Provider: "bol", ExternalID: "2994721002222"},
	}))
	assert.Empty(t, clients.gotID)
}

// Same contract as DeleteProductFromChannelsUseCase: best effort per
// listing -- one failure is logged, the rest still go.
func TestDeleteVariantFromChannels_oneFailureDoesNotStopTheRest(t *testing.T) {
	client := &stubWooClient{deleteProductErr: errors.New("woo down")}
	uc := appchannels.NewDeleteVariantFromChannelsUseCase(&fakeWooClients{client: client})

	err := uc.Execute(context.Background(), []events.VariantUnlistedListing{
		{ConnectionID: "woo", Provider: "woocommerce", ExternalID: "901"},
		{ConnectionID: "woo", Provider: "woocommerce", ExternalID: "902"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"901", "902"}, client.deleteProductCalled)
}
