package channels_test

import (
	"context"
	"testing"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDismissFailedSync_success(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{}
	uc := channelsapp.NewDismissFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo)

	err := uc.Execute(context.Background(), "log-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"log-1"}, syncLogRepo.dismissedIDs)
}

func TestDismissFailedSync_notFound(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{dismissErr: channelsDomain.ErrSyncLogNotFound}
	uc := channelsapp.NewDismissFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo)

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, channelsDomain.ErrSyncLogNotFound)
}
