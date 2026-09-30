package storefront

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	genstorefront "github.com/adirkuhn/commerceos/api/storefront"
	appstorefront "github.com/adirkuhn/commerceos/application/storefront"
	"github.com/adirkuhn/commerceos/domain/locale"
	storefrontDomain "github.com/adirkuhn/commerceos/domain/storefront"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func (h *Handler) ListStorefrontProducts(c *gin.Context, params genstorefront.ListStorefrontProductsParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	in := appstorefront.ListProductsInput{CategoryID: params.CategoryId}
	if params.ProductIds != nil {
		in.ProductIDs = *params.ProductIds
	}
	if params.ExcludeProductIds != nil {
		in.ExcludeProductIDs = *params.ExcludeProductIds
	}
	if params.Search != nil {
		in.Search = *params.Search
	}
	if params.Page != nil {
		in.Page = *params.Page
	}
	if params.PerPage != nil {
		in.PerPage = *params.PerPage
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	var result *appstorefront.ListProductsResult
	var err error
	if params.CollectionId != nil {
		result, err = h.catalog.ListCollectionProducts.Execute(ctx, *resolved.SalesChannelID, params.CollectionId.String(), in)
		if notFound(c, err) {
			return
		}
	} else {
		result, err = h.listProducts.Execute(ctx, *resolved.SalesChannelID, in)
	}
	if err := mapError(c, err); err != nil {
		return
	}
	items := make([]genstorefront.StorefrontProductSummary, len(result.Products))
	for i, p := range result.Products {
		items[i] = toGenProductSummary(p.In(shopLocale(params.Locale)))
	}
	facets := make([]genstorefront.StorefrontCategoryFacet, len(result.CategoryFacets))
	for i, f := range result.CategoryFacets {
		facets[i] = genstorefront.StorefrontCategoryFacet{CategoryId: f.CategoryID, CategoryName: f.CategoryName, Count: f.Count}
	}
	total := result.Total
	middleware.OK(c, struct {
		Items          []genstorefront.StorefrontProductSummary `json:"items"`
		Total          int64                                    `json:"total"`
		CategoryFacets []genstorefront.StorefrontCategoryFacet  `json:"category_facets"`
	}{Items: items, Total: total, CategoryFacets: facets})
}

func (h *Handler) GetStorefrontProduct(c *gin.Context, id string, params genstorefront.GetStorefrontProductParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	detail, err := h.getProduct.Execute(ctx, *resolved.SalesChannelID, id)
	if errors.Is(err, storefrontDomain.ErrProductNotAvailable) {
		// A GET-by-id 404s for "doesn't exist" -- unlike a cart line's SKU
		// code (a request-body field, so the same error is a 422 there),
		// this is a plain resource lookup.
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenProductDetail(detail.In(shopLocale(params.Locale))))
}

func (h *Handler) GetStorefrontProductRecommendations(c *gin.Context, id string, params genstorefront.GetStorefrontProductRecommendationsParams) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	summaries, err := h.getProductRecommendations.Execute(ctx, *resolved.SalesChannelID, id)
	if err := mapError(c, err); err != nil {
		return
	}
	items := make([]genstorefront.StorefrontProductSummary, len(summaries))
	for i, p := range summaries {
		items[i] = toGenProductSummary(p.In(shopLocale(params.Locale)))
	}
	middleware.OK(c, items)
}

func (h *Handler) ListStorefrontCategories(c *gin.Context) {
	resolved, ok := h.authenticateChannel(c)
	if !ok {
		return
	}
	ctx := tenant.WithTenantID(c.Request.Context(), resolved.TenantID)
	categories, err := h.listCategories.Execute(ctx)
	if err := mapError(c, err); err != nil {
		return
	}
	genCategories := make([]genstorefront.StorefrontCategory, len(categories))
	for i, cat := range categories {
		genCategories[i] = genstorefront.StorefrontCategory{
			Id: cat.ID, ParentId: cat.ParentID, Name: cat.Name, Slug: cat.Slug, Position: cat.Position,
		}
	}
	middleware.OK(c, genCategories)
}

func toGenProductSummary(p appstorefront.ProductSummary) genstorefront.StorefrontProductSummary {
	return genstorefront.StorefrontProductSummary{
		Id: p.ID, Name: p.Name, Slug: p.Slug, CategoryId: p.CategoryID, Tags: p.Tags,
		ImageUrl: p.ImageURL, PriceMinorUnits: p.PriceMinorUnits, SalePriceMinorUnits: p.SalePriceMinorUnits,
		CurrencyCode: p.CurrencyCode, SkuCode: p.SKUCode, SkuCount: &p.SKUCount, InStock: &p.InStock,
	}
}

func toGenProductDetail(d appstorefront.ProductDetail) genstorefront.StorefrontProduct {
	variants := make([]genstorefront.StorefrontVariant, len(d.Variants))
	for i, v := range d.Variants {
		skus := make([]genstorefront.StorefrontSKU, len(v.SKUs))
		for j, s := range v.SKUs {
			skus[j] = genstorefront.StorefrontSKU{
				Id: s.ID, SkuCode: s.SKUCode, PriceMinorUnits: s.PriceMinorUnits,
				SalePriceMinorUnits: s.SalePriceMinorUnits, CurrencyCode: s.CurrencyCode, StockQuantity: s.StockQuantity,
			}
		}
		variants[i] = genstorefront.StorefrontVariant{Id: v.ID, Name: v.Name, Skus: skus}
	}
	return genstorefront.StorefrontProduct{
		Id: d.ID, Name: d.Name, Slug: d.Slug, Description: d.Description, CategoryId: d.CategoryID,
		Tags: d.Tags, Images: d.Images, Variants: variants,
	}
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// shopLocale is the language the shopper reads the store in; "" (omitted)
// keeps the default-language texts.
func shopLocale(l *genstorefront.ShopLocale) locale.Locale {
	if l == nil {
		return ""
	}
	return locale.Locale(*l)
}
