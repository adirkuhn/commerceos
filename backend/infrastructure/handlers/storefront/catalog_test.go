package storefront_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/domain/collections"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	storefronthandler "github.com/adirkuhn/commerceos/infrastructure/handlers/storefront"
)

const colID = "33333333-3333-4333-8333-333333333333"

type catalogStub struct {
	err           error
	gotCollection string
	gotIn         appstorefront.ListProductsInput
	gotSlug       string
}

func (s *catalogStub) ListCollectionProducts() interface {
	Execute(context.Context, string, string, appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error)
} {
	return listColF(func(_ context.Context, _, col string, in appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error) {
		s.gotCollection, s.gotIn = col, in
		return &appstorefront.ListProductsResult{Total: 0}, s.err
	})
}

type listColF func(context.Context, string, string, appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error)

func (f listColF) Execute(ctx context.Context, ch, col string, in appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error) {
	return f(ctx, ch, col, in)
}

type listColsF func(context.Context, int, int) ([]appstorefront.StorefrontCollection, int64, error)

func (f listColsF) Execute(ctx context.Context, p, pp int) ([]appstorefront.StorefrontCollection, int64, error) {
	return f(ctx, p, pp)
}

type getColF func(context.Context, string) (*appstorefront.StorefrontCollection, error)

func (f getColF) Execute(ctx context.Context, slug string) (*appstorefront.StorefrontCollection, error) {
	return f(ctx, slug)
}

type bySlugF func(context.Context, string, string) (*appstorefront.ProductDetail, error)

func (f bySlugF) Execute(ctx context.Context, ch, slug string) (*appstorefront.ProductDetail, error) {
	return f(ctx, ch, slug)
}

type storeF func(context.Context, string) (*appstorefront.StoreInfo, error)

func (f storeF) Execute(ctx context.Context, ch string) (*appstorefront.StoreInfo, error) {
	return f(ctx, ch)
}

func sampleCollection() appstorefront.StorefrontCollection {
	url := "https://cdn/c.jpg"
	c := appstorefront.StorefrontCollection{ImageURL: &url}
	c.ID, c.Name, c.Slug = colID, "Summer", "summer"
	return c
}

func catalogRouter(s *catalogStub) *gin.Engine {
	resolve := &stubResolveStorefrontKey{resolved: channelScopedKey("tenant-1", "channel-1")}
	cat := storefronthandler.CatalogUseCases{
		ListCollectionProducts: s.ListCollectionProducts(),
		ListCollections: listColsF(func(context.Context, int, int) ([]appstorefront.StorefrontCollection, int64, error) {
			return []appstorefront.StorefrontCollection{sampleCollection()}, 1, s.err
		}),
		GetCollection: getColF(func(_ context.Context, slug string) (*appstorefront.StorefrontCollection, error) {
			s.gotSlug = slug
			c := sampleCollection()
			return &c, s.err
		}),
		GetProductBySlug: bySlugF(func(_ context.Context, _, slug string) (*appstorefront.ProductDetail, error) {
			s.gotSlug = slug
			return &appstorefront.ProductDetail{}, s.err
		}),
		GetStoreInfo: storeF(func(context.Context, string) (*appstorefront.StoreInfo, error) {
			return &appstorefront.StoreInfo{Name: "My Shop", CurrencyCode: "EUR"}, s.err
		}),
	}
	h := newTestHandler(resolve, &stubGetCartRecommendations{}, &stubCreateCart{}, &stubGetCart{}, &stubUpdateCartItems{}, &stubCheckoutCart{},
		&stubListProducts{}, &stubGetProduct{}, &stubListCategories{}, &stubLookupOrder{}, cat)
	return newTestRouter(h)
}

func TestCatalog_listProductsByCollection(t *testing.T) {
	s := &catalogStub{}
	w := doRequest(catalogRouter(s), http.MethodGet, "/api/v1/storefront/v1/products?collection_id="+colID+"&page=2", "", "key")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, colID, s.gotCollection)
	assert.Equal(t, 2, s.gotIn.Page)

	w = doRequest(catalogRouter(&catalogStub{err: collections.ErrCollectionNotFound}), http.MethodGet, "/api/v1/storefront/v1/products?collection_id="+colID, "", "key")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCatalog_collections(t *testing.T) {
	s := &catalogStub{}
	w := doRequest(catalogRouter(s), http.MethodGet, "/api/v1/storefront/v1/collections", "", "key")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"image_url":"https://cdn/c.jpg"`)

	w = doRequest(catalogRouter(s), http.MethodGet, "/api/v1/storefront/v1/collections/by-slug/summer", "", "key")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "summer", s.gotSlug)

	w = doRequest(catalogRouter(&catalogStub{err: collections.ErrCollectionNotFound}), http.MethodGet, "/api/v1/storefront/v1/collections/by-slug/x", "", "key")
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, http.StatusUnauthorized, doRequest(catalogRouter(s), http.MethodGet, "/api/v1/storefront/v1/collections", "", "").Code)
}

func TestCatalog_productBySlugAndStore(t *testing.T) {
	s := &catalogStub{}
	w := doRequest(catalogRouter(s), http.MethodGet, "/api/v1/storefront/v1/products/by-slug/red-shoes", "", "key")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "red-shoes", s.gotSlug)

	w = doRequest(catalogRouter(&catalogStub{err: storefrontDomain.ErrProductNotAvailable}), http.MethodGet, "/api/v1/storefront/v1/products/by-slug/nope", "", "key")
	assert.Equal(t, http.StatusNotFound, w.Code)

	w = doRequest(catalogRouter(s), http.MethodGet, "/api/v1/storefront/v1/store", "", "key")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"currency_code":"EUR"`)
}
