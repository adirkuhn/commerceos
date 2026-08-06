package procurement

import (
	"context"
	"fmt"
	"time"

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
}

func NewPauseBookingUseCase(db *gorm.DB, bookingRepo procurementDomain.BookingRepository) *PauseBookingUseCase {
	return &PauseBookingUseCase{db: db, bookingRepo: bookingRepo}
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
		return uc.bookingRepo.Update(ctx, booking)
	})
	if err != nil {
		return nil, fmt.Errorf("pauseBooking: %w", err)
	}
	return booking, nil
}
