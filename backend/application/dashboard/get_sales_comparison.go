package dashboard

import (
	"context"
	"fmt"
	"time"

	dashboardDomain "github.com/adirkuhn/commerceos/domain/dashboard"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// salesRangeReader narrows the oms sales report ports to the two [from,
// to) queries this use case needs (ISP) -- the same revenue rules as the
// dashboard stats and the Reports module.
type salesRangeReader interface {
	SalesSummaryBetween(ctx context.Context, from, to time.Time, currencyCode string) (omsDomain.SalesSummary, error)
	RevenueByChannelBetween(ctx context.Context, from, to time.Time, currencyCode string) ([]omsDomain.ChannelRevenue, error)
}

// SalesFigures is one period's sales.
type SalesFigures struct {
	Range             dashboardDomain.DateRange
	RevenueMinorUnits int64
	OrderCount        int64
	ByChannel         []omsDomain.ChannelRevenue
}

// SalesComparison is the dashboard sales card: the period so far against
// the whole previous period (see dashboardDomain.SalesPeriod.Ranges), in
// the tenant's default currency.
type SalesComparison struct {
	Period       dashboardDomain.SalesPeriod
	CurrencyCode string
	Current      SalesFigures
	Previous     SalesFigures
	// ChangePercent is nil when the previous period sold nothing.
	ChangePercent *float64
}

// GetSalesComparisonUseCase backs the dashboard's sales card.
type GetSalesComparisonUseCase struct {
	db           *gorm.DB
	sales        salesRangeReader
	settingsRepo settingsDomain.Repository
	// Now is the injectable clock the periods are computed from.
	Now func() time.Time
}

func NewGetSalesComparisonUseCase(db *gorm.DB, sales salesRangeReader, settingsRepo settingsDomain.Repository) *GetSalesComparisonUseCase {
	return &GetSalesComparisonUseCase{db: db, sales: sales, settingsRepo: settingsRepo, Now: time.Now}
}

func (uc *GetSalesComparisonUseCase) Execute(ctx context.Context, period dashboardDomain.SalesPeriod) (*SalesComparison, error) {
	result := SalesComparison{Period: period}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		result.CurrencyCode = settings.DefaultCurrency
		if result.CurrencyCode == "" {
			result.CurrencyCode = fallbackCurrency
		}
		ranges := period.Ranges(uc.Now(), settings.ReportingLocation())
		if result.Current, err = uc.figures(ctx, ranges.Current, result.CurrencyCode); err != nil {
			return err
		}
		result.Previous, err = uc.figures(ctx, ranges.Previous, result.CurrencyCode)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getSalesComparison: %w", err)
	}
	result.ChangePercent = dashboardDomain.ChangePercent(result.Current.RevenueMinorUnits, result.Previous.RevenueMinorUnits)
	return &result, nil
}

func (uc *GetSalesComparisonUseCase) figures(ctx context.Context, r dashboardDomain.DateRange, currency string) (SalesFigures, error) {
	summary, err := uc.sales.SalesSummaryBetween(ctx, r.From, r.To, currency)
	if err != nil {
		return SalesFigures{}, err
	}
	channels, err := uc.sales.RevenueByChannelBetween(ctx, r.From, r.To, currency)
	if err != nil {
		return SalesFigures{}, err
	}
	if channels == nil {
		channels = []omsDomain.ChannelRevenue{}
	}
	return SalesFigures{Range: r, RevenueMinorUnits: summary.RevenueMinorUnits, OrderCount: summary.OrderCount, ByChannel: channels}, nil
}
