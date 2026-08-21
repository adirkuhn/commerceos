package channels_test

import (
	"context"
	"testing"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerSyncScheduleNow_pullsNextRunAtForward(t *testing.T) {
	repo := &stubSyncScheduleRepo{}
	uc := channelsapp.NewTriggerSyncScheduleNowUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "sched-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"sched-1"}, repo.triggeredIDs)
}

func TestTriggerSyncScheduleNow_propagatesNotFound(t *testing.T) {
	repo := &stubSyncScheduleRepo{triggerErr: channelsDomain.ErrSyncScheduleNotFound}
	uc := channelsapp.NewTriggerSyncScheduleNowUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, channelsDomain.ErrSyncScheduleNotFound)
}
