//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// receiptCostAdapter wires inventory's receipt hook to pms, as the
// composition root does.
type receiptCostAdapter struct {
	uc *apppms.ApplyReceiptCostUseCase
}

func (a receiptCostAdapter) Execute(ctx context.Context, in appinventory.SKUCostReceiptInput) error {
	return a.uc.Execute(ctx, apppms.ApplyReceiptCostInput{
		SKUID: in.SKUID, ReceivedQuantity: in.ReceivedQuantity, LandedUnitCostMinorUnits: in.LandedUnitCostMinorUnits,
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

// TestReceiveStock_Integration_CostPriceIsTheLastReceipt (COM-11): through
// real receive stock + Postgres, the cost price is the landed cost of the
// last purchase order receipt, the inventory cost the moving average of
// the stock, and a free receipt lowers only the average.
func TestReceiveStock_Integration_CostPriceIsTheLastReceipt(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)
	lineA := seedPOProductLine(t, db, ctx, poID, skuID)
	lineB := seedPOProductLine(t, db, ctx, poID, skuID)

	skuRepo := repositories.NewSKURepo(db)
	batchLister := appinventory.NewListBatchesUseCase(db, repositories.NewBatchRepo(db),
		repositories.NewStockMovementRepo(db), repositories.NewWarehouseRepo(db))
	outbox := repositories.NewOutboxRepository(db)
	costs := receiptCostAdapter{apppms.NewApplyReceiptCostUseCase(db, skuRepo, repositories.NewSKUCostPriceLedgerRepo(db), batchLister, outbox)}
	receive := appinventory.NewReceiveStockUseCase(db, repositories.NewBatchRepo(db), repositories.NewStockMovementRepo(db), outbox, costs)

	receiveAt := func(lot string, line string, qty, cost int64) {
		t.Helper()
		_, err := receive.Execute(ctx, appinventory.ReceiveStockInput{
			SKUID: skuID, WarehouseID: warehouseID, Quantity: qty, LotNumber: lot, Reason: "test",
			PurchaseOrderLineItemID: &line, UnitCostMinorUnits: &cost, OccurredAt: time.Now(),
		})
		require.NoError(t, err)
	}
	costsNow := func() (costPrice, inventoryCost int64) {
		t.Helper()
		err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
			sku, err := skuRepo.FindByID(ctx, skuID)
			if err != nil {
				return err
			}
			require.NotNil(t, sku.CostPriceMinorUnits)
			require.NotNil(t, sku.InventoryCostMinorUnits)
			costPrice, inventoryCost = *sku.CostPriceMinorUnits, *sku.InventoryCostMinorUnits
			return nil
		})
		require.NoError(t, err)
		return costPrice, inventoryCost
	}

	receiveAt("L1", lineA, 10, 500)
	receiveAt("L2", lineB, 10, 700)
	costPrice, inventoryCost := costsNow()
	assert.Equal(t, int64(700), costPrice, "last receipt's landed cost")
	assert.Equal(t, int64(600), inventoryCost, "(10*500 + 10*700) / 20")

	receiveAt("L3", lineB, 20, 0)
	costPrice, inventoryCost = costsNow()
	assert.Equal(t, int64(700), costPrice, "a free receipt never becomes the cost price")
	assert.Equal(t, int64(300), inventoryCost, "but it does lower what the stock cost")
}

func seedPOProductLine(t *testing.T, db *gorm.DB, ctx context.Context, poID, skuID string) string {
	t.Helper()
	line := procurementDomain.PurchaseOrderLineItem{
		PurchaseOrderID: poID, SKUID: &skuID, RawDescription: "Guarana 2L", Quantity: 10,
		UnitsPerPurchasedItem: 1, LineType: procurementDomain.LineItemTypeProduct, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		line.TenantID = tenant.IDFromContext(ctx)
		return repositories.NewPurchaseOrderLineItemRepo(db).Create(ctx, &line)
	})
	require.NoError(t, err)
	return line.ID
}
