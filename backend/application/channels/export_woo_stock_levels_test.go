package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestExportWooStockLevels_pushesCurrentSyncableLevelToWoo(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 7}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, &eventrecordtest.Outbox{},
	)

	result, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Updated)
	require.Empty(t, result.Errors)
	require.Len(t, h.wooClient.updateStockCalled, 1)
	require.Equal(t, "23609", h.wooClient.updateStockCalled[0].ExternalID)
	require.Equal(t, 7, h.wooClient.updateStockCalled[0].Quantity)

	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.SyncFlowPush, entry.Direction)
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	require.Equal(t, "7", entry.Fields["stock_quantity"])
}

func TestExportWooStockLevels_includesLotMetadataWhenStockLevelHasASourceBatch(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{
		SKUID: sku.ID, Syncable: 7, SourceBatch: &invapp.BatchSyncInfo{LotNumber: "LOT-42", ExpiryDate: expiry},
	}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, h.wooClient.updateStockCalled, 1)
	require.Equal(t, []integrationsDomain.WooMetaData{
		{Key: "lot_number", Value: "LOT-42"},
		{Key: "expiry_date", Value: "2026-12-01"},
	}, h.wooClient.updateStockCalled[0].MetaData)
	require.Equal(t, "LOT-42", h.syncLogRepo.created[0].Fields["lot_number"])
}

func TestExportWooStockLevels_skipsVirtualSKUs(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	_, variant, _ := simplePhysicalProduct()
	sku.IsVirtual = true
	h.skuRepo.byVariant[variant.ID] = []pmsDomain.SKU{sku}
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 7}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, &eventrecordtest.Outbox{},
	)

	result, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Updated)
	require.Empty(t, h.wooClient.updateStockCalled)
}

func TestExportWooStockLevels_recordsErrorForMultiVariantProductAndContinues(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	product, _, _ := simplePhysicalProduct()
	extraVariant := pmsDomain.Variant{ID: "variant-2", ProductID: product.ID, Name: "Other"}
	h.variantRepo.byProduct[product.ID] = append(h.variantRepo.byProduct[product.ID], extraVariant)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 7}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, &eventrecordtest.Outbox{},
	)

	result, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	require.Empty(t, h.wooClient.updateStockCalled)
}

// TestExportWooStockLevels_pushesEachVariantsStockForAPerVariantListingSet
// is the exact incident reported 2026-09-11/12 for "Guaraná Antarctica
// Garrafa" and "Guaraná Antarctica Zero Garrafa": a real multi-variant
// product (HasVariants=true, e.g. two bottle sizes), each variant with
// exactly one SKU, pushed to Woo in variations mode -- one
// ProductChannelListing per variant, VariantID set on each, IsVariation
// true, plus the sibling parent row (VariantID nil) for the variable
// product itself.
//
// pushOne used to always call variantRepo.ListByProduct(listing.ProductID)
// and require the *whole product* to have exactly one variant, completely
// ignoring listing.VariantID -- so every push for a real multi-variant
// product failed with ErrCannotPushVariableProduct ("every variant must
// have exactly one SKU to be pushed to a channel") even though each
// variant, individually, already had exactly one SKU. The fix mirrors
// sync_stock_to_channels.go's resolveListingSKUID: when listing.VariantID
// is set, resolve that variant's SKU directly instead of re-deriving it
// from the whole product.
func TestExportWooStockLevels_pushesEachVariantsStockForAPerVariantListingSet(t *testing.T) {
	h := newHarness(nil)
	product := pmsDomain.Product{
		ID: "product-1", Name: "Guaraná Antarctica Garrafa", Slug: "guarana-antarctica-garrafa",
		Status: pmsDomain.ProductStatusActive, HasVariants: true,
	}
	variant1 := pmsDomain.Variant{ID: "variant-1", ProductID: product.ID, Name: "1.5L"}
	variant2 := pmsDomain.Variant{ID: "variant-2", ProductID: product.ID, Name: "1.5L"}
	sku1 := pmsDomain.SKU{ID: "sku-1", VariantID: variant1.ID, SKUCode: "5601151960455", Status: pmsDomain.SKUStatusActive}
	sku2 := pmsDomain.SKU{ID: "sku-2", VariantID: variant2.ID, SKUCode: "5601045300008", Status: pmsDomain.SKUStatusActive}

	h.variantRepo.byProduct = map[string][]pmsDomain.Variant{product.ID: {variant1, variant2}}
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{
		variant1.ID: {sku1},
		variant2.ID: {sku2},
	}
	parentVariantID := (*string)(nil)
	v1ID, v2ID := variant1.ID, variant2.ID
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "9000", VariantID: parentVariantID},
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "9001", VariantID: &v1ID, IsVariation: true},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "9002", VariantID: &v2ID, IsVariation: true},
	}
	stockLevel := &stubStockLevelGetterBySKU{levels: map[string]*invapp.StockLevel{
		sku1.ID: {SKUID: sku1.ID, Syncable: 12},
		sku2.ID: {SKUID: sku2.ID, Syncable: 5},
	}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 5), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, &eventrecordtest.Outbox{},
	)

	result, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Equal(t, 2, result.Updated, "the parent listing has no SKU of its own and is skipped, not errored -- only the two variant listings push")
	require.Len(t, h.wooClient.updateStockCalled, 2)

	byExternalID := map[string]stubStockUpdate{}
	for _, c := range h.wooClient.updateStockCalled {
		byExternalID[c.ExternalID] = c
	}
	require.Equal(t, 12, byExternalID["9001"].Quantity)
	require.Equal(t, "9000", byExternalID["9001"].ParentExternalID, "a variation-mode listing must target the parent variable product, not \"\"")
	require.Equal(t, 5, byExternalID["9002"].Quantity)
	require.Equal(t, "9000", byExternalID["9002"].ParentExternalID)
}

func TestExportWooStockLevels_recordsErrorWhenWooUpdateFails(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	h.wooClient.updateStockErr = errors.New("woo 500")
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 7}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, &eventrecordtest.Outbox{},
	)

	result, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Updated)
	require.Len(t, result.Errors, 1)
}

func TestExportWooStockLevels_noopWhenNoListings(t *testing.T) {
	h := newHarness(nil)

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, &stubStockLevelGetter{}, &eventrecordtest.Outbox{},
	)

	result, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Updated)
	require.Empty(t, h.wooClient.updateStockCalled)
}

func TestExportWooStockLevels_wrapsConnectionNotFound(t *testing.T) {
	h := newHarness(nil)
	h.connRepo.findErr = integrationsDomain.ErrConnectionNotFound

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, &stubStockLevelGetter{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "missing"})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
