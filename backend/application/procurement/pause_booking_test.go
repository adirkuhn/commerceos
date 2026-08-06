package procurement_test

import (
	"context"
	"testing"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPauseBooking_pausesInProgressBooking(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusInProgress}}
	uc := appprocurement.NewPauseBookingUseCase(db, bookingRepo)

	booking, err := uc.Execute(context.Background(), "booking-1")

	require.NoError(t, err)
	require.Len(t, bookingRepo.updated, 1)
	assert.Equal(t, procurementDomain.BookingStatusPaused, bookingRepo.updated[0].Status)
	assert.NotNil(t, bookingRepo.updated[0].PausedAt)
	assert.Equal(t, procurementDomain.BookingStatusPaused, booking.Status)
}

func TestPauseBooking_rejectsWhenNotInProgress(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findResult: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusCompleted}}
	uc := appprocurement.NewPauseBookingUseCase(db, bookingRepo)

	_, err := uc.Execute(context.Background(), "booking-1")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotActive)
	assert.Empty(t, bookingRepo.updated)
}

func TestPauseBooking_propagatesNotFound(t *testing.T) {
	db := newMockGormDB(t)
	bookingRepo := &stubBookingRepo{findErr: procurementDomain.ErrBookingNotFound}
	uc := appprocurement.NewPauseBookingUseCase(db, bookingRepo)

	_, err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, procurementDomain.ErrBookingNotFound)
}
