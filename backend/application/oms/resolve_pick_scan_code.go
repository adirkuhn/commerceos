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
// packing: barcode, barcode alias, SKU code, the order's own line items,
// then the weight-embedded scale barcode (and its scale-code alias) --
// exact matches first. Runs inside the caller's transaction.
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

	exact, err := exactCodeMatch(ctx, uc.skuRepo, code, func() (string, error) {
		return uc.resolveViaLineItem(ctx, salesOrderID, code)
	})
	if err != nil {
		return err
	}
	if exact != nil && exact.PricingMode != pmsDomain.SKUPricingModePerWeight {
		return uc.fillResult(ctx, result, *exact, nil)
	}

	label, err := resolveScaleLabel(ctx, uc.skuRepo, uc.scaleCodeAliasRepo, code)
	if err != nil {
		return err
	}
	if label == nil {
		if exact != nil {
			return pmsDomain.ErrWeightRequiredForPerWeightSKU
		}
		return pmsDomain.ErrSKUNotFound
	}
	sku, err = uc.skuRepo.FindByID(ctx, label.skuID)
	if err != nil {
		return err
	}
	return uc.fillResult(ctx, result, *sku, &label.weightGrams)
}

// exactCodeMatch finds the SKU whose code is exactly code: its SKU code,
// else a line on the order (lineSKUID) -- nil when neither matches. An
// exact match wins over reading code as a scale label: a package whose
// label is its SKU code ("~ 428g", no barcode) would otherwise resolve,
// through the label's scale code, to the bulk per-kilo SKU and miss the
// order line. A per-weight match is returned too; the caller still needs
// the label's weight for it.
func exactCodeMatch(ctx context.Context, skuRepo skuResolverRepo, code string, lineSKUID func() (string, error)) (*pmsDomain.SKU, error) {
	byCode, err := skuRepo.FindByCode(ctx, code)
	if err == nil {
		return byCode, nil
	}
	if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
		return nil, err
	}
	id, err := lineSKUID()
	if errors.Is(err, pmsDomain.ErrSKUNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return skuRepo.FindByID(ctx, id)
}

type scaleLabelMatch struct {
	skuID       string
	weightGrams int64
}

// resolveScaleLabel reads code as a weight-embedded scale label: the SKU
// with its scale code (or an additional scale code) and the label's
// weight. nil when code isn't a scale label or no SKU has its scale code;
// a label with a bad check digit is an error (a mis-scan, not "not found").
func resolveScaleLabel(ctx context.Context, skuRepo skuResolverRepo, aliases scaleCodeAliasFinder, code string) (*scaleLabelMatch, error) {
	decoded, err := pmsDomain.DecodeWeightEmbeddedBarcode(code)
	if errors.Is(err, pmsDomain.ErrNotWeightEmbeddedBarcode) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sku, err := skuRepo.FindByScaleCode(ctx, decoded.ScaleCode)
	if err == nil {
		return &scaleLabelMatch{skuID: sku.ID, weightGrams: decoded.WeightGrams}, nil
	}
	if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
		return nil, err
	}
	alias, err := aliases.FindByCode(ctx, decoded.ScaleCode)
	if errors.Is(err, pmsDomain.ErrScaleCodeAliasNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &scaleLabelMatch{skuID: alias.SKUID, weightGrams: decoded.WeightGrams}, nil
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
