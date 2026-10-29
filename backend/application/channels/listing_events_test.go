package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func eventsBySubject(o *eventrecordtest.Outbox, subject string) []eventrecordtest.Event {
	var out []eventrecordtest.Event
	for _, e := range o.Events {
		if e.Subject == subject {
			out = append(out, e)
		}
	}
	return out
}

func TestLinkExistingWooProduct_RecordsListingCreated(t *testing.T) {
	_, productRepo, connRepo, listingRepo, wooClient := newLinkExistingWooProductHarness(t)
	wooClient.findBySKUResult = &integrationsDomain.WooProduct{ID: "23609", SKU: "TSHIRT-1"}
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewLinkExistingWooProductUseCase(newMockGormDB(t, 2), productRepo, connRepo, listingRepo,
		&stubWooClientFactory{client: wooClient}, testEncryptionSecret, outbox)

	_, err := uc.Execute(context.Background(), appchannels.LinkExistingWooProductInput{ProductID: "prod-1", ConnectionID: "conn-1", SKU: "TSHIRT-1"})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "channels.product_listing.created", e.Subject)
	assert.Equal(t, map[string]any{
		"product_id": "prod-1", "variant_id": nil, "connection_id": "conn-1", "provider": "woocommerce",
		"external_id": "23609", "is_variation": false, "source": "link_existing",
	}, e.Data())
}

// Each written sync log entry is also a channels.product_listing.synced
// event, carrying the pushed fields that changed.
func TestExportWooStockLevels_RecordsListingSynced(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	outbox := &eventrecordtest.Outbox{}
	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		&stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 7}}, outbox,
	)

	_, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	synced := eventsBySubject(outbox, "channels.product_listing.synced")
	require.Len(t, synced, 1)
	data := synced[0].Data()
	assert.Equal(t, "push", data["direction"])
	assert.Equal(t, "success", data["status"])
	assert.Equal(t, "23609", data["external_id"])
	assert.Equal(t, [2]any{"", "7"}, synced[0].Changes()["stock_quantity"])
}

func TestSyncProductToChannels_RecordsListingCreatedForNewVariant(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
	}
	h := newMultiVariantSyncHarness(product, variants, skus, listings)
	h.wooClient.createVariationWithID = "903"
	outbox := &eventrecordtest.Outbox{}
	uc := newSyncUCWithOutbox(t, h, 6, outbox)

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	created := eventsBySubject(outbox, "channels.product_listing.created")
	require.Len(t, created, 1)
	assert.Equal(t, "903", created[0].Data()["external_id"])
	assert.Equal(t, variants[1].ID, created[0].Data()["variant_id"])
	assert.Equal(t, "variant_added", created[0].Data()["source"])
}

func TestSyncProductToChannels_RecordsListingDeletedForRemovedVariant(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	listings := []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "900"},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "902", VariantID: strPtr(variants[1].ID), IsVariation: true},
	}
	h := newMultiVariantSyncHarness(product, variants[:1], skus, listings)
	outbox := &eventrecordtest.Outbox{}
	uc := newSyncUCWithOutbox(t, h, 5, outbox)

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	deleted := eventsBySubject(outbox, "channels.product_listing.deleted")
	require.Len(t, deleted, 1)
	assert.Equal(t, "listing-2", deleted[0].Payload["entity_id"])
	assert.Equal(t, "variant_removed", deleted[0].Data()["reason"])
	assert.Equal(t, "902", deleted[0].Data()["external_id"])
}

func TestSyncProductToChannels_RecordsListingGated(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.Status = pmsDomain.ProductStatusDraft
	listing := channelsDomain.ProductChannelListing{
		ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1",
		Provider: string(integrationsDomain.ProviderWooCommerce), ExternalID: "555", SyncStatus: channelsDomain.SyncStatusSynced,
	}
	h := newSyncHarness(product, variant, sku, []channelsDomain.ProductChannelListing{listing})
	outbox := &eventrecordtest.Outbox{}
	uc := newSyncUCWithOutbox(t, h, 10, outbox)

	require.NoError(t, uc.Execute(context.Background(), product.ID))

	gated := eventsBySubject(outbox, "channels.product_listing.gated")
	require.Len(t, gated, 1)
	assert.Equal(t, "listing-1", gated[0].Payload["entity_id"])
	assert.Equal(t, product.ID, gated[0].Data()["product_id"])
}

func newSyncUCWithOutbox(t *testing.T, h *harness, n int, outbox *eventrecordtest.Outbox) *appchannels.SyncProductToChannelsUseCase {
	t.Helper()
	return appchannels.NewSyncProductToChannelsUseCase(
		newMockGormDB(t, n), h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo,
		h.connRepo, h.wooFactory, testEncryptionSecret, &stubProductPusher{}, h.mediaRepo, &stubCategoryLinkResolver{}, nil, nil, nil, nil, outbox,
	)
}
