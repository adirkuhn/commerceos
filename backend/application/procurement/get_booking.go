package procurement

import (
	"context"
	"fmt"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
	skuRepo      pmsDomain.SKURepository
	variantRepo  pmsDomain.VariantRepository
	productRepo  pmsDomain.ProductRepository
}

func NewGetBookingUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	scanRepo procurementDomain.BookingScanRepository,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
) *GetBookingUseCase {
	return &GetBookingUseCase{
		db: db, bookingRepo: bookingRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
		skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo,
	}
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
		// Charge-type rows (shipping/discount/fee/credit_note) have no SKU
		// and can never be scanned -- exclude them from the scanner's
		// booking view entirely, same reasoning as
		// CompleteBookingUseCase's missing-items report.
		productLineItems := make([]procurementDomain.PurchaseOrderLineItem, 0, len(lineItems))
		for _, li := range lineItems {
			if li.LineType.IsProduct() {
				productLineItems = append(productLineItems, li)
			}
		}
		if err := AttachProductDisplayNames(ctx, uc.skuRepo, uc.variantRepo, uc.productRepo, productLineItems); err != nil {
			return err
		}
		detail.LineItems = productLineItems

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
