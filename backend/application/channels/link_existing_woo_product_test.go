package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

func newLinkExistingWooProductHarness(t *testing.T) (*appchannels.LinkExistingWooProductUseCase, *stubChannelsProductRepo, *stubConnectionRepo, *stubListingRepo, *stubWooClient) {
	t.Helper()
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	require.NoError(t, err)
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"prod-1": {ID: "prod-1"}}}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: "conn-1", ConsumerSecretEncrypted: encrypted}}
	listingRepo := &stubListingRepo{listByProduct: map[string][]channelsDomain.ProductChannelListing{}}
	wooClient := &stubWooClient{}
	wooFactory := &stubWooClientFactory{client: wooClient}
	uc := appchannels.NewLinkExistingWooProductUseCase(newMockGormDB(t, 2), productRepo, connRepo, listingRepo, wooFactory, testEncryptionSecret)
	return uc, productRepo, connRepo, listingRepo, wooClient
}

func TestLinkExistingWooProduct_linksTheFoundWooProduct(t *testing.T) {
	uc, _, _, listingRepo, wooClient := newLinkExistingWooProductHarness(t)
	wooClient.findBySKUResult = &integrationsDomain.WooProduct{ID: "23609", SKU: "TSHIRT-1"}

	listing, err := uc.Execute(context.Background(), appchannels.LinkExistingWooProductInput{
		ProductID: "prod-1", ConnectionID: "conn-1", SKU: "TSHIRT-1",
	})

	require.NoError(t, err)
	require.NotNil(t, listing)
	assert.Equal(t, "23609", listing.ExternalID)
	assert.Equal(t, "prod-1", listing.ProductID)
	assert.Equal(t, "conn-1", listing.ConnectionID)
	assert.Equal(t, channelsDomain.SyncStatusSynced, listing.SyncStatus)
	assert.Nil(t, listing.VariantID, "product-level link -- no variant scoping")
	require.Len(t, listingRepo.created, 1)
	assert.Equal(t, []string{"TSHIRT-1"}, wooClient.findBySKUCalled)
}

func TestLinkExistingWooProduct_noMatchingWooProduct_returnsTypedError(t *testing.T) {
	uc, _, _, listingRepo, wooClient := newLinkExistingWooProductHarness(t)
	wooClient.findBySKUResult = nil

	_, err := uc.Execute(context.Background(), appchannels.LinkExistingWooProductInput{
		ProductID: "prod-1", ConnectionID: "conn-1", SKU: "NO-SUCH-SKU",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, integrationsDomain.ErrWooProductNotFoundBySKU)
	assert.Empty(t, listingRepo.created, "must not create a listing when nothing was found")
}

func TestLinkExistingWooProduct_productAlreadyHasAListingForThisConnection_returnsTypedError(t *testing.T) {
	uc, _, _, listingRepo, wooClient := newLinkExistingWooProductHarness(t)
	listingRepo.listByProduct["prod-1"] = []channelsDomain.ProductChannelListing{
		{ID: "listing-existing", ProductID: "prod-1", ConnectionID: "conn-1", ExternalID: "999"},
	}

	_, err := uc.Execute(context.Background(), appchannels.LinkExistingWooProductInput{
		ProductID: "prod-1", ConnectionID: "conn-1", SKU: "TSHIRT-1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, channelsDomain.ErrProductAlreadyLinkedToConnection)
	assert.Empty(t, wooClient.findBySKUCalled, "must not even call Woo once already linked")
	assert.Len(t, listingRepo.created, 0)
}

func TestLinkExistingWooProduct_productNotFound_propagates(t *testing.T) {
	uc, productRepo, _, _, _ := newLinkExistingWooProductHarness(t)
	productRepo.byID = map[string]pmsDomain.Product{}

	_, err := uc.Execute(context.Background(), appchannels.LinkExistingWooProductInput{
		ProductID: "missing", ConnectionID: "conn-1", SKU: "TSHIRT-1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, pmsDomain.ErrProductNotFound)
}

func TestLinkExistingWooProduct_propagatesWooClientError(t *testing.T) {
	uc, _, _, listingRepo, wooClient := newLinkExistingWooProductHarness(t)
	wooClient.findBySKUErr = errors.New("woo unreachable")

	_, err := uc.Execute(context.Background(), appchannels.LinkExistingWooProductInput{
		ProductID: "prod-1", ConnectionID: "conn-1", SKU: "TSHIRT-1",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, integrationsDomain.ErrWooRequestFailed)
	assert.Empty(t, listingRepo.created)
}
