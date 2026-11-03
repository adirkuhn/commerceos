//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestRecordGoodsReceivedCost_Integration: a goods receipt lands in the
// supplier cost history with the PO's supplier and currency, the same
// event twice adds one row, and rows read back oldest first per SKU and
// per line.
func TestRecordGoodsReceivedCost_Integration(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)
	lineA := seedPOProductLine(t, db, ctx, poID, skuID)
	lineB := seedPOProductLine(t, db, ctx, poID, skuID)

	costs := repositories.NewSupplierCostRepo(db)
	uc := appprocurement.NewRecordGoodsReceivedCostUseCase(db, repositories.NewPurchaseOrderLineItemRepo(db),
		repositories.NewPurchaseOrderRepo(db), costs, &eventrecordtest.Outbox{})
	first := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	receive := func(eventID, line string, at time.Time, landed int64) {
		t.Helper()
		require.NoError(t, uc.Execute(ctx, appprocurement.RecordGoodsReceivedCostInput{
			EventID: eventID, SKUID: skuID, PurchaseOrderLineItemID: line, Quantity: 10, LandedUnitCostMinorUnits: landed, ReceivedAt: at,
		}))
	}
	receive("evt-2", lineB, first.Add(time.Minute), 700)
	receive("evt-1", lineA, first, 500)
	receive("evt-1", lineA, first, 500) // redelivered

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		bySKU, err := costs.ListBySKU(ctx, skuID)
		require.NoError(t, err)
		require.Len(t, bySKU, 2)
		assert.Equal(t, lineA, bySKU[0].PurchaseOrderLineItemID, "oldest first")
		assert.Equal(t, procurementDomain.SupplierCostKindReceipt, bySKU[0].Kind)
		assert.Equal(t, poID, bySKU[0].PurchaseOrderID)
		assert.Equal(t, int64(500), bySKU[0].LandedUnitCostMinorUnits)
		assert.NotEmpty(t, bySKU[0].CurrencyCode)

		byLine, err := costs.ListByLine(ctx, lineB)
		require.NoError(t, err)
		require.Len(t, byLine, 1)
		assert.Equal(t, int64(700), byLine[0].LandedUnitCostMinorUnits)

		s := procurementDomain.SummarizeSupplierCosts(bySKU, bySKU[0].CurrencyCode)
		require.NotNil(t, s.Last)
		assert.Equal(t, int64(700), s.Last.LandedUnitCostMinorUnits)
		assert.Equal(t, int64(600), *s.AverageMinorUnits)
		return nil
	})
	require.NoError(t, err)
}
