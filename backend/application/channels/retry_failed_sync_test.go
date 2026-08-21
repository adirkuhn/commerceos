package channels_test

import (
	"context"
	"testing"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryFailedSync_rePushesTheProductToTheSameConnection(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{byID: map[string]channelsDomain.SyncLogEntry{
		"log-1": {ID: "log-1", ProductID: "prod-1", ConnectionID: "conn-1", ListingID: strPtr("listing-1"), Status: channelsDomain.SyncLogStatusError},
	}}
	pusher := &stubProductPusher{}
	uc := channelsapp.NewRetryFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo, pusher)

	err := uc.Execute(context.Background(), "log-1")

	require.NoError(t, err)
	require.Len(t, pusher.calls, 1)
	assert.Equal(t, "prod-1", pusher.calls[0].ProductID)
	assert.Equal(t, "conn-1", pusher.calls[0].ConnectionID)
}

func TestRetryFailedSync_notFound(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{}
	uc := channelsapp.NewRetryFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo, &stubProductPusher{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, channelsDomain.ErrSyncLogNotFound)
}

func TestRetryFailedSync_rejectsAnEntryWhoseListingIsGone(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{byID: map[string]channelsDomain.SyncLogEntry{
		"log-1": {ID: "log-1", ProductID: "prod-1", ConnectionID: "conn-1", ListingID: nil},
	}}
	pusher := &stubProductPusher{}
	uc := channelsapp.NewRetryFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo, pusher)

	err := uc.Execute(context.Background(), "log-1")

	assert.ErrorIs(t, err, channelsDomain.ErrSyncLogHasNoListing)
	assert.Empty(t, pusher.calls)
}

func TestRetryFailedSync_propagatesAPushError(t *testing.T) {
	syncLogRepo := &stubSyncLogRepo{byID: map[string]channelsDomain.SyncLogEntry{
		"log-1": {ID: "log-1", ProductID: "prod-1", ConnectionID: "conn-1", ListingID: strPtr("listing-1")},
	}}
	pusher := &stubProductPusher{errFor: map[string]error{"prod-1": assert.AnError}}
	uc := channelsapp.NewRetryFailedSyncUseCase(newMockGormDB(t, 1), syncLogRepo, pusher)

	err := uc.Execute(context.Background(), "log-1")

	assert.ErrorIs(t, err, assert.AnError)
}
