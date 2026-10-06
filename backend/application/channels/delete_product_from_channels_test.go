package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestDeleteProductFromChannels_deletesEachListingOnWoo(t *testing.T) {
	h := newHarness(nil)
	uc := appchannels.NewDeleteProductFromChannelsUseCase(newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), []events.ProductDeletedListing{
		{ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"555"}, h.wooClient.deleteProductCalled)
}

func TestDeleteProductFromChannels_deletesMultipleListings(t *testing.T) {
	h := newHarness(nil)
	uc := appchannels.NewDeleteProductFromChannelsUseCase(newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), []events.ProductDeletedListing{
		{ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555"},
		{ConnectionID: "conn-2", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "556"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"555", "556"}, h.wooClient.deleteProductCalled)
}

func TestDeleteProductFromChannels_skipsNonWooCommerceProvider(t *testing.T) {
	h := newHarness(nil)
	uc := appchannels.NewDeleteProductFromChannelsUseCase(newMockGormDB(t, 0), h.connRepo, h.wooFactory, testEncryptionSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), []events.ProductDeletedListing{
		{ConnectionID: "conn-1", Provider: "internal", ExternalID: "555"},
	})

	require.NoError(t, err)
	assert.Empty(t, h.wooClient.deleteProductCalled)
}

// One listing's Woo failure must not stop the others from being attempted —
// same "log and continue" contract as EnrollProductInChannelsUseCase, since
// a permanently-deleted product can't be retried in a way that would ever
// self-heal a single bad connection.
func TestDeleteProductFromChannels_onePushFailureDoesNotBlockOthers(t *testing.T) {
	h := newHarness(nil)
	h.wooClient.deleteProductErr = errors.New("woo unreachable")
	uc := appchannels.NewDeleteProductFromChannelsUseCase(newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), []events.ProductDeletedListing{
		{ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555"},
		{ConnectionID: "conn-2", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "556"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"555", "556"}, h.wooClient.deleteProductCalled)
}

func TestDeleteProductFromChannels_noListingsIsANoOp(t *testing.T) {
	h := newHarness(nil)
	uc := appchannels.NewDeleteProductFromChannelsUseCase(newMockGormDB(t, 0), h.connRepo, h.wooFactory, testEncryptionSecret, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), nil)

	require.NoError(t, err)
	assert.Empty(t, h.wooClient.deleteProductCalled)
}
