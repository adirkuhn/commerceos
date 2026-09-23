package oms

import (
	"context"
	"fmt"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// MissingPackItem mirrors MissingPickItem -- one order line item that
// wasn't fully packed by the time the session was finished.
type MissingPackItem struct {
	LineItemID  string
	Name        string
	ExpectedQty int64
	PackedQty   int64
}

type CompletePackingResult struct {
	Session      omsDomain.PackingSession
	MissingItems []MissingPackItem
}

type CompletePackingInput struct {
	SessionID   string
	CompletedBy string
}

// CompletePackingUseCase is the packing screen's "Finish" action -- unlike
// CompletePickingUseCase there's nothing to replay (PackScan rows are
// already real, see that type's doc comment): completion just checks every
// line was fully packed (gaps are reported, not blocking -- same
// "completion always succeeds" philosophy as picking, since a manager may
// knowingly ship a partial box), marks the session completed, releases its
// lock, and flips the order's Packed flag so it drops off the packing
// screens' order list. Fulfilled is set too: picking is optional, and a
// packed order has by definition been physically collected -- shipping and
// reconcile logic keyed on Fulfilled keeps working for orders that skipped
// the picking module.
type CompletePackingUseCase struct {
	db             *gorm.DB
	sessionRepo    omsDomain.PackingSessionRepository
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	scanRepo       omsDomain.PackScanRepository
}

func NewCompletePackingUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PackingSessionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	scanRepo omsDomain.PackScanRepository,
) *CompletePackingUseCase {
	return &CompletePackingUseCase{
		db: db, sessionRepo: sessionRepo, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
	}
}

func (uc *CompletePackingUseCase) Execute(ctx context.Context, in CompletePackingInput) (*CompletePackingResult, error) {
	var result CompletePackingResult
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		session, err := uc.sessionRepo.FindByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if session.Status != omsDomain.PackingSessionStatusInProgress {
			return omsDomain.ErrPackingSessionNotActive
		}

		lineItems, err := uc.lineItemRepo.ListBySalesOrder(ctx, session.SalesOrderID)
		if err != nil {
			return err
		}
		scans, err := uc.scanRepo.ListBySession(ctx, in.SessionID)
		if err != nil {
			return err
		}
		result.MissingItems = missingPackItems(lineItems, scans)

		now := time.Now()
		session.Status = omsDomain.PackingSessionStatusCompleted
		session.CompletedAt = &now
		session.CompletedBy = in.CompletedBy
		session.LockedBy = nil
		session.LockedAt = nil
		if err := uc.sessionRepo.Update(ctx, session); err != nil {
			return err
		}
		if err := uc.sessionRepo.ReleaseLock(ctx, session.ID); err != nil {
			return err
		}
		if err := uc.salesOrderRepo.Update(ctx, session.SalesOrderID, map[string]any{"packed": true, "fulfilled": true}); err != nil {
			return err
		}

		result.Session = *session
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("completePacking: %w", err)
	}
	return &result, nil
}

func missingPackItems(lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PackScan) []MissingPackItem {
	packedByLineItem := make(map[string]int64, len(scans))
	for _, s := range scans {
		packedByLineItem[s.LineItemID] += s.Quantity
	}
	var missing []MissingPackItem
	for _, li := range lineItems {
		packed := packedByLineItem[li.ID]
		if packed < li.Quantity {
			missing = append(missing, MissingPackItem{LineItemID: li.ID, Name: li.Name, ExpectedQty: li.Quantity, PackedQty: packed})
		}
	}
	return missing
}
