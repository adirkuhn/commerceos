package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// PickingSessionDetail is the raw data the scanner app's progress view
// needs -- expected-vs-picked per line is arithmetic (match Scans by
// LineItemID against LineItems), deliberately left to the caller instead
// of precomputed here, same "push simple aggregation to the edge" choice
// made for procurement.BookingDetail. Unlike BookingDetail, no
// AttachProductDisplayNames-style composition is needed -- SalesOrderLineItem
// already carries its own snapshotted Name, no separate product lookup
// required.
type PickingSessionDetail struct {
	Session   omsDomain.PickingSession
	LineItems []omsDomain.SalesOrderLineItem
	Scans     []omsDomain.PickScan
}

type GetPickingSessionUseCase struct {
	db           *gorm.DB
	sessionRepo  omsDomain.PickingSessionRepository
	lineItemRepo omsDomain.SalesOrderLineItemRepository
	scanRepo     omsDomain.PickScanRepository
}

func NewGetPickingSessionUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	scanRepo omsDomain.PickScanRepository,
) *GetPickingSessionUseCase {
	return &GetPickingSessionUseCase{db: db, sessionRepo: sessionRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo}
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
