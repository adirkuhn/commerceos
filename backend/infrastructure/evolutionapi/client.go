// Package evolutionapi is a small, self-built client for Evolution API
// (https://doc.evolution-api.com/), a self-hosted, unofficial (Baileys
// -based WhatsApp Web) gateway -- no official WhatsApp Cloud API access is
// assumed (see domain/integrations.ProviderWhatsApp's doc comment). No
// approved third-party Go SDK exists for this API.
//
// SendText's endpoint/request shape (POST {baseURL}/message/sendText/{instance},
// header apikey: <key>, body {"number":"<phone>","text":"<message>"}) is
// built from Evolution API's published docs, NOT confirmed against a real
// running instance -- same caveat convention as
// infrastructure/myparcel's package doc comment. Confirm against a real
// instance's actual response/error shape before relying on this in
// production; a mismatch here fails safe (SendText returns an error, it
// doesn't silently drop the message).
package evolutionapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/adirkuhn/commerceos/domain/integrations"
)

// apiKeyHeader is the single place this client's auth header name lives --
// Evolution API's docs describe a flat `apikey` header (not `Authorization`).
const apiKeyHeader = "apikey"

// Config holds one WhatsApp connection's gateway details.
type Config struct {
	BaseURL      string
	InstanceName string
	APIKey       string
	HTTPClient   *http.Client
}

type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{cfg: cfg, http: cfg.HTTPClient}
}

type sendTextRequest struct {
	Number string `json:"number"`
	Text   string `json:"text"`
}

func (c *Client) SendText(ctx context.Context, toPhone, body string) error {
	reqBody, err := json.Marshal(sendTextRequest{Number: toPhone, Text: body})
	if err != nil {
		return fmt.Errorf("evolutionapi: encode request: %w", err)
	}

	url := fmt.Sprintf("%s/message/sendText/%s", c.cfg.BaseURL, c.cfg.InstanceName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("evolutionapi: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(apiKeyHeader, c.cfg.APIKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("evolutionapi: send text: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("evolutionapi: send text: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ClientFactory implements domain/integrations.EvolutionAPIClientFactory.
type ClientFactory struct{}

func NewClientFactory() ClientFactory { return ClientFactory{} }

func (ClientFactory) NewClient(baseURL, instanceName, apiKey string) (integrations.EvolutionAPIClient, error) {
	return NewClient(Config{BaseURL: baseURL, InstanceName: instanceName, APIKey: apiKey}), nil
}
