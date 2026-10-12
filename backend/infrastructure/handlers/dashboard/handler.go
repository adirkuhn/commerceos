package dashboard

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	gendashboard "github.com/adirkuhn/commerceos/api/dashboard"
	appdashboard "github.com/adirkuhn/commerceos/application/dashboard"
	dashboardDomain "github.com/adirkuhn/commerceos/domain/dashboard"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type getDashboardStatsUC interface {
	Execute(ctx context.Context) (*appdashboard.DashboardStats, error)
}

type getTodayQueueUC interface {
	Execute(ctx context.Context) (*appdashboard.TodayQueue, error)
}

type getSalesComparisonUC interface {
	Execute(ctx context.Context, period dashboardDomain.SalesPeriod) (*appdashboard.SalesComparison, error)
}

type Handler struct {
	getDashboardStats  getDashboardStatsUC
	getTodayQueue      getTodayQueueUC
	getSalesComparison getSalesComparisonUC
}

func NewHandler(getDashboardStats getDashboardStatsUC, getTodayQueue getTodayQueueUC, getSalesComparison getSalesComparisonUC) *Handler {
	return &Handler{getDashboardStats: getDashboardStats, getTodayQueue: getTodayQueue, getSalesComparison: getSalesComparison}
}

var _ gendashboard.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	stockRepo := repositories.NewStockMovementRepo(db)
	salesRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	h := NewHandler(
		appdashboard.NewGetDashboardStatsUseCase(db, stockRepo, salesRepo, settingsRepo),
		appdashboard.NewGetTodayQueueUseCase(db, salesRepo, batchRepo, settingsRepo),
		appdashboard.NewGetSalesComparisonUseCase(db, salesRepo, settingsRepo),
	)
	gendashboard.RegisterHandlers(rg, h)
}

func (h *Handler) GetDashboardStats(c *gin.Context) {
	stats, err := h.getDashboardStats.Execute(c.Request.Context())
	if err != nil {
		log.Printf("dashboard: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	resp := gendashboard.DashboardStatsResponse{Data: toGenDashboardStats(*stats)}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetTodayQueue(c *gin.Context) {
	q, err := h.getTodayQueue.Execute(c.Request.Context())
	if err != nil {
		log.Printf("dashboard: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	c.JSON(http.StatusOK, gendashboard.TodayQueueResponse{Data: &gendashboard.TodayQueue{
		ToPick: q.ToPick, ToPack: q.ToPack, StockIssues: q.StockIssues, UnmatchedSkus: q.UnmatchedSKUs,
		ExpiringBatches: q.ExpiringBatches, ExpiredBatches: q.ExpiredBatches, ShipToday: q.ShipToday,
	}})
}

func (h *Handler) GetSalesComparison(c *gin.Context, params gendashboard.GetSalesComparisonParams) {
	period, err := dashboardDomain.ParseSalesPeriod(string(params.Period))
	if err != nil {
		middleware.FailErr(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err)
		return
	}
	cmp, err := h.getSalesComparison.Execute(c.Request.Context(), period)
	if err != nil {
		log.Printf("dashboard: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	c.JSON(http.StatusOK, gendashboard.SalesComparisonResponse{Data: gendashboard.SalesComparison{
		Period: gendashboard.SalesPeriod(cmp.Period), CurrencyCode: cmp.CurrencyCode, ChangePercent: cmp.ChangePercent,
		Current: toGenSalesFigures(cmp.Current), Previous: toGenSalesFigures(cmp.Previous),
	}})
}

func toGenSalesFigures(f appdashboard.SalesFigures) gendashboard.SalesFigures {
	return gendashboard.SalesFigures{
		From: f.Range.From, To: f.Range.To, RevenueMinorUnits: f.RevenueMinorUnits, OrderCount: f.OrderCount,
		ByChannel: toGenChannelRevenue(f.ByChannel),
	}
}

func toGenChannelRevenue(in []omsDomain.ChannelRevenue) []gendashboard.ChannelRevenue {
	out := make([]gendashboard.ChannelRevenue, len(in))
	for i, ch := range in {
		out[i] = gendashboard.ChannelRevenue{Source: string(ch.Source), TotalMinorUnits: ch.TotalMinorUnits}
	}
	return out
}

func toGenDashboardStats(s appdashboard.DashboardStats) *gendashboard.DashboardStats {
	flow := make([]gendashboard.DailyStockFlow, len(s.StockFlowLast7Days))
	for i, f := range s.StockFlowLast7Days {
		flow[i] = gendashboard.DailyStockFlow{Date: openapi_types.Date{Time: f.Date}, In: f.In, Out: f.Out}
	}
	channels := toGenChannelRevenue(s.SalesByChannelThisMonth)
	products := make([]gendashboard.TopProduct, len(s.TopProductsThisMonth))
	for i, p := range s.TopProductsThisMonth {
		products[i] = gendashboard.TopProduct{
			SkuId: p.SKUID, SkuCode: p.SKUCode, ProductName: p.ProductName,
			QuantitySold: p.QuantitySold, RevenueMinorUnits: p.RevenueMinorUnits,
		}
	}
	return &gendashboard.DashboardStats{
		TotalStockUnits: s.TotalStockUnits, CurrencyCode: s.CurrencyCode,
		RevenueThisMonthMinorUnits: s.RevenueThisMonthMinorUnits,
		StockFlowLast7Days:         flow,
		SalesByChannelThisMonth:    channels,
		TopProductsThisMonth:       products,
	}
}
