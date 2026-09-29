package storefront_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/domain/collections"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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

type sitemapF func(context.Context, string) (*appstorefront.Sitemap, error)

func (f sitemapF) Execute(ctx context.Context, ch string) (*appstorefront.Sitemap, error) {
	return f(ctx, ch)
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
			return &appstorefront.StoreInfo{Name: "My Shop", CurrencyCode: "EUR", Locale: "pt-BR"}, s.err
		}),
		GetSitemap: sitemapF(func(context.Context, string) (*appstorefront.Sitemap, error) {
			at := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
			return &appstorefront.Sitemap{
				Products:    []pmsDomain.ProductLink{{Slug: "red-shoes", UpdatedAt: at}},
				Collections: []appstorefront.CollectionLink{{Slug: "summer", UpdatedAt: at}},
			}, s.err
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
	assert.Contains(t, w.Body.String(), `"locale":"pt-BR"`)
}

func TestCatalog_sitemap(t *testing.T) {
	w := doRequest(catalogRouter(&catalogStub{}), http.MethodGet, "/api/v1/storefront/v1/sitemap", "", "key")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.JSONEq(t, `{"data":{"products":[{"slug":"red-shoes","updated_at":"2026-09-01T12:00:00Z"}],"collections":[{"slug":"summer","updated_at":"2026-09-01T12:00:00Z"}]},"error":null}`, w.Body.String())
	assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))

	w = doRequest(catalogRouter(&catalogStub{err: storefrontDomain.ErrNoInternalChannel}), http.MethodGet, "/api/v1/storefront/v1/sitemap", "", "key")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, http.StatusUnauthorized, doRequest(catalogRouter(&catalogStub{}), http.MethodGet, "/api/v1/storefront/v1/sitemap", "", "").Code)
}
