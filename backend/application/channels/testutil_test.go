package channels_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

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

type stubWooClientFactory struct {
	client integrationsDomain.WooClient
	err    error
}

func (s *stubWooClientFactory) NewClient(_, _, _ string) (integrationsDomain.WooClient, error) {
	return s.client, s.err
}

// ── pms stubs ────────────────────────────────────────────────────────────────

type stubCategoryRepo struct {
	bySlug map[string]pmsDomain.Category
}

func (s *stubCategoryRepo) Create(_ context.Context, _ *pmsDomain.Category) error { return nil }
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
func (s *stubCategoryRepo) Delete(_ context.Context, _ string) error              { return nil }

type stubChannelsProductRepo struct {
	createdProducts []pmsDomain.Product
	createErr       error
	bySlug          map[string]pmsDomain.Product
	byID            map[string]pmsDomain.Product
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
func (s *stubChannelsProductRepo) List(_ context.Context, _ pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	return nil, 0, nil
}
func (s *stubChannelsProductRepo) Update(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubChannelsProductRepo) Delete(_ context.Context, _ string) error             { return nil }

type stubChannelsVariantRepo struct {
	created   []pmsDomain.Variant
	createErr error
	byProduct map[string][]pmsDomain.Variant
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
	return nil, pmsDomain.ErrVariantNotFound
}
func (s *stubChannelsVariantRepo) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	return s.byProduct[productID], nil
}
func (s *stubChannelsVariantRepo) Update(_ context.Context, _ *pmsDomain.Variant) error { return nil }
func (s *stubChannelsVariantRepo) Delete(_ context.Context, _ string) error             { return nil }

type stubChannelsSKURepo struct {
	created   []pmsDomain.SKU
	createErr error
	byVariant map[string][]pmsDomain.SKU
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
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubChannelsSKURepo) FindByCode(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubChannelsSKURepo) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	return s.byVariant[variantID], nil
}
func (s *stubChannelsSKURepo) Update(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *stubChannelsSKURepo) Delete(_ context.Context, _ string) error         { return nil }

// ── channels stub ────────────────────────────────────────────────────────────

type stubListingRepo struct {
	byExternalID map[string]channelsDomain.ProductChannelListing
	byProduct    map[string]channelsDomain.ProductChannelListing
	created      []channelsDomain.ProductChannelListing
	updated      []channelsDomain.ProductChannelListing
	findErr      error
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
	return nil, nil
}
func (s *stubListingRepo) Update(_ context.Context, l *channelsDomain.ProductChannelListing) error {
	s.updated = append(s.updated, *l)
	return nil
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
