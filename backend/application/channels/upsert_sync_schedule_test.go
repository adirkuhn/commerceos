package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestUpsertSyncSchedule_createsScheduleForRunnableEntity(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 15,
	})

	require.NoError(t, err)
	require.Len(t, repo.upserted, 1)
	require.True(t, s.Enabled)
	require.NotNil(t, s.NextRunAt)
}

func TestUpsertSyncSchedule_disabledScheduleHasNoNextRunAt(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: false, IntervalMinutes: 60,
	})

	require.NoError(t, err)
	require.Nil(t, s.NextRunAt)
}

func TestUpsertSyncSchedule_allowsSavingOrdersDisabled(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders,
		Direction: channelsDomain.SyncDirectionImport, Enabled: false, IntervalMinutes: 60,
	})

	require.NoError(t, err)
}

func TestUpsertSyncSchedule_rejectsEnablingOrdersImportWithoutWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_allowsEnablingOrdersImportWithWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)
	warehouseID := "wh-1"

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 60,
		DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
	require.True(t, s.Enabled)
	require.Len(t, repo.upserted, 1)
}

func TestUpsertSyncSchedule_rejectsEnablingOrdersExport(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)
	warehouseID := "wh-1"

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 60,
		DefaultWarehouseID: &warehouseID,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_rejectsEnablingCategories(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityCategories,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity))
}

func TestUpsertSyncSchedule_rejectsInvalidEntityType(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: "bogus",
		Direction: channelsDomain.SyncDirectionImport, Enabled: false, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrInvalidSyncEntityType))
}

func TestUpsertSyncSchedule_rejectsInvalidDirection(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts,
		Direction: "sideways", Enabled: false, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrInvalidSyncDirection))
}

func TestUpsertSyncSchedule_rejectsNonPositiveInterval(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 0,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrInvalidSyncInterval))
}

func TestUpsertSyncSchedule_rejectsEnablingProductsImportWithoutWarehouseAndCurrency(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 15,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrDefaultWarehouseAndCurrencyRequired))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_allowsEnablingProductsImportWithWarehouseAndCurrency(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)
	warehouseID := "wh-1"
	currency := "USD"

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 15,
		DefaultWarehouseID: &warehouseID, DefaultCurrency: &currency,
	})

	require.NoError(t, err)
	require.Equal(t, "wh-1", *s.DefaultWarehouseID)
	require.Equal(t, "USD", *s.DefaultCurrency)
}

func TestUpsertSyncSchedule_rejectsEnablingStockImportWithoutWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 15,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired))
}

func TestUpsertSyncSchedule_allowsEnablingStockImportWithWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)
	warehouseID := "wh-1"

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 15,
		DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
}

func TestUpsertSyncSchedule_doesNotRequireWarehouseOrCurrencyForStockExport(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 15,
	})

	require.NoError(t, err)
}

func TestUpsertSyncSchedule_propagatesRepoError(t *testing.T) {
	repo := &stubSyncScheduleRepo{upsertErr: errors.New("db down")}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 15,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}
