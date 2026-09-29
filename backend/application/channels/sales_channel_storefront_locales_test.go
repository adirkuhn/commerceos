package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

var (
	en = channelsDomain.StorefrontLocaleEnglish
	pt = channelsDomain.StorefrontLocalePortuguese
	nl = channelsDomain.StorefrontLocaleDutch
)

func TestCreateSalesChannel_offersTheDefaultLanguageAndTheChosenOthers(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	ch, err := newCreateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Loja", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontLocale: pt, StorefrontLocales: []channelsDomain.StorefrontLocale{en, nl},
	})
	require.NoError(t, err)
	assert.Equal(t, []channelsDomain.StorefrontLocale{pt, en, nl}, repo.created[0].StorefrontLocales)
	assert.Equal(t, pt, ch.StorefrontLocale)
}

func TestCreateSalesChannel_noOtherLanguagesOffersOnlyTheDefault(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	_, err := newCreateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})
	require.NoError(t, err)
	assert.Equal(t, []channelsDomain.StorefrontLocale{en}, repo.created[0].StorefrontLocales)
}

func TestCreateSalesChannel_rejectsUnsupportedOfferedLanguage(t *testing.T) {
	_, err := newCreateChannelUC(t, 0, &stubSalesChannelRepo{}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontLocale: en, StorefrontLocales: []channelsDomain.StorefrontLocale{"fr"},
	})
	assert.ErrorIs(t, err, channelsDomain.ErrInvalidStorefrontLocale)
}

func updateLocales(t *testing.T, existing *channelsDomain.SalesChannel, defaultLocale channelsDomain.StorefrontLocale, locales *[]channelsDomain.StorefrontLocale) (*channelsDomain.SalesChannel, error) {
	return newUpdateChannelUC(t, 1, &stubSalesChannelRepo{findResult: existing}).Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true,
		StorefrontLocale: defaultLocale, StorefrontLocales: locales,
	})
}

func multilingualChannel() *channelsDomain.SalesChannel {
	c := existingInternalChannel(channelsDomain.StorefrontModeTheme)
	c.StorefrontLocale = nl
	c.StorefrontLocales = []channelsDomain.StorefrontLocale{nl, en}
	return c
}

func TestUpdateSalesChannel_omittedLanguagesKeepsThem(t *testing.T) {
	ch, err := updateLocales(t, multilingualChannel(), "", nil)
	require.NoError(t, err)
	assert.Equal(t, []channelsDomain.StorefrontLocale{nl, en}, ch.StorefrontLocales)
}

func TestUpdateSalesChannel_setsTheOfferedLanguages(t *testing.T) {
	ch, err := updateLocales(t, multilingualChannel(), "", &[]channelsDomain.StorefrontLocale{pt})
	require.NoError(t, err)
	assert.Equal(t, []channelsDomain.StorefrontLocale{nl, pt}, ch.StorefrontLocales)
}

func TestUpdateSalesChannel_newDefaultLanguageMovesToTheFront(t *testing.T) {
	ch, err := updateLocales(t, multilingualChannel(), en, nil)
	require.NoError(t, err)
	assert.Equal(t, en, ch.StorefrontLocale)
	assert.Equal(t, []channelsDomain.StorefrontLocale{en, nl}, ch.StorefrontLocales)
}

func TestUpdateSalesChannel_legacyChannelGainsALanguage(t *testing.T) {
	c := existingInternalChannel(channelsDomain.StorefrontModeTheme)
	c.StorefrontLocale = pt
	ch, err := updateLocales(t, c, "", &[]channelsDomain.StorefrontLocale{nl})
	require.NoError(t, err)
	assert.Equal(t, []channelsDomain.StorefrontLocale{pt, nl}, ch.StorefrontLocales)
}

func TestUpdateSalesChannel_rejectsUnsupportedOfferedLanguage(t *testing.T) {
	_, err := updateLocales(t, multilingualChannel(), "", &[]channelsDomain.StorefrontLocale{"de"})
	assert.ErrorIs(t, err, channelsDomain.ErrInvalidStorefrontLocale)
}
