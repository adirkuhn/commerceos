package pms

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	genpms "github.com/adirkuhn/commerceos/api/pms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) SetManualCostPrice(c *gin.Context, id genpms.Id) {
	var req genpms.SetManualCostPriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	sku, err := h.setManualCostPrice.Execute(c.Request.Context(), id, req.CostPriceMinorUnits)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKU(*sku)
	c.JSON(http.StatusOK, genpms.SKUResponse{Data: &resp})
}

func (h *Handler) ClearCostPriceOverride(c *gin.Context, id genpms.Id) {
	sku, err := h.clearCostPriceOverride.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKU(*sku)
	c.JSON(http.StatusOK, genpms.SKUResponse{Data: &resp})
}

func (h *Handler) CorrectCost(c *gin.Context, id genpms.Id) {
	var req genpms.CorrectCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	var referenceType, referenceID string
	if req.ReferenceType != nil {
		referenceType = *req.ReferenceType
	}
	if req.ReferenceId != nil {
		referenceID = *req.ReferenceId
	}
	sku, err := h.correctCost.Execute(c.Request.Context(), apppms.CorrectCostInput{
		SKUID: id, AdjustmentValueMinorUnits: req.AdjustmentValueMinorUnits,
		ReferenceType: referenceType, ReferenceID: referenceID, OccurredAt: time.Now(),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenSKU(*sku)
	c.JSON(http.StatusOK, genpms.SKUResponse{Data: &resp})
}

func (h *Handler) GetSKUMarginInsight(c *gin.Context, id genpms.Id) {
	insight, err := h.getSKUMarginInsight.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genpms.SKUMarginInsight{
		CostPriceAvailable:               insight.CostPriceAvailable,
		CostPriceMinorUnits:              insight.CostPriceMinorUnits,
		CostPriceSource:                  genpms.SKUCostPriceSource(insight.CostPriceSource),
		EffectiveTargetMarkupBasisPoints: insight.EffectiveTargetMarkupBasisPoints,
		RecommendedPriceMinorUnits:       insight.RecommendedPriceMinorUnits,
		ProfitAtPriceMinorUnits:          insight.ProfitAtPriceMinorUnits,
		MarginAtPriceBasisPoints:         insight.MarginAtPriceBasisPoints,
		MarkupAtPriceBasisPoints:         insight.MarkupAtPriceBasisPoints,
		ProfitAtSalePriceMinorUnits:      insight.ProfitAtSalePriceMinorUnits,
		MarginAtSalePriceBasisPoints:     insight.MarginAtSalePriceBasisPoints,
		MarkupAtSalePriceBasisPoints:     insight.MarkupAtSalePriceBasisPoints,
		TotalOnHandQuantity:              insight.TotalOnHandQuantity,
		TotalProfitPotentialMinorUnits:   insight.TotalProfitPotentialMinorUnits,
		ShrinkageMinorUnits:              insight.ShrinkageMinorUnits,
	}
	c.JSON(http.StatusOK, genpms.SKUMarginInsightResponse{Data: &resp})
}

func (h *Handler) ListSKUCostPriceHistory(c *gin.Context, id genpms.Id, params genpms.ListSKUCostPriceHistoryParams) {
	limit := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	history, err := h.listCostPriceHistory.Execute(c.Request.Context(), id, limit)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.SKUCostPriceHistoryEntry, len(history))
	for i, hist := range history {
		resp[i] = genpms.SKUCostPriceHistoryEntry{
			Id: hist.ID, SkuId: hist.SKUID,
			CostPriceMinorUnits: hist.CostPriceMinorUnits, RecommendedPriceMinorUnits: hist.RecommendedPriceMinorUnits,
			OccurredOn: openapi_types.Date{Time: hist.OccurredOn},
		}
	}
	c.JSON(http.StatusOK, genpms.SKUCostPriceHistoryListResponse{Data: &resp})
}

func (h *Handler) ListSKUPriceHistory(c *gin.Context, id genpms.Id, params genpms.ListSKUPriceHistoryParams) {
	limit := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	history, err := h.listSKUPriceHistory.Execute(c.Request.Context(), id, limit)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genpms.SKUPriceHistoryEntry, len(history))
	for i, hist := range history {
		resp[i] = genpms.SKUPriceHistoryEntry{
			Id: hist.ID, SkuId: hist.SKUID,
			PriceMinorUnits: hist.PriceMinorUnits, SalePriceMinorUnits: hist.SalePriceMinorUnits,
			CurrencyCode: hist.CurrencyCode, ValidFrom: hist.ValidFrom, ValidUntil: hist.ValidUntil,
			Reason: genpms.SKUPriceHistoryEntryReason(hist.Reason),
		}
	}
	c.JSON(http.StatusOK, genpms.SKUPriceHistoryListResponse{Data: &resp})
}

func (h *Handler) ListMarginReport(c *gin.Context, params genpms.ListMarginReportParams) {
	in := apppms.ListMarginReportInput{}
	if params.Page != nil {
		in.Page = *params.Page
	}
	if params.PerPage != nil {
		in.PerPage = *params.PerPage
	}
	if params.BelowTargetOnly != nil {
		in.BelowTargetOnly = *params.BelowTargetOnly
	}
	result, err := h.listMarginReport.Execute(c.Request.Context(), in)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genpms.MarginReportRow, len(result.Rows))
	for i, row := range result.Rows {
		var productName *string
		if row.ProductName != "" {
			productName = &row.ProductName
		}
		rows[i] = genpms.MarginReportRow{
			SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: productName,
			CostPriceMinorUnits: row.CostPriceMinorUnits, PriceMinorUnits: row.PriceMinorUnits,
			ProfitMinorUnits: row.ProfitMinorUnits, MarginBasisPoints: row.MarginBasisPoints, MarkupBasisPoints: row.MarkupBasisPoints,
			TargetMarkupBasisPoints: row.TargetMarkupBasisPoints, RecommendedPriceMinorUnits: row.RecommendedPriceMinorUnits,
			BelowTarget: row.BelowTarget,
		}
	}
	total := result.Total
	c.JSON(http.StatusOK, genpms.MarginReportResponse{Data: &rows, Meta: &struct {
		Total *int64 `json:"total,omitempty"`
	}{Total: &total}})
}
