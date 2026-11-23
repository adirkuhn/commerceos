package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	"github.com/adirkuhn/commerceos/domain/assistant"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// syncLogScanSize is how many recent sync logs list_sync_logs filters --
// the use cases page, and the model only ever needs the latest few.
const syncLogScanSize = 100

var syncLogStatuses = []string{string(channelsDomain.SyncLogStatusSuccess), string(channelsDomain.SyncLogStatusError)}

type connectionLister interface {
	Execute(ctx context.Context) ([]integrationsDomain.Connection, error)
}

type aiConnectionGetter interface {
	Execute(ctx context.Context) (*integrationsDomain.AIConnection, error)
}

type syncLogPager interface {
	Execute(ctx context.Context, page, perPage int) ([]appchannels.SyncLogEntryWithNames, int64, error)
}

// hostOf shows a store by its host (www.shop.example) -- the full URL can
// carry paths that mean nothing to a reader.
func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return strings.TrimSuffix(raw, "/")
	}
	return u.Host
}

// ListIntegrationConnectionsTool lists the tenant's integrations. Only
// provider, store and status leave this tool -- never keys or secrets.
type ListIntegrationConnectionsTool struct {
	connections connectionLister
	ai          aiConnectionGetter
}

func NewListIntegrationConnectionsTool(connections connectionLister, ai aiConnectionGetter) *ListIntegrationConnectionsTool {
	return &ListIntegrationConnectionsTool{connections: connections, ai: ai}
}

func (t *ListIntegrationConnectionsTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_integration_connections",
		Description: "List the shop's integrations (WooCommerce, Bol, MyParcel, PostNL, WhatsApp, payments, email...) with " +
			"their store and status (connected/error/disconnected), plus the connected AI provider and model. " +
			"Gives connection ids for list_sync_logs.",
		Parameters: object(map[string]any{}),
	}
}

type connectionView struct {
	ID              string `json:"id"`
	Provider        string `json:"provider"`
	Store           string `json:"store,omitempty"`
	Status          string `json:"status"`
	DefaultCurrency string `json:"default_currency,omitempty"`
	ConnectedAt     string `json:"connected_at"`
	UpdatedAt       string `json:"updated_at"`
}

type aiConnectionView struct {
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
	Status   string `json:"status"`
}

func (t *ListIntegrationConnectionsTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct{}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	conns, err := t.connections.Execute(ctx)
	if err != nil {
		return "", err
	}
	out := make([]connectionView, len(conns))
	for i, c := range conns {
		out[i] = connectionView{ID: c.ID, Provider: string(c.Provider), Store: hostOf(string(c.StoreURL)), Status: string(c.Status),
			DefaultCurrency: c.DefaultCurrency, ConnectedAt: formatTime(c.CreatedAt), UpdatedAt: formatTime(c.UpdatedAt)}
	}
	ai, err := t.ai.Execute(ctx)
	if err != nil {
		return "", err
	}
	var aiView *aiConnectionView
	if ai != nil {
		aiView = &aiConnectionView{Provider: string(ai.Provider), Model: ai.Model, Status: string(ai.Status)}
	}
	return toJSON(map[string]any{"connections": out, "ai": aiView})
}

// ListSyncLogsTool lists recent product sync results between CommerceOS
// and the sales channels.
type ListSyncLogsTool struct {
	all       syncLogPager
	attention syncLogPager
}

func NewListSyncLogsTool(all, needingAttention syncLogPager) *ListSyncLogsTool {
	return &ListSyncLogsTool{all: all, attention: needingAttention}
}

func (t *ListSyncLogsTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_sync_logs",
		Description: "List the latest product sync results with sales channels (push to / import from WooCommerce, Bol...), " +
			"newest first: product, store, direction, success/error and the error message. needing_attention lists only " +
			"failures nobody dismissed yet. For one product's last sync per channel use list_product_channel_listings.",
		Parameters: object(map[string]any{
			"connection_id":     str("Only this integration connection (id from list_integration_connections)."),
			"status":            enum("Only results with this status.", syncLogStatuses...),
			"needing_attention": map[string]any{"type": "boolean", "description": "Only failures that still need someone to look at them."},
			"limit":             limitProp(),
		}),
	}
}

type syncLogView struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	ConnectionID string `json:"connection_id"`
	Provider     string `json:"provider"`
	Store        string `json:"store"`
	ExternalID   string `json:"external_id,omitempty"`
	Direction    string `json:"direction"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	SyncedAt     string `json:"synced_at"`
}

func (t *ListSyncLogsTool) Call(ctx context.Context, args json.RawMessage) (string, error) {
	var in struct {
		ConnectionID     string `json:"connection_id"`
		Status           string `json:"status"`
		NeedingAttention bool   `json:"needing_attention"`
		Limit            int    `json:"limit"`
	}
	if err := decode(args, &in); err != nil {
		return "", err
	}
	if in.Status != "" && !contains(syncLogStatuses, in.Status) {
		return "", fmt.Errorf("%w: unknown status %q", ErrInvalidArguments, in.Status)
	}
	source := t.all
	if in.NeedingAttention {
		source = t.attention
	}
	logs, _, err := source.Execute(ctx, 1, syncLogScanSize)
	if err != nil {
		return "", err
	}
	out := []syncLogView{}
	for _, l := range logs {
		if (in.ConnectionID != "" && l.ConnectionID != in.ConnectionID) || (in.Status != "" && string(l.Status) != in.Status) {
			continue
		}
		if len(out) == clampLimit(in.Limit) {
			break
		}
		out = append(out, syncLogView{
			ProductID: l.ProductID, ProductName: l.ProductName, ConnectionID: l.ConnectionID, Provider: l.Provider, Store: hostOf(l.StoreURL),
			ExternalID: l.ExternalID, Direction: string(l.Direction), Status: string(l.Status), ErrorMessage: l.ErrorMessage, SyncedAt: formatTime(l.SyncedAt),
		})
	}
	return toJSON(map[string]any{"sync_logs": out})
}
