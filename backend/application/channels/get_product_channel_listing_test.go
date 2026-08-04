package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestGetProductChannelListing_returnsTheListing(t *testing.T) {
	repo := &stubListingRepo{
		byProduct: map[string]channelsDomain.ProductChannelListing{
			"prod-1": {ID: "listing-1", ProductID: "prod-1", ConnectionID: "conn-1", ExternalID: "23609", SyncStatus: channelsDomain.SyncStatusSynced},
		},
	}
	uc := appchannels.NewGetProductChannelListingUseCase(newMockGormDB(t, 1), repo)

	listing, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.NoError(t, err)
	require.Equal(t, "23609", listing.ExternalID)
	require.Equal(t, channelsDomain.SyncStatusSynced, listing.SyncStatus)
}

func TestGetProductChannelListing_propagatesNotFound(t *testing.T) {
	repo := &stubListingRepo{}
	uc := appchannels.NewGetProductChannelListingUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "prod-1", "conn-1")

	require.Error(t, err)
	require.True(t, errors.Is(err, channelsDomain.ErrListingNotFound))
}
