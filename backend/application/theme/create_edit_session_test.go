package theme_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appTheme "github.com/adirkuhn/commerceos/application/theme"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestCreateEditSession_mintsATokenScopedToTheChannel(t *testing.T) {
	channelRepo := &fakeChannelRepo{byID: map[string]channelsDomain.SalesChannel{"channel-1": {ID: "channel-1", TenantID: "tenant-1", Type: channelsDomain.ChannelTypeInternal}}}
	repo := &fakeEditSessionRepo{}
	uc := appTheme.NewCreateEditSessionUseCase(newMockGormDB(t, 1), repo, channelRepo)

	session, err := uc.Execute(context.Background(), "channel-1")

	require.NoError(t, err)
	require.NotEmpty(t, session.Token)
	require.Equal(t, "channel-1", session.SalesChannelID)
	require.True(t, session.ExpiresAt.After(time.Now()))
	require.Equal(t, session.Token, repo.created.Token)
}

func TestCreateEditSession_propagatesUnknownChannel(t *testing.T) {
	uc := appTheme.NewCreateEditSessionUseCase(newMockGormDB(t, 1), &fakeEditSessionRepo{}, &fakeChannelRepo{})

	_, err := uc.Execute(context.Background(), "missing-channel")

	require.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
}
