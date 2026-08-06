package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestListSalesChannels_returnsTenantChannels(t *testing.T) {
	repo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{{ID: "channel-1", Name: "My Store"}}}
	uc := appchannels.NewListSalesChannelsUseCase(newMockGormDB(t, 1), repo)

	channels, err := uc.Execute(context.Background())

	require.NoError(t, err)
	require.Len(t, channels, 1)
	require.Equal(t, "My Store", channels[0].Name)
}
