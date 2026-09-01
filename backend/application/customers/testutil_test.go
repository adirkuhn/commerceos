package customers_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
)

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

// ── Stub repo ────────────────────────────────────────────────────────────────

type stubCustomerRepo struct {
	customers    []customersDomain.Customer
	findResult   *customersDomain.Customer
	findErr      error
	createErr    error
	createCalled bool
	updateErr    error
	updated      []customersDomain.Customer
	deleteErr    error
	deleted      []string
}

func (s *stubCustomerRepo) Create(_ context.Context, c *customersDomain.Customer) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	c.ID = "customer-1"
	return nil
}
func (s *stubCustomerRepo) FindByID(_ context.Context, _ string) (*customersDomain.Customer, error) {
	return s.findResult, s.findErr
}
func (s *stubCustomerRepo) FindByWooCustomerID(_ context.Context, _ int64) (*customersDomain.Customer, error) {
	return s.findResult, s.findErr
}
func (s *stubCustomerRepo) List(_ context.Context) ([]customersDomain.Customer, error) {
	return s.customers, nil
}
func (s *stubCustomerRepo) Search(_ context.Context, _ string, _ int) ([]customersDomain.Customer, error) {
	return nil, nil
}
func (s *stubCustomerRepo) Update(_ context.Context, c *customersDomain.Customer) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *c)
	return nil
}
func (s *stubCustomerRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}
