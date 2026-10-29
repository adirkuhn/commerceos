package channels_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// repairHarness: a separate-products-mode product with two variants listed
// on Woo as 701 (managed) and 702 (created unmanaged -- the bug).
func repairHarness() (*harness, []pmsDomain.SKU) {
	product, variants, skus := twoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "701", VariantID: strPtr(variants[0].ID)},
		{ID: "listing-2", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "702", VariantID: strPtr(variants[1].ID)},
	}
	qty := 3
	h.wooClient.products = []integrationsDomain.WooProduct{
		{ID: "701", Name: "Carne Seca - 378g", Type: "simple", ManageStock: true, StockQuantity: &qty},
		{ID: "702", Name: "Carne Seca - 412g", Type: "simple"},
	}
	return h, skus
}

func newRepairUseCase(t *testing.T, h *harness, syncer *stubStockForceSyncer) *appchannels.RepairUnmanagedWooStockUseCase {
	t.Helper()
	return appchannels.NewRepairUnmanagedWooStockUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.listingRepo, h.productRepo, h.variantRepo, h.skuRepo, syncer,
	)
}

func TestRepairUnmanagedWooStock_forceSyncsStockForUnmanagedPhysicalProduct(t *testing.T) {
	h, skus := repairHarness()
	syncer := &stubStockForceSyncer{}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 2, result.Checked)
	require.Equal(t, []appchannels.UnmanagedWooListing{{ExternalID: "702", Name: "Carne Seca - 412g", SKUID: skus[1].ID}}, result.Unmanaged)
	require.Equal(t, 1, result.Repaired)
	require.Equal(t, []string{skus[1].ID}, syncer.calls, "the managed 701 is left alone")
	require.Empty(t, result.Errors)
}

func TestRepairUnmanagedWooStock_dryRunReportsWithoutPushing(t *testing.T) {
	h, _ := repairHarness()
	syncer := &stubStockForceSyncer{}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1", DryRun: true})

	require.NoError(t, err)
	require.Len(t, result.Unmanaged, 1)
	require.Zero(t, result.Repaired)
	require.Empty(t, syncer.calls)
}

func TestRepairUnmanagedWooStock_leavesVirtualProductsUnmanaged(t *testing.T) {
	h, _ := repairHarness()
	h.wooClient.products[1].Virtual = true
	syncer := &stubStockForceSyncer{}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Empty(t, result.Unmanaged)
	require.Empty(t, syncer.calls)
}

func TestRepairUnmanagedWooStock_skipsVirtualSKUs(t *testing.T) {
	h, skus := repairHarness()
	virtual := skus[1]
	virtual.IsVirtual = true
	h.skuRepo.byVariant[virtual.VariantID] = []pmsDomain.SKU{virtual}
	syncer := &stubStockForceSyncer{}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Empty(t, result.Unmanaged)
	require.Empty(t, syncer.calls)
}

func TestRepairUnmanagedWooStock_skipsGatedListings(t *testing.T) {
	h, _ := repairHarness()
	gatedAt := time.Now()
	h.listingRepo.byConnection[1].GatedAt = &gatedAt
	syncer := &stubStockForceSyncer{}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Checked)
	require.Empty(t, syncer.calls, "a gated listing gets no stock sync until its product is Active again")
}

func TestRepairUnmanagedWooStock_countsVariationListingsAsNotChecked(t *testing.T) {
	h, _ := repairHarness()
	h.listingRepo.byConnection[1].IsVariation = true
	syncer := &stubStockForceSyncer{}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.SkippedVariations)
	require.Empty(t, syncer.calls)
}

func TestRepairUnmanagedWooStock_reportsSyncFailurePerListing(t *testing.T) {
	h, _ := repairHarness()
	syncer := &stubStockForceSyncer{err: errors.New("woo 500")}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Zero(t, result.Repaired)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0], "702")
}

func TestRepairUnmanagedWooStock_singleSKUProductListing(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "555"},
	}
	h.wooClient.products = []integrationsDomain.WooProduct{{ID: "555", Name: "T-Shirt", Type: "simple"}}
	syncer := &stubStockForceSyncer{}

	result, err := newRepairUseCase(t, h, syncer).Execute(context.Background(), appchannels.RepairUnmanagedWooStockInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, []string{sku.ID}, syncer.calls)
	require.Equal(t, 1, result.Repaired)
}
