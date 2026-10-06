package channels_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestSetProductChannelOverride_add_setsIncludedTrue(t *testing.T) {
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	channelRepo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1"}}
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	uc := appchannels.NewSetProductChannelOverrideUseCase(newMockGormDB(t, 1), overrideRepo, channelRepo, productRepo, &eventrecordtest.Outbox{})

	override, err := uc.Execute(context.Background(), appchannels.SetProductChannelOverrideInput{
		ChannelID: "channel-1", ProductID: "product-1", Included: true,
	})

	require.NoError(t, err)
	require.Len(t, overrideRepo.upserted, 1)
	assert.True(t, override.Included)
}

func TestSetProductChannelOverride_remove_setsIncludedFalse(t *testing.T) {
	overrideRepo := &stubSalesChannelProductOverrideRepo{}
	channelRepo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1"}}
	productRepo := &stubChannelsProductRepo{byID: map[string]pmsDomain.Product{"product-1": {ID: "product-1"}}}
	uc := appchannels.NewSetProductChannelOverrideUseCase(newMockGormDB(t, 1), overrideRepo, channelRepo, productRepo, &eventrecordtest.Outbox{})

	override, err := uc.Execute(context.Background(), appchannels.SetProductChannelOverrideInput{
		ChannelID: "channel-1", ProductID: "product-1", Included: false,
	})

	require.NoError(t, err)
	assert.False(t, override.Included)
}

func TestSetProductChannelOverride_propagatesChannelNotFound(t *testing.T) {
	channelRepo := &stubSalesChannelRepo{findErr: channelsDomain.ErrSalesChannelNotFound}
	uc := appchannels.NewSetProductChannelOverrideUseCase(newMockGormDB(t, 1), &stubSalesChannelProductOverrideRepo{}, channelRepo, &stubChannelsProductRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.SetProductChannelOverrideInput{ChannelID: "missing", ProductID: "product-1"})

	assert.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
}

func TestSetProductChannelOverride_propagatesProductNotFound(t *testing.T) {
	channelRepo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1"}}
	uc := appchannels.NewSetProductChannelOverrideUseCase(newMockGormDB(t, 1), &stubSalesChannelProductOverrideRepo{}, channelRepo, &stubChannelsProductRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appchannels.SetProductChannelOverrideInput{ChannelID: "channel-1", ProductID: "missing"})

	assert.ErrorIs(t, err, pmsDomain.ErrProductNotFound)
}
