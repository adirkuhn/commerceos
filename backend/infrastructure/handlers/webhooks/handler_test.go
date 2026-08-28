package webhooks_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genwebhooks "github.com/adirkuhn/commerceos/api/webhooks"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	webhookshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/webhooks"
)

func init() { gin.SetMode(gin.TestMode) }

type stubReceiveWebhook struct {
	result appwebhooks.Result
	err    error
	// captured last call, for header/body-forwarding assertions
	tenantID, provider string
	rawBody            []byte
	headers            map[string]string
}

func (s *stubReceiveWebhook) Execute(_ context.Context, tenantID, provider string, rawBody []byte, headers map[string]string) (appwebhooks.Result, error) {
	s.tenantID, s.provider, s.rawBody, s.headers = tenantID, provider, rawBody, headers
	return s.result, s.err
}

func newTestRouter(uc *stubReceiveWebhook) *gin.Engine {
	r := gin.New()
	h := webhookshandler.NewHandler(uc)
	webhookshandler.RegisterPublicRoutes(r, h)
	return r
}

func doWebhookRequest(t *testing.T, r *gin.Engine, tenantID, provider string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/v1/"+tenantID+"/"+provider, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

const testTenantID = "11111111-1111-1111-1111-111111111111"

func TestReceiveWebhook_200OnSuccess(t *testing.T) {
	uc := &stubReceiveWebhook{result: appwebhooks.Result{EventID: "evt-1"}}
	r := newTestRouter(uc)

	w := doWebhookRequest(t, r, testTenantID, "woocommerce", []byte(`{"id":1}`), map[string]string{"X-Wc-Webhook-Signature": "sig"})

	require.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	assert.Equal(t, "evt-1", data["event_id"])
	assert.Equal(t, false, data["fallback"])
	assert.Equal(t, testTenantID, uc.tenantID)
	assert.Equal(t, "woocommerce", uc.provider)
	assert.Equal(t, "sig", uc.headers["X-Wc-Webhook-Signature"])
}

func TestReceiveWebhook_202OnFallback(t *testing.T) {
	uc := &stubReceiveWebhook{result: appwebhooks.Result{EventID: "evt-1", FallbackUsed: true}}
	r := newTestRouter(uc)

	w := doWebhookRequest(t, r, testTenantID, "woocommerce", []byte(`{}`), nil)

	require.Equal(t, http.StatusAccepted, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data := body["data"].(map[string]any)
	assert.Equal(t, true, data["fallback"])
}

func TestReceiveWebhook_404OnUnknownProvider(t *testing.T) {
	uc := &stubReceiveWebhook{err: webhooks.ErrProviderUnknown}
	r := newTestRouter(uc)

	w := doWebhookRequest(t, r, testTenantID, "not-a-real-vendor", []byte(`{}`), nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestReceiveWebhook_501OnNotImplemented(t *testing.T) {
	uc := &stubReceiveWebhook{err: webhooks.ErrProviderNotImplemented}
	r := newTestRouter(uc)

	w := doWebhookRequest(t, r, testTenantID, "shopify", []byte(`{}`), nil)

	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestReceiveWebhook_401OnInvalidSignature(t *testing.T) {
	uc := &stubReceiveWebhook{err: webhooks.ErrInvalidSignature}
	r := newTestRouter(uc)

	w := doWebhookRequest(t, r, testTenantID, "woocommerce", []byte(`{}`), map[string]string{"X-Wc-Webhook-Signature": "bad"})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Confirms the route is genuinely unauthenticated -- registered on a bare
// gin.New() router (no JWT middleware at all) and still resolves.
func TestReceiveWebhook_mountedOutsideAuth(t *testing.T) {
	uc := &stubReceiveWebhook{result: appwebhooks.Result{EventID: "evt-1"}}
	r := gin.New()
	genwebhooks.RegisterHandlersWithOptions(r, webhookshandler.NewHandler(uc), genwebhooks.GinServerOptions{BaseURL: "/api/v1"})

	w := doWebhookRequest(t, r, testTenantID, "woocommerce", []byte(`{}`), nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

// MyParcel signs nothing -- its verifier checks a `?token=` query
// parameter instead (see webhookproviders.MyParcelVerifier), so the
// handler must forward it into the headers map under the key the verifier
// reads.
func TestReceiveWebhook_forwardsQueryTokenForMyParcel(t *testing.T) {
	uc := &stubReceiveWebhook{result: appwebhooks.Result{EventID: "evt-1"}}
	r := newTestRouter(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/v1/"+testTenantID+"/myparcel?token=secret-abc", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "secret-abc", uc.headers["__query_token"])
}

// ReceiveWebhookWithToken is the current registration shape for MyParcel
// (application/shipping.RegisterMyParcelWebhookUseCase /
// application/integrations.CreateProviderWebhookUseCase both build this
// URL now) -- the token travels as a path segment instead of a `?token=`
// query string.
func TestReceiveWebhookWithToken_forwardsPathTokenForMyParcel(t *testing.T) {
	uc := &stubReceiveWebhook{result: appwebhooks.Result{EventID: "evt-1"}}
	r := newTestRouter(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/v1/"+testTenantID+"/myparcel/secret-abc", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "secret-abc", uc.headers["__query_token"])
}
