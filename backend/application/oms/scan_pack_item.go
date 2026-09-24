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

type ScanPackItemInput struct {
	SessionID string
	// Code is the raw scanned text -- tried as a unit barcode, then a
	// case/box barcode alias, then a directly-typed SKU code, then, last
	// resort, this order's own line items' SKUCode/Barcode. Mutually
	// exclusive with SKUID.
	Code string
	// SKUID lets a caller scan directly against an already-known SKU.
	SKUID     string
	Quantity  int64
	ScannedBy string
	// ActionType empty means a plain pack. Freebie/new item add an extra
	// item to the order (applied when the session completes) -- the SKU
	// needn't be on the order; new item requires PriceMinorUnits.
	ActionType      omsDomain.PackScanActionType
	PriceMinorUnits *int64
	Reason          string
}

type ScanPackItemResult struct {
	Scan omsDomain.PackScan
}

// ScanPackItemUseCase is the packing screen's "scan a barcode, confirm"
// action -- resolves the code to a SKU, matches it against the order's own
// (already SKU-linked) line items, and writes one PackScan row. Unlike
// ScanPickItemUseCase there's no substitute/freebie/new-item/staged-replay
// complexity: packing never touches stock or the order's line items, and by
// the time an order reaches packing every line is already linked to a real
// catalog SKU (ScanPickItemUseCase's auto-link already ran during picking),
// so an unresolvable line item here is a genuine mismatch, not something to
// auto-link.
type ScanPackItemUseCase struct {
	db               *gorm.DB
	sessionRepo      omsDomain.PackingSessionRepository
	scanRepo         omsDomain.PackScanRepository
	lineItemRepo     omsDomain.SalesOrderLineItemRepository
	skuRepo          skuResolverRepo
	barcodeAliasRepo barcodeAliasFinder
}

func NewScanPackItemUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PackingSessionRepository,
	scanRepo omsDomain.PackScanRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	skuRepo skuResolverRepo,
	barcodeAliasRepo barcodeAliasFinder,
) *ScanPackItemUseCase {
	return &ScanPackItemUseCase{
		db: db, sessionRepo: sessionRepo, scanRepo: scanRepo, lineItemRepo: lineItemRepo,
		skuRepo: skuRepo, barcodeAliasRepo: barcodeAliasRepo,
	}
}

func (uc *ScanPackItemUseCase) Execute(ctx context.Context, in ScanPackItemInput) (*ScanPackItemResult, error) {
	in.Code = strings.TrimSpace(in.Code)
	if in.Code == "" && in.SKUID == "" {
		return nil, fmt.Errorf("scanPackItem: %w", omsDomain.ErrPackScanCodeOrSKURequired)
	}
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("scanPackItem: %w", invDomain.ErrInvalidQuantity)
	}
	if in.ActionType == "" {
		in.ActionType = omsDomain.PackScanActionPack
	}
	switch in.ActionType {
	case omsDomain.PackScanActionPack, omsDomain.PackScanActionFreebie:
	case omsDomain.PackScanActionNewItem:
		if in.PriceMinorUnits == nil {
			return nil, fmt.Errorf("scanPackItem: %w", omsDomain.ErrPriceRequiredForNewItem)
		}
	default:
		return nil, fmt.Errorf("scanPackItem: %w", omsDomain.ErrInvalidPackScanActionType)
	}

	var scan omsDomain.PackScan
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PackingSessionStatusInProgress {
			return omsDomain.ErrPackingSessionNotActive
		}
		if err := claimPackingLock(ctx, uc.sessionRepo, session, in.ScannedBy, time.Now()); err != nil {
			return err
		}

		lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, session.SalesOrderID)
		if err != nil {
			return err
		}

		skuID := in.SKUID
		if skuID == "" {
			skuID, err = uc.resolveCode(ctx, lineItems, in.Code)
			if err != nil {
				return err
			}
		}

		if in.ActionType != omsDomain.PackScanActionPack {
			scan = omsDomain.PackScan{
				TenantID: tenant.IDFromContext(ctx), SessionID: in.SessionID, SKUID: skuID, ActionType: in.ActionType,
				Quantity: in.Quantity, PriceMinorUnits: in.PriceMinorUnits, Reason: in.Reason,
				ScannedAt: time.Now(), ScannedBy: in.ScannedBy,
			}
			return uc.scanRepo.Create(ctx, &scan)
		}

		var lineItem *omsDomain.SalesOrderLineItem
		for i := range lineItems {
			if lineItems[i].SKUID != nil && *lineItems[i].SKUID == skuID {
				lineItem = &lineItems[i]
				break
			}
		}
		if lineItem == nil {
			return omsDomain.ErrPackScanSKUMismatch
		}

		scans, err := uc.scanRepo.ListBySession(ctx, in.SessionID)
		if err != nil {
			return err
		}
		// Substituted and refunded units count as handled too.
		var alreadyCovered int64
		for _, s := range scans {
			if id, qty := s.Covers(); id == lineItem.ID {
				alreadyCovered += qty
			}
		}
		if alreadyCovered+in.Quantity > lineItem.Quantity {
			return omsDomain.ErrPackQuantityExceedsOrdered
		}

		lineItemID := lineItem.ID
		scan = omsDomain.PackScan{
			TenantID: tenant.IDFromContext(ctx), SessionID: in.SessionID, LineItemID: &lineItemID, SKUID: skuID,
			ActionType: omsDomain.PackScanActionPack, Quantity: in.Quantity, ScannedAt: time.Now(), ScannedBy: in.ScannedBy,
		}
		return uc.scanRepo.Create(ctx, &scan)
	})
	if err != nil {
		return nil, fmt.Errorf("scanPackItem: %w", err)
	}
	return &ScanPackItemResult{Scan: scan}, nil
}

// resolveCode is ScanPickItemUseCase.resolveCode's trimmed packing
// counterpart -- deliberately duplicated, not shared, same "each use case
// owns its own resolution algorithm" convention that resolve_pick_scan_code.go's
// doc comment establishes. Skips the weight-embedded-barcode step: a
// per-weight item's actual weight was already captured at pick time, packing
// only needs to confirm which line it belongs to.
func (uc *ScanPackItemUseCase) resolveCode(ctx context.Context, lineItems []omsDomain.SalesOrderLineItem, code string) (string, error) {
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
