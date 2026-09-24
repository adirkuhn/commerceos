package oms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type ResolvedScanCode struct {
	SKUID       string
	ProductName string
	// WeightGrams is non-nil only when the scanned code was a
	// weight-embedded barcode (see pmsDomain.DecodeWeightEmbeddedBarcode)
	// -- the actual weight of the specific package scanned, for the
	// scanner app to prefill the pick quantity with (in kg) instead of
	// defaulting to 1.
	WeightGrams *int64
}

// variantFinderForPickScanResolve/productFinderForPickScanResolve mirror
// procurement's identically-named types for Booking -- kept local rather
// than imported since those are unexported in that package.
type variantFinderForPickScanResolve interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
}
type productFinderForPickScanResolve interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Product, error)
}

// ResolvePickScanCodeUseCase runs ScanPickItemUseCase's exact code
// resolution chain (barcode -> alias -> SKU code -> this order's own line
// items) WITHOUT writing a PickScan row -- lets the scanner app show a
// confirm screen (product, quantity) before anything is staged, instead of
// scanning instantly. Mirrors procurement.ResolveBookingScanCodeUseCase.
//
// Deliberately duplicates ScanPickItemUseCase's resolution logic rather
// than composing it (same trade ResolveBookingScanCodeUseCase's own doc
// comment documents) -- both call their own tenant.WithTenant, and nesting
// transactions is forbidden by CLAUDE.md.
type ResolvePickScanCodeUseCase struct {
	db          *gorm.DB
	sessionRepo omsDomain.PickingSessionRepository
	resolver    scanCodeResolver
}

func NewResolvePickScanCodeUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	skuRepo skuResolverRepo,
	barcodeAliasRepo barcodeAliasFinder,
	scaleCodeAliasRepo scaleCodeAliasFinder,
	variantRepo variantFinderForPickScanResolve,
	productRepo productFinderForPickScanResolve,
) *ResolvePickScanCodeUseCase {
	return &ResolvePickScanCodeUseCase{db: db, sessionRepo: sessionRepo, resolver: scanCodeResolver{
		lineItemRepo: lineItemRepo, skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo,
		scaleCodeAliasRepo: scaleCodeAliasRepo, variantRepo: variantRepo, productRepo: productRepo,
	}}
}

func (uc *ResolvePickScanCodeUseCase) Execute(ctx context.Context, sessionID, code string) (*ResolvedScanCode, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("resolvePickScanCode: %w", omsDomain.ErrPickScanCodeOrSKURequired)
	}

	var result *ResolvedScanCode
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PickingSessionStatusInProgress {
			return omsDomain.ErrPickingSessionNotActive
		}
		result, err = uc.resolver.resolve(ctx, session.SalesOrderID, code)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("resolvePickScanCode: %w", err)
	}
	return result, nil
}

// scanCodeResolver is the code-resolution chain shared by picking and
// packing's resolve-scan: barcode, barcode alias, weight-embedded scale
// barcode (and its scale-code alias), SKU code, then the order's own line
// items. Runs inside the caller's transaction.
type scanCodeResolver struct {
	lineItemRepo       omsDomain.SalesOrderLineItemRepository
	skuRepo            skuResolverRepo
	barcodeAliasRepo   barcodeAliasFinder
	scaleCodeAliasRepo scaleCodeAliasFinder
	variantRepo        variantFinderForPickScanResolve
	productRepo        productFinderForPickScanResolve
}

func (uc scanCodeResolver) resolve(ctx context.Context, salesOrderID, code string) (*ResolvedScanCode, error) {
	var result ResolvedScanCode
	if err := uc.resolveInto(ctx, &result, salesOrderID, code); err != nil {
		return nil, err
	}
	return &result, nil
}

func (uc scanCodeResolver) resolveInto(ctx context.Context, result *ResolvedScanCode, salesOrderID, code string) error {
	sku, err := uc.skuRepo.FindByBarcode(ctx, code)
	if err == nil {
		return uc.fillResult(ctx, result, *sku, nil)
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
		return uc.fillResult(ctx, result, *sku, nil)
	}
	if !errors.Is(aerr, pmsDomain.ErrBarcodeAliasNotFound) {
		return aerr
	}

	if decoded, derr := pmsDomain.DecodeWeightEmbeddedBarcode(code); derr == nil {
		sku, err = uc.skuRepo.FindByScaleCode(ctx, decoded.ScaleCode)
		if err == nil {
			return uc.fillResult(ctx, result, *sku, &decoded.WeightGrams)
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
			return uc.fillResult(ctx, result, *sku, &decoded.WeightGrams)
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
		return uc.fillResult(ctx, result, *byCode, nil)
	}
	if !errors.Is(cerr, pmsDomain.ErrSKUNotFound) {
		return cerr
	}

	liSKUID, lerr := uc.resolveViaLineItem(ctx, salesOrderID, code)
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
	return uc.fillResult(ctx, result, *bySKU, nil)
}

// resolveViaLineItem mirrors ScanPickItemUseCase.resolveViaLineItem exactly
// -- see that method's doc comment.
func (uc scanCodeResolver) resolveViaLineItem(ctx context.Context, salesOrderID, code string) (string, error) {
	lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, salesOrderID)
	if err != nil {
		return "", err
	}
	for _, li := range lineItems {
		if li.SKUID == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(li.SKUCode), code) || strings.EqualFold(strings.TrimSpace(li.Barcode), code) {
			return *li.SKUID, nil
		}
	}
	return "", pmsDomain.ErrSKUNotFound
}

func (uc scanCodeResolver) fillResult(ctx context.Context, result *ResolvedScanCode, sku pmsDomain.SKU, weightGrams *int64) error {
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
	result.WeightGrams = weightGrams
	return nil
}
