package channels_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

func strPtr(s string) *string { return &s }

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// however many tenant.WithTenant Begin/set_config/Commit sequences the test
// needs — same pattern as application/pms and application/inventory's
// use-case tests. n is the number of separate tenant.WithTenant calls the
// use case under test will make.
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

// ── integrations stubs ────────────────────────────────────────────────────────

type stubConnectionRepo struct {
	conn        *integrationsDomain.Connection
	findErr     error
	allByTenant []integrationsDomain.Connection
	listErr     error
}

func (s *stubConnectionRepo) Create(_ context.Context, _ *integrationsDomain.Connection) error {
	return nil
}
func (s *stubConnectionRepo) FindByID(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
	return s.conn, s.findErr
}
func (s *stubConnectionRepo) ListByTenant(_ context.Context) ([]integrationsDomain.Connection, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.allByTenant, nil
}
func (s *stubConnectionRepo) Update(_ context.Context, _ *integrationsDomain.Connection) error {
	return nil
}
func (s *stubConnectionRepo) Delete(_ context.Context, _ string) error { return nil }

type stubWooClient struct {
	products      []integrationsDomain.WooProduct
	err           error
	createCalled  []integrationsDomain.WooProduct
	createErr     error
	createdWithID string // external ID assigned to the pushed product

	updateStockCalled []stubStockUpdate
	updateStockErr    error

	updateProductCalled []stubUpdateProductCall
	updateProductErr    error

	categories    []integrationsDomain.WooCategory
	categoriesErr error

	weightUnit    string
	dimensionUnit string
	unitsErr      error

	downloadedURLs []string
	downloadErr    error
	downloadImage  []byte // defaults to a small placeholder if nil
	downloadType   string // defaults to "image/jpeg" if ""

	deleteProductCalled []string
	deleteProductErr    error

	// ordersByPage maps page number -> orders returned for that page,
	// nil/missing means empty (end of pagination). ordersModifiedAfter
	// captures the last ListOrders call's filter, for assertions.
	ordersByPage map[int][]integrationsDomain.WooOrder
	ordersErr    error
	// ordersErrOnPage, when nonzero, limits ordersErr to that one page --
	// every other page behaves normally. Zero (the default) means ordersErr
	// applies to every page, same as before this field existed.
	ordersErrOnPage     int
	ordersModifiedAfter *time.Time
	ordersCalls         int

	getOrderCalled []string
	getOrderErr    error
	getOrderResult *integrationsDomain.WooOrder

	// customersByPage mirrors ordersByPage's convention.
	customersByPage map[int][]integrationsDomain.WooCustomer
	customersErr    error
	customersCalls  int

	createVariableCalled []stubCreateVariableCall
	createVariableErr    error
	createVariableWithID string

	taxRates    []integrationsDomain.WooTaxRate
	taxRatesErr error

	createVariationCalled []stubVariationCall
	createVariationErr    error
	createVariationWithID string

	updateVariationCalled []stubVariationCall
	updateVariationErr    error

	deleteVariationCalled []stubDeleteVariationCall
	deleteVariationErr    error
}

type stubCreateVariableCall struct {
	Product         integrationsDomain.WooProduct
	AttributeName   string
	AttributeValues []string
}

type stubVariationCall struct {
	ParentExternalID    string
	VariationExternalID string // empty for a Create call
	Variation           integrationsDomain.WooVariation
}

type stubDeleteVariationCall struct {
	ParentExternalID    string
	VariationExternalID string
}

type stubStockUpdate struct {
	ExternalID string
	Quantity   int
	MetaData   []integrationsDomain.WooMetaData
}

type stubUpdateProductCall struct {
	ExternalID string
	Product    integrationsDomain.WooProduct
}

func (s *stubWooClient) TestConnection(_ context.Context) error { return nil }
func (s *stubWooClient) ListProducts(_ context.Context, page, _ int) ([]integrationsDomain.WooProduct, error) {
	if s.err != nil {
		return nil, s.err
	}
	if page > 1 {
		return nil, nil // single page of results in tests
	}
	return s.products, nil
}
func (s *stubWooClient) CreateProduct(_ context.Context, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	s.createCalled = append(s.createCalled, p)
	if s.createErr != nil {
		return nil, s.createErr
	}
	id := s.createdWithID
	if id == "" {
		id = "new-external-id"
	}
	p.ID = id
	return &p, nil
}

func (s *stubWooClient) UpdateProductStock(_ context.Context, externalID string, quantity int, metadata []integrationsDomain.WooMetaData) error {
	s.updateStockCalled = append(s.updateStockCalled, stubStockUpdate{ExternalID: externalID, Quantity: quantity, MetaData: metadata})
	return s.updateStockErr
}

func (s *stubWooClient) UpdateProduct(_ context.Context, externalID string, p integrationsDomain.WooProduct) (*integrationsDomain.WooProduct, error) {
	s.updateProductCalled = append(s.updateProductCalled, stubUpdateProductCall{ExternalID: externalID, Product: p})
	if s.updateProductErr != nil {
		return nil, s.updateProductErr
	}
	p.ID = externalID
	return &p, nil
}

func (s *stubWooClient) DeleteProduct(_ context.Context, externalID string) error {
	s.deleteProductCalled = append(s.deleteProductCalled, externalID)
	return s.deleteProductErr
}

func (s *stubWooClient) CreateVariableProduct(_ context.Context, p integrationsDomain.WooProduct, attributeName string, attributeValues []string) (*integrationsDomain.WooProduct, error) {
	s.createVariableCalled = append(s.createVariableCalled, stubCreateVariableCall{Product: p, AttributeName: attributeName, AttributeValues: attributeValues})
	if s.createVariableErr != nil {
		return nil, s.createVariableErr
	}
	id := s.createVariableWithID
	if id == "" {
		id = "new-parent-external-id"
	}
	p.ID = id
	p.Type = "variable"
	return &p, nil
}

func (s *stubWooClient) CreateVariation(_ context.Context, parentExternalID string, v integrationsDomain.WooVariation) (*integrationsDomain.WooVariation, error) {
	s.createVariationCalled = append(s.createVariationCalled, stubVariationCall{ParentExternalID: parentExternalID, Variation: v})
	if s.createVariationErr != nil {
		return nil, s.createVariationErr
	}
	id := s.createVariationWithID
	if id == "" {
		id = "new-variation-external-id"
	}
	v.ID = id
	return &v, nil
}

func (s *stubWooClient) UpdateVariation(_ context.Context, parentExternalID, variationExternalID string, v integrationsDomain.WooVariation) (*integrationsDomain.WooVariation, error) {
	s.updateVariationCalled = append(s.updateVariationCalled, stubVariationCall{ParentExternalID: parentExternalID, VariationExternalID: variationExternalID, Variation: v})
	if s.updateVariationErr != nil {
		return nil, s.updateVariationErr
	}
	v.ID = variationExternalID
	return &v, nil
}

func (s *stubWooClient) DeleteVariation(_ context.Context, parentExternalID, variationExternalID string) error {
	s.deleteVariationCalled = append(s.deleteVariationCalled, stubDeleteVariationCall{ParentExternalID: parentExternalID, VariationExternalID: variationExternalID})
	return s.deleteVariationErr
}

func (s *stubWooClient) ListCategories(_ context.Context) ([]integrationsDomain.WooCategory, error) {
	return s.categories, s.categoriesErr
}

func (s *stubWooClient) GetProductUnits(_ context.Context) (string, string, error) {
	return s.weightUnit, s.dimensionUnit, s.unitsErr
}

func (s *stubWooClient) DownloadImage(_ context.Context, url string) (io.ReadCloser, string, error) {
	s.downloadedURLs = append(s.downloadedURLs, url)
	if s.downloadErr != nil {
		return nil, "", s.downloadErr
	}
	data := s.downloadImage
	if data == nil {
		data = []byte("fake-image-bytes")
	}
	contentType := s.downloadType
	if contentType == "" {
		contentType = "image/jpeg"
	}
	return io.NopCloser(bytes.NewReader(data)), contentType, nil
}

func (s *stubWooClient) ListOrders(_ context.Context, page, _ int, modifiedAfter *time.Time) ([]integrationsDomain.WooOrder, error) {
	s.ordersCalls++
	s.ordersModifiedAfter = modifiedAfter
	if s.ordersErr != nil && (s.ordersErrOnPage == 0 || s.ordersErrOnPage == page) {
		return nil, s.ordersErr
	}
	return s.ordersByPage[page], nil
}

func (s *stubWooClient) GetOrder(_ context.Context, externalOrderID string) (*integrationsDomain.WooOrder, error) {
	s.getOrderCalled = append(s.getOrderCalled, externalOrderID)
	if s.getOrderErr != nil {
		return nil, s.getOrderErr
	}
	if s.getOrderResult != nil {
		return s.getOrderResult, nil
	}
	return nil, errors.New("stubWooClient: getOrderResult not set")
}

func (s *stubWooClient) ListCustomers(_ context.Context, page, _ int) ([]integrationsDomain.WooCustomer, error) {
	s.customersCalls++
	if s.customersErr != nil {
		return nil, s.customersErr
	}
	return s.customersByPage[page], nil
}

func (s *stubWooClient) ListTaxRates(_ context.Context) ([]integrationsDomain.WooTaxRate, error) {
	if s.taxRatesErr != nil {
		return nil, s.taxRatesErr
	}
	return s.taxRates, nil
}

// stubMediaUploader stands in for application/pms's UploadMediaUseCase.
type stubMediaUploader struct {
	calls []apppms.UploadMediaInput
	err   error
}

func (s *stubMediaUploader) Execute(_ context.Context, in apppms.UploadMediaInput) (*pmsDomain.Media, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	return &pmsDomain.Media{ID: "media-1"}, nil
}

// stubMediaRepo stands in for pmsDomain.MediaRepository's ListByProduct --
// the read side PushProductToWooUseCase/SyncProductToChannelsUseCase use to
// fetch a product's image gallery, as opposed to stubMediaUploader's write
// side.
type stubMediaRepo struct {
	byProduct map[string][]pmsDomain.Media
	err       error
}

func (s *stubMediaRepo) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Media, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.byProduct[productID], nil
}

type stubWooClientFactory struct {
	client integrationsDomain.WooClient
	err    error
}

func (s *stubWooClientFactory) NewClient(_, _, _ string) (integrationsDomain.WooClient, error) {
	return s.client, s.err
}

// ── pms stubs ────────────────────────────────────────────────────────────────

type stubCategoryRepo struct {
	bySlug    map[string]pmsDomain.Category
	created   []pmsDomain.Category
	createErr error
	deleted   []string
	deleteErr error
}

func (s *stubCategoryRepo) Create(_ context.Context, c *pmsDomain.Category) error {
	if s.createErr != nil {
		return s.createErr
	}
	// Unique per call (not a fixed "new-category-1") so tests that create
	// more than one category in a run (e.g. a Woo parent+child pair) can
	// assert on each one's own id and ParentID linkage.
	c.ID = fmt.Sprintf("new-category-%d", len(s.created)+1)
	s.created = append(s.created, *c)
	return nil
}
func (s *stubCategoryRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Category, error) {
	return nil, pmsDomain.ErrCategoryNotFound
}
func (s *stubCategoryRepo) FindBySlug(_ context.Context, slug string) (*pmsDomain.Category, error) {
	if c, ok := s.bySlug[slug]; ok {
		return &c, nil
	}
	return nil, pmsDomain.ErrCategoryNotFound
}
func (s *stubCategoryRepo) List(_ context.Context) ([]pmsDomain.Category, error)  { return nil, nil }
func (s *stubCategoryRepo) Update(_ context.Context, _ *pmsDomain.Category) error { return nil }
func (s *stubCategoryRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubChannelsProductRepo struct {
	createdProducts []pmsDomain.Product
	createErr       error
	bySlug          map[string]pmsDomain.Product
	byID            map[string]pmsDomain.Product
	listResult      []pmsDomain.Product
	listErr         error
	// countByCategory, when set, makes List(filter.CategoryID) return this
	// count instead of len(listResult) — lets tests give different
	// categories different remaining-product counts.
	countByCategory map[string]int64
	deleted         []string
	deleteErr       error
}

func (s *stubChannelsProductRepo) Create(_ context.Context, p *pmsDomain.Product) error {
	if s.createErr != nil {
		return s.createErr
	}
	p.ID = "product-1"
	s.createdProducts = append(s.createdProducts, *p)
	return nil
}
func (s *stubChannelsProductRepo) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	if p, ok := s.byID[id]; ok {
		return &p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}
func (s *stubChannelsProductRepo) FindBySlug(_ context.Context, slug string) (*pmsDomain.Product, error) {
	if p, ok := s.bySlug[slug]; ok {
		return &p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}
func (s *stubChannelsProductRepo) List(_ context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	if filter.CategoryID != nil && s.countByCategory != nil {
		return nil, s.countByCategory[*filter.CategoryID], nil
	}
	return s.listResult, int64(len(s.listResult)), nil
}
func (s *stubChannelsProductRepo) Update(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubChannelsProductRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubChannelsProductRepo) SearchByAnyNameToken(_ context.Context, _ []string, _ int) ([]pmsDomain.Product, error) {
	return nil, nil
}

type stubChannelsVariantRepo struct {
	created        []pmsDomain.Variant
	createErr      error
	byProduct      map[string][]pmsDomain.Variant
	findByIDResult *pmsDomain.Variant
	findByIDErr    error
}

func (s *stubChannelsVariantRepo) Create(_ context.Context, v *pmsDomain.Variant) error {
	if s.createErr != nil {
		return s.createErr
	}
	v.ID = "variant-1"
	s.created = append(s.created, *v)
	return nil
}
func (s *stubChannelsVariantRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Variant, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return nil, pmsDomain.ErrVariantNotFound
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubChannelsVariantRepo) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	return s.byProduct[productID], nil
}
func (s *stubChannelsVariantRepo) Update(_ context.Context, _ *pmsDomain.Variant) error { return nil }
func (s *stubChannelsVariantRepo) Delete(_ context.Context, _ string) error             { return nil }

type stubChannelsSKURepo struct {
	created        []pmsDomain.SKU
	createErr      error
	byVariant      map[string][]pmsDomain.SKU
	byID           map[string]pmsDomain.SKU
	findByIDResult *pmsDomain.SKU
	findByIDErr    error
	byCode         map[string]pmsDomain.SKU
	updated        []pmsDomain.SKU
	updateErr      error
}

func (s *stubChannelsSKURepo) Create(_ context.Context, sku *pmsDomain.SKU) error {
	if s.createErr != nil {
		return s.createErr
	}
	sku.ID = "sku-1"
	s.created = append(s.created, *sku)
	return nil
}
func (s *stubChannelsSKURepo) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	if sku, ok := s.byID[id]; ok {
		return &sku, nil
	}
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return nil, pmsDomain.ErrSKUNotFound
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubChannelsSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	if sku, ok := s.byCode[code]; ok {
		return &sku, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubChannelsSKURepo) FindByBarcode(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubChannelsSKURepo) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	return s.byVariant[variantID], nil
}
func (s *stubChannelsSKURepo) Update(_ context.Context, sku *pmsDomain.SKU) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *sku)
	return nil
}
func (s *stubChannelsSKURepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubChannelsSKURepo) ListWithExpiredSale(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

func (s *stubChannelsSKURepo) ExistsWithTaxClass(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (s *stubChannelsSKURepo) ListActive(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

func (s *stubChannelsSKURepo) UpdateCostPrice(_ context.Context, _ string, _ int64) error {
	return nil
}

// ── channels stub ────────────────────────────────────────────────────────────

type stubListingRepo struct {
	byExternalID  map[string]channelsDomain.ProductChannelListing
	byProduct     map[string]channelsDomain.ProductChannelListing
	byConnection  []channelsDomain.ProductChannelListing
	listByProduct map[string][]channelsDomain.ProductChannelListing
	created       []channelsDomain.ProductChannelListing
	updated       []channelsDomain.ProductChannelListing
	deleted       []string
	findErr       error
	listByProdErr error
}

func (s *stubListingRepo) Create(_ context.Context, l *channelsDomain.ProductChannelListing) error {
	l.ID = "listing-1"
	s.created = append(s.created, *l)
	return nil
}
func (s *stubListingRepo) FindByExternalID(_ context.Context, _, externalID string) (*channelsDomain.ProductChannelListing, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if l, ok := s.byExternalID[externalID]; ok {
		return &l, nil
	}
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubListingRepo) FindByProductAndConnection(_ context.Context, productID, _ string) (*channelsDomain.ProductChannelListing, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if l, ok := s.byProduct[productID]; ok {
		return &l, nil
	}
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubListingRepo) ListByConnection(_ context.Context, _ string) ([]channelsDomain.ProductChannelListing, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return s.byConnection, nil
}
func (s *stubListingRepo) ListByProduct(_ context.Context, productID string) ([]channelsDomain.ProductChannelListing, error) {
	if s.listByProdErr != nil {
		return nil, s.listByProdErr
	}
	return s.listByProduct[productID], nil
}
func (s *stubListingRepo) ListByProductAndConnection(_ context.Context, productID, connectionID string) ([]channelsDomain.ProductChannelListing, error) {
	if s.listByProdErr != nil {
		return nil, s.listByProdErr
	}
	var out []channelsDomain.ProductChannelListing
	for _, l := range s.listByProduct[productID] {
		if l.ConnectionID == connectionID {
			out = append(out, l)
		}
	}
	return out, nil
}
func (s *stubListingRepo) FindByProductVariantAndConnection(_ context.Context, productID string, variantID *string, connectionID string) (*channelsDomain.ProductChannelListing, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	for _, l := range s.listByProduct[productID] {
		if l.ConnectionID != connectionID || !sameVariantID(l.VariantID, variantID) {
			continue
		}
		found := l
		return &found, nil
	}
	return nil, channelsDomain.ErrListingNotFound
}
func (s *stubListingRepo) Update(_ context.Context, l *channelsDomain.ProductChannelListing) error {
	s.updated = append(s.updated, *l)
	return nil
}
func (s *stubListingRepo) Delete(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}

func sameVariantID(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

type stubSyncLogRepo struct {
	created          []channelsDomain.SyncLogEntry
	createErr        error
	byListing        map[string][]channelsDomain.SyncLogEntry // newest-first
	all              []channelsDomain.SyncLogEntry            // newest-first, for ListAll
	listErr          error
	byID             map[string]channelsDomain.SyncLogEntry
	findErr          error
	needingAttention []channelsDomain.SyncLogEntry // newest-first, for ListNeedingAttention
	needingAttnErr   error
	dismissedIDs     []string
	dismissErr       error
}

func (s *stubSyncLogRepo) Create(_ context.Context, e *channelsDomain.SyncLogEntry) error {
	if s.createErr != nil {
		return s.createErr
	}
	e.ID = fmt.Sprintf("sync-log-%d", len(s.created)+1)
	s.created = append(s.created, *e)
	if s.byListing == nil {
		s.byListing = map[string][]channelsDomain.SyncLogEntry{}
	}
	if e.ListingID != nil {
		s.byListing[*e.ListingID] = append([]channelsDomain.SyncLogEntry{*e}, s.byListing[*e.ListingID]...)
	}
	return nil
}
func (s *stubSyncLogRepo) FindByID(_ context.Context, id string) (*channelsDomain.SyncLogEntry, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if e, ok := s.byID[id]; ok {
		return &e, nil
	}
	return nil, channelsDomain.ErrSyncLogNotFound
}
func (s *stubSyncLogRepo) ListNeedingAttention(_ context.Context, page, perPage int) ([]channelsDomain.SyncLogEntry, int64, error) {
	if s.needingAttnErr != nil {
		return nil, 0, s.needingAttnErr
	}
	total := int64(len(s.needingAttention))
	start := (page - 1) * perPage
	if start >= len(s.needingAttention) {
		return nil, total, nil
	}
	end := start + perPage
	if end > len(s.needingAttention) {
		end = len(s.needingAttention)
	}
	return s.needingAttention[start:end], total, nil
}
func (s *stubSyncLogRepo) Dismiss(_ context.Context, id string) error {
	if s.dismissErr != nil {
		return s.dismissErr
	}
	s.dismissedIDs = append(s.dismissedIDs, id)
	return nil
}
func (s *stubSyncLogRepo) ListByListing(_ context.Context, listingID string, limit int) ([]channelsDomain.SyncLogEntry, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	entries := s.byListing[listingID]
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}
func (s *stubSyncLogRepo) ListAll(_ context.Context, page, perPage int) ([]channelsDomain.SyncLogEntry, int64, error) {
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	if s.all == nil {
		return nil, 0, nil
	}
	total := int64(len(s.all))
	start := (page - 1) * perPage
	if start >= len(s.all) {
		return nil, total, nil
	}
	end := start + perPage
	if end > len(s.all) {
		end = len(s.all)
	}
	return s.all[start:end], total, nil
}

type stubSyncScheduleRepo struct {
	upserted     []channelsDomain.SyncSchedule
	upsertErr    error
	byConnection []channelsDomain.SyncSchedule
	all          []channelsDomain.SyncSchedule
	allErr       error
	due          []channelsDomain.SyncSchedule
	recorded     []stubRunResult
	triggeredIDs []string
	triggerErr   error
}

type stubRunResult struct {
	ID               string
	RanAt            time.Time
	Status           string
	ErrMsg           string
	NextRunAt        time.Time
	CursorAt         time.Time
	AdvanceLastRunAt bool
}

func (s *stubSyncScheduleRepo) Upsert(_ context.Context, sc *channelsDomain.SyncSchedule) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	if sc.ID == "" {
		sc.ID = "schedule-1"
	}
	s.upserted = append(s.upserted, *sc)
	return nil
}
func (s *stubSyncScheduleRepo) ListByConnection(_ context.Context, _ string) ([]channelsDomain.SyncSchedule, error) {
	return s.byConnection, nil
}
func (s *stubSyncScheduleRepo) ListAll(_ context.Context) ([]channelsDomain.SyncSchedule, error) {
	if s.allErr != nil {
		return nil, s.allErr
	}
	return s.all, nil
}
func (s *stubSyncScheduleRepo) TriggerNow(_ context.Context, id string) error {
	if s.triggerErr != nil {
		return s.triggerErr
	}
	s.triggeredIDs = append(s.triggeredIDs, id)
	return nil
}
func (s *stubSyncScheduleRepo) ListDue(_ context.Context, _ time.Time) ([]channelsDomain.SyncSchedule, error) {
	return s.due, nil
}
func (s *stubSyncScheduleRepo) RecordRunResult(_ context.Context, id string, ranAt time.Time, status, errMsg string, nextRunAt time.Time, cursorAt time.Time, advanceLastRunAt bool) error {
	s.recorded = append(s.recorded, stubRunResult{ID: id, RanAt: ranAt, Status: status, ErrMsg: errMsg, NextRunAt: nextRunAt, CursorAt: cursorAt, AdvanceLastRunAt: advanceLastRunAt})
	return nil
}

// ── import job stub ──────────────────────────────────────────────────────────

type stubImportJobRepo struct {
	hasActive    bool
	hasActiveErr error
	createErr    error
	created      []channelsDomain.ImportJob
}

func (s *stubImportJobRepo) Create(_ context.Context, j *channelsDomain.ImportJob) error {
	if s.createErr != nil {
		return s.createErr
	}
	j.ID = "job-1"
	s.created = append(s.created, *j)
	return nil
}
func (s *stubImportJobRepo) HasActiveForConnection(_ context.Context, _ string) (bool, error) {
	return s.hasActive, s.hasActiveErr
}
func (s *stubImportJobRepo) ListPending(_ context.Context) ([]channelsDomain.ImportJob, error) {
	return nil, nil
}
func (s *stubImportJobRepo) MarkRunning(_ context.Context, _ string, _ time.Time) error { return nil }
func (s *stubImportJobRepo) MarkCompleted(_ context.Context, _ string, _, _, _ int, _ []string, _ time.Time) error {
	return nil
}
func (s *stubImportJobRepo) MarkFailed(_ context.Context, _ string, _ string, _ time.Time) error {
	return nil
}
func (s *stubImportJobRepo) RecoverOrphanedRunning(_ context.Context, _ int, _ string, _ time.Time) (int64, int64, error) {
	return 0, 0, nil
}

// ── sales channel stubs ──────────────────────────────────────────────────────

type stubSalesChannelRepo struct {
	created    []channelsDomain.SalesChannel
	createErr  error
	findResult *channelsDomain.SalesChannel
	findErr    error
	list       []channelsDomain.SalesChannel
	listErr    error
	updated    []channelsDomain.SalesChannel
	updateErr  error
	deleted    []string
	deleteErr  error
}

func (s *stubSalesChannelRepo) Create(_ context.Context, c *channelsDomain.SalesChannel) error {
	if s.createErr != nil {
		return s.createErr
	}
	c.ID = "channel-1"
	s.created = append(s.created, *c)
	return nil
}
func (s *stubSalesChannelRepo) FindByID(_ context.Context, _ string) (*channelsDomain.SalesChannel, error) {
	return s.findResult, s.findErr
}
func (s *stubSalesChannelRepo) ListByTenant(_ context.Context) ([]channelsDomain.SalesChannel, error) {
	return s.list, s.listErr
}
func (s *stubSalesChannelRepo) Update(_ context.Context, c *channelsDomain.SalesChannel) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *c)
	return nil
}
func (s *stubSalesChannelRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubSalesChannelProductOverrideRepo struct {
	upserted   []channelsDomain.SalesChannelProductOverride
	upsertErr  error
	findResult *channelsDomain.SalesChannelProductOverride
	findErr    error
	byChannel  []channelsDomain.SalesChannelProductOverride
	listErr    error
	deleted    []string
	deleteErr  error
}

func (s *stubSalesChannelProductOverrideRepo) Upsert(_ context.Context, o *channelsDomain.SalesChannelProductOverride) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	o.ID = "override-1"
	s.upserted = append(s.upserted, *o)
	return nil
}
func (s *stubSalesChannelProductOverrideRepo) FindByChannelAndProduct(_ context.Context, _, _ string) (*channelsDomain.SalesChannelProductOverride, error) {
	return s.findResult, s.findErr
}
func (s *stubSalesChannelProductOverrideRepo) ListByChannel(_ context.Context, _ string) ([]channelsDomain.SalesChannelProductOverride, error) {
	return s.byChannel, s.listErr
}
func (s *stubSalesChannelProductOverrideRepo) Delete(_ context.Context, channelID, _ string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, channelID)
	return nil
}

type stubNotifier struct {
	calls []appnotifications.CreateNotificationInput
	err   error
}

func (s *stubNotifier) Execute(_ context.Context, in appnotifications.CreateNotificationInput) (*notificationsDomain.Notification, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	return &notificationsDomain.Notification{ID: "notif-1"}, nil
}

// ── inventory stub ───────────────────────────────────────────────────────────

type stubStockRecorder struct {
	calls  []invDomain.StockMovement
	inputs []invapp.RecordStockMovementInput
	err    error
}

func (s *stubStockRecorder) Execute(_ context.Context, in invapp.RecordStockMovementInput) (*invDomain.StockMovement, error) {
	s.inputs = append(s.inputs, in)
	if s.err != nil {
		return nil, s.err
	}
	m := invDomain.StockMovement{
		ID: "movement-1", SKUID: in.SKUID, WarehouseID: in.WarehouseID, BatchID: in.BatchID,
		Type: in.Type, QuantityDelta: in.QuantityDelta, Reason: in.Reason, OccurredAt: in.OccurredAt,
	}
	s.calls = append(s.calls, m)
	return &m, nil
}

// stubSaleStockAllocator backs saleStockAllocator (import_woo_orders.go) --
// distinct from stubStockRecorder above since AllocateSaleStockUseCase's
// Execute has a different shape (Quantity is positive, returns a slice).
type stubSaleStockAllocator struct {
	calls    []invapp.AllocateSaleStockInput
	err      error
	errBySKU map[string]error
}

func (s *stubSaleStockAllocator) Execute(_ context.Context, in invapp.AllocateSaleStockInput) ([]invDomain.StockMovement, error) {
	s.calls = append(s.calls, in)
	if err, ok := s.errBySKU[in.SKUID]; ok {
		return nil, err
	}
	if s.err != nil {
		return nil, s.err
	}
	return []invDomain.StockMovement{{
		ID: "movement-1", SKUID: in.SKUID, WarehouseID: in.WarehouseID,
		Type: in.Type, QuantityDelta: -in.Quantity, Reason: in.Reason, OccurredAt: in.OccurredAt,
	}}, nil
}

// ── oms stubs (import_woo_orders_test.go) ───────────────────────────────────

type stubOrdersSalesOrderRepo struct {
	created             []omsDomain.SalesOrder
	createErr           error
	byExternalID        map[string]omsDomain.SalesOrder
	findByExternalIDErr error
	byConnection        []omsDomain.SalesOrder
	listByConnectionErr error
	updateCalls         []struct {
		ID     string
		Fields map[string]any
	}
	updateErr error
	// createAttempts/winnerAfterCreateFail simulate the create-race: a
	// caller who lost the DB unique-constraint race sees Create fail with
	// createErr (typically ErrDuplicateExternalOrder), then re-queries
	// FindByExternalID and must find the winner's order now -- even though
	// the *first* FindByExternalID call (importOne's initial existence
	// check, before Create was ever attempted) correctly saw nothing.
	createAttempts        int
	winnerAfterCreateFail map[string]omsDomain.SalesOrder
}

func (s *stubOrdersSalesOrderRepo) Create(_ context.Context, o *omsDomain.SalesOrder) error {
	s.createAttempts++
	if s.createErr != nil {
		return s.createErr
	}
	o.ID = fmt.Sprintf("sales-order-%d", len(s.created)+1)
	s.created = append(s.created, *o)
	if s.byExternalID == nil {
		s.byExternalID = map[string]omsDomain.SalesOrder{}
	}
	s.byExternalID[o.ExternalOrderID] = *o
	return nil
}
func (s *stubOrdersSalesOrderRepo) FindByID(_ context.Context, _ string) (*omsDomain.SalesOrder, error) {
	return nil, omsDomain.ErrSalesOrderNotFound
}
func (s *stubOrdersSalesOrderRepo) FindByExternalID(_ context.Context, _ omsDomain.SalesOrderSource, externalOrderID string) (*omsDomain.SalesOrder, error) {
	if s.findByExternalIDErr != nil {
		return nil, s.findByExternalIDErr
	}
	if o, ok := s.byExternalID[externalOrderID]; ok {
		return &o, nil
	}
	if s.createAttempts > 0 {
		if o, ok := s.winnerAfterCreateFail[externalOrderID]; ok {
			return &o, nil
		}
	}
	return nil, nil
}
func (s *stubOrdersSalesOrderRepo) List(_ context.Context, _ omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	return nil, 0, nil
}
func (s *stubOrdersSalesOrderRepo) ListByConnection(_ context.Context, _ string) ([]omsDomain.SalesOrder, error) {
	return s.byConnection, s.listByConnectionErr
}
func (s *stubOrdersSalesOrderRepo) Update(_ context.Context, id string, fields map[string]any) error {
	s.updateCalls = append(s.updateCalls, struct {
		ID     string
		Fields map[string]any
	}{id, fields})
	return s.updateErr
}
func (s *stubOrdersSalesOrderRepo) TrySetStockDecremented(_ context.Context, _ string, _, _ bool) (bool, error) {
	return true, nil
}
func (s *stubOrdersSalesOrderRepo) Delete(_ context.Context, _ string) error { return nil }

// stubDeleteSalesOrderUC stands in for application/oms's
// DeleteSalesOrderUseCase (delete_all_imported_products_test.go) --
// errsByID lets a specific order fail (e.g. to prove one bad order doesn't
// abort the rest of the batch) while every other call succeeds.
type stubDeleteSalesOrderUC struct {
	calls    []string
	err      error
	errsByID map[string]error
}

func (s *stubDeleteSalesOrderUC) Execute(_ context.Context, id string) error {
	s.calls = append(s.calls, id)
	if err, ok := s.errsByID[id]; ok {
		return err
	}
	return s.err
}

type stubOrdersLineItemRepo struct {
	created     []omsDomain.SalesOrderLineItem
	createErr   error
	updateCalls []struct {
		ID     string
		Fields map[string]any
	}
	updateErr error
}

func (s *stubOrdersLineItemRepo) Create(_ context.Context, li *omsDomain.SalesOrderLineItem) error {
	if s.createErr != nil {
		return s.createErr
	}
	li.ID = fmt.Sprintf("line-item-%d", len(s.created)+1)
	s.created = append(s.created, *li)
	return nil
}
func (s *stubOrdersLineItemRepo) FindByID(_ context.Context, _ string) (*omsDomain.SalesOrderLineItem, error) {
	return nil, omsDomain.ErrSalesOrderLineItemNotFound
}
func (s *stubOrdersLineItemRepo) ListBySalesOrder(_ context.Context, _ string) ([]omsDomain.SalesOrderLineItem, error) {
	return nil, nil
}
func (s *stubOrdersLineItemRepo) DeleteBySalesOrder(_ context.Context, _ string) error { return nil }
func (s *stubOrdersLineItemRepo) Update(_ context.Context, id string, fields map[string]any) error {
	s.updateCalls = append(s.updateCalls, struct {
		ID     string
		Fields map[string]any
	}{id, fields})
	return s.updateErr
}

type stubOrdersLinkRepo struct {
	created []omsDomain.SalesOrderLineItemStockMovement
}

func (s *stubOrdersLinkRepo) ExistsForLineItem(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (s *stubOrdersLinkRepo) Create(_ context.Context, l *omsDomain.SalesOrderLineItemStockMovement) error {
	s.created = append(s.created, *l)
	return nil
}

func (s *stubOrdersLinkRepo) ListByLineItem(_ context.Context, lineItemID string) ([]omsDomain.SalesOrderLineItemStockMovement, error) {
	var out []omsDomain.SalesOrderLineItemStockMovement
	for _, l := range s.created {
		if l.SalesOrderLineItemID == lineItemID {
			out = append(out, l)
		}
	}
	return out, nil
}

type stubOrdersCustomerRepo struct {
	byWooID   map[int64]customersDomain.Customer
	created   []customersDomain.Customer
	createErr error
	findErr   error
}

func (s *stubOrdersCustomerRepo) Create(_ context.Context, c *customersDomain.Customer) error {
	if s.createErr != nil {
		return s.createErr
	}
	c.ID = fmt.Sprintf("customer-%d", len(s.created)+1)
	s.created = append(s.created, *c)
	return nil
}
func (s *stubOrdersCustomerRepo) FindByID(_ context.Context, _ string) (*customersDomain.Customer, error) {
	return nil, customersDomain.ErrCustomerNotFound
}
func (s *stubOrdersCustomerRepo) FindByWooCustomerID(_ context.Context, wooCustomerID int64) (*customersDomain.Customer, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if c, ok := s.byWooID[wooCustomerID]; ok {
		return &c, nil
	}
	return nil, nil
}
func (s *stubOrdersCustomerRepo) List(_ context.Context) ([]customersDomain.Customer, error) {
	return nil, nil
}
func (s *stubOrdersCustomerRepo) Update(_ context.Context, _ *customersDomain.Customer) error {
	return nil
}
func (s *stubOrdersCustomerRepo) Delete(_ context.Context, _ string) error { return nil }

type stubUpdateSalesOrderUC struct {
	calls []appoms.UpdateSalesOrderInput
	ids   []string
	err   error
}

func (s *stubUpdateSalesOrderUC) Execute(_ context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
	s.ids = append(s.ids, id)
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	return &omsDomain.SalesOrder{ID: id}, nil
}

// ── tax stubs (ImportWooProductsUseCase's class/rate/link reconciliation) ──

type stubChannelsTaxClassRepo struct {
	created   []taxDomain.TaxClass
	createErr error
}

func (s *stubChannelsTaxClassRepo) Create(_ context.Context, c *taxDomain.TaxClass) error {
	if s.createErr != nil {
		return s.createErr
	}
	if c.ID == "" {
		c.ID = fmt.Sprintf("class-%d", len(s.created)+1)
	}
	s.created = append(s.created, *c)
	return nil
}
func (s *stubChannelsTaxClassRepo) FindByID(_ context.Context, id string) (*taxDomain.TaxClass, error) {
	for _, c := range s.created {
		if c.ID == id {
			cc := c
			return &cc, nil
		}
	}
	return nil, taxDomain.ErrTaxClassNotFound
}
func (s *stubChannelsTaxClassRepo) List(_ context.Context) ([]taxDomain.TaxClass, error) {
	return s.created, nil
}
func (s *stubChannelsTaxClassRepo) Update(_ context.Context, _ *taxDomain.TaxClass) error { return nil }
func (s *stubChannelsTaxClassRepo) Delete(_ context.Context, _ string) error              { return nil }

type stubChannelsTaxRateRepo struct {
	upserted  []taxDomain.TaxRate
	upsertErr error
}

func (s *stubChannelsTaxRateRepo) Upsert(_ context.Context, r *taxDomain.TaxRate) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	if r.ID == "" {
		r.ID = fmt.Sprintf("rate-%d", len(s.upserted)+1)
	}
	s.upserted = append(s.upserted, *r)
	return nil
}
func (s *stubChannelsTaxRateRepo) FindByID(_ context.Context, _ string) (*taxDomain.TaxRate, error) {
	return nil, taxDomain.ErrTaxRateNotFound
}
func (s *stubChannelsTaxRateRepo) List(_ context.Context) ([]taxDomain.TaxRate, error) {
	return s.upserted, nil
}
func (s *stubChannelsTaxRateRepo) Delete(_ context.Context, _ string) error { return nil }

type stubChannelsExternalTaxClassLinkRepo struct {
	links     map[string]taxDomain.ExternalTaxClassLink // key: provider+"|"+externalSlug
	upsertErr error
}

func (s *stubChannelsExternalTaxClassLinkRepo) Upsert(_ context.Context, l *taxDomain.ExternalTaxClassLink) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	if s.links == nil {
		s.links = map[string]taxDomain.ExternalTaxClassLink{}
	}
	if l.ID == "" {
		l.ID = fmt.Sprintf("link-%d", len(s.links)+1)
	}
	s.links[l.Provider+"|"+l.ExternalSlug] = *l
	return nil
}
func (s *stubChannelsExternalTaxClassLinkRepo) FindByID(_ context.Context, id string) (*taxDomain.ExternalTaxClassLink, error) {
	for _, l := range s.links {
		if l.ID == id {
			ll := l
			return &ll, nil
		}
	}
	return nil, taxDomain.ErrExternalTaxClassLinkNotFound
}
func (s *stubChannelsExternalTaxClassLinkRepo) FindByProviderAndSlug(_ context.Context, provider, externalSlug string) (*taxDomain.ExternalTaxClassLink, error) {
	if l, ok := s.links[provider+"|"+externalSlug]; ok {
		ll := l
		return &ll, nil
	}
	return nil, taxDomain.ErrExternalTaxClassLinkNotFound
}
func (s *stubChannelsExternalTaxClassLinkRepo) List(_ context.Context) ([]taxDomain.ExternalTaxClassLink, error) {
	out := make([]taxDomain.ExternalTaxClassLink, 0, len(s.links))
	for _, l := range s.links {
		out = append(out, l)
	}
	return out, nil
}
func (s *stubChannelsExternalTaxClassLinkRepo) Delete(_ context.Context, _ string) error { return nil }
