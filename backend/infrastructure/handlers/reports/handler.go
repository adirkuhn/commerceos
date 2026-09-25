package reports

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genreports "github.com/adirkuhn/commerceos/api/reports"
	appinsights "github.com/adirkuhn/commerceos/application/insights"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
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
type getCustomerCohortsReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.CustomerCohortsReport, error)
}
type getPromotionPerformanceReportUC interface {
	Execute(ctx context.Context, promotionID string) (*appreports.PromotionPerformanceReport, error)
}
type getCustomerRFMReportUC interface {
	Execute(ctx context.Context, from, to time.Time) (*appreports.CustomerRFMReport, error)
}
type getFrequentlyBoughtTogetherReportUC interface {
	Execute(ctx context.Context, from, to time.Time, minSupport int) (*appreports.FrequentlyBoughtTogetherReport, error)
}
type getStockoutRiskReportUC interface {
	Execute(ctx context.Context, from, to time.Time, leadTimeDays int) (*appreports.StockoutRiskReport, error)
}
type getProductRecommendationsUC interface {
	Execute(ctx context.Context, skuID string, from, to time.Time, minSupport, limit int) (*appreports.ProductRecommendationsReport, error)
}
type getProfitabilityTrendReportUC interface {
	Execute(ctx context.Context, from, to time.Time, granularity omsDomain.SalesTrendGranularity) (*appreports.ProfitabilityTrendReport, error)
}
type getRevenueByChannelReportUC interface {
	Execute(ctx context.Context, from, to time.Time, granularity omsDomain.SalesTrendGranularity) (*appreports.RevenueByChannelReport, error)
}

type Handler struct {
	getSalesSummaryReport         getSalesSummaryReportUC
	getProductPerformanceReport   getProductPerformanceReportUC
	getProductPerformanceMatrix   getProductPerformanceMatrixUC
	getCustomerPerformanceReport  getCustomerPerformanceReportUC
	getCategoryPerformanceReport  getCategoryPerformanceReportUC
	getSlowMovingProductsReport   getSlowMovingProductsReportUC
	getProfitabilityReport        getProfitabilityReportUC
	getInventorySalesReport       getInventorySalesReportUC
	getBusinessInsights           getBusinessInsightsUC
	getSalesTrendsReport          getSalesTrendsReportUC
	getProductLifecycleReport     getProductLifecycleReportUC
	getCustomerCohortsReport      getCustomerCohortsReportUC
	getPromotionPerformanceReport getPromotionPerformanceReportUC
	getCustomerRFMReport          getCustomerRFMReportUC
	getFrequentlyBoughtTogether   getFrequentlyBoughtTogetherReportUC
	getStockoutRiskReport         getStockoutRiskReportUC
	getProductRecommendations     getProductRecommendationsUC
	getProfitabilityTrendReport   getProfitabilityTrendReportUC
	getRevenueByChannelReport     getRevenueByChannelReportUC
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
	getCustomerCohortsReport getCustomerCohortsReportUC,
	getPromotionPerformanceReport getPromotionPerformanceReportUC,
	getCustomerRFMReport getCustomerRFMReportUC,
	getFrequentlyBoughtTogether getFrequentlyBoughtTogetherReportUC,
	getStockoutRiskReport getStockoutRiskReportUC,
	getProductRecommendations getProductRecommendationsUC,
	getProfitabilityTrendReport getProfitabilityTrendReportUC,
	getRevenueByChannelReport getRevenueByChannelReportUC,
) *Handler {
	return &Handler{
		getSalesSummaryReport: getSalesSummaryReport, getProductPerformanceReport: getProductPerformanceReport,
		getProductPerformanceMatrix: getProductPerformanceMatrix, getCustomerPerformanceReport: getCustomerPerformanceReport,
		getCategoryPerformanceReport: getCategoryPerformanceReport, getSlowMovingProductsReport: getSlowMovingProductsReport,
		getProfitabilityReport: getProfitabilityReport, getInventorySalesReport: getInventorySalesReport,
		getBusinessInsights: getBusinessInsights, getSalesTrendsReport: getSalesTrendsReport,
		getProductLifecycleReport: getProductLifecycleReport, getCustomerCohortsReport: getCustomerCohortsReport,
		getPromotionPerformanceReport: getPromotionPerformanceReport, getCustomerRFMReport: getCustomerRFMReport,
		getFrequentlyBoughtTogether: getFrequentlyBoughtTogether, getStockoutRiskReport: getStockoutRiskReport,
		getProductRecommendations:   getProductRecommendations,
		getProfitabilityTrendReport: getProfitabilityTrendReport, getRevenueByChannelReport: getRevenueByChannelReport,
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
	promotionRepo := repositories.NewPromotionRepo(db)
	promotionSKURepo := repositories.NewPromotionSKURepo(db)
	customerPerformanceUC := appreports.NewGetCustomerPerformanceReportUseCase(db, salesRepo, settingsRepo)
	h := NewHandler(
		salesSummaryUC,
		productPerformanceUC,
		productPerformanceMatrixUC,
		customerPerformanceUC,
		appreports.NewGetCategoryPerformanceReportUseCase(db, salesRepo, settingsRepo),
		slowMovingProductsUC,
		profitabilityUC,
		inventorySalesUC,
		appinsights.NewGetBusinessInsightsUseCase(salesSummaryUC, profitabilityUC, productPerformanceMatrixUC, slowMovingProductsUC, inventorySalesUC),
		appreports.NewGetSalesTrendsReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetProductLifecycleReportUseCase(productPerformanceUC),
		appreports.NewGetCustomerCohortsReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetPromotionPerformanceReportUseCase(db, promotionRepo, promotionSKURepo, salesRepo, settingsRepo),
		appreports.NewGetCustomerRFMReportUseCase(customerPerformanceUC),
		appreports.NewGetFrequentlyBoughtTogetherReportUseCase(db, salesRepo, settingsRepo, skuRepo, variantRepo, productRepo),
		appreports.NewGetStockoutRiskReportUseCase(db, stockLevelsUC, productPerformanceUC, skuRepo, variantRepo, productRepo),
		appreports.NewGetProductRecommendationsUseCase(db, salesRepo, settingsRepo, skuRepo, variantRepo, productRepo),
		appreports.NewGetProfitabilityTrendReportUseCase(db, salesRepo, settingsRepo),
		appreports.NewGetRevenueByChannelReportUseCase(db, salesRepo, settingsRepo),
	)
	genreports.RegisterHandlers(rg, h)
}
