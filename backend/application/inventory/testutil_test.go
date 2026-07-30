package inventory_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// tenant.WithTenant's Begin/set_config/Commit sequence — same pattern as
// internal/tenant/rls_sql_test.go and application/pms's use-case tests. The
// use case under test never issues real SQL itself (that's the stub repos'
// job), so only the transaction envelope needs to be mocked.
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

// ── Stub repos ────────────────────────────────────────────────────────────────

type stubWarehouseRepo struct {
	warehouses   []invDomain.Warehouse
	findResult   *invDomain.Warehouse
	findErr      error
	createErr    error
	createCalled bool
}

func (s *stubWarehouseRepo) Create(_ context.Context, w *invDomain.Warehouse) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	w.ID = "warehouse-1"
	return nil
}
func (s *stubWarehouseRepo) FindByID(_ context.Context, _ string) (*invDomain.Warehouse, error) {
	return s.findResult, s.findErr
}
func (s *stubWarehouseRepo) List(_ context.Context) ([]invDomain.Warehouse, error) {
	return s.warehouses, nil
}
func (s *stubWarehouseRepo) Update(_ context.Context, _ *invDomain.Warehouse) error { return nil }
func (s *stubWarehouseRepo) Delete(_ context.Context, _ string) error               { return nil }

type stubBatchRepo struct {
	batches      []invDomain.Batch
	findResult   *invDomain.Batch
	findErr      error
	createErr    error
	createCalled bool
}

func (s *stubBatchRepo) Create(_ context.Context, b *invDomain.Batch) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	b.ID = "batch-1"
	return nil
}
func (s *stubBatchRepo) FindByID(_ context.Context, _ string) (*invDomain.Batch, error) {
	return s.findResult, s.findErr
}
func (s *stubBatchRepo) ListBySKU(_ context.Context, _ string) ([]invDomain.Batch, error) {
	return s.batches, nil
}
func (s *stubBatchRepo) Update(_ context.Context, _ *invDomain.Batch) error { return nil }

type stubStockMovementRepo struct {
	bySKU     []invDomain.StockMovement
	byBatch   []invDomain.StockMovement
	createErr error
	created   []invDomain.StockMovement
}

func (s *stubStockMovementRepo) Create(_ context.Context, m *invDomain.StockMovement) error {
	if s.createErr != nil {
		return s.createErr
	}
	m.ID = "movement-1"
	s.created = append(s.created, *m)
	return nil
}
func (s *stubStockMovementRepo) ListBySKU(_ context.Context, _ string) ([]invDomain.StockMovement, error) {
	return s.bySKU, nil
}
func (s *stubStockMovementRepo) ListByBatch(_ context.Context, _ string) ([]invDomain.StockMovement, error) {
	return s.byBatch, nil
}
