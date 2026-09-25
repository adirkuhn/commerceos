package reports

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// RevenueByChannelReport is the "Revenue by Channel" report: whole-period
// totals per source (Channels) plus a per-source time series bucketed at
// Granularity (Points) -- the merged Reports Overview page's daily view.
type RevenueByChannelReport struct {
	CurrencyCode string
	Channels     []omsDomain.ChannelRevenue
	Granularity  omsDomain.SalesTrendGranularity
	Points       []omsDomain.ChannelRevenuePoint
}

type GetRevenueByChannelReportUseCase struct {
	db             *gorm.DB
	channelRevenue omsDomain.ChannelRevenueReportRepository
	settingsRepo   settingsDomain.Repository
}

func NewGetRevenueByChannelReportUseCase(db *gorm.DB, channelRevenue omsDomain.ChannelRevenueReportRepository, settingsRepo settingsDomain.Repository) *GetRevenueByChannelReportUseCase {
	return &GetRevenueByChannelReportUseCase{db: db, channelRevenue: channelRevenue, settingsRepo: settingsRepo}
}

func (uc *GetRevenueByChannelReportUseCase) Execute(ctx context.Context, from, to time.Time, granularity omsDomain.SalesTrendGranularity) (*RevenueByChannelReport, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("getRevenueByChannelReport: %w", ErrToBeforeFrom)
	}
	if granularity == "" {
		granularity = omsDomain.SalesTrendGranularityDay
	}
	if !granularity.Valid() {
		return nil, fmt.Errorf("getRevenueByChannelReport: %w", ErrInvalidGranularity)
	}

	var report RevenueByChannelReport
	report.Granularity = granularity
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		currency := settings.DefaultCurrency
		if currency == "" {
			currency = "USD"
		}
		report.CurrencyCode = currency

		report.Channels, err = uc.channelRevenue.RevenueByChannelBetween(ctx, from, to, currency)
		if err != nil {
			return err
		}
		report.Points, err = uc.channelRevenue.RevenueByChannelTrendBetween(ctx, from, to, currency, granularity, settings.ReportingTimezone())
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("getRevenueByChannelReport: %w", err)
	}
	return &report, nil
}
