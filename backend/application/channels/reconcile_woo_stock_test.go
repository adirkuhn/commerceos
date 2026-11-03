package channels_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func intp(v int) *int { return &v }

// snapshot is what Woo reports for a simple product in step with a SKU
// holding qty and no batch metadata.
func snapshot(id string, qty int) integrationsDomain.WooStockSnapshot {
	return integrationsDomain.WooStockSnapshot{ID: id, Type: "simple", ManageStock: true, StockQuantity: intp(qty),
		Meta: map[string]string{"lot_number": "", "expiry_date": ""}}
}

type reconcileFixture struct {
	h      *harness
	levels *stubStockLevelGetterBySKU
	outbox *eventrecordtest.Outbox
}

// newReconcileFixture: product-1 has variant-1 with sku-1 (12 syncable, no
// batch), listed on conn-1 as Woo product 23981.
func newReconcileFixture(listings ...channelsDomain.ProductChannelListing) *reconcileFixture {
	h := newHarness(nil)
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{"variant-1": {{ID: "sku-1", VariantID: "variant-1"}}}
	h.variantRepo.byProduct = map[string][]pmsDomain.Variant{"product-1": {{ID: "variant-1", ProductID: "product-1"}}}
	if len(listings) == 0 {
		listings = []channelsDomain.ProductChannelListing{wooListing("listing-1", "23981")}
	}
	h.listingRepo.byConnection = listings
	return &reconcileFixture{
		h:      h,
		levels: &stubStockLevelGetterBySKU{levels: map[string]*invapp.StockLevel{"sku-1": {SKUID: "sku-1", Syncable: 12}}},
		outbox: &eventrecordtest.Outbox{},
	}
}

func wooListing(id, externalID string) channelsDomain.ProductChannelListing {
	return channelsDomain.ProductChannelListing{
		ID: id, ProductID: "product-1", ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce),
		ExternalID: externalID, SyncStatus: channelsDomain.SyncStatusSynced,
	}
}

func (f *reconcileFixture) run(t *testing.T) (*appchannels.StockReconcileResult, error) {
	t.Helper()
	uc := appchannels.NewReconcileWooStockUseCase(newMockGormDB(t, 20), f.h.connRepo, f.h.wooFactory, testEncryptionSecret,
		f.h.variantRepo, f.h.skuRepo, f.h.listingRepo, f.h.syncLogRepo, f.levels, f.outbox)
	return uc.Execute(context.Background(), "conn-1")
}

func TestReconcileWooStock_pushesNothingWhenWooIsInStep(t *testing.T) {
	f := newReconcileFixture()
	f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{snapshot("23981", 12)}}

	result, err := f.run(t)

	require.NoError(t, err)
	assert.Empty(t, f.h.wooClient.updateStockCalled, "steady state: zero PUTs")
	assert.Equal(t, appchannels.StockReconcileResult{Checked: 1}, *result)
	assert.Empty(t, f.h.syncLogRepo.created)
}

func TestReconcileWooStock_pushesOnlyAListingWhoseWooStockDrifted(t *testing.T) {
	f := newReconcileFixture(wooListing("listing-1", "23981"), wooListing("listing-2", "23982"))
	f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{snapshot("23981", 9)}, {snapshot("23982", 12)}}

	result, err := f.run(t)

	require.NoError(t, err)
	require.Len(t, f.h.wooClient.updateStockCalled, 1, "every page is read; only the drifted listing is pushed")
	push := f.h.wooClient.updateStockCalled[0]
	assert.Equal(t, "23981", push.ExternalID)
	assert.Equal(t, 12, push.Quantity)
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 1, result.Pushed)
	require.Len(t, f.h.syncLogRepo.created, 1)
	assert.Equal(t, channelsDomain.SyncLogStatusSuccess, f.h.syncLogRepo.created[0].Status)
	assert.Equal(t, "12", f.h.syncLogRepo.created[0].Fields["stock_quantity"])
}

// The stock rules also govern batch metadata: a different lot, or stock
// not managed by Woo, is drift even at the right quantity.
func TestReconcileWooStock_batchMetadataAndManageStockCountAsDrift(t *testing.T) {
	for name, woo := range map[string]integrationsDomain.WooStockSnapshot{
		"other lot":            {ID: "23981", ManageStock: true, StockQuantity: intp(12), Meta: map[string]string{"lot_number": "OLD", "expiry_date": "2027-01-31"}},
		"stock not managed":    {ID: "23981", ManageStock: false, StockQuantity: intp(12), Meta: map[string]string{"lot_number": "L42", "expiry_date": "2027-01-31"}},
		"no quantity reported": {ID: "23981", ManageStock: true, Meta: map[string]string{"lot_number": "L42", "expiry_date": "2027-01-31"}},
	} {
		t.Run(name, func(t *testing.T) {
			f := newReconcileFixture()
			f.levels.levels["sku-1"].SourceBatch = &invapp.BatchSyncInfo{LotNumber: "L42", ExpiryDate: time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)}
			f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{woo}}

			result, err := f.run(t)

			require.NoError(t, err)
			require.Len(t, f.h.wooClient.updateStockCalled, 1)
			assert.Equal(t, []integrationsDomain.WooMetaData{{Key: "lot_number", Value: "L42"}, {Key: "expiry_date", Value: "2027-01-31"}},
				f.h.wooClient.updateStockCalled[0].MetaData)
			assert.Equal(t, 1, result.Pushed)
		})
	}
}

func TestReconcileWooStock_aProductGoneFromWooIsMarkedMissingNotPushed(t *testing.T) {
	f := newReconcileFixture()
	f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{snapshot("1", 3)}}

	result, err := f.run(t)

	require.NoError(t, err)
	assert.Empty(t, f.h.wooClient.updateStockCalled)
	assert.Equal(t, 1, result.Missing)
	require.NotEmpty(t, f.h.listingRepo.updated)
	assert.Equal(t, channelsDomain.SyncStatusMissing, f.h.listingRepo.updated[0].SyncStatus)
	orphaned := eventsBySubject(f.outbox, "channels.product_listing.orphaned")
	require.Len(t, orphaned, 1)
	assert.Equal(t, "listing-1", orphaned[0].Payload["entity_id"])
}

// A listing already marked missing is raised again each run, so a product
// enrolled (or activated) later is created again by its handler.
func TestReconcileWooStock_reRaisesAListingAlreadyMissing(t *testing.T) {
	missing := wooListing("listing-1", "23981")
	missing.SyncStatus = channelsDomain.SyncStatusMissing
	f := newReconcileFixture(missing)

	result, err := f.run(t)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Missing)
	assert.Empty(t, f.h.wooClient.updateStockCalled)
	assert.Len(t, eventsBySubject(f.outbox, "channels.product_listing.orphaned"), 1)
}

func TestReconcileWooStock_aVariationIsComparedWithItsVariationAndPushedUnderItsParent(t *testing.T) {
	variant := "variant-1"
	parent := wooListing("listing-parent", "500")
	variation := wooListing("listing-var", "501")
	variation.VariantID, variation.IsVariation = &variant, true
	f := newReconcileFixture(parent, variation)
	f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{{ID: "500", Type: "variable"}}}
	f.h.wooClient.variationStock = map[string][]integrationsDomain.WooStockSnapshot{"500": {snapshot("501", 2)}}

	result, err := f.run(t)

	require.NoError(t, err)
	require.Len(t, f.h.wooClient.updateStockCalled, 1, "the parent row has no stock of its own")
	assert.Equal(t, "500", f.h.wooClient.updateStockCalled[0].ParentExternalID)
	assert.Equal(t, "501", f.h.wooClient.updateStockCalled[0].ExternalID)
	assert.Equal(t, 1, result.Checked)
}

func TestReconcileWooStock_skipsGatedListingsAndVirtualSKUs(t *testing.T) {
	gatedAt := time.Now()
	gated := wooListing("listing-1", "23981")
	gated.GatedAt = &gatedAt
	f := newReconcileFixture(gated)
	f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{snapshot("23981", 0)}}

	result, err := f.run(t)
	require.NoError(t, err)
	assert.Empty(t, f.h.wooClient.updateStockCalled)
	assert.Equal(t, 0, result.Checked)

	f = newReconcileFixture()
	f.h.skuRepo.byVariant["variant-1"][0].IsVirtual = true
	f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{snapshot("23981", 0)}}
	_, err = f.run(t)
	require.NoError(t, err)
	assert.Empty(t, f.h.wooClient.updateStockCalled)
}

func TestReconcileWooStock_aFailedPushIsReportedAndTheRestContinue(t *testing.T) {
	f := newReconcileFixture()
	f.h.wooClient.stockPages = [][]integrationsDomain.WooStockSnapshot{{snapshot("23981", 1)}}
	f.h.wooClient.updateStockErr = errors.New("woo down")

	result, err := f.run(t)

	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "woo down")
	assert.Equal(t, 0, result.Pushed)
}

func TestReconcileWooStock_cannotReadTheStore(t *testing.T) {
	f := newReconcileFixture()
	f.h.wooClient.stockErr = errors.New("woo down")

	_, err := f.run(t)

	require.Error(t, err)
	assert.Empty(t, f.h.listingRepo.updated, "nothing is marked missing when the store can't be read")
}
