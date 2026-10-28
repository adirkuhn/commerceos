package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ScanBookingItemInput struct {
	BookingID string
	// Code is the raw scanned text -- tried first as a unit barcode
	// (SKU.Barcode), then as a case/box BarcodeAlias.Code, then as a
	// directly-typed SKU.SKUCode (same fallback order as
	// pms.ResolveBarcodeUseCase), then, last resort, against this
	// booking's own PO line items' RawSKU (see resolveViaLineItem).
	// Mutually exclusive with SKUID -- exactly one must be set.
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
	db                 *gorm.DB
	bookingRepo        procurementDomain.BookingRepository
	poRepo             procurementDomain.PurchaseOrderRepository
	scanRepo           procurementDomain.BookingScanRepository
	skuRepo            pmsDomain.SKURepository
	barcodeAliasRepo   pmsDomain.BarcodeAliasRepository
	scaleCodeAliasRepo pmsDomain.ScaleCodeAliasRepository
	lineItemRepo       procurementDomain.PurchaseOrderLineItemRepository
	outbox             eventrecord.Enqueuer
	stock              scopedStockReader
}

func NewScanBookingItemUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	poRepo procurementDomain.PurchaseOrderRepository,
	scanRepo procurementDomain.BookingScanRepository,
	skuRepo pmsDomain.SKURepository,
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository,
	scaleCodeAliasRepo pmsDomain.ScaleCodeAliasRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	outbox eventrecord.Enqueuer,
	stock scopedStockReader,
) *ScanBookingItemUseCase {
	return &ScanBookingItemUseCase{
		db: db, bookingRepo: bookingRepo, poRepo: poRepo, scanRepo: scanRepo,
		skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo, scaleCodeAliasRepo: scaleCodeAliasRepo,
		lineItemRepo: lineItemRepo, outbox: outbox, stock: stock,
	}
}

func (uc *ScanBookingItemUseCase) Execute(ctx context.Context, in ScanBookingItemInput) (*ScanBookingItemResult, error) {
	// A scanner/manual-entry code can carry stray whitespace -- trim before
	// the emptiness check below and every lookup that follows, same as
	// pms.ResolveBarcodeUseCase.
	in.Code = strings.TrimSpace(in.Code)
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
	var staged []procurementDomain.BookingScan
	var warehouseID string
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

		skuID := in.SKUID
		effectiveQuantity := in.Quantity
		// True whenever the quantity was entered against this booking's own
		// PO line item rather than resolved from a barcode/alias/catalog
		// code -- SKUID set directly (tapping a line item row) or the
		// RawSKU fallback below. Both mean the operator typed a number in
		// the same units as PurchaseOrderLineItem.Quantity itself (boxes/
		// purchased items), not stock units, so it must be scaled by that
		// line's UnitsPerPurchasedItem before it's comparable to anything
		// else this package writes as stock units (BookingScan.Quantity/
		// WriteOffQuantity, see missingBookingItems). A real barcode/alias/
		// SKU-code scan identifies a physical unit directly -- no PO-line
		// convention applies there, so those paths are left unscaled.
		scaleByLineItem := in.SKUID != ""
		if skuID == "" {
			sku, err := uc.skuRepo.FindByBarcode(ctx, in.Code)
			if err == nil {
				skuID = sku.ID
			} else if errors.Is(err, pmsDomain.ErrSKUNotFound) {
				alias, aerr := uc.barcodeAliasRepo.FindByCode(ctx, in.Code)
				if aerr == nil {
					skuID = alias.SKUID
					effectiveQuantity = in.Quantity * alias.UnitsPerScan
				} else if errors.Is(aerr, pmsDomain.ErrBarcodeAliasNotFound) {
					weightSKUID, weightErr := uc.resolveWeightEmbeddedBarcode(ctx, in.Code)
					if weightErr != nil {
						// ErrNotWeightEmbeddedBarcode (wrong shape) and
						// ErrSKUNotFound (right shape, but no SKU/alias
						// registered under that scale code) both fall
						// through to the weaker text-match fallbacks below
						// -- same as pms.ResolveBarcodeUseCase's own
						// ordering. Anything else (a corrupted checksum) is
						// a mis-scan, not "keep trying".
						if !errors.Is(weightErr, pmsDomain.ErrNotWeightEmbeddedBarcode) && !errors.Is(weightErr, pmsDomain.ErrSKUNotFound) {
							return weightErr
						}
						// Same "type a SKU code" fallback pms.ResolveBarcodeUseCase
						// already has -- Booking's scan resolution was missing it,
						// even though the input's still called Code (not
						// BarcodeOrSKUCode) and every scanner prompt promises it.
						byCode, cerr := uc.skuRepo.FindByCode(ctx, in.Code)
						if cerr == nil {
							if byCode.PricingMode == pmsDomain.SKUPricingModePerWeight {
								return pmsDomain.ErrWeightRequiredForPerWeightSKU
							}
							skuID = byCode.ID
						} else if errors.Is(cerr, pmsDomain.ErrSKUNotFound) {
							liSKUID, lerr := uc.resolveViaLineItem(ctx, po.ID, in.Code)
							if lerr != nil {
								return lerr
							}
							liSKU, ferr := uc.skuRepo.FindByID(ctx, liSKUID)
							if ferr != nil {
								return ferr
							}
							if liSKU.PricingMode == pmsDomain.SKUPricingModePerWeight {
								return pmsDomain.ErrWeightRequiredForPerWeightSKU
							}
							skuID = liSKUID
							scaleByLineItem = true
						} else {
							return cerr
						}
					} else {
						skuID = weightSKUID
					}
				} else {
					return aerr
				}
			} else {
				return err
			}
		}

		if scaleByLineItem {
			li, lerr := uc.findLineItemBySKU(ctx, po.ID, skuID)
			if lerr != nil {
				return lerr
			}
			if li != nil {
				effectiveQuantity *= li.EffectiveUnitsPerPurchasedItem()
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

		staged, err = uc.scanRepo.ListByBooking(ctx, in.BookingID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("scanBookingItem: %w", err)
	}

	// A row that takes stock is checked now, while the scanner can still fix
	// it, instead of failing Finish halfway through the replay.
	if scan.StockDelta() < 0 {
		if err := checkBookingScanSource(ctx, uc.stock, warehouseID, append(staged, scan), scan); err != nil {
			return nil, fmt.Errorf("scanBookingItem: %w", err)
		}
	}

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.scanRepo.Create(ctx, &scan); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.BookingScanCreated, scan.ID, eventrecord.Data{
			"booking_id": scan.BookingID, "sku_id": scan.SKUID, "action_type": string(scan.ActionType),
			"quantity": scan.Quantity, "write_off_quantity": scan.WriteOffQuantity,
			"lot_number": scan.LotNumber, "expiry_date": scan.ExpiryDate,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("scanBookingItem: %w", err)
	}

	return &ScanBookingItemResult{Scan: scan}, nil
}

// resolveWeightEmbeddedBarcode tries code as a GS1 "prefix 2x"
// variable-weight EAN-13 barcode (butcher/deli scale label) -- the
// primary SKU.ScaleCode first, then a ScaleCodeAlias for a tenant with
// more than one physical scale. Returns ErrNotWeightEmbeddedBarcode
// unchanged when code isn't shaped like one at all (the caller's signal
// to keep trying the SKU-code/line-item fallbacks), or
// ErrInvalidCheckDigit for a mis-scan (right shape, bad checksum).
func (uc *ScanBookingItemUseCase) resolveWeightEmbeddedBarcode(ctx context.Context, code string) (string, error) {
	decoded, err := pmsDomain.DecodeWeightEmbeddedBarcode(code)
	if err != nil {
		return "", err
	}
	sku, err := uc.skuRepo.FindByScaleCode(ctx, decoded.ScaleCode)
	if err == nil {
		return sku.ID, nil
	}
	if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
		return "", err
	}
	alias, aerr := uc.scaleCodeAliasRepo.FindByCode(ctx, decoded.ScaleCode)
	if aerr != nil {
		if errors.Is(aerr, pmsDomain.ErrScaleCodeAliasNotFound) {
			return "", pmsDomain.ErrSKUNotFound
		}
		return "", aerr
	}
	return alias.SKUID, nil
}

// resolveViaLineItem is the last-resort fallback: when code doesn't match
// anything in the tenant-wide catalog (barcode, alias, SKU code), try
// matching it against this booking's own PO line items' RawSKU -- the
// identifier exactly as printed on the supplier's document, which is often
// what a physical barcode/label actually carries even when it was never
// entered as the linked SKU's own Barcode or SKUCode. Only usable once that
// line item has been linked to a real SKU (RawSKU alone isn't enough to
// book against -- there's no product to receive into yet).
func (uc *ScanBookingItemUseCase) resolveViaLineItem(ctx context.Context, purchaseOrderID, code string) (string, error) {
	lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, purchaseOrderID)
	if err != nil {
		return "", err
	}
	for _, li := range lineItems {
		if li.SKUID != nil && strings.EqualFold(strings.TrimSpace(li.RawSKU), code) {
			return *li.SKUID, nil
		}
	}
	return "", pmsDomain.ErrSKUNotFound
}

// findLineItemBySKU looks up this PO's line item for an already-resolved
// SKU, for scaleByLineItem's UnitsPerPurchasedItem lookup above. Returns
// (nil, nil), not an error, when the SKU doesn't match any line item on
// this PO (e.g. a substitute product received instead of what was
// ordered) -- there's no box convention to scale by, so the caller leaves
// the quantity as-is.
func (uc *ScanBookingItemUseCase) findLineItemBySKU(ctx context.Context, purchaseOrderID, skuID string) (*procurementDomain.PurchaseOrderLineItem, error) {
	lineItems, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, purchaseOrderID)
	if err != nil {
		return nil, err
	}
	for i := range lineItems {
		if lineItems[i].SKUID != nil && *lineItems[i].SKUID == skuID {
			return &lineItems[i], nil
		}
	}
	return nil, nil
}
