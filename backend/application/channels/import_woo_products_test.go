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
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
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
	syncLogRepo  *stubSyncLogRepo
	stock        *stubStockRecorder
	media        *stubMediaUploader
	mediaRepo    *stubMediaRepo
	taxClassRepo *stubChannelsTaxClassRepo
	taxRateRepo  *stubChannelsTaxRateRepo
	taxLinkRepo  *stubChannelsExternalTaxClassLinkRepo
}

func newHarness(products []integrationsDomain.WooProduct) *harness {
	// Default categories to a flat listing derived from each product's own
	// embedded category (no parent) — matches every existing test's
	// expectations. Tests exercising hierarchy specifically override
	// h.wooClient.categories after construction.
	var categories []integrationsDomain.WooCategory
	seen := map[string]bool{}
	for _, p := range products {
		for _, c := range p.Categories {
			if seen[c.ExternalID] {
				continue
			}
			seen[c.ExternalID] = true
			categories = append(categories, integrationsDomain.WooCategory{ExternalID: c.ExternalID, Name: c.Name, Slug: c.Slug})
		}
	}
	wooClient := &stubWooClient{products: products, categories: categories}
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
		syncLogRepo:  &stubSyncLogRepo{},
		stock:        &stubStockRecorder{},
		media:        &stubMediaUploader{},
		mediaRepo:    &stubMediaRepo{},
		taxClassRepo: &stubChannelsTaxClassRepo{},
		taxRateRepo:  &stubChannelsTaxRateRepo{},
		taxLinkRepo:  &stubChannelsExternalTaxClassLinkRepo{},
	}
}

// newImportUseCase builds an ImportWooProductsUseCase from the harness with
// n tenant.WithTenant calls expected — same convention as newMockGormDB.
func (h *harness) newImportUseCase(t *testing.T, n int) *appchannels.ImportWooProductsUseCase {
	t.Helper()
	return appchannels.NewImportWooProductsUseCase(
		newMockGormDB(t, n), h.connRepo, h.wooFactory, testEncryptionSecret,
		h.categoryRepo, h.productRepo, h.variantRepo, h.skuRepo, h.listingRepo, h.syncLogRepo, h.stock, h.media,
		h.taxClassRepo, h.taxRateRepo, h.taxLinkRepo,
	)
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

	uc := h.newImportUseCase(t, 4)

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

	require.Len(t, h.stock.calls, 1, "manage_stock true must record initial stock")
	require.Equal(t, int64(2), h.stock.calls[0].QuantityDelta)
	require.Equal(t, invDomain.MovementTypeAdjustment, h.stock.calls[0].Type)

	require.Len(t, h.syncLogRepo.created, 1, "only the import fields log -- initial stock raises no push/echo entry")
	entry := h.syncLogRepo.created[0]
	require.Equal(t, channelsDomain.SyncFlowImport, entry.Direction)
	require.Equal(t, channelsDomain.SyncLogStatusSuccess, entry.Status)
	require.Equal(t, "listing-1", *entry.ListingID)
}

// TestImportWooProducts_importsBarcodeFromGTIN proves Woo's
// global_unique_id ("GTIN, UPC, EAN, or ISBN") lands on the new SKU's
// Barcode field on import.
func TestImportWooProducts_importsBarcodeFromGTIN(t *testing.T) {
	wp := simpleWooProduct()
	wp.GTIN = "0012345678905"
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}

	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	require.Equal(t, "0012345678905", h.skuRepo.created[0].Barcode)
}

// TestImportWooProducts_importsActiveSalePrice proves a currently-active
// Woo sale (on_sale=true, sale_price set) lands on the new SKU.
func TestImportWooProducts_importsActiveSalePrice(t *testing.T) {
	wp := simpleWooProduct()
	wp.SalePrice, wp.OnSale = "0.99", true
	wp.SaleEndDate = "2026-12-01"
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}

	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	require.NotNil(t, h.skuRepo.created[0].SalePriceMinorUnits)
	require.Equal(t, int64(99), *h.skuRepo.created[0].SalePriceMinorUnits)
	require.NotNil(t, h.skuRepo.created[0].SaleEndDate)
	require.Equal(t, "2026-12-01", h.skuRepo.created[0].SaleEndDate.Format("2006-01-02"))
}

// TestImportWooProducts_ignoresSalePriceWhenNotCurrentlyOnSale proves a
// sale_price Woo carries for a future/expired window (on_sale=false)
// doesn't get imported as an active discount.
func TestImportWooProducts_ignoresSalePriceWhenNotCurrentlyOnSale(t *testing.T) {
	wp := simpleWooProduct()
	wp.SalePrice, wp.OnSale = "0.99", false
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}

	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	require.Nil(t, h.skuRepo.created[0].SalePriceMinorUnits)
}

// TestImportWooProducts_ignoresSalePriceExceedingRegularPrice proves a
// malformed/inconsistent Woo sale_price greater than the regular price
// (shouldn't happen, but Woo doesn't guarantee it) is dropped rather than
// violating this codebase's own sale<=regular invariant.
func TestImportWooProducts_ignoresSalePriceExceedingRegularPrice(t *testing.T) {
	wp := simpleWooProduct()
	wp.RegularPrice = "1.49"
	wp.SalePrice, wp.OnSale = "1.99", true
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}

	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	require.Nil(t, h.skuRepo.created[0].SalePriceMinorUnits)
}

// TestImportWooProducts_suppressesStockChangedEventSoAsyncConsumerDoesNotEchoToWoo
// proves the initial-stock Adjustment doesn't cause an unwanted push right
// back to Woo: RecordStockMovementUseCase enqueues inventory.stock.changed
// unless told to suppress it, and InventoryStockChangedConsumer/
// SyncStockToChannelsUseCase reacts to that event by re-pushing the SKU's
// stock to every linked listing -- including the one this exact number was
// just read from. Import passes SuppressEvent so the event is never raised
// in the first place -- no event, no consumer trigger, no echo, and no
// misleading "CommerceOS -> Channel" sync-log row for a number CommerceOS
// never actually pushed anywhere.
func TestImportWooProducts_suppressesStockChangedEventSoAsyncConsumerDoesNotEchoToWoo(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}

	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.stock.inputs, 1)
	require.True(t, h.stock.inputs[0].SuppressEvent,
		"initial import stock must not raise inventory.stock.changed")
	require.Len(t, h.syncLogRepo.created, 1, "no stock-quantity sync-log row should be seeded anymore")
}

func TestImportWooProducts_recordsInitialStockAsAdjustmentEvenWhenNegative(t *testing.T) {
	wp := simpleWooProduct()
	negative := -5
	wp.StockQuantity = &negative
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}

	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Empty(t, result.Errors, "negative Woo stock must not be rejected as an invalid Receipt")
	require.Len(t, h.stock.calls, 1)
	require.Equal(t, int64(-5), h.stock.calls[0].QuantityDelta)
	require.Equal(t, invDomain.MovementTypeAdjustment, h.stock.calls[0].Type)
}

func TestImportWooProducts_createsCategoryWhenItDoesNotExistYet(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	// h.categoryRepo.bySlug deliberately left empty — the product's Woo
	// category ("salgadinhos-e-snacks") doesn't exist in CommerceOS yet.

	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Len(t, h.categoryRepo.created, 1)
	require.Equal(t, "Salgadinhos e snacks", h.categoryRepo.created[0].Name)
	require.Equal(t, "salgadinhos-e-snacks", h.categoryRepo.created[0].Slug)
	require.Len(t, h.productRepo.createdProducts, 1)
	require.Equal(t, "new-category-1", *h.productRepo.createdProducts[0].CategoryID)
}

func TestImportWooProducts_propagatesCategoryCreateFailure(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.categoryRepo.createErr = errors.New("db down")

	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err) // per-product failure, not a top-level error
	require.Len(t, result.Errors, 1)
	require.Empty(t, h.productRepo.createdProducts)
}

func TestImportWooProducts_skipsNonSimpleProductType(t *testing.T) {
	variable := simpleWooProduct()
	variable.Type = "variable"
	h := newHarness([]integrationsDomain.WooProduct{variable})

	uc := h.newImportUseCase(t, 2)

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

	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.AlreadyLinked)
	require.Empty(t, h.productRepo.createdProducts)
}

func TestImportWooProducts_linksToExistingProductBySKUCodeInsteadOfDuplicating(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.skuRepo.byCode = map[string]pmsDomain.SKU{
		"7896222211910": {ID: "existing-sku", VariantID: "existing-variant"},
	}
	h.variantRepo.byProduct = map[string][]pmsDomain.Variant{
		"existing-product": {{ID: "existing-variant", ProductID: "existing-product"}},
	}
	h.variantRepo.findByIDResult = &pmsDomain.Variant{ID: "existing-variant", ProductID: "existing-product"}
	h.productRepo.byID = map[string]pmsDomain.Product{
		"existing-product": {ID: "existing-product", Name: "Existing Product"},
	}

	uc := h.newImportUseCase(t, 3)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.AlreadyLinked)
	require.Equal(t, 0, result.Imported)
	require.Empty(t, h.productRepo.createdProducts, "must not create a duplicate product")
	require.Empty(t, h.skuRepo.created, "must not create a duplicate SKU")
	require.Empty(t, h.stock.calls, "CommerceOS already owns this SKU's stock -- Woo's number must not overwrite it")
	require.Empty(t, h.media.calls, "no new product was created, so no images to import")

	require.Len(t, h.listingRepo.created, 1)
	require.Equal(t, "existing-product", h.listingRepo.created[0].ProductID)
	require.Equal(t, "23609", h.listingRepo.created[0].ExternalID)

	require.Len(t, h.syncLogRepo.created, 1, "the link itself is still worth an audit trail entry")
	require.Equal(t, channelsDomain.SyncFlowImport, h.syncLogRepo.created[0].Direction)
}

func TestImportWooProducts_createsNewProductWhenNoSKUCodeMatch(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}
	// h.skuRepo.byCode deliberately left empty -- no existing SKU matches.

	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.AlreadyLinked)
	require.Len(t, h.productRepo.createdProducts, 1)
}

func TestImportWooProducts_neverMatchesOnEmptySKUCode(t *testing.T) {
	wp := simpleWooProduct()
	wp.SKU = ""
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.categoryRepo.bySlug["salgadinhos-e-snacks"] = pmsDomain.Category{ID: "cat-1", Slug: "salgadinhos-e-snacks"}
	// An existing SKU with an empty code must never false-match another
	// empty-code product.
	h.skuRepo.byCode = map[string]pmsDomain.SKU{"": {ID: "some-other-sku"}}

	uc := h.newImportUseCase(t, 4)

	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Equal(t, 0, result.AlreadyLinked)
}

func TestImportWooProducts_doesNotRecordStockWhenManageStockFalse(t *testing.T) {
	p := simpleWooProduct()
	p.ManageStock = false
	p.StockQuantity = nil
	h := newHarness([]integrationsDomain.WooProduct{p})

	uc := h.newImportUseCase(t, 3)

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

	uc := h.newImportUseCase(t, 5)

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

	uc := h.newImportUseCase(t, 2)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.Error(t, err)
}

func TestImportWooProducts_wrapsWooAPIFailureAsErrWooRequestFailed(t *testing.T) {
	h := newHarness(nil)
	h.wooClient.err = errors.New("500 from woocommerce")

	uc := h.newImportUseCase(t, 2)

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

	uc := h.newImportUseCase(t, 3)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	require.True(t, h.skuRepo.created[0].IsVirtual)
}

// ── Tax class reconciliation ─────────────────────────────────────────────────

func TestImportWooProducts_setsTaxClassFromWooTaxClassSlugCreatingItOnFirstSight(t *testing.T) {
	product := simpleWooProduct()
	product.TaxClass = "reduced-rate"
	product.TaxStatus = "taxable"
	h := newHarness([]integrationsDomain.WooProduct{product})
	h.wooClient.taxRates = []integrationsDomain.WooTaxRate{
		{ID: "1", Country: "NL", Rate: "21.0000", Class: "reduced-rate"},
	}

	uc := h.newImportUseCase(t, 5)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.taxClassRepo.created, 1)
	require.Equal(t, "Reduced Rate", h.taxClassRepo.created[0].Name)
	require.Len(t, h.taxRateRepo.upserted, 1)
	require.Equal(t, "NL", h.taxRateRepo.upserted[0].Country)
	require.Equal(t, int32(2100), h.taxRateRepo.upserted[0].RateBasisPoints)
	require.Len(t, h.skuRepo.created, 1)
	require.NotNil(t, h.skuRepo.created[0].TaxClassID)
	require.Equal(t, h.taxClassRepo.created[0].ID, *h.skuRepo.created[0].TaxClassID)
}

func TestImportWooProducts_reusesAnExistingExternalTaxClassLinkInsteadOfCreatingANewClass(t *testing.T) {
	product := simpleWooProduct()
	product.TaxClass = "reduced-rate"
	h := newHarness([]integrationsDomain.WooProduct{product})
	h.wooClient.taxRates = []integrationsDomain.WooTaxRate{
		{ID: "1", Country: "NL", Rate: "21.0000", Class: "reduced-rate"},
	}
	h.taxLinkRepo.links = map[string]taxDomain.ExternalTaxClassLink{
		"woocommerce|reduced-rate": {ID: "link-1", Provider: "woocommerce", ExternalSlug: "reduced-rate", TaxClassID: "existing-class-1"},
	}

	uc := h.newImportUseCase(t, 5)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Empty(t, h.taxClassRepo.created, "must not create a duplicate class when a link already exists")
	require.Len(t, h.skuRepo.created, 1)
	require.NotNil(t, h.skuRepo.created[0].TaxClassID)
	require.Equal(t, "existing-class-1", *h.skuRepo.created[0].TaxClassID)
}

func TestImportWooProducts_storesWooTaxClassAndStatusOnSKUEvenWithNoRatesConfigured(t *testing.T) {
	product := simpleWooProduct()
	product.TaxClass = "reduced-rate"
	product.TaxStatus = "taxable"
	h := newHarness([]integrationsDomain.WooProduct{product})
	// No tax rates fetched from Woo at all -- reconciliation short-circuits,
	// TaxClassID stays nil (falls back to the tenant default), but the raw
	// Woo values are still recorded for traceability.
	uc := h.newImportUseCase(t, 4)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	require.Nil(t, h.skuRepo.created[0].TaxClassID)
	require.NotNil(t, h.skuRepo.created[0].WooTaxClassSlug)
	require.Equal(t, "reduced-rate", *h.skuRepo.created[0].WooTaxClassSlug)
	require.NotNil(t, h.skuRepo.created[0].WooTaxStatus)
	require.Equal(t, "taxable", *h.skuRepo.created[0].WooTaxStatus)
}

func TestImportWooProducts_skipsStateSpecificAndCountrylessTaxRates(t *testing.T) {
	product := simpleWooProduct()
	product.TaxClass = "reduced-rate"
	h := newHarness([]integrationsDomain.WooProduct{product})
	h.wooClient.taxRates = []integrationsDomain.WooTaxRate{
		{ID: "1", Country: "US", State: "CA", Rate: "8.25", Class: "reduced-rate"}, // state-specific -- skipped
		{ID: "2", Country: "", Rate: "5.00", Class: "reduced-rate"},                // no country -- skipped
		{ID: "3", Country: "NL", Rate: "21.0000", Class: "reduced-rate"},           // country-level -- kept
	}

	uc := h.newImportUseCase(t, 5)

	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.taxRateRepo.upserted, 1)
	require.Equal(t, "NL", h.taxRateRepo.upserted[0].Country)
}

// ── Category hierarchy ───────────────────────────────────────────────────────
//
// Woo's product-embedded category only carries {id, name, slug} — no
// parent — so hierarchy comes from a separate ListCategories call
// (client.categories in these tests), keyed by external id with
// ParentExternalID set. See resolveCategoryChain's doc comment.

func TestImportWooProducts_createsParentChainPreservingHierarchy(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.wooClient.categories = []integrationsDomain.WooCategory{
		{ExternalID: "1", Name: "Food", Slug: "food"},
		{ExternalID: "167", Name: "Salgadinhos e snacks", Slug: "salgadinhos-e-snacks", ParentExternalID: "1"},
	}

	uc := h.newImportUseCase(t, 4)
	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Len(t, h.categoryRepo.created, 2, "both the parent and the leaf must be created")

	parent := h.categoryRepo.created[0]
	leaf := h.categoryRepo.created[1]
	require.Equal(t, "food", parent.Slug)
	require.Nil(t, parent.ParentID)
	require.Equal(t, "salgadinhos-e-snacks", leaf.Slug)
	require.NotNil(t, leaf.ParentID)
	require.Equal(t, parent.ID, *leaf.ParentID)

	require.Equal(t, leaf.ID, *h.productRepo.createdProducts[0].CategoryID, "product links to the leaf, not the parent")
}

func TestImportWooProducts_reusesExistingParentCategory(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.wooClient.categories = []integrationsDomain.WooCategory{
		{ExternalID: "1", Name: "Food", Slug: "food"},
		{ExternalID: "167", Name: "Salgadinhos e snacks", Slug: "salgadinhos-e-snacks", ParentExternalID: "1"},
	}
	h.categoryRepo.bySlug["food"] = pmsDomain.Category{ID: "existing-parent", Slug: "food"}

	uc := h.newImportUseCase(t, 4)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.categoryRepo.created, 1, "only the missing leaf is created, the existing parent is reused")
	require.Equal(t, "salgadinhos-e-snacks", h.categoryRepo.created[0].Slug)
	require.Equal(t, "existing-parent", *h.categoryRepo.created[0].ParentID)
}

func TestImportWooProducts_categoryMissingFromListDegradesToUncategorized(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})
	h.wooClient.categories = nil // ListCategories didn't return the product's own category

	uc := h.newImportUseCase(t, 4)
	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Empty(t, result.Errors)
	require.Nil(t, h.productRepo.createdProducts[0].CategoryID)
}

func TestImportWooProducts_propagatesListCategoriesFailureAsErrWooRequestFailed(t *testing.T) {
	h := newHarness(nil)
	h.wooClient.categoriesErr = errors.New("500 from woocommerce")

	uc := h.newImportUseCase(t, 2)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrWooRequestFailed))
}

// ── Images ───────────────────────────────────────────────────────────────────

func TestImportWooProducts_downloadsAndUploadsProductImages(t *testing.T) {
	wp := simpleWooProduct()
	wp.Images = []integrationsDomain.WooProductImage{
		{URL: "https://mystore.com/wp-content/uploads/bacon.jpg", Alt: "Bacon snack"},
		{URL: "https://mystore.com/wp-content/uploads/bacon-2.jpg", Alt: ""},
	}
	h := newHarness([]integrationsDomain.WooProduct{wp})

	uc := h.newImportUseCase(t, 4)
	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Equal(t, []string{
		"https://mystore.com/wp-content/uploads/bacon.jpg",
		"https://mystore.com/wp-content/uploads/bacon-2.jpg",
	}, h.wooClient.downloadedURLs)

	require.Len(t, h.media.calls, 2)
	productID := h.productRepo.createdProducts[0].ID
	require.Equal(t, &productID, h.media.calls[0].ProductID)
	require.Equal(t, "bacon.jpg", h.media.calls[0].Filename)
	require.Equal(t, "image/jpeg", h.media.calls[0].ContentType)
	require.Equal(t, "Bacon snack", h.media.calls[0].AltText)
}

func TestImportWooProducts_imageDownloadFailureIsNonFatalToTheProduct(t *testing.T) {
	wp := simpleWooProduct()
	wp.Images = []integrationsDomain.WooProductImage{{URL: "https://mystore.com/broken.jpg", Alt: ""}}
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.wooClient.downloadErr = errors.New("connection reset")

	uc := h.newImportUseCase(t, 4)
	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported, "the product itself still counts as imported")
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0], "connection reset")
	require.Empty(t, h.media.calls)
}

func TestImportWooProducts_imageUploadFailureIsNonFatalToTheProduct(t *testing.T) {
	wp := simpleWooProduct()
	wp.Images = []integrationsDomain.WooProductImage{{URL: "https://mystore.com/bacon.jpg", Alt: ""}}
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.media.err = errors.New("unsupported content type")

	uc := h.newImportUseCase(t, 4)
	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0], "unsupported content type")
}

func TestImportWooProducts_noImagesMeansNoUploadCalls(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()})

	uc := h.newImportUseCase(t, 4)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Empty(t, h.media.calls)
}

// ── Weight and dimensions ────────────────────────────────────────────────────
//
// WooCommerce's product endpoint returns raw numeric strings with no unit
// attached (the unit is a store-wide setting, fetched separately via
// GetProductUnits) — these tests exercise the conversion end to end through
// Execute rather than the unexported converters directly.

func TestImportWooProducts_convertsWeightAndDimensionsUsingStoreUnits(t *testing.T) {
	wp := simpleWooProduct()
	wp.Weight = "0.5"
	wp.Dimensions = integrationsDomain.WooProductDimensions{Length: "10", Width: "5", Height: "2"}
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.wooClient.weightUnit = "kg"
	h.wooClient.dimensionUnit = "cm"

	uc := h.newImportUseCase(t, 4)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, h.skuRepo.created, 1)
	sku := h.skuRepo.created[0]
	require.NotNil(t, sku.WeightGrams)
	require.Equal(t, int64(500), *sku.WeightGrams, "0.5kg must convert to 500g, not 0.5g")
	require.Equal(t, int64(100), *sku.LengthMM)
	require.Equal(t, int64(50), *sku.WidthMM)
	require.Equal(t, int64(20), *sku.HeightMM)
}

func TestImportWooProducts_convertsPoundsAndInches(t *testing.T) {
	wp := simpleWooProduct()
	wp.Weight = "1"
	wp.Dimensions = integrationsDomain.WooProductDimensions{Length: "1"}
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.wooClient.weightUnit = "lbs"
	h.wooClient.dimensionUnit = "in"

	uc := h.newImportUseCase(t, 4)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, int64(454), *h.skuRepo.created[0].WeightGrams, "1 lb ≈ 454g")
	require.Equal(t, int64(25), *h.skuRepo.created[0].LengthMM, "1 in = 25.4mm, rounded")
}

func TestImportWooProducts_leavesWeightNilWhenWooOmitsIt(t *testing.T) {
	h := newHarness([]integrationsDomain.WooProduct{simpleWooProduct()}) // no Weight/Dimensions set
	h.wooClient.weightUnit = "kg"
	h.wooClient.dimensionUnit = "cm"

	uc := h.newImportUseCase(t, 4)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Nil(t, h.skuRepo.created[0].WeightGrams)
	require.Nil(t, h.skuRepo.created[0].LengthMM)
}

func TestImportWooProducts_leavesWeightNilWhenUnitsAreUnknown(t *testing.T) {
	wp := simpleWooProduct()
	wp.Weight = "0.5"
	h := newHarness([]integrationsDomain.WooProduct{wp})
	// h.wooClient.weightUnit deliberately left "" — e.g. GetProductUnits
	// failed and Execute degraded to best-effort rather than propagating.

	uc := h.newImportUseCase(t, 4)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Nil(t, h.skuRepo.created[0].WeightGrams, "must never guess a unit")
}

func TestImportWooProducts_neverSetsWeightOrDimensionsForVirtualProducts(t *testing.T) {
	wp := simpleWooProduct()
	wp.Virtual = true
	wp.ManageStock = false
	wp.StockQuantity = nil
	wp.Weight = "0.5"
	wp.Dimensions = integrationsDomain.WooProductDimensions{Length: "10"}
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.wooClient.weightUnit = "kg"
	h.wooClient.dimensionUnit = "cm"

	uc := h.newImportUseCase(t, 3)
	_, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Nil(t, h.skuRepo.created[0].WeightGrams)
	require.Nil(t, h.skuRepo.created[0].LengthMM)
}

func TestImportWooProducts_getProductUnitsFailureDegradesGracefullyInsteadOfAbortingImport(t *testing.T) {
	wp := simpleWooProduct()
	wp.Weight = "0.5"
	h := newHarness([]integrationsDomain.WooProduct{wp})
	h.wooClient.unitsErr = errors.New("500 from woocommerce")

	uc := h.newImportUseCase(t, 4)
	result, err := uc.Execute(context.Background(), appchannels.ImportWooProductsInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	require.Nil(t, h.skuRepo.created[0].WeightGrams)
}
