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
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	themeDomain "github.com/adirkuhn/commerceos/domain/theme"
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

type stubListProducts struct {
	out *appstorefront.ListProductsResult
	err error
}

func (s *stubListProducts) Execute(_ context.Context, _ string, _ appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appstorefront.ListProductsResult{}, nil
}

type stubGetProduct struct {
	out     *appstorefront.ProductDetail
	err     error
	gotID   string
	gotChan string
}

func (s *stubGetProduct) Execute(_ context.Context, salesChannelID, productID string) (*appstorefront.ProductDetail, error) {
	s.gotChan, s.gotID = salesChannelID, productID
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubListCategories struct {
	out []pmsDomain.Category
	err error
}

func (s *stubListCategories) Execute(_ context.Context) ([]pmsDomain.Category, error) {
	return s.out, s.err
}

type stubLookupOrder struct {
	out *appstorefront.OrderStatus
	err error
}

func (s *stubLookupOrder) Execute(_ context.Context, _, _, _ string) (*appstorefront.OrderStatus, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubGetStorefrontTheme struct {
	out *themeDomain.ThemePage
	err error

	gotChannelID string
	gotSlug      string
}

func (s *stubGetStorefrontTheme) Execute(_ context.Context, channelID, slug string) (*themeDomain.ThemePage, error) {
	s.gotChannelID, s.gotSlug = channelID, slug
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &themeDomain.ThemePage{SalesChannelID: channelID, Slug: themeDomain.PageSlug(slug), CanvasWidth: 1440}, nil
}

type stubGetShippingOptions struct {
	out []appstorefront.ShippingOption

	gotShipping appstorefront.CheckoutAddressInput
}

func (s *stubGetShippingOptions) Execute(_ context.Context, shipping appstorefront.CheckoutAddressInput) []appstorefront.ShippingOption {
	s.gotShipping = shipping
	return s.out
}

func newTestHandler(
	resolve *stubResolveStorefrontKey, get *stubGetCartRecommendations, create *stubCreateCart, getCart *stubGetCart,
	update *stubUpdateCartItems, checkout *stubCheckoutCart, listProducts *stubListProducts, getProduct *stubGetProduct,
	listCategories *stubListCategories, lookupOrder *stubLookupOrder, extra ...any,
) *storefronthandler.Handler {
	getTheme := &stubGetStorefrontTheme{}
	getShippingOptions := &stubGetShippingOptions{}
	for _, e := range extra {
		switch v := e.(type) {
		case *stubGetStorefrontTheme:
			getTheme = v
		case *stubGetShippingOptions:
			getShippingOptions = v
		}
	}
	return storefronthandler.NewHandler(resolve, get, create, getCart, update, checkout, listProducts, getProduct, listCategories, lookupOrder, getTheme, getShippingOptions)
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
	h := newTestHandler(resolve, get, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

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
	h := newTestHandler(&stubResolveStorefrontKey{}, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartRecommendationsPath, `{"items":[]}`, "")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartRecommendationsHandler_invalidKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{err: errors.New("invalid key")}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartRecommendationsPath, `{"items":[]}`, "wrong-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartRecommendationsHandler_propagatesValidationError(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	get := &stubGetCartRecommendations{err: appreports.ErrCartEmpty}
	h := newTestHandler(resolve, get, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodPost, cartRecommendationsPath, `{"items":[]}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func channelScopedKey(tenantID, channelID string) apikeysDomain.ResolvedAPIKey {
	return apikeysDomain.ResolvedAPIKey{TenantID: tenantID, SalesChannelID: strPtr(channelID)}
}

func TestCreateCartHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	create := &stubCreateCart{out: &storefrontDomain.Cart{Token: "cart-token-1"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, create, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts", `{}`, "test-key")

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "channel-1", create.gotChannelID)
	assert.Contains(t, w.Body.String(), `"token":"cart-token-1"`)
}

func TestCreateCartHandler_rejectsTenantWideKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts", `{}`, "test-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetCartHandler_notFound(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	get := &stubGetCart{err: storefrontDomain.ErrCartNotFound}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, get, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/carts/unknown-token", "", "test-key")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "unknown-token", get.gotToken)
}

func TestUpdateCartItemsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	update := &stubUpdateCartItems{out: &storefrontDomain.Cart{Token: "cart-token-1", Items: []storefrontDomain.CartItem{
		{SKUCode: "SKU-A", Quantity: 2, PriceMinorUnits: 500, CurrencyCode: "EUR"},
	}}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, update, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

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
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, update, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/carts/cart-token-1/items", `{"items":[{"sku_code":"SKU-A","quantity":999}]}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCheckoutCartHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	checkout := &stubCheckoutCart{out: &appstorefront.CheckoutResult{
		OrderNumber: "WEB-ABCD1234", TotalMinorUnits: 1000, CurrencyCode: "EUR", PaymentURL: strPtr("https://pay.example/abc"),
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, checkout, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	body := `{"email":"shopper@example.com","billing":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"}}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/checkout", body, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "cart-token-1", checkout.gotToken)
	assert.Equal(t, "shopper@example.com", checkout.gotInput.Email)
	assert.Equal(t, "NL", checkout.gotInput.Billing.Country)
	assert.Contains(t, w.Body.String(), `"order_number":"WEB-ABCD1234"`)
	assert.Contains(t, w.Body.String(), `"payment_url":"https://pay.example/abc"`)
}

func TestCheckoutCartHandler_passesThroughTheChosenShippingMethodTitle(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	checkout := &stubCheckoutCart{out: &appstorefront.CheckoutResult{OrderNumber: "WEB-ABCD1234", TotalMinorUnits: 1000, CurrencyCode: "EUR"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, checkout, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	body := `{"email":"shopper@example.com","billing":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping_method_title":"Evening"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/checkout", body, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Evening", checkout.gotInput.ShippingMethodTitle)
}

func TestGetShippingOptionsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	shippingOptions := &stubGetShippingOptions{out: []appstorefront.ShippingOption{
		{Title: "Standard", AmountMinorUnits: 495, CurrencyCode: "EUR"},
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, shippingOptions)

	body := `{"first_name":"A","last_name":"B","address1":"Damrak 1","city":"Amsterdam","postcode":"1012 LG","country":"NL"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/shipping-options", body, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"title":"Standard"`)
	assert.Contains(t, w.Body.String(), `"amount_minor_units":495`)
	assert.Equal(t, "NL", shippingOptions.gotShipping.Country)
	assert.Equal(t, "Damrak 1", shippingOptions.gotShipping.Address1)
}

func TestGetShippingOptionsHandler_returnsEmptyListWhenNoneAvailable(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	body := `{"first_name":"A","last_name":"B","address1":"Main St 1","city":"Berlin","postcode":"10115","country":"DE"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/shipping-options", body, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":[]`)
}

func TestGetShippingOptionsHandler_rejectsTenantWideKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	body := `{"first_name":"A","last_name":"B","address1":"Damrak 1","city":"Amsterdam","postcode":"1012 LG","country":"NL"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/shipping-options", body, "test-key")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCheckoutCartHandler_alreadyCheckedOut(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	checkout := &stubCheckoutCart{err: storefrontDomain.ErrCartAlreadyCheckedOut}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, checkout, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	body := `{"email":"shopper@example.com","billing":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"}}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/checkout", body, "test-key")

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestListStorefrontProductsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	price := int64(500)
	list := &stubListProducts{out: &appstorefront.ListProductsResult{
		Total: 1,
		Products: []appstorefront.ProductSummary{{
			Product: pmsDomain.Product{ID: "product-1", Name: "Widget", Slug: "widget"}, PriceMinorUnits: &price,
		}},
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, list, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Widget"`)
	assert.Contains(t, w.Body.String(), `"total":1`)
}

func TestListStorefrontProductsHandler_rejectsTenantWideKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products", "", "test-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetStorefrontProductHandler_notAvailableIs404(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	get := &stubGetProduct{err: storefrontDomain.ErrProductNotAvailable}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, get, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products/unknown-id", "", "test-key")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "channel-1", get.gotChan)
	assert.Equal(t, "unknown-id", get.gotID)
}

func TestGetStorefrontProductHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	get := &stubGetProduct{out: &appstorefront.ProductDetail{Product: pmsDomain.Product{ID: "product-1", Name: "Widget"}}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, get, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products/product-1", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Widget"`)
}

func TestListStorefrontCategoriesHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	list := &stubListCategories{out: []pmsDomain.Category{{ID: "cat-1", Name: "Shoes", Slug: "shoes"}}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, list, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/categories", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Shoes"`)
}

func TestLookupStorefrontOrderHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	lookup := &stubLookupOrder{out: &appstorefront.OrderStatus{OrderNumber: "WEB-ABCD1234", Status: "processing", TotalMinorUnits: 1000, CurrencyCode: "EUR"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, lookup)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/orders/lookup?order_number=WEB-ABCD1234&email=shopper@example.com", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"order_number":"WEB-ABCD1234"`)
}

func TestLookupStorefrontOrderHandler_notFoundIs404(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	lookup := &stubLookupOrder{err: storefrontDomain.ErrOrderNotFound}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, lookup)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/orders/lookup?order_number=WEB-NOPE0000&email=shopper@example.com", "", "test-key")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetStorefrontThemeHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	theme := &stubGetStorefrontTheme{out: &themeDomain.ThemePage{
		SalesChannelID: "channel-1",
		Slug:           themeDomain.PageSlugHome,
		CanvasWidth:    1440,
		Blocks: []themeDomain.Block{
			{ID: "b1", Type: themeDomain.BlockTypeHero, Layout: themeDomain.BlockLayout{Desktop: themeDomain.Rect{X: 1, Y: 2, Width: 3, Height: 4}}, Props: map[string]any{}},
		},
		CustomCSS: ".hero { color: red; }",
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, theme)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/theme/pages/home", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"canvas_width":1440`)
	assert.Contains(t, w.Body.String(), `"id":"b1"`)
	assert.Contains(t, w.Body.String(), `"custom_css":".hero { color: red; }"`)
	assert.Equal(t, "channel-1", theme.gotChannelID)
	assert.Equal(t, "home", theme.gotSlug)
}

func TestGetStorefrontThemeHandler_channelUnscopedKeyIs401(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/theme/pages/home", "", "test-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetStorefrontThemeHandler_invalidSlugIs422(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	theme := &stubGetStorefrontTheme{err: themeDomain.ErrInvalidPageSlug}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, theme)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/theme/pages/not-a-real-page", "", "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
