package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubProductFinderForMissing struct {
	product *pmsDomain.Product
	err     error
}

func (s stubProductFinderForMissing) FindByID(context.Context, string) (*pmsDomain.Product, error) {
	return s.product, s.err
}

type missingFixture struct {
	listings  *stubListingRepo
	channels  *stubSalesChannelRepo
	overrides *stubSalesChannelProductOverrideRepo
	pusher    *stubProductPusher
	outbox    *eventrecordtest.Outbox
	product   pmsDomain.Product
}

func newMissingFixture() *missingFixture {
	conn := "conn-1"
	return &missingFixture{
		listings: &stubListingRepo{listByProduct: map[string][]channelsDomain.ProductChannelListing{"product-1": {
			{ID: "parent", ProductID: "product-1", ConnectionID: conn, Provider: "woocommerce", ExternalID: "23981", SyncStatus: channelsDomain.SyncStatusMissing},
			{ID: "bol", ProductID: "product-1", ConnectionID: "conn-bol", Provider: "bol", ExternalID: "8710001", SyncStatus: channelsDomain.SyncStatusSynced},
		}}},
		channels: &stubSalesChannelRepo{list: []channelsDomain.SalesChannel{
			{ID: "ch-woo", Type: channelsDomain.ChannelTypeWooCommerce, Active: true, ConnectionID: &conn, EnrollmentMode: channelsDomain.EnrollmentModeAll},
		}},
		overrides: &stubSalesChannelProductOverrideRepo{},
		pusher:    &stubProductPusher{},
		outbox:    &eventrecordtest.Outbox{},
		product:   pmsDomain.Product{ID: "product-1", Status: pmsDomain.ProductStatusActive},
	}
}

func (f *missingFixture) run(t *testing.T) error {
	t.Helper()
	uc := appchannels.NewHandleMissingListingUseCase(newMockGormDB(t, 1), f.listings, stubProductFinderForMissing{product: &f.product},
		f.channels, f.overrides, f.pusher, f.outbox)
	return uc.Execute(context.Background(), appchannels.HandleMissingListingInput{ProductID: "product-1", ConnectionID: "conn-1"})
}

func TestHandleMissingListing_recreatesAProductTheChannelStillAllows(t *testing.T) {
	f := newMissingFixture()

	require.NoError(t, f.run(t))

	assert.Equal(t, []string{"parent"}, f.listings.deleted, "only this connection's stale rows")
	deleted := eventsBySubject(f.outbox, "channels.product_listing.deleted")
	require.Len(t, deleted, 1)
	assert.Equal(t, "missing_in_channel", deleted[0].Data()["reason"])
	require.Len(t, f.pusher.calls, 1)
	assert.Equal(t, appchannels.PushProductToWooInput{ProductID: "product-1", ConnectionID: "conn-1"}, f.pusher.calls[0])
}

func TestHandleMissingListing_leavesItMarkedWhenTheChannelDoesNotAllowIt(t *testing.T) {
	excluded := &channelsDomain.SalesChannelProductOverride{Included: false}
	for name, change := range map[string]func(f *missingFixture){
		"product is a draft":         func(f *missingFixture) { f.product.Status = pmsDomain.ProductStatusDraft },
		"manually excluded":          func(f *missingFixture) { f.overrides.findResult = excluded },
		"channel enrolls by hand":    func(f *missingFixture) { f.channels.list[0].EnrollmentMode = channelsDomain.EnrollmentModeManual },
		"channel is inactive":        func(f *missingFixture) { f.channels.list[0].Active = false },
		"channel is another store":   func(f *missingFixture) { other := "conn-2"; f.channels.list[0].ConnectionID = &other },
		"channel is not woocommerce": func(f *missingFixture) { f.channels.list[0].Type = channelsDomain.ChannelTypeBol },
	} {
		t.Run(name, func(t *testing.T) {
			f := newMissingFixture()
			change(f)

			require.NoError(t, f.run(t))

			assert.Empty(t, f.listings.deleted)
			assert.Empty(t, f.pusher.calls)
		})
	}
}

func TestHandleMissingListing_nothingMissingIsANoOp(t *testing.T) {
	f := newMissingFixture()
	f.listings.listByProduct["product-1"][0].SyncStatus = channelsDomain.SyncStatusSynced

	require.NoError(t, f.run(t))

	assert.Empty(t, f.listings.deleted)
	assert.Empty(t, f.pusher.calls)
}

func TestHandleMissingListing_aFailedRecreateIsReturned(t *testing.T) {
	f := newMissingFixture()
	boom := errors.New("woo down")
	f.pusher.errFor = map[string]error{"product-1": boom}

	require.ErrorIs(t, f.run(t), boom)
}
