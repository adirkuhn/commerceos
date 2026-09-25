package woocommerce

import (
	"context"
	"strconv"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func (c *wooClient) ListWebhooks(ctx context.Context) ([]integrationsDomain.WooWebhook, error) {
	hooks, _, err := c.client.Webhooks.List(ctx, &wc.ListWebhooksParams{PerPage: 100})
	if err != nil {
		return nil, err
	}
	out := make([]integrationsDomain.WooWebhook, len(hooks))
	for i, h := range hooks {
		out[i] = integrationsDomain.WooWebhook{
			ID: strconv.Itoa(h.ID), Topic: h.Topic, DeliveryURL: h.DeliveryURL, Status: h.Status,
		}
	}
	return out, nil
}

func (c *wooClient) CreateWebhook(ctx context.Context, topic, deliveryURL, secret string) (integrationsDomain.WooWebhook, error) {
	created, _, err := c.client.Webhooks.Create(ctx, &wc.Webhook{
		Name: "CommerceOS: " + topic, Topic: topic, DeliveryURL: deliveryURL, Secret: secret,
	})
	if err != nil {
		return integrationsDomain.WooWebhook{}, err
	}
	return integrationsDomain.WooWebhook{
		ID: strconv.Itoa(created.ID), Topic: created.Topic, DeliveryURL: created.DeliveryURL, Status: created.Status,
	}, nil
}

func (c *wooClient) DeleteWebhook(ctx context.Context, id string) error {
	_, _, err := c.client.Webhooks.Delete(ctx, id, &wc.DeleteWebhookParams{Force: "true"})
	return err
}
