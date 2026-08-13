//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	suppliersDomain "github.com/adirkuhn/commerceos/domain/suppliers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func seedPurchaseOrderForBookingDB(t *testing.T, db *gorm.DB, ctx context.Context, warehouseID string) string {
	t.Helper()
	supplierRepo := repositories.NewSupplierRepo(db)
	poRepo := repositories.NewPurchaseOrderRepo(db)
	var poID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		supplier := &suppliersDomain.Supplier{
			TenantID: tenant.IDFromContext(ctx), Name: "Acme Foods", Status: suppliersDomain.SupplierStatusActive,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := supplierRepo.Create(ctx, supplier); err != nil {
			return err
		}
		po := procurementDomain.PurchaseOrder{
			TenantID: tenant.IDFromContext(ctx), SupplierID: &supplier.ID, WarehouseID: &warehouseID,
			Status: procurementDomain.PurchaseOrderStatusSent, CurrencyCode: "USD", CreatedAt: now, UpdatedAt: now,
		}
		if err := poRepo.Create(ctx, &po); err != nil {
			return err
		}
		poID = po.ID
		return nil
	})
	require.NoError(t, err)
	return poID
}

func TestBookingRepo_Integration_CreateFindActiveUpdate(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	bookingRepo := repositories.NewBookingRepo(db)
	_, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)

	var created procurementDomain.Booking
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		created = procurementDomain.Booking{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
			Status: procurementDomain.BookingStatusInProgress, StartedAt: time.Now(),
		}
		return bookingRepo.Create(ctx, &created)
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)

	var active *procurementDomain.Booking
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		active, err = bookingRepo.FindActiveByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, created.ID, active.ID)

	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		now := time.Now()
		created.Status = procurementDomain.BookingStatusCompleted
		created.CompletedAt = &now
		created.CompletedBy = "user-1"
		return bookingRepo.Update(ctx, &created)
	})
	require.NoError(t, err)

	var found *procurementDomain.Booking
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		found, err = bookingRepo.FindByID(ctx, created.ID)
		return err
	})
	require.NoError(t, err)
	require.Equal(t, procurementDomain.BookingStatusCompleted, found.Status)
	require.Equal(t, "user-1", found.CompletedBy)

	// Completed bookings are no longer "active" for a new scan session to resume.
	var activeAfterComplete *procurementDomain.Booking
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		activeAfterComplete, err = bookingRepo.FindActiveByPurchaseOrder(ctx, poID)
		return err
	})
	require.NoError(t, err)
	require.Nil(t, activeAfterComplete)
}

func TestBookingRepo_Integration_NotFound(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	bookingRepo := repositories.NewBookingRepo(db)

	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		_, err := bookingRepo.FindByID(ctx, "00000000-0000-0000-0000-000000000000")
		return err
	})
	require.ErrorIs(t, err, procurementDomain.ErrBookingNotFound)
}

func TestBookingScanRepo_Integration_CreateAndList(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	bookingRepo := repositories.NewBookingRepo(db)
	scanRepo := repositories.NewBookingScanRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)

	var bookingID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		booking := procurementDomain.Booking{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
			Status: procurementDomain.BookingStatusInProgress, StartedAt: time.Now(),
		}
		if err := bookingRepo.Create(ctx, &booking); err != nil {
			return err
		}
		bookingID = booking.ID

		scan := procurementDomain.BookingScan{
			TenantID: tenant.IDFromContext(ctx), BookingID: bookingID, SKUID: skuID,
			Quantity: 5, ScannedAt: time.Now(), ScannedBy: "user-1",
		}
		return scanRepo.Create(ctx, &scan)
	})
	require.NoError(t, err)

	var scans []procurementDomain.BookingScan
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		scans, err = scanRepo.ListByBooking(ctx, bookingID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, scans, 1)
	require.Equal(t, int64(5), scans[0].Quantity)
	require.Equal(t, skuID, scans[0].SKUID)
	require.Nil(t, scans[0].ReplayedAt, "a freshly-staged scan is not replayed yet")
}

func TestBookingScanRepo_Integration_MarkReplayed(t *testing.T) {
	db := newIntegrationDB(t)
	_, ctx := newTestTenant()
	bookingRepo := repositories.NewBookingRepo(db)
	scanRepo := repositories.NewBookingScanRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	skuID, warehouseID := seedProductSKUWarehouse(t, db, ctx)
	poID := seedPurchaseOrderForBookingDB(t, db, ctx, warehouseID)

	var bookingID, scanID, realBatchID string
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		booking := procurementDomain.Booking{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: poID,
			Status: procurementDomain.BookingStatusInProgress, StartedAt: time.Now(),
		}
		if err := bookingRepo.Create(ctx, &booking); err != nil {
			return err
		}
		bookingID = booking.ID

		// LotNumber set, no BatchID -- this row's own receive is what
		// creates the batch at replay time (see BookingScan's doc comment).
		scan := procurementDomain.BookingScan{
			TenantID: tenant.IDFromContext(ctx), BookingID: bookingID, SKUID: skuID,
			ActionType: procurementDomain.BookingScanActionReceive, Quantity: 5, LotNumber: "LOT-INT-1",
			ScannedAt: time.Now(), ScannedBy: "user-1",
		}
		if err := scanRepo.Create(ctx, &scan); err != nil {
			return err
		}
		scanID = scan.ID

		// The real batch replay would have created for this row -- MarkReplayed
		// must satisfy booking_scans_batch_id_fkey, same as the real path.
		now := time.Now()
		batch := invDomain.Batch{
			TenantID: tenant.IDFromContext(ctx), SKUID: skuID, WarehouseID: warehouseID,
			LotNumber: "LOT-INT-1", ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if err := batchRepo.Create(ctx, &batch); err != nil {
			return err
		}
		realBatchID = batch.ID
		return nil
	})
	require.NoError(t, err)

	replayedAt := time.Now().Truncate(time.Microsecond)
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		return scanRepo.MarkReplayed(ctx, scanID, replayedAt, &realBatchID)
	})
	require.NoError(t, err)

	var scans []procurementDomain.BookingScan
	err = tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		var err error
		scans, err = scanRepo.ListByBooking(ctx, bookingID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, scans, 1)
	require.NotNil(t, scans[0].ReplayedAt, "MarkReplayed must persist -- a retried Finish needs this to skip the row")
	require.WithinDuration(t, replayedAt, *scans[0].ReplayedAt, time.Second)
	require.NotNil(t, scans[0].BatchID, "MarkReplayed also resolves the not-yet-real batch to its real id")
	require.Equal(t, realBatchID, *scans[0].BatchID)
}
