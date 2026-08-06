package procurement

import (
	"context"
	"fmt"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type StartOrResumeBookingInput struct {
	PurchaseOrderID string
	// WarehouseID is only consulted when the PO has none assigned yet --
	// the first booking session for a PO assigns it (see
	// PurchaseOrder.WarehouseID's doc); every later start/resume for the
	// same PO ignores this field. Required in that first case: returns
	// ErrWarehouseRequired if both are unset.
	WarehouseID *string
}

// StartOrResumeBookingUseCase reuses the PO's existing in_progress/paused
// booking rather than creating a duplicate -- two staff scanning the same
// PO, or one pausing and resuming later, share the same running total. A
// paused booking is flipped back to in_progress as part of "resuming" it.
type StartOrResumeBookingUseCase struct {
	db          *gorm.DB
	poRepo      procurementDomain.PurchaseOrderRepository
	bookingRepo procurementDomain.BookingRepository
}

func NewStartOrResumeBookingUseCase(db *gorm.DB, poRepo procurementDomain.PurchaseOrderRepository, bookingRepo procurementDomain.BookingRepository) *StartOrResumeBookingUseCase {
	return &StartOrResumeBookingUseCase{db: db, poRepo: poRepo, bookingRepo: bookingRepo}
}

func (uc *StartOrResumeBookingUseCase) Execute(ctx context.Context, in StartOrResumeBookingInput) (*procurementDomain.Booking, error) {
	var booking procurementDomain.Booking
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		po, err := uc.poRepo.FindByID(ctx, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		if po.WarehouseID == nil {
			if in.WarehouseID == nil {
				return procurementDomain.ErrWarehouseRequired
			}
			po.WarehouseID = in.WarehouseID
			po.UpdatedAt = time.Now()
			if err := uc.poRepo.Update(ctx, po); err != nil {
				return err
			}
		}

		existing, err := uc.bookingRepo.FindActiveByPurchaseOrder(ctx, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.Status == procurementDomain.BookingStatusPaused {
				existing.Status = procurementDomain.BookingStatusInProgress
				existing.PausedAt = nil
				if err := uc.bookingRepo.Update(ctx, existing); err != nil {
					return err
				}
			}
			booking = *existing
			return nil
		}

		booking = procurementDomain.Booking{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID,
			Status: procurementDomain.BookingStatusInProgress, StartedAt: time.Now(),
		}
		return uc.bookingRepo.Create(ctx, &booking)
	})
	if err != nil {
		return nil, fmt.Errorf("startOrResumeBooking: %w", err)
	}
	return &booking, nil
}
