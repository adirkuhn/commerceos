package channels_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/locale"
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

func newCreateChannelUCWithCompany(t *testing.T, txs int, repo *stubSalesChannelRepo, company *stubTenantSettings) *appchannels.CreateSalesChannelUseCase {
	return appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, txs), repo, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, company, &eventrecordtest.Outbox{})
}

func TestCreateSalesChannel_storefrontLocaleDefaultsToTheCompanyLanguage(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	ch, err := newCreateChannelUCWithCompany(t, 1, repo, &stubTenantSettings{language: locale.Portuguese}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Loja", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})
	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontLocalePortuguese, ch.StorefrontLocale)
}

func TestCreateSalesChannel_chosenStorefrontLocaleBeatsTheCompanyLanguage(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	ch, err := newCreateChannelUCWithCompany(t, 1, repo, &stubTenantSettings{language: locale.Portuguese}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Winkel", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontLocale: channelsDomain.StorefrontLocaleDutch,
	})
	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontLocaleDutch, ch.StorefrontLocale)
}

func TestCreateSalesChannel_companySettingsErrorFailsTheCreate(t *testing.T) {
	repo := &stubSalesChannelRepo{}
	_, err := newCreateChannelUCWithCompany(t, 1, repo, &stubTenantSettings{err: errors.New("db down")}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})
	require.Error(t, err)
	assert.Empty(t, repo.created)
}
