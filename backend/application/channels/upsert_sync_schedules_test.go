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

func validSyncScheduleInput(entity channelsDomain.SyncEntityType, direction channelsDomain.SyncDirection) channelsapp.UpsertSyncScheduleInput {
	return channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: entity, Direction: direction, Enabled: false, IntervalMinutes: 60,
	}
}

func TestUpsertSyncSchedules_persistsEveryRowInOneTransactionWhenAllValid(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncSchedulesUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	schedules, err := uc.Execute(context.Background(), channelsapp.UpsertSyncSchedulesInput{
		Schedules: []channelsapp.UpsertSyncScheduleInput{
			validSyncScheduleInput(channelsDomain.SyncEntityOrders, channelsDomain.SyncDirectionImport),
			validSyncScheduleInput(channelsDomain.SyncEntityBolProducts, channelsDomain.SyncDirectionExport),
			validSyncScheduleInput(channelsDomain.SyncEntityOrdersReconciliation, channelsDomain.SyncDirectionImport),
		},
	})

	require.NoError(t, err)
	require.Len(t, schedules, 3)
	require.Len(t, repo.upserted, 3)
}

// The real bug this use case fixes: the old per-row handler loop aborted
// on the first invalid row, silently leaving every row already processed
// in that request persisted -- so a user editing only the Orders row could
// have their change dropped by an unrelated row's stale validation error,
// with no clear explanation. Validating everything up front, with zero
// writes, before persisting anything means either the whole PUT succeeds
// or nothing in it does.
func TestUpsertSyncSchedules_writesNothingWhenAnyRowFailsValidation(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncSchedulesUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	invalid := channelsapp.UpsertSyncScheduleInput{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
		Enabled: true, IntervalMinutes: 60, // enabled orders/import with no warehouse
	}

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncSchedulesInput{
		Schedules: []channelsapp.UpsertSyncScheduleInput{
			validSyncScheduleInput(channelsDomain.SyncEntityOrdersReconciliation, channelsDomain.SyncDirectionImport),
			invalid,
			validSyncScheduleInput(channelsDomain.SyncEntityOrders, channelsDomain.SyncDirectionImport),
		},
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrDefaultWarehouseRequired))
	require.Empty(t, repo.upserted, "no row -- including the valid ones before the bad one -- should ever be written")
}

func TestUpsertSyncSchedules_errorNamesTheFailingRow(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncSchedulesUseCase(newMockGormDB(t, 0), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncSchedulesInput{
		Schedules: []channelsapp.UpsertSyncScheduleInput{
			validSyncScheduleInput(channelsDomain.SyncEntityOrders, channelsDomain.SyncDirectionImport),
			{ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport, Enabled: true, IntervalMinutes: 60},
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "orders")
	require.ErrorContains(t, err, "import")
}

func TestUpsertSyncSchedules_propagatesRepoErrorAndWritesNothingElseInThatTransaction(t *testing.T) {
	repo := &stubSyncScheduleRepo{upsertErr: errors.New("db down")}
	uc := channelsapp.NewUpsertSyncSchedulesUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), channelsapp.UpsertSyncSchedulesInput{
		Schedules: []channelsapp.UpsertSyncScheduleInput{
			validSyncScheduleInput(channelsDomain.SyncEntityOrders, channelsDomain.SyncDirectionImport),
		},
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "db down")
}

func TestUpsertSyncSchedules_emptyBatchIsANoOp(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewUpsertSyncSchedulesUseCase(newMockGormDB(t, 1), repo, &eventrecordtest.Outbox{})

	schedules, err := uc.Execute(context.Background(), channelsapp.UpsertSyncSchedulesInput{})

	require.NoError(t, err)
	require.Empty(t, schedules)
	require.Empty(t, repo.upserted)
}
