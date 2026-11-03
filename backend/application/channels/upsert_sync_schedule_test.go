package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/require"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestUpsertSyncSchedule_createsScheduleForRunnableEntity(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 15,
	})

	require.NoError(t, err)
	require.Len(t, repo.upserted, 1)
	require.True(t, s.Enabled)
	require.NotNil(t, s.NextRunAt)
}

func TestUpsertSyncSchedule_disabledScheduleHasNoNextRunAt(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: false, IntervalMinutes: 60,
	})

	require.NoError(t, err)
	require.Nil(t, s.NextRunAt)
}

func TestUpsertSyncSchedule_allowsSavingOrdersDisabled(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders,
		Direction: channelsDomain.SyncDirectionImport, Enabled: false, IntervalMinutes: 60,
	})

	require.NoError(t, err)
}

func TestUpsertSyncSchedule_rejectsEnablingOrdersImportWithoutWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

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
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})
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
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})
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

func TestUpsertSyncSchedule_rejectsEnablingOrdersReconciliationWithoutWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrdersReconciliation,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 1440,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_allowsEnablingOrdersReconciliationWithWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})
	warehouseID := "wh-1"

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrdersReconciliation,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 1440,
		DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
	require.True(t, s.Enabled)
	require.Len(t, repo.upserted, 1)
}

func TestUpsertSyncSchedule_rejectsEnablingOrdersReconciliationExport(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})
	warehouseID := "wh-1"

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrdersReconciliation,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 1440,
		DefaultWarehouseID: &warehouseID,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_rejectsEnablingCategories(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityCategories,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity))
}

func TestUpsertSyncSchedule_rejectsInvalidEntityType(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: "bogus",
		Direction: channelsDomain.SyncDirectionImport, Enabled: false, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrInvalidSyncEntityType))
}

func TestUpsertSyncSchedule_rejectsInvalidDirection(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders,
		Direction: "sideways", Enabled: false, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrInvalidSyncDirection))
}

func TestUpsertSyncSchedule_rejectsNonPositiveInterval(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 0,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrInvalidSyncInterval))
}

func TestUpsertSyncSchedule_rejectsEnablingBolOrdersImportWithoutWarehouse(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolOrders,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_allowsEnablingBolOrdersImportWithWarehouseAndNoCurrency(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})
	warehouseID := "wh-1"

	s, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolOrders,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 60,
		DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
	require.True(t, s.Enabled)
}

func TestUpsertSyncSchedule_rejectsEnablingBolOrdersExport(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})
	warehouseID := "wh-1"

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolOrders,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 60,
		DefaultWarehouseID: &warehouseID,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_rejectsEnablingBolProductsImport(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolProducts,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_rejectsEnablingBolStockImport(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolStock,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 60,
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrCannotEnableUnrunnableEntity))
	require.Empty(t, repo.upserted)
}

func TestUpsertSyncSchedule_allowsEnablingBolProductsAndBolStockExportWithNoWarehouseOrCurrency(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 2), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolProducts,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 60,
	})
	require.NoError(t, err)

	_, err = uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 15,
	})
	require.NoError(t, err)
}

func TestUpsertSyncSchedule_propagatesRepoError(t *testing.T) {
	repo := &stubSyncScheduleRepo{upsertErr: errors.New("db down")}
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolStock,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 15,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}

// Woo products and stock have no schedule any more (COM-18): changes reach
// the store through events. Saving one is refused.
func TestUpsertSyncSchedule_rejectsTheRemovedWooProductsAndStockSchedules(t *testing.T) {
	for _, entity := range []channelsDomain.SyncEntityType{"products", "stock"} {
		uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), &stubSyncScheduleRepo{}, &eventrecordtest.Outbox{})

		_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
			ConnectionID: "conn-1", EntityType: entity, Direction: channelsDomain.SyncDirectionExport, IntervalMinutes: 60,
		})

		require.ErrorIs(t, err, channelsDomain.ErrInvalidSyncEntityType, string(entity))
	}
}

// Stock reconciliation (COM-18) pushes CommerceOS's stock to Woo: export
// only, no warehouse or currency needed.
func TestUpsertSyncSchedule_stockReconciliationIsExportOnly(t *testing.T) {
	uc := channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 1), &stubSyncScheduleRepo{}, &eventrecordtest.Outbox{})
	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStockReconciliation,
		Direction: channelsDomain.SyncDirectionExport, Enabled: true, IntervalMinutes: 1440,
	})
	require.NoError(t, err)

	uc = channelsapp.NewUpsertSyncScheduleUseCase(newMockGormDB(t, 0), &stubSyncScheduleRepo{}, &eventrecordtest.Outbox{})
	_, err = uc.Execute(context.Background(), channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStockReconciliation,
		Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 1440,
	})
	require.ErrorIs(t, err, channelsDomain.ErrCannotEnableUnrunnableEntity)
}
