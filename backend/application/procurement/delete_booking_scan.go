package procurement

import (
	"context"
	"errors"
	"fmt"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
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
	poRepo      procurementDomain.PurchaseOrderRepository
	scanRepo    procurementDomain.BookingScanRepository
	outbox      eventrecord.Enqueuer
	stock       scopedStockReader
}

func NewDeleteBookingScanUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	poRepo procurementDomain.PurchaseOrderRepository,
	scanRepo procurementDomain.BookingScanRepository,
	outbox eventrecord.Enqueuer,
	stock scopedStockReader,
) *DeleteBookingScanUseCase {
	return &DeleteBookingScanUseCase{db: db, bookingRepo: bookingRepo, poRepo: poRepo, scanRepo: scanRepo, outbox: outbox, stock: stock}
}

func (uc *DeleteBookingScanUseCase) Execute(ctx context.Context, bookingID, scanID string) error {
	var found procurementDomain.BookingScan
	var remaining []procurementDomain.BookingScan
	var warehouseID string
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
		ok := false
		for _, s := range scans {
			if s.ID == scanID {
				found, ok = s, true
			} else {
				remaining = append(remaining, s)
			}
		}
		if !ok {
			return procurementDomain.ErrBookingScanNotFound
		}
		if found.StockDelta() <= 0 {
			return nil
		}
		po, err := uc.poRepo.FindByID(ctx, booking.PurchaseOrderID)
		if err != nil {
			return err
		}
		if po.WarehouseID == nil {
			return procurementDomain.ErrWarehouseRequired
		}
		warehouseID = *po.WarehouseID
		return nil
	})
	if err != nil {
		return fmt.Errorf("deleteBookingScan: %w", err)
	}

	// Deleting a receive must not strand a correction or damaged/lost row
	// that takes from what it puts in -- Finish could never replay it.
	if found.StockDelta() > 0 {
		if err := uc.requireNoDependents(ctx, warehouseID, remaining, found); err != nil {
			return fmt.Errorf("deleteBookingScan: %w", err)
		}
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.scanRepo.Delete(ctx, scanID); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.BookingScanDeleted, scanID, eventrecord.Data{
			"booking_id": bookingID, "sku_id": found.SKUID, "action_type": string(found.ActionType),
		})
	})
	if err != nil {
		return fmt.Errorf("deleteBookingScan: %w", err)
	}
	return nil
}

func (uc *DeleteBookingScanUseCase) requireNoDependents(ctx context.Context, warehouseID string, remaining []procurementDomain.BookingScan, deleted procurementDomain.BookingScan) error {
	err := checkBookingScanSource(ctx, uc.stock, warehouseID, remaining, deleted)
	if errors.Is(err, procurementDomain.ErrPendingBatchNotFound) || errors.Is(err, procurementDomain.ErrPendingLotInsufficient) ||
		errors.Is(err, invDomain.ErrInsufficientStock) {
		return procurementDomain.ErrBookingScanHasDependents
	}
	return err
}
