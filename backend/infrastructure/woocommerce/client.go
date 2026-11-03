// Package woocommerce adapts github.com/adirkuhn/go-woocommerce-api to
// domain/integrations' WooClient/WooClientFactory ports, so the domain and
// application layers never import the third-party client directly.
package woocommerce

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

type Factory struct{}

func NewFactory() *Factory { return &Factory{} }

func (f *Factory) NewClient(storeURL, consumerKey, consumerSecret string) (integrationsDomain.WooClient, error) {
	// go-woocommerce-api builds its base URL as ShopURL + "/wp-json/wc/v3/"
	// with no trailing-slash check of its own -- a store URL saved with a
	// trailing slash (e.g. "https://example.com/") produces a doubled
	// slash ("https://example.com//wp-json/...") on every single request
	// this client makes. Trimming it here, once, is cheaper than fixing
	// the vendored SDK.
	trimmedStoreURL := strings.TrimRight(storeURL, "/")
	httpClient := newStoreHTTPClient()
	client, err := wc.New(wc.Config{
		ShopURL:        trimmedStoreURL,
		ConsumerKey:    consumerKey,
		ConsumerSecret: consumerSecret,
		HTTPClient:     httpClient,
	})
	if err != nil {
		return nil, err
	}
	return &wooClient{
		client: client, httpClient: httpClient,
		// go-woocommerce-api has no Categories resource and doesn't expose
		// its internal httpClient/baseURL, so ListCategories builds its own
		// request against the same REST base the library itself targets —
		// these three are all it needs to replicate that.
		storeURL: trimmedStoreURL, consumerKey: consumerKey, consumerSecret: consumerSecret,
	}, nil
}

type wooClient struct {
	client *wc.WooCommerce
	// httpClient is shared with the SDK, so our own raw requests get the
	// same 401/403 classification (see authFailureTransport).
	httpClient                            *http.Client
	storeURL, consumerKey, consumerSecret string
}

// storeRequestTimeout matches go-woocommerce-api's own default.
const storeRequestTimeout = 10 * time.Second

func newStoreHTTPClient() *http.Client {
	return &http.Client{
		Timeout:   storeRequestTimeout,
		Transport: newAuthFailureTransport(http.DefaultTransport),
	}
}

// TestConnection lists a single product as the cheapest real call that
// exercises WooCommerce's REST auth — a non-2xx response (bad key/secret,
// revoked access) surfaces as an error here.
func (c *wooClient) TestConnection(ctx context.Context) error {
	_, _, err := c.client.Products.List(ctx, &wc.ListProductParams{PerPage: 1})
	return err
}

func (c *wooClient) basicAuthHeader() string {
	token := base64.StdEncoding.EncodeToString([]byte(c.consumerKey + ":" + c.consumerSecret))
	return "Basic " + token
}

// DownloadImage is a plain GET — Woo product images are regular media URLs,
// not behind the store's REST API auth.
func (c *wooClient) DownloadImage(ctx context.Context, url string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, "", fmt.Errorf("download image: %s: status %d", url, resp.StatusCode)
	}
	return resp.Body, resp.Header.Get("Content-Type"), nil
}
