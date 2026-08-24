package channels_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func newRunScheduledBolExportUseCase(
	t *testing.T, n int, productRepo *stubChannelsProductRepo, channelRepo *stubSalesChannelRepo,
	overrideRepo *stubSalesChannelProductOverrideRepo, pusher *stubBolProductPusher,
) *appchannels.RunScheduledBolExportUseCase {
	t.Helper()
	return appchannels.NewRunScheduledBolExportUseCase(newMockGormDB(t, n), productRepo, channelRepo, overrideRepo, pusher)
}

func TestRunScheduledBolExport_pushesEnrolledProducts(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "prod-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeBol, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	pusher := &stubBolProductPusher{}
	uc := newRunScheduledBolExportUseCase(t, 2, productRepo, channelRepo, overrideRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledBolExportInput{ConnectionID: connID})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Pushed)
	assert.Equal(t, 0, result.Skipped)
	require.Len(t, pusher.calls, 1)
	assert.Equal(t, "prod-1", pusher.calls[0].ProductID)
	assert.Equal(t, connID, pusher.calls[0].ConnectionID)
}

func TestRunScheduledBolExport_skipsProductsNotEnrolled(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "prod-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeBol, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	pusher := &stubBolProductPusher{}
	uc := newRunScheduledBolExportUseCase(t, 2, productRepo, channelRepo, overrideRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledBolExportInput{ConnectionID: connID})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Pushed)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, pusher.calls)
}

func TestRunScheduledBolExport_manualOverrideIncludesAnOtherwiseUnenrolledProduct(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "prod-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeBol, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{findResult: &channelsDomain.SalesChannelProductOverride{Included: true}}
	pusher := &stubBolProductPusher{}
	uc := newRunScheduledBolExportUseCase(t, 2, productRepo, channelRepo, overrideRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledBolExportInput{ConnectionID: connID})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Pushed)
	require.Len(t, pusher.calls, 1)
}

func TestRunScheduledBolExport_pushesAlreadyEnrolledProductsAgainToRefreshStock(t *testing.T) {
	// Unlike Woo's scheduled export (only ever pushes products with no
	// listing yet), this must also re-push already-linked enrolled
	// products -- Bol has no separate stock-only endpoint, so refreshing
	// stock IS re-pushing the full offer. PushProductToBolUseCase itself
	// (not exercised here, just its narrow bolProductPusher interface)
	// already handles create-vs-update internally.
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "prod-1"}, {ID: "prod-2"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeBol, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	pusher := &stubBolProductPusher{}
	uc := newRunScheduledBolExportUseCase(t, 3, productRepo, channelRepo, overrideRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledBolExportInput{ConnectionID: connID})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Pushed, "every enrolled product is pushed every run, regardless of whether it already has a listing")
	require.Len(t, pusher.calls, 2)
}

func TestRunScheduledBolExport_noMatchingChannelReportsAClearErrorInsteadOfSilentSuccess(t *testing.T) {
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "prod-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	pusher := &stubBolProductPusher{}
	uc := newRunScheduledBolExportUseCase(t, 1, productRepo, channelRepo, overrideRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledBolExportInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, result.Pushed)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "no active Bol sales channel")
	assert.Empty(t, pusher.calls)
}

func TestRunScheduledBolExport_ignoresChannelsForOtherConnectionsAndProviders(t *testing.T) {
	otherConn, wooConn := "conn-other", "conn-woo"
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "prod-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeBol, ConnectionID: &otherConn, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
		{ID: "channel-2", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &wooConn, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	pusher := &stubBolProductPusher{}
	uc := newRunScheduledBolExportUseCase(t, 1, productRepo, channelRepo, overrideRepo, pusher)

	result, err := uc.Execute(context.Background(), appchannels.RunScheduledBolExportInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "no active Bol sales channel")
	assert.Empty(t, pusher.calls)
}
