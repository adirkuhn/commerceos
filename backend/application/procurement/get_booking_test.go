package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBooking_returnsBookingLineItemsAndScans(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1"}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", Quantity: 10}}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{{ID: "scan-1", BookingID: "booking-1", Quantity: 4}}}
	uc := appprocurement.NewGetBookingUseCase(db, bookingRepo, lineItemRepo, scanRepo)

	detail, err := uc.Execute(context.Background(), "booking-1")

	require.NoError(t, err)
	assert.Equal(t, "booking-1", detail.Booking.ID)
	require.Len(t, detail.LineItems, 1)
	require.Len(t, detail.Scans, 1)
	assert.Equal(t, int64(4), detail.Scans[0].Quantity)
}

func TestGetBooking_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findErr: procurementDomain.ErrBookingNotFound}
	uc := appprocurement.NewGetBookingUseCase(db, bookingRepo, &stubLineItemRepo{}, &stubBookingScanRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotFound)
}
