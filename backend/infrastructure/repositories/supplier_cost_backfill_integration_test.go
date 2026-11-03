//go:build integration

package repositories_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// backfillStatements are migration 242's backfill INSERTs: everything from
// "-- Backfill 1" up to the Down section, split on ";".
func backfillStatements(t *testing.T) []string {
	t.Helper()
	raw, err := os.ReadFile("../db/migrations/242_supplier_cost_entries.sql")
	require.NoError(t, err)
	s := string(raw)
	s = s[strings.Index(s, "-- Backfill 1"):strings.Index(s, "-- +goose Down")]
	var out []string
	for _, stmt := range strings.Split(s, ";") {
		if strings.Contains(stmt, "INSERT INTO") {
			out = append(out, stmt)
		}
	}
	require.Len(t, out, 2)
	return out
}

// TestSupplierCostBackfill_Integration runs migration 242's backfill over
// past data: a booked receipt (batch with landed cost) becomes one row per
// receipt movement; a line of a received PO that no batch accounts for
// becomes one row whose landed cost matches AllocateLandedUnitCost; and
// re-running adds nothing.
func TestSupplierCostBackfill_Integration(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)
	booked := seedPOProductLine(t, db, ctx, poID, skuID)   // 10 x 0
	unbooked := seedPOProductLine(t, db, ctx, poID, skuID) // 10 x 0
	lineRepo := repositories.NewPurchaseOrderLineItemRepo(db)

	var lines []procurementDomain.PurchaseOrderLineItem
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		for _, id := range []string{booked, unbooked} {
			li, err := lineRepo.FindByID(ctx, id)
			if err != nil {
				return err
			}
			li.UnitCostMinorUnits = 500
			if err := lineRepo.Update(ctx, li); err != nil {
				return err
			}
		}
		if err := lineRepo.Create(ctx, &procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID, RawDescription: "Shipping", Quantity: 1,
			UnitsPerPurchasedItem: 1, UnitCostMinorUnits: 200, LineType: procurementDomain.LineItemTypeShipping,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}); err != nil {
			return err
		}
		poRepo := repositories.NewPurchaseOrderRepo(db)
		po, err := poRepo.FindByID(ctx, poID)
		if err != nil {
			return err
		}
		po.Status = procurementDomain.PurchaseOrderStatusReceived
		if err := poRepo.Update(ctx, po); err != nil {
			return err
		}
		lines, err = lineRepo.ListByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)

	landed := int64(520) // what the booking stored on the batch
	receive := appinventory.NewReceiveStockUseCase(db, repositories.NewBatchRepo(db), repositories.NewStockMovementRepo(db),
		repositories.NewOutboxRepository(db), nil)
	_, err = receive.Execute(ctx, appinventory.ReceiveStockInput{
		SKUID: skuID, WarehouseID: warehouseID, Quantity: 10, LotNumber: "L1", Reason: "test",
		PurchaseOrderLineItemID: &booked, UnitCostMinorUnits: &landed, OccurredAt: time.Now(),
	})
	require.NoError(t, err)

	for range 2 { // a second run adds nothing
		for _, stmt := range backfillStatements(t) {
			require.NoError(t, db.Exec(stmt).Error)
		}
	}

	costs := repositories.NewSupplierCostRepo(db)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		rows, err := costs.ListBySKU(ctx, skuID)
		require.NoError(t, err)
		require.Len(t, rows, 2)
		byLine := map[string]procurementDomain.SupplierCostEntry{}
		for _, r := range rows {
			byLine[r.PurchaseOrderLineItemID] = r
			assert.Equal(t, procurementDomain.SupplierCostKindReceipt, r.Kind)
			assert.Equal(t, poID, r.PurchaseOrderID)
			assert.NotNil(t, r.SupplierID)
			assert.Equal(t, "USD", r.CurrencyCode)
			assert.Equal(t, int64(500), r.UnitCostMinorUnits)
			assert.Equal(t, int64(10), r.Quantity)
		}
		assert.Equal(t, int64(520), byLine[booked].LandedUnitCostMinorUnits, "the batch's landed cost")
		assert.NotNil(t, byLine[booked].BatchID)
		want, err := appprocurement.AllocateLandedUnitCost(lines, unbooked)
		require.NoError(t, err)
		assert.Equal(t, int64(510), want)
		assert.Equal(t, want, byLine[unbooked].LandedUnitCostMinorUnits, "same allocation as the Go code")
		return nil
	})
	require.NoError(t, err)
}
