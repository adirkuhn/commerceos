package reports_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	appreports "github.com/adirkuhn/commerceos/application/reports"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	promotionsDomain "github.com/adirkuhn/commerceos/domain/promotions"
)

type stubPromotionLookupForPerformance struct {
	out *promotionsDomain.Promotion
	err error
}

func (s *stubPromotionLookupForPerformance) FindByID(_ context.Context, _ string) (*promotionsDomain.Promotion, error) {
	return s.out, s.err
}

type stubPromotionLineListerForPerformance struct {
	out []promotionsDomain.PromotionSKU
	err error
}

func (s *stubPromotionLineListerForPerformance) ListByPromotion(_ context.Context, _ string) ([]promotionsDomain.PromotionSKU, error) {
	return s.out, s.err
}

type stubPromotionPerformanceRepo struct {
	sequence []omsDomain.SKUSetSalesSummary
	call     int
	err      error
	gotCalls []skuSetSalesCall
}

type skuSetSalesCall struct {
	SKUIDs   []string
	From, To time.Time
}

func (s *stubPromotionPerformanceRepo) SKUSetSalesBetween(_ context.Context, skuIDs []string, from, to time.Time, _ string) (omsDomain.SKUSetSalesSummary, error) {
	s.gotCalls = append(s.gotCalls, skuSetSalesCall{SKUIDs: skuIDs, From: from, To: to})
	if s.err != nil {
		return omsDomain.SKUSetSalesSummary{}, s.err
	}
	out := s.sequence[s.call]
	s.call++
	return out, nil
}

func TestGetPromotionPerformanceReport_comparesDuringAgainstEquivalentBeforePeriod(t *testing.T) {
	startsAt := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	endsAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC) // 10-day promotion, already ended
	promo := &stubPromotionLookupForPerformance{out: &promotionsDomain.Promotion{
		ID: "promo-1", Name: "Summer Acai", StartsAt: startsAt, EndsAt: &endsAt,
	}}
	lines := &stubPromotionLineListerForPerformance{out: []promotionsDomain.PromotionSKU{
		{SKUID: "sku-1"}, {SKUID: "sku-2"},
	}}
	perf := &stubPromotionPerformanceRepo{sequence: []omsDomain.SKUSetSalesSummary{
		{RevenueMinorUnits: 2000, UnitsSold: 20, OrderCount: 10}, // during (queried first)
		{RevenueMinorUnits: 1000, UnitsSold: 10, OrderCount: 5},  // before
	}}
	uc := appreports.NewGetPromotionPerformanceReportUseCase(newMockGormDB(t), promo, lines, perf, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.Equal(t, "promo-1", result.PromotionID)
	require.Equal(t, "Summer Acai", result.PromotionName)
	require.True(t, result.During.From.Equal(startsAt))
	require.True(t, result.During.To.Equal(endsAt))
	require.Equal(t, int64(2000), result.During.RevenueMinorUnits)
	require.True(t, result.Before.From.Equal(startsAt.AddDate(0, 0, -10)), "before window must be the same 10-day length, immediately preceding")
	require.True(t, result.Before.To.Equal(startsAt))
	require.Equal(t, int64(1000), result.Before.RevenueMinorUnits)
	require.NotNil(t, result.RevenueChangePercent)
	require.InDelta(t, 100.0, *result.RevenueChangePercent, 0.01) // 1000 -> 2000 = +100%
	require.NotNil(t, result.UnitsSoldChangePercent)

	require.Len(t, perf.gotCalls, 2)
	require.ElementsMatch(t, []string{"sku-1", "sku-2"}, perf.gotCalls[0].SKUIDs)
}

func TestGetPromotionPerformanceReport_ongoingPromotionUsesNowAsDuringEnd(t *testing.T) {
	startsAt := time.Now().Add(-2 * time.Hour)
	promo := &stubPromotionLookupForPerformance{out: &promotionsDomain.Promotion{
		ID: "promo-1", Name: "Ongoing", StartsAt: startsAt, EndsAt: nil,
	}}
	lines := &stubPromotionLineListerForPerformance{out: []promotionsDomain.PromotionSKU{{SKUID: "sku-1"}}}
	perf := &stubPromotionPerformanceRepo{sequence: []omsDomain.SKUSetSalesSummary{{}, {}}}
	uc := appreports.NewGetPromotionPerformanceReportUseCase(newMockGormDB(t), promo, lines, perf, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.True(t, result.During.To.After(startsAt))
	require.WithinDuration(t, time.Now(), result.During.To, 5*time.Second)
}

func TestGetPromotionPerformanceReport_nilBeforeBaselineGivesNilChangePercent(t *testing.T) {
	startsAt := time.Now().Add(-time.Hour)
	promo := &stubPromotionLookupForPerformance{out: &promotionsDomain.Promotion{ID: "promo-1", Name: "New", StartsAt: startsAt}}
	lines := &stubPromotionLineListerForPerformance{out: []promotionsDomain.PromotionSKU{{SKUID: "sku-1"}}}
	perf := &stubPromotionPerformanceRepo{sequence: []omsDomain.SKUSetSalesSummary{
		{RevenueMinorUnits: 500, UnitsSold: 5},
		{RevenueMinorUnits: 0, UnitsSold: 0},
	}}
	uc := appreports.NewGetPromotionPerformanceReportUseCase(newMockGormDB(t), promo, lines, perf, &stubSettingsRepo{})

	result, err := uc.Execute(context.Background(), "promo-1")

	require.NoError(t, err)
	require.Nil(t, result.RevenueChangePercent)
	require.Nil(t, result.UnitsSoldChangePercent)
}

func TestGetPromotionPerformanceReport_propagatesPromotionNotFound(t *testing.T) {
	promo := &stubPromotionLookupForPerformance{err: promotionsDomain.ErrPromotionNotFound}
	uc := appreports.NewGetPromotionPerformanceReportUseCase(newMockGormDB(t), promo, &stubPromotionLineListerForPerformance{}, &stubPromotionPerformanceRepo{}, &stubSettingsRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	require.Error(t, err)
}
