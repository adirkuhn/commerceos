package procurement

import (
	"context"
	"fmt"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// BookingDetail is the raw data the scanner app's progress view needs --
// expected-vs-scanned per line is arithmetic (sum Scans by SKUID against
// LineItems' Quantity), deliberately left to the caller instead of
// precomputed here, same "push simple aggregation to the edge" choice made
// for PurchaseOrderDetail.
type BookingDetail struct {
	Booking   procurementDomain.Booking
	LineItems []procurementDomain.PurchaseOrderLineItem
	Scans     []procurementDomain.BookingScan
}

type GetBookingUseCase struct {
	db           *gorm.DB
	bookingRepo  procurementDomain.BookingRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	scanRepo     procurementDomain.BookingScanRepository
}

func NewGetBookingUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	scanRepo procurementDomain.BookingScanRepository,
) *GetBookingUseCase {
	return &GetBookingUseCase{db: db, bookingRepo: bookingRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo}
}

func (uc *GetBookingUseCase) Execute(ctx context.Context, id string) (*BookingDetail, error) {
	var detail BookingDetail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		booking, err := uc.bookingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		detail.Booking = *booking

		lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, booking.PurchaseOrderID)
		if err != nil {
			return err
		}
		detail.LineItems = lineItems

		scans, err := uc.scanRepo.ListByBooking(ctx, id)
		if err != nil {
			return err
		}
		detail.Scans = scans
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getBooking: %w", err)
	}
	return &detail, nil
}
