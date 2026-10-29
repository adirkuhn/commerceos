package channels_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// These cover a variant added to a product that is already listed on Woo
// (e.g. "split into packages" adding a "~380g" variant): the new Woo
// product/variation must go up with managed stock, never unmanaged -- an
// unmanaged Woo product is "in stock" with no limit, so it can be ordered
// 10000x. The stock event for the new SKU usually lands before the listing
// row exists (SyncStockToChannelsUseCase then has nothing to push to), so a
// stock catch-up must also run once the listing is recorded.

func (h *harness) newSyncUseCaseWithStock(t *testing.T, n int, stock *stubStockLevelGetter, catchUp *stubStockForceSyncer) *appchannels.SyncProductToChannelsUseCase {
	t.Helper()
	return appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, n), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo, &stubCategoryLinkResolver{}, nil, nil, catchUp, stock, &eventrecordtest.Outbox{},
	)
}

func separateListingForFirstVariantOnly(productID, variantID string) []channelsDomain.ProductChannelListing {
	return []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: productID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "701", VariantID: strPtr(variantID)},
	}
}

func variationListingsForFirstVariantOnly(productID, variantID string) []channelsDomain.ProductChannelListing {
	return []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: productID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: productID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variantID), IsVariation: true},
	}
}

func TestSyncProductToChannels_newSeparateProduct_goesUpWithManagedStockAndLot(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, separateListingForFirstVariantOnly(product.ID, variants[0].ID))
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	stock := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: skus[1].ID, Syncable: 7, SourceBatch: &invapp.BatchSyncInfo{LotNumber: "L1", ExpiryDate: expiry}}}
	uc := h.newSyncUseCaseWithStock(t, 8, stock, &stubStockForceSyncer{})

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	require.Len(t, h.wooClient.createCalled, 1)
	created := h.wooClient.createCalled[0]
	require.True(t, created.ManageStock, "an unmanaged Woo product has unlimited stock")
	require.NotNil(t, created.StockQuantity)
	require.Equal(t, 7, *created.StockQuantity)
	require.Contains(t, created.MetaData, integrationsDomain.WooMetaData{Key: "expiry_date", Value: "2026-12-01"})
}

func TestSyncProductToChannels_newVariation_goesUpWithManagedStock(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	h := newMultiVariantSyncHarness(product, variants, skus, variationListingsForFirstVariantOnly(product.ID, variants[0].ID))
	h.wooClient.createVariationWithID = "903"
	stock := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: skus[1].ID, Syncable: 4}}
	uc := h.newSyncUseCaseWithStock(t, 8, stock, &stubStockForceSyncer{})

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	require.Len(t, h.wooClient.createVariationCalled, 1)
	created := h.wooClient.createVariationCalled[0].Variation
	require.True(t, created.ManageStock)
	require.NotNil(t, created.StockQuantity)
	require.Equal(t, 4, *created.StockQuantity)
}

func TestSyncProductToChannels_newSeparateProduct_failsClosedWhenStockLookupFails(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, separateListingForFirstVariantOnly(product.ID, variants[0].ID))
	stock := &stubStockLevelGetter{err: errors.New("db down")}
	uc := h.newSyncUseCaseWithStock(t, 8, stock, &stubStockForceSyncer{})

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	require.Len(t, h.wooClient.createCalled, 1)
	created := h.wooClient.createCalled[0]
	require.True(t, created.ManageStock, "out of stock until the next sync, never unlimited")
	require.NotNil(t, created.StockQuantity)
	require.Equal(t, 0, *created.StockQuantity)
}

func TestSyncProductToChannels_newListing_catchesUpStockOnceRecorded(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, separateListingForFirstVariantOnly(product.ID, variants[0].ID))
	catchUp := &stubStockForceSyncer{}
	uc := h.newSyncUseCaseWithStock(t, 8, &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 0}}, catchUp)

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	require.Len(t, h.listingRepo.created, 1)
	require.Equal(t, []string{skus[1].ID}, catchUp.calls, "only the newly listed SKU -- existing listings get stock from their own events")
}

func TestSyncProductToChannels_newVariation_catchesUpStockOnceRecorded(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	h := newMultiVariantSyncHarness(product, variants, skus, variationListingsForFirstVariantOnly(product.ID, variants[0].ID))
	h.wooClient.createVariationWithID = "903"
	catchUp := &stubStockForceSyncer{}
	uc := h.newSyncUseCaseWithStock(t, 8, &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 2}}, catchUp)

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	require.Equal(t, []string{skus[1].ID}, catchUp.calls)
}

func TestSyncProductToChannels_updateOfExistingListing_neverSendsStock(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "701", VariantID: strPtr(variants[0].ID)},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "702", VariantID: strPtr(variants[1].ID)},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	catchUp := &stubStockForceSyncer{}
	uc := h.newSyncUseCaseWithStock(t, 8, &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 9}}, catchUp)

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	require.Len(t, h.wooClient.updateProductCalled, 2)
	for _, call := range h.wooClient.updateProductCalled {
		require.False(t, call.Product.ManageStock, "stock on an existing listing is SyncStockToChannelsUseCase's job")
		require.Nil(t, call.Product.StockQuantity)
	}
	require.Empty(t, catchUp.calls)
}

func TestSyncProductToChannels_newListing_syncLogRecordsStockSent(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, separateListingForFirstVariantOnly(product.ID, variants[0].ID))
	uc := h.newSyncUseCaseWithStock(t, 8, &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 5}}, &stubStockForceSyncer{})

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	var createLog *channelsDomain.SyncLogEntry
	for i, e := range h.syncLogRepo.created {
		if e.ExternalID == "new-external-id" {
			createLog = &h.syncLogRepo.created[i]
		}
	}
	require.NotNil(t, createLog)
	require.Equal(t, "5", createLog.Fields["stock_quantity"])
}
