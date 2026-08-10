package channels_test

import (
	"bytes"
	"context"
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
	apppms "github.com/adirkuhn/commerceos/application/pms"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
	conn    *integrationsDomain.Connection
	findErr error
}

func (s *stubConnectionRepo) Create(_ context.Context, _ *integrationsDomain.Connection) error {
	return nil
}
func (s *stubConnectionRepo) FindByID(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
	return s.conn, s.findErr
}
func (s *stubConnectionRepo) ListByTenant(_ context.Context) ([]integrationsDomain.Connection, error) {
	return nil, nil
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

	createVariableCalled []stubCreateVariableCall
	createVariableErr    error
	createVariableWithID string

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

func (s *stubWooClient) UpdateProductStock(_ context.Context, externalID string, quantity int) error {
	s.updateStockCalled = append(s.updateStockCalled, stubStockUpdate{ExternalID: externalID, Quantity: quantity})
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
	findByIDResult *pmsDomain.SKU
	findByIDErr    error
	byCode         map[string]pmsDomain.SKU
}

func (s *stubChannelsSKURepo) Create(_ context.Context, sku *pmsDomain.SKU) error {
	if s.createErr != nil {
		return s.createErr
	}
	sku.ID = "sku-1"
	s.created = append(s.created, *sku)
	return nil
}
func (s *stubChannelsSKURepo) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
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
func (s *stubChannelsSKURepo) Update(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *stubChannelsSKURepo) Delete(_ context.Context, _ string) error         { return nil }
func (s *stubChannelsSKURepo) ListWithExpiredSale(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
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
	created   []channelsDomain.SyncLogEntry
	createErr error
	byListing map[string][]channelsDomain.SyncLogEntry // newest-first
	all       []channelsDomain.SyncLogEntry            // newest-first, for ListAll
	listErr   error
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
	s.byListing[e.ListingID] = append([]channelsDomain.SyncLogEntry{*e}, s.byListing[e.ListingID]...)
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
	due          []channelsDomain.SyncSchedule
	recorded     []stubRunResult
}

type stubRunResult struct {
	ID        string
	RanAt     time.Time
	Status    string
	ErrMsg    string
	NextRunAt time.Time
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
func (s *stubSyncScheduleRepo) ListDue(_ context.Context, _ time.Time) ([]channelsDomain.SyncSchedule, error) {
	return s.due, nil
}
func (s *stubSyncScheduleRepo) RecordRunResult(_ context.Context, id string, ranAt time.Time, status, errMsg string, nextRunAt time.Time) error {
	s.recorded = append(s.recorded, stubRunResult{ID: id, RanAt: ranAt, Status: status, ErrMsg: errMsg, NextRunAt: nextRunAt})
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
func (s *stubImportJobRepo) MarkOrphanedRunningAsFailed(_ context.Context, _ string, _ time.Time) (int64, error) {
	return 0, nil
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
	calls []invDomain.StockMovement
	err   error
}

func (s *stubStockRecorder) Execute(_ context.Context, in invapp.RecordStockMovementInput) (*invDomain.StockMovement, error) {
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
