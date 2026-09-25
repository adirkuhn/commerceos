package reports

import (
	"github.com/gin-gonic/gin"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

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

func (h *Handler) GetProfitabilityTrendReport(c *gin.Context, params genreports.GetProfitabilityTrendReportParams) {
	granularity := omsDomain.SalesTrendGranularityDay
	if params.Granularity != nil {
		granularity = omsDomain.SalesTrendGranularity(*params.Granularity)
	}
	report, err := h.getProfitabilityTrendReport.Execute(c.Request.Context(), params.From, params.To, granularity)
	if err := mapError(c, err); err != nil {
		return
	}
	points := make([]genreports.ProfitabilityPoint, len(report.Points))
	for i, p := range report.Points {
		points[i] = genreports.ProfitabilityPoint{
			PeriodStart: p.PeriodStart, RevenueMinorUnits: p.RevenueMinorUnits, CostKnownRevenueMinorUnits: p.CostKnownRevenueMinorUnits,
			CostMinorUnits: p.CostMinorUnits, ProfitMinorUnits: p.ProfitMinorUnits, MarginBasisPoints: p.MarginBasisPoints,
		}
	}
	middleware.OK(c, genreports.ProfitabilityTrendReport{
		CurrencyCode: report.CurrencyCode, Granularity: genreports.SalesTrendGranularity(report.Granularity), Points: points,
	})
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
