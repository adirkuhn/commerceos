package oms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
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
//
// A replay failure that ISN'T a pre-flight-checked shortfall (some other
// error) is still a hard stop -- the session stays incomplete, the error
// propagates, already-replayed rows stay marked. Only scan-coverage gaps (a
// line item never scanned as Pick/Substitute, or a Substitute resolved
// StockShortfallWithoutItem) are soft-reported via MissingItems, which
// never blocks completion.
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
	stockChecker    stockAvailabilityChecker
	skuRepo         skuFinderForShortfallNames
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
) *CompletePickingUseCase {
	return &CompletePickingUseCase{
		db: db, sessionRepo: sessionRepo, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo,
		linkRepo: linkRepo, scanRepo: scanRepo, addLineItem: addLineItem,
		recordMovement: recordMovement, reverseMovement: reverseMovement,
		stockChecker: stockChecker, skuRepo: skuRepo,
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
		scans, err = uc.scanRepo.ListBySession(ctx, in.SessionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}

	shortfalls, err := uc.checkStockShortfalls(ctx, order.WarehouseID, scans, in.Resolutions)
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}
	if len(shortfalls) > 0 {
		return &CompletePickingResult{StockShortfalls: shortfalls}, nil
	}

	if err := uc.replay(ctx, in.SessionID, order.ID, order.WarehouseID, lineItems, scans, in.Resolutions); err != nil {
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
		return uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{"fulfilled": true, "updated_at": now})
	})
	if err != nil {
		return nil, fmt.Errorf("completePicking: %w", err)
	}
	return &result, nil
}

// checkStockShortfalls dry-runs every unreplayed substitute/freebie/
// new-item scan that doesn't already have a Resolution -- see this type's
// doc comment. Never writes anything.
func (uc *CompletePickingUseCase) checkStockShortfalls(ctx context.Context, warehouseID string, scans []omsDomain.PickScan, resolutions map[string]StockShortfallResolution) ([]StockShortfall, error) {
	var shortfalls []StockShortfall
	for _, s := range scans {
		if s.ReplayedAt != nil {
			continue
		}
		if s.ActionType != omsDomain.PickScanActionSubstitute && s.ActionType != omsDomain.PickScanActionFreebie && s.ActionType != omsDomain.PickScanActionNewItem {
			continue
		}
		if _, resolved := resolutions[s.ID]; resolved {
			continue
		}
		short, err := uc.stockChecker.CheckAvailability(ctx, s.SKUID, warehouseID, s.Quantity)
		if err != nil {
			return nil, err
		}
		if short <= 0 {
			continue
		}
		shortfalls = append(shortfalls, StockShortfall{
			ScanID: s.ID, SKUID: s.SKUID, Name: uc.shortfallName(ctx, s), Needed: s.Quantity, Short: short,
		})
	}
	return shortfalls, nil
}

func (uc *CompletePickingUseCase) shortfallName(ctx context.Context, s omsDomain.PickScan) string {
	if s.ActionType == omsDomain.PickScanActionFreebie || s.ActionType == omsDomain.PickScanActionNewItem {
		if s.Reason != "" {
			return s.Reason
		}
	}
	sku, err := uc.skuRepo.FindByID(ctx, s.SKUID)
	if err != nil || sku == nil {
		return s.SKUID
	}
	return sku.SKUCode
}

func (uc *CompletePickingUseCase) replay(ctx context.Context, sessionID, salesOrderID, warehouseID string, lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PickScan, resolutions map[string]StockShortfallResolution) error {
	lineItemsByID := make(map[string]omsDomain.SalesOrderLineItem, len(lineItems))
	for _, li := range lineItems {
		lineItemsByID[li.ID] = li
	}

	handledSubstituteLines := make(map[string]bool)
	handledScanIDs := make(map[string]bool)

	for _, s := range scans {
		// Already applied by an earlier Finish attempt on this session, or
		// by replaySubstituteGroup earlier in this same pass -- skip
		// re-applying it.
		if s.ReplayedAt != nil || handledScanIDs[s.ID] {
			continue
		}

		switch s.ActionType {
		case omsDomain.PickScanActionPick:
			if err := uc.markReplayed(ctx, s.ID); err != nil {
				return err
			}

		case omsDomain.PickScanActionSubstitute:
			lineItemID := derefOrEmpty(s.LineItemID)
			li, ok := lineItemsByID[lineItemID]
			if !ok {
				return omsDomain.ErrLineItemNotOnOrder
			}
			if handledSubstituteLines[lineItemID] {
				continue
			}
			handledSubstituteLines[lineItemID] = true
			if err := uc.replaySubstituteGroup(ctx, sessionID, warehouseID, li, scans, resolutions, handledScanIDs); err != nil {
				return err
			}

		case omsDomain.PickScanActionFreebie, omsDomain.PickScanActionNewItem:
			if resolutions[s.ID] == StockShortfallWithoutItem {
				// Staff chose to ship without this extra item -- it's
				// never created and nothing is ever decremented for it.
				if err := uc.markReplayed(ctx, s.ID); err != nil {
					return err
				}
				continue
			}
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
				ReconcileShortfall: resolutions[s.ID] == StockShortfallWithItem,
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

// replaySubstituteGroup handles every unreplayed Substitute row for one
// original line item together, exactly once per replay call -- see
// replay's call site, guarded by handledSubstituteLines so a line with N
// Substitute rows never has its original stock reversed more than once.
// Reverses the original line's stock ONCE, summing OriginalQtyCovered only
// across rows in the group whose own ReturnToStock is true, then applies
// each row's own substitute-SKU decrement independently.
func (uc *CompletePickingUseCase) replaySubstituteGroup(ctx context.Context, sessionID, warehouseID string, li omsDomain.SalesOrderLineItem, scans []omsDomain.PickScan, resolutions map[string]StockShortfallResolution, handledScanIDs map[string]bool) error {
	var group []omsDomain.PickScan
	for _, s := range scans {
		if s.ActionType == omsDomain.PickScanActionSubstitute && s.ReplayedAt == nil && derefOrEmpty(s.LineItemID) == li.ID {
			group = append(group, s)
		}
	}

	var reverseQty int64
	for _, s := range group {
		if s.ReturnToStock {
			reverseQty += derefInt64(s.OriginalQtyCovered)
		}
	}
	if li.SKUID != nil && reverseQty > 0 {
		if err := uc.reverseOriginalStock(ctx, sessionID, warehouseID, li, reverseQty, group[0].ScannedAt); err != nil {
			return err
		}
	}

	for _, s := range group {
		if resolutions[s.ID] != StockShortfallWithoutItem {
			movements, err := uc.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, Type: invDomain.MovementTypeSale,
				Quantity: s.Quantity, Reason: fmt.Sprintf("picking session %s substitute", sessionID), OccurredAt: s.ScannedAt,
				ReconcileShortfall: resolutions[s.ID] == StockShortfallWithItem,
			})
			if err != nil {
				return err
			}
			for _, m := range movements {
				if err := linkLineItemMovement(ctx, uc.db, uc.linkRepo, li.ID, m); err != nil {
					return err
				}
			}
		}
		if err := uc.markReplayed(ctx, s.ID); err != nil {
			return err
		}
		handledScanIDs[s.ID] = true
	}
	return nil
}

// reverseOriginalStock adds qty back to the original line's SKU, walking
// its existing decrement movements batch-by-batch (same shape as
// restockLineItems) and reversing only up to qty in total -- a partial
// reversal primitive, unlike the old whole-line-only version, since not
// every Substitute row covering this line necessarily opts into
// ReturnToStock.
func (uc *CompletePickingUseCase) reverseOriginalStock(ctx context.Context, sessionID, warehouseID string, li omsDomain.SalesOrderLineItem, qty int64, occurredAt time.Time) error {
	links, err := listLineItemMovements(ctx, uc.db, uc.linkRepo, li.ID)
	if err != nil {
		return err
	}
	remaining := qty
	for _, l := range links {
		if remaining <= 0 {
			break
		}
		if l.QuantityDelta >= 0 {
			continue
		}
		take := -l.QuantityDelta
		if take > remaining {
			take = remaining
		}
		if _, err := uc.reverseMovement.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: *li.SKUID, WarehouseID: warehouseID, BatchID: l.BatchID, Type: invDomain.MovementTypeAdjustment,
			QuantityDelta: take, Reason: fmt.Sprintf("picking session %s substitute", sessionID), OccurredAt: occurredAt,
		}); err != nil {
			return err
		}
		remaining -= take
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

func missingPickItems(lineItems []omsDomain.SalesOrderLineItem, scans []omsDomain.PickScan, resolutions map[string]StockShortfallResolution) []MissingPickItem {
	pickedByLineItem := make(map[string]int64, len(scans))
	for _, s := range scans {
		if s.LineItemID == nil {
			continue
		}
		if s.ActionType != omsDomain.PickScanActionPick && s.ActionType != omsDomain.PickScanActionSubstitute {
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
