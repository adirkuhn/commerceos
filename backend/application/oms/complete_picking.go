package oms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// MissingPickItem is one order line item that wasn't fully picked (or
// substituted) by the time the session was finished -- the "missing-items
// report" for a manager to review, not persisted: cheap to recompute from
// LineItems + Scans on demand, same reasoning as
// procurement.MissingBookingItem.
type MissingPickItem struct {
	LineItemID  string
	Name        string
	ExpectedQty int64
	PickedQty   int64
}

type CompletePickingResult struct {
	Session      omsDomain.PickingSession
	MissingItems []MissingPickItem
}

// addLineItemForPickCompletionUC is *AddSalesOrderLineItemUseCase, narrowed
// to point of use (ISP).
type addLineItemForPickCompletionUC interface {
	Execute(ctx context.Context, in AddSalesOrderLineItemInput) (*omsDomain.SalesOrderLineItem, error)
}

// CompletePickingUseCase is the scanner app's "Finish" action -- and the
// ONLY place staged PickScan rows (see that type's doc comment) ever become
// real. Finishing with every line fully picked is a clean complete
// (MissingItems is empty); finishing with gaps still completes the session
// and returns the gaps for a manager to review -- completion always
// succeeds regardless of gaps, mirroring
// procurement.CompleteBookingUseCase. Unlike Booking, completion always
// marks the order Fulfilled, gaps or not -- a partial pick is still a
// picking decision staff made and confirmed by finishing, not an
// automatic-retry state the way a partially_received PO is.
//
// Replay algorithm: every unreplayed scan is processed in ScannedAt order
// (oldest first -- ListBySession already orders this way). Each row's
// actual effect runs through its own composed use case call, each opening
// its own transaction (tenant.WithTenant must never nest). Immediately
// after a row's effect succeeds, its ReplayedAt is persisted in a separate
// transaction -- so a row is skipped, not re-applied, on any later replay
// of this session. Worst case on a crash is scoped to the single row that
// was mid-flight, same accepted category of gap as
// CompleteBookingUseCase's replay and ReceivePOSSalesOrderUseCase's
// movement loop.
//
// Per action type:
//   - Pick: no stock effect (stock already left the ledger the moment the
//     order's Status entered a stock-reducing state) -- markReplayed only.
//   - Substitute: reverses the ORIGINAL line item's own decrement
//     movements batch-by-batch (same shape as restockLineItems), then
//     decrements the substitute SKU for real, linking the new movements to
//     the ORIGINAL line item's ID (documents "this line shipped via a
//     substitute"). The original line item row is never mutated.
//   - Freebie/NewItem: creates a real SalesOrderLineItem via
//     AddSalesOrderLineItemUseCase (Freebie forced to price 0, NewItem
//     picker-entered), decrements its SKU for real, links the new
//     movements to the NEW line item's ID.
//
// A replay failure (e.g. ErrInsufficientStock on a substitute/freebie/new
// item) is a hard stop -- the session stays incomplete, the error
// propagates, already-replayed rows stay marked. Only scan-coverage gaps (a
// line item never scanned as Pick/Substitute) are soft-reported via
// MissingItems, which never blocks completion.
type CompletePickingUseCase struct {
	db              *gorm.DB
	sessionRepo     omsDomain.PickingSessionRepository
	salesOrderRepo  omsDomain.SalesOrderRepository
	lineItemRepo    omsDomain.SalesOrderLineItemRepository
	linkRepo        omsDomain.SalesOrderLineItemStockMovementRepository
	scanRepo        omsDomain.PickScanRepository
	addLineItem     addLineItemForPickCompletionUC
	recordMovement  recordSaleMovementUC
	reverseMovement movementReverser
}

func NewCompletePickingUseCase(
	db *gorm.DB,
	sessionRepo omsDomain.PickingSessionRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	linkRepo omsDomain.SalesOrderLineItemStockMovementRepository,
	scanRepo omsDomain.PickScanRepository,
	addLineItem addLineItemForPickCompletionUC,
	recordMovement recordSaleMovementUC,
	reverseMovement movementReverser,
) *CompletePickingUseCase {
	return &CompletePickingUseCase{
		db: db, sessionRepo: sessionRepo, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo,
		linkRepo: linkRepo, scanRepo: scanRepo, addLineItem: addLineItem,
		recordMovement: recordMovement, reverseMovement: reverseMovement,
	}
}

func (uc *CompletePickingUseCase) Execute(ctx context.Context, id, completedBy string) (*CompletePickingResult, error) {
	var session *omsDomain.PickingSession
	var order *omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	var scans []omsDomain.PickScan

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		session, err = uc.sessionRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if session.Status == omsDomain.PickingSessionStatusCompleted {
			return omsDomain.ErrPickingSessionNotActive
		}

		order, err = uc.salesOrderRepo.FindByID(ctx, session.SalesOrderID)
		if err != nil {
			return err
		}

		lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, session.SalesOrderID)
		if err != nil {
			return err
		}
		scans, err = uc.scanRepo.ListBySession(ctx, id)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}

	if err := uc.replay(ctx, id, order.ID, order.WarehouseID, lineItems, scans); err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}

	var result CompletePickingResult
	result.MissingItems = missingPickItems(lineItems, scans)

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		session.Status = omsDomain.PickingSessionStatusCompleted
		session.CompletedAt = &now
		session.CompletedBy = completedBy
		if err := uc.sessionRepo.Update(ctx, session); err != nil {
			return err
		}
		result.Session = *session

		// Completion always marks the order fulfilled, gaps or not -- see
		// this type's doc comment for why that differs from Booking's
		// received/partially_received PO status.
		return uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{"fulfilled": true, "updated_at": now})
	})
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}
	return &result, nil
}

func (uc *CompletePickingUseCase) replay(ctx context.Context, sessionID, salesOrderID, warehouseID string, lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PickScan) error {
	lineItemsByID := make(map[string]omsDomain.SalesOrderLineItem, len(lineItems))
	for _, li := range lineItems {
		lineItemsByID[li.ID] = li
	}

	for _, s := range scans {
		// Already applied by an earlier Finish attempt on this session --
		// skip re-applying it.
		if s.ReplayedAt != nil {
			continue
		}

		switch s.ActionType {
		case omsDomain.PickScanActionPick:
			if err := uc.markReplayed(ctx, s.ID); err != nil {
				return err
			}

		case omsDomain.PickScanActionSubstitute:
			li, ok := lineItemsByID[derefOrEmpty(s.LineItemID)]
			if !ok {
				return omsDomain.ErrLineItemNotOnOrder
			}
			// A line item with no SKUID (an unmatched Woo import line)
			// never had a decrement to reverse in the first place --
			// decrementLineItems skips it.
			if li.SKUID != nil {
				links, err := listLineItemMovements(ctx, uc.db, uc.linkRepo, li.ID)
				if err != nil {
					return err
				}
				for _, l := range links {
					if l.QuantityDelta >= 0 {
						continue
					}
					if _, err := uc.reverseMovement.Execute(ctx, invapp.RecordStockMovementInput{
						SKUID: *li.SKUID, WarehouseID: warehouseID, BatchID: l.BatchID, Type: invDomain.MovementTypeAdjustment,
						QuantityDelta: -l.QuantityDelta, Reason: fmt.Sprintf("picking session %s substitute", sessionID), OccurredAt: s.ScannedAt,
					}); err != nil {
						return err
					}
				}
			}
			movements, err := uc.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, Type: invDomain.MovementTypeSale,
				Quantity: s.Quantity, Reason: fmt.Sprintf("picking session %s substitute", sessionID), OccurredAt: s.ScannedAt,
			})
			if err != nil {
				return err
			}
			for _, m := range movements {
				if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, m); err != nil {
					return err
				}
			}
			if err := uc.markReplayed(ctx, s.ID); err != nil {
				return err
			}

		case omsDomain.PickScanActionFreebie, omsDomain.PickScanActionNewItem:
			price := int64(0)
			if s.PriceMinorUnits != nil {
				price = *s.PriceMinorUnits
			}
			newLI, err := uc.addLineItem.Execute(ctx, AddSalesOrderLineItemInput{
				SalesOrderID: salesOrderID, SKUID: s.SKUID, Name: s.Reason, Quantity: s.Quantity, PriceMinorUnits: price,
			})
			if err != nil {
				return err
			}
			movements, err := uc.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, Type: invDomain.MovementTypeSale,
				Quantity: s.Quantity, Reason: fmt.Sprintf("picking session %s %s", sessionID, s.ActionType), OccurredAt: s.ScannedAt,
			})
			if err != nil {
				return err
			}
			for _, m := range movements {
				if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, newLI.ID, m); err != nil {
					return err
				}
			}
			if err := uc.markReplayed(ctx, s.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// markReplayed persists that a row's effect already committed -- its own
// transaction, separate from the effect itself (tenant.WithTenant must
// never nest), so this is the one small window a crash can still land in:
// see replay's doc comment.
func (uc *CompletePickingUseCase) markReplayed(ctx context.Context, scanID string) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.scanRepo.MarkReplayed(ctx, scanID, time.Now())
	})
}

func missingPickItems(lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PickScan) []MissingPickItem {
	pickedByLineItem := make(map[string]int64, len(scans))
	for _, s := range scans {
		if s.LineItemID == nil {
			continue
		}
		if s.ActionType != omsDomain.PickScanActionPick && s.ActionType != omsDomain.PickScanActionSubstitute {
			continue
		}
		pickedByLineItem[*s.LineItemID] += s.Quantity
	}

	var missing []MissingPickItem
	for _, li := range lineItems {
		picked := pickedByLineItem[li.ID]
		if picked < li.Quantity {
			missing = append(missing, MissingPickItem{LineItemID: li.ID, Name: li.Name, ExpectedQty: li.Quantity, PickedQty: picked})
		}
	}
	return missing
}

func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
