package promotions_test

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

// newMockGormDB returns a *gorm.DB backed by sqlmock, wired to satisfy
// tenant.WithTenant's Begin/set_config/Commit sequence -- same pattern as
// application/tax's own newMockGormDB. The use case under test never issues
// real SQL itself (that's the stub repos' job), so only the transaction
// envelope needs to be mocked.
func newMockGormDB(t *testing.T) *gorm.DB {
	return newMockGormDBN(t, 1)
}

// newMockGormDBN is newMockGormDB generalized to n separate
// tenant.WithTenant calls against the same db -- needed by use cases that
// call it more than once per Execute (e.g. ApplyPromotionSKUUseCase's own
// SKU lookup plus its later MarkApplied write).
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

type stubPromotionRepo struct {
	created   []promotionsDomain.Promotion
	createErr error

	findResult *promotionsDomain.Promotion
	findErr    error

	list    []promotionsDomain.Promotion
	listErr error

	updated   []promotionsDomain.Promotion
	updateErr error

	deleted   []string
	deleteErr error
}

func (s *stubPromotionRepo) Create(_ context.Context, p *promotionsDomain.Promotion) error {
	if s.createErr != nil {
		return s.createErr
	}
	if p.ID == "" {
		p.ID = "promo-1"
	}
	s.created = append(s.created, *p)
	return nil
}
func (s *stubPromotionRepo) FindByID(_ context.Context, _ string) (*promotionsDomain.Promotion, error) {
	if s.findResult == nil && s.findErr == nil {
		return nil, promotionsDomain.ErrPromotionNotFound
	}
	return s.findResult, s.findErr
}
func (s *stubPromotionRepo) List(_ context.Context) ([]promotionsDomain.Promotion, error) {
	return s.list, s.listErr
}
func (s *stubPromotionRepo) Update(_ context.Context, p *promotionsDomain.Promotion) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *p)
	return nil
}
func (s *stubPromotionRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubPromotionSKURepo struct {
	created   []promotionsDomain.PromotionSKU
	createErr error

	listResult []promotionsDomain.PromotionSKU
	listErr    error

	deletedByPromotion []string
	deleteErr          error

	dueForActivation    []promotionsDomain.PromotionSKU
	dueForActivationErr error
	dueForReversal      []promotionsDomain.PromotionSKU
	dueForReversalErr   error
	markedApplied       []markAppliedCall
	markAppliedErr      error
	markedReverted      []string
	markRevertedErr     error
}

func (s *stubPromotionSKURepo) Create(_ context.Context, line *promotionsDomain.PromotionSKU) error {
	if s.createErr != nil {
		return s.createErr
	}
	if line.ID == "" {
		line.ID = "promo-sku-1"
	}
	s.created = append(s.created, *line)
	return nil
}
func (s *stubPromotionSKURepo) ListByPromotion(_ context.Context, _ string) ([]promotionsDomain.PromotionSKU, error) {
	return s.listResult, s.listErr
}
func (s *stubPromotionSKURepo) DeleteByPromotion(_ context.Context, promotionID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedByPromotion = append(s.deletedByPromotion, promotionID)
	return nil
}
func (s *stubPromotionSKURepo) ListDueForActivation(_ context.Context, _ time.Time) ([]promotionsDomain.PromotionSKU, error) {
	return s.dueForActivation, s.dueForActivationErr
}
func (s *stubPromotionSKURepo) ListDueForReversal(_ context.Context, _ time.Time) ([]promotionsDomain.PromotionSKU, error) {
	return s.dueForReversal, s.dueForReversalErr
}
func (s *stubPromotionSKURepo) MarkApplied(_ context.Context, lineID string, previousSalePriceMinorUnits *int64, previousSaleEndDate *time.Time) error {
	if s.markAppliedErr != nil {
		return s.markAppliedErr
	}
	s.markedApplied = append(s.markedApplied, markAppliedCall{lineID, previousSalePriceMinorUnits, previousSaleEndDate})
	return nil
}
func (s *stubPromotionSKURepo) MarkReverted(_ context.Context, lineID string) error {
	if s.markRevertedErr != nil {
		return s.markRevertedErr
	}
	s.markedReverted = append(s.markedReverted, lineID)
	return nil
}

type markAppliedCall struct {
	lineID                      string
	previousSalePriceMinorUnits *int64
	previousSaleEndDate         *time.Time
}

type stubSKUExistenceChecker struct {
	findResult *pmsDomain.SKU
	findErr    error
	gotIDs     []string
}

func (s *stubSKUExistenceChecker) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	s.gotIDs = append(s.gotIDs, id)
	if s.findResult == nil && s.findErr == nil {
		return &pmsDomain.SKU{ID: id}, nil
	}
	return s.findResult, s.findErr
}
