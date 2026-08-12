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

type stubStockLevelGetter struct {
	level *invapp.StockLevel
	err   error
}

func (s *stubStockLevelGetter) Execute(_ context.Context, _ string) (*invapp.StockLevel, error) {
	return s.level, s.err
}

func simplePhysicalProduct() (pmsDomain.Product, pmsDomain.Variant, pmsDomain.SKU) {
	product := pmsDomain.Product{
		ID: "product-1", Name: "T-Shirt", Slug: "t-shirt", Description: "A shirt",
		Status: pmsDomain.ProductStatusActive, HasVariants: false,
	}
	variant := pmsDomain.Variant{ID: "variant-1", ProductID: "product-1", Name: "T-Shirt"}
	sku := pmsDomain.SKU{
		ID: "sku-1", VariantID: "variant-1", SKUCode: "TSHIRT-1",
		PriceMinorUnits: 1999, CurrencyCode: "EUR", IsVirtual: false, Status: pmsDomain.SKUStatusActive,
	}
	return product, variant, sku
}

func newPushHarness(product pmsDomain.Product, variant pmsDomain.Variant, sku pmsDomain.SKU) *harness {
	h := newHarness(nil)
	h.productRepo.bySlug = map[string]pmsDomain.Product{}
	h.productRepo.byID = map[string]pmsDomain.Product{product.ID: product}
	h.variantRepo.byProduct = map[string][]pmsDomain.Variant{product.ID: {variant}}
	h.skuRepo.byVariant = map[string][]pmsDomain.SKU{variant.ID: {sku}}
	return h
}

func TestPushProductToWoo_pushesNewPhysicalSimpleProduct(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	syncable := int64(4)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: syncable}}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, listings, 1)
	require.Equal(t, "555", listings[0].ExternalID)
	require.Equal(t, channelsDomain.SyncStatusSynced, listings[0].SyncStatus)
	require.Len(t, h.wooClient.createCalled, 1)
	pushed := h.wooClient.createCalled[0]
	require.Equal(t, "T-Shirt", pushed.Name)
	require.Equal(t, "publish", pushed.Status)
	require.Equal(t, "TSHIRT-1", pushed.SKU)
	require.Equal(t, "19.99", pushed.RegularPrice)
	require.False(t, pushed.Virtual)
	require.True(t, pushed.ManageStock)
	require.NotNil(t, pushed.StockQuantity)
	require.Equal(t, 4, *pushed.StockQuantity)
}

func TestPushProductToWoo_pushesSalePriceAndEndDate(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	salePrice := int64(1499)
	end := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	sku.SalePriceMinorUnits = &salePrice
	sku.SaleEndDate = &end
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	pushed := h.wooClient.createCalled[0]
	require.Equal(t, "14.99", pushed.SalePrice)
	require.Equal(t, "2026-08-10", pushed.SaleEndDate)
}

func TestPushProductToWoo_returnsExistingListingWithoutRepushing(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	existing := channelsDomain.ProductChannelListing{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "999"}
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{product.ID: {existing}}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, listings, 1)
	require.Equal(t, "999", listings[0].ExternalID)
	require.Empty(t, h.wooClient.createCalled)
}

// TestPushProductToWoo_rejectsVariantWithoutExactlyOneSKU proves the
// remaining guard once HasVariants=true products are allowed to push at
// all: every Variant still needs exactly one sellable SKU, since there's
// no single price/stock source of truth otherwise.
func TestPushProductToWoo_rejectsVariantWithoutExactlyOneSKU(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.HasVariants = true
	h := newPushHarness(product, variant, sku)
	h.skuRepo.byVariant[variant.ID] = nil // no SKU for this variant
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotPushVariableProduct))
	require.Empty(t, h.wooClient.createCalled)
}

func TestPushProductToWoo_wrapsWooCreateFailureAsErrWooRequestFailed(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.wooClient.createErr = errors.New("422 from woocommerce")
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrWooRequestFailed))
}

func TestPushProductToWoo_pushesVirtualProductWithoutStockManagement(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.IsVirtual = true
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{err: errors.New("no ledger entries yet")}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	pushed := h.wooClient.createCalled[0]
	require.True(t, pushed.Virtual)
	require.False(t, pushed.ManageStock)
	require.Nil(t, pushed.StockQuantity)
}

func TestPushProductToWoo_writesASyncLogEntryForTheNewListing(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{ProductID: product.ID, ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.SyncFlowPush, entry.Direction)
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	require.Equal(t, "listing-1", entry.ListingID)
	require.Equal(t, product.Name, entry.Fields["name"])
}

func TestPushProductToWoo_doesNotLogWhenAlreadyLinked(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{
		product.ID: {{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "999"}},
	}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{ProductID: product.ID, ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Empty(t, h.syncLogRepo.created, "no push happened, so nothing to log")
}

// TestPushProductToWoo_pushesEachVariantAsASeparateSimpleProduct is the
// "commerceos 1 product -> N variants -> woo N products" shape --
// SyncVariantsAsWooVariations is off, so every Variant becomes its own
// standalone simple Woo product.
func TestPushProductToWoo_pushesEachVariantAsASeparateSimpleProduct(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 5), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, listings, 2)
	require.Len(t, h.wooClient.createCalled, 2)
	require.Empty(t, h.wooClient.createVariableCalled, "no variable product in separate-products mode")
	names := []string{h.wooClient.createCalled[0].Name, h.wooClient.createCalled[1].Name}
	require.ElementsMatch(t, []string{"Carne Seca - 378g", "Carne Seca - 412g"}, names)
	for _, l := range listings {
		require.NotNil(t, l.VariantID)
		require.False(t, l.IsVariation)
	}
}

// TestPushProductToWoo_pushesVariantsAsOneVariableProductWithVariations is
// the "commerceos 1 product -> N variants -> woo 1 product -> N variants"
// shape -- SyncVariantsAsWooVariations is on, so one Woo variable product
// is created with one variation per Variant.
func TestPushProductToWoo_pushesVariantsAsOneVariableProductWithVariations(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	h.wooClient.createVariableWithID = "900"
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 7), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, listings, 3) // parent + 2 variations
	require.Len(t, h.wooClient.createVariableCalled, 1)
	call := h.wooClient.createVariableCalled[0]
	require.Equal(t, "Carne Seca", call.Product.Name)
	require.Equal(t, "Variant", call.AttributeName)
	require.ElementsMatch(t, []string{"378g", "412g"}, call.AttributeValues)
	require.Len(t, h.wooClient.createVariationCalled, 2)
	for _, c := range h.wooClient.createVariationCalled {
		require.Equal(t, "900", c.ParentExternalID)
	}
	require.Empty(t, h.wooClient.createCalled, "no simple products created in variations mode")

	var parentCount int
	for _, l := range listings {
		if l.VariantID == nil {
			parentCount++
			require.Equal(t, "900", l.ExternalID)
		} else {
			require.True(t, l.IsVariation)
		}
	}
	require.Equal(t, 1, parentCount)
}

// TestPushProductToWoo_variationsMode_onlyCreatesMissingVariations proves
// per-row idempotency: a parent that already exists (from an earlier
// partial push) isn't recreated, and a Variant that already has a
// variation isn't pushed again -- only the genuinely missing row is
// created.
func TestPushProductToWoo_variationsMode_onlyCreatesMissingVariations(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	existingParent := channelsDomain.ProductChannelListing{ID: "listing-parent", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "900"}
	existingVariant1 := channelsDomain.ProductChannelListing{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "901", VariantID: strPtr(variants[0].ID), IsVariation: true}
	h := newMultiVariantSyncHarness(product, variants, skus, []channelsDomain.ProductChannelListing{existingParent, existingVariant1})
	h.wooClient.createVariationWithID = "902"
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, listings, 3)
	require.Empty(t, h.wooClient.createVariableCalled, "parent already exists")
	require.Len(t, h.wooClient.createVariationCalled, 1)
	require.Equal(t, "900", h.wooClient.createVariationCalled[0].ParentExternalID)
}

// bulkPlusTwoVariantProduct is twoVariantProduct's sibling for the bulk-pool
// exclusion tests -- adds a third variant, a kilo (weight-pool) SKU the
// other two are (nominally) portioned from, which must never itself reach
// the channel.
func bulkPlusTwoVariantProduct(syncAsVariations bool) (pmsDomain.Product, []pmsDomain.Variant, []pmsDomain.SKU) {
	product, variants, skus := twoVariantProduct(syncAsVariations)
	bulkVariant := pmsDomain.Variant{ID: "variant-bulk", ProductID: product.ID, Name: "Bulk"}
	bulkSKU := pmsDomain.SKU{
		ID: "sku-bulk", VariantID: "variant-bulk", SKUCode: "CARNE-BULK", CurrencyCode: "EUR",
		Status: pmsDomain.SKUStatusActive, UnitType: pmsDomain.SKUUnitTypeKilo,
	}
	return product, append(variants, bulkVariant), append(skus, bulkSKU)
}

// TestPushProductToWoo_excludesBulkPoolVariantFromSeparateProductsPush
// proves a TracksWeight variant never reaches Woo alongside its sellable
// siblings in separate-products mode -- it's an internal weight ledger,
// not something to sell.
func TestPushProductToWoo_excludesBulkPoolVariantFromSeparateProductsPush(t *testing.T) {
	product, variants, skus := bulkPlusTwoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 5), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, listings, 2, "only the two sellable variants, never the bulk pool")
	require.Len(t, h.wooClient.createCalled, 2)
	for _, l := range listings {
		require.NotEqual(t, "variant-bulk", l.VariantID)
	}
}

// TestPushProductToWoo_excludesBulkPoolVariantFromVariationsPush is the
// variations-mode counterpart -- the bulk pool is never declared as an
// attribute option nor pushed as a variation.
func TestPushProductToWoo_excludesBulkPoolVariantFromVariationsPush(t *testing.T) {
	product, variants, skus := bulkPlusTwoVariantProduct(true)
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	h.wooClient.createVariableWithID = "900"
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 7), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, listings, 3, "parent + 2 sellable variations, never the bulk pool")
	require.ElementsMatch(t, []string{"378g", "412g"}, h.wooClient.createVariableCalled[0].AttributeValues)
	require.Len(t, h.wooClient.createVariationCalled, 2)
}

// TestPushProductToWoo_rejectsPushWhenOnlyVariantIsBulkPool covers a
// single-SKU product whose one and only SKU is itself the bulk pool --
// nothing sellable exists to push.
func TestPushProductToWoo_rejectsPushWhenOnlyVariantIsBulkPool(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.UnitType = pmsDomain.SKUUnitTypeKilo
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotPushBulkPoolSKU))
	require.Empty(t, h.wooClient.createCalled)
}
