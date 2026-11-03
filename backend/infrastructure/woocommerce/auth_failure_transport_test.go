package woocommerce

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/errs"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

const wpJSON = "application/json; charset=UTF-8"

func storeAnswering(t *testing.T, status int, contentType, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func getThroughTransport(t *testing.T, url string) error {
	t.Helper()
	client := &http.Client{Transport: newAuthFailureTransport(http.DefaultTransport)}
	resp, err := client.Get(url)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func TestAuthFailureTransport_classifiesStoreRefusals(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		contentType string
		body        string
		want        error
		wantDetail  string
	}{
		{"woo 401 means the key is wrong", 401, wpJSON,
			`{"code":"woocommerce_rest_cannot_view","message":"Sorry","data":{"status":401}}`,
			integrationsDomain.ErrWooAuthRejected, "status 401: Sorry"},
		{"woo 403 means the key's user lacks rights", 403, wpJSON,
			`{"code":"woocommerce_rest_cannot_view","message":"Sem permissão","data":{"status":403}}`,
			integrationsDomain.ErrWooPermissionDenied, "status 403: Sem permissão"},
		{"html 403 is a firewall", 403, "text/html", "<html><body>Access   denied</body></html>",
			integrationsDomain.ErrWooBlocked, "status 403: <html><body>Access denied</body></html>"},
		{"empty 403 is a firewall", 403, "text/plain", "",
			integrationsDomain.ErrWooBlocked, "status 403"},
		{"non-wordpress json 403 is a firewall", 403, "application/json", `{"error":"banned"}`,
			integrationsDomain.ErrWooBlocked, `status 403: {"error":"banned"}`},
		{"html 401 is a firewall too", 401, "text/html", "auth required",
			integrationsDomain.ErrWooBlocked, "status 401: auth required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := getThroughTransport(t, storeAnswering(t, tc.status, tc.contentType, tc.body))

			require.Error(t, got)
			assert.True(t, errors.Is(got, tc.want), "got %v", got)
			assert.True(t, errors.Is(got, integrationsDomain.ErrWooRequestFailed))
			detail, ok := errs.DetailOf(got)
			assert.True(t, ok)
			assert.Equal(t, tc.wantDetail, detail)
		})
	}
}

func TestAuthFailureTransport_truncatesALongBody(t *testing.T) {
	long := make([]byte, 500)
	for i := range long {
		long[i] = 'x'
	}
	got := getThroughTransport(t, storeAnswering(t, 403, "text/html", string(long)))

	detail, _ := errs.DetailOf(got)
	assert.LessOrEqual(t, len(detail), len("status 403: ")+maxBodySnippet+len("…"))
}

func TestAuthFailureTransport_passesOtherResponsesThrough(t *testing.T) {
	for _, status := range []int{200, 404, 500} {
		got := getThroughTransport(t, storeAnswering(t, status, wpJSON, `{}`))
		assert.NoError(t, got, "status %d is not a refusal", status)
	}
}

// Both request paths -- the SDK's and our own raw requests (categories,
// settings) -- go through the classifying transport.
func TestClient_bothRequestPathsClassifyAFirewallBlock(t *testing.T) {
	client, err := (&Factory{}).NewClient(storeAnswering(t, 403, "text/html", "blocked"), "ck", "cs")
	require.NoError(t, err)

	sdkErr := client.TestConnection(context.Background())
	_, rawErr := client.ListCategories(context.Background())

	assert.True(t, errors.Is(sdkErr, integrationsDomain.ErrWooBlocked), "sdk path: %v", sdkErr)
	assert.True(t, errors.Is(rawErr, integrationsDomain.ErrWooBlocked), "raw path: %v", rawErr)
}

// A product (or variation) deleted in the store: WooCommerce answers with
// its invalid-id error code -- 404 on read, 400 on update.
func TestAuthFailureTransport_classifiesAProductThatNoLongerExists(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{
		{404, "woocommerce_rest_product_invalid_id"},
		{400, "woocommerce_rest_product_invalid_id"},
		{404, "woocommerce_rest_product_variation_invalid_id"},
	} {
		got := getThroughTransport(t, storeAnswering(t, tc.status, wpJSON,
			`{"code":"`+tc.code+`","message":"Invalid ID.","data":{"status":`+map[int]string{400: "400", 404: "404"}[tc.status]+`}}`))

		require.Error(t, got)
		assert.True(t, errors.Is(got, integrationsDomain.ErrWooProductNotFound), "%d %s: %v", tc.status, tc.code, got)
		assert.True(t, errors.Is(got, integrationsDomain.ErrWooRequestFailed))
	}
}

// Any other 400/404 -- a missing route, a validation error -- is not "the
// product is gone" and passes through untouched, body intact.
func TestAuthFailureTransport_passesOtherNotFoundsThroughWithTheirBody(t *testing.T) {
	body := `{"code":"rest_no_route","message":"No route was found","data":{"status":404}}`
	client := &http.Client{Transport: newAuthFailureTransport(http.DefaultTransport)}

	resp, err := client.Get(storeAnswering(t, 404, wpJSON, body))

	require.NoError(t, err)
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	assert.Equal(t, body, string(got))
}
