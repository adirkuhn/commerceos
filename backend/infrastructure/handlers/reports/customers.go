package reports

import (
	"github.com/gin-gonic/gin"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

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

func (h *Handler) GetCustomerCohortsReport(c *gin.Context, params genreports.GetCustomerCohortsReportParams) {
	report, err := h.getCustomerCohortsReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	cohorts := make([]genreports.CustomerCohortRow, len(report.Cohorts))
	for i, cohort := range report.Cohorts {
		retention := make([]genreports.CohortRetentionCell, len(cohort.Retention))
		for j, cell := range cohort.Retention {
			retention[j] = genreports.CohortRetentionCell{
				MonthsSinceCohort: cell.MonthsSinceCohort, RetainedCustomers: cell.RetainedCustomers,
				RevenueMinorUnits: cell.RevenueMinorUnits, RetentionRateBasisPoints: cell.RetentionRateBasisPoints,
			}
		}
		cohorts[i] = genreports.CustomerCohortRow{CohortMonth: cohort.CohortMonth, CohortSize: cohort.CohortSize, Retention: retention}
	}
	middleware.OK(c, genreports.CustomerCohortsReport{CurrencyCode: report.CurrencyCode, Cohorts: cohorts})
}

func (h *Handler) GetCustomerRFMReport(c *gin.Context, params genreports.GetCustomerRFMReportParams) {
	report, err := h.getCustomerRFMReport.Execute(c.Request.Context(), params.From, params.To)
	if err := mapError(c, err); err != nil {
		return
	}
	rows := make([]genreports.CustomerRFMRow, len(report.Rows))
	for i, row := range report.Rows {
		rows[i] = genreports.CustomerRFMRow{
			CustomerId: row.CustomerID, CustomerName: row.CustomerName,
			OrderCount: row.OrderCount, RevenueMinorUnits: row.RevenueMinorUnits, AverageOrderValueMinorUnits: row.AverageOrderValueMinorUnits,
			FirstOrderAt: row.FirstOrderAt, LastOrderAt: row.LastOrderAt,
			RecencyScore: row.RecencyScore, FrequencyScore: row.FrequencyScore, MonetaryScore: row.MonetaryScore,
			Segment: genreports.RFMSegment(row.Segment),
		}
	}
	middleware.OK(c, genreports.CustomerRFMReport{CurrencyCode: report.CurrencyCode, Rows: rows})
}
