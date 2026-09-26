package pms

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

type previewProductMergeUC interface {
	Execute(ctx context.Context, in apppms.PreviewProductMergeInput) (*apppms.ProductMergePreview, error)
}

type mergeProductsUC interface {
	Execute(ctx context.Context, in apppms.MergeProductsInput) (*apppms.MergeProductsResult, error)
}

// ProductMergeUseCases groups the merge endpoints' use cases.
type ProductMergeUseCases struct {
	Preview previewProductMergeUC
	Merge   mergeProductsUC
}

func (h *Handler) PreviewProductMerge(c *gin.Context, id genpms.Id) {
	var req genpms.ProductMergeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetProductId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "target_product_id is required")
		return
	}
	preview, err := h.merge.Preview.Execute(c.Request.Context(), apppms.PreviewProductMergeInput{
		SourceProductID: id, TargetProductID: req.TargetProductId, Pairs: fromGenMergePairs(req.SkuMapping),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genpms.ProductMergePreview{
		Source: toGenMergePreviewProduct(preview.Source), Target: toGenMergePreviewProduct(preview.Target),
		SkuMapping: toGenMergePairs(preview.Pairs), Targets: make([]genpms.MergePreviewTarget, len(preview.Targets)),
	}
	for i, t := range preview.Targets {
		resp.Targets[i] = genpms.MergePreviewTarget{SkuId: t.SKUID, OnHandAfter: t.OnHandAfter, CostBeforeMinorUnits: t.CostBefore, CostAfterMinorUnits: t.CostAfter}
	}
	c.JSON(http.StatusOK, genpms.ProductMergePreviewResponse{Data: &resp})
}

func (h *Handler) MergeProduct(c *gin.Context, id genpms.Id) {
	var req genpms.ProductMergeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetProductId == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "target_product_id is required")
		return
	}
	mergedBy := ""
	if claims := middleware.ClaimsFromGin(c); claims != nil {
		mergedBy = claims.UserID
	}
	result, err := h.merge.Merge.Execute(c.Request.Context(), apppms.MergeProductsInput{
		SourceProductID: id, TargetProductID: req.TargetProductId, Pairs: fromGenMergePairs(req.SkuMapping), MergedBy: mergedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genpms.ProductMergeResult{TargetProductId: result.TargetProductID, Skus: make([]genpms.MergedSKU, len(result.SKUs))}
	for i, s := range result.SKUs {
		resp.Skus[i] = genpms.MergedSKU{SourceSkuId: s.SourceSKUID, SourceSkuCode: s.SourceSKUCode, SourceBarcode: s.SourceBarcode, TargetSkuId: s.TargetSKUID}
	}
	c.JSON(http.StatusOK, genpms.ProductMergeResultResponse{Data: &resp})
}

func fromGenMergePairs(pairs *[]genpms.SKUMergePair) []pmsDomain.SKUMergePair {
	if pairs == nil {
		return nil
	}
	out := make([]pmsDomain.SKUMergePair, len(*pairs))
	for i, p := range *pairs {
		out[i] = pmsDomain.SKUMergePair{SourceSKUID: p.SourceSkuId, TargetSKUID: p.TargetSkuId}
	}
	return out
}

func toGenMergePairs(pairs []pmsDomain.SKUMergePair) []genpms.SKUMergePair {
	out := make([]genpms.SKUMergePair, len(pairs))
	for i, p := range pairs {
		out[i] = genpms.SKUMergePair{SourceSkuId: p.SourceSKUID, TargetSkuId: p.TargetSKUID}
	}
	return out
}

func toGenMergePreviewProduct(p apppms.MergePreviewProduct) genpms.MergePreviewProduct {
	out := genpms.MergePreviewProduct{Id: p.ID, Name: p.Name, Skus: make([]genpms.MergePreviewSKU, len(p.SKUs))}
	for i, s := range p.SKUs {
		out.Skus[i] = genpms.MergePreviewSKU{Id: s.ID, SkuCode: s.SKUCode, Barcode: s.Barcode, OnHand: s.OnHand, CostPriceMinorUnits: s.CostPriceMinorUnits}
	}
	return out
}
