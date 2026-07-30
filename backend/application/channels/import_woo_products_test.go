package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
)

const testEncryptionSecret = "test-encryption-secret-32bytes!"

type harness struct {
	connRepo     *stubConnectionRepo
	wooFactory   *stubWooClientFactory
	wooClient    *stubWooClient
	categoryRepo *stubCategoryRepo
	productRepo  *stubChannelsProductRepo
	variantRepo  *stubChannelsVariantRepo
	skuRepo      *stubChannelsSKURepo
	listingRepo  *stubListingRepo
	stock        *stubStockRecorder
}

func newHarness(products []integrationsDomain.WooProduct) *harness {
	wooClient := &stubWooClient{products: products}
	encrypted, err := crypto.Encrypt(testEncryptionSecret, "consumer-secret")
	if err != nil {
		panic(err)
	}
	return &harness{
		connRepo: &stubConnectionRepo{conn: &integrationsDomain.Connection{
			ID: "conn-1", Provider: integrationsDomain.ProviderWooCommerce,
			StoreURL: "https://mystore.com", ConsumerKey: "ck", ConsumerSecretEncrypted: encrypted,
		}},
		wooFactory:   &stubWooClientFactory{client: wooClient},
		wooClient:    wooClient,
		categoryRepo: &stubCategoryRepo{bySlug: map[string]pmsDomain.Category{}},
		productRepo:  &stubChannelsProductRepo{bySlug: map[string]pmsDomain.Product{}},
		variantRepo:  &stubChannelsVariantRepo{},
		skuRepo:      &stubChannelsSKURepo{},
		listingRepo:  &stubListingRepo{byExternalID: map[string]channelsDomain.ProductChannelListing{}, byProduct: map[string]channelsDomain.ProductChannelListing{}},
		stock:        &stubStockRecorder{},
	}
}

func simpleWooProduct() integrationsDomain.WooProduct {
	stock := 2
	return integrationsDomain.WooProduct{
		ID: "23609", Name: "Salgadinho Pic sabor Bacon 40g", Slug: "salgadinho-pic-sabor-bacon-40g",
		SKU: "7896222211910", Description: "Salgadinho Pic sabor Bacon 40g",
		Status: "publish", Type: "simple", RegularPrice: "1.49",
		ManageStock: true, StockQuantity: &stock,
		Categories: []integrationsDomain.WooProductCategory{{ExternalID: "167", Name: "Salgadinhos e snacks", Slug: "salgadinhos-e-snacks"}},
	}
}

func TestImportWooProducts_importsNewSimpleProduct(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Len(t, h.productRepo.createdProducts, 1)
	require.Equal(t, "cat-1", *h.productRepo.createdProducts[0].CategoryID)
	require.Equal(t, pmsDomain.ProductStatusActive, h.productRepo.createdProducts[0].Status)
	require.False(t, h.productRepo.createdProducts[0].HasVariants)

	require.Len(t, h.variantRepo.created, 1)
	require.Len(t, h.skuRepo.created, 1)
	require.Equal(t, "7896222211910", h.skuRepo.created[0].SKUCode)
	require.Equal(t, int64(149), h.skuRepo.created[0].PriceMinorUnits)
	require.Equal(t, "EUR", h.skuRepo.created[0].CurrencyCode)
	require.False(t, h.skuRepo.created[0].IsVirtual)

	require.Len(t, h.listingRepo.created, 1)
	require.Equal(t, "23609", h.listingRepo.created[0].ExternalID)
	require.Equal(t, channelsDomain.SyncStatusSynced, h.listingRepo.created[0].SyncStatus)

	require.Len(t, h.stock.calls, 1, "manage_stock true must record an initial receipt")
	require.Equal(t, int64(2), h.stock.calls[0].QuantityDelta)
	require.Equal(t, invDomain.MovementTypeReceipt, h.stock.calls[0].Type)
}

func TestImportWooProducts_skipsNonSimpleProductType(t *testing.T) {
	variable := simpleWooProduct()
	variable.Type = "variable"
	h := newHarness([]integrationsDomain.WooProduct{variable})

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.SkippedType)
	require.Empty(t, h.productRepo.createdProducts)
}

func TestImportWooProducts_noopsWhenAlreadyLinked(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.listingRepo.byExternalID["23609"] = channelsDomain.ProductChannelListing{ID: "existing", ExternalID: "23609"}

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.AlreadyLinked)
	require.Empty(t, h.productRepo.createdProducts)
}

func TestImportWooProducts_doesNotRecordStockWhenManageStockFalse(t *testing.T) {
	p := simpleWooProduct()
	p.ManageStock = false
	p.StockQuantity = nil
	h := newHarness([]integrationsDomain.WooProduct{p})

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Empty(t, h.stock.calls)
}

func TestImportWooProducts_continuesAfterOneProductFailsToParse(t *testing.T) {
	bad := simpleWooProduct()
	bad.ID = "1"
	bad.RegularPrice = "not-a-price"
	good := simpleWooProduct()
	good.ID = "2"
	h := newHarness([]integrationsDomain.WooProduct{bad, good})

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 3), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Len(t, result.Errors, 1)
	require.Len(t, h.productRepo.createdProducts, 1)
}

func TestImportWooProducts_propagatesConnectionNotFound(t *testing.T) {
	h := newHarness(nil)
	h.connRepo.findErr = errors.New("connection not found")

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.Error(t, err)
}

func TestImportWooProducts_wrapsWooAPIFailureAsErrWooRequestFailed(t *testing.T) {
	h := newHarness(nil)
	h.wooClient.err = errors.New("500 from woocommerce")

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 1), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrWooRequestFailed))
}

func TestImportWooProducts_marksSKUVirtualWhenWooProductIsVirtual(t *testing.T) {
	digital := simpleWooProduct()
	digital.Virtual = true
	digital.ManageStock = false
	digital.StockQuantity = nil
	h := newHarness([]integrationsDomain.WooProduct{digital})

	uc := appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, 2), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.stock,
	)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	require.True(t, h.skuRepo.created[0].IsVirtual)
}
