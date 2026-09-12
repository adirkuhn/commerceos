package procurement

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ResolveBookingScanCodeResult struct {
	SKUID       string
	ProductName string
	// UnitsPerScan is 1 for a direct unit-barcode/SKU-code/line-item match,
	// or the case/box multiplier when resolved via a BarcodeAlias -- the
	// caller (BookingScreen.tsx) multiplies the scanned quantity by this
	// before defaulting the confirm screen's quantity field, since the
	// eventual ScanBookingItemInput sent on Confirm carries sku_id (not
	// code) and so skips ScanBookingItemUseCase's own alias-multiplier step.
	UnitsPerScan int64
	// WeightGrams is non-nil only when the scanned code was a
	// weight-embedded barcode (see pmsDomain.DecodeWeightEmbeddedBarcode)
	// -- the actual weight of the specific package scanned.
	WeightGrams *int64
}

// variantFinderForBookingScanResolve/productFinderForBookingScanResolve
// mirror pms.ResolveBarcodeUseCase's own narrow interfaces (ISP) -- kept
// local rather than imported since those are unexported in package pms.
type variantFinderForBookingScanResolve interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
}
type productFinderForBookingScanResolve interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
}

// ResolveBookingScanCodeUseCase runs ScanBookingItemUseCase's exact code
// resolution chain (barcode -> alias -> SKU code -> this PO's own line
// item RawSKU) WITHOUT writing a BookingScan row -- lets the scanner app
// show a confirm screen (product, quantity, batch) before anything is
// staged, instead of booking instantly. See BookingScreen.tsx's
// resolveAndOpenSheet, which calls this first and only calls
// ScanBookingItemUseCase (via sku_id, skipping Code resolution entirely)
// once the scanner hits Confirm.
//
// Deliberately duplicates ScanBookingItemUseCase's resolution logic rather
// than composing it (same trade pms.ResolveBarcodeUseCase's doc comment
// already documents) -- both call their own tenant.WithTenant, and nesting
// transactions is forbidden by CLAUDE.md.
type ResolveBookingScanCodeUseCase struct {
	db                 *gorm.DB
	bookingRepo        procurementDomain.BookingRepository
	poRepo             procurementDomain.PurchaseOrderRepository
	skuRepo            pmsDomain.SKURepository
	barcodeAliasRepo   pmsDomain.BarcodeAliasRepository
	scaleCodeAliasRepo pmsDomain.ScaleCodeAliasRepository
	lineItemRepo       procurementDomain.PurchaseOrderLineItemRepository
	variantRepo        variantFinderForBookingScanResolve
	productRepo        productFinderForBookingScanResolve
}

func NewResolveBookingScanCodeUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	poRepo procurementDomain.PurchaseOrderRepository,
	skuRepo pmsDomain.SKURepository,
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository,
	scaleCodeAliasRepo pmsDomain.ScaleCodeAliasRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	variantRepo variantFinderForBookingScanResolve,
	productRepo productFinderForBookingScanResolve,
) *ResolveBookingScanCodeUseCase {
	return &ResolveBookingScanCodeUseCase{
		db: db, bookingRepo: bookingRepo, poRepo: poRepo, skuRepo: skuRepo,
		barcodeAliasRepo: barcodeAliasRepo, scaleCodeAliasRepo: scaleCodeAliasRepo, lineItemRepo: lineItemRepo,
		variantRepo: variantRepo, productRepo: productRepo,
	}
}

func (uc *ResolveBookingScanCodeUseCase) Execute(ctx context.Context, bookingID, code string) (*ResolveBookingScanCodeResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("resolveBookingScanCode: %w", procurementDomain.ErrScanCodeOrSKURequired)
	}

	var result ResolveBookingScanCodeResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		booking, err := uc.bookingRepo.FindByID(ctx, bookingID)
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

		sku, err := uc.skuRepo.FindByBarcode(ctx, code)
		if err == nil {
			return uc.fillResult(ctx, &result, *sku, 1, nil)
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		alias, aerr := uc.barcodeAliasRepo.FindByCode(ctx, code)
		if aerr == nil {
			sku, err = uc.skuRepo.FindByID(ctx, alias.SKUID)
			if err != nil {
				return err
			}
			return uc.fillResult(ctx, &result, *sku, alias.UnitsPerScan, nil)
		}
		if !errors.Is(aerr, pmsDomain.ErrBarcodeAliasNotFound) {
			return aerr
		}

		if decoded, derr := pmsDomain.DecodeWeightEmbeddedBarcode(code); derr == nil {
			sku, err = uc.skuRepo.FindByScaleCode(ctx, decoded.ScaleCode)
			if err == nil {
				return uc.fillResult(ctx, &result, *sku, 1, &decoded.WeightGrams)
			}
			if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
				return err
			}
			scaleAlias, saerr := uc.scaleCodeAliasRepo.FindByCode(ctx, decoded.ScaleCode)
			if saerr == nil {
				sku, err = uc.skuRepo.FindByID(ctx, scaleAlias.SKUID)
				if err != nil {
					return err
				}
				return uc.fillResult(ctx, &result, *sku, 1, &decoded.WeightGrams)
			}
			if !errors.Is(saerr, pmsDomain.ErrScaleCodeAliasNotFound) {
				return saerr
			}
		} else if !errors.Is(derr, pmsDomain.ErrNotWeightEmbeddedBarcode) {
			return derr // right shape, bad checksum -- a mis-scan, not "not found"
		}

		byCode, cerr := uc.skuRepo.FindByCode(ctx, code)
		if cerr == nil {
			if byCode.PricingMode == pmsDomain.SKUPricingModePerWeight {
				return pmsDomain.ErrWeightRequiredForPerWeightSKU
			}
			return uc.fillResult(ctx, &result, *byCode, 1, nil)
		}
		if !errors.Is(cerr, pmsDomain.ErrSKUNotFound) {
			return cerr
		}

		liSKUID, lerr := uc.resolveViaLineItem(ctx, po.ID, code)
		if lerr != nil {
			return lerr
		}
		bySKU, err := uc.skuRepo.FindByID(ctx, liSKUID)
		if err != nil {
			return err
		}
		if bySKU.PricingMode == pmsDomain.SKUPricingModePerWeight {
			return pmsDomain.ErrWeightRequiredForPerWeightSKU
		}
		return uc.fillResult(ctx, &result, *bySKU, 1, nil)
	})
	if err != nil {
		return nil, fmt.Errorf("resolveBookingScanCode: %w", err)
	}
	return &result, nil
}

// resolveViaLineItem mirrors ScanBookingItemUseCase.resolveViaLineItem
// exactly -- see that method's doc comment.
func (uc *ResolveBookingScanCodeUseCase) resolveViaLineItem(ctx context.Context, purchaseOrderID, code string) (string, error) {
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

func (uc *ResolveBookingScanCodeUseCase) fillResult(ctx context.Context, result *ResolveBookingScanCodeResult, sku pmsDomain.SKU, unitsPerScan int64, weightGrams *int64) error {
	variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		return err
	}
	product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		return err
	}
	result.SKUID = sku.ID
	result.ProductName = product.Name
	result.UnitsPerScan = unitsPerScan
	result.WeightGrams = weightGrams
	return nil
}
