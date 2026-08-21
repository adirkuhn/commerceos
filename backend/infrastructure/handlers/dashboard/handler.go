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
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

type getDashboardStatsUC interface {
	Execute(ctx context.Context) (*appdashboard.DashboardStats, error)
}

type Handler struct {
	getDashboardStats getDashboardStatsUC
}

func NewHandler(getDashboardStats getDashboardStatsUC) *Handler {
	return &Handler{getDashboardStats: getDashboardStats}
}

var _ gendashboard.ServerInterface = (*Handler)(nil)

func Register(rg gin.IRouter, db *gorm.DB) {
	stockRepo := repositories.NewStockMovementRepo(db)
	salesRepo := repositories.NewSalesOrderRepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	h := NewHandler(appdashboard.NewGetDashboardStatsUseCase(db, stockRepo, salesRepo, settingsRepo))
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

func toGenDashboardStats(s appdashboard.DashboardStats) *gendashboard.DashboardStats {
	flow := make([]gendashboard.DailyStockFlow, len(s.StockFlowLast7Days))
	for i, f := range s.StockFlowLast7Days {
		flow[i] = gendashboard.DailyStockFlow{Date: openapi_types.Date{Time: f.Date}, In: f.In, Out: f.Out}
	}
	channels := make([]gendashboard.ChannelRevenue, len(s.SalesByChannelThisMonth))
	for i, ch := range s.SalesByChannelThisMonth {
		channels[i] = gendashboard.ChannelRevenue{Source: string(ch.Source), TotalMinorUnits: ch.TotalMinorUnits}
	}
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
