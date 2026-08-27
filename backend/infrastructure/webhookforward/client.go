// Package webhookforward is the real implementation of
// application/webhooks.HTTPPoster -- the outbound side of the webhook
// fan-out feature, POSTing a received webhook's exact raw payload to a
// tenant-configured ForwardRule.TargetURL.
package webhookforward

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"
)

// postTimeout bounds one delivery attempt -- a hung third-party endpoint
// must not stall the whole batch (see
// application/webhooks.DeliverForwardedWebhooksUseCase, which processes
// deliveries one at a time, synchronously, within a single tick).
const postTimeout = 15 * time.Second

// Client posts with a bounded per-request timeout, distinct from any
// caller-supplied ctx deadline (there normally isn't one on the worker's
// tick loop) -- see postTimeout.
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: postTimeout}}
}

func (c *Client) Post(ctx context.Context, targetURL string, contentType string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("webhookforward: build request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("webhookforward: post: %w", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
