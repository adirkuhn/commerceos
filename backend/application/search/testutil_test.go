package search_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// tenant.WithTenant's Begin/set_config/Commit sequence -- same pattern used
// across this codebase's other application-layer test packages.
func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	return gormDB
}

// Only the methods GlobalSearchUseCase actually calls (List/Search) are
// exercised here -- the rest of each interface panics if ever called,
// same "fake the whole interface, panic on the unused rest" convention as
// application/integrations' fakeFullMyParcelClient.

type stubProductRepo struct {
	pmsDomain.ProductRepository
	listFn func(ctx context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error)
}

func (s *stubProductRepo) List(ctx context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	return s.listFn(ctx, filter)
}

type stubSupplierRepo struct {
	suppliersDomain.SupplierRepository
	searchFn func(ctx context.Context, query string, limit int) ([]suppliersDomain.Supplier, error)
}

func (s *stubSupplierRepo) Search(ctx context.Context, query string, limit int) ([]suppliersDomain.Supplier, error) {
	return s.searchFn(ctx, query, limit)
}

type stubCustomerRepo struct {
	customersDomain.CustomerRepository
	searchFn func(ctx context.Context, query string, limit int) ([]customersDomain.Customer, error)
}

func (s *stubCustomerRepo) Search(ctx context.Context, query string, limit int) ([]customersDomain.Customer, error) {
	return s.searchFn(ctx, query, limit)
}

type stubSalesOrderRepo struct {
	omsDomain.SalesOrderRepository
	searchFn func(ctx context.Context, query string, limit int) ([]omsDomain.SalesOrder, error)
}

func (s *stubSalesOrderRepo) Search(ctx context.Context, query string, limit int) ([]omsDomain.SalesOrder, error) {
	return s.searchFn(ctx, query, limit)
}
