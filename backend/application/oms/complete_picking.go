package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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

// StockShortfallResolution is how staff decide to handle one
// substitute/freebie/new-item scan whose SKU came back short on tracked
// stock -- see CompletePickingInput.Resolutions.
type StockShortfallResolution string

const (
	// StockShortfallWithItem ships the item anyway -- staff know it's
	// really on the shelf, the ledger's just wrong. Writes a reconciliation
	// Adjustment for the shortfall (AllocateSaleStockInput.ReconcileShortfall
	// -- the same escape hatch ForceAcceptStockUseCase and POS sales already
	// use) before decrementing for real.
	StockShortfallWithItem StockShortfallResolution = "with_item"
	// StockShortfallWithoutItem ships without this item -- no stock moves
	// and, for Freebie/NewItem, no line item is ever created. For a
	// Substitute, the original line's stock was already reversed (see
	// replay's Substitute case) and nothing replaces it, so the line shows
	// up in MissingItems like any other unpicked line.
	StockShortfallWithoutItem StockShortfallResolution = "without_item"
)

// StockShortfall is one substitute/freebie/new-item scan whose SKU doesn't
// have enough tracked stock in the order's warehouse -- surfaced to staff
// for a decision (see StockShortfallResolution) instead of hard-failing
// completion the way a bare AllocateSaleStockUseCase call would.
type StockShortfall struct {
	ScanID string
	SKUID  string
	// Name is the scan's own Reason for Freebie/NewItem (already the
	// intended display name -- see PickScan.Reason's doc comment), or the
	// substitute SKU's own SKUCode otherwise.
	Name   string
	Needed int64
	Short  int64
}

type CompletePickingResult struct {
	// Completed is false when StockShortfalls is non-empty -- nothing was
	// replayed or completed yet. Resubmit with Resolutions covering every
	// listed ScanID; Session/MissingItems are zero values until then.
	Completed       bool
	Session         omsDomain.PickingSession
	MissingItems    []MissingPickItem
	StockShortfalls []StockShortfall
}

type CompletePickingInput struct {
	SessionID   string
	CompletedBy string
	// Resolutions maps a PickScan.ID (one already returned in a prior
	// call's StockShortfalls) to how staff want it handled. A scan with no
	// entry here that still comes back short blocks completion again the
	// same way -- safe to retry without resolving everything at once.
	Resolutions map[string]StockShortfallResolution
}

// addLineItemForPickCompletionUC is *AddSalesOrderLineItemUseCase, narrowed
// to point of use (ISP).
type addLineItemForPickCompletionUC interface {
	Execute(ctx context.Context, in AddSalesOrderLineItemInput) (*omsDomain.SalesOrderLineItem, error)
}

// stockAvailabilityChecker is *inventory.AllocateSaleStockUseCase narrowed
// to point of use (ISP) -- lets CompletePickingUseCase detect a genuine
// shortfall up front, without writing anything, so staff get a decision
// instead of a hard failure.
type stockAvailabilityChecker interface {
	CheckAvailability(ctx context.Context, skuID, warehouseID string, quantity int64) (short int64, err error)
}

// skuFinderForShortfallNames is pmsDomain.SKURepository narrowed to point
// of use (ISP) -- just enough to show a substitute's own SKUCode in a
// StockShortfall report.
type skuFinderForShortfallNames interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.SKU, error)
}

// paymentRefundClientResolver narrows *PaymentRefundClientResolver to point
// of use (ISP) -- CompletePickingUseCase's refund replay only ever needs to
// resolve one live client for one provider name.
type paymentRefundClientResolver interface {
	Resolve(ctx context.Context, provider string) (integrationsDomain.PaymentLinkClient, bool, error)
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
// Before ever replaying anything, every unreplayed substitute/freebie/
// new-item scan without an already-supplied Resolution is checked for
// stock availability (no writes) -- see StockShortfall. If any come back
// short, Execute returns immediately with Completed:false and the list,
// touching nothing; the caller resubmits with Resolutions once staff have
// decided, and this check runs again (already-resolved scans are skipped)
// before replay actually proceeds.
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
//   - Substitute: one original line item can have several unreplayed
//     Substitute rows at once (see SubstituteLineItemUseCase); the first
//     one hit in ScannedAt order for a given line triggers
//     replaySubstituteGroup, which reverses that line's original stock
//     ONCE -- summing OriginalQtyCovered across only the group's
//     ReturnToStock rows -- then, for every row in the group (unless
//     resolved StockShortfallWithoutItem), decrements that row's own
//     substitute SKU for real, linking the new movements to the ORIGINAL
//     line item's ID (documents "this line shipped via a substitute"). The
//     original line item row is never mutated.
//   - Freebie/NewItem: unless resolved StockShortfallWithoutItem, creates a
//     real SalesOrderLineItem via AddSalesOrderLineItemUseCase (Freebie
//     forced to price 0, NewItem picker-entered), decrements its SKU for
//     real, links the new movements to the NEW line item's ID. Resolved
//     StockShortfallWithoutItem: the line item is never created at all.
//   - Refund: attempts a real refund (Quantity * the line's own
//     PriceMinorUnits) through the order's own PaymentProvider/
//     PaymentExternalID when both are set; on success the order's Status
//     becomes PartiallyRefunded. No provider info, or the API call itself
//     failing, instead sets SalesOrder.NeedsManualRefundReview -- a refund
//     never blocks completion. Either way the line item's RefundedQuantity/
//     RefundedAmountMinorUnits accumulate (see replayRefund) and the line
//     item row is never otherwise mutated.
//
// A replay failure that ISN'T a pre-flight-checked shortfall (some other
// error) is still a hard stop -- the session stays incomplete, the error
// propagates, already-replayed rows stay marked. Only scan-coverage gaps (a
// line item never scanned as Pick/Substitute, or a Substitute resolved
// StockShortfallWithoutItem) are soft-reported via MissingItems, which
// never blocks completion.
//
// The changes themselves are applied by OrderChangeApplier, shared with
// CompletePackingUseCase.
type CompletePickingUseCase struct {
	db             *gorm.DB
	sessionRepo    omsDomain.PickingSessionRepository
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	scanRepo       omsDomain.PickScanRepository
	applier        *OrderChangeApplier
	outbox         eventrecord.Enqueuer
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
	stockChecker stockAvailabilityChecker,
	skuRepo skuFinderForShortfallNames,
	outboxRepo outbox.Repository,
	refundClientResolver paymentRefundClientResolver,
) *CompletePickingUseCase {
	return &CompletePickingUseCase{
		db: db, sessionRepo: sessionRepo, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
		outbox: outboxRepo,
		applier: NewOrderChangeApplier(db, salesOrderRepo, lineItemRepo, linkRepo, addLineItem, recordMovement,
			reverseMovement, stockChecker, skuRepo, outboxRepo, refundClientResolver),
	}
}

func (uc *CompletePickingUseCase) Execute(ctx context.Context, in CompletePickingInput) (*CompletePickingResult, error) {
	var session *omsDomain.PickingSession
	var order *omsDomain.SalesOrder
	var lineItems []omsDomain.SalesOrderLineItem
	var scans []omsDomain.PickScan

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		session, err = uc.sessionRepo.FindByID(ctx, in.SessionID)
		if err != nil {
			return err
		}
		if !session.IsOpen() {
			return omsDomain.ErrPickingSessionNotActive
		}
		if session.Lock().IsHeldByOther(in.CompletedBy, time.Now()) {
			return omsDomain.ErrPickingSessionLocked
		}

		order, err = uc.salesOrderRepo.FindByID(ctx, session.SalesOrderID)
		if err != nil {
			return err
		}

		lineItems, err = uc.lineItemRepo.ListBySalesOrder(ctx, session.SalesOrderID)
		if err != nil {
			return err
		}
		scans, err = uc.scanRepo.ListBySession(ctx, in.SessionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}

	changes := pickScanChanges(scans)
	shortfalls, err := uc.applier.CheckStockShortfalls(ctx, order.WarehouseID, changes, in.Resolutions)
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}
	if len(shortfalls) > 0 {
		return &CompletePickingResult{StockShortfalls: shortfalls}, nil
	}

	if err := uc.markPlainPicksReplayed(ctx, scans); err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}
	if err := uc.applier.Apply(ctx, OrderChangeStepPicking, "picking session "+in.SessionID, order, lineItems, changes, in.Resolutions, uc.scanRepo); err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}

	var result CompletePickingResult
	result.Completed = true
	result.MissingItems = missingPickItems(lineItems, scans, in.Resolutions)

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		session.Status = omsDomain.PickingSessionStatusCompleted
		session.CompletedAt = &now
		session.CompletedBy = in.CompletedBy
		if err := uc.sessionRepo.Update(ctx, session); err != nil {
			return err
		}
		result.Session = *session

		// Completion always marks the order fulfilled, gaps or not -- see
		// this type's doc comment for why that differs from Booking's
		// received/partially_received PO status.
		if err := uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{"fulfilled": true, "updated_at": now}); err != nil {
			return err
		}
		if err := recordSession(ctx, uc.outbox, events.PickingSessionCompleted, session.ID, order.ID, eventrecord.Data{
			"completed_by": in.CompletedBy, "missing_item_count": len(result.MissingItems), "change_count": len(changes),
		}); err != nil {
			return err
		}
		return recordSalesOrderUpdated(ctx, uc.outbox, order.ID,
			eventrecord.NewChanges().Set("fulfilled", order.Fulfilled, true), OrderChangeReasonPickingCompleted)
	})
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}
	return &result, nil
}

// pickScanChanges is every order change among a session's scans -- plain
// picks change nothing.
func pickScanChanges(scans []omsDomain.PickScan) []omsDomain.StagedChange {
	var changes []omsDomain.StagedChange
	for _, s := range scans {
		if c, ok := s.Change(); ok {
			changes = append(changes, c)
		}
	}
	return changes
}

// markPlainPicksReplayed marks every not-yet-marked plain pick as replayed
// -- a plain pick changes nothing, but marking it keeps "every row of a
// completed session is replayed" true for the audit trail.
func (uc *CompletePickingUseCase) markPlainPicksReplayed(ctx context.Context, scans []omsDomain.PickScan) error {
	for _, s := range scans {
		if s.ActionType != omsDomain.PickScanActionPick || s.ReplayedAt != nil {
			continue
		}
		if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			return uc.scanRepo.MarkReplayed(ctx, s.ID, time.Now())
		}); err != nil {
			return err
		}
	}
	return nil
}

func missingPickItems(lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PickScan, resolutions map[string]StockShortfallResolution) []MissingPickItem {
	pickedByLineItem := make(map[string]int64, len(scans))
	for _, s := range scans {
		if s.LineItemID == nil {
			continue
		}
		if s.ActionType != omsDomain.PickScanActionPick && s.ActionType != omsDomain.PickScanActionSubstitute && s.ActionType != omsDomain.PickScanActionRefund {
			continue
		}
		// A substitute staff chose to ship without (StockShortfallWithoutItem)
		// never actually replaced the original line's stock -- it must not
		// count as picked.
		if s.ActionType == omsDomain.PickScanActionSubstitute && resolutions[s.ID] == StockShortfallWithoutItem {
			continue
		}
		if s.ActionType == omsDomain.PickScanActionSubstitute {
			// OriginalQtyCovered, not Quantity -- Quantity is the
			// substitute SKU's own quantity, which needn't match how much
			// of the original line it covers (e.g. 4 cheap units covering
			// 1 original unit).
			pickedByLineItem[*s.LineItemID] += derefInt64(s.OriginalQtyCovered)
			continue
		}
		// A Refund's Quantity IS the original line's own quantity being
		// resolved (no substitute-style split) -- refunded counts as
		// "resolved", same reasoning as a fulfilled Pick/Substitute.
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

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
