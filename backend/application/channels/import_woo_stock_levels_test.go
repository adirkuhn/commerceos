package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func wooProductWithStock(externalID string, qty int, manageStock bool) integrationsDomain.WooProduct {
	return integrationsDomain.WooProduct{ID: externalID, ManageStock: manageStock, StockQuantity: &qty}
}

// newStockSyncHarness wires a harness with one existing listing (product-1,
// external "23609") whose single variant/SKU is ready for stock
// reconciliation.
func newStockSyncHarness(products []integrationsDomain.WooProduct) (*harness, pmsDomain.SKU) {
	h := newHarness(products)
	product, variant, sku := simplePhysicalProduct()
	h.variantRepo.byProduct = map[string][]pmsDomain.Variant{product.ID: {variant}}
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{variant.ID: {sku}}
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "23609"},
	}
	return h, sku
}

func TestImportWooStockLevels_recordsAdjustmentWhenQuantitiesDiffer(t *testing.T) {
	h, sku := newStockSyncHarness([]integrationsDomain.WooProduct{wooProductWithStock("23609", 10, true)})
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}

	uc := appchannels.NewImportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooStockLevelsInput{ConnectionID: "conn-1", DefaultWarehouseID: "wh-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Updated)
	require.Equal(t, 0, result.Unchanged)
	require.Empty(t, result.Errors)
	require.Len(t, h.stock.calls, 1)
	require.Equal(t, int64(6), h.stock.calls[0].QuantityDelta)
	require.Equal(t, sku.ID, h.stock.calls[0].SKUID)

	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.SyncFlowImport, entry.Direction)
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	require.Equal(t, "10", entry.Fields["stock_quantity"])
}

func TestImportWooStockLevels_noopWhenQuantitiesMatch(t *testing.T) {
	h, sku := newStockSyncHarness([]integrationsDomain.WooProduct{wooProductWithStock("23609", 10, true)})
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 10}}

	uc := appchannels.NewImportWooStockLevelsUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Updated)
	require.Equal(t, 1, result.Unchanged)
	require.Empty(t, h.stock.calls)
}

func TestImportWooStockLevels_skipsProductsWooIsNotManagingStock(t *testing.T) {
	h, sku := newStockSyncHarness([]integrationsDomain.WooProduct{wooProductWithStock("23609", 10, false)})
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}

	uc := appchannels.NewImportWooStockLevelsUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Updated)
	require.Equal(t, 0, result.Unchanged)
	require.Empty(t, h.stock.calls)
}

func TestImportWooStockLevels_recordsErrorForMultiVariantProductAndContinues(t *testing.T) {
	h, sku := newStockSyncHarness([]integrationsDomain.WooProduct{wooProductWithStock("23609", 10, true)})
	product, _, _ := simplePhysicalProduct()
	extraVariant := pmsDomain.Variant{ID: "variant-2", ProductID: product.ID, Name: "Other"}
	h.variantRepo.byProduct[product.ID] = append(h.variantRepo.byProduct[product.ID], extraVariant)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}

	uc := appchannels.NewImportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	require.Empty(t, h.stock.calls)
}

func TestImportWooStockLevels_wrapsWooListFailure(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	h.wooClient.err = errors.New("timeout")
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}

	uc := appchannels.NewImportWooStockLevelsUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.stock,
	)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrWooRequestFailed))
}

func TestImportWooStockLevels_noopWhenNoListings(t *testing.T) {
	h := newHarness(nil)

	uc := appchannels.NewImportWooStockLevelsUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, &stubStockLevelGetter{}, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Updated)
	require.Empty(t, h.wooClient.createCalled)
}
