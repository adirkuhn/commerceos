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
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	storefronthandler "github.com/adirkuhn/commerceos/infrastructure/handlers/storefront"
)

func init() { gin.SetMode(gin.TestMode) }

func strPtr(s string) *string { return &s }

type stubResolveStorefrontKey struct {
	resolved apikeysDomain.ResolvedAPIKey
	err      error
	gotKey   string
}

func (s *stubResolveStorefrontKey) Execute(_ context.Context, key string) (apikeysDomain.ResolvedAPIKey, error) {
	s.gotKey = key
	if s.err != nil {
		return apikeysDomain.ResolvedAPIKey{}, s.err
	}
	return s.resolved, nil
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

type stubCreateCart struct {
	out          *storefrontDomain.Cart
	err          error
	gotChannelID string
}

func (s *stubCreateCart) Execute(_ context.Context, salesChannelID string) (*storefrontDomain.Cart, error) {
	s.gotChannelID = salesChannelID
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubGetCart struct {
	out      *storefrontDomain.Cart
	err      error
	gotToken string
}

func (s *stubGetCart) Execute(_ context.Context, token string) (*storefrontDomain.Cart, error) {
	s.gotToken = token
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubUpdateCartItems struct {
	out      *storefrontDomain.Cart
	err      error
	gotToken string
	gotLines []appstorefront.UpdateCartItemLineInput
}

func (s *stubUpdateCartItems) Execute(_ context.Context, token string, lines []appstorefront.UpdateCartItemLineInput) (*storefrontDomain.Cart, error) {
	s.gotToken, s.gotLines = token, lines
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubCheckoutCart struct {
	out      *appstorefront.CheckoutResult
	err      error
	gotToken string
	gotInput appstorefront.CheckoutInput
}

func (s *stubCheckoutCart) Execute(_ context.Context, token string, in appstorefront.CheckoutInput) (*appstorefront.CheckoutResult, error) {
	s.gotToken, s.gotInput = token, in
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func newTestHandler(resolve *stubResolveStorefrontKey, get *stubGetCartRecommendations, create *stubCreateCart, getCart *stubGetCart, update *stubUpdateCartItems, checkout *stubCheckoutCart) *storefronthandler.Handler {
	return storefronthandler.NewHandler(resolve, get, create, getCart, update, checkout)
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

const cartRecommendationsPath = "/api/v1/storefront/v1/recommendations/cart"

func TestGetCartRecommendationsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	get := &stubGetCartRecommendations{out: &appreports.CartRecommendationsReport{
		TotalOrders: 5,
		Recommendations: []appreports.CartRecommendation{
			{SKUCode: "SKU-B", ProductName: "Widget B", Confidence: 0.6, IsBundleCandidate: true},
		},
	}}
	h := newTestHandler(resolve, get, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartRecommendationsPath, `{"items":[{"sku_code":"SKU-A","quantity":2}]}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-key", resolve.gotKey)
	require.Len(t, get.gotLines, 1)
	assert.Equal(t, appreports.CartLine{SKUCode: "SKU-A", Quantity: 2}, get.gotLines[0])
	assert.Contains(t, w.Body.String(), `"sku_code":"SKU-B"`)
	assert.Contains(t, w.Body.String(), `"is_bundle_candidate":true`)
	assert.Contains(t, w.Body.String(), `"total_orders":5`)
}

func TestGetCartRecommendationsHandler_missingBearerToken(t *testing.T) {
	h := newTestHandler(&stubResolveStorefrontKey{}, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartRecommendationsPath, `{"items":[]}`, "")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartRecommendationsHandler_invalidKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{err: errors.New("invalid key")}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartRecommendationsPath, `{"items":[]}`, "wrong-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartRecommendationsHandler_propagatesValidationError(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	get := &stubGetCartRecommendations{err: appreports.ErrCartEmpty}
	h := newTestHandler(resolve, get, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartRecommendationsPath, `{"items":[]}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func channelScopedKey(tenantID, channelID string) apikeysDomain.ResolvedAPIKey {
	return apikeysDomain.ResolvedAPIKey{TenantID: tenantID, SalesChannelID: strPtr(channelID)}
}

func TestCreateCartHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	create := &stubCreateCart{out: &storefrontDomain.Cart{Token: "cart-token-1"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, create, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts", `{}`, "test-key")

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "channel-1", create.gotChannelID)
	assert.Contains(t, w.Body.String(), `"token":"cart-token-1"`)
}

func TestCreateCartHandler_rejectsTenantWideKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts", `{}`, "test-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartHandler_notFound(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	get := &stubGetCart{err: storefrontDomain.ErrCartNotFound}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, get, &stubUpdateCartItems{}, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/carts/unknown-token", "", "test-key")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "unknown-token", get.gotToken)
}

func TestUpdateCartItemsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	update := &stubUpdateCartItems{out: &storefrontDomain.Cart{Token: "cart-token-1", Items: []storefrontDomain.CartItem{
		{SKUCode: "SKU-A", Quantity: 2, PriceMinorUnits: 500, CurrencyCode: "EUR"},
	}}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, update, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/carts/cart-token-1/items", `{"items":[{"sku_code":"SKU-A","quantity":2}]}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "cart-token-1", update.gotToken)
	require.Len(t, update.gotLines, 1)
	assert.Equal(t, appstorefront.UpdateCartItemLineInput{SKUCode: "SKU-A", Quantity: 2}, update.gotLines[0])
	assert.Contains(t, w.Body.String(), `"subtotal_minor_units":1000`)
}

func TestUpdateCartItemsHandler_insufficientStock(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	update := &stubUpdateCartItems{err: storefrontDomain.ErrInsufficientStock}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, update, &stubCheckoutCart{})

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/carts/cart-token-1/items", `{"items":[{"sku_code":"SKU-A","quantity":999}]}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCheckoutCartHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	checkout := &stubCheckoutCart{out: &appstorefront.CheckoutResult{
		OrderNumber: "WEB-ABCD1234", TotalMinorUnits: 1000, CurrencyCode: "EUR", PaymentURL: strPtr("https://pay.example/abc"),
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, checkout)

	body := `{"email":"shopper@example.com","billing":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"}}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/checkout", body, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "cart-token-1", checkout.gotToken)
	assert.Equal(t, "shopper@example.com", checkout.gotInput.Email)
	assert.Equal(t, "NL", checkout.gotInput.Billing.Country)
	assert.Contains(t, w.Body.String(), `"order_number":"WEB-ABCD1234"`)
	assert.Contains(t, w.Body.String(), `"payment_url":"https://pay.example/abc"`)
}

func TestCheckoutCartHandler_alreadyCheckedOut(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	checkout := &stubCheckoutCart{err: storefrontDomain.ErrCartAlreadyCheckedOut}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, checkout)

	body := `{"email":"shopper@example.com","billing":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"}}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/checkout", body, "test-key")

	require.Equal(t, http.StatusConflict, w.Code)
}
