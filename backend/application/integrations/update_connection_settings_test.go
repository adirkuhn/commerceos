package integrations_test

import (
	"context"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	"github.com/adirkuhn/commerceos/application/integrations/mocks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
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

func TestUpdateConnectionSettings_setsDefaultCurrency(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", DefaultCurrency: "EUR"}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, id string) (*integrationsDomain.Connection, error) {
			require.Equal(t, "conn-1", id)
			return conn, nil
		},
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: "USD"})

	require.NoError(t, err)
	require.Equal(t, "USD", result.DefaultCurrency)
	require.NotNil(t, updated)
	require.Equal(t, "USD", updated.DefaultCurrency)
}

func TestUpdateConnectionSettings_normalizesCasingAndWhitespace(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1"}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo)

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: " usd "})

	require.NoError(t, err)
	require.Equal(t, "USD", result.DefaultCurrency)
	require.Equal(t, "USD", updated.DefaultCurrency)
}

func TestUpdateConnectionSettings_propagatesNotFound(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "missing", DefaultCurrency: "USD"})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrConnectionNotFound))
}

func TestUpdateConnectionSettings_propagatesUpdateError(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1"}
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, _ *integrationsDomain.Connection) error {
			return errors.New("db down")
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo)

	_, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: "USD"})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}
