package reports

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PromotionPerformancePeriod is revenue/units/orders for the promotion's
// covered SKUs within one window.
type PromotionPerformancePeriod struct {
	From, To          time.Time
	RevenueMinorUnits int64
	UnitsSold         int64
	OrderCount        int64
}

// PromotionPerformanceReport is the Phase 0 "Promotion Analytics" report:
// a single promotion's During window (its active period, capped at "now"
// if still ongoing) compared against the immediately preceding Before
// window of equal length -- same "previous period of equal length"
// convention as SalesSummaryReport/ProductLifecycleReport.
type PromotionPerformanceReport struct {
	PromotionID            string
	PromotionName          string
	CurrencyCode           string
	Before                 PromotionPerformancePeriod
	During                 PromotionPerformancePeriod
	RevenueChangePercent   *float64
	UnitsSoldChangePercent *float64
}

// promotionLookup narrows promotions.PromotionRepository to the one method
// this report needs (ISP).
type promotionLookup interface {
	FindByID(ctx context.Context, id string) (*promotionsDomain.Promotion, error)
}

// promotionLineLister narrows promotions.PromotionSKURepository to the one
// method this report needs (ISP).
type promotionLineLister interface {
	ListByPromotion(ctx context.Context, promotionID string) ([]promotionsDomain.PromotionSKU, error)
}

type GetPromotionPerformanceReportUseCase struct {
	db           *gorm.DB
	promoRepo    promotionLookup
	lineRepo     promotionLineLister
	perfRepo     omsDomain.PromotionPerformanceRepository
	settingsRepo settingsDomain.Repository
}

func NewGetPromotionPerformanceReportUseCase(db *gorm.DB, promoRepo promotionLookup, lineRepo promotionLineLister, perfRepo omsDomain.PromotionPerformanceRepository, settingsRepo settingsDomain.Repository) *GetPromotionPerformanceReportUseCase {
	return &GetPromotionPerformanceReportUseCase{db: db, promoRepo: promoRepo, lineRepo: lineRepo, perfRepo: perfRepo, settingsRepo: settingsRepo}
}

func (uc *GetPromotionPerformanceReportUseCase) Execute(ctx context.Context, promotionID string) (*PromotionPerformanceReport, error) {
	var report PromotionPerformanceReport
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		promo, err := uc.promoRepo.FindByID(ctx, promotionID)
		if err != nil {
			return err
		}
		lines, err := uc.lineRepo.ListByPromotion(ctx, promotionID)
		if err != nil {
			return err
		}
		skuIDs := make([]string, len(lines))
		for i, l := range lines {
			skuIDs[i] = l.SKUID
		}

		settings, err := uc.settingsRepo.Get(ctx)
		if err != nil {
			return err
		}
		currency := settings.DefaultCurrency
		if currency == "" {
			currency = "USD"
		}
		report.PromotionID = promo.ID
		report.PromotionName = promo.Name
		report.CurrencyCode = currency

		now := time.Now()
		duringTo := now
		if promo.EndsAt != nil && promo.EndsAt.Before(now) {
			duringTo = *promo.EndsAt
		}
		duringFrom := promo.StartsAt
		duration := duringTo.Sub(duringFrom)
		if duration < 0 {
			duration = 0
		}
		beforeFrom := duringFrom.Add(-duration)
		beforeTo := duringFrom

		duringSummary, err := uc.perfRepo.SKUSetSalesBetween(ctx, skuIDs, duringFrom, duringTo, currency)
		if err != nil {
			return err
		}
		report.During = PromotionPerformancePeriod{
			From: duringFrom, To: duringTo, RevenueMinorUnits: duringSummary.RevenueMinorUnits,
			UnitsSold: duringSummary.UnitsSold, OrderCount: duringSummary.OrderCount,
		}

		beforeSummary, err := uc.perfRepo.SKUSetSalesBetween(ctx, skuIDs, beforeFrom, beforeTo, currency)
		if err != nil {
			return err
		}
		report.Before = PromotionPerformancePeriod{
			From: beforeFrom, To: beforeTo, RevenueMinorUnits: beforeSummary.RevenueMinorUnits,
			UnitsSold: beforeSummary.UnitsSold, OrderCount: beforeSummary.OrderCount,
		}

		report.RevenueChangePercent = percentChange(beforeSummary.RevenueMinorUnits, duringSummary.RevenueMinorUnits)
		report.UnitsSoldChangePercent = percentChange(beforeSummary.UnitsSold, duringSummary.UnitsSold)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPromotionPerformanceReport: %w", err)
	}
	return &report, nil
}
