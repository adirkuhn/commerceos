package myparcel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

// WebhookSubscription is this package's own response shape for a
// registered webhook subscription -- adapter.go maps it to
// domain/integrations.WebhookSubscription at the boundary.
type WebhookSubscription struct {
	ID   string
	Hook string
	URL  string
}

// webhookSubscriptionRequest is POST /webhook_subscriptions' request body
// -- confirmed against developer.myparcel.nl/api-reference/10.webhooks.html's
// own example: `{"data":{"webhook_subscriptions":[{"hook":...,"url":...}]}}`.
// Was previously `data.hooks[]`, which MyParcel rejects with a 422 ("the
// property hooks is not defined and the definition does not allow
// additional properties").
type webhookSubscriptionRequest struct {
	Data struct {
		WebhookSubscriptions []struct {
			Hook string `json:"hook"`
			URL  string `json:"url"`
		} `json:"webhook_subscriptions"`
	} `json:"data"`
}

// webhookSubscriptionListResponse is GET/POST /webhook_subscriptions'
// response envelope -- the exact nesting (data.webhook_subscriptions[])
// mirrors the create request's own data.hooks[] nesting convention and the
// id/hook/url field names documented for the resource, but -- like the
// three items already flagged in this package's doc comment -- was NOT
// confirmed against a real sandbox response during this build.
type webhookSubscriptionListResponse struct {
	Data struct {
		WebhookSubscriptions []struct {
			ID   int    `json:"id"`
			Hook string `json:"hook"`
			URL  string `json:"url"`
		} `json:"webhook_subscriptions"`
	} `json:"data"`
}

// RegisterWebhook registers this shop's callback URL for every hook type
// this codebase knows about (integrations.AllMyParcelHookTypes), not just
// shipment_status_change. MyParcel: "you can only have one webhook of each
// type at a time for a shop, so adding the same webhook overwrites any
// existing one" -- so this is safe to call more than once for the same
// connection/URL, and also overwrites any stale subscription (e.g. one left
// behind by a previous integration such as a WooCommerce plugin) pointing
// at a different delivery URL for the same hook.
//
// Every hook type is attempted even if an earlier one fails, so one bad
// hook can't silently leave the rest unregistered; all failures are joined
// into a single returned error naming which hook(s) failed.
func (c *Client) RegisterWebhook(ctx context.Context, callbackURL string) error {
	var errs []error
	for _, hook := range integrations.AllMyParcelHookTypes {
		if _, err := c.CreateWebhookSubscription(ctx, string(hook), callbackURL); err != nil {
			errs = append(errs, fmt.Errorf("hook %s: %w", hook, err))
		}
	}
	return errors.Join(errs...)
}

// CreateWebhookSubscription is RegisterWebhook generalized to any hook type
// -- see domain/integrations.MyParcelClient.CreateWebhookSubscription's doc
// comment for why this exists separately (the provider-webhooks admin
// feature, which lets an admin pick any of MyParcel's own hook types, not
// just shipment_status_change).
func (c *Client) CreateWebhookSubscription(ctx context.Context, hook, deliveryURL string) (WebhookSubscription, error) {
	var req webhookSubscriptionRequest
	req.Data.WebhookSubscriptions = []struct {
		Hook string `json:"hook"`
		URL  string `json:"url"`
	}{{Hook: hook, URL: deliveryURL}}
	var resp webhookSubscriptionListResponse
	if err := c.doJSON(ctx, http.MethodPost, "/webhook_subscriptions", "application/json", req, &resp); err != nil {
		return WebhookSubscription{}, err
	}
	if len(resp.Data.WebhookSubscriptions) == 0 {
		// MyParcel accepted the request but the response envelope wasn't
		// what this client expected (see webhookSubscriptionListResponse's
		// doc comment) -- return what we asked for rather than an empty
		// struct, since the subscription was still created.
		return WebhookSubscription{Hook: hook, URL: deliveryURL}, nil
	}
	s := resp.Data.WebhookSubscriptions[0]
	return WebhookSubscription{ID: strconv.Itoa(s.ID), Hook: s.Hook, URL: s.URL}, nil
}

// ListWebhookSubscriptions lists every subscription currently registered
// for this shop/API key, across all hook types.
func (c *Client) ListWebhookSubscriptions(ctx context.Context) ([]WebhookSubscription, error) {
	var resp webhookSubscriptionListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/webhook_subscriptions", "application/json", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]WebhookSubscription, len(resp.Data.WebhookSubscriptions))
	for i, s := range resp.Data.WebhookSubscriptions {
		out[i] = WebhookSubscription{ID: strconv.Itoa(s.ID), Hook: s.Hook, URL: s.URL}
	}
	return out, nil
}

// DeleteWebhookSubscription removes one subscription by its MyParcel id.
func (c *Client) DeleteWebhookSubscription(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/webhook_subscriptions/"+id, "application/json", nil, nil)
}
