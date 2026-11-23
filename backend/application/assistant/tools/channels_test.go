package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

type stubChannels struct {
	out []channelsDomain.SalesChannel
	err error
}

func (s *stubChannels) Execute(context.Context) ([]channelsDomain.SalesChannel, error) {
	return s.out, s.err
}

type stubListings struct {
	byConn map[string][]channelsDomain.ProductChannelListing
	err    error
	calls  []string
}

func (s *stubListings) Execute(_ context.Context, productID, connectionID string) ([]channelsDomain.ProductChannelListing, error) {
	s.calls = append(s.calls, productID+"@"+connectionID)
	return s.byConn[connectionID], s.err
}

func channels() []channelsDomain.SalesChannel {
	return []channelsDomain.SalesChannel{
		{ID: "ch-woo", Name: "Webshop", Type: channelsDomain.ChannelTypeWooCommerce, ConnectionID: ptr("conn-woo"), Active: true, CurrencyCode: "EUR", EnrollmentMode: channelsDomain.EnrollmentModeAll},
		{ID: "ch-bol", Name: "Bol", Type: channelsDomain.ChannelTypeBol, ConnectionID: ptr("conn-bol"), Active: true},
		{ID: "ch-store", Name: "B2B store", Type: channelsDomain.ChannelTypeInternal, Active: false},
	}
}

func TestListSalesChannelsTool(t *testing.T) {
	tool := tools.NewListSalesChannelsTool(&stubChannels{out: channels()})
	assertStrictDefinition(t, tool, "list_sales_channels")

	got := call(t, tool, `{}`)

	list := got["sales_channels"].([]any)
	require.Len(t, list, 3)
	woo := list[0].(map[string]any)
	assert.Equal(t, map[string]any{"id": "ch-woo", "name": "Webshop", "type": "woocommerce", "active": true,
		"currency_code": "EUR", "enrollment_mode": "all", "connected": true}, woo)
	assert.Equal(t, false, list[2].(map[string]any)["connected"])

	assert.ErrorContains(t, callErr(t, tools.NewListSalesChannelsTool(&stubChannels{err: errors.New("db down")}), `{}`), "db down")
	assert.ErrorIs(t, callErr(t, tool, `{"x":1}`), tools.ErrInvalidArguments)
}

func TestListProductChannelListingsTool(t *testing.T) {
	synced := time.Date(2026, 11, 22, 8, 30, 0, 0, time.UTC)
	listings := &stubListings{byConn: map[string][]channelsDomain.ProductChannelListing{
		"conn-woo": {{ProductID: "p-1", Provider: "woocommerce", ExternalID: "4710", SyncStatus: channelsDomain.SyncStatusSynced, LastSyncedAt: &synced}},
		"conn-bol": {{ProductID: "p-1", Provider: "bol", ExternalID: "8712345", SyncStatus: channelsDomain.SyncStatusError, LastSyncError: "EAN unknown"}},
	}}
	tool := tools.NewListProductChannelListingsTool(&stubChannels{out: channels()}, listings)
	assertStrictDefinition(t, tool, "list_product_channel_listings")

	got := call(t, tool, `{"product_id":"p-1"}`)

	assert.Equal(t, []string{"p-1@conn-woo", "p-1@conn-bol"}, listings.calls, "internal channels have no external listing")
	assert.Equal(t, "/products/p-1", got["admin_path"])
	list := got["listings"].([]any)
	require.Len(t, list, 2)
	woo := list[0].(map[string]any)
	assert.Equal(t, "Webshop", woo["channel_name"])
	assert.Equal(t, "4710", woo["external_id"])
	assert.Equal(t, "synced", woo["sync_status"])
	assert.Equal(t, "2026-11-22T08:30:00Z", woo["last_synced_at"])
	assert.Equal(t, "EAN unknown", list[1].(map[string]any)["last_sync_error"])
}

func TestListProductChannelListingsTool_oneChannel(t *testing.T) {
	listings := &stubListings{}
	tool := tools.NewListProductChannelListingsTool(&stubChannels{out: channels()}, listings)

	got := call(t, tool, `{"product_id":"p-1","sales_channel_id":"ch-bol"}`)

	assert.Equal(t, []string{"p-1@conn-bol"}, listings.calls)
	assert.Empty(t, got["listings"])
	assert.ErrorContains(t, callErr(t, tool, `{"product_id":"p-1","sales_channel_id":"nope"}`), "no sales channel")
}

func TestListProductChannelListingsTool_errors(t *testing.T) {
	assert.ErrorIs(t, callErr(t, tools.NewListProductChannelListingsTool(&stubChannels{}, &stubListings{}), `{}`), tools.ErrInvalidArguments)
	assert.ErrorContains(t, callErr(t, tools.NewListProductChannelListingsTool(&stubChannels{out: channels()}, &stubListings{err: errors.New("db down")}), `{"product_id":"p-1"}`), "db down")
}
