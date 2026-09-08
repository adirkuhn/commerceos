package pms_test

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubProductRepo struct {
	product *pmsDomain.Product

	// slugOwner, when non-nil, is what FindBySlug returns for slugTaken --
	// simulates another existing product already holding that slug.
	slugTaken string
	slugOwner *pmsDomain.Product

	listResult []pmsDomain.Product
	listTotal  int64
	listErr    error
	gotFilter  pmsDomain.ProductFilter
}

func (s *stubProductRepo) Create(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubProductRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Product, error) {
	return s.product, nil
}
func (s *stubProductRepo) FindBySlug(_ context.Context, slug string) (*pmsDomain.Product, error) {
	if s.slugTaken != "" && slug == s.slugTaken {
		return s.slugOwner, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}
func (s *stubProductRepo) List(_ context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	s.gotFilter = filter
	return s.listResult, s.listTotal, s.listErr
}
func (s *stubProductRepo) Update(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubProductRepo) Delete(_ context.Context, _ string) error             { return nil }
func (s *stubProductRepo) SearchByAnyNameToken(_ context.Context, _ []string, _ int) ([]pmsDomain.Product, error) {
	return nil, nil
}

func (s *stubProductRepo) CountByCategory(_ context.Context, _ pmsDomain.ProductFilter) (map[string]int64, error) {
	return nil, nil
}

type stubVariantRepo struct {
	existing     []pmsDomain.Variant
	createCalled bool
	findResult   *pmsDomain.Variant
	findErr      error
}

func (s *stubVariantRepo) Create(_ context.Context, v *pmsDomain.Variant) error {
	s.createCalled = true
	return nil
}
func (s *stubVariantRepo) FindByID(_ context.Context, _ string) (*pmsDomain.Variant, error) {
	if s.findResult == nil && s.findErr == nil {
		return &pmsDomain.Variant{ID: "variant-1"}, nil
	}
	return s.findResult, s.findErr
}
func (s *stubVariantRepo) ListByProduct(_ context.Context, _ string) ([]pmsDomain.Variant, error) {
	return s.existing, nil
}
func (s *stubVariantRepo) Update(_ context.Context, _ *pmsDomain.Variant) error { return nil }
func (s *stubVariantRepo) Delete(_ context.Context, _ string) error             { return nil }

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// tenant.WithTenant's Begin/set_config/Commit sequence — same pattern as
// internal/tenant/rls_sql_test.go. The use case under test never issues real
// SQL itself (that's the stub repos' job), so only the transaction envelope
// needs to be mocked.
func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	return gormDB
}

func TestCreateVariant_rejectsSecondVariantWhenHasVariantsIsFalse(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}}}
	uc := apppms.NewCreateVariantUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateVariantInput{ProductID: "p1", Name: "Large"})

	require.Error(t, err)
	require.True(t, errors.Is(err, pmsDomain.ErrVariantsNotEnabled))
	require.False(t, variantRepo.createCalled, "must not create a 2nd variant when has_variants is false")
}

func TestCreateVariant_allowsFirstVariantEvenWhenHasVariantsIsFalse(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: false}}
	variantRepo := &stubVariantRepo{existing: nil}
	uc := apppms.NewCreateVariantUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateVariantInput{ProductID: "p1", Name: "Default"})

	require.NoError(t, err)
	require.True(t, variantRepo.createCalled)
}

// TestCreateVariant_enqueuesProductChanged is a regression test for a
// brand-new variant-mode product never getting enrolled in any channel:
// CreateProductUseCase's own pms.product.changed fires enrollment before
// any variant exists, which finds zero variants and bails without ever
// creating a ProductChannelListing. Nothing re-triggers enrollment unless
// this create path also raises the event.
func TestCreateVariant_enqueuesProductChanged(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: true}}
	variantRepo := &stubVariantRepo{existing: nil}
	outboxRepo := &stubOutboxRepo{}
	uc := apppms.NewCreateVariantUseCase(newMockGormDB(t), productRepo, variantRepo, outboxRepo)

	_, err := uc.Execute(context.Background(), apppms.CreateVariantInput{ProductID: "p1", Name: "900ml"})

	require.NoError(t, err)
	require.Len(t, outboxRepo.calls, 1)
	require.Equal(t, "pms.product.changed", outboxRepo.calls[0])
	require.Contains(t, string(outboxRepo.payloads[0]), `"product_id":"p1"`)
}

func TestCreateVariant_allowsMultipleVariantsWhenHasVariantsIsTrue(t *testing.T) {
	productRepo := &stubProductRepo{product: &pmsDomain.Product{ID: "p1", HasVariants: true}}
	variantRepo := &stubVariantRepo{existing: []pmsDomain.Variant{{ID: "v1", ProductID: "p1"}}}
	uc := apppms.NewCreateVariantUseCase(newMockGormDB(t), productRepo, variantRepo, &stubOutboxRepo{})

	_, err := uc.Execute(context.Background(), apppms.CreateVariantInput{ProductID: "p1", Name: "Large"})

	require.NoError(t, err)
	require.True(t, variantRepo.createCalled)
}
