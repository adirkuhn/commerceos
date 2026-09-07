package oms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// skuResolverRepo/barcodeAliasFinder narrow pms's SKURepository/
// BarcodeAliasRepository to point of use (ISP) -- shared by
// ScanPickItemUseCase and ResolveBookingScanCodeUseCase's picking
// counterpart, ResolvePickScanCodeUseCase.
type skuResolverRepo interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
	FindByCode(ctx context.Context, code string) (*pmsDomain.SKU, error)
	FindByBarcode(ctx context.Context, barcode string) (*pmsDomain.SKU, error)
}
type barcodeAliasFinder interface {
	FindByCode(ctx context.Context, code string) (*pmsDomain.BarcodeAlias, error)
}

type ScanPickItemInput struct {
	SessionID string
	// Code is the raw scanned text -- tried first as a unit barcode
	// (SKU.Barcode), then as a case/box BarcodeAlias.Code, then as a
	// directly-typed SKU.SKUCode, then, last resort, against this
	// session's own order's line items' SKUCode/Barcode (see
	// resolveViaLineItem). Mutually exclusive with SKUID -- exactly one
	// must be set.
	Code string
	// SKUID lets a caller scan directly against an already-known SKU (e.g.
	// tapping a line item row on the scanner screen) without a barcode --
	// skips the Code resolution path entirely when set.
	SKUID string
	// LineItemID is unused here -- Substitute (the only action type that
	// ever needed a caller-supplied line item) is rejected by this use
	// case, see ErrSubstituteViaBatchEndpointOnly. Pick resolves its own
	// line item by matching the scanned SKU against the order's line items
	// (never trusts a caller-supplied one); Freebie/NewItem have no line
	// item yet.
	LineItemID string
	// ActionType defaults to PickScanActionPick when empty.
	ActionType omsDomain.PickScanActionType
	// Quantity is always a positive magnitude.
	Quantity int64
	// PriceMinorUnits is required for NewItem, ignored (forced to 0) for
	// Freebie, and ignored (nil) for Pick/Substitute.
	PriceMinorUnits *int64
	Reason          string
	ScannedBy       string
}

type ScanPickItemResult struct {
	Scan omsDomain.PickScan
}

// ScanPickItemUseCase is the scanner app's "scan a barcode, confirm" action
// for picking -- and its Substitute/Freebie/New-Item siblings. Resolves the
// code to a SKU and writes ONE staged PickScan row. That's it -- this use
// case never touches real inventory or the order's line items. A session in
// progress can still be paused, resumed, or have the picker switch to
// another order, so nothing should reach real state until
// CompletePickingUseCase replays every staged row when the session
// finishes.
type ScanPickItemUseCase struct {
	db               *gorm.DB
	sessionRepo      omsDomain.PickingSessionRepository
	scanRepo         omsDomain.PickScanRepository
	lineItemRepo     omsDomain.SalesOrderLineItemRepository
	salesOrderRepo   omsDomain.SalesOrderRepository
	skuRepo          skuResolverRepo
	barcodeAliasRepo barcodeAliasFinder
	variantRepo      variantFinderForPickScanResolve
	productRepo      productFinderForPickScanResolve
}

func NewScanPickItemUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	scanRepo omsDomain.PickScanRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	skuRepo skuResolverRepo,
	barcodeAliasRepo barcodeAliasFinder,
	variantRepo variantFinderForPickScanResolve,
	productRepo productFinderForPickScanResolve,
) *ScanPickItemUseCase {
	return &ScanPickItemUseCase{
		db: db, sessionRepo: sessionRepo, scanRepo: scanRepo, lineItemRepo: lineItemRepo, salesOrderRepo: salesOrderRepo,
		skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo, variantRepo: variantRepo, productRepo: productRepo,
	}
}

func (uc *ScanPickItemUseCase) Execute(ctx context.Context, in ScanPickItemInput) (*ScanPickItemResult, error) {
	in.Code = strings.TrimSpace(in.Code)
	if in.Code == "" && in.SKUID == "" {
		return nil, fmt.Errorf("scanPickItem: %w", omsDomain.ErrPickScanCodeOrSKURequired)
	}
	actionType := in.ActionType
	if actionType == "" {
		actionType = omsDomain.PickScanActionPick
	}
	switch actionType {
	case omsDomain.PickScanActionPick, omsDomain.PickScanActionSubstitute,
		omsDomain.PickScanActionFreebie, omsDomain.PickScanActionNewItem:
	default:
		return nil, fmt.Errorf("scanPickItem: %w", omsDomain.ErrInvalidPickScanActionType)
	}
	if actionType == omsDomain.PickScanActionSubstitute {
		return nil, fmt.Errorf("scanPickItem: %w", omsDomain.ErrSubstituteViaBatchEndpointOnly)
	}
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("scanPickItem: %w", invDomain.ErrInvalidQuantity)
	}
	if actionType == omsDomain.PickScanActionNewItem && in.PriceMinorUnits == nil {
		return nil, fmt.Errorf("scanPickItem: %w", omsDomain.ErrPriceRequiredForNewItem)
	}

	var scan omsDomain.PickScan
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PickingSessionStatusInProgress {
			return omsDomain.ErrPickingSessionNotActive
		}

		skuID := in.SKUID
		if skuID == "" {
			skuID, err = uc.resolveCode(ctx, session.SalesOrderID, in.Code)
			if err != nil {
				return err
			}
		}

		var lineItemID *string
		quantity := in.Quantity
		var price *int64

		switch actionType {
		case omsDomain.PickScanActionPick:
			id, err := uc.matchLineItemBySKU(ctx, session.SalesOrderID, skuID)
			if err != nil {
				return err
			}
			lineItemID = &id

		case omsDomain.PickScanActionFreebie:
			zero := int64(0)
			price = &zero

		case omsDomain.PickScanActionNewItem:
			price = in.PriceMinorUnits
		}

		scan = omsDomain.PickScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: in.SessionID, LineItemID: lineItemID, SKUID: skuID,
			ActionType: actionType, Quantity: quantity, PriceMinorUnits: price, Reason: in.Reason,
			ScannedAt: time.Now(), ScannedBy: in.ScannedBy,
		}
		return uc.scanRepo.Create(ctx, &scan)
	})
	if err != nil {
		return nil, fmt.Errorf("scanPickItem: %w", err)
	}
	return &ScanPickItemResult{Scan: scan}, nil
}

// resolveCode runs the barcode -> alias -> SKU code -> order's own line
// items fallback chain -- same shape as ScanBookingItemUseCase's inline
// resolution, factored into a method here only because ScanPickItemUseCase
// has more than one action type to apply it before (kept private to this
// use case, NOT shared with ResolvePickScanCodeUseCase -- see that type's
// doc comment for why the algorithm itself is deliberately duplicated,
// not composed, across the two).
func (uc *ScanPickItemUseCase) resolveCode(ctx context.Context, salesOrderID, code string) (string, error) {
	sku, err := uc.skuRepo.FindByBarcode(ctx, code)
	if err == nil {
		return sku.ID, nil
	}
	if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
		return "", err
	}

	alias, aerr := uc.barcodeAliasRepo.FindByCode(ctx, code)
	if aerr == nil {
		return alias.SKUID, nil
	}
	if !errors.Is(aerr, pmsDomain.ErrBarcodeAliasNotFound) {
		return "", aerr
	}

	byCode, cerr := uc.skuRepo.FindByCode(ctx, code)
	if cerr == nil {
		return byCode.ID, nil
	}
	if !errors.Is(cerr, pmsDomain.ErrSKUNotFound) {
		return "", cerr
	}

	return uc.resolveViaLineItem(ctx, salesOrderID, code)
}

// resolveViaLineItem is the last-resort fallback: when code doesn't match
// anything in the tenant-wide catalog, try matching it against this
// session's own order's line items' SKUCode/Barcode -- the identifier
// exactly as snapshotted at sale time, which is often what a physical
// pick-list printout actually carries.
func (uc *ScanPickItemUseCase) resolveViaLineItem(ctx context.Context, salesOrderID, code string) (string, error) {
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

// matchLineItemBySKU finds the order's own line item for skuID -- a Pick
// scan always resolves its LineItemID this way (never trusts a
// caller-supplied one) since the whole point is confirming what's
// genuinely on the order.
func (uc *ScanPickItemUseCase) matchLineItemBySKU(ctx context.Context, salesOrderID, skuID string) (string, error) {
	lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, salesOrderID)
	if err != nil {
		return "", err
	}
	for _, li := range lineItems {
		if li.SKUID != nil && *li.SKUID == skuID {
			return li.ID, nil
		}
	}
	return uc.autoLinkUnmatchedLine(ctx, salesOrderID, skuID, lineItems)
}

// autoLinkUnmatchedLine is the "scan to resolve" path for a Woo-import line
// item that never matched a catalog SKU (SKUID nil, see
// ImportWooOrdersUseCase) -- staff physically holding the item scan its
// real barcode/code, which resolveCode above already resolved to a genuine
// catalog SKU; if that SKU's product name exactly matches (normalized)
// exactly one of this order's still-unlinked lines' own stored Name, link
// it now (same effect as LinkLineItemSKUUseCase from the Unmatched SKUs
// admin page) so the pick can proceed immediately, instead of forcing staff
// to that separate page first. Zero or more-than-one candidate match is
// left as ErrLineItemSKUMismatch -- guessing wrong here would silently
// mislink the order to the wrong product.
func (uc *ScanPickItemUseCase) autoLinkUnmatchedLine(ctx context.Context, salesOrderID, skuID string, lineItems []omsDomain.SalesOrderLineItem) (string, error) {
	sku, err := uc.skuRepo.FindByID(ctx, skuID)
	if err != nil {
		return "", err
	}
	variant, err := uc.variantRepo.FindByID(ctx, sku.VariantID)
	if err != nil {
		return "", err
	}
	product, err := uc.productRepo.FindByID(ctx, variant.ProductID)
	if err != nil {
		return "", err
	}
	targetName := normalizeCatalogName(product.Name)

	var match *omsDomain.SalesOrderLineItem
	for i := range lineItems {
		li := &lineItems[i]
		if li.SKUID != nil || normalizeCatalogName(li.Name) != targetName {
			continue
		}
		if match != nil {
			return "", omsDomain.ErrLineItemSKUMismatch // more than one candidate -- ambiguous, don't guess
		}
		match = li
	}
	if match == nil {
		return "", omsDomain.ErrLineItemSKUMismatch
	}

	if err := uc.lineItemRepo.Update(ctx, match.ID, map[string]any{"sku_id": skuID}); err != nil {
		return "", err
	}
	stillUnlinked := false
	for _, li := range lineItems {
		if li.ID != match.ID && li.SKUID == nil {
			stillUnlinked = true
			break
		}
	}
	if err := uc.salesOrderRepo.Update(ctx, salesOrderID, map[string]any{"needs_sku_review": stillUnlinked}); err != nil {
		return "", err
	}
	return match.ID, nil
}

// normalizeCatalogName is the Go-side counterpart to infrastructure/
// repositories' normalizedNameSQL -- lower-case, strip periods/commas,
// collapse whitespace, trim. Kept consistent with that SQL expression so a
// name considered "the same" at Woo-import time is also considered the
// same here, comparing two already-in-memory strings rather than a SQL
// query.
func normalizeCatalogName(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer(".", "", ",", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}
