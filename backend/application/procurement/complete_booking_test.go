package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteBooking_noMissingItemsWhenEverythingScanned(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: &skuID, RawDescription: "Widget A", Quantity: 10}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{{SKUID: "sku-1", Quantity: 10}}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, lineItemRepo, scanRepo)

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	assert.Empty(t, result.MissingItems)
	require.Len(t, bookingRepo.updated, 1)
	assert.Equal(t, procurementDomain.BookingStatusCompleted, bookingRepo.updated[0].Status)
	assert.Equal(t, "user-1", bookingRepo.updated[0].CompletedBy)
	assert.NotNil(t, bookingRepo.updated[0].CompletedAt)
}

func TestCompleteBooking_reportsMissingItemsWhenUnderScanned(t *testing.T) {
	db := newMockGormDB(t)
	skuID := "sku-1"
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusInProgress}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", SKUID: &skuID, RawDescription: "Widget A", Quantity: 10},
		{ID: "line-2", SKUID: nil, RawDescription: "Widget B (unlinked)", Quantity: 5},
	}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{{SKUID: "sku-1", Quantity: 6}}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, lineItemRepo, scanRepo)

	result, err := uc.Execute(context.Background(), "booking-1", "user-1")

	require.NoError(t, err)
	require.Len(t, result.MissingItems, 2)
	assert.Equal(t, "line-1", result.MissingItems[0].LineItemID)
	assert.Equal(t, int64(10), result.MissingItems[0].ExpectedQty)
	assert.Equal(t, int64(6), result.MissingItems[0].ScannedQty)
	assert.Equal(t, "line-2", result.MissingItems[1].LineItemID)
	assert.Equal(t, int64(0), result.MissingItems[1].ScannedQty)
	// Completion still succeeds even with gaps -- the report is surfaced,
	// not a blocker.
	assert.Equal(t, procurementDomain.BookingStatusCompleted, result.Booking.Status)
}

func TestCompleteBooking_rejectsAlreadyCompletedBooking(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusCompleted}}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, &stubLineItemRepo{}, &stubBookingScanRepo{})

	_, err := uc.Execute(context.Background(), "booking-1", "user-1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotActive)
}

func TestCompleteBooking_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findErr: procurementDomain.ErrBookingNotFound}
	uc := appprocurement.NewCompleteBookingUseCase(db, bookingRepo, &stubLineItemRepo{}, &stubBookingScanRepo{})

	_, err := uc.Execute(context.Background(), "missing", "user-1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotFound)
}
