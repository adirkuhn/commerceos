package dashboard

import (
	"context"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

const (
	topProductsLimit    = 5
	stockFlowWindowDays = 7
	// fallbackCurrency matches sales_orders.currency_code's own DB default
	// -- used when a tenant has never saved settings (DefaultCurrency ""),
	// same zero-value convention as settingsDomain.TenantSettings' doc.
	fallbackCurrency = "USD"
)

// DashboardStats is the tenant-wide read-model the dashboard screen
// renders in one call. Every figure here comes straight from an existing
// domain repository's aggregate query -- see
// invDomain.DashboardStockReportRepository and
// omsDomain.DashboardSalesReportRepository's docs for exactly what each
// one counts. Revenue/channel/top-products figures are scoped to
// CurrencyCode (the tenant's default currency, or "USD" if never set) --
// a tenant that takes orders in more than one currency has orders outside
// that currency excluded from those three figures entirely, not blended
// or erroring. All three also use the same "counted as revenue" status
// filter (see domain/oms.SalesOrderStatus.CountsAsRevenue): a
// pending/cancelled/refunded/failed order was never real revenue.
type DashboardStats struct {
	TotalStockUnits            int64
	CurrencyCode               string
	RevenueThisMonthMinorUnits int64
	StockFlowLast7Days         []invDomain.DailyStockFlow
	SalesByChannelThisMonth    []omsDomain.ChannelRevenue
	TopProductsThisMonth       []omsDomain.ProductSales
}

// GetDashboardStatsUseCase composes read-only aggregate queries from the
// inventory and oms domains (plus settings, for the reporting currency)
// into one screen's worth of data -- mirrors
// application/pms.ListMarginReportUseCase's pattern of a report use case
// depending directly on another domain's repository port.
type GetDashboardStatsUseCase struct {
	db           *gorm.DB
	stockRepo    invDomain.DashboardStockReportRepository
	salesRepo    omsDomain.DashboardSalesReportRepository
	settingsRepo settingsDomain.Repository
	// Now is the injectable clock month/window boundaries are computed
	// from -- defaults to time.Now, overridden in tests for deterministic
	// boundaries.
	Now func() time.Time
}

func NewGetDashboardStatsUseCase(
	db *gorm.DB,
	stockRepo invDomain.DashboardStockReportRepository,
	salesRepo omsDomain.DashboardSalesReportRepository,
	settingsRepo settingsDomain.Repository,
) *GetDashboardStatsUseCase {
	return &GetDashboardStatsUseCase{db: db, stockRepo: stockRepo, salesRepo: salesRepo, settingsRepo: settingsRepo, Now: time.Now}
}

func (uc *GetDashboardStatsUseCase) Execute(ctx context.Context) (*DashboardStats, error) {
	var result DashboardStats
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		currency := settings.DefaultCurrency
		if currency == "" {
			currency = fallbackCurrency
		}
		result.CurrencyCode = currency

		// Calendar boundaries are the tenant's local ones -- "this month"
		// starts at local midnight on the 1st, not UTC midnight.
		loc := settings.ReportingLocation()
		now := uc.Now().In(loc)
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		flowSince := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -(stockFlowWindowDays - 1))

		result.TotalStockUnits, err = uc.stockRepo.TotalOnHand(ctx)
		if err != nil {
			return err
		}
		result.StockFlowLast7Days, err = uc.stockRepo.DailyFlow(ctx, flowSince, settings.ReportingTimezone())
		if err != nil {
			return err
		}
		result.RevenueThisMonthMinorUnits, err = uc.salesRepo.RevenueSince(ctx, monthStart, currency)
		if err != nil {
			return err
		}
		result.SalesByChannelThisMonth, err = uc.salesRepo.RevenueByChannelSince(ctx, monthStart, currency)
		if err != nil {
			return err
		}
		result.TopProductsThisMonth, err = uc.salesRepo.TopProductsSince(ctx, monthStart, currency, topProductsLimit)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getDashboardStats: %w", err)
	}
	return &result, nil
}
