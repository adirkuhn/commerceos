package storefront_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	storefronthandler "github.com/adirkuhn/commerceos/infrastructure/handlers/storefront"
)

const slowMovingPath = "/api/v1/storefront/v1/slow-moving-products"

type stubListSlowMoving struct {
	out      *appstorefront.SlowMovingProductsResult
	err      error
	gotDays  int
	gotLimit int
	calls    int
}

func (s *stubListSlowMoving) Execute(_ context.Context, days, limit int) (*appstorefront.SlowMovingProductsResult, error) {
	s.calls++
	s.gotDays, s.gotLimit = days, limit
	return s.out, s.err
}

// slowMovingRouter mounts the handler behind the same OpenAPI request
// validator cmd/api/main.go puts in front of every storefront route, so
// these tests exercise the published contract, not just the handler.
func slowMovingRouter(t *testing.T, resolve *stubResolveStorefrontKey, uc *stubListSlowMoving) *gin.Engine {
	t.Helper()
	spec, err := genstorefront.GetSpec()
	require.NoError(t, err)
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{}, &stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{},
		storefronthandler.CatalogUseCases{ListSlowMovingProducts: uc})
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genstorefront.RegisterHandlers(g, h)
	return r
}

func tenantKey() *stubResolveStorefrontKey {
	return &stubResolveStorefrontKey{resolved: apikeysDomain.ResolvedAPIKey{TenantID: "tenant-1"}}
}

func TestListSlowMovingProductsHandler_success(t *testing.T) {
	uc := &stubListSlowMoving{out: &appstorefront.SlowMovingProductsResult{
		WindowDays: 30,
		Products:   []appstorefront.SlowMovingProduct{{SKUCode: "SKU-A", ProductName: "Widget A", StockOnHand: 12}},
	}}

	w := doRequest(slowMovingRouter(t, tenantKey(), uc), http.MethodGet, slowMovingPath+"?days=30&limit=5", "", "test-key")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 30, uc.gotDays)
	assert.Equal(t, 5, uc.gotLimit)
	var body genstorefront.SlowMovingProductsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.NotNil(t, body.Data)
	assert.Equal(t, 30, body.Data.WindowDays)
	assert.Equal(t, []genstorefront.SlowMovingProduct{{SkuCode: "SKU-A", ProductName: "Widget A", StockOnHand: 12}}, body.Data.Products)
}

func TestListSlowMovingProductsHandler_appliesContractDefaults(t *testing.T) {
	uc := &stubListSlowMoving{out: &appstorefront.SlowMovingProductsResult{WindowDays: 60, Products: []appstorefront.SlowMovingProduct{}}}

	w := doRequest(slowMovingRouter(t, tenantKey(), uc), http.MethodGet, slowMovingPath, "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 60, uc.gotDays)
	assert.Equal(t, 12, uc.gotLimit)
	assert.Contains(t, w.Body.String(), `"products":[]`)
}

func TestListSlowMovingProductsHandler_acceptsTenantWideKey(t *testing.T) {
	// The WooCommerce plugin's key is tenant-wide (no sales channel) --
	// must not be rejected the way the cart/checkout routes reject it.
	uc := &stubListSlowMoving{out: &appstorefront.SlowMovingProductsResult{WindowDays: 60}}

	w := doRequest(slowMovingRouter(t, tenantKey(), uc), http.MethodGet, slowMovingPath, "", "test-key")

	require.Equal(t, http.StatusOK, w.Code)
}

func TestListSlowMovingProductsHandler_missingBearerToken(t *testing.T) {
	uc := &stubListSlowMoving{}

	w := doRequest(slowMovingRouter(t, tenantKey(), uc), http.MethodGet, slowMovingPath, "", "")

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Zero(t, uc.calls)
}

func TestListSlowMovingProductsHandler_invalidKey(t *testing.T) {
	resolve := &stubResolveStorefrontKey{err: assert.AnError}
	uc := &stubListSlowMoving{}

	w := doRequest(slowMovingRouter(t, resolve, uc), http.MethodGet, slowMovingPath, "", "bad-key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Zero(t, uc.calls)
}

func TestListSlowMovingProductsHandler_contractRejectsOutOfRangeParams(t *testing.T) {
	for _, query := range []string{"?days=0", "?days=366", "?limit=0", "?limit=51", "?days=abc"} {
		uc := &stubListSlowMoving{}

		w := doRequest(slowMovingRouter(t, tenantKey(), uc), http.MethodGet, slowMovingPath+query, "", "test-key")

		assert.Equal(t, http.StatusBadRequest, w.Code, query)
		assert.Zero(t, uc.calls, query)
	}
}

func TestListSlowMovingProductsHandler_mapsUseCaseValidationErrorTo422(t *testing.T) {
	uc := &stubListSlowMoving{err: storefrontDomain.ErrInvalidSlowMovingWindow}

	w := doRequest(slowMovingRouter(t, tenantKey(), uc), http.MethodGet, slowMovingPath, "", "test-key")

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "INVALID_SLOW_MOVING_WINDOW")
}
