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
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	couponsDomain "github.com/adirkuhn/commerceos/domain/coupons"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	reviewsDomain "github.com/adirkuhn/commerceos/domain/reviews"
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
	out      *appstorefront.CartResult
	err      error
	gotToken string
}

func (s *stubGetCart) Execute(_ context.Context, token string) (*appstorefront.CartResult, error) {
	s.gotToken = token
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubUpdateCartItems struct {
	out      *appstorefront.CartResult
	err      error
	gotToken string
	gotLines []appstorefront.UpdateCartItemLineInput
}

func (s *stubUpdateCartItems) Execute(_ context.Context, token string, lines []appstorefront.UpdateCartItemLineInput) (*appstorefront.CartResult, error) {
	s.gotToken, s.gotLines = token, lines
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubApplyCoupon struct {
	out      *appstorefront.CartResult
	err      error
	gotToken string
	gotCode  string
}

func (s *stubApplyCoupon) Execute(_ context.Context, token string, code string) (*appstorefront.CartResult, error) {
	s.gotToken, s.gotCode = token, code
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubSubmitReview struct {
	out   *reviewsDomain.Review
	err   error
	gotIn appstorefront.SubmitReviewInput
}

func (s *stubSubmitReview) Execute(_ context.Context, in appstorefront.SubmitReviewInput) (*reviewsDomain.Review, error) {
	s.gotIn = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &reviewsDomain.Review{ID: "review-1", CustomerName: in.CustomerName, Rating: in.Rating, Body: in.Body}, nil
}

type stubListReviews struct {
	out *appstorefront.ReviewsResult
	err error
}

func (s *stubListReviews) Execute(_ context.Context, _ appstorefront.ListReviewsInput) (*appstorefront.ReviewsResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appstorefront.ReviewsResult{}, nil
}

type stubCreateWishlist struct {
	out       *storefrontDomain.Wishlist
	err       error
	gotChanID string
}

func (s *stubCreateWishlist) Execute(_ context.Context, salesChannelID string) (*storefrontDomain.Wishlist, error) {
	s.gotChanID = salesChannelID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &storefrontDomain.Wishlist{Token: "wishlist-token-1", SalesChannelID: salesChannelID}, nil
}

type stubGetWishlist struct {
	out      *storefrontDomain.Wishlist
	err      error
	gotToken string
}

func (s *stubGetWishlist) Execute(_ context.Context, token string) (*storefrontDomain.Wishlist, error) {
	s.gotToken = token
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubToggleWishlistItem struct {
	out          *storefrontDomain.Wishlist
	err          error
	gotToken     string
	gotProductID string
}

func (s *stubToggleWishlistItem) Execute(_ context.Context, token, productID string) (*storefrontDomain.Wishlist, error) {
	s.gotToken, s.gotProductID = token, productID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &storefrontDomain.Wishlist{Token: token, ProductIDs: []string{productID}}, nil
}

type stubGetProductRecommendations struct {
	out     []appstorefront.ProductSummary
	err     error
	gotChan string
	gotID   string
}

func (s *stubGetProductRecommendations) Execute(_ context.Context, salesChannelID, productID string) ([]appstorefront.ProductSummary, error) {
	s.gotChan, s.gotID = salesChannelID, productID
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubRequestMagicLink struct {
	err   error
	gotIn appstorefront.RequestMagicLinkInput
}

func (s *stubRequestMagicLink) Execute(_ context.Context, in appstorefront.RequestMagicLinkInput) error {
	s.gotIn = in
	return s.err
}

type stubVerifySession struct {
	out      *storefrontDomain.CustomerSession
	err      error
	gotChan  string
	gotToken string
}

func (s *stubVerifySession) Execute(_ context.Context, salesChannelID, token string) (*storefrontDomain.CustomerSession, error) {
	s.gotChan, s.gotToken = salesChannelID, token
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &storefrontDomain.CustomerSession{Token: "session-token-1", SalesChannelID: salesChannelID}, nil
}

type stubGetAccount struct {
	out      *appstorefront.Account
	err      error
	gotChan  string
	gotToken string
}

func (s *stubGetAccount) Execute(_ context.Context, salesChannelID, sessionToken string) (*appstorefront.Account, error) {
	s.gotChan, s.gotToken = salesChannelID, sessionToken
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appstorefront.Account{Email: "shopper@example.com"}, nil
}

type stubListAccountOrders struct {
	out      []appstorefront.AccountOrder
	err      error
	gotToken string
}

func (s *stubListAccountOrders) Execute(_ context.Context, _, sessionToken string) ([]appstorefront.AccountOrder, error) {
	s.gotToken = sessionToken
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

type stubLogout struct {
	err      error
	gotToken string
}

func (s *stubLogout) Execute(_ context.Context, sessionToken string) error {
	s.gotToken = sessionToken
	return s.err
}

type stubCreatePaymentIntent struct {
	out      *appstorefront.PaymentIntentResult
	err      error
	gotChan  string
	gotToken string
	gotAddr  appstorefront.CheckoutAddressInput
	gotTitle string
}

func (s *stubCreatePaymentIntent) Execute(_ context.Context, salesChannelID, cartToken string, shipping appstorefront.CheckoutAddressInput, shippingMethodTitle string) (*appstorefront.PaymentIntentResult, error) {
	s.gotChan, s.gotToken, s.gotAddr, s.gotTitle = salesChannelID, cartToken, shipping, shippingMethodTitle
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appstorefront.PaymentIntentResult{ClientSecret: "pi_secret_1", PublishableKey: "pk_test_1", AmountMinorUnits: 1000, CurrencyCode: "EUR"}, nil
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

	gotIn appstorefront.ListProductsInput
}

func (s *stubListProducts) Execute(_ context.Context, _ string, in appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error) {
	s.gotIn = in
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
	getShippingOptions := &stubGetShippingOptions{}
	applyCoupon := &stubApplyCoupon{}
	submitReview := &stubSubmitReview{}
	listReviews := &stubListReviews{}
	createWishlist := &stubCreateWishlist{}
	getWishlist := &stubGetWishlist{}
	toggleWishlistItem := &stubToggleWishlistItem{}
	getProductRecommendations := &stubGetProductRecommendations{}
	requestMagicLink := &stubRequestMagicLink{}
	verifySession := &stubVerifySession{}
	getAccount := &stubGetAccount{}
	listAccountOrders := &stubListAccountOrders{}
	logout := &stubLogout{}
	createPaymentIntent := &stubCreatePaymentIntent{}
	var catalog storefronthandler.CatalogUseCases
	var news storefronthandler.NewsletterUseCases
	for _, e := range extra {
		switch v := e.(type) {
		case storefronthandler.NewsletterUseCases:
			news = v
		case storefronthandler.CatalogUseCases:
			catalog = v
		case *stubGetShippingOptions:
			getShippingOptions = v
		case *stubApplyCoupon:
			applyCoupon = v
		case *stubSubmitReview:
			submitReview = v
		case *stubListReviews:
			listReviews = v
		case *stubCreateWishlist:
			createWishlist = v
		case *stubGetWishlist:
			getWishlist = v
		case *stubToggleWishlistItem:
			toggleWishlistItem = v
		case *stubGetProductRecommendations:
			getProductRecommendations = v
		case *stubRequestMagicLink:
			requestMagicLink = v
		case *stubVerifySession:
			verifySession = v
		case *stubGetAccount:
			getAccount = v
		case *stubListAccountOrders:
			listAccountOrders = v
		case *stubLogout:
			logout = v
		case *stubCreatePaymentIntent:
			createPaymentIntent = v
		}
	}
	return storefronthandler.NewHandler(
		resolve, get, create, getCart, update, checkout, listProducts, getProduct, listCategories, lookupOrder,
		getShippingOptions, applyCoupon, submitReview, listReviews,
		createWishlist, getWishlist, toggleWishlistItem, getProductRecommendations,
		requestMagicLink, verifySession, getAccount, listAccountOrders, logout,
		createPaymentIntent, catalog, news,
	)
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

func TestGetCartHandler_includesCouponAndDiscount(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	code := "SUMMER10"
	get := &stubGetCart{out: &appstorefront.CartResult{
		Cart:               &storefrontDomain.Cart{Token: "cart-token-1", CouponCode: &code},
		DiscountMinorUnits: 100,
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, get, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/carts/cart-token-1", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"coupon_code":"SUMMER10"`)
	assert.Contains(t, w.Body.String(), `"discount_minor_units":100`)
}

func TestApplyCouponHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	code := "SUMMER10"
	apply := &stubApplyCoupon{out: &appstorefront.CartResult{
		Cart:               &storefrontDomain.Cart{Token: "cart-token-1", CouponCode: &code},
		DiscountMinorUnits: 100,
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, apply)

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/carts/cart-token-1/coupon", `{"code":"summer10"}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "cart-token-1", apply.gotToken)
	assert.Equal(t, "summer10", apply.gotCode)
	assert.Contains(t, w.Body.String(), `"discount_minor_units":100`)
}

func TestApplyCouponHandler_notRedeemableIs422(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	apply := &stubApplyCoupon{err: couponsDomain.ErrCouponNotRedeemable}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, apply)

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/carts/cart-token-1/coupon", `{"code":"NOPE"}`, "test-key")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestApplyCouponHandler_cartNotFoundIs404(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	apply := &stubApplyCoupon{err: storefrontDomain.ErrCartNotFound}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, apply)

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/carts/unknown-token/coupon", `{"code":"X"}`, "test-key")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateCartItemsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	update := &stubUpdateCartItems{out: &appstorefront.CartResult{Cart: &storefrontDomain.Cart{Token: "cart-token-1", Items: []storefrontDomain.CartItem{
		{SKUCode: "SKU-A", Quantity: 2, PriceMinorUnits: 500, CurrencyCode: "EUR"},
	}}}}
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

func TestListStorefrontProductsHandler_includesCategoryFacets(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	list := &stubListProducts{out: &appstorefront.ListProductsResult{
		Total: 1,
		CategoryFacets: []appstorefront.CategoryFacet{
			{CategoryID: "cat-1", CategoryName: "Shoes", Count: 3},
		},
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, list, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"category_name":"Shoes"`)
	assert.Contains(t, w.Body.String(), `"count":3`)
}

func TestListStorefrontProductsHandler_parsesProductIDsQueryParam(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	list := &stubListProducts{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, list, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products?product_ids=p1,p2", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"p1", "p2"}, list.gotIn.ProductIDs)
}

func TestListStorefrontProductsHandler_parsesExcludeProductIDsQueryParam(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	list := &stubListProducts{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, list, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products?exclude_product_ids=p3", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"p3"}, list.gotIn.ExcludeProductIDs)
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

func TestGetStorefrontProductRecommendationsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	price := int64(500)
	rec := &stubGetProductRecommendations{out: []appstorefront.ProductSummary{
		{Product: pmsDomain.Product{ID: "product-2", Name: "Gadget"}, PriceMinorUnits: &price},
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, rec)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products/product-1/recommendations", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "channel-1", rec.gotChan)
	assert.Equal(t, "product-1", rec.gotID)
	assert.Contains(t, w.Body.String(), `"name":"Gadget"`)
}

func TestGetStorefrontProductRecommendationsHandler_emptyIsOK(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products/product-1/recommendations", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"data":[]`)
}

func TestListStorefrontReviewsHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	listReviews := &stubListReviews{out: &appstorefront.ReviewsResult{
		Reviews: []reviewsDomain.Review{{ID: "review-1", CustomerName: "Ada", Rating: 5, Body: "Great"}},
		Summary: reviewsDomain.Summary{Average: 5, Count: 1},
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, listReviews)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/products/product-1/reviews", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"customer_name":"Ada"`)
	assert.Contains(t, w.Body.String(), `"average":5`)
}

func TestSubmitStorefrontReviewHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	submit := &stubSubmitReview{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, submit)

	body := `{"customer_name":"Ada","customer_email":"ada@example.com","rating":5,"body":"Loved it"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/products/product-1/reviews", body, "test-key")

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "product-1", submit.gotIn.ProductID)
	assert.Equal(t, "channel-1", submit.gotIn.SalesChannelID)
	assert.Equal(t, "Ada", submit.gotIn.CustomerName)
	assert.Equal(t, 5, submit.gotIn.Rating)
}

func TestSubmitStorefrontReviewHandler_productNotAvailableIs404(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	submit := &stubSubmitReview{err: storefrontDomain.ErrProductNotAvailable}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, submit)

	body := `{"customer_name":"Ada","customer_email":"ada@example.com","rating":5,"body":"Loved it"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/products/unknown-id/reviews", body, "test-key")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestSubmitStorefrontReviewHandler_invalidRatingIs422(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	submit := &stubSubmitReview{err: reviewsDomain.ErrInvalidRating}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, submit)

	body := `{"customer_name":"Ada","customer_email":"ada@example.com","rating":9,"body":"Loved it"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/products/product-1/reviews", body, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
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

func TestCreateWishlistHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	create := &stubCreateWishlist{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, create)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/wishlists", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "channel-1", create.gotChanID)
	assert.Contains(t, w.Body.String(), `"token":"wishlist-token-1"`)
}

func TestGetWishlistHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	get := &stubGetWishlist{out: &storefrontDomain.Wishlist{Token: "wishlist-token-1", ProductIDs: []string{"product-1"}}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, get)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/wishlists/wishlist-token-1", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "wishlist-token-1", get.gotToken)
	assert.Contains(t, w.Body.String(), `"product-1"`)
}

func TestGetWishlistHandler_notFoundIs404(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	get := &stubGetWishlist{err: storefrontDomain.ErrWishlistNotFound}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, get)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/wishlists/missing-token", "", "test-key")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestToggleWishlistItemHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	toggle := &stubToggleWishlistItem{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, toggle)

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/wishlists/wishlist-token-1/items", `{"product_id":"product-1"}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "wishlist-token-1", toggle.gotToken)
	assert.Equal(t, "product-1", toggle.gotProductID)
}

func TestToggleWishlistItemHandler_productNotAvailableIs422(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	toggle := &stubToggleWishlistItem{err: storefrontDomain.ErrProductNotAvailable}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, toggle)

	w := doRequest(newTestRouter(h), http.MethodPut, "/api/v1/storefront/v1/wishlists/wishlist-token-1/items", `{"product_id":"missing-product"}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRequestMagicLinkHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	request := &stubRequestMagicLink{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, request)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/auth/magic-link", `{"email":"shopper@example.com","redirect_url":"https://shop.example.com/verify"}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "channel-1", request.gotIn.SalesChannelID)
	assert.Equal(t, "shopper@example.com", request.gotIn.Email)
}

// Old clients may still send redirect_url; it's accepted and ignored.
func TestRequestMagicLinkHandler_worksWithoutRedirectURL(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	request := &stubRequestMagicLink{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, request)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/auth/magic-link", `{"email":"shopper@example.com"}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "shopper@example.com", request.gotIn.Email)
}

func TestRequestMagicLinkHandler_signInNotConfigured(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	request := &stubRequestMagicLink{err: channelsDomain.ErrSignInNotConfigured}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, request)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/auth/magic-link", `{"email":"shopper@example.com"}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "SIGN_IN_NOT_CONFIGURED")
}

func TestVerifySessionHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	verify := &stubVerifySession{}
	getAccount := &stubGetAccount{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, verify, getAccount)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/auth/session", `{"token":"LINK123"}`, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "LINK123", verify.gotToken)
	assert.Contains(t, w.Body.String(), `"session_token":"session-token-1"`)
	assert.Contains(t, w.Body.String(), `"email":"shopper@example.com"`)
}

func TestVerifySessionHandler_invalidTokenIs422(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	verify := &stubVerifySession{err: storefrontDomain.ErrInvalidOrExpiredMagicLink}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, verify)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/auth/session", `{"token":"bad-token"}`, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetAccountHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	getAccount := &stubGetAccount{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, getAccount)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/account/session-token-1", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "channel-1", getAccount.gotChan)
	assert.Equal(t, "session-token-1", getAccount.gotToken)
}

func TestGetAccountHandler_notFoundIs404(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	getAccount := &stubGetAccount{err: storefrontDomain.ErrSessionNotFound}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, getAccount)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/account/missing-session", "", "test-key")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListAccountOrdersHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	list := &stubListAccountOrders{out: []appstorefront.AccountOrder{{OrderNumber: "WEB-1", TotalMinorUnits: 1000, CurrencyCode: "EUR"}}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, list)

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/storefront/v1/account/session-token-1/orders", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"order_number":"WEB-1"`)
}

func TestLogoutHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	logout := &stubLogout{}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, logout)

	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/storefront/v1/account/session-token-1", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-token-1", logout.gotToken)
}

func TestCreatePaymentIntentHandler_success(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	createPaymentIntent := &stubCreatePaymentIntent{out: &appstorefront.PaymentIntentResult{
		ClientSecret: "pi_secret_1", PublishableKey: "pk_test_1", AmountMinorUnits: 4599, CurrencyCode: "EUR",
	}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, createPaymentIntent)

	body := `{"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"}}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/payment-intent", body, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "cart-token-1", createPaymentIntent.gotToken)
	assert.Equal(t, "channel-1", createPaymentIntent.gotChan)
	assert.Contains(t, w.Body.String(), `"client_secret":"pi_secret_1"`)
	assert.Contains(t, w.Body.String(), `"publishable_key":"pk_test_1"`)
	assert.Contains(t, w.Body.String(), `"amount_minor_units":4599`)
}

func TestCreatePaymentIntentHandler_notSupportedIs422(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	createPaymentIntent := &stubCreatePaymentIntent{err: storefrontDomain.ErrCardPaymentNotSupported}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, createPaymentIntent)

	body := `{"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"}}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/payment-intent", body, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCheckoutCartHandler_withPaymentIntentID_passesItThrough(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	checkout := &stubCheckoutCart{out: &appstorefront.CheckoutResult{OrderNumber: "WEB-ABCD1234", TotalMinorUnits: 1000, CurrencyCode: "EUR"}}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, checkout, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	body := `{"email":"shopper@example.com","billing":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"payment_intent_id":"pi_123"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/checkout", body, "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pi_123", checkout.gotInput.PaymentIntentID)
}

func TestCheckoutCartHandler_cardPaymentMismatchIs422(t *testing.T) {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	checkout := &stubCheckoutCart{err: storefrontDomain.ErrCardPaymentMismatch}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, checkout, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{})

	body := `{"email":"shopper@example.com","billing":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"shipping":{"first_name":"A","last_name":"B","address1":"Street 1","city":"Amsterdam","postcode":"1000AA","country":"NL"},"payment_intent_id":"pi_123"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/storefront/v1/carts/cart-token-1/checkout", body, "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
