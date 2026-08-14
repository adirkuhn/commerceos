package tax_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// however many tenant.WithTenant Begin/set_config/Commit sequences the test
// needs -- same pattern as application/channels' use-case tests. n is the
// number of separate tenant.WithTenant calls the use case under test makes.
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

type stubTaxClassRepo struct {
	created   []taxDomain.TaxClass
	createErr error

	findResult *taxDomain.TaxClass
	findErr    error

	list    []taxDomain.TaxClass
	listErr error

	updated   []taxDomain.TaxClass
	updateErr error

	deleted   []string
	deleteErr error
}

func (s *stubTaxClassRepo) Create(_ context.Context, c *taxDomain.TaxClass) error {
	if s.createErr != nil {
		return s.createErr
	}
	if c.ID == "" {
		c.ID = "class-1"
	}
	s.created = append(s.created, *c)
	return nil
}
func (s *stubTaxClassRepo) FindByID(_ context.Context, _ string) (*taxDomain.TaxClass, error) {
	return s.findResult, s.findErr
}
func (s *stubTaxClassRepo) List(_ context.Context) ([]taxDomain.TaxClass, error) {
	return s.list, s.listErr
}
func (s *stubTaxClassRepo) Update(_ context.Context, c *taxDomain.TaxClass) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *c)
	return nil
}
func (s *stubTaxClassRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubTaxRateRepo struct {
	upserted  []taxDomain.TaxRate
	upsertErr error

	findResult *taxDomain.TaxRate
	findErr    error

	list    []taxDomain.TaxRate
	listErr error

	deleted   []string
	deleteErr error
}

func (s *stubTaxRateRepo) Upsert(_ context.Context, r *taxDomain.TaxRate) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	if r.ID == "" {
		r.ID = "rate-1"
	}
	s.upserted = append(s.upserted, *r)
	return nil
}
func (s *stubTaxRateRepo) FindByID(_ context.Context, _ string) (*taxDomain.TaxRate, error) {
	return s.findResult, s.findErr
}
func (s *stubTaxRateRepo) List(_ context.Context) ([]taxDomain.TaxRate, error) {
	return s.list, s.listErr
}
func (s *stubTaxRateRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubSKUTaxClassChecker struct {
	exists bool
	err    error
}

func (s *stubSKUTaxClassChecker) ExistsWithTaxClass(_ context.Context, _ string) (bool, error) {
	return s.exists, s.err
}
