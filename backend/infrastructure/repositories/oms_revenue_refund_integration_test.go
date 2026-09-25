//go:build integration

package repositories_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// refundLine records a refund against a line the way RefundLineItemUseCase
// does: RefundedQuantity units at the line's unit price.
func (f revenueFixture) refundLine(t *testing.T, lineID string, qty, amountMinorUnits int64) {
	t.Helper()
	require.NoError(t, tenant.WithTenant(f.ctx, f.db, func(ctx context.Context) error {
		return f.lineRepo.Update(ctx, lineID, map[string]any{"refunded_quantity": qty, "refunded_amount_minor_units": amountMinorUnits})
	}))
}

// Revenue is what the customer actually kept: a partially refunded order
// counts, net of its refunded lines -- and so does an order whose refund
// is recorded on its lines but still awaits manual review (status left
// untouched, see SalesOrder.NeedsManualRefundReview). A fully refunded
// order is excluded entirely.
func TestSalesOrderRepo_Integration_RevenueReportsAreNetOfRefundedLines(t *testing.T) {
	f := newRevenueFixture(t)
	// 3 units @ 500, 1 refunded -> 2 units / 1000 kept.
	partial := f.seedOrder(t, "rev-partial", omsDomain.SalesOrderStatusPartiallyRefunded, 3, 1500)
	f.refundLine(t, partial, 1, 500)
	// Refund recorded, manual review pending: 4 units @ 250, 2 refunded -> 2 units / 500 kept.
	review := f.seedOrder(t, "rev-review", omsDomain.SalesOrderStatusProcessing, 4, 1000)
	f.refundLine(t, review, 2, 500)
	// Fully refunded order: never revenue.
	f.seedOrder(t, "rev-refunded", omsDomain.SalesOrderStatusRefunded, 1, 9999)

	for name, got := range f.reports(t) {
		switch name {
		case "TopProductsSince":
			assert.Equal(t, int64(4), got.units, name)
		case "ListOrderSKUsBetween":
			assert.Equal(t, int64(2), got.orders, name)
		case "CohortSizesBetween":
			assert.Equal(t, int64(1), got.orders, name)
		default:
			assert.Equal(t, int64(1500), got.revenue, name)
			if got.units != 0 || name == "SalesSummaryBetween" || name == "ProductPerformanceBetween" {
				assert.Equal(t, int64(4), got.units, name+" units")
			}
		}
	}
}

// A line refunded in full no longer counts as the SKU having sold in that
// order -- it's dropped from line-level order counts and basket pairs.
func TestSalesOrderRepo_Integration_FullyRefundedLineDropsOutOfLineLevelReports(t *testing.T) {
	f := newRevenueFixture(t)
	line := f.seedOrder(t, "rev-line-gone", omsDomain.SalesOrderStatusPartiallyRefunded, 2, 1000)
	f.refundLine(t, line, 2, 1000)

	got := f.reports(t)
	assert.Equal(t, int64(0), got["ListOrderSKUsBetween"].orders)
	assert.Equal(t, int64(0), got["SKUSetSalesBetween"].orders)
	assert.Equal(t, int64(0), got["ProductPerformanceBetween"].units)
}
