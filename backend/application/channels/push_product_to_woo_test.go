package channels_test

import (
	"context"
	"errors"
	"fmt"
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

type stubStockLevelGetter struct {
	level *invapp.StockLevel
	err   error
}

func (s *stubStockLevelGetter) Execute(_ context.Context, _ string) (*invapp.StockLevel, error) {
	return s.level, s.err
}

// stubCategoryLinkResolver defaults to "no match" ("" , nil) -- the vast
// majority of push tests don't exercise category linking, so a fixed
// external ID would be a lie for them.
type stubCategoryLinkResolver struct {
	externalID string
	err        error
}

func (s *stubCategoryLinkResolver) Execute(_ context.Context, _ integrationsDomain.WooClient, _, _ string) (string, error) {
	return s.externalID, s.err
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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

// TestPushProductToWoo_draftProduct_neverPushed proves a Draft/Archived
// product is never created on Woo -- the initial-push counterpart to
// SyncProductToChannelsUseCase's gating of an already-linked product.
func TestPushProductToWoo_draftProduct_neverPushed(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.Status = pmsDomain.ProductStatusDraft
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.ErrorIs(t, err, channelsDomain.ErrProductNotSyncable)
	require.Empty(t, listings)
	require.Empty(t, h.wooClient.createCalled)
	require.Empty(t, h.listingRepo.created)
}

// TestPushProductToWoo_setsWooCategoryWhenLinkResolves proves a product's
// category is resolved via categoryLinkResolver and set on the pushed Woo
// product -- the fix for products always landing "Uncategorized" on Woo.
func TestPushProductToWoo_setsWooCategoryWhenLinkResolves(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	categoryID := "cat-1"
	product.CategoryID = &categoryID
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo,
		&stubCategoryLinkResolver{externalID: "20"}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	require.Equal(t, []integrationsDomain.WooProductCategory{{ExternalID: "20"}}, h.wooClient.createCalled[0].Categories)
}

// TestPushProductToWoo_noCategoryLinkLeavesProductUncategorized proves a
// product with no category set, or one with no Woo match, is pushed with
// no Categories -- same "Uncategorized" behavior as before this feature.
func TestPushProductToWoo_noCategoryLinkLeavesProductUncategorized(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo,
		&stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	require.Nil(t, h.wooClient.createCalled[0].Categories)
}

func TestPushProductToWoo_includesLotMetadataWhenStockLevelHasASourceBatch(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	expiry := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{
		SKUID: sku.ID, Syncable: 4,
		SourceBatch: &invapp.BatchSyncInfo{LotNumber: "LOT-42", ExpiryDate: expiry},
	}}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	require.Equal(t, []integrationsDomain.WooMetaData{
		{Key: "lot_number", Value: "LOT-42"},
		{Key: "expiry_date", Value: "2026-12-01"},
	}, h.wooClient.createCalled[0].MetaData)
}

func TestPushProductToWoo_sendsEmptyLotMetadataWhenStockLevelHasNoSourceBatch(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{level: &invapp.StockLevel{SKUID: sku.ID, Syncable: 4}}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	// Explicit empty values, not an omitted key -- so a SKU that later
	// drops out of the nearest-expiry tier clears any stale lot/expiry
	// still sitting on the Woo side from an earlier sync, rather than
	// leaving it untouched.
	require.Equal(t, []integrationsDomain.WooMetaData{
		{Key: "lot_number", Value: ""},
		{Key: "expiry_date", Value: ""},
	}, h.wooClient.createCalled[0].MetaData)
}

func TestPushProductToWoo_roundTripsWooTaxClassWhenSKUWasImportedFromWoo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	slug, status := "reduced-rate", "taxable"
	sku.WooTaxClassSlug, sku.WooTaxStatus = &slug, &status
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	require.Equal(t, "reduced-rate", h.wooClient.createCalled[0].TaxClass)
	require.Equal(t, "taxable", h.wooClient.createCalled[0].TaxStatus)
}

func TestPushProductToWoo_sendsNoTaxClassForANativeSKUNeverImportedFromWoo(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	require.Equal(t, "", h.wooClient.createCalled[0].TaxClass)
	require.Equal(t, "", h.wooClient.createCalled[0].TaxStatus)
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotPushVariableProduct))
	require.Empty(t, h.wooClient.createCalled)
}

// TestPushProductToWoo_relinksOrphanedProductLevelListingWhenFlippedToVariants
// is the direct fix for a real production bug: a product pushed while flat
// (!HasVariants, a plain simple product on Woo) later flipped into variants
// mode. Its one variant's ID never changes, but the pre-existing listing
// row stayed VariantID=nil (the flat shape) while this push now looks it up
// keyed by the variant's own ID -- finds nothing, tries to CREATE the same
// SKU on Woo again, which Woo correctly rejects since it already has it.
// createProductSelfHealingDuplicateSKU's own Woo-side SKU search is meant
// to catch this, but proved unreliable in production (Woo's search came up
// empty despite the create rejection proving the SKU exists) -- this
// relink needs no Woo API call at all, using only the row CommerceOS
// already has, so it can't be defeated by Woo's search being wrong.
func TestPushProductToWoo_relinksOrphanedProductLevelListingWhenFlippedToVariants(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.HasVariants = true // flipped after the original flat push
	h := newPushHarness(product, variant, sku)
	existing := channelsDomain.ProductChannelListing{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "999", VariantID: nil}
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{product.ID: {existing}}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Empty(t, h.wooClient.createCalled, "must not re-create on Woo -- the orphaned row already points at the real product")
	require.Len(t, listings, 1)
	require.Equal(t, "999", listings[0].ExternalID)
	require.NotNil(t, listings[0].VariantID)
	require.Equal(t, variant.ID, *listings[0].VariantID)
	require.Len(t, h.listingRepo.updated, 1)
	require.Equal(t, "listing-1", h.listingRepo.updated[0].ID)
	require.NotNil(t, h.listingRepo.updated[0].VariantID)
	require.Equal(t, variant.ID, *h.listingRepo.updated[0].VariantID)
}

// TestPushProductToWoo_doesNotRelinkTheVariablesParentRow guards the other
// nil-VariantID meaning: when SyncVariantsAsWooVariations is on, a
// nil-VariantID row is the variable product's own parent, not an orphaned
// legacy listing -- it must never be relinked to a variant's ID.
func TestPushProductToWoo_doesNotRelinkTheVariablesParentRow(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.HasVariants = true
	product.SyncVariantsAsWooVariations = true
	h := newPushHarness(product, variant, sku)
	parent := channelsDomain.ProductChannelListing{ID: "parent-listing", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "900", VariantID: nil}
	h.listingRepo.listByProduct = map[string][]channelsDomain.ProductChannelListing{product.ID: {parent}}
	h.wooClient.createVariationWithID = "901"
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Empty(t, h.listingRepo.updated, "the parent row must not be relinked to a variant")
	require.Empty(t, h.wooClient.createVariableCalled, "parent already exists, must not be recreated")
	require.Len(t, h.wooClient.createVariationCalled, 1)
}

func TestPushProductToWoo_wrapsWooCreateFailureAsErrWooRequestFailed(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.wooClient.createErr = errors.New("422 from woocommerce")
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrWooRequestFailed))
}

// TestPushProductToWoo_selfHealsWhenWooAlreadyHasTheSKU is a regression test:
// WooCommerce rejecting a create because the store already has a product
// with this SKU (its own uniqueness constraint) used to fail this push
// forever, even though the product genuinely IS already there -- this
// codebase's own product_channel_listings link to it had just been lost
// (e.g. a connection disconnect/reconnect). The fix looks the existing
// product up by SKU and links to it instead of retrying a create Woo will
// always reject the same way.
func TestPushProductToWoo_selfHealsWhenWooAlreadyHasTheSKU(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.wooClient.createErr = fmt.Errorf("%w: %w", integrationsDomain.ErrWooDuplicateSKU, errors.New("400 product_invalid_sku"))
	h.wooClient.findBySKUResult = &integrationsDomain.WooProduct{ID: "existing-woo-id", SKU: sku.SKUCode}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Equal(t, []string{sku.SKUCode}, h.wooClient.findBySKUCalled)
	require.Len(t, listings, 1)
	require.Equal(t, "existing-woo-id", listings[0].ExternalID)
	require.Equal(t, channelsDomain.SyncStatusSynced, listings[0].SyncStatus)
}

// TestPushProductToWoo_duplicateSKUSelfHealMissFallsBackToTheOriginalError
// proves that if the self-heal search itself comes up empty (or errors),
// the original duplicate-SKU failure still surfaces as ErrWooRequestFailed
// rather than silently succeeding with nothing linked.
func TestPushProductToWoo_duplicateSKUSelfHealMissFallsBackToTheOriginalError(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.wooClient.createErr = fmt.Errorf("%w: %w", integrationsDomain.ErrWooDuplicateSKU, errors.New("400 product_invalid_sku"))
	h.wooClient.findBySKUResult = nil // search itself finds nothing
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrWooRequestFailed))
	require.True(t, errors.Is(err, integrationsDomain.ErrWooDuplicateSKU))
}

// TestPushProductToWoo_selfHealsFromTrashWhenNormalSearchMisses is the fix
// for a real production case FindProductBySKU alone couldn't catch: the
// conflicting product was sitting in WordPress's trash, whose SKU is still
// reserved (blocking the create) despite being invisible to a normal
// "any"-status search. Confirmed in production: deleting the trashed
// product by hand let the push through immediately after -- this
// automates exactly that instead of requiring manual cleanup every time.
func TestPushProductToWoo_selfHealsFromTrashWhenNormalSearchMisses(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.wooClient.createErr = fmt.Errorf("%w: %w", integrationsDomain.ErrWooDuplicateSKU, errors.New("400 product_invalid_sku"))
	h.wooClient.createErrUntilCall = 1 // fails once (the original create), succeeds on retry after cleanup
	h.wooClient.createdWithID = "recreated-id"
	h.wooClient.findBySKUResult = nil // normal search misses -- it's in trash
	h.wooClient.findTrashedBySKUResult = &integrationsDomain.WooProduct{ID: "trashed-id", SKU: sku.SKUCode}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	listings, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Equal(t, []string{sku.SKUCode}, h.wooClient.findBySKUCalled)
	require.Equal(t, []string{sku.SKUCode}, h.wooClient.findTrashedBySKUCalled)
	require.Equal(t, []string{"trashed-id"}, h.wooClient.deleteProductCalled)
	require.Len(t, h.wooClient.createCalled, 2, "original create + retry after cleanup")
	require.Len(t, listings, 1)
	require.Equal(t, "recreated-id", listings[0].ExternalID)
}

// TestPushProductToWoo_trashSelfHealMissFallsBackToTheOriginalError proves
// that when nothing is in trash either (or deleting/retrying still fails),
// the original duplicate-SKU error still surfaces -- no confusing
// secondary failure, and no silent success with nothing linked.
func TestPushProductToWoo_trashSelfHealMissFallsBackToTheOriginalError(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.wooClient.createErr = fmt.Errorf("%w: %w", integrationsDomain.ErrWooDuplicateSKU, errors.New("400 product_invalid_sku"))
	h.wooClient.findBySKUResult = nil
	h.wooClient.findTrashedBySKUResult = nil // trash search also finds nothing
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrWooRequestFailed))
	require.True(t, errors.Is(err, integrationsDomain.ErrWooDuplicateSKU))
	require.Empty(t, h.wooClient.deleteProductCalled)
	require.Len(t, h.wooClient.createCalled, 1, "must not retry create when nothing was found in trash either")
}

func TestPushProductToWoo_pushesVirtualProductWithoutStockManagement(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.IsVirtual = true
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{err: errors.New("no ledger entries yet")}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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

// TestPushProductToWoo_physicalProduct_failsClosedWhenStockLookupFails: an
// unmanaged Woo product is "in stock" with no limit, so a failed stock
// lookup must create it out of stock (managed, 0), never unmanaged.
func TestPushProductToWoo_physicalProduct_failsClosedWhenStockLookupFails(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{err: errors.New("db down")}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	pushed := h.wooClient.createCalled[0]
	require.True(t, pushed.ManageStock)
	require.NotNil(t, pushed.StockQuantity)
	require.Equal(t, 0, *pushed.StockQuantity)
}

func TestPushProductToWoo_writesASyncLogEntryForTheNewListing(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}
	h.wooClient.createdWithID = "555"

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{ProductID: product.ID, ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, h.syncLogRepo.created, 1)
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.SyncFlowPush, entry.Direction)
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	require.Equal(t, "listing-1", *entry.ListingID)
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
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
// TestPushProductToWoo_includesProductImageGalleryOnCreate proves a
// product's media rows are pushed as the WooProduct's image gallery, in
// the same order MediaRepo.ListByProduct returns them (position asc, so
// the lowest-Position/"featured" row lands first, matching Woo's own
// "first image = featured" convention).
func TestPushProductToWoo_includesProductImageGalleryOnCreate(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	h.mediaRepo.byProduct = map[string][]pmsDomain.Media{
		product.ID: {
			{ID: "media-1", ProductID: &product.ID, URL: "https://cdn.example.com/front.jpg", AltText: "Front", Position: 0},
			{ID: "media-2", ProductID: &product.ID, URL: "https://cdn.example.com/back.jpg", AltText: "Back", Position: 1},
		},
	}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	require.Equal(t, []integrationsDomain.WooProductImage{
		{URL: "https://cdn.example.com/front.jpg", Alt: "Front"},
		{URL: "https://cdn.example.com/back.jpg", Alt: "Back"},
	}, h.wooClient.createCalled[0].Images)
}

func TestPushProductToWoo_sendsNoImagesWhenProductHasNoMedia(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Empty(t, h.wooClient.createCalled[0].Images)
}

// TestPushProductToWoo_separateProductsMode_eachVariantGetsTheSameProductImages
// proves every standalone simple product created in separate-products mode
// carries the same product-level gallery -- there's no per-variant media in
// the domain model, only per-product.
func TestPushProductToWoo_separateProductsMode_eachVariantGetsTheSameProductImages(t *testing.T) {
	product, variants, skus := twoVariantProduct(false)
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	h.mediaRepo.byProduct = map[string][]pmsDomain.Media{
		product.ID: {{ID: "media-1", ProductID: &product.ID, URL: "https://cdn.example.com/front.jpg", Position: 0}},
	}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 5), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 2)
	for _, p := range h.wooClient.createCalled {
		require.Equal(t, []integrationsDomain.WooProductImage{{URL: "https://cdn.example.com/front.jpg", Alt: "Carne Seca"}}, p.Images)
	}
}

// TestPushProductToWoo_variationsMode_onlyTheParentGetsImages proves the
// product's images land on the variable parent product, never on any
// per-variant WooVariation -- Woo variations have no images of their own,
// they inherit the parent's (see WooVariation's own doc comment).
func TestPushProductToWoo_variationsMode_onlyTheParentGetsImages(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	h.mediaRepo.byProduct = map[string][]pmsDomain.Media{
		product.ID: {{ID: "media-1", ProductID: &product.ID, URL: "https://cdn.example.com/front.jpg", Position: 0}},
	}
	h.wooClient.createVariableWithID = "900"
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 7), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createVariableCalled, 1)
	require.Equal(t, []integrationsDomain.WooProductImage{{URL: "https://cdn.example.com/front.jpg", Alt: "Carne Seca"}},
		h.wooClient.createVariableCalled[0].Product.Images)
}

func TestPushProductToWoo_pushesBarcodeAsGTIN(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.Barcode = "0012345678905"
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 4), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Len(t, h.wooClient.createCalled, 1)
	require.Equal(t, "0012345678905", h.wooClient.createCalled[0].GTIN)
}

// TestPushProductToWoo_variationsMode_eachVariationGetsItsOwnGTIN proves
// GTIN is per-SKU like SKU/price/stock, not parent-level like images --
// each variation carries its own barcode, not the product's.
func TestPushProductToWoo_variationsMode_eachVariationGetsItsOwnGTIN(t *testing.T) {
	product, variants, skus := twoVariantProduct(true)
	skus[0].Barcode = "0012345678905"
	skus[1].Barcode = "0019876543210"
	h := newMultiVariantSyncHarness(product, variants, skus, nil)
	h.wooClient.createVariableWithID = "900"
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 7), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Empty(t, h.wooClient.createVariableCalled[0].Product.GTIN, "parent has no GTIN of its own")
	require.Len(t, h.wooClient.createVariationCalled, 2)
	gtins := []string{h.wooClient.createVariationCalled[0].Variation.GTIN, h.wooClient.createVariationCalled[1].Variation.GTIN}
	require.ElementsMatch(t, []string{"0012345678905", "0019876543210"}, gtins)
}

func TestPushProductToWoo_rejectsPushWhenOnlyVariantIsBulkPool(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	sku.UnitType = pmsDomain.SKUUnitTypeKilo
	h := newPushHarness(product, variant, sku)
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel, h.mediaRepo, &stubCategoryLinkResolver{}, &eventrecordtest.Outbox{},
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotPushBulkPoolSKU))
	require.Empty(t, h.wooClient.createCalled)
}
