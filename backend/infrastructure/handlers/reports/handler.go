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

type Handler struct {
	getSalesSummaryReport        getSalesSummaryReportUC
	getProductPerformanceReport  getProductPerformanceReportUC
	getProductPerformanceMatrix  getProductPerformanceMatrixUC
	getCustomerPerformanceReport getCustomerPerformanceReportUC
	getCategoryPerformanceReport getCategoryPerformanceReportUC
	getSlowMovingProductsReport  getSlowMovingProductsReportUC
}

func NewHandler(
	getSalesSummaryReport getSalesSummaryReportUC,
	getProductPerformanceReport getProductPerformanceReportUC,
	getProductPerformanceMatrix getProductPerformanceMatrixUC,
	getCustomerPerformanceReport getCustomerPerformanceReportUC,
	getCategoryPerformanceReport getCategoryPerformanceReportUC,
	getSlowMovingProductsReport getSlowMovingProductsReportUC,
) *Handler {
	return &Handler{
		getSalesSummaryReport: getSalesSummaryReport, getProductPerformanceReport: getProductPerformanceReport,
		getProductPerformanceMatrix: getProductPerformanceMatrix, getCustomerPerformanceReport: getCustomerPerformanceReport,
		getCategoryPerformanceReport: getCategoryPerformanceReport, getSlowMovingProductsReport: getSlowMovingProductsReport,
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
	h := NewHandler(
		appreports.NewGetSalesSummaryReportUseCase(db, salesRepo, settingsRepo),
		productPerformanceUC,
		appreports.NewGetProductPerformanceMatrixUseCase(productPerformanceUC),
		appreports.NewGetCustomerPerformanceReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetCategoryPerformanceReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetSlowMovingProductsReportUseCase(db, stockLevelsUC, productPerformanceUC, skuRepo, variantRepo, productRepo),
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
	case errors.Is(err, appreports.ErrToBeforeFrom):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("reports: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
