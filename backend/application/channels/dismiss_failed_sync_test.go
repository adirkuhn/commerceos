package channels_test

import (
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"context"
	"testing"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDismissFailedSync_success(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{}
	uc := channelsapp.NewDismissFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "log-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"log-1"}, syncLogRepo.dismissedIDs)
}

func TestDismissFailedSync_notFound(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{dismissErr: channelsDomain.ErrSyncLogNotFound}
	uc := channelsapp.NewDismissFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo, &eventrecordtest.Outbox{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, channelsDomain.ErrSyncLogNotFound)
}

func TestDismissFailedSync_RecordsDismissed(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := channelsapp.NewDismissFailedSyncUseCase(newMockGormDB(t, 1), &stubSyncLogRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "log-1"))

	e, err := outbox.Only()
	require.NoError(t, err)
	assert.Equal(t, "channels.sync_log_entry.dismissed", e.Subject)
	assert.Equal(t, "log-1", e.Payload["entity_id"])
}

func TestDismissFailedSync_OutboxErrorFails(t *testing.T) {
	boom := errors.New("outbox down")
	uc := channelsapp.NewDismissFailedSyncUseCase(newMockGormDB(t, 1), &stubSyncLogRepo{}, &eventrecordtest.Outbox{Err: boom})
	assert.ErrorIs(t, uc.Execute(context.Background(), "log-1"), boom)
}
