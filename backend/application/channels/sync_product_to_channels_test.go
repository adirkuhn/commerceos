package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func newSyncHarness(product pmsDomain.Product, variant pmsDomain.Variant, sku pmsDomain.SKU, listings []channelsDomain.ProductChannelListing) *harness {
	h := newPushHarness(product, variant, sku)
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{product.ID: listings}
	return h
}

func (h *harness) newSyncUseCase(t *testing.T, n int) *appchannels.SyncProductToChannelsUseCase {
	t.Helper()
	return appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, n), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret,
	)
}

func TestSyncProductToChannels_noListings_isNoOp(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newSyncHarness(product, variant, sku, nil)
	uc := h.newSyncUseCase(t, 1)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled)
}

func TestSyncProductToChannels_pushesUpdatedFieldsToWoo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	call := h.wooClient.updateProductCalled[0]
	require.Equal(t, "555", call.ExternalID)
	require.Equal(t, product.Name, call.Product.Name)
	require.Equal(t, product.Description, call.Product.Description)
	require.Equal(t, "publish", call.Product.Status)
	require.Equal(t, "19.99", call.Product.RegularPrice)

	require.Len(t, h.listingRepo.updated, 1)
	require.Equal(t, channelsDomain.SyncStatusSynced, h.listingRepo.updated[0].SyncStatus)
	require.NotNil(t, h.listingRepo.updated[0].LastSyncedAt)
}

func TestSyncProductToChannels_convertsWeightAndDimensionsUsingStoreUnits(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	weight, length, width, height := int64(500), int64(300), int64(200), int64(100) // grams/mm
	sku.WeightGrams = &weight
	sku.LengthMM = &length
	sku.WidthMM = &width
	sku.HeightMM = &height
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.weightUnit = "kg"
	h.wooClient.dimensionUnit = "cm"
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateProductCalled, 1)
	call := h.wooClient.updateProductCalled[0].Product
	require.Equal(t, "0.5", call.Weight)
	require.Equal(t, "30", call.Dimensions.Length)
	require.Equal(t, "20", call.Dimensions.Width)
	require.Equal(t, "10", call.Dimensions.Height)
}

func TestSyncProductToChannels_neverSetsWeightOrDimensionsForVirtualProducts(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.IsVirtual = true
	weight := int64(500)
	sku.WeightGrams = &weight
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.weightUnit = "kg"
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled[0].Product.Weight)
	require.True(t, h.wooClient.updateProductCalled[0].Product.Virtual)
}

func TestSyncProductToChannels_recordsErrorStatusWhenWooUpdateFails_withoutFailingTheUseCase(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.updateProductErr = errors.New("woo unreachable")
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.listingRepo.updated, 1)
	require.Equal(t, channelsDomain.SyncStatusError, h.listingRepo.updated[0].SyncStatus)
	require.Contains(t, h.listingRepo.updated[0].LastSyncError, "woo unreachable")
}

func TestSyncProductToChannels_skipsListingsWhenProductHasGrownVariants(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.HasVariants = true
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 1)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled)
}

func TestSyncProductToChannels_skipsListingsForUnimplementedProviders(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: "shopify", ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 1)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateProductCalled)
	require.Empty(t, h.listingRepo.updated)
}

func TestSyncProductToChannels_writesASyncLogEntryOnSuccess(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, "listing-1", entry.ListingID)
	require.Equal(t, channelsDomain.SyncFlowPush, entry.Direction)
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	require.Equal(t, product.Name, entry.Fields["name"])
	require.Equal(t, "19.99", entry.Fields["regular_price"])
	// First-ever entry for this listing -- everything shows as new (from "").
	require.Equal(t, channelsDomain.FieldChange{From: "", To: product.Name}, entry.Changes["name"])
}

func TestSyncProductToChannels_diffsAgainstThePreviousSyncLogEntry(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{ListingID: "listing-1", Fields: map[string]string{"name": "Old Name", "regular_price": "19.99"}}},
	}
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	changes := h.syncLogRepo.created[0].Changes
	require.Equal(t, channelsDomain.FieldChange{From: "Old Name", To: product.Name}, changes["name"])
	_, priceChanged := changes["regular_price"]
	require.False(t, priceChanged, "unchanged field must not appear in the diff")
}

func TestSyncProductToChannels_writesASyncLogEntryOnFailureToo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	h.wooClient.updateProductErr = errors.New("woo unreachable")
	uc := h.newSyncUseCase(t, 4)

	err := uc.Execute(context.Background(), product.ID)

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.SyncLogStatusError, entry.Status)
	require.Contains(t, entry.ErrorMessage, "woo unreachable")
}
