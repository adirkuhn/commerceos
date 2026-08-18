package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// DeleteBookingScanUseCase removes a staged BookingScan row before it's
// ever reached real inventory -- the scanner app's fix for a mis-scan
// (wrong action type, wrong quantity, a damaged/lost entry logged in
// error) that the existing "correction" action type can't undo, since
// correction only nets out received quantity and never touches
// write_off_quantity. Only legal while the booking is still in_progress:
// once Finish runs, every row is either real inventory already or gone,
// so there's nothing left here to undo (see CompleteBookingUseCase's own
// doc comment -- replay is the only place a staged row becomes real).
type DeleteBookingScanUseCase struct {
	db          *gorm.DB
	bookingRepo procurementDomain.BookingRepository
	scanRepo    procurementDomain.BookingScanRepository
}

func NewDeleteBookingScanUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	scanRepo procurementDomain.BookingScanRepository,
) *DeleteBookingScanUseCase {
	return &DeleteBookingScanUseCase{db: db, bookingRepo: bookingRepo, scanRepo: scanRepo}
}

func (uc *DeleteBookingScanUseCase) Execute(ctx context.Context, bookingID, scanID string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		booking, err := uc.bookingRepo.FindByID(ctx, bookingID)
		if err != nil {
			return err
		}
		if booking.Status != procurementDomain.BookingStatusInProgress {
			return procurementDomain.ErrBookingNotActive
		}

		scans, err := uc.scanRepo.ListByBooking(ctx, bookingID)
		if err != nil {
			return err
		}
		found := false
		for _, s := range scans {
			if s.ID == scanID {
				found = true
				break
			}
		}
		if !found {
			return procurementDomain.ErrBookingScanNotFound
		}

		return uc.scanRepo.Delete(ctx, scanID)
	})
	if err != nil {
		return fmt.Errorf("deleteBookingScan: %w", err)
	}
	return nil
}
