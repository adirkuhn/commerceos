package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

func newCreateChannelUC(t *testing.T, txs int, repo *stubSalesChannelRepo) *appchannels.CreateSalesChannelUseCase {
	return appchannels.NewCreateSalesChannelUseCase(newMockGormDB(t, txs), repo, &stubConnectionRepo{}, &stubExchangeRateRepo{}, &stubWarehouseExistenceChecker{}, &stubTenantSettings{})
}

func TestCreateSalesChannel_defaultsStorefrontModeToHeadless(t *testing.T) {
	repo := &stubSalesChannelRepo{}

	channel, err := newCreateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontModeHeadless, channel.StorefrontMode)
	require.Len(t, repo.created, 1)
	assert.Equal(t, channelsDomain.StorefrontModeHeadless, repo.created[0].StorefrontMode)
}

func TestCreateSalesChannel_internalChannelCanUseThemedStorefront(t *testing.T) {
	repo := &stubSalesChannelRepo{}

	channel, err := newCreateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontMode: channelsDomain.StorefrontModeTheme,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontModeTheme, channel.StorefrontMode)
}

func TestCreateSalesChannel_rejectsThemedStorefrontOnExternalChannel(t *testing.T) {
	_, err := newCreateChannelUC(t, 0, &stubSalesChannelRepo{}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Woo", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: strPtr("conn-1"),
		EnrollmentMode: channelsDomain.EnrollmentModeAll, StorefrontMode: channelsDomain.StorefrontModeTheme,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrThemedStorefrontRequiresInternal)
}

func TestCreateSalesChannel_rejectsUnknownStorefrontMode(t *testing.T) {
	_, err := newCreateChannelUC(t, 0, &stubSalesChannelRepo{}).Execute(context.Background(), appchannels.CreateSalesChannelInput{
		Name: "Shop", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontMode: channelsDomain.StorefrontMode("spa"),
	})

	assert.ErrorIs(t, err, channelsDomain.ErrInvalidStorefrontMode)
}

func existingInternalChannel(mode channelsDomain.StorefrontMode) *channelsDomain.SalesChannel {
	return &channelsDomain.SalesChannel{
		ID: "channel-1", Type: channelsDomain.ChannelTypeInternal, EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontMode: mode, Active: true,
	}
}

func newUpdateChannelUC(t *testing.T, txs int, repo *stubSalesChannelRepo) *appchannels.UpdateSalesChannelUseCase {
	return appchannels.NewUpdateSalesChannelUseCase(newMockGormDB(t, txs), repo, &stubConnectionRepo{}, &stubWarehouseExistenceChecker{})
}

func TestUpdateSalesChannel_switchesInternalChannelToThemedStorefront(t *testing.T) {
	repo := &stubSalesChannelRepo{findResult: existingInternalChannel(channelsDomain.StorefrontModeHeadless)}

	channel, err := newUpdateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true,
		StorefrontMode: channelsDomain.StorefrontModeTheme,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontModeTheme, channel.StorefrontMode)
	require.Len(t, repo.updated, 1)
	assert.Equal(t, channelsDomain.StorefrontModeTheme, repo.updated[0].StorefrontMode)
}

// Omitting storefront_mode on update keeps the current mode, so older
// clients that never send it can't silently switch a live themed store
// back to headless.
func TestUpdateSalesChannel_omittedStorefrontModeKeepsCurrent(t *testing.T) {
	repo := &stubSalesChannelRepo{findResult: existingInternalChannel(channelsDomain.StorefrontModeTheme)}

	channel, err := newUpdateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", Name: "Shop", EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true,
	})

	require.NoError(t, err)
	assert.Equal(t, channelsDomain.StorefrontModeTheme, channel.StorefrontMode)
}

func TestUpdateSalesChannel_rejectsThemedStorefrontOnExternalChannel(t *testing.T) {
	repo := &stubSalesChannelRepo{findResult: existingWooChannel()}

	_, err := newUpdateChannelUC(t, 1, repo).Execute(context.Background(), appchannels.UpdateSalesChannelInput{
		ID: "channel-1", ConnectionID: strPtr("conn-1"), EnrollmentMode: channelsDomain.EnrollmentModeAll,
		StorefrontMode: channelsDomain.StorefrontModeTheme,
	})

	assert.ErrorIs(t, err, channelsDomain.ErrThemedStorefrontRequiresInternal)
	assert.Empty(t, repo.updated)
}
