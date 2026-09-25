//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// revenueFixture is one customer + one SKU, shared by the revenue-status
// tests below -- every report query needs some subset of customer/SKU/
// line item to find an order at all.
type revenueFixture struct {
	db          *gorm.DB
	ctx         context.Context
	repo        *repositories.SalesOrderRepo
	lineRepo    *repositories.SalesOrderLineItemRepo
	skuID       string
	warehouseID string
	customerID  string
	now         time.Time
}

func newRevenueFixture(t *testing.T) revenueFixture {
	t.Helper()
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	f := revenueFixture{
		db: db, ctx: ctx, repo: repositories.NewSalesOrderRepo(db), lineRepo: repositories.NewSalesOrderLineItemRepo(db),
		skuID: skuID, warehouseID: warehouseID, now: time.Now(),
	}
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		c := customersDomain.Customer{
			TenantID: tenant.IDFromContext(ctx), Email: "rev@example.com", FirstName: "Rev", LastName: "Enue",
			DateCreated: f.now, DateModified: f.now,
		}
		if err := repositories.NewCustomerRepo(db).Create(ctx, &c); err != nil {
			return err
		}
		f.customerID = c.ID
		return nil
	})
	require.NoError(t, err)
	return f
}

// seedOrder creates one order in the given status with a single line of
// qty units totalling totalMinorUnits, for the fixture's customer and SKU.
// Returns the line item id.
func (f revenueFixture) seedOrder(t *testing.T, extID string, status omsDomain.SalesOrderStatus, qty, totalMinorUnits int64) string {
	t.Helper()
	var lineID string
	err := tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		o := omsDomain.SalesOrder{
			TenantID: tenant.IDFromContext(ctx), Source: omsDomain.SalesOrderSourceManual, ExternalOrderID: extID,
			WarehouseID: f.warehouseID, Status: status, CurrencyCode: "EUR", CustomerID: &f.customerID,
			TotalMinorUnits: totalMinorUnits, OccurredAt: f.now, CreatedAt: f.now,
		}
		if err := f.repo.Create(ctx, &o); err != nil {
			return err
		}
		li := omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: o.ID, SKUID: &f.skuID, SKUCode: "SKU-REV", Name: "Rev",
			Quantity: qty, PriceMinorUnits: totalMinorUnits / qty, TotalMinorUnits: totalMinorUnits, CreatedAt: f.now,
		}
		if err := f.lineRepo.Create(ctx, &li); err != nil {
			return err
		}
		lineID = li.ID
		return nil
	})
	require.NoError(t, err)
	return lineID
}

// revenueReport is what every revenue query reports for the fixture's
// single currency/window, reduced to comparable numbers.
type revenueReport struct {
	revenue int64 // order-level revenue (sum of order totals)
	units   int64 // units sold
	orders  int64 // orders counted
}

// reports runs every revenue query in the repository over a window around
// f.now and returns each one's view, keyed by query name -- so one test can
// assert all of them agree on which orders count.
func (f revenueFixture) reports(t *testing.T) map[string]revenueReport {
	t.Helper()
	from, to := f.now.AddDate(0, 0, -1), f.now.AddDate(0, 0, 1)
	out := map[string]revenueReport{}
	err := tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		r := f.repo
		rev, err := r.RevenueSince(ctx, from, "EUR")
		if err != nil {
			return err
		}
		out["RevenueSince"] = revenueReport{revenue: rev}

		byCh, err := r.RevenueByChannelSince(ctx, from, "EUR")
		if err != nil {
			return err
		}
		out["RevenueByChannelSince"] = revenueReport{revenue: sumChannel(byCh)}

		byChB, err := r.RevenueByChannelBetween(ctx, from, to, "EUR")
		if err != nil {
			return err
		}
		out["RevenueByChannelBetween"] = revenueReport{revenue: sumChannel(byChB)}

		sum, err := r.SalesSummaryBetween(ctx, from, to, "EUR")
		if err != nil {
			return err
		}
		out["SalesSummaryBetween"] = revenueReport{revenue: sum.RevenueMinorUnits, units: sum.UnitsSold, orders: sum.OrderCount}

		trend, err := r.SalesTrendBetween(ctx, from, to, "EUR", omsDomain.SalesTrendGranularityDay, "UTC")
		if err != nil {
			return err
		}
		var tr revenueReport
		for _, p := range trend {
			tr.revenue += p.RevenueMinorUnits
			tr.units += p.UnitsSold
			tr.orders += p.OrderCount
		}
		out["SalesTrendBetween"] = tr

		wd, err := r.SalesByWeekdayBetween(ctx, from, to, "EUR", "UTC")
		if err != nil {
			return err
		}
		var w revenueReport
		for _, p := range wd {
			w.revenue += p.RevenueMinorUnits
			w.orders += p.OrderCount
		}
		out["SalesByWeekdayBetween"] = w

		cht, err := r.RevenueByChannelTrendBetween(ctx, from, to, "EUR", omsDomain.SalesTrendGranularityDay, "UTC")
		if err != nil {
			return err
		}
		var c revenueReport
		for _, p := range cht {
			c.revenue += p.RevenueMinorUnits
		}
		out["RevenueByChannelTrendBetween"] = c

		pp, err := r.ProductPerformanceBetween(ctx, from, to, "EUR")
		if err != nil {
			return err
		}
		var p revenueReport
		for _, row := range pp {
			p.revenue += row.RevenueMinorUnits
			p.units += row.QuantitySold
		}
		out["ProductPerformanceBetween"] = p

		cp, err := r.CustomerPerformanceBetween(ctx, from, to, "EUR")
		if err != nil {
			return err
		}
		var cu revenueReport
		for _, row := range cp {
			cu.revenue += row.RevenueMinorUnits
			cu.orders += row.OrderCount
		}
		out["CustomerPerformanceBetween"] = cu

		cat, err := r.CategoryPerformanceBetween(ctx, from, to, "EUR")
		if err != nil {
			return err
		}
		var ca revenueReport
		for _, row := range cat {
			ca.revenue += row.RevenueMinorUnits
			ca.units += row.QuantitySold
		}
		out["CategoryPerformanceBetween"] = ca

		prof, err := r.ProfitabilityBetween(ctx, from, to, "EUR")
		if err != nil {
			return err
		}
		out["ProfitabilityBetween"] = revenueReport{revenue: prof.RevenueMinorUnits}

		pt, err := r.ProfitabilityTrendBetween(ctx, from, to, "EUR", omsDomain.SalesTrendGranularityDay, "UTC")
		if err != nil {
			return err
		}
		var ptr revenueReport
		for _, row := range pt {
			ptr.revenue += row.RevenueMinorUnits
		}
		out["ProfitabilityTrendBetween"] = ptr

		top, err := r.TopProductsSince(ctx, from, "EUR", 10)
		if err != nil {
			return err
		}
		var tp revenueReport
		for _, row := range top {
			tp.units += row.QuantitySold
		}
		out["TopProductsSince"] = tp

		set, err := r.SKUSetSalesBetween(ctx, []string{f.skuID}, from, to, "EUR")
		if err != nil {
			return err
		}
		out["SKUSetSalesBetween"] = revenueReport{revenue: set.RevenueMinorUnits, units: set.UnitsSold, orders: set.OrderCount}

		osk, err := r.ListOrderSKUsBetween(ctx, from, to, "EUR")
		if err != nil {
			return err
		}
		out["ListOrderSKUsBetween"] = revenueReport{orders: int64(len(osk))}

		cohortFrom := time.Date(f.now.Year(), f.now.Month(), 1, 0, 0, 0, 0, f.now.Location())
		sizes, err := r.CohortSizesBetween(ctx, cohortFrom, cohortFrom.AddDate(0, 1, 0), "EUR", "UTC")
		if err != nil {
			return err
		}
		var cs revenueReport
		for _, s := range sizes {
			cs.orders += s.CustomerCount
		}
		out["CohortSizesBetween"] = cs

		ret, err := r.CohortRetentionBetween(ctx, cohortFrom, cohortFrom.AddDate(0, 1, 0), "EUR", "UTC")
		if err != nil {
			return err
		}
		var rt revenueReport
		for _, p := range ret {
			rt.revenue += p.RevenueMinorUnits
		}
		out["CohortRetentionBetween"] = rt
		return nil
	})
	require.NoError(t, err)
	return out
}

func sumChannel(rows []omsDomain.ChannelRevenue) int64 {
	var s int64
	for _, r := range rows {
		s += r.TotalMinorUnits
	}
	return s
}

// A shipped order (set by MyParcel's order_status_change webhook) is a real
// sale: it took stock (SalesOrderStatus.ReducesStock) and was paid for. It
// used to be missing from every report because each query hard-coded its
// own copy of the revenue status list, and those copies predated "shipped".
func TestSalesOrderRepo_Integration_ShippedOrderCountsInEveryRevenueReport(t *testing.T) {
	f := newRevenueFixture(t)
	f.seedOrder(t, "rev-shipped", omsDomain.SalesOrderStatusShipped, 2, 1000)

	for name, got := range f.reports(t) {
		switch name {
		case "TopProductsSince":
			assert.Equal(t, int64(2), got.units, name)
		case "ListOrderSKUsBetween", "CohortSizesBetween":
			assert.Equal(t, int64(1), got.orders, name)
		default:
			assert.Equal(t, int64(1000), got.revenue, name)
		}
	}
}
