package pms

import (
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) ResolveBarcode(c *gin.Context, code string) {
	result, err := h.resolveBarcode.Execute(c.Request.Context(), code)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genpms.ResolveBarcodeResult{
		Sku: toGenSKU(result.SKU), UnitsPerScan: result.UnitsPerScan, ProductName: result.ProductName,
		WeightGrams: result.WeightGrams, LineTotalMinorUnits: result.LineTotalMinorUnits,
	}
	c.JSON(http.StatusOK, genpms.ResolveBarcodeResponse{Data: &resp})
}

func (h *Handler) ListBarcodeAliases(c *gin.Context, id genpms.Id) {
	aliases, err := h.listBarcodeAliases.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.BarcodeAlias, len(aliases))
	for i, a := range aliases {
		resp[i] = toGenBarcodeAlias(a)
	}
	c.JSON(http.StatusOK, genpms.BarcodeAliasListResponse{Data: &resp})
}

func (h *Handler) CreateBarcodeAlias(c *gin.Context, id genpms.Id) {
	var req genpms.CreateBarcodeAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	var unitsPerScan int64
	if req.UnitsPerScan != nil {
		unitsPerScan = *req.UnitsPerScan
	}
	alias, err := h.createBarcodeAlias.Execute(c.Request.Context(), apppms.CreateBarcodeAliasInput{
		SKUID: id, Code: req.Code, UnitsPerScan: unitsPerScan,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBarcodeAlias(*alias)
	c.JSON(http.StatusCreated, genpms.BarcodeAliasResponse{Data: &resp})
}

func (h *Handler) DeleteBarcodeAlias(c *gin.Context, id genpms.Id) {
	err := h.deleteBarcodeAlias.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

func toGenBarcodeAlias(a pmsDomain.BarcodeAlias) genpms.BarcodeAlias {
	return genpms.BarcodeAlias{
		Id: a.ID, SkuId: a.SKUID, Code: a.Code, UnitsPerScan: a.UnitsPerScan,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}

func (h *Handler) ListScaleCodeAliases(c *gin.Context, id genpms.Id) {
	aliases, err := h.listScaleCodeAliases.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.ScaleCodeAlias, len(aliases))
	for i, a := range aliases {
		resp[i] = toGenScaleCodeAlias(a)
	}
	c.JSON(http.StatusOK, genpms.ScaleCodeAliasListResponse{Data: &resp})
}

func (h *Handler) CreateScaleCodeAlias(c *gin.Context, id genpms.Id) {
	var req genpms.CreateScaleCodeAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	alias, err := h.createScaleCodeAlias.Execute(c.Request.Context(), apppms.CreateScaleCodeAliasInput{SKUID: id, Code: req.Code})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenScaleCodeAlias(*alias)
	c.JSON(http.StatusCreated, genpms.ScaleCodeAliasResponse{Data: &resp})
}

func (h *Handler) DeleteScaleCodeAlias(c *gin.Context, id genpms.Id) {
	err := h.deleteScaleCodeAlias.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genpms.EmptyResponse{})
}

func toGenScaleCodeAlias(a pmsDomain.ScaleCodeAlias) genpms.ScaleCodeAlias {
	return genpms.ScaleCodeAlias{
		Id: a.ID, SkuId: a.SKUID, Code: a.Code,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}
