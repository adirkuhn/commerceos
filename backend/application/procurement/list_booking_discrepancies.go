package procurement

import (
	"context"
	"errors"
	"fmt"
	"time"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// MissingBookingItemReport is one completed booking's under-scanned line
// item -- the same arithmetic missingBookingItems already does for a single
// booking's finish screen, aggregated here across every completed booking
// for the discrepancies report. Not persisted -- cheap to recompute from
// PurchaseOrderLineItem + BookingScan on demand, same reasoning as
// MissingBookingItem itself.
type MissingBookingItemReport struct {
	BookingID       string
	PurchaseOrderID string
	CompletedAt     *time.Time
	LineItemID      string
	SKUCode         *string
	ProductName     *string
	RawDescription  string
	ExpectedQty     int64
	ScannedQty      int64
	MissingQty      int64
}

// DamagedLostBookingItemReport is one completed booking's damaged/lost
// write-off scan.
type DamagedLostBookingItemReport struct {
	BookingID       string
	PurchaseOrderID string
	CompletedAt     *time.Time
	ScanID          string
	SKUID           string
	SKUCode         *string
	ProductName     *string
	ActionType      procurementDomain.BookingScanActionType
	Quantity        int64
	Reason          string
	ScannedAt       time.Time
	ScannedBy       string
}

type BookingDiscrepancyReport struct {
	MissingItems []MissingBookingItemReport
	DamagedLost  []DamagedLostBookingItemReport
}

// ListBookingDiscrepanciesUseCase is the tenant-wide "what went wrong
// across every finished booking" report: line items that never got fully
// scanned, and stock that arrived damaged or was lost. Derived on demand
// from every Completed booking's PurchaseOrderLineItems + BookingScans --
// same "not a persisted table" reasoning as MissingBookingItem.
type ListBookingDiscrepanciesUseCase struct {
	db           *gorm.DB
	bookingRepo  procurementDomain.BookingRepository
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	scanRepo     procurementDomain.BookingScanRepository
	skuRepo      pmsDomain.SKURepository
	variantRepo  pmsDomain.VariantRepository
	productRepo  pmsDomain.ProductRepository
}

func NewListBookingDiscrepanciesUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	scanRepo procurementDomain.BookingScanRepository,
	skuRepo pmsDomain.SKURepository,
	variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository,
) *ListBookingDiscrepanciesUseCase {
	return &ListBookingDiscrepanciesUseCase{
		db: db, bookingRepo: bookingRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
		skuRepo: skuRepo, variantRepo: variantRepo, productRepo: productRepo,
	}
}

func (uc *ListBookingDiscrepanciesUseCase) Execute(ctx context.Context) (*BookingDiscrepancyReport, error) {
	report := &BookingDiscrepancyReport{}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		bookings, err := uc.bookingRepo.ListCompleted(ctx)
		if err != nil {
			return err
		}

		for _, b := range bookings {
			lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, b.PurchaseOrderID)
			if err != nil {
				return err
			}
			scans, err := uc.scanRepo.ListByBooking(ctx, b.ID)
			if err != nil {
				return err
			}
			lineItemsByID := make(map[string]procurementDomain.PurchaseOrderLineItem, len(lineItems))
			for _, li := range lineItems {
				lineItemsByID[li.ID] = li
			}

			for _, m := range missingBookingItems(lineItems, scans) {
				li := lineItemsByID[m.LineItemID]
				skuCode, productName, err := uc.displayNames(ctx, li.SKUID)
				if err != nil {
					return err
				}
				report.MissingItems = append(report.MissingItems, MissingBookingItemReport{
					BookingID: b.ID, PurchaseOrderID: b.PurchaseOrderID, CompletedAt: b.CompletedAt,
					LineItemID: m.LineItemID, SKUCode: skuCode, ProductName: productName,
					RawDescription: m.RawDescription, ExpectedQty: m.ExpectedQty, ScannedQty: m.ScannedQty,
					MissingQty: m.ExpectedQty - m.ScannedQty,
				})
			}

			for _, s := range scans {
				if s.ActionType != procurementDomain.BookingScanActionDamaged && s.ActionType != procurementDomain.BookingScanActionLost {
					continue
				}
				skuCode, productName, err := uc.displayNames(ctx, &s.SKUID)
				if err != nil {
					return err
				}
				report.DamagedLost = append(report.DamagedLost, DamagedLostBookingItemReport{
					BookingID: b.ID, PurchaseOrderID: b.PurchaseOrderID, CompletedAt: b.CompletedAt,
					ScanID: s.ID, SKUID: s.SKUID, SKUCode: skuCode, ProductName: productName,
					ActionType: s.ActionType, Quantity: s.WriteOffQuantity, Reason: s.Reason,
					ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("listBookingDiscrepancies: %w", err)
	}
	return report, nil
}

// displayNames mirrors GetPurchaseOrderUseCase.attachProductDisplayNames --
// not found at any hop is tolerated (nil pair, row still renders by its raw
// text/ID) rather than failing the whole report over one stale reference.
func (uc *ListBookingDiscrepanciesUseCase) displayNames(ctx context.Context, skuID *string) (skuCode, productName *string, err error) {
	if skuID == nil {
		return nil, nil, nil
	}
	sku, err := uc.skuRepo.FindByID(ctx, *skuID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrVariantNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrProductNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return &sku.SKUCode, &product.Name, nil
}
