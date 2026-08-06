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

func TestEnrollProductInChannels_allMode_pushesToTheChannel(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{}, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	require.Len(t, pusher.calls, 1)
	assert.Equal(t, "product-1", pusher.calls[0].ProductID)
	assert.Equal(t, connID, pusher.calls[0].ConnectionID)
}

func TestEnrollProductInChannels_manualMode_doesNotPush(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true},
	}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{}, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	assert.Empty(t, pusher.calls)
}

func TestEnrollProductInChannels_categoryMode_matchesOnlySameCategory(t *testing.T) {
	connID, catID := "conn-1", "cat-1"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1", CategoryID: &catID}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeCategory, EnrollmentCategoryID: &catID, Active: true},
	}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{}, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	require.Len(t, pusher.calls, 1)
}

func TestEnrollProductInChannels_tagMode_matchesOnlyTaggedProduct(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1", Tags: []string{"summer"}}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeTag, EnrollmentTag: "summer", Active: true},
	}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{}, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	require.Len(t, pusher.calls, 1)
}

func TestEnrollProductInChannels_excludeOverride_winsOverAllMode(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{findResult: &channelsDomain.SalesChannelProductOverride{Included: false}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, overrideRepo, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	assert.Empty(t, pusher.calls)
}

func TestEnrollProductInChannels_includeOverride_winsOverManualMode(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeManual, Active: true},
	}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{findResult: &channelsDomain.SalesChannelProductOverride{Included: true}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, overrideRepo, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	require.Len(t, pusher.calls, 1)
}

func TestEnrollProductInChannels_skipsInactiveChannels(t *testing.T) {
	connID := "conn-1"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &connID, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: false},
	}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{}, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	assert.Empty(t, pusher.calls)
}

func TestEnrollProductInChannels_skipsChannelsWithNoConnection(t *testing.T) {
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeInternal, ConnectionID: nil, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	pusher := &stubProductPusher{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{}, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err)
	assert.Empty(t, pusher.calls)
}

func TestEnrollProductInChannels_onePushFailureDoesNotBlockOthers(t *testing.T) {
	conn1, conn2 := "conn-1", "conn-2"
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	channelRepo := &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
		{ID: "channel-1", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &conn1, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
		{ID: "channel-2", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: &conn2, EnrollmentMode: channelsDomain.EnrollmentModeAll, Active: true},
	}}
	// stubProductPusher's errFor is keyed by ProductID, not per-channel, so
	// both calls below fail -- what this proves is that the first failure
	// doesn't abort the loop before the second channel is even attempted.
	pusher := &stubProductPusher{errFor: map[string]error{"product-1": assert.AnError}}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{}, pusher)

	err := uc.Execute(context.Background(), "product-1")

	require.NoError(t, err, "a per-channel push failure must not fail the whole use case")
	require.Len(t, pusher.calls, 2, "both channels must still be attempted")
}

func TestEnrollProductInChannels_propagatesProductNotFound(t *testing.T) {
	productRepo := &stubChannelsProductRepo{}
	uc := appchannels.NewEnrollProductInChannelsUseCase(newMockGormDB(t, 1), productRepo, &stubSalesChannelRepo{}, &stubSalesChannelProductOverrideRepo{}, &stubProductPusher{})

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, pmsDomain.ErrProductNotFound)
}
