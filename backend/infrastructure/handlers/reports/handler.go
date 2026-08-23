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

type Handler struct {
	getSalesSummaryReport       getSalesSummaryReportUC
	getProductPerformanceReport getProductPerformanceReportUC
}

func NewHandler(getSalesSummaryReport getSalesSummaryReportUC, getProductPerformanceReport getProductPerformanceReportUC) *Handler {
	return &Handler{getSalesSummaryReport: getSalesSummaryReport, getProductPerformanceReport: getProductPerformanceReport}
}

var _ genreports.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	salesRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	h := NewHandler(
		appreports.NewGetSalesSummaryReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetProductPerformanceReportUseCase(db, salesRepo, settingsRepo),
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
		rows[i] = genreports.ProductPerformanceRow{
			SkuId: row.SKUID, SkuCode: row.SKUCode, ProductName: row.ProductName,
			QuantitySold: row.QuantitySold, RevenueMinorUnits: row.RevenueMinorUnits, OrderCount: row.OrderCount,
			FirstSaleAt: row.FirstSaleAt, LastSaleAt: row.LastSaleAt,
		}
	}
	middleware.OK(c, genreports.ProductPerformanceReport{CurrencyCode: report.CurrencyCode, Rows: rows})
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
