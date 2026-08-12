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

func TestGetBooking_excludesNonProductLineItems(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1"}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", Quantity: 10, LineType: procurementDomain.LineItemTypeProduct},
		{ID: "line-2", Quantity: 1, LineType: procurementDomain.LineItemTypeShipping},
		{ID: "line-3", Quantity: 1, LineType: procurementDomain.LineItemTypeDiscount},
		// Empty LineType means "product" -- see LineItemType.IsProduct's
		// doc comment (pre-dates the classification column existing).
		{ID: "line-4", Quantity: 5, LineType: ""},
	}}
	uc := appprocurement.NewGetBookingUseCase(db, bookingRepo, lineItemRepo, &stubBookingScanRepo{})

	detail, err := uc.Execute(context.Background(), "booking-1")

	require.NoError(t, err)
	gotIDs := make([]string, len(detail.LineItems))
	for i, li := range detail.LineItems {
		gotIDs[i] = li.ID
	}
	assert.ElementsMatch(t, []string{"line-1", "line-4"}, gotIDs)
}

func TestGetBooking_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findErr: procurementDomain.ErrBookingNotFound}
	uc := appprocurement.NewGetBookingUseCase(db, bookingRepo, &stubLineItemRepo{}, &stubBookingScanRepo{})

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotFound)
}
