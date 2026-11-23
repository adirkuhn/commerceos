package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/adirkuhn/commerceos/domain/assistant"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

type salesChannelLister interface {
	Execute(ctx context.Context) ([]channelsDomain.SalesChannel, error)
}

type productListingLister interface {
	Execute(ctx context.Context, productID, connectionID string) ([]channelsDomain.ProductChannelListing, error)
}

// ListSalesChannelsTool lists where the shop sells.
type ListSalesChannelsTool struct{ list salesChannelLister }

func NewListSalesChannelsTool(list salesChannelLister) *ListSalesChannelsTool {
	return &ListSalesChannelsTool{list: list}
}

func (t *ListSalesChannelsTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_sales_channels",
		Description: "List the shop's sales channels (WooCommerce, Bol, own storefronts, WhatsApp) with type, active flag, " +
			"currency and which products they enroll; ids are for list_product_channel_listings.",
		Parameters: object(map[string]any{}),
	}
}

type channelView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Active         bool   `json:"active"`
	CurrencyCode   string `json:"currency_code,omitempty"`
	EnrollmentMode string `json:"enrollment_mode,omitempty"`
	EnrollmentTag  string `json:"enrollment_tag,omitempty"`
	// Connected: the channel has an integration connection (an external
	// system with its own product listings).
	Connected bool `json:"connected"`
}

func (t *ListSalesChannelsTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct{}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	chs, err := t.list.Execute(ctx)
	if err != nil {
		return "", err
	}
	out := struct {
		SalesChannels []channelView `json:"sales_channels"`
	}{SalesChannels: make([]channelView, 0, len(chs))}
	for _, c := range chs {
		out.SalesChannels = append(out.SalesChannels, channelView{
			ID: c.ID, Name: c.Name, Type: string(c.Type), Active: c.Active, CurrencyCode: c.CurrencyCode,
			EnrollmentMode: string(c.EnrollmentMode), EnrollmentTag: c.EnrollmentTag, Connected: c.ConnectionID != nil,
		})
	}
	return toJSON(out)
}

// ListProductChannelListingsTool shows how a product is listed on each
// connected channel -- sync status, external id, last sync and its error.
type ListProductChannelListingsTool struct {
	channels salesChannelLister
	listings productListingLister
}

func NewListProductChannelListingsTool(channels salesChannelLister, listings productListingLister) *ListProductChannelListingsTool {
	return &ListProductChannelListingsTool{channels: channels, listings: listings}
}

func (t *ListProductChannelListingsTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "list_product_channel_listings",
		Description: "For one product (id from search), its listing on each connected sales channel (WooCommerce, Bol): " +
			"sync status, external id, when it was last synced and the last sync error. Own storefronts have no listing.",
		Parameters: object(map[string]any{
			"product_id":       str("The product id."),
			"sales_channel_id": str("Optional: only this sales channel (id from list_sales_channels)."),
		}, "product_id"),
	}
}

type listingView struct {
	ChannelID      string `json:"channel_id"`
	ChannelName    string `json:"channel_name"`
	ChannelType    string `json:"channel_type"`
	ExternalID     string `json:"external_id"`
	VariantID      string `json:"variant_id,omitempty"`
	SyncStatus     string `json:"sync_status"`
	LastSyncedAt   string `json:"last_synced_at,omitempty"`
	LastSyncError  string `json:"last_sync_error,omitempty"`
	NotForSaleNote string `json:"not_for_sale_note,omitempty"`
	Gated          bool   `json:"gated"`
}

func (t *ListProductChannelListingsTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct {
		ProductID      string `json:"product_id"`
		SalesChannelID string `json:"sales_channel_id"`
	}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	if err := required("product_id", args.ProductID); err != nil {
		return "", err
	}
	chs, err := t.channels.Execute(ctx)
	if err != nil {
		return "", err
	}
	if args.SalesChannelID != "" {
		chs, err = onlyChannel(chs, args.SalesChannelID)
		if err != nil {
			return "", err
		}
	}
	out := struct {
		Listings []listingView `json:"listings"`
	}{Listings: []listingView{}}
	for _, c := range chs {
		if c.ConnectionID == nil {
			continue
		}
		ls, err := t.listings.Execute(ctx, args.ProductID, *c.ConnectionID)
		if err != nil {
			return "", err
		}
		for _, l := range ls {
			out.Listings = append(out.Listings, listingView{
				ChannelID: c.ID, ChannelName: c.Name, ChannelType: string(c.Type), ExternalID: l.ExternalID,
				VariantID: deref(l.VariantID), SyncStatus: string(l.SyncStatus), LastSyncedAt: formatTimePtr(l.LastSyncedAt),
				LastSyncError: l.LastSyncError, NotForSaleNote: l.NotForSaleNote, Gated: l.IsGated(),
			})
		}
	}
	return toJSON(out)
}

func onlyChannel(chs []channelsDomain.SalesChannel, id string) ([]channelsDomain.SalesChannel, error) {
	for _, c := range chs {
		if c.ID == id {
			return []channelsDomain.SalesChannel{c}, nil
		}
	}
	return nil, fmt.Errorf("no sales channel with id %q", id)
}
