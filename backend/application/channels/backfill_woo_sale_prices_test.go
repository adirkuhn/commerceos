package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// backfillHarness sets up one already-imported, single-SKU product (no
// sale price recorded) linked to Woo product "23609" -- the shape every
// test below starts from and mutates.
func backfillHarness() (*harness, pmsDomain.Product, pmsDomain.Variant, pmsDomain.SKU) {
	product, variant, sku := simplePhysicalProduct()
	sku.SalePriceMinorUnits = nil
	h := newPushHarness(product, variant, sku)
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "23609"},
	}
	h.skuRepo.byID = map[string]pmsDomain.SKU{sku.ID: sku}
	return h, product, variant, sku
}

func TestBackfillWooSalePrices_fillsInMissingSalePriceForACurrentlyOnSaleProduct(t *testing.T) {
	h, _, _, _ := backfillHarness()
	wp := simpleWooProduct()
	wp.SalePrice, wp.OnSale = "0.99", true
	wp.SaleEndDate = "2026-12-01"
	h.wooClient.products = []integrationsDomain.WooProduct{wp}

	uc := appchannels.NewBackfillWooSalePricesUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.listingRepo, h.productRepo, h.variantRepo, h.skuRepo,
	)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooSalePricesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Candidates)
	require.Equal(t, 1, result.Updated)
	require.Empty(t, result.Errors)
	require.Len(t, h.skuRepo.updated, 1)
	require.NotNil(t, h.skuRepo.updated[0].SalePriceMinorUnits)
	require.Equal(t, int64(99), *h.skuRepo.updated[0].SalePriceMinorUnits)
	require.Equal(t, "2026-12-01", h.skuRepo.updated[0].SaleEndDate.Format("2006-01-02"))
}

func TestBackfillWooSalePrices_skipsSKUsThatAlreadyHaveASalePrice(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	salePrice := int64(120)
	sku.SalePriceMinorUnits = &salePrice // already set -- not a candidate
	h := newPushHarness(product, variant, sku)
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "23609"},
	}
	wp := simpleWooProduct()
	wp.SalePrice, wp.OnSale = "0.99", true
	h.wooClient.products = []integrationsDomain.WooProduct{wp}

	uc := appchannels.NewBackfillWooSalePricesUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.listingRepo, h.productRepo, h.variantRepo, h.skuRepo,
	)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooSalePricesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Candidates, "already has a sale price -- not a candidate at all")
	require.Equal(t, 0, result.Updated)
	require.Empty(t, h.skuRepo.updated)
}

func TestBackfillWooSalePrices_doesNotUpdateWhenWooHasNoActiveSale(t *testing.T) {
	h, _, _, _ := backfillHarness()
	wp := simpleWooProduct() // no sale configured at all
	h.wooClient.products = []integrationsDomain.WooProduct{wp}

	uc := appchannels.NewBackfillWooSalePricesUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.listingRepo, h.productRepo, h.variantRepo, h.skuRepo,
	)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooSalePricesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 1, result.Candidates)
	require.Equal(t, 0, result.Updated)
	require.Empty(t, h.skuRepo.updated)
}

func TestBackfillWooSalePrices_skipsVariableProducts(t *testing.T) {
	product, variants, skus := twoVariantProduct(true) // HasVariants + variations mode
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	h.listingRepo.byConnection = []channelsDomain.ProductChannelListing{
		// Variations-mode parent row -- VariantID nil, same as a single-SKU
		// product's row, but must still be excluded via HasVariants.
		{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "900"},
	}

	uc := appchannels.NewBackfillWooSalePricesUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.listingRepo, h.productRepo, h.variantRepo, h.skuRepo,
	)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooSalePricesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Candidates, "variable products were never importable in the first place")
	require.Empty(t, h.wooClient.products, "no Woo scan needed with zero candidates")
}

func TestBackfillWooSalePrices_noCandidatesIsANoOpWithoutCallingWoo(t *testing.T) {
	h, _, _, _ := backfillHarness()
	h.listingRepo.byConnection = nil // nothing linked at all

	uc := appchannels.NewBackfillWooSalePricesUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.listingRepo, h.productRepo, h.variantRepo, h.skuRepo,
	)

	result, err := uc.Execute(context.Background(), appchannels.BackfillWooSalePricesInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Equal(t, 0, result.Candidates)
	require.Equal(t, 0, result.Updated)
}
