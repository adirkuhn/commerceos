package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestDeleteSalesChannel_success(t *testing.T) {
	repo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1"}}
	uc := appchannels.NewDeleteSalesChannelUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "channel-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"channel-1"}, repo.deleted)
}

func TestDeleteSalesChannel_propagatesNotFound(t *testing.T) {
	repo := &stubSalesChannelRepo{findErr: channelsDomain.ErrSalesChannelNotFound}
	uc := appchannels.NewDeleteSalesChannelUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
	assert.Empty(t, repo.deleted)
}
