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

	listing, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Equal(t, "555", listing.ExternalID)
	require.Equal(t, channelsDomain.SyncStatusSynced, listing.SyncStatus)
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

func TestPushProductToWoo_returnsExistingListingWithoutRepushing(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	h := newPushHarness(product, variant, sku)
	existing := channelsDomain.ProductChannelListing{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "999"}
	h.listingRepo.byProduct[product.ID] = existing
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	listing, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{
		ProductID: product.ID, ConnectionID: "conn-1",
	})

	require.NoError(t, err)
	require.Equal(t, "999", listing.ExternalID)
	require.Empty(t, h.wooClient.createCalled)
}

func TestPushProductToWoo_rejectsProductWithVariants(t *testing.T) {
	product, variant, sku := simplePhysicalProduct()
	product.HasVariants = true
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
	h.listingRepo.byProduct[product.ID] = channelsDomain.ProductChannelListing{ID: "listing-1", ProductID: product.ID, ConnectionID: "conn-1", ExternalID: "999"}
	stockLevel := &stubStockLevelGetter{}

	uc := appchannels.NewPushProductToWooUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, stockLevel,
	)

	_, err := uc.Execute(context.Background(), appchannels.PushProductToWooInput{ProductID: product.ID, ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Empty(t, h.syncLogRepo.created, "no push happened, so nothing to log")
}
