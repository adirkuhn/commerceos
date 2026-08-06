//go:build integration

package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

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
}
