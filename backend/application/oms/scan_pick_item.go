package oms

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
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
	FindByScaleCode(ctx context.Context, scaleCode string) (*pmsDomain.SKU, error)
}
type barcodeAliasFinder interface {
	FindByCode(ctx context.Context, code string) (*pmsDomain.BarcodeAlias, error)
}
type scaleCodeAliasFinder interface {
	FindByCode(ctx context.Context, code string) (*pmsDomain.ScaleCodeAlias, error)
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
	// ActionType defaults to PickScanActionPick when empty -- the only
	// action picking accepts; order changes are recorded in packing (see
	// ErrOrderChangesHappenInPacking). The line item is always resolved by
	// matching the scanned SKU against the order's lines.
	ActionType omsDomain.PickScanActionType
	// Quantity is always a positive magnitude.
	Quantity  int64
	Reason    string
	ScannedBy string
}

type ScanPickItemResult struct {
	Scan omsDomain.PickScan
}

// ScanPickItemUseCase is the scanner app's "scan a barcode, confirm" action
// for picking. Picking only picks (order changes are recorded in packing).
// Resolves the code to a SKU and writes ONE staged PickScan row. That's it -- this use
// case never touches real inventory or the order's line items. A session in
// progress can still be paused, resumed, or have the picker switch to
// another order, so nothing should reach real state until
// CompletePickingUseCase replays every staged row when the session
// finishes.
type ScanPickItemUseCase struct {
	outbox             eventrecord.Enqueuer
	db                 *gorm.DB
	sessionRepo        omsDomain.PickingSessionRepository
	scanRepo           omsDomain.PickScanRepository
	lineItemRepo       omsDomain.SalesOrderLineItemRepository
	salesOrderRepo     omsDomain.SalesOrderRepository
	skuRepo            skuResolverRepo
	barcodeAliasRepo   barcodeAliasFinder
	scaleCodeAliasRepo scaleCodeAliasFinder
	variantRepo        variantFinderForPickScanResolve
	productRepo        productFinderForPickScanResolve
}

func NewScanPickItemUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	scanRepo omsDomain.PickScanRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	skuRepo skuResolverRepo,
	barcodeAliasRepo barcodeAliasFinder,
	scaleCodeAliasRepo scaleCodeAliasFinder,
	variantRepo variantFinderForPickScanResolve,
	productRepo productFinderForPickScanResolve,
	outbox eventrecord.Enqueuer,
) *ScanPickItemUseCase {
	return &ScanPickItemUseCase{outbox: outbox,
		db: db, sessionRepo: sessionRepo, scanRepo: scanRepo, lineItemRepo: lineItemRepo, salesOrderRepo: salesOrderRepo,
		skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo, scaleCodeAliasRepo: scaleCodeAliasRepo,
		variantRepo: variantRepo, productRepo: productRepo,
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
	case omsDomain.PickScanActionPick:
	case omsDomain.PickScanActionSubstitute, omsDomain.PickScanActionFreebie, omsDomain.PickScanActionNewItem, omsDomain.PickScanActionRefund:
		return nil, fmt.Errorf("scanPickItem: %w", omsDomain.ErrOrderChangesHappenInPacking)
	default:
		return nil, fmt.Errorf("scanPickItem: %w", omsDomain.ErrInvalidPickScanActionType)
	}
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("scanPickItem: %w", invDomain.ErrInvalidQuantity)
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
		if err := claimPickingLock(ctx, uc.sessionRepo, session, in.ScannedBy, time.Now()); err != nil {
			return err
		}

		skuID := in.SKUID
		if skuID == "" {
			skuID, _, err = uc.resolveCode(ctx, session.SalesOrderID, in.Code)
			if err != nil {
				return err
			}
		}

		lineItemID, err := uc.matchLineItemBySKU(ctx, session.SalesOrderID, skuID)
		if err != nil {
			return err
		}

		scan = omsDomain.PickScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: in.SessionID, LineItemID: &lineItemID, SKUID: skuID,
			ActionType: actionType, Quantity: in.Quantity, Reason: in.Reason,
			ScannedAt: time.Now(), ScannedBy: in.ScannedBy,
		}
		if err := uc.scanRepo.Create(ctx, &scan); err != nil {
			return err
		}
		return recordPickScan(ctx, uc.outbox, events.PickScanCreated, scan)
	})
	if err != nil {
		return nil, fmt.Errorf("scanPickItem: %w", err)
	}
	return &ScanPickItemResult{Scan: scan}, nil
}

// resolveCode runs the barcode -> alias -> weight-embedded barcode ->
// SKU code -> order's own line items fallback chain -- same shape as
// ScanBookingItemUseCase's inline resolution, factored into a method here
// only because ScanPickItemUseCase has more than one action type to apply
// it before (kept private to this use case, NOT shared with
// ResolvePickScanCodeUseCase -- see that type's doc comment for why the
// algorithm itself is deliberately duplicated, not composed, across the
// two).
//
// The weight-embedded step is tried before SKU-code/line-item so a
// per_weight SKU (see ErrWeightRequiredForPerWeightSKU below) gets every
// chance to resolve via its actual weight first. weightGrams is non-nil
// only when resolved that way.
func (uc *ScanPickItemUseCase) resolveCode(ctx context.Context, salesOrderID, code string) (string, *int64, error) {
	sku, err := uc.skuRepo.FindByBarcode(ctx, code)
	if err == nil {
		return sku.ID, nil, nil
	}
	if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
		return "", nil, err
	}

	alias, aerr := uc.barcodeAliasRepo.FindByCode(ctx, code)
	if aerr == nil {
		return alias.SKUID, nil, nil
	}
	if !errors.Is(aerr, pmsDomain.ErrBarcodeAliasNotFound) {
		return "", nil, aerr
	}

	if decoded, derr := pmsDomain.DecodeWeightEmbeddedBarcode(code); derr == nil {
		sku, err = uc.skuRepo.FindByScaleCode(ctx, decoded.ScaleCode)
		if err == nil {
			return sku.ID, &decoded.WeightGrams, nil
		}
		if !errors.Is(err, pmsDomain.ErrSKUNotFound) {
			return "", nil, err
		}
		scaleAlias, saerr := uc.scaleCodeAliasRepo.FindByCode(ctx, decoded.ScaleCode)
		if saerr == nil {
			return scaleAlias.SKUID, &decoded.WeightGrams, nil
		}
		if !errors.Is(saerr, pmsDomain.ErrScaleCodeAliasNotFound) {
			return "", nil, saerr
		}
	} else if !errors.Is(derr, pmsDomain.ErrNotWeightEmbeddedBarcode) {
		return "", nil, derr // right shape, bad checksum -- a mis-scan, not "not found"
	}

	byCode, cerr := uc.skuRepo.FindByCode(ctx, code)
	if cerr == nil {
		if byCode.PricingMode == pmsDomain.SKUPricingModePerWeight {
			return "", nil, pmsDomain.ErrWeightRequiredForPerWeightSKU
		}
		return byCode.ID, nil, nil
	}
	if !errors.Is(cerr, pmsDomain.ErrSKUNotFound) {
		return "", nil, cerr
	}

	liSKUID, lerr := uc.resolveViaLineItem(ctx, salesOrderID, code)
	if lerr != nil {
		return "", nil, lerr
	}
	liSKU, err := uc.skuRepo.FindByID(ctx, liSKUID)
	if err != nil {
		return "", nil, err
	}
	if liSKU.PricingMode == pmsDomain.SKUPricingModePerWeight {
		return "", nil, pmsDomain.ErrWeightRequiredForPerWeightSKU
	}
	return liSKUID, nil, nil
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
	if err := recordLineItemLinked(ctx, uc.outbox, *match, skuID, LineLinkSourcePickScan); err != nil {
		return "", err
	}
	stillUnlinked := false
	for _, li := range lineItems {
		if li.ID != match.ID && li.SKUID == nil {
			stillUnlinked = true
			break
		}
	}
	if err := setNeedsSKUReview(ctx, uc.salesOrderRepo, uc.outbox, salesOrderID, stillUnlinked); err != nil {
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
