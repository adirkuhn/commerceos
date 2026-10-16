package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	"github.com/adirkuhn/commerceos/domain/events"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// MissingBookingItem is one PO line item that wasn't fully scanned by the
// time the booking was finished -- the plan's "missing-items report" for a
// manager to review, not a persisted table: it's cheap to recompute from
// LineItems + Scans on demand, same reasoning as BookingDetail.
type MissingBookingItem struct {
	LineItemID     string
	RawDescription string
	ExpectedQty    int64
	ScannedQty     int64
}

type CompleteBookingResult struct {
	Booking      procurementDomain.Booking
	MissingItems []MissingBookingItem
}

// Narrow interfaces at point of use (ISP) -- *inventory.ReceiveStockUseCase,
// *inventory.RecordStockMovementUseCase, *inventory.MarkDamagedOrLostUseCase
// satisfy these structurally, no adapter needed in production wiring.
type receiveStockForCompletionUC interface {
	Execute(ctx context.Context, in invapp.ReceiveStockInput) (*invapp.ReceiveStockResult, error)
}
type recordAdjustmentForCompletionUC interface {
	Execute(ctx context.Context, in invapp.RecordStockMovementInput) (*invDomain.StockMovement, error)
}
type markDamagedOrLostForCompletionUC interface {
	Execute(ctx context.Context, in invapp.MarkDamagedOrLostInput) (*invDomain.StockMovement, *invDomain.DiscardRecord, error)
}

// pendingBatchKey correlates a not-yet-real batch across BookingScan rows in
// the same booking, regardless of which row (or action type) references it.
type pendingBatchKey struct{ skuID, lotNumber string }

// CompleteBookingUseCase is the scanner app's "Finish" action -- and the
// ONLY place staged BookingScan rows (see that type's doc comment) ever
// become real. Finishing with every line fully scanned is a clean complete
// (MissingItems is empty); finishing with gaps still completes the booking
// and returns the gaps for a manager to review -- completion always
// succeeds regardless of gaps, per the plan.
//
// Replay algorithm: every scan is processed in ScannedAt order (oldest
// first -- ListByBooking already orders this way). A receive row with a
// LotNumber and no real BatchID creates a new batch; that batch's real ID
// is remembered by SKUID+LotNumber so a LATER row (a correction, a
// damaged/lost write-off, or even another receive into the same lot) that
// references the same not-yet-real batch resolves to the batch the earlier
// row actually created, instead of creating a second one or failing.
//
// Each row's actual inventory effect runs through its own use case call,
// each opening its own transaction (tenant.WithTenant must never nest, so a
// use case composing several others' Execute calls can't wrap them in one
// transaction -- same reasoning as ScanBookingItemUseCase used to document
// for its old two-phase receive+audit shape). Immediately after a row's
// effect succeeds, its ReplayedAt (and resolved BatchID) is persisted in a
// separate transaction -- so a row is skipped, not re-applied, on any later
// replay of this booking. Worst case on a crash is now scoped to the single
// row that was mid-flight (its own inventory effect committed but the
// ReplayedAt write didn't land, so a retry re-applies just that one row) --
// far smaller than double-applying the whole booking, and the same category
// of accepted gap as elsewhere in this codebase where composing use cases
// can't share one transaction.
type CompleteBookingUseCase struct {
	db                *gorm.DB
	bookingRepo       procurementDomain.BookingRepository
	poRepo            procurementDomain.PurchaseOrderRepository
	lineItemRepo      procurementDomain.PurchaseOrderLineItemRepository
	scanRepo          procurementDomain.BookingScanRepository
	receiveStock      receiveStockForCompletionUC
	recordAdjustment  recordAdjustmentForCompletionUC
	markDamagedOrLost markDamagedOrLostForCompletionUC
	outbox            eventrecord.Enqueuer
}

func NewCompleteBookingUseCase(
	db *gorm.DB,
	bookingRepo procurementDomain.BookingRepository,
	poRepo procurementDomain.PurchaseOrderRepository,
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository,
	scanRepo procurementDomain.BookingScanRepository,
	receiveStock receiveStockForCompletionUC,
	recordAdjustment recordAdjustmentForCompletionUC,
	markDamagedOrLost markDamagedOrLostForCompletionUC,
	outbox eventrecord.Enqueuer,
) *CompleteBookingUseCase {
	return &CompleteBookingUseCase{
		db: db, bookingRepo: bookingRepo, poRepo: poRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
		receiveStock: receiveStock, recordAdjustment: recordAdjustment, markDamagedOrLost: markDamagedOrLost,
		outbox: outbox,
	}
}

func (uc *CompleteBookingUseCase) Execute(ctx context.Context, id string, completedBy string) (*CompleteBookingResult, error) {
	var booking *procurementDomain.Booking
	var po *procurementDomain.PurchaseOrder
	var warehouseID string
	var lineItems []procurementDomain.PurchaseOrderLineItem
	var scans []procurementDomain.BookingScan
	// earlierScans are the PO's earlier, already completed deliveries --
	// counted for what is missing, never replayed again.
	var earlierScans []procurementDomain.BookingScan

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		booking, err = uc.bookingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if booking.Status == procurementDomain.BookingStatusCompleted {
			return procurementDomain.ErrBookingNotActive
		}

		po, err = uc.poRepo.FindByID(ctx, booking.PurchaseOrderID)
		if err != nil {
			return err
		}
		if po.WarehouseID == nil {
			return procurementDomain.ErrWarehouseRequired
		}
		warehouseID = *po.WarehouseID

		lineItems, err = uc.lineItemRepo.ListByPurchaseOrder(ctx, booking.PurchaseOrderID)
		if err != nil {
			return err
		}
		scans, err = uc.scanRepo.ListByBooking(ctx, id)
		if err != nil {
			return err
		}
		earlierScans, err = uc.earlierDeliveryScans(ctx, booking)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("completeBooking: %w", err)
	}

	if err := uc.replay(ctx, id, warehouseID, lineItems, scans); err != nil {
		return nil, fmt.Errorf("completeBooking: %w", err)
	}

	var result CompleteBookingResult
	// A PO delivered in parts has one booking per delivery: what is missing
	// is judged over all of them, so a later delivery that brings the rest
	// makes the PO received.
	result.MissingItems = missingBookingItems(lineItems, append(earlierScans, scans...))

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		booking.Status = procurementDomain.BookingStatusCompleted
		booking.CompletedAt = &now
		booking.CompletedBy = completedBy
		if err := uc.bookingRepo.Update(ctx, booking); err != nil {
			return err
		}
		result.Booking = *booking
		if err := eventrecord.Record(ctx, uc.outbox, events.BookingCompleted, booking.ID, eventrecord.Data{
			"purchase_order_id": booking.PurchaseOrderID, "completed_by": completedBy, "missing_item_count": len(result.MissingItems),
		}); err != nil {
			return err
		}
		before := *po

		// Every product line fully accounted for (received + damaged/lost,
		// see missingBookingItems) means the PO itself is done -- taking it
		// off PurchaseOrderPickerScreen's sent/partially_received filter is
		// what stops it from being tapped into a second, empty booking (see
		// StartOrResumeBookingUseCase: FindActiveByPurchaseOrder only
		// matches in_progress/paused, so a completed booking never blocks a
		// fresh one). A gap leaves it partially_received -- still listed,
		// since a later shipment legitimately needs its own new booking.
		if len(result.MissingItems) == 0 {
			po.Status = procurementDomain.PurchaseOrderStatusReceived
		} else {
			po.Status = procurementDomain.PurchaseOrderStatusPartiallyReceived
		}
		if err := uc.poRepo.Update(ctx, po); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderUpdated, po.ID, purchaseOrderChanges(before, *po),
			eventrecord.Data{"reason": PurchaseOrderChangeReasonBookingCompleted})
	})
	if err != nil {
		return nil, fmt.Errorf("completeBooking: %w", err)
	}
	return &result, nil
}

func (uc *CompleteBookingUseCase) replay(ctx context.Context, bookingID, warehouseID string, lineItems []procurementDomain.PurchaseOrderLineItem, scans []procurementDomain.BookingScan) error {
	pending := make(map[pendingBatchKey]string)

	resolveBatchID := func(s procurementDomain.BookingScan) (*string, error) {
		if s.BatchID != nil {
			return s.BatchID, nil
		}
		if s.LotNumber == "" {
			return nil, nil
		}
		if resolved, ok := pending[pendingBatchKey{s.SKUID, s.LotNumber}]; ok {
			return &resolved, nil
		}
		if s.ActionType == procurementDomain.BookingScanActionReceive {
			return nil, nil // not created yet -- this row creates it below
		}
		return nil, procurementDomain.ErrPendingBatchNotFound
	}

	for _, s := range scans {
		// Already applied by an earlier Finish attempt on this booking --
		// skip re-applying it, but still let a later row resolve against
		// the real batch this one already created.
		if s.ReplayedAt != nil {
			if s.BatchID != nil && s.LotNumber != "" {
				pending[pendingBatchKey{s.SKUID, s.LotNumber}] = *s.BatchID
			}
			continue
		}

		batchID, err := resolveBatchID(s)
		if err != nil {
			return err
		}

		switch s.ActionType {
		case procurementDomain.BookingScanActionReceive:
			var lineItemID *string
			var unitCost *int64
			// Best-effort match by SKU: BookingScan has no LineItemID of its
			// own (it's resolved by scanning a barcode, not picking a line),
			// so if two line items on the same PO share a SKU the first one
			// found wins the landed-cost attribution. Rare in practice (a PO
			// wouldn't normally list the same product twice) and not worse
			// than having no cost basis at all.
			for i := range lineItems {
				if lineItems[i].SKUID != nil && *lineItems[i].SKUID == s.SKUID {
					li := lineItems[i].ID
					lineItemID = &li
					cost, err := AllocateLandedUnitCost(lineItems, li)
					if err == nil {
						unitCost = &cost
					}
					break
				}
			}
			receipt, err := uc.receiveStock.Execute(ctx, invapp.ReceiveStockInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, Quantity: s.Quantity, BatchID: batchID,
				LotNumber: s.LotNumber, ExpiryDate: s.ExpiryDate, Reason: fmt.Sprintf("scanner booking %s", bookingID),
				PurchaseOrderLineItemID: lineItemID, UnitCostMinorUnits: unitCost,
			})
			if err != nil {
				return err
			}
			if receipt.Batch != nil {
				batchID = &receipt.Batch.ID
				if s.LotNumber != "" {
					pending[pendingBatchKey{s.SKUID, s.LotNumber}] = receipt.Batch.ID
				}
			}
			if err := uc.markReplayed(ctx, s.ID, batchID); err != nil {
				return err
			}

		case procurementDomain.BookingScanActionCorrection:
			reason := s.Reason
			if reason == "" {
				reason = fmt.Sprintf("scanner booking correction %s", bookingID)
			}
			if _, err := uc.recordAdjustment.Execute(ctx, invapp.RecordStockMovementInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, BatchID: batchID, Type: invDomain.MovementTypeAdjustment,
				QuantityDelta: s.Quantity, Reason: reason, OccurredAt: s.ScannedAt,
			}); err != nil {
				return err
			}
			if err := uc.markReplayed(ctx, s.ID, batchID); err != nil {
				return err
			}

		case procurementDomain.BookingScanActionDamaged, procurementDomain.BookingScanActionLost:
			movementType := invDomain.MovementTypeDamaged
			if s.ActionType == procurementDomain.BookingScanActionLost {
				movementType = invDomain.MovementTypeLost
			}
			reason := s.Reason
			if reason == "" {
				reason = fmt.Sprintf("scanner booking %s", bookingID)
			}
			if _, _, err := uc.markDamagedOrLost.Execute(ctx, invapp.MarkDamagedOrLostInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, BatchID: batchID, Quantity: s.WriteOffQuantity,
				Type: movementType, Reason: reason, OccurredAt: s.ScannedAt, DiscardedBy: s.ScannedBy,
			}); err != nil {
				return err
			}
			if err := uc.markReplayed(ctx, s.ID, batchID); err != nil {
				return err
			}
		}
	}
	return nil
}

// earlierDeliveryScans returns the scans of the PO's other completed
// bookings. Call inside tenant.WithTenant.
func (uc *CompleteBookingUseCase) earlierDeliveryScans(ctx context.Context, booking *procurementDomain.Booking) ([]procurementDomain.BookingScan, error) {
	earlier, err := uc.bookingRepo.ListCompletedByPurchaseOrder(ctx, booking.PurchaseOrderID)
	if err != nil {
		return nil, err
	}
	var scans []procurementDomain.BookingScan
	for _, b := range earlier {
		if b.ID == booking.ID {
			continue
		}
		bookingScans, err := uc.scanRepo.ListByBooking(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		scans = append(scans, bookingScans...)
	}
	return scans, nil
}

// markReplayed persists that a row's inventory effect already committed --
// its own transaction, separate from the effect itself (tenant.WithTenant
// must never nest), so this is the one small window a crash can still land
// in: see replay's doc comment.
func (uc *CompleteBookingUseCase) markReplayed(ctx context.Context, scanID string, batchID *string) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.scanRepo.MarkReplayed(ctx, scanID, time.Now(), batchID)
	})
}

// missingBookingItems is procurementDomain.Shortfalls shaped for the
// booking finish screen.
func missingBookingItems(lineItems []procurementDomain.PurchaseOrderLineItem, scans []procurementDomain.BookingScan) []MissingBookingItem {
	var missing []MissingBookingItem
	for _, sf := range procurementDomain.Shortfalls(lineItems, scans) {
		missing = append(missing, MissingBookingItem{
			LineItemID: sf.LineItem.ID, RawDescription: sf.LineItem.RawDescription, ExpectedQty: sf.ExpectedQty, ScannedQty: sf.ReceivedQty,
		})
	}
	return missing
}
