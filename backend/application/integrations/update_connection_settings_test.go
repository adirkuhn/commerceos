package integrations_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
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

// newMockGormDBNoTx is for validation-failure tests where Execute must
// reject the input before ever opening a tenant.WithTenant transaction.
func newMockGormDBNoTx(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)
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
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

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
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: " usd "})

	require.NoError(t, err)
	require.Equal(t, "USD", result.DefaultCurrency)
	require.Equal(t, "USD", updated.DefaultCurrency)
}

func TestUpdateConnectionSettings_setsDefaultWarehouse(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1"}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})
	warehouseID := "wh-1"

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{
		ID: "conn-1", DefaultCurrency: "USD", DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
	require.NotNil(t, result.DefaultWarehouseID)
	require.Equal(t, "wh-1", *result.DefaultWarehouseID)
	require.NotNil(t, updated.DefaultWarehouseID)
	require.Equal(t, "wh-1", *updated.DefaultWarehouseID)
}

func TestUpdateConnectionSettings_clearsDefaultWarehouseWhenNilOrEmpty(t *testing.T) {
	existingWarehouse := "wh-old"
	conn := &integrationsDomain.Connection{ID: "conn-1", DefaultWarehouseID: &existingWarehouse}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})
	empty := ""

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{
		ID: "conn-1", DefaultCurrency: "USD", DefaultWarehouseID: &empty,
	})

	require.NoError(t, err)
	require.Nil(t, result.DefaultWarehouseID)
	require.Nil(t, updated.DefaultWarehouseID)
}

func TestUpdateConnectionSettings_setsBolEconomicOperatorIDAndDeliveryCode(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1"}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})
	economicOperatorID := "3fa85f64-5717-4562-b3fc-2c963f66afa6"
	deliveryCode := "1-2d"

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{
		ID: "conn-1", DefaultCurrency: "USD",
		BolEconomicOperatorID: &economicOperatorID, BolDeliveryCode: &deliveryCode,
	})

	require.NoError(t, err)
	require.NotNil(t, result.BolEconomicOperatorID)
	require.Equal(t, economicOperatorID, *result.BolEconomicOperatorID)
	require.NotNil(t, result.BolDeliveryCode)
	require.Equal(t, "1-2d", *result.BolDeliveryCode)
	require.NotNil(t, updated.BolEconomicOperatorID)
	require.NotNil(t, updated.BolDeliveryCode)
}

func TestUpdateConnectionSettings_clearsBolFieldsWhenNilOrEmpty(t *testing.T) {
	existingOperatorID := "old-id"
	existingDeliveryCode := "1-2d"
	conn := &integrationsDomain.Connection{ID: "conn-1", BolEconomicOperatorID: &existingOperatorID, BolDeliveryCode: &existingDeliveryCode}
	var updated *integrationsDomain.Connection
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, c *integrationsDomain.Connection) error {
			updated = c
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})
	empty := ""

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{
		ID: "conn-1", DefaultCurrency: "USD",
		BolEconomicOperatorID: &empty, BolDeliveryCode: &empty,
	})

	require.NoError(t, err)
	require.Nil(t, result.BolEconomicOperatorID)
	require.Nil(t, result.BolDeliveryCode)
	require.Nil(t, updated.BolEconomicOperatorID)
	require.Nil(t, updated.BolDeliveryCode)
}

func TestUpdateConnectionSettings_rejectsInvalidBolDeliveryCode(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1"}
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn: func(_ context.Context, _ *integrationsDomain.Connection) error {
			t.Fatal("must not persist an invalid deliveryCode")
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDBNoTx(t), repo, &eventrecordtest.Outbox{})
	bogus := "next-week"

	_, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{
		ID: "conn-1", DefaultCurrency: "USD", BolDeliveryCode: &bogus,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, integrationsDomain.ErrInvalidBolDeliveryCode))
}

func TestUpdateConnectionSettings_propagatesNotFound(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) {
			return nil, integrationsDomain.ErrConnectionNotFound
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

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
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: "USD"})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}

func TestUpdateConnectionSettings_setsBolShippingCost(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1"}
	outbox := &eventrecordtest.Outbox{}
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn:   func(_ context.Context, _ *integrationsDomain.Connection) error { return nil },
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, outbox)
	cost := int64(495)

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{
		ID: "conn-1", DefaultCurrency: "EUR", BolShippingCostMinorUnits: &cost,
	})

	require.NoError(t, err)
	require.Equal(t, int64(495), result.BolShippingCostMinorUnits)
}

func TestUpdateConnectionSettings_omittedBolShippingCostIsKept(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "conn-1", BolShippingCostMinorUnits: 495}
	repo := &mocks.ConnectionRepo{
		FindByIDFn: func(_ context.Context, _ string) (*integrationsDomain.Connection, error) { return conn, nil },
		UpdateFn:   func(_ context.Context, _ *integrationsDomain.Connection) error { return nil },
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDB(t), repo, &eventrecordtest.Outbox{})

	result, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: "EUR"})

	require.NoError(t, err)
	require.Equal(t, int64(495), result.BolShippingCostMinorUnits)
}

func TestUpdateConnectionSettings_rejectsNegativeBolShippingCost(t *testing.T) {
	repo := &mocks.ConnectionRepo{
		UpdateFn: func(_ context.Context, _ *integrationsDomain.Connection) error {
			t.Fatal("must not persist a negative shipping cost")
			return nil
		},
	}
	uc := appintegrations.NewUpdateConnectionSettingsUseCase(newMockGormDBNoTx(t), repo, &eventrecordtest.Outbox{})
	cost := int64(-1)

	_, err := uc.Execute(context.Background(), appintegrations.UpdateConnectionSettingsInput{ID: "conn-1", DefaultCurrency: "EUR", BolShippingCostMinorUnits: &cost})

	require.ErrorIs(t, err, integrationsDomain.ErrInvalidBolShippingCost)
}
