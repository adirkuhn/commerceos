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

func TestListProductsWithChannelEnrollment_allMode_marksEveryProductEnrolled(t *testing.T) {
	channelRepo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1", EnrollmentMode: channelsDomain.EnrollmentModeAll}}
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "p1"}, {ID: "p2"}}}
	uc := appchannels.NewListProductsWithChannelEnrollmentUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{})

	rows, total, err := uc.Execute(context.Background(), appchannels.ListProductsWithChannelEnrollmentInput{ChannelID: "channel-1"})

	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, rows, 2)
	assert.True(t, rows[0].Enrolled)
	assert.True(t, rows[1].Enrolled)
}

func TestListProductsWithChannelEnrollment_manualMode_marksNothingEnrolledWithoutOverrides(t *testing.T) {
	channelRepo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1", EnrollmentMode: channelsDomain.EnrollmentModeManual}}
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "p1"}}}
	uc := appchannels.NewListProductsWithChannelEnrollmentUseCase(newMockGormDB(t, 1), productRepo, channelRepo, &stubSalesChannelProductOverrideRepo{})

	rows, _, err := uc.Execute(context.Background(), appchannels.ListProductsWithChannelEnrollmentInput{ChannelID: "channel-1"})

	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Enrolled)
}

func TestListProductsWithChannelEnrollment_appliesPerProductOverrides(t *testing.T) {
	channelRepo := &stubSalesChannelRepo{findResult: &channelsDomain.SalesChannel{ID: "channel-1", EnrollmentMode: channelsDomain.EnrollmentModeManual}}
	productRepo := &stubChannelsProductRepo{listResult: []pmsDomain.Product{{ID: "p1"}, {ID: "p2"}}}
	overrideRepo := &stubSalesChannelProductOverrideRepo{byChannel: []channelsDomain.SalesChannelProductOverride{
		{ChannelID: "channel-1", ProductID: "p1", Included: true},
	}}
	uc := appchannels.NewListProductsWithChannelEnrollmentUseCase(newMockGormDB(t, 1), productRepo, channelRepo, overrideRepo)

	rows, _, err := uc.Execute(context.Background(), appchannels.ListProductsWithChannelEnrollmentInput{ChannelID: "channel-1"})

	require.NoError(t, err)
	byID := map[string]bool{}
	for _, r := range rows {
		byID[r.Product.ID] = r.Enrolled
	}
	assert.True(t, byID["p1"], "p1 has an include override")
	assert.False(t, byID["p2"], "p2 has no override, channel default is manual")
}

func TestListProductsWithChannelEnrollment_propagatesChannelNotFound(t *testing.T) {
	channelRepo := &stubSalesChannelRepo{findErr: channelsDomain.ErrSalesChannelNotFound}
	uc := appchannels.NewListProductsWithChannelEnrollmentUseCase(newMockGormDB(t, 1), &stubChannelsProductRepo{}, channelRepo, &stubSalesChannelProductOverrideRepo{})

	_, _, err := uc.Execute(context.Background(), appchannels.ListProductsWithChannelEnrollmentInput{ChannelID: "missing"})

	assert.ErrorIs(t, err, channelsDomain.ErrSalesChannelNotFound)
}
