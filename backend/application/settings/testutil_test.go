package settings_test

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

var errBoom = errors.New("boom")

// newMockGormDB mirrors application/inventory's helper of the same name —
// see that package for the full rationale.
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

type stubSettingsRepo struct {
	getResult *settingsDomain.TenantSettings
	getErr    error
	upsertErr error
	upserted  []settingsDomain.TenantSettings
}

func (s *stubSettingsRepo) Get(_ context.Context) (*settingsDomain.TenantSettings, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResult != nil {
		return s.getResult, nil
	}
	return &settingsDomain.TenantSettings{}, nil
}

func (s *stubSettingsRepo) Upsert(_ context.Context, ts *settingsDomain.TenantSettings) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, *ts)
	return nil
}
