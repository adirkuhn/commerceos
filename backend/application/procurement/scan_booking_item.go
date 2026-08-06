package procurement

import (
	"context"
	"errors"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// Narrow interface at point of use (ISP) so this use case can be unit
// tested without a real DB -- *inventory.ReceiveStockUseCase satisfies it
// structurally, no adapter needed in production wiring.
type receiveStockForBookingUC interface {
	Execute(ctx context.Context, in invapp.ReceiveStockInput) (*invapp.ReceiveStockResult, error)
}

type ScanBookingItemInput struct {
	BookingID string
	// Code is the raw scanned text -- tried first as a unit barcode
	// (SKU.Barcode), then as a case/box BarcodeAlias.Code.
	Code       string
	Quantity   int64
	BatchID    *string
	LotNumber  string
	ExpiryDate *time.Time
	ScannedBy  string
}

type ScanBookingItemResult struct {
	Receipt *invapp.ReceiveStockResult
	Scan    procurementDomain.BookingScan
}

// ScanBookingItemUseCase is the scanner app's "scan a barcode, pick/create a
// batch, enter quantity" action: resolve the code to a SKU (and unit
// multiplier, for a case/box alias), book the stock via
// inventory.ReceiveStockUseCase, then record the audit-trail row. See
// domain/procurement/booking.go's BookingScan doc for why this is two
// transactions, not one.
type ScanBookingItemUseCase struct {
	db               *gorm.DB
	bookingRepo      procurementDomain.BookingRepository
	poRepo           procurementDomain.PurchaseOrderRepository
	scanRepo         procurementDomain.BookingScanRepository
	skuRepo          pmsDomain.SKURepository
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository
	receiveStock     receiveStockForBookingUC
}

func NewScanBookingItemUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	poRepo procurementDomain.PurchaseOrderRepository,
	scanRepo procurementDomain.BookingScanRepository,
	skuRepo pmsDomain.SKURepository,
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository,
	receiveStock receiveStockForBookingUC,
) *ScanBookingItemUseCase {
	return &ScanBookingItemUseCase{
		db: db, bookingRepo: bookingRepo, poRepo: poRepo, scanRepo: scanRepo,
		skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo, receiveStock: receiveStock,
	}
}

func (uc *ScanBookingItemUseCase) Execute(ctx context.Context, in ScanBookingItemInput) (*ScanBookingItemResult, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("scanBookingItem: %w", invDomain.ErrInvalidQuantity)
	}

	var skuID, warehouseID string
	var effectiveQuantity int64
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
		warehouseID = *po.WarehouseID

		sku, err := uc.skuRepo.FindByBarcode(ctx, in.Code)
		if err == nil {
			skuID = sku.ID
			effectiveQuantity = in.Quantity
			return nil
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		alias, err := uc.barcodeAliasRepo.FindByCode(ctx, in.Code)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrBarcodeAliasNotFound) {
				return pmsDomain.ErrSKUNotFound
			}
			return err
		}
		skuID = alias.SKUID
		effectiveQuantity = in.Quantity * alias.UnitsPerScan
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanBookingItem: %w", err)
	}

	receipt, err := uc.receiveStock.Execute(ctx, invapp.ReceiveStockInput{
		SKUID: skuID, WarehouseID: warehouseID, Quantity: effectiveQuantity, BatchID: in.BatchID,
		LotNumber: in.LotNumber, ExpiryDate: in.ExpiryDate, Reason: fmt.Sprintf("scanner booking %s", in.BookingID),
	})
	if err != nil {
		return nil, fmt.Errorf("scanBookingItem: %w", err)
	}

	batchID := in.BatchID
	if receipt.Batch != nil {
		batchID = &receipt.Batch.ID
	}

	var scan procurementDomain.BookingScan
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		scan = procurementDomain.BookingScan{
			TenantID: tenant.IDFromContext(ctx), BookingID: in.BookingID, SKUID: skuID, BatchID: batchID,
			Quantity: effectiveQuantity, ScannedAt: time.Now(), ScannedBy: in.ScannedBy,
		}
		return uc.scanRepo.Create(ctx, &scan)
	})
	if err != nil {
		return nil, fmt.Errorf("scanBookingItem: %w", err)
	}

	return &ScanBookingItemResult{Receipt: receipt, Scan: scan}, nil
}
