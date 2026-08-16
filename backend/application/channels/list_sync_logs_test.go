package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestListSyncLogs_returnsEntriesForTheListing(t *testing.T) {
	listingRepo := &stubListingRepo{byProduct: map[string]channelsDomain.ProductChannelListing{
		"prod-1": {ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1"},
	}}
	syncLogRepo := &stubSyncLogRepo{byListing: map[string][]channelsDomain.SyncLogEntry{
		"listing-1": {{ID: "log-2", ListingID: strPtr("listing-1"), Status: channelsDomain.SyncLogStatusSuccess}, {ID: "log-1", ListingID: strPtr("listing-1"), Status: channelsDomain.SyncLogStatusError}},
	}}
	uc := appchannels.NewListSyncLogsUseCase(newMockGormDB(t, 1), listingRepo, syncLogRepo)

	entries, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "log-2", entries[0].ID)
}

func TestListSyncLogs_propagatesListingNotFound(t *testing.T) {
	listingRepo := &stubListingRepo{}
	syncLogRepo := &stubSyncLogRepo{}
	uc := appchannels.NewListSyncLogsUseCase(newMockGormDB(t, 1), listingRepo, syncLogRepo)

	_, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.Error(t, err)
}
