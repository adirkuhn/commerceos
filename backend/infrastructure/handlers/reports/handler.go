package reports

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	appinsights "github.com/adirkuhn/commerceos/application/insights"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type getSalesSummaryReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.SalesSummaryReport, error)
}
type getProductPerformanceReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.ProductPerformanceReport, error)
}
type getProductPerformanceMatrixUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.ProductPerformanceMatrix, error)
}
type getCustomerPerformanceReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.CustomerPerformanceReport, error)
}
type getCategoryPerformanceReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.CategoryPerformanceReport, error)
}
type getSlowMovingProductsReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.SlowMovingProductsReport, error)
}
type getProfitabilityReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.ProfitabilityReport, error)
}
type getInventorySalesReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.InventorySalesReport, error)
}
type getBusinessInsightsUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appinsights.BusinessInsightsReport, error)
}
type getSalesTrendsReportUC interface {
	Execute(ctx context.Context, from, to time.Time, granularity omsDomain.SalesTrendGranularity) (*appreports.SalesTrendsReport, error)
}
type getProductLifecycleReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.ProductLifecycleReport, error)
}

type Handler struct {
	getSalesSummaryReport        getSalesSummaryReportUC
	getProductPerformanceReport  getProductPerformanceReportUC
	getProductPerformanceMatrix  getProductPerformanceMatrixUC
	getCustomerPerformanceReport getCustomerPerformanceReportUC
	getCategoryPerformanceReport getCategoryPerformanceReportUC
	getSlowMovingProductsReport  getSlowMovingProductsReportUC
	getProfitabilityReport       getProfitabilityReportUC
	getInventorySalesReport      getInventorySalesReportUC
	getBusinessInsights          getBusinessInsightsUC
	getSalesTrendsReport         getSalesTrendsReportUC
	getProductLifecycleReport    getProductLifecycleReportUC
}

func NewHandler(
	getSalesSummaryReport getSalesSummaryReportUC,
	getProductPerformanceReport getProductPerformanceReportUC,
	getProductPerformanceMatrix getProductPerformanceMatrixUC,
	getCustomerPerformanceReport getCustomerPerformanceReportUC,
	getCategoryPerformanceReport getCategoryPerformanceReportUC,
	getSlowMovingProductsReport getSlowMovingProductsReportUC,
	getProfitabilityReport getProfitabilityReportUC,
	getInventorySalesReport getInventorySalesReportUC,
	getBusinessInsights getBusinessInsightsUC,
	getSalesTrendsReport getSalesTrendsReportUC,
	getProductLifecycleReport getProductLifecycleReportUC,
) *Handler {
	return &Handler{
		getSalesSummaryReport: getSalesSummaryReport, getProductPerformanceReport: getProductPerformanceReport,
		getProductPerformanceMatrix: getProductPerformanceMatrix, getCustomerPerformanceReport: getCustomerPerformanceReport,
		getCategoryPerformanceReport: getCategoryPerformanceReport, getSlowMovingProductsReport: getSlowMovingProductsReport,
		getProfitabilityReport: getProfitabilityReport, getInventorySalesReport: getInventorySalesReport,
		getBusinessInsights: getBusinessInsights, getSalesTrendsReport: getSalesTrendsReport,
		getProductLifecycleReport: getProductLifecycleReport,
	}
}

var _ genreports.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	salesRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	productRepo := repositories.NewProductRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	productPerformanceUC := appreports.NewGetProductPerformanceReportUseCase(db, salesRepo, settingsRepo, skuRepo)
	stockLevelsUC := invapp.NewListStockLevelsUseCase(db, movementRepo, batchRepo, settingsRepo)
	salesSummaryUC := appreports.NewGetSalesSummaryReportUseCase(db, salesRepo, settingsRepo)
	productPerformanceMatrixUC := appreports.NewGetProductPerformanceMatrixUseCase(productPerformanceUC)
	slowMovingProductsUC := appreports.NewGetSlowMovingProductsReportUseCase(db, stockLevelsUC, productPerformanceUC, skuRepo, variantRepo, productRepo)
	profitabilityUC := appreports.NewGetProfitabilityReportUseCase(db, salesRepo, settingsRepo)
	inventorySalesUC := appreports.NewGetInventorySalesReportUseCase(db, stockLevelsUC, productPerformanceUC, skuRepo, variantRepo, productRepo)
	h := NewHandler(
		salesSummaryUC,
		productPerformanceUC,
		productPerformanceMatrixUC,
		appreports.NewGetCustomerPerformanceReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetCategoryPerformanceReportUseCase(db, salesRepo, settingsRepo),
		slowMovingProductsUC,
		profitabilityUC,
		inventorySalesUC,
		appinsights.NewGetBusinessInsightsUseCase(salesSummaryUC, profitabilityUC, productPerformanceMatrixUC, slowMovingProductsUC, inventorySalesUC),
		appreports.NewGetSalesTrendsReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetProductLifecycleReportUseCase(productPerformanceUC),
	)
	genreports.RegisterHandlers(rg, h)
}

func (h *Handler) GetSalesSummaryReport(c *gin.Context, params genreports.GetSalesSummaryReportParams) {
	report, err := h.getSalesSummaryReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genreports.SalesSummaryReport{
		CurrencyCode:            report.CurrencyCode,
		Current:                 toGenSalesSummary(report.Current),
		Previous:                toGenSalesSummary(report.Previous),
		RevenueChangePercent:    report.RevenueChangePercent,
		OrderCountChangePercent: report.OrderCountChangePercent,
		UnitsSoldChangePercent:  report.UnitsSoldChangePercent,
		AovChangePercent:        report.AOVChangePercent,
	})
}

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

func (h *Handler) GetCustomerPerformanceReport(c *gin.Context, params genreports.GetCustomerPerformanceReportParams) {
	report, err := h.getCustomerPerformanceReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.CustomerPerformanceRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.CustomerPerformanceRow{
			CustomerId: row.CustomerID, CustomerName: row.CustomerName, OrderCount: row.OrderCount,
			RevenueMinorUnits: row.RevenueMinorUnits, AverageOrderValueMinorUnits: row.AverageOrderValueMinorUnits,
			FirstOrderAt: row.FirstOrderAt, LastOrderAt: row.LastOrderAt,
		}
	}
	middleware.OK(c, genreports.CustomerPerformanceReport{CurrencyCode: report.CurrencyCode, Rows: rows})
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

func (h *Handler) GetProfitabilityReport(c *gin.Context, params genreports.GetProfitabilityReportParams) {
	report, err := h.getProfitabilityReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	s := report.Summary
	middleware.OK(c, genreports.ProfitabilityReport{
		CurrencyCode: report.CurrencyCode,
		Summary: genreports.ProfitabilitySummary{
			RevenueMinorUnits: s.RevenueMinorUnits, CostKnownRevenueMinorUnits: s.CostKnownRevenueMinorUnits,
			CostMinorUnits: s.CostMinorUnits, ProfitMinorUnits: s.ProfitMinorUnits, MarginBasisPoints: s.MarginBasisPoints,
		},
	})
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

func (h *Handler) GetBusinessInsights(c *gin.Context, params genreports.GetBusinessInsightsParams) {
	report, err := h.getBusinessInsights.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	insights := make([]genreports.Insight, len(report.Insights))
	for i, insight := range report.Insights {
		insights[i] = genreports.Insight{
			Code: insight.Code, Message: insight.Message, Severity: genreports.InsightSeverity(insight.Severity),
		}
	}
	middleware.OK(c, genreports.BusinessInsightsReport{Insights: insights})
}

func (h *Handler) GetSalesTrendsReport(c *gin.Context, params genreports.GetSalesTrendsReportParams) {
	granularity := omsDomain.SalesTrendGranularityDay
	if params.Granularity != nil {
		granularity = omsDomain.SalesTrendGranularity(*params.Granularity)
	}
	report, err := h.getSalesTrendsReport.Execute(c.Request.Context(), params.From, params.To, granularity)
	if err := mapError(c, err); err != nil {
		return
	}
	points := make([]genreports.SalesTrendPoint, len(report.Points))
	for i, p := range report.Points {
		points[i] = genreports.SalesTrendPoint{
			PeriodStart: p.PeriodStart, RevenueMinorUnits: p.RevenueMinorUnits, OrderCount: p.OrderCount, UnitsSold: p.UnitsSold,
		}
	}
	byWeekday := make([]genreports.WeekdaySales, len(report.ByWeekday))
	for i, w := range report.ByWeekday {
		byWeekday[i] = genreports.WeekdaySales{
			Weekday: int32(w.Weekday), RevenueMinorUnits: w.RevenueMinorUnits, OrderCount: w.OrderCount,
		}
	}
	middleware.OK(c, genreports.SalesTrendsReport{
		CurrencyCode: report.CurrencyCode, Granularity: genreports.SalesTrendGranularity(report.Granularity),
		Points: points, ByWeekday: byWeekday,
	})
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

func toGenProductPerformanceRow(row appreports.ProductPerformanceRow) genreports.ProductPerformanceRow {
	return genreports.ProductPerformanceRow{
		SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName,
		QuantitySold: row.QuantitySold, RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
		FirstSaleAt: row.FirstSaleAt, LastSaleAt: row.LastSaleAt,
		CostPriceMinorUnits: row.CostPriceMinorUnits, ProfitMinorUnits: row.ProfitMinorUnits, MarginBasisPoints: row.MarginBasisPoints,
	}
}

func toGenSalesSummary(s omsDomain.SalesSummary) genreports.SalesSummary {
	return genreports.SalesSummary{
		RevenueMinorUnits: s.RevenueMinorUnits, OrderCount: s.OrderCount,
		UnitsSold: s.UnitsSold, AverageOrderValueMinorUnits: s.AverageOrderValueMinorUnits,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, appreports.ErrToBeforeFrom), errors.Is(err, appreports.ErrInvalidGranularity):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("reports: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
