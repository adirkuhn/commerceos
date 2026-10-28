package procurement_test

import (
	"context"
	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteBookingScan_deletesAStagedRowBelongingToAnInProgressBooking(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo, _ := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 1},
	}}
	uc := appprocurement.NewDeleteBookingScanUseCase(db, bookingRepo, &stubPurchaseOrderRepo{}, scanRepo, &eventrecordtest.Outbox{}, &stubScopedStock{})

	err := uc.Execute(context.Background(), "booking-1", "scan-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"scan-1"}, scanRepo.deleted)
}

func TestDeleteBookingScan_rejectsAScanNotBelongingToThisBooking(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo, _ := activeBookingFixtures()
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-9", BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionReceive, Quantity: 1},
	}}
	uc := appprocurement.NewDeleteBookingScanUseCase(db, bookingRepo, &stubPurchaseOrderRepo{}, scanRepo, &eventrecordtest.Outbox{}, &stubScopedStock{})

	err := uc.Execute(context.Background(), "booking-1", "scan-does-not-exist")

	require.Error(t, err)
	assert.ErrorIs(t, err, procurementDomain.ErrBookingScanNotFound)
	assert.Empty(t, scanRepo.deleted)
}

func TestDeleteBookingScan_rejectsWhenBookingIsNotInProgress(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: "po-1", Status: procurementDomain.BookingStatusCompleted}}
	scanRepo := &stubBookingScanRepo{byBooking: []procurementDomain.BookingScan{
		{ID: "scan-1", BookingID: "booking-1", SKUID: "sku-1", ActionType: procurementDomain.BookingScanActionDamaged, WriteOffQuantity: 1},
	}}
	uc := appprocurement.NewDeleteBookingScanUseCase(db, bookingRepo, &stubPurchaseOrderRepo{}, scanRepo, &eventrecordtest.Outbox{}, &stubScopedStock{})

	err := uc.Execute(context.Background(), "booking-1", "scan-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotActive)
	assert.Empty(t, scanRepo.deleted)
}

func TestDeleteBookingScan_propagatesBookingNotFound(t *testing.T) {
	db := newMockGormDBN(t, 2)
	bookingRepo := &stubBookingRepo{findErr: procurementDomain.ErrBookingNotFound}
	scanRepo := &stubBookingScanRepo{}
	uc := appprocurement.NewDeleteBookingScanUseCase(db, bookingRepo, &stubPurchaseOrderRepo{}, scanRepo, &eventrecordtest.Outbox{}, &stubScopedStock{})

	err := uc.Execute(context.Background(), "booking-1", "scan-1")

	require.Error(t, err)
	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotFound)
}
