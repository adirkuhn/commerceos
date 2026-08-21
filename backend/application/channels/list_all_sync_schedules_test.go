package channels_test

import (
	"context"
	"testing"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAllSyncSchedules_enrichesEachRowWithItsConnectionStoreURL(t *testing.T) {
	scheduleRepo := &stubSyncScheduleRepo{all: []channelsDomain.SyncSchedule{
		{ID: "sched-1", ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport},
	}}
	connRepo := &stubConnectionRepo{allByTenant: []integrationsDomain.Connection{
		{ID: "conn-1", TenantID: "tenant-1", StoreURL: "https://mystore.com"},
	}}
	uc := channelsapp.NewListAllSyncSchedulesUseCase(newMockGormDB(t, 1), scheduleRepo, connRepo)

	rows, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "sched-1", rows[0].Schedule.ID)
	assert.Equal(t, "https://mystore.com", rows[0].ConnectionStoreURL)
}

func TestListAllSyncSchedules_leavesStoreURLEmptyWhenConnectionIsGone(t *testing.T) {
	scheduleRepo := &stubSyncScheduleRepo{all: []channelsDomain.SyncSchedule{
		{ID: "sched-1", ConnectionID: "deleted-conn"},
	}}
	connRepo := &stubConnectionRepo{}
	uc := channelsapp.NewListAllSyncSchedulesUseCase(newMockGormDB(t, 1), scheduleRepo, connRepo)

	rows, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Empty(t, rows[0].ConnectionStoreURL)
}
