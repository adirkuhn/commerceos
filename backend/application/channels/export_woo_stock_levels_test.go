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

func TestExportWooStockLevels_pushesCurrentSyncableLevelToWoo(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 7}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
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
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
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
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
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
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	result, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	require.Empty(t, h.wooClient.updateStockCalled)
}

func TestExportWooStockLevels_recordsErrorWhenWooUpdateFails(t *testing.T) {
	h, sku := newStockSyncHarness(nil)
	h.wooClient.updateStockErr = errors.New("woo 500")
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 7}}

	uc := appchannels.NewExportWooStockLevelsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
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
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, &stubStockLevelGetter{},
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
		h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, &stubStockLevelGetter{},
	)

	_, err := uc.Execute(context.Background(), appchannels.ExportWooStockLevelsInput{ConnectionID: "missing"})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}
