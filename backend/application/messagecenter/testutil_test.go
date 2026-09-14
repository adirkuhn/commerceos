package messagecenter_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
)

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

type stubMessageTemplateRepo struct {
	created   []mcDomain.MessageTemplate
	updated   []mcDomain.MessageTemplate
	deletedID string
	byID      map[string]mcDomain.MessageTemplate
	all       []mcDomain.MessageTemplate
	findErr   error
}

func (s *stubMessageTemplateRepo) Create(_ context.Context, t *mcDomain.MessageTemplate) error {
	t.ID = "template-1"
	s.created = append(s.created, *t)
	return nil
}

func (s *stubMessageTemplateRepo) FindByID(_ context.Context, id string) (*mcDomain.MessageTemplate, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	tmpl, ok := s.byID[id]
	if !ok {
		return nil, mcDomain.ErrTemplateNotFound
	}
	return &tmpl, nil
}

func (s *stubMessageTemplateRepo) ListByTenant(_ context.Context) ([]mcDomain.MessageTemplate, error) {
	return s.all, nil
}

func (s *stubMessageTemplateRepo) Update(_ context.Context, t *mcDomain.MessageTemplate) error {
	s.updated = append(s.updated, *t)
	return nil
}

func (s *stubMessageTemplateRepo) Delete(_ context.Context, id string) error {
	s.deletedID = id
	return nil
}
