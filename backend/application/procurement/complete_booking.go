package procurement

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
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
	Execute(ctx context.Context, in invapp.MarkDamagedOrLostInput) (*invDomain.StockMovement, error)
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
) *CompleteBookingUseCase {
	return &CompleteBookingUseCase{
		db: db, bookingRepo: bookingRepo, poRepo: poRepo, lineItemRepo: lineItemRepo, scanRepo: scanRepo,
		receiveStock: receiveStock, recordAdjustment: recordAdjustment, markDamagedOrLost: markDamagedOrLost,
	}
}

func (uc *CompleteBookingUseCase) Execute(ctx context.Context, id string, completedBy string) (*CompleteBookingResult, error) {
	var booking *procurementDomain.Booking
	var warehouseID string
	var lineItems []procurementDomain.PurchaseOrderLineItem
	var scans []procurementDomain.BookingScan

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		booking, err = uc.bookingRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if booking.Status == procurementDomain.BookingStatusCompleted {
			return procurementDomain.ErrBookingNotActive
		}

		po, err := uc.poRepo.FindByID(ctx, booking.PurchaseOrderID)
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
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("completeBooking: %w", err)
	}

	if err := uc.replay(ctx, id, warehouseID, scans); err != nil {
		return nil, fmt.Errorf("completeBooking: %w", err)
	}

	var result CompleteBookingResult
	result.MissingItems = missingBookingItems(lineItems, scans)

	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		now := time.Now()
		booking.Status = procurementDomain.BookingStatusCompleted
		booking.CompletedAt = &now
		booking.CompletedBy = completedBy
		if err := uc.bookingRepo.Update(ctx, booking); err != nil {
			return err
		}
		result.Booking = *booking
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("completeBooking: %w", err)
	}
	return &result, nil
}

func (uc *CompleteBookingUseCase) replay(ctx context.Context, bookingID, warehouseID string, scans []procurementDomain.BookingScan) error {
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
			receipt, err := uc.receiveStock.Execute(ctx, invapp.ReceiveStockInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, Quantity: s.Quantity, BatchID: batchID,
				LotNumber: s.LotNumber, ExpiryDate: s.ExpiryDate, Reason: fmt.Sprintf("scanner booking %s", bookingID),
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
			if _, err := uc.markDamagedOrLost.Execute(ctx, invapp.MarkDamagedOrLostInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, BatchID: batchID, Quantity: s.WriteOffQuantity,
				Type: movementType, Reason: reason, OccurredAt: s.ScannedAt,
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

// markReplayed persists that a row's inventory effect already committed --
// its own transaction, separate from the effect itself (tenant.WithTenant
// must never nest), so this is the one small window a crash can still land
// in: see replay's doc comment.
func (uc *CompleteBookingUseCase) markReplayed(ctx context.Context, scanID string, batchID *string) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.scanRepo.MarkReplayed(ctx, scanID, time.Now(), batchID)
	})
}

func missingBookingItems(lineItems []procurementDomain.PurchaseOrderLineItem, scans []procurementDomain.BookingScan) []MissingBookingItem {
	scannedBySKU := make(map[string]int64, len(scans))
	for _, s := range scans {
		// WriteOffQuantity counts too -- a damaged/lost unit still arrived
		// (see BookLineSheet's own copy: "stays counted as received;
		// written off separately"), it just isn't sellable. Quantity alone
		// is 0 for damaged/lost rows, so omitting this made every
		// damaged/lost unit show up as permanently missing even after
		// Finish, which contradicted that documented intent.
		scannedBySKU[s.SKUID] += s.Quantity + s.WriteOffQuantity
	}

	var missing []MissingBookingItem
	for _, li := range lineItems {
		// Charge-type rows (shipping/discount/fee/credit_note) have no SKU
		// and can never be scanned -- without this they'd show up as
		// permanently "missing" on every booking that has one.
		if !li.LineType.IsProduct() {
			continue
		}
		var scannedQty int64
		if li.SKUID != nil {
			scannedQty = scannedBySKU[*li.SKUID]
		}
		// BookingScan.Quantity is already in stock-unit terms (a
		// case-barcode scan credits UnitsPerScan units at once) --
		// li.Quantity is purchased-item terms, so it must be scaled by
		// UnitsPerPurchasedItem before comparing the two (see
		// PurchaseOrderLineItem's doc comment).
		expectedQty := li.Quantity * li.EffectiveUnitsPerPurchasedItem()
		if scannedQty < expectedQty {
			missing = append(missing, MissingBookingItem{
				LineItemID: li.ID, RawDescription: li.RawDescription, ExpectedQty: expectedQty, ScannedQty: scannedQty,
			})
		}
	}
	return missing
}
