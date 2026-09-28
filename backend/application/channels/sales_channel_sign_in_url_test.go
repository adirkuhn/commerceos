package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestCreateSalesChannel_storesTheHeadlessSignInPage(t *testing.T) {
	repo := &stubSalesChannelRepo{}

	channel, err := newCreateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		SignInURL: strPtr("https://shop.example.com/verify"),
	})

	require.NoError(t, err)
	require.NotNil(t, channel.SignInURL)
	assert.Equal(t, "https://shop.example.com/verify", *repo.created[0].SignInURL)
}

func TestCreateSalesChannel_rejectsAnUnsafeSignInPage(t *testing.T) {
	_, err := newCreateChannelUC(t, 0, &stubSalesChannelRepo{}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		SignInURL: strPtr("javascript:alert(1)"),
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidSignInURL)
}

func TestUpdateSalesChannel_signInPage_omittedKeeps_blankClears_valueSets(t *testing.T) {
	current := "https://shop.example.com/old"
	existing := func() *channelsDomain.SalesChannel {
		c := existingInternalChannel(channelsDomain.StorefrontModeHeadless)
		c.SignInURL = &current
		return c
	}
	update := func(signIn *string) *channelsDomain.SalesChannel {
		repo := &stubSalesChannelRepo{findResult: existing()}
		ch, err := newUpdateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.UpdateSalesChannelInput{
			ID: "channel-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true, SignInURL: signIn,
		})
		require.NoError(t, err)
		return ch
	}

	assert.Equal(t, current, *update(nil).SignInURL)
	assert.Nil(t, update(strPtr("")).SignInURL)
	assert.Equal(t, "https://shop.example.com/new", *update(strPtr("https://shop.example.com/new")).SignInURL)
}

func TestUpdateSalesChannel_rejectsSignInPageOnExternalChannel(t *testing.T) {
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}

	_, err := newUpdateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", ConnectionID: strPtr("conn-1"), EnrollmentMode: channelsDomain.EnrollmentModeAll,
		SignInURL: strPtr("https://shop.example.com/verify"),
	})

	assert.ErrorIs(t, err, channelsDomain.ErrSignInURLRequiresInternal)
	assert.Empty(t, repo.updated)
}
