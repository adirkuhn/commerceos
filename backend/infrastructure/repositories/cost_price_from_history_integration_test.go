//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/costhistory"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// TestCostPriceFromHistory_Integration (COM-11) on real Postgres: the cost
// price follows the history's last receipt -- a fully reversed receipt
// falls back to the one before, and a correction of the last receipt
// reprices it.
func TestCostPriceFromHistory_Integration(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)
	older := seedPOProductLine(t, db, ctx, poID, skuID)
	newer := seedPOProductLine(t, db, ctx, poID, skuID)

	costs := repositories.NewSupplierCostRepo(db)
	outbox := &eventrecordtest.Outbox{}
	record := appprocurement.NewRecordGoodsReceivedCostUseCase(db, repositories.NewPurchaseOrderLineItemRepo(db),
		repositories.NewPurchaseOrderRepo(db), costs, outbox)
	reverse := appprocurement.NewReverseGoodsReceivedCostUseCase(db, costs, outbox)
	correct := appprocurement.NewCorrectSupplierCostUseCase(db, costs, outbox)
	skuRepo := repositories.NewSKURepo(db)
	sync := apppms.NewSyncCostPriceFromHistoryUseCase(db, skuRepo, costhistory.NewLastReceiptFromDB(db), outbox)

	costPrice := func() int64 {
		t.Helper()
		require.NoError(t, sync.Execute(ctx, skuID))
		var cost int64
		require.NoError(t, tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			sku, err := skuRepo.FindByID(ctx, skuID)
			if err == nil && sku.CostPriceMinorUnits != nil {
				cost = *sku.CostPriceMinorUnits
			}
			return err
		}))
		return cost
	}

	at := time.Now().Add(-time.Hour)
	require.NoError(t, record.Execute(ctx, appprocurement.RecordGoodsReceivedCostInput{EventID: "e1", SKUID: skuID, PurchaseOrderLineItemID: older, Quantity: 10, LandedUnitCostMinorUnits: 500, ReceivedAt: at}))
	require.NoError(t, record.Execute(ctx, appprocurement.RecordGoodsReceivedCostInput{EventID: "e2", SKUID: skuID, PurchaseOrderLineItemID: newer, Quantity: 10, LandedUnitCostMinorUnits: 800, ReceivedAt: at.Add(time.Minute)}))
	assert.Equal(t, int64(800), costPrice())

	require.NoError(t, reverse.Execute(ctx, appprocurement.ReverseGoodsReceivedCostInput{EventID: "e3", SKUID: skuID, PurchaseOrderLineItemID: newer, Quantity: -10, At: at.Add(2 * time.Minute)}))
	assert.Equal(t, int64(500), costPrice(), "the last receipt was taken back: back to the one before")

	require.NoError(t, correct.Execute(ctx, appprocurement.CorrectSupplierCostInput{EventID: "e4", SKUID: skuID, PurchaseOrderLineItemID: older, AdjustmentValueMinorUnits: 500, At: at.Add(3 * time.Minute)}))
	assert.Equal(t, int64(550), costPrice(), "its invoice was corrected: 500 + 500/10")
}
