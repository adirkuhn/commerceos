//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// seedStuckPurchaseOrder makes a PO for 10 units delivered as receivedA +
// receivedB over two completed bookings, left partially_received.
func seedStuckPurchaseOrder(t *testing.T, db *gorm.DB, ctx context.Context, skuID, warehouseID string, receivedA, receivedB int64) string {
	t.Helper()
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	bookingRepo := repositories.NewBookingRepo(db)
	scanRepo := repositories.NewBookingScanRepo(db)
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		tenantID := tenant.IDFromContext(ctx)
		if err := repositories.NewPurchaseOrderLineItemRepo(db).Create(ctx, &procurementDomain.PurchaseOrderLineItem{
			TenantID: tenantID, PurchaseOrderID: poID, SKUID: &skuID, RawDescription: "Guarana 2L", Quantity: 10,
			UnitsPerPurchasedItem: 1, LineType: procurementDomain.LineItemTypeProduct, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}); err != nil {
			return err
		}
		for _, qty := range []int64{receivedA, receivedB} {
			now := time.Now()
			b := procurementDomain.Booking{TenantID: tenantID, PurchaseOrderID: poID, Status: procurementDomain.BookingStatusCompleted, StartedAt: now, CompletedAt: &now}
			if err := bookingRepo.Create(ctx, &b); err != nil {
				return err
			}
			if err := scanRepo.Create(ctx, &procurementDomain.BookingScan{
				TenantID: tenantID, BookingID: b.ID, SKUID: skuID, ActionType: procurementDomain.BookingScanActionReceive,
				Quantity: qty, ScannedAt: now, ScannedBy: "user-1",
			}); err != nil {
				return err
			}
		}
		po, err := poRepo.FindByID(ctx, poID)
		if err != nil {
			return err
		}
		po.Status = procurementDomain.PurchaseOrderStatusPartiallyReceived
		return poRepo.Update(ctx, po)
	})
	require.NoError(t, err)
	return poID
}

func poStatus(t *testing.T, db *gorm.DB, ctx context.Context, id string) procurementDomain.PurchaseOrderStatus {
	t.Helper()
	var status procurementDomain.PurchaseOrderStatus
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		po, err := repositories.NewPurchaseOrderRepo(db).FindByID(ctx, id)
		if err == nil {
			status = po.Status
		}
		return err
	})
	require.NoError(t, err)
	return status
}

// Runs the repair as commerceos_app, one tenant at a time like
// cmd/repair-partially-received-pos does: a complete PO becomes received
// with its event in the outbox, a really short one stays, and another
// tenant's stuck PO is not touched.
func TestReconcilePartiallyReceived_Integration(t *testing.T) {
	superDB, dsn := newIntegrationDBAndDSN(t)
	db := newAppRoleDB(t, dsn)
	_, ctx := newTestTenant()
	_, otherCtx := newTestTenant()
	skuID, warehouseID := seedProductSKUWarehouse(t, superDB, ctx)
	otherSKUID, otherWarehouseID := seedProductSKUWarehouse(t, superDB, otherCtx)
	stuck := seedStuckPurchaseOrder(t, superDB, ctx, skuID, warehouseID, 4, 6)
	short := seedStuckPurchaseOrder(t, superDB, ctx, skuID, warehouseID, 4, 3)
	otherTenants := seedStuckPurchaseOrder(t, superDB, otherCtx, otherSKUID, otherWarehouseID, 5, 5)

	uc := appprocurement.NewReconcilePartiallyReceivedPurchaseOrdersUseCase(db, repositories.NewPurchaseOrderRepo(db),
		repositories.NewBookingRepo(db), repositories.NewPurchaseOrderLineItemRepo(db), repositories.NewBookingScanRepo(db),
		repositories.NewOutboxRepository(db))

	dry, err := uc.Execute(ctx, appprocurement.ReconcilePartiallyReceivedInput{DryRun: true})
	require.NoError(t, err)
	require.Equal(t, []string{stuck}, dry.Fixed)
	require.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, poStatus(t, superDB, ctx, stuck), "dry run writes nothing")

	got, err := uc.Execute(ctx, appprocurement.ReconcilePartiallyReceivedInput{})
	require.NoError(t, err)
	require.Equal(t, 2, got.Checked)
	require.Equal(t, []string{stuck}, got.Fixed)
	require.Equal(t, []string{short}, got.StillShort)

	require.Equal(t, procurementDomain.PurchaseOrderStatusReceived, poStatus(t, superDB, ctx, stuck))
	require.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, poStatus(t, superDB, ctx, short))
	require.Equal(t, procurementDomain.PurchaseOrderStatusPartiallyReceived, poStatus(t, superDB, otherCtx, otherTenants))

	var events int64
	require.NoError(t, superDB.Table("outbox_messages").
		Where("subject = ? AND payload::text LIKE ?", "procurement.purchase_order.updated", "%deliveries_reconciled%").
		Count(&events).Error)
	require.Equal(t, int64(1), events)

	again, err := uc.Execute(ctx, appprocurement.ReconcilePartiallyReceivedInput{})
	require.NoError(t, err)
	require.Empty(t, again.Fixed, "safe to run again")
}
