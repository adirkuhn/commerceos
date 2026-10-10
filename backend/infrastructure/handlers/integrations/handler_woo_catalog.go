package integrations

import (
	"github.com/adirkuhn/commerceos/infrastructure/httperrs"
	"net/http"

	"github.com/gin-gonic/gin"

	genintegrations "github.com/adirkuhn/commerceos/api/integrations"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ListWooTaxes(c *gin.Context, id genintegrations.Id) {
	options, err := h.listWooTaxes.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.WooTaxClass, len(options))
	for i, o := range options {
		resp[i] = genintegrations.WooTaxClass{Slug: o.Slug, Label: o.Label}
	}
	middleware.OK(c, resp)
}

func (h *Handler) ListWooCategories(c *gin.Context, id genintegrations.Id) {
	categories, err := h.listWooCategories.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.WooCategory, len(categories))
	for i, cat := range categories {
		resp[i] = genintegrations.WooCategory{ExternalId: cat.ExternalID, Name: cat.Name, Slug: cat.Slug}
	}
	middleware.OK(c, resp)
}

func (h *Handler) ListExternalCategoryLinks(c *gin.Context, id genintegrations.Id) {
	links, err := h.listExternalCategoryLinks.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genintegrations.ExternalCategoryLink, len(links))
	for i, l := range links {
		resp[i] = toGenExternalCategoryLink(l)
	}
	middleware.OK(c, resp)
}

func (h *Handler) UpsertExternalCategoryLink(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.UpsertExternalCategoryLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	link, err := h.upsertExternalCategoryLink.Execute(c.Request.Context(), appchannels.UpsertExternalCategoryLinkInput{
		ConnectionID: id, ExternalID: req.ExternalId, CategoryID: req.CategoryId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenExternalCategoryLink(*link))
}

func (h *Handler) DeleteExternalCategoryLink(c *gin.Context, id genintegrations.Id, linkId genintegrations.LinkId) {
	err := h.deleteExternalCategoryLink.Execute(c.Request.Context(), linkId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, nil)
}

func toGenExternalCategoryLink(l channelsDomain.ExternalCategoryLink) genintegrations.ExternalCategoryLink {
	return genintegrations.ExternalCategoryLink{
		Id: l.ID, ConnectionId: l.ConnectionID, Provider: l.Provider,
		ExternalId: l.ExternalID, CategoryId: l.CategoryID,
		CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt,
	}
}

func (h *Handler) ImportWooProducts(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.ImportWooProductsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.DefaultWarehouseId == "" || req.DefaultCurrency == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	job, err := h.createImportJob.Execute(c.Request.Context(), appchannels.CreateImportJobInput{
		ConnectionID:       id,
		DefaultWarehouseID: req.DefaultWarehouseId,
		DefaultCurrency:    req.DefaultCurrency,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusAccepted, middleware.APIResponse{Data: toGenImportJob(*job)})
}

func toGenImportJob(j channelsDomain.ImportJob) genintegrations.ImportJob {
	return genintegrations.ImportJob{
		Id: j.ID, ConnectionId: j.ConnectionID,
		Status:        genintegrations.ImportJobStatus(j.Status),
		Imported:      j.Imported,
		AlreadyLinked: j.AlreadyLinked,
		SkippedType:   j.SkippedType,
		Errors:        emptySliceIfNil(j.Errors),
		ErrorMessage:  j.ErrorMessage,
		CreatedAt:     j.CreatedAt,
		StartedAt:     j.StartedAt,
		FinishedAt:    j.FinishedAt,
	}
}

func (h *Handler) FetchWooOrder(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.FetchWooOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.ExternalOrderId == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	result, err := h.fetchWooOrder.ExecuteOne(c.Request.Context(), appchannels.FetchWooOrderInput{
		ConnectionID: id, ExternalOrderID: req.ExternalOrderId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	var salesOrderID *string
	if result.SalesOrderID != "" {
		salesOrderID = &result.SalesOrderID
	}
	middleware.OK(c, genintegrations.FetchWooOrderResult{
		ExternalOrderId: result.ExternalOrderID, SalesOrderId: salesOrderID,
		Created: result.Created, Updated: result.Updated, Unchanged: result.Unchanged,
		Errors: emptySliceIfNil(result.Errors),
	})
}

func (h *Handler) PushProductToWoo(c *gin.Context, id genintegrations.Id) {
	var req genintegrations.PushProductToWooRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	if req.ProductId == "" {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", httperrs.ErrRequiredFieldsMissing)
		return
	}
	listings, err := h.pushProductToWoo.Execute(c.Request.Context(), appchannels.PushProductToWooInput{
		ProductID: req.ProductId, ConnectionID: id,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListings(listings))
}

func (h *Handler) DeleteAllImportedProducts(c *gin.Context, id genintegrations.Id) {
	result, err := h.deleteAllImportedProds.Execute(c.Request.Context(), appchannels.DeleteAllImportedProductsInput{ConnectionID: id})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genintegrations.DeleteAllImportedProductsResult{
		DeletedSalesOrders: result.DeletedSalesOrders,
		DeletedProducts:    result.DeletedProducts,
		DeletedCategories:  result.DeletedCategories,
		Errors:             emptySliceIfNil(result.Errors),
	})
}

func (h *Handler) LinkExistingWooProduct(c *gin.Context, productId genintegrations.ProductId) {
	var req genintegrations.LinkExistingWooProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	listing, err := h.linkExistingWooProduct.Execute(c.Request.Context(), appchannels.LinkExistingWooProductInput{
		ProductID: productId, ConnectionID: req.ConnectionId, SKU: req.Sku,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, toGenListing(*listing))
}

// emptySliceIfNil normalizes a nil slice to an empty one — a nil []string
// marshals to JSON `null`, but the OpenAPI schema declares errors as an
// array; frontend code that unconditionally reads `.errors.length` crashes
// on `null`. This keeps the wire contract's implied non-nullability true.
func emptySliceIfNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
