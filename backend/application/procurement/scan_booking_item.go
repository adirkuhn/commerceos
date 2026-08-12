package procurement

import (
	"context"
	"errors"
	"fmt"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ScanBookingItemInput struct {
	BookingID string
	// Code is the raw scanned text -- tried first as a unit barcode
	// (SKU.Barcode), then as a case/box BarcodeAlias.Code. Mutually
	// exclusive with SKUID -- exactly one must be set.
	Code string
	// SKUID lets a caller book directly against an already-known SKU (e.g.
	// tapping a booking line item row on the scanner screen) without a
	// barcode -- skips the Code resolution path entirely when set.
	SKUID string
	// ActionType defaults to BookingScanActionReceive when empty (keeps the
	// original scan-a-barcode-to-receive call shape working unchanged).
	ActionType procurementDomain.BookingScanActionType
	// Quantity is always a positive magnitude here -- ActionType carries
	// the direction (see BookingScan.Quantity/WriteOffQuantity doc).
	Quantity   int64
	BatchID    *string
	LotNumber  string
	ExpiryDate *time.Time
	Reason     string
	ScannedBy  string
}

type ScanBookingItemResult struct {
	Scan procurementDomain.BookingScan
}

// ScanBookingItemUseCase is the scanner app's "scan a barcode, pick/create a
// batch, enter quantity" action -- and its Remove/Damaged/Lost siblings.
// Resolves the code to a SKU (and unit multiplier, for a case/box alias)
// and writes ONE staged BookingScan row. That's it -- this use case never
// touches real inventory, never creates a batch, and never publishes a
// stock-changed event. A booking in progress is not yet trustworthy (it can
// be corrected, paused, re-scanned), so nothing should reach real
// inventory -- or, downstream, sync to an external sales channel -- until
// it's done. See CompleteBookingUseCase, which replays every staged row for
// real when the booking finishes.
type ScanBookingItemUseCase struct {
	db               *gorm.DB
	bookingRepo      procurementDomain.BookingRepository
	poRepo           procurementDomain.PurchaseOrderRepository
	scanRepo         procurementDomain.BookingScanRepository
	skuRepo          pmsDomain.SKURepository
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository
}

func NewScanBookingItemUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	poRepo procurementDomain.PurchaseOrderRepository,
	scanRepo procurementDomain.BookingScanRepository,
	skuRepo pmsDomain.SKURepository,
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository,
) *ScanBookingItemUseCase {
	return &ScanBookingItemUseCase{
		db: db, bookingRepo: bookingRepo, poRepo: poRepo, scanRepo: scanRepo,
		skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo,
	}
}

func (uc *ScanBookingItemUseCase) Execute(ctx context.Context, in ScanBookingItemInput) (*ScanBookingItemResult, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("scanBookingItem: %w", invDomain.ErrInvalidQuantity)
	}
	if in.Code == "" && in.SKUID == "" {
		return nil, fmt.Errorf("scanBookingItem: %w", procurementDomain.ErrScanCodeOrSKURequired)
	}
	actionType := in.ActionType
	if actionType == "" {
		actionType = procurementDomain.BookingScanActionReceive
	}
	switch actionType {
	case procurementDomain.BookingScanActionReceive, procurementDomain.BookingScanActionCorrection,
		procurementDomain.BookingScanActionDamaged, procurementDomain.BookingScanActionLost:
	default:
		return nil, fmt.Errorf("scanBookingItem: %w", procurementDomain.ErrInvalidBookingScanActionType)
	}

	var scan procurementDomain.BookingScan
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		booking, err := uc.bookingRepo.FindByID(ctx, in.BookingID)
		if err != nil {
			return err
		}
		if booking.Status != procurementDomain.BookingStatusInProgress {
			return procurementDomain.ErrBookingNotActive
		}

		po, err := uc.poRepo.FindByID(ctx, booking.PurchaseOrderID)
		if err != nil {
			return err
		}
		// A booking can only exist for a PO that already has a warehouse --
		// StartOrResumeBookingUseCase assigns one before ever creating the
		// booking row -- so this is defensive, not a real runtime path.
		if po.WarehouseID == nil {
			return procurementDomain.ErrWarehouseRequired
		}

		skuID := in.SKUID
		effectiveQuantity := in.Quantity
		if skuID == "" {
			sku, err := uc.skuRepo.FindByBarcode(ctx, in.Code)
			if err == nil {
				skuID = sku.ID
			} else if errors.Is(err, pmsDomain.ErrSKUNotFound) {
				alias, aerr := uc.barcodeAliasRepo.FindByCode(ctx, in.Code)
				if aerr != nil {
					if errors.Is(aerr, pmsDomain.ErrBarcodeAliasNotFound) {
						return pmsDomain.ErrSKUNotFound
					}
					return aerr
				}
				skuID = alias.SKUID
				effectiveQuantity = in.Quantity * alias.UnitsPerScan
			} else {
				return err
			}
		}

		scan = procurementDomain.BookingScan{
			TenantID: tenant.IDFromContext(ctx), BookingID: in.BookingID, SKUID: skuID, BatchID: in.BatchID,
			LotNumber: in.LotNumber, ExpiryDate: in.ExpiryDate, ActionType: actionType, Reason: in.Reason,
			ScannedAt: time.Now(), ScannedBy: in.ScannedBy,
		}
		switch actionType {
		case procurementDomain.BookingScanActionReceive:
			scan.Quantity = effectiveQuantity
		case procurementDomain.BookingScanActionCorrection:
			scan.Quantity = -effectiveQuantity
		case procurementDomain.BookingScanActionDamaged, procurementDomain.BookingScanActionLost:
			scan.WriteOffQuantity = effectiveQuantity
		}

		return uc.scanRepo.Create(ctx, &scan)
	})
	if err != nil {
		return nil, fmt.Errorf("scanBookingItem: %w", err)
	}

	return &ScanBookingItemResult{Scan: scan}, nil
}
