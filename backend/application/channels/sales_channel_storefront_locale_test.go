package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func TestCreateSalesChannel_storefrontLocaleDefaultsToEnglish(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	_, err := newCreateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})
	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontLocaleEnglish, repo.created[0].StorefrontLocale)
}

func TestCreateSalesChannel_storesTheStorefrontLocale(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	ch, err := newCreateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Loja", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontLocale: channelsDomain.StorefrontLocalePortuguese,
	})
	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontLocalePortuguese, ch.StorefrontLocale)
}

func TestCreateSalesChannel_rejectsUnknownStorefrontLocale(t *testing.T) {
	_, err := newCreateChannelUC(t, 0, &stubSalesChannelRepo{}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontLocale: "de",
	})
	assert.ErrorIs(t, err, channelsDomain.ErrInvalidStorefrontLocale)
}

func TestUpdateSalesChannel_storefrontLocale_omittedKeeps_valueSets(t *testing.T) {
	existing := func() *channelsDomain.SalesChannel {
		c := existingInternalChannel(channelsDomain.StorefrontModeTheme)
		c.StorefrontLocale = channelsDomain.StorefrontLocaleDutch
		return c
	}
	update := func(l channelsDomain.StorefrontLocale) (*channelsDomain.SalesChannel, error) {
		return newUpdateChannelUC(t, 1, &stubSalesChannelRepo{findResult: existing()}).Execute(context.Background(), appchannels.UpdateSalesChannelInput{
			ID: "channel-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true, StorefrontLocale: l,
		})
	}
	kept, err := update("")
	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontLocaleDutch, kept.StorefrontLocale)
	set, err := update(channelsDomain.StorefrontLocalePortuguese)
	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontLocalePortuguese, set.StorefrontLocale)
	_, err = update("xx")
	assert.ErrorIs(t, err, channelsDomain.ErrInvalidStorefrontLocale)
}
