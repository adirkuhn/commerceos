package inventory_test

import (
	"context"
	"io"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

// stubOutboxRepo is a no-op outbox — tests that need a stock-changing use
// case to raise an inventory.stock.changed event only care that Enqueue
// was called (via `calls`), not that it actually persists anything.
type stubOutboxRepo struct {
	calls []string // subjects enqueued
	err   error
}

func (s *stubOutboxRepo) Enqueue(_ context.Context, subject, _ string, _ []byte, _ time.Time) error {
	s.calls = append(s.calls, subject)
	return s.err
}
func (s *stubOutboxRepo) FetchPending(_ context.Context, _ int) ([]outbox.Message, error) {
	return nil, nil
}
func (s *stubOutboxRepo) MarkPublished(_ context.Context, _ uint) error { return nil }
func (s *stubOutboxRepo) MarkFailed(_ context.Context, _ uint) error    { return nil }

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// tenant.WithTenant's Begin/set_config/Commit sequence — same pattern as
// internal/tenant/rls_sql_test.go and application/pms's use-case tests. The
// use case under test never issues real SQL itself (that's the stub repos'
// job), so only the transaction envelope needs to be mocked.
func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	return newMockGormDBN(t, 1)
}

// newMockGormDBN is newMockGormDB for a use case that opens more than one
// tenant.WithTenant transaction sequentially -- e.g.
// WriteOffExpiredBatchUseCase, which can't nest MarkDamagedOrLostUseCase's
// own transaction inside its own (see damagedOrLostRecorder's doc comment).
// n is the total number of Begin/set_config/Commit cycles expected across
// the whole test, including any opened by an injected use case's own
// Execute call.
func newMockGormDBN(t *testing.T, n int) *gorm.DB {
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

// ── Stub repos ────────────────────────────────────────────────────────────────

type stubWarehouseRepo struct {
	warehouses   []invDomain.Warehouse
	findResult   *invDomain.Warehouse
	findErr      error
	createErr    error
	createCalled bool
	updateErr    error
	updated      []invDomain.Warehouse
	deleteErr    error
	deleted      []string
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
func (s *stubWarehouseRepo) Update(_ context.Context, w *invDomain.Warehouse) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *w)
	return nil
}
func (s *stubWarehouseRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubBatchRepo struct {
	batches      []invDomain.Batch
	findResult   *invDomain.Batch
	findErr      error
	createErr    error
	createCalled bool
	created      *invDomain.Batch
	deleteErr    error
	deleted      []string

	expiringSoon    []invDomain.ExpiringBatch
	expiringSoonErr error

	toExpire       []invDomain.BatchToExpire
	toExpireErr    error
	markExpiredErr error
	markExpired    []string

	pendingReview      []invDomain.ExpiredBatchReview
	pendingReviewErr   error
	markReviewedErr    error
	markReviewed       []string
	extendExpiryErr    error
	extendedID         string
	extendedExpiryDate time.Time
}

func (s *stubBatchRepo) Create(_ context.Context, b *invDomain.Batch) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	b.ID = "batch-1"
	s.created = b
	return nil
}
func (s *stubBatchRepo) FindByID(_ context.Context, _ string) (*invDomain.Batch, error) {
	return s.findResult, s.findErr
}
func (s *stubBatchRepo) ListBySKU(_ context.Context, _ string) ([]invDomain.Batch, error) {
	return s.batches, nil
}
func (s *stubBatchRepo) ListBySKUs(_ context.Context, _ []string) ([]invDomain.Batch, error) {
	return s.batches, nil
}
func (s *stubBatchRepo) Update(_ context.Context, _ *invDomain.Batch) error { return nil }
func (s *stubBatchRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubBatchRepo) ListExpiringSoon(_ context.Context, _ int) ([]invDomain.ExpiringBatch, error) {
	return s.expiringSoon, s.expiringSoonErr
}
func (s *stubBatchRepo) ListToExpire(_ context.Context) ([]invDomain.BatchToExpire, error) {
	return s.toExpire, s.toExpireErr
}
func (s *stubBatchRepo) MarkExpired(_ context.Context, id string) error {
	if s.markExpiredErr != nil {
		return s.markExpiredErr
	}
	s.markExpired = append(s.markExpired, id)
	return nil
}
func (s *stubBatchRepo) ListPendingReview(_ context.Context) ([]invDomain.ExpiredBatchReview, error) {
	return s.pendingReview, s.pendingReviewErr
}
func (s *stubBatchRepo) MarkReviewed(_ context.Context, id string) error {
	if s.markReviewedErr != nil {
		return s.markReviewedErr
	}
	s.markReviewed = append(s.markReviewed, id)
	return nil
}
func (s *stubBatchRepo) ExtendExpiry(_ context.Context, id string, newExpiryDate time.Time) error {
	if s.extendExpiryErr != nil {
		return s.extendExpiryErr
	}
	s.extendedID = id
	s.extendedExpiryDate = newExpiryDate
	return nil
}

type stubStockMovementRepo struct {
	bySKU     []invDomain.StockMovement
	byBatch   []invDomain.StockMovement
	createErr error
	created   []invDomain.StockMovement

	bySKUs              []invDomain.StockMovement
	distinctSKUIDs      []string
	distinctSKUIDsTotal int64
	distinctSKUIDsErr   error
	gotChangedSince     *time.Time
	gotLimit, gotOffset int
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
func (s *stubStockMovementRepo) ListBySKUs(_ context.Context, _ []string) ([]invDomain.StockMovement, error) {
	return s.bySKUs, nil
}
func (s *stubStockMovementRepo) ListDistinctSKUIDs(_ context.Context, changedSince *time.Time, limit, offset int) ([]string, int64, error) {
	s.gotChangedSince, s.gotLimit, s.gotOffset = changedSince, limit, offset
	if s.distinctSKUIDsErr != nil {
		return nil, 0, s.distinctSKUIDsErr
	}
	return s.distinctSKUIDs, s.distinctSKUIDsTotal, nil
}

// stubSettingsRepo satisfies the settingsGetter interface used by
// GetSKUStockLevelUseCase/ListStockLevelsUseCase -- a zero-value result
// (InventorySyncRulesEnabled: false) matches an unsaved tenant, preserving
// today's default Syncable rule.
type stubSettingsRepo struct {
	result *settingsDomain.TenantSettings
	err    error
}

func (s *stubSettingsRepo) Get(_ context.Context) (*settingsDomain.TenantSettings, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.result != nil {
		return s.result, nil
	}
	return &settingsDomain.TenantSettings{}, nil
}

// stubSKURepo implements pms.SKURepository -- only FindByID is exercised
// (by SplitCaseIntoUnitsUseCase), the rest are unused no-ops.
type stubSKURepo struct {
	findByIDResult *pmsDomain.SKU
	findByIDErr    error
}

func (s *stubSKURepo) Create(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *stubSKURepo) FindByID(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return s.findByIDResult, s.findByIDErr
}
func (s *stubSKURepo) FindByCode(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubSKURepo) FindByBarcode(_ context.Context, _ string) (*pmsDomain.SKU, error) {
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubSKURepo) ListByVariant(_ context.Context, _ string) ([]pmsDomain.SKU, error) {
	return nil, nil
}
func (s *stubSKURepo) Update(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *stubSKURepo) Delete(_ context.Context, _ string) error         { return nil }
func (s *stubSKURepo) ListWithExpiredSale(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

func (s *stubSKURepo) ExistsWithTaxClass(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (s *stubSKURepo) ListActive(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

func (s *stubSKURepo) UpdateCostPrice(_ context.Context, _ string, _ int64) error {
	return nil
}
func (s *stubSKURepo) SetManualCostPrice(_ context.Context, _ string, _ int64) error {
	return nil
}
func (s *stubSKURepo) ClearCostPriceOverride(_ context.Context, _ string) error {
	return nil
}

type stubDiscardRecordRepo struct {
	createErr error
	created   []invDomain.DiscardRecord
	list      []invDomain.DiscardRecordReport
	listTotal int64
	listErr   error
	gotFilter invDomain.DiscardRecordFilter
}

func (s *stubDiscardRecordRepo) Create(_ context.Context, d *invDomain.DiscardRecord) error {
	if s.createErr != nil {
		return s.createErr
	}
	d.ID = "discard-1"
	s.created = append(s.created, *d)
	return nil
}
func (s *stubDiscardRecordRepo) List(_ context.Context, filter invDomain.DiscardRecordFilter) ([]invDomain.DiscardRecordReport, int64, error) {
	s.gotFilter = filter
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.list, s.listTotal, nil
}

type stubFileStorage struct {
	saveURL   string
	saveErr   error
	savedName string
	savedData []byte
}

func (s *stubFileStorage) Save(_ context.Context, _, filename string, data io.Reader) (string, error) {
	if s.saveErr != nil {
		return "", s.saveErr
	}
	s.savedName = filename
	if data != nil {
		s.savedData, _ = io.ReadAll(data)
	}
	if s.saveURL != "" {
		return s.saveURL, nil
	}
	return "https://example.com/" + filename, nil
}
func (s *stubFileStorage) Delete(_ context.Context, _ string) error { return nil }
