package pms

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

// ── Product handlers ─────────────────────────────────────────────────────────

func (h *Handler) ListProducts(c *gin.Context, params genpms.ListProductsParams) {
	in := apppms.ListProductsInput{}
	if params.Page != nil {
		in.Page = *params.Page
	}
	if params.PerPage != nil {
		in.PerPage = *params.PerPage
	}
	if params.Search != nil {
		in.Search = *params.Search
	}
	result, err := h.listProducts.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.Product, len(result.Products))
	for i, p := range result.Products {
		resp[i] = toGenProductListItem(p)
	}
	total := int(result.Total)
	c.JSON(http.StatusOK, genpms.ProductListResponse{Data: &resp, Total: &total})
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var req genpms.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	var tags []string
	if req.Tags != nil {
		tags = *req.Tags
	}
	product, err := h.createProduct.Execute(c.Request.Context(), apppms.CreateProductInput{
		Name:                        req.Name,
		Description:                 description,
		CategoryID:                  req.CategoryId,
		Status:                      pmsDomain.ProductStatus(req.Status),
		HasVariants:                 req.HasVariants != nil && *req.HasVariants,
		SyncVariantsAsWooVariations: req.SyncVariantsAsWooVariations != nil && *req.SyncVariantsAsWooVariations,
		Tags:                        tags,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProduct(*product)
	c.JSON(http.StatusCreated, genpms.ProductResponse{Data: &resp})
}

func (h *Handler) GetProduct(c *gin.Context, id genpms.Id) {
	detail, err := h.getProduct.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProductDetail(*detail)
	c.JSON(http.StatusOK, genpms.ProductDetailResponse{Data: &resp})
}

func (h *Handler) DuplicateProduct(c *gin.Context, id genpms.Id) {
	detail, err := h.duplicateProduct.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProductDetail(*detail)
	c.JSON(http.StatusOK, genpms.ProductDetailResponse{Data: &resp})
}

func (h *Handler) UpdateProduct(c *gin.Context, id genpms.Id) {
	var req genpms.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	var tags []string
	if req.Tags != nil {
		tags = *req.Tags
	}
	product, err := h.updateProduct.Execute(c.Request.Context(), apppms.UpdateProductInput{
		ID:                          id,
		Name:                        req.Name,
		Description:                 description,
		CategoryID:                  req.CategoryId,
		Status:                      pmsDomain.ProductStatus(req.Status),
		HasVariants:                 req.HasVariants != nil && *req.HasVariants,
		SyncVariantsAsWooVariations: req.SyncVariantsAsWooVariations != nil && *req.SyncVariantsAsWooVariations,
		Tags:                        tags,
		Slug:                        req.Slug,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenProduct(*product)
	c.JSON(http.StatusOK, genpms.ProductResponse{Data: &resp})
}

func (h *Handler) DeleteProduct(c *gin.Context, id genpms.Id) {
	err := h.deleteProduct.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

func (h *Handler) ListProductHistory(c *gin.Context, id genpms.Id) {
	history, err := h.listProductHistory.Execute(c.Request.Context(), pmsDomain.ProductHistoryEntityProduct, id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.ProductHistoryListResponse{Data: toGenProductHistoryEntries(history)})
}

func (h *Handler) ListVariantHistory(c *gin.Context, id genpms.Id) {
	history, err := h.listProductHistory.Execute(c.Request.Context(), pmsDomain.ProductHistoryEntityVariant, id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.ProductHistoryListResponse{Data: toGenProductHistoryEntries(history)})
}

func toGenProductHistoryEntries(history []pmsDomain.ProductHistory) *[]genpms.ProductHistoryEntry {
	resp := make([]genpms.ProductHistoryEntry, len(history))
	for i, h := range history {
		resp[i] = genpms.ProductHistoryEntry{
			Id: h.ID, EntityType: genpms.ProductHistoryEntryEntityType(h.EntityType), EntityId: h.EntityID,
			Field: genpms.ProductHistoryEntryField(h.Field), OldValue: h.OldValue, NewValue: h.NewValue, ChangedAt: h.ChangedAt,
		}
	}
	return &resp
}

func toGenProduct(p pmsDomain.Product) genpms.Product {
	syncVariantsAsWooVariations := p.SyncVariantsAsWooVariations
	return genpms.Product{
		Id:                          p.ID,
		TenantId:                    p.TenantID,
		CategoryId:                  p.CategoryID,
		Name:                        p.Name,
		Slug:                        p.Slug,
		Description:                 p.Description,
		Status:                      genpms.ProductStatus(p.Status),
		HasVariants:                 p.HasVariants,
		SyncVariantsAsWooVariations: &syncVariantsAsWooVariations,
		Tags:                        p.Tags,
		CreatedAt:                   p.CreatedAt,
		UpdatedAt:                   p.UpdatedAt,
	}
}

// toGenProductListItem overlays the featured image and first variant's
// price/sale price/currency onto the base Product shape -- only the list
// response (ListProducts) has this data; detail/create/update responses
// go through the plain toGenProduct instead (see Product's own doc
// comment in openapi.yaml).
func toGenProductListItem(entry apppms.POSProductWithSKUs) genpms.Product {
	p := toGenProduct(entry.Product)
	p.ImageUrl = entry.ImageURL
	if len(entry.SKUs) > 0 {
		sku := entry.SKUs[0].SKU
		price := sku.PriceMinorUnits
		currency := sku.CurrencyCode
		p.PriceMinorUnits = &price
		p.SalePriceMinorUnits = sku.SalePriceMinorUnits
		p.CurrencyCode = &currency
	}
	return p
}

func linkedSKUSummary(linked map[string]apppms.LinkedSKU, id *string) *genpms.SKULinkSummary {
	if id == nil {
		return nil
	}
	l, ok := linked[*id]
	if !ok {
		return nil
	}
	return &genpms.SKULinkSummary{Id: l.ID, ProductName: l.ProductName, SkuCode: l.SKUCode, Barcode: l.Barcode}
}

func toGenProductDetail(d apppms.ProductDetail) genpms.ProductDetail {
	variants := make([]genpms.VariantWithSKUs, len(d.Variants))
	for i, vd := range d.Variants {
		skus := make([]genpms.SKU, len(vd.SKUs))
		for j, s := range vd.SKUs {
			skus[j] = toGenSKU(s)
			skus[j].CaseOfSku = linkedSKUSummary(d.LinkedSKUs, s.CaseOfSKUID)
			skus[j].PortionedFromSku = linkedSKUSummary(d.LinkedSKUs, s.PortionedFromSKUID)
		}
		variants[i] = genpms.VariantWithSKUs{
			Id:              vd.Variant.ID,
			TenantId:        vd.Variant.TenantID,
			ProductId:       vd.Variant.ProductID,
			Name:            vd.Variant.Name,
			Position:        vd.Variant.Position,
			FeaturedMediaId: vd.Variant.FeaturedMediaID,
			CreatedAt:       vd.Variant.CreatedAt,
			UpdatedAt:       vd.Variant.UpdatedAt,
			Skus:            &skus,
		}
	}
	return genpms.ProductDetail{
		Product:  toGenProduct(d.Product),
		Variants: variants,
	}
}
