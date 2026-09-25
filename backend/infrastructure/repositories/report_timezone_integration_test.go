//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// setOccurredAt moves a seeded order to a precise instant.
func (f revenueFixture) setOccurredAt(t *testing.T, extID string, at time.Time) {
	t.Helper()
	require.NoError(t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		return f.db.WithContext(ctx).Exec(`UPDATE sales_orders SET occurred_at = ? WHERE external_order_id = ?`, at, extID).Error
	}))
}

// Reports bucket days and weekdays in the tenant's timezone: an order at
// 23:30 UTC on Friday 2026-03-06 was placed at 00:30 on Saturday in
// Amsterdam (CET, UTC+1), so every day/weekday view must put it on Saturday.
func TestSalesOrderRepo_Integration_TrendBucketsUseTheTenantTimezone(t *testing.T) {
	const tz = "Europe/Amsterdam"
	ams, err := time.LoadLocation(tz)
	require.NoError(t, err)
	f := newRevenueFixture(t)
	f.seedOrder(t, "tz-late", omsDomain.SalesOrderStatusCompleted, 1, 1000)
	f.setOccurredAt(t, "tz-late", time.Date(2026, 3, 6, 23, 30, 0, 0, time.UTC))

	from, to := time.Date(2026, 3, 1, 0, 0, 0, 0, ams), time.Date(2026, 3, 31, 0, 0, 0, 0, ams)
	saturday := time.Date(2026, 3, 7, 0, 0, 0, 0, ams)
	require.NoError(t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		wd, err := f.repo.SalesByWeekdayBetween(ctx, from, to, "EUR", tz)
		require.NoError(t, err)
		require.Len(t, wd, 1)
		assert.Equal(t, time.Saturday, wd[0].Weekday)

		trend, err := f.repo.SalesTrendBetween(ctx, from, to, "EUR", omsDomain.SalesTrendGranularityDay, tz)
		require.NoError(t, err)
		require.Len(t, trend, 1)
		assert.True(t, saturday.Equal(trend[0].PeriodStart), "sales trend bucket %s", trend[0].PeriodStart)
		assert.Equal(t, int64(1), trend[0].UnitsSold, "units bucket must match the revenue bucket")

		ch, err := f.repo.RevenueByChannelTrendBetween(ctx, from, to, "EUR", omsDomain.SalesTrendGranularityDay, tz)
		require.NoError(t, err)
		require.Len(t, ch, 1)
		assert.True(t, saturday.Equal(ch[0].PeriodStart), "channel trend bucket %s", ch[0].PeriodStart)

		prof, err := f.repo.ProfitabilityTrendBetween(ctx, from, to, "EUR", omsDomain.SalesTrendGranularityDay, tz)
		require.NoError(t, err)
		require.Len(t, prof, 1)
		assert.True(t, saturday.Equal(prof[0].PeriodStart), "profitability trend bucket %s", prof[0].PeriodStart)
		return nil
	}))
}

// Cohort months are local months too: 22:30 UTC on 2026-03-31 is already
// 00:30 on April 1 in Amsterdam (CEST, UTC+2).
func TestSalesOrderRepo_Integration_CohortMonthsUseTheTenantTimezone(t *testing.T) {
	const tz = "Europe/Amsterdam"
	ams, err := time.LoadLocation(tz)
	require.NoError(t, err)
	f := newRevenueFixture(t)
	f.seedOrder(t, "tz-cohort", omsDomain.SalesOrderStatusCompleted, 1, 1000)
	f.setOccurredAt(t, "tz-cohort", time.Date(2026, 3, 31, 22, 30, 0, 0, time.UTC))

	april := time.Date(2026, 4, 1, 0, 0, 0, 0, ams)
	require.NoError(t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		sizes, err := f.repo.CohortSizesBetween(ctx, april, april.AddDate(0, 1, 0), "EUR", tz)
		require.NoError(t, err)
		require.Len(t, sizes, 1)
		assert.True(t, april.Equal(sizes[0].CohortMonth), "cohort month %s", sizes[0].CohortMonth)

		ret, err := f.repo.CohortRetentionBetween(ctx, april, april.AddDate(0, 1, 0), "EUR", tz)
		require.NoError(t, err)
		require.Len(t, ret, 1)
		assert.Equal(t, int64(0), ret[0].MonthsSinceCohort)
		assert.Equal(t, int64(1000), ret[0].RevenueMinorUnits)
		return nil
	}))
}

// The dashboard's daily stock flow uses local days: a movement at 00:30
// yesterday in Tokyo (UTC+9, no DST) is 15:30 UTC the day before that.
func TestStockMovementRepo_Integration_DailyFlowUsesTheTenantTimezone(t *testing.T) {
	const tz = "Asia/Tokyo"
	tokyo, err := time.LoadLocation(tz)
	require.NoError(t, err)
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	movementRepo := repositories.NewStockMovementRepo(db)

	now := time.Now().In(tokyo)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, tokyo)
	yesterdayEarly := todayStart.AddDate(0, 0, -1).Add(30 * time.Minute)
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return movementRepo.Create(ctx, &invDomain.StockMovement{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID, Type: invDomain.MovementTypeReceipt,
			QuantityDelta: 5, OccurredAt: yesterdayEarly, CreatedAt: yesterdayEarly,
		})
	}))

	var flow []invDomain.DailyStockFlow
	require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		flow, err = movementRepo.DailyFlow(ctx, todayStart.AddDate(0, 0, -2), tz)
		return err
	}))
	byDate := map[string]int64{}
	for _, d := range flow {
		byDate[d.Date.Format("2006-01-02")] = d.In
	}
	assert.Len(t, flow, 3, "two days ago, yesterday, today -- local days")
	assert.Equal(t, int64(5), byDate[yesterdayEarly.Format("2006-01-02")], "flow by date: %v", byDate)
}
