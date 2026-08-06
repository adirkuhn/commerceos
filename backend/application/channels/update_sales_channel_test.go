package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func existingWooChannel() *channelsDomain.SalesChannel {
	connID := "conn-1"
	return &channelsDomain.SalesChannel{
		ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true,
	}
}

func TestUpdateSalesChannel_changesToAllMode_isTheAddAllBulkAction(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo)

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.EnrollmentModeAll, channel.EnrollmentMode)
	require.Len(t, repo.updated, 1)
}

func TestUpdateSalesChannel_changesToTagMode_isTheAddByTagBulkAction(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo)

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID,
		EnrollmentMode: channelsDomain.EnrollmentModeTag, EnrollmentTag: "summer", Active: true,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.EnrollmentModeTag, channel.EnrollmentMode)
	assert.Equal(t, "summer", channel.EnrollmentTag)
}

func TestUpdateSalesChannel_propagatesNotFound(t *testing.T) {
	repo := &stubSalesChannelRepo{findErr: channelsDomain.ErrSalesChannelNotFound}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{ID: "missing", EnrollmentMode: channelsDomain.EnrollmentModeAll})

	assert.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
}

func TestUpdateSalesChannel_rejectsInvalidEnrollmentModeOnExistingChannel(t *testing.T) {
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, &stubConnectionRepo{})

	_, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", ConnectionID: strPtr("conn-1"), EnrollmentMode: channelsDomain.EnrollmentMode("bogus"),
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidEnrollmentMode)
}

func TestUpdateSalesChannel_canDeactivate(t *testing.T) {
	connID := "conn-1"
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}
	connRepo := &stubConnectionRepo{conn: &integrationsDomain.Connection{ID: connID}}
	uc := appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, 1), repo, connRepo)

	channel, err := uc.Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "My Store", ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: false,
	})

	require.NoError(t, err)
	assert.False(t, channel.Active)
}
