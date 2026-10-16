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

		// Shortfalls are per PO, not per booking: a PO delivered in parts
		// has one booking per delivery, and a later one makes up for an
		// earlier short one. ListCompleted is newest first, so a PO's
		// first booking seen is its latest -- the one a shortfall is
		// listed under.
		var poOrder []string
		bookingsByPO := map[string][]procurementDomain.Booking{}
		for _, b := range bookings {
			if _, seen := bookingsByPO[b.PurchaseOrderID]; !seen {
				poOrder = append(poOrder, b.PurchaseOrderID)
			}
			bookingsByPO[b.PurchaseOrderID] = append(bookingsByPO[b.PurchaseOrderID], b)
		}

		for _, poID := range poOrder {
			poBookings := bookingsByPO[poID]
			latest := poBookings[0]
			lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, poID)
			if err != nil {
				return err
			}
			lineItemsByID := make(map[string]procurementDomain.PurchaseOrderLineItem, len(lineItems))
			for _, li := range lineItems {
				lineItemsByID[li.ID] = li
			}

			var poScans []procurementDomain.BookingScan
			for _, b := range poBookings {
				scans, err := uc.scanRepo.ListByBooking(ctx, b.ID)
				if err != nil {
					return err
				}
				poScans = append(poScans, scans...)
				if err := uc.appendDamagedLost(ctx, report, b, scans); err != nil {
					return err
				}
			}

			for _, m := range missingBookingItems(lineItems, poScans) {
				li := lineItemsByID[m.LineItemID]
				skuCode, productName, err := uc.displayNames(ctx, li.SKUID)
				if err != nil {
					return err
				}
				report.MissingItems = append(report.MissingItems, MissingBookingItemReport{
					BookingID: latest.ID, PurchaseOrderID: poID, CompletedAt: latest.CompletedAt,
					LineItemID: m.LineItemID, SKUCode: skuCode, ProductName: productName,
					RawDescription: m.RawDescription, ExpectedQty: m.ExpectedQty, ScannedQty: m.ScannedQty,
					MissingQty: m.ExpectedQty - m.ScannedQty,
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

func (uc *ListBookingDiscrepanciesUseCase) appendDamagedLost(ctx context.Context, report *BookingDiscrepancyReport, b procurementDomain.Booking, scans []procurementDomain.BookingScan) error {
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
	return nil
}

func (uc *ListBookingDiscrepanciesUseCase) displayNames(ctx context.Context, skuID *string) (skuCode, productName *string, err error) {
	return skuDisplayNames(ctx, uc.skuRepo, uc.variantRepo, uc.productRepo, skuID)
}

// skuDisplayNames mirrors GetPurchaseOrderUseCase.attachProductDisplayNames --
// not found at any hop is tolerated (nil pair, row still renders by its raw
// text/ID) rather than failing a whole report over one stale reference.
func skuDisplayNames(ctx context.Context, skuRepo pmsDomain.SKURepository, variantRepo pmsDomain.VariantRepository,
	productRepo pmsDomain.ProductRepository, skuID *string) (skuCode, productName *string, err error) {
	if skuID == nil {
		return nil, nil, nil
	}
	sku, err := skuRepo.FindByID(ctx, *skuID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	variant, err := variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrVariantNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	product, err := productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		if errors.Is(err, pmsDomain.ErrProductNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return &sku.SKUCode, &product.Name, nil
}
