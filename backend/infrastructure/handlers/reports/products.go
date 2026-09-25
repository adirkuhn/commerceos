package reports

import (
	"github.com/gin-gonic/gin"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func (h *Handler) GetProductPerformanceReport(c *gin.Context, params genreports.GetProductPerformanceReportParams) {
	report, err := h.getProductPerformanceReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.ProductPerformanceRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = toGenProductPerformanceRow(row)
	}
	middleware.OK(c, genreports.ProductPerformanceReport{CurrencyCode: report.CurrencyCode, Rows: rows})
}

func (h *Handler) GetProductPerformanceMatrix(c *gin.Context, params genreports.GetProductPerformanceMatrixParams) {
	matrix, err := h.getProductPerformanceMatrix.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.ProductPerformanceMatrixRow, len(matrix.Rows))
	for i, row := range matrix.Rows {
		base := toGenProductPerformanceRow(row.ProductPerformanceRow)
		rows[i] = genreports.ProductPerformanceMatrixRow{
			SkuId: base.SkuId, SkuCode: base.SkuCode, ProductName: base.ProductName,
			QuantitySold: base.QuantitySold, RevenueMinorUnits: base.RevenueMinorUnits, OrderCount: base.OrderCount,
			FirstSaleAt: base.FirstSaleAt, LastSaleAt: base.LastSaleAt,
			CostPriceMinorUnits: base.CostPriceMinorUnits, ProfitMinorUnits: base.ProfitMinorUnits, MarginBasisPoints: base.MarginBasisPoints,
			Quadrant: genreports.ProductPerformanceQuadrant(row.Quadrant),
		}
	}
	middleware.OK(c, genreports.ProductPerformanceMatrix{
		CurrencyCode: matrix.CurrencyCode, MedianRevenueMinorUnits: matrix.MedianRevenueMinorUnits,
		MedianMarginBasisPoints: matrix.MedianMarginBasisPoints, Rows: rows,
	})
}

func (h *Handler) GetCategoryPerformanceReport(c *gin.Context, params genreports.GetCategoryPerformanceReportParams) {
	report, err := h.getCategoryPerformanceReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.CategoryPerformanceRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.CategoryPerformanceRow{
			CategoryId: row.CategoryID, CategoryName: row.CategoryName,
			QuantitySold: row.QuantitySold, RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
		}
	}
	middleware.OK(c, genreports.CategoryPerformanceReport{CurrencyCode: report.CurrencyCode, Rows: rows})
}

func (h *Handler) GetSlowMovingProductsReport(c *gin.Context, params genreports.GetSlowMovingProductsReportParams) {
	report, err := h.getSlowMovingProductsReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.SlowMovingProductRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.SlowMovingProductRow{
			SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName,
			StockOnHand: row.StockOnHand, QuantitySoldInPeriod: row.QuantitySoldInPeriod,
		}
	}
	middleware.OK(c, genreports.SlowMovingProductsReport{Rows: rows})
}

func (h *Handler) GetInventorySalesReport(c *gin.Context, params genreports.GetInventorySalesReportParams) {
	report, err := h.getInventorySalesReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.InventorySalesRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.InventorySalesRow{
			SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName,
			StockOnHand: row.StockOnHand, QuantitySoldInPeriod: row.QuantitySoldInPeriod,
			TurnoverBasisPoints: row.TurnoverBasisPoints,
		}
	}
	middleware.OK(c, genreports.InventorySalesReport{Rows: rows})
}

func (h *Handler) GetProductLifecycleReport(c *gin.Context, params genreports.GetProductLifecycleReportParams) {
	report, err := h.getProductLifecycleReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.ProductLifecycleRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.ProductLifecycleRow{
			SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName,
			CurrentQuantitySold: row.CurrentQuantitySold, PreviousQuantitySold: row.PreviousQuantitySold,
			CurrentRevenueMinorUnits: row.CurrentRevenueMinorUnits, PreviousRevenueMinorUnits: row.PreviousRevenueMinorUnits,
			Stage: genreports.ProductLifecycleStage(row.Stage),
		}
	}
	middleware.OK(c, genreports.ProductLifecycleReport{CurrencyCode: report.CurrencyCode, Rows: rows})
}

func (h *Handler) GetFrequentlyBoughtTogetherReport(c *gin.Context, params genreports.GetFrequentlyBoughtTogetherReportParams) {
	minSupport := 0
	if params.MinSupport != nil {
		minSupport = int(*params.MinSupport)
	}
	report, err := h.getFrequentlyBoughtTogether.Execute(c.Request.Context(), params.From, params.To, minSupport)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.FrequentlyBoughtTogetherRow, len(report.Rows))
	for i, row := range report.Rows {
		items := make([]genreports.FrequentlyBoughtTogetherItem, len(row.Items))
		for j, item := range row.Items {
			items[j] = genreports.FrequentlyBoughtTogetherItem{
				SkuId: item.SKUID, SkuCode: item.SKUCode, ProductName: item.ProductName,
			}
		}
		rows[i] = genreports.FrequentlyBoughtTogetherRow{Items: items, Support: row.Support, SupportPercent: row.SupportPercent}
	}
	middleware.OK(c, genreports.FrequentlyBoughtTogetherReport{Rows: rows, TotalOrders: report.TotalOrders})
}

func (h *Handler) GetStockoutRiskReport(c *gin.Context, params genreports.GetStockoutRiskReportParams) {
	leadTimeDays := 0
	if params.LeadTimeDays != nil {
		leadTimeDays = int(*params.LeadTimeDays)
	}
	report, err := h.getStockoutRiskReport.Execute(c.Request.Context(), params.From, params.To, leadTimeDays)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.StockoutRiskRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.StockoutRiskRow{
			SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName,
			StockOnHand: row.StockOnHand, QuantitySoldInPeriod: row.QuantitySoldInPeriod,
			DailyVelocity: row.DailyVelocity, DaysOfStockRemaining: row.DaysOfStockRemaining,
			AtRisk: row.AtRisk, SuggestedReorderQuantity: row.SuggestedReorderQuantity,
		}
	}
	middleware.OK(c, genreports.StockoutRiskReport{Rows: rows})
}

func (h *Handler) GetProductRecommendations(c *gin.Context, params genreports.GetProductRecommendationsParams) {
	minSupport, limit := 0, 0
	if params.MinSupport != nil {
		minSupport = int(*params.MinSupport)
	}
	if params.Limit != nil {
		limit = int(*params.Limit)
	}
	report, err := h.getProductRecommendations.Execute(c.Request.Context(), params.SkuId, params.From, params.To, minSupport, limit)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.ProductRecommendationRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.ProductRecommendationRow{
			SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName, Support: row.Support,
		}
	}
	middleware.OK(c, genreports.ProductRecommendationsReport{SkuId: report.SKUID, Rows: rows, TotalOrders: report.TotalOrders})
}

func toGenProductPerformanceRow(row appreports.ProductPerformanceRow) genreports.ProductPerformanceRow {
	return genreports.ProductPerformanceRow{
		SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName,
		QuantitySold: row.QuantitySold, RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
		FirstSaleAt: row.FirstSaleAt, LastSaleAt: row.LastSaleAt,
		CostPriceMinorUnits: row.CostPriceMinorUnits, ProfitMinorUnits: row.ProfitMinorUnits, MarginBasisPoints: row.MarginBasisPoints,
	}
}
