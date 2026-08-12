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
}

// ResolveBarcodeUseCase resolves a scanned code to a SKU -- tried first as a
// unit barcode (SKU.Barcode), then as a case/box BarcodeAlias.Code. Shared
// by every scanner mini-app that turns a scan into a SKU (inventory adjust/
// move/remove, price lookup, pick order).
//
// Deliberately standalone rather than extracted from
// procurement.ScanBookingItemUseCase, which has its own well-tested inline
// resolution: composing this use case's Execute (which opens its own
// tenant.WithTenant) inside that one's existing transaction would nest
// transactions, which CLAUDE.md forbids. Small duplication of the lookup
// logic is the safer trade here.
type ResolveBarcodeUseCase struct {
	db               *gorm.DB
	skuRepo          pmsDomain.SKURepository
	barcodeAliasRepo pmsDomain.BarcodeAliasRepository
}

func NewResolveBarcodeUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, barcodeAliasRepo pmsDomain.BarcodeAliasRepository) *ResolveBarcodeUseCase {
	return &ResolveBarcodeUseCase{db: db, skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo}
}

func (uc *ResolveBarcodeUseCase) Execute(ctx context.Context, code string) (*ResolveBarcodeResult, error) {
	var result ResolveBarcodeResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByBarcode(ctx, code)
		if err == nil {
			result = ResolveBarcodeResult{SKU: *sku, UnitsPerScan: 1}
			return nil
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return err
		}

		alias, err := uc.barcodeAliasRepo.FindByCode(ctx, code)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrBarcodeAliasNotFound) {
				return pmsDomain.ErrSKUNotFound
			}
			return err
		}
		sku, err = uc.skuRepo.FindByID(ctx, alias.SKUID)
		if err != nil {
			return err
		}
		result = ResolveBarcodeResult{SKU: *sku, UnitsPerScan: alias.UnitsPerScan}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("resolveBarcode: %w", err)
	}
	return &result, nil
}
