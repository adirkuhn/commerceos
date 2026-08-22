package oms

import (
	"context"
	"errors"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PickingSessionDetail is the raw data the scanner app's progress view
// needs -- expected-vs-picked per line is arithmetic (match Scans by
// LineItemID against LineItems), deliberately left to the caller instead
// of precomputed here, same "push simple aggregation to the edge" choice
// made for procurement.BookingDetail. Unlike BookingDetail's ProductName,
// no display-name composition is needed here -- SalesOrderLineItem already
// carries its own snapshotted Name. ProductID still needs the same
// SKU->Variant lookup as AttachProductDisplayNames though, so the picking
// screens can fetch a current product thumbnail (see
// SalesOrderLineItem.ProductID's doc).
type PickingSessionDetail struct {
	Session   omsDomain.PickingSession
	LineItems []omsDomain.SalesOrderLineItem
	Scans     []omsDomain.PickScan
}

// skuFinderForPickingImages/variantFinderForPickingImages are narrow,
// point-of-use interfaces (ISP) -- this use case only ever calls FindByID,
// never the rest of pmsDomain.SKURepository/VariantRepository.
type skuFinderForPickingImages interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}
type variantFinderForPickingImages interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
}

type GetPickingSessionUseCase struct {
	db           *gorm.DB
	sessionRepo  omsDomain.PickingSessionRepository
	lineItemRepo omsDomain.SalesOrderLineItemRepository
	scanRepo     omsDomain.PickScanRepository
	skuRepo      skuFinderForPickingImages
	variantRepo  variantFinderForPickingImages
}

func NewGetPickingSessionUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	scanRepo omsDomain.PickScanRepository,
	skuRepo skuFinderForPickingImages,
	variantRepo variantFinderForPickingImages,
) *GetPickingSessionUseCase {
	return &GetPickingSessionUseCase{
		db: db, sessionRepo: sessionRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
		skuRepo: skuRepo, variantRepo: variantRepo,
	}
}

func (uc *GetPickingSessionUseCase) Execute(ctx context.Context, id string) (*PickingSessionDetail, error) {
	var detail PickingSessionDetail
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		detail.Session = *session

		lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, session.SalesOrderID)
		if err != nil {
			return err
		}
		if err := attachProductIDsForPicking(ctx, uc.skuRepo, uc.variantRepo, lineItems); err != nil {
			return err
		}
		detail.LineItems = lineItems

		scans, err := uc.scanRepo.ListBySession(ctx, id)
		if err != nil {
			return err
		}
		detail.Scans = scans
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("getPickingSession: %w", err)
	}
	return &detail, nil
}

// attachProductIDsForPicking fills in each line item's display-only
// ProductID (see its doc) via a SKU->Variant lookup -- same per-item
// composition style as procurement.AttachProductDisplayNames, minus the
// Name/SKUCode fields SalesOrderLineItem doesn't need (it already
// snapshots its own). A line with no SKUID, or whose SKU/variant has since
// been deleted, is left with a nil ProductID rather than failing the whole
// session -- an unlinked or since-deleted product just shows the fallback
// placeholder icon, same as BookingScreen's LineItemThumb does today.
func attachProductIDsForPicking(ctx context.Context, skuRepo skuFinderForPickingImages, variantRepo variantFinderForPickingImages, items []omsDomain.SalesOrderLineItem) error {
	for i := range items {
		if items[i].SKUID == nil {
			continue
		}
		sku, err := skuRepo.FindByID(ctx, *items[i].SKUID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrSKUNotFound) {
				continue
			}
			return err
		}
		variant, err := variantRepo.FindByID(ctx, sku.VariantID)
		if err != nil {
			if errors.Is(err, pmsDomain.ErrVariantNotFound) {
				continue
			}
			return err
		}
		items[i].ProductID = &variant.ProductID
	}
	return nil
}
