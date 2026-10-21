// Package postnl is the direct PostNL shipping integration: labels through
// the Shipment v4 API (POST /shipment/delivery/v4/labelconfirm -- label and
// pre-announcement in one call) and tracking through the Shipping Status
// API (by our own reference: PostNL's docs say the barcode lookup doesn't
// cover the barcodes v4 generates). Request/response shapes follow PostNL's
// developer portal (developer.postnl.nl, Shipment v4 + Shippingstatus).
package postnl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

const (
	productionBaseURL = "https://api.postnl.nl"
	sandboxBaseURL    = "https://api-sandbox.postnl.nl"
)

// CarrierClientFactory builds a PostNL CarrierClient per connection.
type CarrierClientFactory struct {
	httpClient *http.Client
	baseURL    string // overrides production/sandbox; tests only
}

var _ integrationsDomain.CarrierClientFactory = CarrierClientFactory{}

type Option func(*CarrierClientFactory)

// WithBaseURL points every client at baseURL instead of PostNL (tests).
func WithBaseURL(baseURL string) Option {
	return func(f *CarrierClientFactory) { f.baseURL = baseURL }
}

func NewCarrierClientFactory(opts ...Option) CarrierClientFactory {
	f := CarrierClientFactory{httpClient: &http.Client{Timeout: 30 * time.Second}}
	for _, opt := range opts {
		opt(&f)
	}
	return f
}

func (f CarrierClientFactory) NewClient(creds integrationsDomain.CarrierCredentials) (integrationsDomain.CarrierClient, error) {
	if creds.APIKey == "" || creds.CustomerCode == "" || creds.CustomerNumber == "" {
		return nil, errors.New("postnl: api key, customer code and customer number are all required")
	}
	baseURL := f.baseURL
	if baseURL == "" {
		baseURL = productionBaseURL
		if creds.Sandbox {
			baseURL = sandboxBaseURL
		}
	}
	return &Client{
		http: f.httpClient, baseURL: baseURL, apiKey: creds.APIKey,
		customerCode: creds.CustomerCode, customerNumber: creds.CustomerNumber,
	}, nil
}

// Client is one PostNL account's CarrierClient.
type Client struct {
	http           *http.Client
	baseURL        string
	apiKey         string
	customerCode   string
	customerNumber string
}

var _ integrationsDomain.CarrierClient = (*Client)(nil)

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errorDetail(raw))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// errorBody covers the error shapes PostNL answers with: the v4 APIs'
// RFC 9457 problem details, its API gateway's faults (seen live on the
// sandbox), and the older APIs' Errors list.
type errorBody struct {
	Title   string              `json:"title"`
	Errors  json.RawMessage     `json:"errors"`
	TraceID string              `json:"traceId"`
	Faults  []fault             `json:"faults"`
	Fault   *fault              `json:"fault"`
	Legacy  []map[string]string `json:"Errors"`
}

type fault struct {
	FaultString string `json:"faultstring"`
}

func errorDetail(raw []byte) string {
	var b errorBody
	if err := json.Unmarshal(raw, &b); err != nil {
		return strings.TrimSpace(string(raw))
	}
	var parts []string
	if b.Title != "" {
		parts = append(parts, b.Title)
	}
	if len(b.Errors) > 0 && string(b.Errors) != "null" {
		parts = append(parts, string(b.Errors))
	}
	if b.Fault != nil {
		parts = append(parts, b.Fault.FaultString)
	}
	for _, f := range b.Faults {
		parts = append(parts, f.FaultString)
	}
	for _, e := range b.Legacy {
		for _, v := range e {
			parts = append(parts, v)
		}
	}
	if len(parts) == 0 {
		return strings.TrimSpace(string(raw))
	}
	detail := strings.Join(parts, ": ")
	if b.TraceID != "" {
		detail += " (traceId " + b.TraceID + ")"
	}
	return detail
}

// CancelShipment: PostNL has no delete for a pre-announced shipment. Its
// customer service says it's free as long as the barcode is never scanned
// and drops out of "Voorgemelde zendingen" after 7 days.
func (c *Client) CancelShipment(context.Context, string) error {
	return fmt.Errorf("postnl: cancel shipment: %w", integrationsDomain.ErrCarrierLabelNotDeletable)
}

// GetLabel isn't available: PostNL returns the label once, in the create
// response, and CreateShipmentUseCase stores it.
func (c *Client) GetLabel(context.Context, string, string, string) (integrationsDomain.LabelResult, error) {
	return integrationsDomain.LabelResult{}, fmt.Errorf("postnl: get label: %w", integrationsDomain.ErrCarrierOperationNotSupported)
}

func (c *Client) GetDeliveryPriceEstimate(context.Context, integrationsDomain.DeliveryPriceEstimateInput) ([]integrationsDomain.DeliveryPriceOption, error) {
	return nil, fmt.Errorf("postnl: delivery price estimate: %w", integrationsDomain.ErrCarrierOperationNotSupported)
}
