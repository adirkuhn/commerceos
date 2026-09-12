package pms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ResolveBarcodeResult struct {
	SKU pmsDomain.SKU
	// UnitsPerScan is 1 for a direct unit-barcode match, or the case/box
	// multiplier when resolved via a BarcodeAlias.
	UnitsPerScan int64
	// ProductName is the resolved SKU's product's name -- the scanner apps
	// show this instead of the SKU code alone, which is rarely human
	// readable on its own (often just a raw barcode).
	ProductName string
	// WeightGrams is non-nil only when the scanned code was a
	// weight-embedded barcode (see pmsDomain.DecodeWeightEmbeddedBarcode)
	// -- the actual weight of the specific package scanned.
	WeightGrams *int64
	// LineTotalMinorUnits is non-nil only alongside WeightGrams: the
	// amount to charge for this specific scanned package. Fixed pricing
	// (the SKU default) ignores the scanned weight and is just
	// SKU.PriceMinorUnits; per-weight pricing is
	// SKU.PricePerKiloMinorUnits * WeightGrams / 1000, rounded to the
	// nearest minor unit.
	LineTotalMinorUnits *int64
}

// ResolveBarcodeUseCase resolves a scanned or typed code to a SKU -- tried
// first as a unit barcode (SKU.Barcode), then as a case/box
// BarcodeAlias.Code, then as a directly-typed SKU.SKUCode (the scanner
// apps' "type a SKU code" fallback for stock with no barcode yet). Shared
// by every scanner mini-app that turns a scan into a SKU (inventory adjust/
// move/remove, price lookup, pick order).
//
// Deliberately standalone rather than extracted from
// procurement.ScanBookingItemUseCase, which has its own well-tested inline
// resolution: composing this use case's Execute (which opens its own
// tenant.WithTenant) inside that one's existing transaction would nest
// transactions, which CLAUDE.md forbids. Small duplication of the lookup
// logic is the safer trade here.
// Narrow interfaces at point of use (ISP) -- *pms.VariantRepo/*pms.ProductRepo
// satisfy these structurally, no adapter needed in production wiring.
type variantFinderForResolveBarcode interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
}
type productFinderForResolveBarcode interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
}

type ResolveBarcodeUseCase struct {
	db                 *gorm.DB
	skuRepo            pmsDomain.SKURepository
	barcodeAliasRepo   pmsDomain.BarcodeAliasRepository
	scaleCodeAliasRepo pmsDomain.ScaleCodeAliasRepository
	variantRepo        variantFinderForResolveBarcode
	productRepo        productFinderForResolveBarcode
}

func NewResolveBarcodeUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, barcodeAliasRepo pmsDomain.BarcodeAliasRepository, scaleCodeAliasRepo pmsDomain.ScaleCodeAliasRepository, variantRepo variantFinderForResolveBarcode, productRepo productFinderForResolveBarcode) *ResolveBarcodeUseCase {
	return &ResolveBarcodeUseCase{db: db, skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo, scaleCodeAliasRepo: scaleCodeAliasRepo, variantRepo: variantRepo, productRepo: productRepo}
}

func (uc *ResolveBarcodeUseCase) Execute(ctx context.Context, code string) (*ResolveBarcodeResult, error) {
	// Defensive trim even though the scanner apps already trim client-side
	// -- a direct API caller or a scanner that emits a trailing CR/LF
	// shouldn't silently miss an exact-match lookup over stray whitespace.
	code = strings.TrimSpace(code)
	var result ResolveBarcodeResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByBarcode(ctx, code)
		if err == nil {
			return uc.fillResult(ctx, &result, *sku, 1, nil)
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		alias, err := uc.barcodeAliasRepo.FindByCode(ctx, code)
		if err == nil {
			sku, err = uc.skuRepo.FindByID(ctx, alias.SKUID)
			if err != nil {
				return err
			}
			return uc.fillResult(ctx, &result, *sku, alias.UnitsPerScan, nil)
		}
		if !errors.Is(err, pmsDomain.ErrBarcodeAliasNotFound) {
			return err
		}

		sku, err = uc.skuRepo.FindByCode(ctx, code)
		if err == nil {
			return uc.fillResult(ctx, &result, *sku, 1, nil)
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		// Last resort: a butcher/deli scale barcode, whose full value
		// never matches any of the exact lookups above since the weight
		// digits differ on every printed package (see
		// pmsDomain.DecodeWeightEmbeddedBarcode's doc comment).
		decoded, decodeErr := pmsDomain.DecodeWeightEmbeddedBarcode(code)
		if decodeErr != nil {
			if errors.Is(decodeErr, pmsDomain.ErrNotWeightEmbeddedBarcode) {
				return pmsDomain.ErrSKUNotFound
			}
			// Right shape, bad checksum -- a mis-scan, not "not found".
			return decodeErr
		}
		sku, err = uc.skuRepo.FindByScaleCode(ctx, decoded.ScaleCode)
		if err == nil {
			return uc.fillResult(ctx, &result, *sku, 1, &decoded.WeightGrams)
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		// A tenant with more than one physical scale can have this same
		// product's code programmed differently on each -- see
		// pmsDomain.ScaleCodeAlias.
		scaleAlias, err := uc.scaleCodeAliasRepo.FindByCode(ctx, decoded.ScaleCode)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrScaleCodeAliasNotFound) {
				return pmsDomain.ErrSKUNotFound
			}
			return err
		}
		sku, err = uc.skuRepo.FindByID(ctx, scaleAlias.SKUID)
		if err != nil {
			return err
		}
		return uc.fillResult(ctx, &result, *sku, 1, &decoded.WeightGrams)
	})
	if err != nil {
		return nil, fmt.Errorf("resolveBarcode: %w", err)
	}
	return &result, nil
}

// fillResult resolves the SKU's product name (via its variant) alongside
// the SKU/UnitsPerScan every resolution path already found -- one place so
// each of the four lookup paths above doesn't repeat the two extra hops.
// weightGrams is non-nil only for the weight-embedded-barcode path, and
// drives the LineTotalMinorUnits computation below.
func (uc *ResolveBarcodeUseCase) fillResult(ctx context.Context, result *ResolveBarcodeResult, sku pmsDomain.SKU, unitsPerScan int64, weightGrams *int64) error {
	variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		return err
	}
	product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		return err
	}
	*result = ResolveBarcodeResult{SKU: sku, UnitsPerScan: unitsPerScan, ProductName: product.Name, WeightGrams: weightGrams}
	if weightGrams != nil {
		total := lineTotalForScannedWeight(sku, *weightGrams)
		result.LineTotalMinorUnits = &total
	}
	return nil
}

// lineTotalForScannedWeight computes the amount to charge for one scanned
// weight-embedded package -- see ResolveBarcodeResult.LineTotalMinorUnits.
// Rounds to the nearest minor unit (round-half-up): fine here since both
// operands are always non-negative.
func lineTotalForScannedWeight(sku pmsDomain.SKU, weightGrams int64) int64 {
	if sku.PricingMode != pmsDomain.SKUPricingModePerWeight || sku.PricePerKiloMinorUnits == nil {
		return sku.PriceMinorUnits
	}
	return (*sku.PricePerKiloMinorUnits*weightGrams + 500) / 1000
}
