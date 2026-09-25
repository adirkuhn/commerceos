package reports

import (
	"github.com/gin-gonic/gin"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

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

func (h *Handler) GetRevenueByChannelReport(c *gin.Context, params genreports.GetRevenueByChannelReportParams) {
	granularity := omsDomain.SalesTrendGranularityDay
	if params.Granularity != nil {
		granularity = omsDomain.SalesTrendGranularity(*params.Granularity)
	}
	report, err := h.getRevenueByChannelReport.Execute(c.Request.Context(), params.From, params.To, granularity)
	if err := mapError(c, err); err != nil {
		return
	}
	channels := make([]genreports.ChannelRevenue, len(report.Channels))
	for i, ch := range report.Channels {
		channels[i] = genreports.ChannelRevenue{Source: string(ch.Source), TotalMinorUnits: ch.TotalMinorUnits}
	}
	points := make([]genreports.ChannelRevenuePoint, len(report.Points))
	for i, p := range report.Points {
		points[i] = genreports.ChannelRevenuePoint{PeriodStart: p.PeriodStart, Source: string(p.Source), RevenueMinorUnits: p.RevenueMinorUnits}
	}
	middleware.OK(c, genreports.RevenueByChannelReport{
		CurrencyCode: report.CurrencyCode, Channels: channels, Granularity: genreports.SalesTrendGranularity(report.Granularity), Points: points,
	})
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

func (h *Handler) GetPromotionPerformanceReport(c *gin.Context, promotionId string) {
	report, err := h.getPromotionPerformanceReport.Execute(c.Request.Context(), promotionId)
	if err := mapError(c, err); err != nil {
		return
	}
	middleware.OK(c, genreports.PromotionPerformanceReport{
		PromotionId: report.PromotionID, PromotionName: report.PromotionName, CurrencyCode: report.CurrencyCode,
		Before: toGenPromotionPerformancePeriod(report.Before), During: toGenPromotionPerformancePeriod(report.During),
		RevenueChangePercent: report.RevenueChangePercent, UnitsSoldChangePercent: report.UnitsSoldChangePercent,
	})
}

func toGenPromotionPerformancePeriod(p appreports.PromotionPerformancePeriod) genreports.PromotionPerformancePeriod {
	return genreports.PromotionPerformancePeriod{
		From: p.From, To: p.To, RevenueMinorUnits: p.RevenueMinorUnits, UnitsSold: p.UnitsSold, OrderCount: p.OrderCount,
	}
}

func toGenSalesSummary(s omsDomain.SalesSummary) genreports.SalesSummary {
	return genreports.SalesSummary{
		RevenueMinorUnits: s.RevenueMinorUnits, OrderCount: s.OrderCount,
		UnitsSold: s.UnitsSold, AverageOrderValueMinorUnits: s.AverageOrderValueMinorUnits,
	}
}
