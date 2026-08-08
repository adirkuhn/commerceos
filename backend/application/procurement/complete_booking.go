package procurement

import (
	"context"
	"fmt"
	"time"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// MissingBookingItem is one PO line item that wasn't fully scanned by the
// time the booking was finished -- the plan's "missing-items report" for a
// manager to review, not a persisted table: it's cheap to recompute from
// LineItems + Scans on demand, same reasoning as BookingDetail.
type MissingBookingItem struct {
	LineItemID     string
	RawDescription string
	ExpectedQty    int64
	ScannedQty     int64
}

type CompleteBookingResult struct {
	Booking      procurementDomain.Booking
	MissingItems []MissingBookingItem
}

// CompleteBookingUseCase is the scanner app's "Finish" action. Finishing
// with every line fully scanned is a clean complete (MissingItems is
// empty); finishing with gaps still completes the booking but returns the
// gaps for a manager to review -- per the plan, that report is surfaced,
// not auto-sent anywhere, so completion always succeeds regardless of gaps.
type CompleteBookingUseCase struct {
	db           *gorm.DB
	bookingRepo  procurementDomain.BookingRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	scanRepo     procurementDomain.BookingScanRepository
}

func NewCompleteBookingUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	scanRepo procurementDomain.BookingScanRepository,
) *CompleteBookingUseCase {
	return &CompleteBookingUseCase{db: db, bookingRepo: bookingRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo}
}

func (uc *CompleteBookingUseCase) Execute(ctx context.Context, id string, completedBy string) (*CompleteBookingResult, error) {
	var result CompleteBookingResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		booking, err := uc.bookingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if booking.Status == procurementDomain.BookingStatusCompleted {
			return procurementDomain.ErrBookingNotActive
		}

		lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, booking.PurchaseOrderID)
		if err != nil {
			return err
		}
		scans, err := uc.scanRepo.ListByBooking(ctx, id)
		if err != nil {
			return err
		}

		scannedBySKU := make(map[string]int64, len(scans))
		for _, s := range scans {
			scannedBySKU[s.SKUID] += s.Quantity
		}

		for _, li := range lineItems {
			var scannedQty int64
			if li.SKUID != nil {
				scannedQty = scannedBySKU[*li.SKUID]
			}
			// BookingScan.Quantity is already in stock-unit terms (a
			// case-barcode scan credits UnitsPerScan units at once, see
			// ScanBookingItemUseCase) -- li.Quantity is purchased-item
			// terms, so it must be scaled by UnitsPerPurchasedItem before
			// comparing the two (see PurchaseOrderLineItem's doc comment).
			expectedQty := li.Quantity * li.EffectiveUnitsPerPurchasedItem()
			if scannedQty < expectedQty {
				result.MissingItems = append(result.MissingItems, MissingBookingItem{
					LineItemID: li.ID, RawDescription: li.RawDescription, ExpectedQty: expectedQty, ScannedQty: scannedQty,
				})
			}
		}

		now := time.Now()
		booking.Status = procurementDomain.BookingStatusCompleted
		booking.CompletedAt = &now
		booking.CompletedBy = completedBy
		if err := uc.bookingRepo.Update(ctx, booking); err != nil {
			return err
		}
		result.Booking = *booking
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("completeBooking: %w", err)
	}
	return &result, nil
}
