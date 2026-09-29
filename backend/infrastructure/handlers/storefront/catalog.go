package storefront

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/domain/collections"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// CatalogUseCases are the storefront catalog reads added for the theme
// system: collections, product-by-slug and store info.
type CatalogUseCases struct {
	ListCollectionProducts interface {
		Execute(ctx context.Context, salesChannelID, collectionID string, in appstorefront.ListProductsInput) (*appstorefront.ListProductsResult, error)
	}
	ListCollections interface {
		Execute(ctx context.Context, page, perPage int) ([]appstorefront.StorefrontCollection, int64, error)
	}
	GetCollection interface {
		Execute(ctx context.Context, slug string) (*appstorefront.StorefrontCollection, error)
	}
	GetProductBySlug interface {
		Execute(ctx context.Context, salesChannelID, slug string) (*appstorefront.ProductDetail, error)
	}
	GetStoreInfo interface {
		Execute(ctx context.Context, salesChannelID string) (*appstorefront.StoreInfo, error)
	}
	GetSitemap interface {
		Execute(ctx context.Context, salesChannelID string) (*appstorefront.Sitemap, error)
	}
}

// sitemapCache: sitemaps change slowly and crawlers fetch them rarely.
const sitemapCache = "public, max-age=3600"

func toGenCollection(c appstorefront.StorefrontCollection) genstorefront.StorefrontCollection {
	id, _ := uuid.Parse(c.ID)
	return genstorefront.StorefrontCollection{
		Id: id, Name: c.Name, Slug: c.Slug, Description: c.Description, ImageUrl: c.ImageURL,
		SeoTitle: c.SEOTitle, SeoDescription: c.SEODescription,
	}
}

func notFound(c *gin.Context, err error) bool {
	if errors.Is(err, collections.ErrCollectionNotFound) || errors.Is(err, storefrontDomain.ErrProductNotAvailable) {
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return true
	}
	return false
}

func (h *Handler) ListStorefrontCollections(c *gin.Context, params genstorefront.ListStorefrontCollectionsParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	page, perPage := 1, 50
	if params.Page != nil {
		page = *params.Page
	}
	if params.PerPage != nil {
		perPage = *params.PerPage
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	list, total, err := h.catalog.ListCollections.Execute(ctx, page, perPage)
	if err := mapError(c, err); err != nil {
		return
	}
	data := make([]genstorefront.StorefrontCollection, len(list))
	for i, col := range list {
		data[i] = toGenCollection(col)
	}
	t := int(total)
	c.JSON(http.StatusOK, genstorefront.StorefrontCollectionListResponse{Data: &data, Total: &t})
}

func (h *Handler) GetStorefrontCollection(c *gin.Context, slug string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	col, err := h.catalog.GetCollection.Execute(tenant.WithTenantID(c.Request.Context(), resolved.TenantID), slug)
	if notFound(c, err) {
		return
	}
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenCollection(*col))
}

func (h *Handler) GetStorefrontProductBySlug(c *gin.Context, slug string) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	detail, err := h.catalog.GetProductBySlug.Execute(tenant.WithTenantID(c.Request.Context(), resolved.TenantID), *resolved.SalesChannelID, slug)
	if notFound(c, err) {
		return
	}
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenProductDetail(*detail))
}

func (h *Handler) GetStorefrontStore(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	info, err := h.catalog.GetStoreInfo.Execute(tenant.WithTenantID(c.Request.Context(), resolved.TenantID), *resolved.SalesChannelID)
	if err := mapError(c, err); err != nil {
		return
	}
	locales := make([]genstorefront.StorefrontStoreLocales, len(info.Locales))
	for i, l := range info.Locales {
		locales[i] = genstorefront.StorefrontStoreLocales(l)
	}
	middleware.OK(c, genstorefront.StorefrontStore{
		Name: info.Name, CurrencyCode: info.CurrencyCode, Locale: genstorefront.StorefrontStoreLocale(info.Locale), Locales: locales,
	})
}

func (h *Handler) GetStorefrontSitemap(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	sm, err := h.catalog.GetSitemap.Execute(tenant.WithTenantID(c.Request.Context(), resolved.TenantID), *resolved.SalesChannelID)
	if err := mapError(c, err); err != nil {
		return
	}
	out := genstorefront.StorefrontSitemap{
		Products:    make([]genstorefront.SitemapLink, len(sm.Products)),
		Collections: make([]genstorefront.SitemapLink, len(sm.Collections)),
	}
	for i, p := range sm.Products {
		out.Products[i] = genstorefront.SitemapLink{Slug: p.Slug, UpdatedAt: p.UpdatedAt}
	}
	for i, col := range sm.Collections {
		out.Collections[i] = genstorefront.SitemapLink{Slug: col.Slug, UpdatedAt: col.UpdatedAt}
	}
	c.Header("Cache-Control", sitemapCache)
	middleware.OK(c, out)
}
