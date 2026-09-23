package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PackingSessionDetail mirrors PickingSessionDetail -- see that type's doc
// comment.
type PackingSessionDetail struct {
	Session   omsDomain.PackingSession
	LineItems []omsDomain.SalesOrderLineItem
	Scans     []omsDomain.PackScan
}

type GetPackingSessionUseCase struct {
	db           *gorm.DB
	sessionRepo  omsDomain.PackingSessionRepository
	lineItemRepo omsDomain.SalesOrderLineItemRepository
	scanRepo     omsDomain.PackScanRepository
	skuRepo      skuFinderForPickingImages
	variantRepo  variantFinderForPickingImages
}

func NewGetPackingSessionUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PackingSessionRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	scanRepo omsDomain.PackScanRepository,
	skuRepo skuFinderForPickingImages,
	variantRepo variantFinderForPickingImages,
) *GetPackingSessionUseCase {
	return &GetPackingSessionUseCase{
		db: db, sessionRepo: sessionRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
		skuRepo: skuRepo, variantRepo: variantRepo,
	}
}

func (uc *GetPackingSessionUseCase) Execute(ctx context.Context, id string) (*PackingSessionDetail, error) {
	var detail PackingSessionDetail
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
		// attachProductIDsForPicking is generic over []SalesOrderLineItem --
		// reused as-is rather than duplicated (see that function's doc).
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
		return nil, fmt.Errorf("getPackingSession: %w", err)
	}
	return &detail, nil
}
