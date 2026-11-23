package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type stubConnections struct {
	out []integrationsDomain.Connection
	err error
}

func (s stubConnections) Execute(context.Context) ([]integrationsDomain.Connection, error) {
	return s.out, s.err
}

type stubAIConnection struct {
	out []integrationsDomain.AIConnection
	err error
}

func (s stubAIConnection) Execute(context.Context) ([]integrationsDomain.AIConnection, error) {
	return s.out, s.err
}

type stubSyncLogs struct {
	out        []appchannels.SyncLogEntryWithNames
	err        error
	gotPerPage int
	called     bool
}

func (s *stubSyncLogs) Execute(_ context.Context, _, perPage int) ([]appchannels.SyncLogEntryWithNames, int64, error) {
	s.called, s.gotPerPage = true, perPage
	return s.out, int64(len(s.out)), s.err
}

func TestListIntegrationConnections_neverLeaksCredentials(t *testing.T) {
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	tool := tools.NewListIntegrationConnectionsTool(
		stubConnections{out: []integrationsDomain.Connection{{
			ID: "c-1", Provider: integrationsDomain.Provider("woocommerce"), StoreURL: "https://www.shop.example/",
			ConsumerKey: "ck_secret", ConsumerSecretEncrypted: "enc", WebhookSecretEncrypted: "whsec",
			Status: integrationsDomain.StatusError, DefaultCurrency: "EUR", CreatedAt: at, UpdatedAt: at,
		}}},
		stubAIConnection{out: []integrationsDomain.AIConnection{
			{Provider: integrationsDomain.AIProviderNVIDIA, APIKeyEncrypted: "enc-key", Label: "key 1", Enabled: true,
				Model: "meta/llama-3.3-70b-instruct", Status: integrationsDomain.StatusConnected},
			{Provider: integrationsDomain.AIProviderGemini, APIKeyEncrypted: "enc-key", Position: 1, Status: integrationsDomain.StatusConnected},
		}},
	)
	assertStrictDefinition(t, tool, "list_integration_connections")

	raw, err := tool.Call(context.Background(), nil)
	require.NoError(t, err)
	for _, secret := range []string{"ck_secret", "enc", "whsec", "enc-key"} {
		assert.NotContains(t, raw, `"`+secret+`"`)
	}
	out := call(t, tool, `{}`)
	c := out["connections"].([]any)[0].(map[string]any)
	assert.Equal(t, "woocommerce", c["provider"])
	assert.Equal(t, "www.shop.example", c["store"])
	assert.Equal(t, "error", c["status"])
	ai := out["ai_connections"].([]any)
	require.Len(t, ai, 2, "every AI connection, in the order AI features try them")
	first := ai[0].(map[string]any)
	assert.Equal(t, "nvidia", first["provider"])
	assert.Equal(t, "key 1", first["label"])
	assert.Equal(t, "meta/llama-3.3-70b-instruct", first["model"])
	assert.Equal(t, true, first["enabled"])
	assert.Equal(t, false, ai[1].(map[string]any)["enabled"])
}

func TestListIntegrationConnections_noAIAndErrors(t *testing.T) {
	out := call(t, tools.NewListIntegrationConnectionsTool(stubConnections{}, stubAIConnection{}), `{}`)
	assert.Empty(t, out["ai_connections"])
	assert.Empty(t, out["connections"])

	err := callErr(t, tools.NewListIntegrationConnectionsTool(stubConnections{err: errors.New("boom")}, stubAIConnection{}), `{}`)
	assert.ErrorContains(t, err, "boom")
}

func syncLog(conn string, status channelsDomain.SyncLogStatus, msg string, at time.Time) appchannels.SyncLogEntryWithNames {
	return appchannels.SyncLogEntryWithNames{
		SyncLogEntry: channelsDomain.SyncLogEntry{ProductID: "p-1", ConnectionID: conn, Provider: "woocommerce", ExternalID: "4710",
			Direction: channelsDomain.SyncFlowPush, Status: status, ErrorMessage: msg, SyncedAt: at},
		ProductName: "Doce de leite", StoreURL: "https://shop.example",
	}
}

func TestListSyncLogs(t *testing.T) {
	at := time.Date(2026, 11, 6, 21, 43, 0, 0, time.UTC)
	all := &stubSyncLogs{out: []appchannels.SyncLogEntryWithNames{
		syncLog("c-1", channelsDomain.SyncLogStatusError, "403 forbidden", at),
		syncLog("c-1", channelsDomain.SyncLogStatusSuccess, "", at),
		syncLog("c-2", channelsDomain.SyncLogStatusSuccess, "", at),
	}}
	attention := &stubSyncLogs{out: []appchannels.SyncLogEntryWithNames{syncLog("c-1", channelsDomain.SyncLogStatusError, "403 forbidden", at)}}
	tool := tools.NewListSyncLogsTool(all, attention)
	assertStrictDefinition(t, tool, "list_sync_logs")

	out := call(t, tool, `{"connection_id":"c-1","status":"error"}`)
	rows := out["sync_logs"].([]any)
	require.Len(t, rows, 1)
	r := rows[0].(map[string]any)
	assert.Equal(t, "Doce de leite", r["product_name"])
	assert.Equal(t, "/products/"+r["product_id"].(string), r["admin_path"])
	assert.Equal(t, "shop.example", r["store"])
	assert.Equal(t, "push", r["direction"])
	assert.Equal(t, "403 forbidden", r["error_message"])
	assert.Equal(t, "2026-11-06T21:43:00Z", r["synced_at"])
	assert.False(t, attention.called)

	needing := call(t, tool, `{"needing_attention":true}`)
	assert.True(t, attention.called)
	assert.Len(t, needing["sync_logs"], 1)

	limited := call(t, tool, `{"limit":2}`)
	assert.Len(t, limited["sync_logs"], 2)

	assert.ErrorIs(t, callErr(t, tool, `{"status":"meh"}`), tools.ErrInvalidArguments)
	all.err = errors.New("boom")
	assert.ErrorContains(t, callErr(t, tool, `{}`), "boom")
}
