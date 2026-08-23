package promotions_test

import (
	"context"
	"testing"

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
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	mock.ExpectBegin()
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

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
