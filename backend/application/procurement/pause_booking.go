package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PauseBookingUseCase is the scanner app's "Pause" action -- "Continue"
// doesn't need its own use case, since re-calling
// StartOrResumeBookingUseCase for the same PO already flips a paused
// booking back to in_progress and hands the same booking back.
type PauseBookingUseCase struct {
	db          *gorm.DB
	bookingRepo procurementDomain.BookingRepository
	outbox      eventrecord.Enqueuer
}

func NewPauseBookingUseCase(db *gorm.DB, bookingRepo procurementDomain.BookingRepository, outbox eventrecord.Enqueuer) *PauseBookingUseCase {
	return &PauseBookingUseCase{db: db, bookingRepo: bookingRepo, outbox: outbox}
}

func (uc *PauseBookingUseCase) Execute(ctx context.Context, id string) (*procurementDomain.Booking, error) {
	var booking *procurementDomain.Booking
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		booking, err = uc.bookingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if booking.Status != procurementDomain.BookingStatusInProgress {
			return procurementDomain.ErrBookingNotActive
		}
		now := time.Now()
		booking.Status = procurementDomain.BookingStatusPaused
		booking.PausedAt = &now
		if err := uc.bookingRepo.Update(ctx, booking); err != nil {
			return err
		}
		return recordBooking(ctx, uc.outbox, events.BookingPaused, *booking)
	})
	if err != nil {
		return nil, fmt.Errorf("pauseBooking: %w", err)
	}
	return booking, nil
}
