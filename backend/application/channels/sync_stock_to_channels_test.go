package channels_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func newStockSyncTestHarness(listings []channelsDomain.ProductChannelListing) *harness {
	h := newHarness(nil)
	h.skuRepo.findByIDResult = &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}
	h.variantRepo.findByIDResult = &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{"product-1": listings}
	return h
}

func (h *harness) newSyncStockUseCase(t *testing.T, n int, stockLevel *stubStockLevelGetter) *appchannels.SyncStockToChannelsUseCase {
	t.Helper()
	return appchannels.NewSyncStockToChannelsUseCase(
		newMockGormDB(t, n), h.skuRepo, h.variantRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, stockLevel, testEncryptionSecret,
	)
}

func TestSyncStockToChannels_noListings_isNoOp(t *testing.T) {
	h := newStockSyncTestHarness(nil)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 5}}
	uc := h.newSyncStockUseCase(t, 1, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateStockCalled)
}

func TestSyncStockToChannels_pushesCurrentSyncableStockToWoo(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}}
	uc := h.newSyncStockUseCase(t, 5, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1)
	require.Equal(t, "555", h.wooClient.updateStockCalled[0].ExternalID)
	require.Equal(t, 12, h.wooClient.updateStockCalled[0].Quantity)

	require.Len(t, h.syncLogRepo.created, 1)
	require.Equal(t, "12", h.syncLogRepo.created[0].Fields["stock_quantity"])
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, h.syncLogRepo.created[0].Status)
}

func TestSyncStockToChannels_skipsWooCallWhenAlreadySyncedToSameQuantity(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{ListingID: strPtr("listing-1"), Status: channelsDomain.SyncLogStatusSuccess, Fields: map[string]string{
			"stock_quantity": "12", "lot_number": "", "expiry_date": "",
		}}},
	}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}}
	uc := h.newSyncStockUseCase(t, 5, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateStockCalled, "quantity unchanged since last successful sync -- no Woo call needed")
	require.Empty(t, h.syncLogRepo.created)
}

func TestSyncStockToChannels_pushesAgainWhenQuantityDiffers(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{ListingID: strPtr("listing-1"), Status: channelsDomain.SyncLogStatusSuccess, Fields: map[string]string{"stock_quantity": "5"}}},
	}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}}
	uc := h.newSyncStockUseCase(t, 5, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1)
	require.Equal(t, 12, h.wooClient.updateStockCalled[0].Quantity)
}

func TestSyncStockToChannels_includesLotMetadataWhenStockLevelHasASourceBatch(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{
		Syncable: 12, SourceBatch: &invapp.BatchSyncInfo{LotNumber: "LOT-42", ExpiryDate: expiry},
	}}
	uc := h.newSyncStockUseCase(t, 5, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1)
	require.Equal(t, []integrationsDomain.WooMetaData{
		{Key: "lot_number", Value: "LOT-42"},
		{Key: "expiry_date", Value: "2026-12-01"},
	}, h.wooClient.updateStockCalled[0].MetaData)
	require.Equal(t, "LOT-42", h.syncLogRepo.created[0].Fields["lot_number"])
}

func TestSyncStockToChannels_pushesAgainWhenOnlyTheLotChangesEvenIfQuantityIsTheSame(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{ListingID: strPtr("listing-1"), Status: channelsDomain.SyncLogStatusSuccess, Fields: map[string]string{
			"stock_quantity": "12", "lot_number": "LOT-OLD", "expiry_date": "2026-09-01",
		}}},
	}
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{
		Syncable: 12, SourceBatch: &invapp.BatchSyncInfo{LotNumber: "LOT-NEW", ExpiryDate: expiry},
	}}
	uc := h.newSyncStockUseCase(t, 5, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1, "same qty but a different lot must still re-push")
	require.Equal(t, "LOT-NEW", h.wooClient.updateStockCalled[0].MetaData[0].Value)
}

func TestSyncStockToChannels_recordsErrorStatusWhenWooUpdateFails(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	h.wooClient.updateStockErr = errors.New("woo unreachable")
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}}
	uc := h.newSyncStockUseCase(t, 5, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	require.Equal(t, channelsDomain.SyncLogStatusError, h.syncLogRepo.created[0].Status)
	require.Contains(t, h.syncLogRepo.created[0].ErrorMessage, "woo unreachable")
}

func TestSyncStockToChannels_skipsListingsForUnimplementedProviders(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: "shopify", ExternalID: "555",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}}
	uc := h.newSyncStockUseCase(t, 5, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateStockCalled)
	require.Empty(t, h.syncLogRepo.created)
}

// TestSyncStockToChannels_variationListingUsesParentAsRoutingID proves a
// variation-mode listing (IsVariation=true, ExternalID is the Woo
// variation id, not a standalone product id) gets its sibling
// product-level row's ExternalID passed as the parent id -- Woo variation
// stock lives at /products/{parent}/variations/{id}, a different endpoint
// than a standalone product's /products/{id}, so pushing without the
// parent id would hit the wrong resource entirely.
func TestSyncStockToChannels_variationListingUsesParentAsRoutingID(t *testing.T) {
	variantID := "variant-1"
	parentListing := channelsDomain.ProductChannelListing{
		ID: "listing-parent", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900",
	}
	variationListing := channelsDomain.ProductChannelListing{
		ID: "listing-variation", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901",
		VariantID: &variantID, IsVariation: true,
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{parentListing, variationListing})
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 0}}
	uc := h.newSyncStockUseCase(t, 10, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 2)
	byExternalID := map[string]stubStockUpdate{}
	for _, call := range h.wooClient.updateStockCalled {
		byExternalID[call.ExternalID] = call
	}
	require.Equal(t, "", byExternalID["900"].ParentExternalID, "parent-level listing is not itself a variation")
	require.Equal(t, "900", byExternalID["901"].ParentExternalID, "variation listing must route through its parent product id")
}

func TestSyncStockToChannels_propagatesSKUNotFound(t *testing.T) {
	h := newHarness(nil)
	stockLevel := &stubStockLevelGetter{}
	uc := h.newSyncStockUseCase(t, 1, stockLevel)

	err := uc.Execute(context.Background(), "missing-sku")

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}
