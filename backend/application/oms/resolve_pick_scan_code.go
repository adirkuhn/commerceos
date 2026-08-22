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

type ResolvePickScanCodeResult struct {
	SKUID       string
	ProductName string
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
	db               *gorm.DB
	sessionRepo      omsDomain.PickingSessionRepository
	lineItemRepo     omsDomain.SalesOrderLineItemRepository
	skuRepo          skuResolverRepo
	barcodeAliasRepo barcodeAliasFinder
	variantRepo      variantFinderForPickScanResolve
	productRepo      productFinderForPickScanResolve
}

func NewResolvePickScanCodeUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	skuRepo skuResolverRepo,
	barcodeAliasRepo barcodeAliasFinder,
	variantRepo variantFinderForPickScanResolve,
	productRepo productFinderForPickScanResolve,
) *ResolvePickScanCodeUseCase {
	return &ResolvePickScanCodeUseCase{
		db: db, sessionRepo: sessionRepo, lineItemRepo: lineItemRepo, skuRepo: skuRepo,
		barcodeAliasRepo: barcodeAliasRepo, variantRepo: variantRepo, productRepo: productRepo,
	}
}

func (uc *ResolvePickScanCodeUseCase) Execute(ctx context.Context, sessionID, code string) (*ResolvePickScanCodeResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("resolvePickScanCode: %w", omsDomain.ErrPickScanCodeOrSKURequired)
	}

	var result ResolvePickScanCodeResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, sessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PickingSessionStatusInProgress {
			return omsDomain.ErrPickingSessionNotActive
		}

		sku, err := uc.skuRepo.FindByBarcode(ctx, code)
		if err == nil {
			return uc.fillResult(ctx, &result, *sku)
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
			return uc.fillResult(ctx, &result, *sku)
		}
		if !errors.Is(aerr, pmsDomain.ErrBarcodeAliasNotFound) {
			return aerr
		}

		byCode, cerr := uc.skuRepo.FindByCode(ctx, code)
		if cerr == nil {
			return uc.fillResult(ctx, &result, *byCode)
		}
		if !errors.Is(cerr, pmsDomain.ErrSKUNotFound) {
			return cerr
		}

		liSKUID, lerr := uc.resolveViaLineItem(ctx, session.SalesOrderID, code)
		if lerr != nil {
			return lerr
		}
		bySKU, err := uc.skuRepo.FindByID(ctx, liSKUID)
		if err != nil {
			return err
		}
		return uc.fillResult(ctx, &result, *bySKU)
	})
	if err != nil {
		return nil, fmt.Errorf("resolvePickScanCode: %w", err)
	}
	return &result, nil
}

// resolveViaLineItem mirrors ScanPickItemUseCase.resolveViaLineItem exactly
// -- see that method's doc comment.
func (uc *ResolvePickScanCodeUseCase) resolveViaLineItem(ctx context.Context, salesOrderID, code string) (string, error) {
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

func (uc *ResolvePickScanCodeUseCase) fillResult(ctx context.Context, result *ResolvePickScanCodeResult, sku pmsDomain.SKU) error {
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
	return nil
}
