package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartOrResumeBooking_createsNewBookingWhenNoneActive(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("warehouse-1")}}
	bookingRepo := &stubBookingRepo{}
	uc := appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, bookingRepo)

	booking, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1"})

	require.NoError(t, err)
	require.Len(t, bookingRepo.created, 1)
	assert.Equal(t, "po-1", booking.PurchaseOrderID)
	assert.Equal(t, procurementDomain.BookingStatusInProgress, booking.Status)
}

func TestStartOrResumeBooking_reusesInProgressBooking(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("warehouse-1")}}
	bookingRepo := &stubBookingRepo{activeResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}}
	uc := appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, bookingRepo)

	booking, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1"})

	require.NoError(t, err)
	assert.Empty(t, bookingRepo.created, "must not create a duplicate booking")
	assert.Equal(t, "booking-1", booking.ID)
}

func TestStartOrResumeBooking_resumesPausedBooking(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("warehouse-1")}}
	bookingRepo := &stubBookingRepo{activeResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusPaused}}
	uc := appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, bookingRepo)

	booking, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1"})

	require.NoError(t, err)
	require.Len(t, bookingRepo.updated, 1)
	assert.Equal(t, procurementDomain.BookingStatusInProgress, bookingRepo.updated[0].Status)
	assert.Nil(t, bookingRepo.updated[0].PausedAt)
	assert.Equal(t, procurementDomain.BookingStatusInProgress, booking.Status)
}

func TestStartOrResumeBooking_propagatesPurchaseOrderNotFound(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{findErr: procurementDomain.ErrPurchaseOrderNotFound}
	uc := appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, &stubBookingRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "missing"})

	assert.ErrorIs(t, err, procurementDomain.ErrPurchaseOrderNotFound)
}

func TestStartOrResumeBooking_assignsWarehouseWhenPOHasNone(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	bookingRepo := &stubBookingRepo{}
	uc := appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, bookingRepo)

	booking, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{
		PurchaseOrderID: "po-1", WarehouseID: strPtr("warehouse-1"),
	})

	require.NoError(t, err)
	require.Len(t, poRepo.updated, 1)
	require.NotNil(t, poRepo.updated[0].WarehouseID)
	assert.Equal(t, "warehouse-1", *poRepo.updated[0].WarehouseID)
	assert.Equal(t, "po-1", booking.PurchaseOrderID)
}

func TestStartOrResumeBooking_rejectsWhenPOHasNoWarehouseAndNoneGiven(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}}
	uc := appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, &stubBookingRepo{})

	_, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{PurchaseOrderID: "po-1"})

	assert.ErrorIs(t, err, procurementDomain.ErrWarehouseRequired)
	assert.Empty(t, poRepo.updated)
}

func TestStartOrResumeBooking_ignoresWarehouseInputWhenPOAlreadyHasOne(t *testing.T) {
	db := newMockGormDB(t)
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", WarehouseID: strPtr("warehouse-1")}}
	bookingRepo := &stubBookingRepo{}
	uc := appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, bookingRepo)

	_, err := uc.Execute(context.Background(), appprocurement.StartOrResumeBookingInput{
		PurchaseOrderID: "po-1", WarehouseID: strPtr("warehouse-2"),
	})

	require.NoError(t, err)
	assert.Empty(t, poRepo.updated, "an already-assigned warehouse must never be silently overwritten")
}
