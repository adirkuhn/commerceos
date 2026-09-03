package storefront_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	storefronthandler "github.com/adirkuhn/commerceos/infrastructure/handlers/storefront"
)

func init() { gin.SetMode(gin.TestMode) }

type stubResolveStorefrontKey struct {
	tenantID string
	err      error
	gotKey   string
}

func (s *stubResolveStorefrontKey) Execute(_ context.Context, key string) (string, error) {
	s.gotKey = key
	if s.err != nil {
		return "", s.err
	}
	return s.tenantID, nil
}

type stubGetCartRecommendations struct {
	out      *appreports.CartRecommendationsReport
	err      error
	gotLines []appreports.CartLine
}

func (s *stubGetCartRecommendations) Execute(_ context.Context, lines []appreports.CartLine) (*appreports.CartRecommendationsReport, error) {
	s.gotLines = lines
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appreports.CartRecommendationsReport{}, nil
}

func newTestRouter(h *storefronthandler.Handler) *gin.Engine {
	r := gin.New()
	genstorefront.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path, body, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

const cartPath = "/api/v1/storefront/v1/recommendations/cart"

func TestGetCartRecommendationsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{tenantID: "tenant-1"}
	get := &stubGetCartRecommendations{out: &appreports.CartRecommendationsReport{
		TotalOrders: 5,
		Recommendations: []appreports.CartRecommendation{
			{SKUCode: "SKU-B", ProductName: "Widget B", Confidence: 0.6, IsBundleCandidate: true},
		},
	}}
	h := storefronthandler.NewHandler(resolve, get)

	w := doRequest(newTestRouter(h), http.MethodPost, cartPath, `{"items":[{"sku_code":"SKU-A","quantity":2}]}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-key", resolve.gotKey)
	require.Len(t, get.gotLines, 1)
	assert.Equal(t, appreports.CartLine{SKUCode: "SKU-A", Quantity: 2}, get.gotLines[0])
	assert.Contains(t, w.Body.String(), `"sku_code":"SKU-B"`)
	assert.Contains(t, w.Body.String(), `"is_bundle_candidate":true`)
	assert.Contains(t, w.Body.String(), `"total_orders":5`)
}

func TestGetCartRecommendationsHandler_missingBearerToken(t *testing.T) {
	h := storefronthandler.NewHandler(&stubResolveStorefrontKey{}, &stubGetCartRecommendations{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartPath, `{"items":[]}`, "")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartRecommendationsHandler_invalidKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{err: errors.New("invalid key")}
	h := storefronthandler.NewHandler(resolve, &stubGetCartRecommendations{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartPath, `{"items":[]}`, "wrong-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartRecommendationsHandler_propagatesValidationError(t *testing.T) {
	resolve := &stubResolveStorefrontKey{tenantID: "tenant-1"}
	get := &stubGetCartRecommendations{err: appreports.ErrCartEmpty}
	h := storefronthandler.NewHandler(resolve, get)

	w := doRequest(newTestRouter(h), http.MethodPost, cartPath, `{"items":[]}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
