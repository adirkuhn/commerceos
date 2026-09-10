package theme_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
)

type fakeEditSessionRepo struct {
	byToken   map[string]themeDomain.EditSession
	createErr error
	created   *themeDomain.EditSession
}

func (f *fakeEditSessionRepo) Create(_ context.Context, s *themeDomain.EditSession) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.byToken == nil {
		f.byToken = map[string]themeDomain.EditSession{}
	}
	s.ID = "session-" + s.Token
	f.byToken[s.Token] = *s
	f.created = s
	return nil
}

func (f *fakeEditSessionRepo) FindByToken(_ context.Context, token string) (*themeDomain.EditSession, error) {
	if s, ok := f.byToken[token]; ok {
		return &s, nil
	}
	return nil, nil
}

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

type fakeThemeRepo struct {
	byChannelSlug map[string]themeDomain.ThemePage
	upsertErr     error
	upserted      *themeDomain.ThemePage
}

func themeKey(channelID string, slug themeDomain.PageSlug) string {
	return channelID + "|" + string(slug)
}

func (f *fakeThemeRepo) FindBySlug(_ context.Context, channelID string, slug themeDomain.PageSlug) (*themeDomain.ThemePage, error) {
	if f.byChannelSlug == nil {
		return nil, nil
	}
	if p, ok := f.byChannelSlug[themeKey(channelID, slug)]; ok {
		return &p, nil
	}
	return nil, nil
}

func (f *fakeThemeRepo) Upsert(_ context.Context, p *themeDomain.ThemePage) error {
	if f.upsertErr != nil {
		return f.upsertErr
	}
	if f.byChannelSlug == nil {
		f.byChannelSlug = map[string]themeDomain.ThemePage{}
	}
	f.byChannelSlug[themeKey(p.SalesChannelID, p.Slug)] = *p
	f.upserted = p
	return nil
}

type fakeChannelRepo struct {
	channelsDomain.SalesChannelRepository
	byID    map[string]channelsDomain.SalesChannel
	findErr error
}

func (f *fakeChannelRepo) FindByID(_ context.Context, id string) (*channelsDomain.SalesChannel, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if c, ok := f.byID[id]; ok {
		return &c, nil
	}
	return nil, channelsDomain.ErrSalesChannelNotFound
}
