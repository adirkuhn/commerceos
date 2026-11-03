package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestListSyncSchedules_returnsSchedulesForConnection(t *testing.T) {
	repo := &stubSyncScheduleRepo{byConnection: []channelsDomain.SyncSchedule{
		{ID: "s1", ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport},
	}}
	uc := channelsapp.NewListSyncSchedulesUseCase(newMockGormDB(t, 1), repo)

	result, err := uc.Execute(context.Background(), "conn-1")

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, channelsDomain.SyncEntityOrders, result[0].EntityType)
}
