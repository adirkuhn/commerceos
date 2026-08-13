package pms

import (
	"context"
	"errors"
	"fmt"

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
	db               *gorm.DB
	skuRepo          pmsDomain.SKURepository
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository
	variantRepo      variantFinderForResolveBarcode
	productRepo      productFinderForResolveBarcode
}

func NewResolveBarcodeUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, barcodeAliasRepo pmsDomain.BarcodeAliasRepository, variantRepo variantFinderForResolveBarcode, productRepo productFinderForResolveBarcode) *ResolveBarcodeUseCase {
	return &ResolveBarcodeUseCase{db: db, skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo, variantRepo: variantRepo, productRepo: productRepo}
}

func (uc *ResolveBarcodeUseCase) Execute(ctx context.Context, code string) (*ResolveBarcodeResult, error) {
	var result ResolveBarcodeResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByBarcode(ctx, code)
		if err == nil {
			return uc.fillResult(ctx, &result, *sku, 1)
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
			return uc.fillResult(ctx, &result, *sku, alias.UnitsPerScan)
		}
		if !errors.Is(err, pmsDomain.ErrBarcodeAliasNotFound) {
			return err
		}

		sku, err = uc.skuRepo.FindByCode(ctx, code)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrSKUNotFound) {
				return pmsDomain.ErrSKUNotFound
			}
			return err
		}
		return uc.fillResult(ctx, &result, *sku, 1)
	})
	if err != nil {
		return nil, fmt.Errorf("resolveBarcode: %w", err)
	}
	return &result, nil
}

// fillResult resolves the SKU's product name (via its variant) alongside
// the SKU/UnitsPerScan every resolution path already found -- one place so
// each of the three lookup paths above doesn't repeat the two extra hops.
func (uc *ResolveBarcodeUseCase) fillResult(ctx context.Context, result *ResolveBarcodeResult, sku pmsDomain.SKU, unitsPerScan int64) error {
	variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		return err
	}
	product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		return err
	}
	*result = ResolveBarcodeResult{SKU: sku, UnitsPerScan: unitsPerScan, ProductName: product.Name}
	return nil
}
