package channels_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// stubStockLevelGetterBySKU is stubStockLevelGetter's per-SKU counterpart --
// needed to prove a per-listing fix actually resolves EACH listing's own
// SKU rather than reusing one shared value (stubStockLevelGetter's single
// fixed level can't distinguish that).
type stubStockLevelGetterBySKU struct {
	levels map[string]*invapp.StockLevel
}

func (s *stubStockLevelGetterBySKU) Execute(_ context.Context, skuID string) (*invapp.StockLevel, error) {
	level, ok := s.levels[skuID]
	if !ok {
		return nil, fmt.Errorf("stubStockLevelGetterBySKU: no level stubbed for SKU %s", skuID)
	}
	return level, nil
}

func newStockSyncTestHarness(listings []channelsDomain.ProductChannelListing) *harness {
	h := newHarness(nil)
	h.skuRepo.findByIDResult = &pmsDomain.SKU{ID: "sku-1", VariantID: "variant-1"}
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{"variant-1": {{ID: "sku-1"}}}
	h.variantRepo.findByIDResult = &pmsDomain.Variant{ID: "variant-1", ProductID: "product-1"}
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{"product-1": listings}
	return h
}

func (h *harness) newSyncStockUseCase(t *testing.T, n int, stockLevel *stubStockLevelGetter) *appchannels.SyncStockToChannelsUseCase {
	t.Helper()
	// nil literal, not a typed nil *stubBolProductPusher -- converts to a
	// genuinely nil bolProductPusher interface so uc.bolPusher == nil holds
	// inside the use case (a typed-nil pointer boxed into an interface is
	// famously NOT == nil, and would panic on stub.Execute for any test
	// that happens to include a Bol listing).
	return appchannels.NewSyncStockToChannelsUseCase(
		newMockGormDB(t, n), h.skuRepo, h.variantRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, stockLevel, testEncryptionSecret, nil, &eventrecordtest.Outbox{},
	)
}

func (h *harness) newSyncStockUseCaseWithBol(t *testing.T, n int, stockLevel *stubStockLevelGetter, bolPusher *stubBolProductPusher) *appchannels.SyncStockToChannelsUseCase {
	t.Helper()
	return appchannels.NewSyncStockToChannelsUseCase(
		newMockGormDB(t, n), h.skuRepo, h.variantRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, stockLevel, testEncryptionSecret, bolPusher, &eventrecordtest.Outbox{},
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

// TestSyncStockToChannels_skipsGatedListing proves a listing whose product
// left Active (GatedAt set by SyncProductToChannelsUseCase) gets no stock
// push at all -- not even the "quantity differs, worth a call" path above --
// until SyncProductToChannelsUseCase clears the gate on reactivation.
func TestSyncStockToChannels_skipsGatedListing(t *testing.T) {
	gatedAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
		GatedAt: &gatedAt,
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}}
	uc := h.newSyncStockUseCase(t, 1, stockLevel)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Empty(t, h.wooClient.updateStockCalled, "gated listing must receive no channel writes at all")
	require.Empty(t, h.syncLogRepo.created)
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

// TestSyncStockToChannels_pushesStockToBolViaFullOfferPush reproduces the
// reported bug: a stock change must reach every channel a product is
// enrolled in, not just Woo. Bol has no stock-only endpoint, so this
// dispatches to bolPusher.Execute (the same full-offer push used elsewhere)
// rather than a bespoke Bol stock call.
func TestSyncStockToChannels_pushesStockToBolViaFullOfferPush(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-bol", ProductID: "product-1", ConnectionID: "conn-bol",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "999",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	bolPusher := &stubBolProductPusher{}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 13}}
	uc := h.newSyncStockUseCaseWithBol(t, 1, stockLevel, bolPusher)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, bolPusher.calls, 1)
	require.Equal(t, "product-1", bolPusher.calls[0].ProductID)
	require.Equal(t, "conn-bol", bolPusher.calls[0].ConnectionID)
}

// TestSyncStockToChannels_bolPushHappensEvenWhenLastSyncReportedTheSameFields
// proves Bol has no "skip if unchanged" guard, unlike Woo -- a Bol listing
// existing at all already means the product is enrolled on that channel
// (EnrollProductInChannelsUseCase's job), so once enrolled, stock must
// always reach Bol, never lag on the bol_stock schedule's interval.
func TestSyncStockToChannels_bolPushHappensEvenWhenLastSyncReportedTheSameFields(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-bol", ProductID: "product-1", ConnectionID: "conn-bol",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "999",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	h.syncLogRepo.byListing = map[string][]channelsDomain.SyncLogEntry{
		"listing-bol": {{ListingID: strPtr("listing-bol"), Status: channelsDomain.SyncLogStatusSuccess, Fields: map[string]string{
			"stock_quantity": "13",
		}}},
	}
	bolPusher := &stubBolProductPusher{}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 13}}
	uc := h.newSyncStockUseCaseWithBol(t, 1, stockLevel, bolPusher)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, bolPusher.calls, 1, "Bol has no unchanged-skip guard -- every event must reach it")
}

// TestSyncStockToChannels_bolPushedOnceForMultipleListingsOnSameConnection
// proves a multi-variant product with several Bol listings under the same
// connection triggers exactly one full-offer push (which already covers
// every variant), not one redundant push per listing.
func TestSyncStockToChannels_bolPushedOnceForMultipleListingsOnSameConnection(t *testing.T) {
	variantID := "variant-2"
	listingA := channelsDomain.ProductChannelListing{
		ID: "listing-bol-a", ProductID: "product-1", ConnectionID: "conn-bol",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "999",
	}
	listingB := channelsDomain.ProductChannelListing{
		ID: "listing-bol-b", ProductID: "product-1", ConnectionID: "conn-bol",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "1000", VariantID: &variantID,
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listingA, listingB})
	bolPusher := &stubBolProductPusher{}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 13}}
	uc := h.newSyncStockUseCaseWithBol(t, 1, stockLevel, bolPusher)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, bolPusher.calls, 1, "one full-offer push already covers every listing on the connection")
}

// TestSyncStockToChannels_syncsBothWooAndBolListingsForTheSameProduct is the
// direct regression test for the reported bug -- a product enrolled in both
// Woo and Bol must have its stock change reach both, from one event.
func TestSyncStockToChannels_syncsBothWooAndBolListingsForTheSameProduct(t *testing.T) {
	wooListing := channelsDomain.ProductChannelListing{
		ID: "listing-woo", ProductID: "product-1", ConnectionID: "conn-woo",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555",
	}
	bolListing := channelsDomain.ProductChannelListing{
		ID: "listing-bol", ProductID: "product-1", ConnectionID: "conn-bol",
		Provider: string(integrationsDomain.ProviderBol), ExternalID: "999",
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{wooListing, bolListing})
	bolPusher := &stubBolProductPusher{}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 13}}
	uc := h.newSyncStockUseCaseWithBol(t, 5, stockLevel, bolPusher)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1)
	require.Len(t, bolPusher.calls, 1)
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

// TestSyncStockToChannels_forceExecuteBypassesTheAlreadySyncedSkip proves
// ForceExecute re-pushes even when the last successful sync already
// reported the exact same fields -- Execute's alreadySyncedTo short-circuit
// exists to avoid redundant Woo calls on the routine
// inventory.stock.changed path, but a deliberate manual/scheduled resync
// must always actually push.
func TestSyncStockToChannels_forceExecuteBypassesTheAlreadySyncedSkip(t *testing.T) {
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

	err := uc.ForceExecute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1, "force must push even though quantity/fields are unchanged from the last successful sync")
}

// TestSyncStockToChannels_eachVariantListingGetsItsOwnStockLevel is the
// direct regression test for the reported bug: adjusting one variant's
// stock (SKU "sku-1", triggering this event) must push ONLY that variant's
// own quantity to its own Woo listing -- the sibling variant's listing
// (SKU "sku-2", a different VariantID) must be pushed with ITS OWN current
// stock level, not sku-1's. Before the fix, stock level was computed once
// from the triggering SKU and reused for every listing in the fan-out
// loop, so both variants ended up pushed to the same (wrong, for one of
// them) quantity/lot/expiry.
func TestSyncStockToChannels_eachVariantListingGetsItsOwnStockLevel(t *testing.T) {
	variantA, variantB := "variant-1", "variant-2"
	listingA := channelsDomain.ProductChannelListing{
		ID: "listing-a", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555", VariantID: &variantA,
	}
	listingB := channelsDomain.ProductChannelListing{
		ID: "listing-b", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "556", VariantID: &variantB,
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listingA, listingB})
	// sku-1 (the triggering SKU) belongs to variant-1; sku-2 belongs to
	// variant-2, a sibling variant that was NOT touched by this event.
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{
		"variant-1": {{ID: "sku-1"}},
		"variant-2": {{ID: "sku-2"}},
	}
	stockLevel := &stubStockLevelGetterBySKU{levels: map[string]*invapp.StockLevel{
		"sku-1": {Syncable: 0, SourceBatch: nil},                                         // just zeroed out
		"sku-2": {Syncable: 71, SourceBatch: &invapp.BatchSyncInfo{LotNumber: "L61281"}}, // untouched, its own batch
	}}
	uc := appchannels.NewSyncStockToChannelsUseCase(
		newMockGormDB(t, 15), h.skuRepo, h.variantRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, stockLevel, testEncryptionSecret, nil, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 2)
	byExternalID := map[string]stubStockUpdate{}
	for _, call := range h.wooClient.updateStockCalled {
		byExternalID[call.ExternalID] = call
	}
	require.Equal(t, 0, byExternalID["555"].Quantity, "variant-1's own listing gets variant-1's own (just-changed) stock")
	require.Equal(t, "", byExternalID["555"].MetaData[0].Value, "variant-1 has no SourceBatch of its own -- must not inherit variant-2's lot")
	require.Equal(t, 71, byExternalID["556"].Quantity, "variant-2's listing must keep getting variant-2's OWN stock, not variant-1's")
	require.Equal(t, "L61281", byExternalID["556"].MetaData[0].Value, "variant-2's own lot must reach its own listing, not get overwritten by variant-1's (nil) batch")
}

// TestSyncStockToChannels_resolveListingSKUID_skipsBulkPoolSKUForTheSellableOne
// is a latent-bug fix found while investigating the eachVariantListingGets-
// ItsOwnStockLevel regression above: a variant with a bulk pool (UnitType
// kilo) ledger SKU alongside its actual sellable (portioned) SKU must push
// the SELLABLE SKU's stock, not whichever ListByVariant happens to return
// first (oldest by created_at) -- push_product_to_woo.go already excludes
// bulk-pool SKUs from what it pushes for exactly this reason;
// resolveListingSKUID didn't, until now.
func TestSyncStockToChannels_resolveListingSKUID_skipsBulkPoolSKUForTheSellableOne(t *testing.T) {
	variantID := "variant-2"
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555", VariantID: &variantID,
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	// The bulk-pool ledger SKU is listed first (oldest), the sellable
	// portioned SKU second -- resolveListingSKUID must skip past the pool.
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{
		"variant-2": {
			{ID: "sku-pool", UnitType: pmsDomain.SKUUnitTypeKilo},
			{ID: "sku-sellable", UnitType: pmsDomain.SKUUnitTypeUnit},
		},
	}
	stockLevel := &stubStockLevelGetterBySKU{levels: map[string]*invapp.StockLevel{
		"sku-sellable": {Syncable: 5},
	}}
	uc := appchannels.NewSyncStockToChannelsUseCase(
		newMockGormDB(t, 15), h.skuRepo, h.variantRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, stockLevel, testEncryptionSecret, nil, &eventrecordtest.Outbox{},
	)

	err := uc.Execute(context.Background(), "sku-1")

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1)
	require.Equal(t, 5, h.wooClient.updateStockCalled[0].Quantity, "must use the sellable SKU's stock, not the bulk-pool ledger's")
}

func TestSyncStockToChannels_propagatesSKUNotFound(t *testing.T) {
	h := newHarness(nil)
	stockLevel := &stubStockLevelGetter{}
	uc := h.newSyncStockUseCase(t, 1, stockLevel)

	err := uc.Execute(context.Background(), "missing-sku")

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrSKUNotFound))
}

// A product deleted in Woo (COM-18): the listing is marked missing and
// channels.product_listing.orphaned is raised -- once -- so it is handled
// (re-created or left marked) instead of retried on every change.
func TestSyncStockToChannels_marksTheListingMissingWhenWooNoLongerHasTheProduct(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "23981", SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	h.wooClient.updateStockErr = fmt.Errorf("%w: status 400: Invalid ID.", integrationsDomain.ErrWooProductNotFound)
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewSyncStockToChannelsUseCase(
		newMockGormDB(t, 5), h.skuRepo, h.variantRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}}, testEncryptionSecret, nil, outbox,
	)

	require.NoError(t, uc.Execute(context.Background(), "sku-1"))

	require.NotEmpty(t, h.listingRepo.updated)
	assert.Equal(t, channelsDomain.SyncStatusMissing, h.listingRepo.updated[len(h.listingRepo.updated)-1].SyncStatus)
	missing := eventsBySubject(outbox, "channels.product_listing.orphaned")
	require.Len(t, missing, 1)
	assert.Equal(t, "listing-1", missing[0].Payload["entity_id"])
}

func TestSyncStockToChannels_pushesNothingToAConnectionWithAMissingListing(t *testing.T) {
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: "product-1", ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "23981", SyncStatus: channelsDomain.SyncStatusMissing,
	}
	h := newStockSyncTestHarness([]channelsDomain.ProductChannelListing{listing})
	uc := h.newSyncStockUseCase(t, 1, &stubStockLevelGetter{level: &invapp.StockLevel{Syncable: 12}})

	require.NoError(t, uc.Execute(context.Background(), "sku-1"))

	assert.Empty(t, h.wooClient.updateStockCalled)
	assert.Empty(t, h.syncLogRepo.created)
}
