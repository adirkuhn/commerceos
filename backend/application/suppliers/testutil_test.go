package suppliers_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
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

type stubSupplierRepo struct {
	suppliers    []suppliersDomain.Supplier
	findResult   *suppliersDomain.Supplier
	findErr      error
	createErr    error
	createCalled bool
	updateErr    error
	updated      []suppliersDomain.Supplier
	deleteErr    error
	deleted      []string
}

func (s *stubSupplierRepo) Create(_ context.Context, sup *suppliersDomain.Supplier) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	sup.ID = "supplier-1"
	return nil
}
func (s *stubSupplierRepo) FindByID(_ context.Context, _ string) (*suppliersDomain.Supplier, error) {
	return s.findResult, s.findErr
}
func (s *stubSupplierRepo) List(_ context.Context) ([]suppliersDomain.Supplier, error) {
	return s.suppliers, nil
}
func (s *stubSupplierRepo) Update(_ context.Context, sup *suppliersDomain.Supplier) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *sup)
	return nil
}
func (s *stubSupplierRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}
