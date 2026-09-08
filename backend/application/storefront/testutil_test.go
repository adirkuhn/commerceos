package storefront_test

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy n
// tenant.WithTenant Begin/set_config/Commit round trips -- same pattern as
// application/whatsapp's own testutil_test.go.
func newMockGormDB(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

// ── channels ─────────────────────────────────────────────────────────────

type fakeChannelRepo struct {
	channelsDomain.SalesChannelRepository
	byID map[string]channelsDomain.SalesChannel
}

func (f *fakeChannelRepo) FindByID(_ context.Context, id string) (*channelsDomain.SalesChannel, error) {
	if c, ok := f.byID[id]; ok {
		return &c, nil
	}
	return nil, channelsDomain.ErrSalesChannelNotFound
}

func timePtr() *time.Time {
	now := time.Now()
	return &now
}

func internalChannel(id string) channelsDomain.SalesChannel {
	return channelsDomain.SalesChannel{ID: id, Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true}
}

// ── carts ────────────────────────────────────────────────────────────────

type fakeCartRepo struct {
	storefrontDomain.CartRepository
	byToken   map[string]storefrontDomain.Cart
	createErr error
	updateErr error
	created   *storefrontDomain.Cart
	updated   *storefrontDomain.Cart
}

func (f *fakeCartRepo) Create(_ context.Context, c *storefrontDomain.Cart) error {
	if f.createErr != nil {
		return f.createErr
	}
	c.ID = "cart-1"
	f.created = c
	return nil
}

func (f *fakeCartRepo) FindByToken(_ context.Context, token string) (*storefrontDomain.Cart, error) {
	if c, ok := f.byToken[token]; ok {
		return &c, nil
	}
	return nil, nil
}

func (f *fakeCartRepo) Update(_ context.Context, c *storefrontDomain.Cart) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = c
	return nil
}

type fakeWishlistRepo struct {
	storefrontDomain.WishlistRepository
	byToken   map[string]storefrontDomain.Wishlist
	createErr error
	updateErr error
	created   *storefrontDomain.Wishlist
	updated   *storefrontDomain.Wishlist
}

func (f *fakeWishlistRepo) Create(_ context.Context, w *storefrontDomain.Wishlist) error {
	if f.createErr != nil {
		return f.createErr
	}
	w.ID = "wishlist-1"
	f.created = w
	return nil
}

func (f *fakeWishlistRepo) FindByToken(_ context.Context, token string) (*storefrontDomain.Wishlist, error) {
	if w, ok := f.byToken[token]; ok {
		return &w, nil
	}
	return nil, nil
}

func (f *fakeWishlistRepo) Update(_ context.Context, w *storefrontDomain.Wishlist) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updated = w
	return nil
}

// ── catalog ──────────────────────────────────────────────────────────────

type fakeSKURepo struct {
	pmsDomain.SKURepository
	byCode    map[string]pmsDomain.SKU
	byVariant map[string][]pmsDomain.SKU
}

func (f *fakeSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	if s, ok := f.byCode[code]; ok {
		return &s, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}

func (f *fakeSKURepo) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	return f.byVariant[variantID], nil
}

type fakeVariantRepo struct {
	pmsDomain.VariantRepository
	byID      map[string]pmsDomain.Variant
	byProduct map[string][]pmsDomain.Variant
}

func (f *fakeVariantRepo) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	if v, ok := f.byID[id]; ok {
		return &v, nil
	}
	return nil, pmsDomain.ErrVariantNotFound
}

func (f *fakeVariantRepo) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	return f.byProduct[productID], nil
}

type fakeProductRepo struct {
	pmsDomain.ProductRepository
	byID map[string]pmsDomain.Product
}

func (f *fakeProductRepo) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	if p, ok := f.byID[id]; ok {
		return &p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}

type fakeOverrideRepo struct {
	channelsDomain.SalesChannelProductOverrideRepository
	byChannel map[string][]channelsDomain.SalesChannelProductOverride
}

func (f *fakeOverrideRepo) FindByChannelAndProduct(_ context.Context, _, _ string) (*channelsDomain.SalesChannelProductOverride, error) {
	return nil, nil
}

func (f *fakeOverrideRepo) ListByChannel(_ context.Context, channelID string) ([]channelsDomain.SalesChannelProductOverride, error) {
	return f.byChannel[channelID], nil
}

type fakeStockLevel struct {
	byID map[string]int64
	err  error
}

func (f *fakeStockLevel) Execute(_ context.Context, skuID string) (*appinventory.StockLevel, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &appinventory.StockLevel{SKUID: skuID, Syncable: f.byID[skuID]}, nil
}

// seedCatalogOneSKU wires the four catalog fakes so SKU code "SKU-A"
// resolves to one enrolled product with the given price/stock.
func seedCatalogOneSKU(skuCode, skuID string, priceMinorUnits, stock int64) (*fakeSKURepo, *fakeVariantRepo, *fakeProductRepo, *fakeOverrideRepo, *fakeStockLevel) {
	sku := &fakeSKURepo{byCode: map[string]pmsDomain.SKU{
		skuCode: {ID: skuID, VariantID: "variant-1", SKUCode: skuCode, PriceMinorUnits: priceMinorUnits, CurrencyCode: "EUR"},
	}}
	variant := &fakeVariantRepo{byID: map[string]pmsDomain.Variant{"variant-1": {ID: "variant-1", ProductID: "product-1"}}}
	product := &fakeProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1", Name: "Widget"}}}
	override := &fakeOverrideRepo{}
	stockLevel := &fakeStockLevel{byID: map[string]int64{skuID: stock}}
	return sku, variant, product, override, stockLevel
}

type fakeMediaRepo struct {
	pmsDomain.MediaRepository
	byProduct map[string][]pmsDomain.Media
}

func (f *fakeMediaRepo) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Media, error) {
	return f.byProduct[productID], nil
}

type fakeCategoryRepo struct {
	pmsDomain.CategoryRepository
	out []pmsDomain.Category
	err error
}

func (f *fakeCategoryRepo) List(_ context.Context) ([]pmsDomain.Category, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

type fakeListPOSProductCatalog struct {
	out    *apppms.ListPOSProductCatalogResult
	err    error
	gotIn  apppms.ListProductsInput
	gotCtx context.Context
}

func (f *fakeListPOSProductCatalog) Execute(ctx context.Context, in apppms.ListProductsInput) (*apppms.ListPOSProductCatalogResult, error) {
	f.gotIn, f.gotCtx = in, ctx
	if f.err != nil {
		return nil, f.err
	}
	if f.out != nil {
		return f.out, nil
	}
	return &apppms.ListPOSProductCatalogResult{}, nil
}

type fakeSalesOrderRepo struct {
	omsDomain.SalesOrderRepository
	byNumber map[string]omsDomain.SalesOrder
}

func (f *fakeSalesOrderRepo) FindByNumber(_ context.Context, number string) (*omsDomain.SalesOrder, error) {
	if o, ok := f.byNumber[number]; ok {
		return &o, nil
	}
	return nil, nil
}

// ── customers ────────────────────────────────────────────────────────────

type fakeCustomerRepo struct {
	customersDomain.CustomerRepository
	byEmail map[string]customersDomain.Customer
}

func (f *fakeCustomerRepo) FindByEmail(_ context.Context, email string) (*customersDomain.Customer, error) {
	if c, ok := f.byEmail[email]; ok {
		return &c, nil
	}
	return nil, nil
}

// ── tax ──────────────────────────────────────────────────────────────────

type fakeTaxClassRepo struct {
	taxDomain.TaxClassRepository
	out []taxDomain.TaxClass
}

func (f *fakeTaxClassRepo) List(_ context.Context) ([]taxDomain.TaxClass, error) { return f.out, nil }

type fakeTaxRateRepo struct {
	taxDomain.TaxRateRepository
	out []taxDomain.TaxRate
}

func (f *fakeTaxRateRepo) List(_ context.Context) ([]taxDomain.TaxRate, error) { return f.out, nil }

// ── outbox ───────────────────────────────────────────────────────────────

type fakeOutboxRepo struct {
	outbox.Repository
	subjects []string
	err      error
}

func (f *fakeOutboxRepo) Enqueue(_ context.Context, subject, _ string, _ []byte, _ time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.subjects = append(f.subjects, subject)
	return nil
}
