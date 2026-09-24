package oms

import (
	"context"
	"fmt"
	"time"

	invapp "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// changeMarker records that a staged change has been applied -- the picking
// or packing scan repository, whichever the change came from.
type changeMarker interface {
	MarkReplayed(ctx context.Context, id string, replayedAt time.Time) error
}

// OrderChangeApplier makes staged order changes (domain/oms.StagedChange)
// real when a picking or packing session completes: substitutes reverse the
// original line's stock and decrement the substitute's, freebies/new items
// become line items with their own stock movements, refunds go through the
// order's payment provider (or flag the order for manual review). Shared by
// CompletePickingUseCase and CompletePackingUseCase -- the rules don't depend
// on which step the change was recorded in.
type OrderChangeApplier struct {
	db                   *gorm.DB
	salesOrderRepo       omsDomain.SalesOrderRepository
	lineItemRepo         omsDomain.SalesOrderLineItemRepository
	linkRepo             omsDomain.SalesOrderLineItemStockMovementRepository
	addLineItem          addLineItemForPickCompletionUC
	recordMovement       recordSaleMovementUC
	reverseMovement      movementReverser
	stockChecker         stockAvailabilityChecker
	skuRepo              skuFinderForShortfallNames
	outboxRepo           outbox.Repository
	refundClientResolver paymentRefundClientResolver
}

func NewOrderChangeApplier(
	db *gorm.DB,
	salesOrderRepo omsDomain.SalesOrderRepository,
	lineItemRepo omsDomain.SalesOrderLineItemRepository,
	linkRepo omsDomain.SalesOrderLineItemStockMovementRepository,
	addLineItem addLineItemForPickCompletionUC,
	recordMovement recordSaleMovementUC,
	reverseMovement movementReverser,
	stockChecker stockAvailabilityChecker,
	skuRepo skuFinderForShortfallNames,
	outboxRepo outbox.Repository,
	refundClientResolver paymentRefundClientResolver,
) *OrderChangeApplier {
	return &OrderChangeApplier{
		db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo,
		addLineItem: addLineItem, recordMovement: recordMovement, reverseMovement: reverseMovement,
		stockChecker: stockChecker, skuRepo: skuRepo, outboxRepo: outboxRepo, refundClientResolver: refundClientResolver,
	}
}

// CheckStockShortfalls dry-runs every unapplied substitute/freebie/new-item
// change that doesn't already have a Resolution, so staff get a decision
// instead of a hard failure. Never writes anything.
func (a *OrderChangeApplier) CheckStockShortfalls(ctx context.Context, warehouseID string, changes []omsDomain.StagedChange, resolutions map[string]StockShortfallResolution) ([]StockShortfall, error) {
	var shortfalls []StockShortfall
	for _, s := range changes {
		if s.ReplayedAt != nil {
			continue
		}
		if s.Action != omsDomain.ChangeSubstitute && s.Action != omsDomain.ChangeFreebie && s.Action != omsDomain.ChangeNewItem {
			continue
		}
		if _, resolved := resolutions[s.ID]; resolved {
			continue
		}
		short, err := a.stockChecker.CheckAvailability(ctx, s.SKUID, warehouseID, s.Quantity)
		if err != nil {
			return nil, err
		}
		if short <= 0 {
			continue
		}
		shortfalls = append(shortfalls, StockShortfall{
			ScanID: s.ID, SKUID: s.SKUID, Name: a.shortfallName(ctx, s), Needed: s.Quantity, Short: short,
		})
	}
	return shortfalls, nil
}

func (a *OrderChangeApplier) shortfallName(ctx context.Context, s omsDomain.StagedChange) string {
	if s.Action == omsDomain.ChangeFreebie || s.Action == omsDomain.ChangeNewItem {
		if s.Reason != "" {
			return s.Reason
		}
	}
	sku, err := a.skuRepo.FindByID(ctx, s.SKUID)
	if err != nil || sku == nil {
		return s.SKUID
	}
	return sku.SKUCode
}

// Apply makes every unapplied change real -- substitutes, extra items,
// refunds -- marking each applied through marker as it commits, so a retry
// after a partial failure never applies one twice. source labels the stock
// movements' reason (e.g. "picking session <id>").
func (a *OrderChangeApplier) Apply(ctx context.Context, source string, order *omsDomain.SalesOrder, lineItems []omsDomain.SalesOrderLineItem, changes []omsDomain.StagedChange, resolutions map[string]StockShortfallResolution, marker changeMarker) error {
	salesOrderID, warehouseID := order.ID, order.WarehouseID
	lineItemsByID := make(map[string]omsDomain.SalesOrderLineItem, len(lineItems))
	for _, li := range lineItems {
		lineItemsByID[li.ID] = li
	}

	handledSubstituteLines := make(map[string]bool)
	handledScanIDs := make(map[string]bool)
	// refundTotals seeds lazily from each line item's own already-persisted
	// RefundedQuantity/RefundedAmountMinorUnits on first touch, then
	// accumulates in memory across however many Refund scans land on the
	// same line within this one replay call -- see replayRefund's doc.
	refundTotals := make(map[string]refundTotal)

	for _, s := range changes {
		// Already applied by an earlier Finish attempt on this session, or
		// by replaySubstituteGroup earlier in this same pass -- skip
		// re-applying it.
		if s.ReplayedAt != nil || handledScanIDs[s.ID] {
			continue
		}

		switch s.Action {
		case omsDomain.ChangeSubstitute:
			lineItemID := derefOrEmpty(s.LineItemID)
			li, ok := lineItemsByID[lineItemID]
			if !ok {
				return omsDomain.ErrLineItemNotOnOrder
			}
			if handledSubstituteLines[lineItemID] {
				continue
			}
			handledSubstituteLines[lineItemID] = true
			if err := a.replaySubstituteGroup(ctx, source, warehouseID, li, changes, resolutions, handledScanIDs, marker); err != nil {
				return err
			}

		case omsDomain.ChangeRefund:
			lineItemID := derefOrEmpty(s.LineItemID)
			li, ok := lineItemsByID[lineItemID]
			if !ok {
				return omsDomain.ErrLineItemNotOnOrder
			}
			if err := a.replayRefund(ctx, order, li, s, refundTotals, marker); err != nil {
				return err
			}

		case omsDomain.ChangeFreebie, omsDomain.ChangeNewItem:
			if resolutions[s.ID] == StockShortfallWithoutItem {
				// Staff chose to ship without this extra item -- it's
				// never created and nothing is ever decremented for it.
				if err := a.markApplied(ctx, marker, s.ID); err != nil {
					return err
				}
				continue
			}
			price := int64(0)
			if s.PriceMinorUnits != nil {
				price = *s.PriceMinorUnits
			}
			newLI, err := a.addLineItem.Execute(ctx, AddSalesOrderLineItemInput{
				SalesOrderID: salesOrderID, SKUID: s.SKUID, Name: s.Reason, Quantity: s.Quantity, PriceMinorUnits: price,
			})
			if err != nil {
				return err
			}
			movements, err := a.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, Type: invDomain.MovementTypeSale,
				Quantity: s.Quantity, Reason: fmt.Sprintf("%s %s", source, s.Action), OccurredAt: s.ScannedAt,
				ReconcileShortfall: resolutions[s.ID] == StockShortfallWithItem,
			})
			if err != nil {
				return err
			}
			for _, m := range movements {
				if err := linkLineItemMovement(ctx, a.db, a.linkRepo, newLI.ID, m); err != nil {
					return err
				}
			}
			if err := freezeCostAtSale(ctx, a.db, a.lineItemRepo, a.skuRepo, newLI.ID, s.SKUID, s.Quantity); err != nil {
				return err
			}
			if err := a.markApplied(ctx, marker, s.ID); err != nil {
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
func (a *OrderChangeApplier) replaySubstituteGroup(ctx context.Context, source, warehouseID string, li omsDomain.SalesOrderLineItem, changes []omsDomain.StagedChange, resolutions map[string]StockShortfallResolution, handledScanIDs map[string]bool, marker changeMarker) error {
	var group []omsDomain.StagedChange
	for _, s := range changes {
		if s.Action == omsDomain.ChangeSubstitute && s.ReplayedAt == nil && derefOrEmpty(s.LineItemID) == li.ID {
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
		if err := a.reverseOriginalStock(ctx, source, warehouseID, li, reverseQty, group[0].ScannedAt); err != nil {
			return err
		}
	}

	for _, s := range group {
		if resolutions[s.ID] != StockShortfallWithoutItem {
			movements, err := a.recordMovement.Execute(ctx, invapp.AllocateSaleStockInput{
				SKUID: s.SKUID, WarehouseID: warehouseID, Type: invDomain.MovementTypeSale,
				Quantity: s.Quantity, Reason: fmt.Sprintf("%s substitute", source), OccurredAt: s.ScannedAt,
				ReconcileShortfall: resolutions[s.ID] == StockShortfallWithItem,
			})
			if err != nil {
				return err
			}
			for _, m := range movements {
				if err := linkLineItemMovement(ctx, a.db, a.linkRepo, li.ID, m); err != nil {
					return err
				}
			}
			if err := freezeCostAtSale(ctx, a.db, a.lineItemRepo, a.skuRepo, li.ID, s.SKUID, s.Quantity); err != nil {
				return err
			}
		}
		if err := a.markApplied(ctx, marker, s.ID); err != nil {
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
func (a *OrderChangeApplier) reverseOriginalStock(ctx context.Context, source, warehouseID string, li omsDomain.SalesOrderLineItem, qty int64, occurredAt time.Time) error {
	links, err := listLineItemMovements(ctx, a.db, a.linkRepo, li.ID)
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
		if _, err := a.reverseMovement.Execute(ctx, invapp.RecordStockMovementInput{
			SKUID: *li.SKUID, WarehouseID: warehouseID, BatchID: l.BatchID, Type: invDomain.MovementTypeAdjustment,
			QuantityDelta: take, Reason: fmt.Sprintf("%s substitute", source), OccurredAt: occurredAt,
		}); err != nil {
			return err
		}
		remaining -= take
	}
	return nil
}

// refundTotal is replayRefund's running in-memory accumulator for one line
// item across however many Refund scans land on it within a single replay
// call -- see replay's refundTotals doc.
type refundTotal struct {
	Quantity         int64
	AmountMinorUnits int64
}

// replayRefund applies one refund change: attempts a real refund
// through the order's own payment provider when known, otherwise (or on any
// API failure) falls back to flagging the order for manual review -- either
// way the line item's refunded totals accumulate and picking completion is
// never blocked by a payment-provider problem. See PickScanActionRefund's
// domain doc comment for the full policy.
func (a *OrderChangeApplier) replayRefund(ctx context.Context, order *omsDomain.SalesOrder, li omsDomain.SalesOrderLineItem, s omsDomain.StagedChange, totals map[string]refundTotal, marker changeMarker) error {
	amount := li.PriceMinorUnits * s.Quantity

	issued := false
	if order.PaymentProvider != "" && order.PaymentExternalID != "" {
		client, ok, err := a.refundClientResolver.Resolve(ctx, order.PaymentProvider)
		if err == nil && ok {
			if _, rerr := client.Refund(ctx, order.PaymentExternalID, amount, s.Reason); rerr == nil {
				issued = true
			}
		}
	}

	total, ok := totals[li.ID]
	if !ok {
		total = refundTotal{Quantity: li.RefundedQuantity, AmountMinorUnits: li.RefundedAmountMinorUnits}
	}
	total.Quantity += s.Quantity
	total.AmountMinorUnits += amount
	totals[li.ID] = total

	err := tenant.WithTenant(ctx, a.db, func(ctx context.Context) error {
		if err := a.lineItemRepo.Update(ctx, li.ID, map[string]any{
			"refunded_quantity": total.Quantity, "refunded_amount_minor_units": total.AmountMinorUnits,
		}); err != nil {
			return err
		}
		if issued {
			if err := a.salesOrderRepo.Update(ctx, order.ID, map[string]any{"status": string(omsDomain.SalesOrderStatusPartiallyRefunded)}); err != nil {
				return err
			}
			return enqueuePaymentRefundIssued(ctx, a.outboxRepo, order.ID, li.ID, s.SKUID, s.Quantity, amount)
		}
		if err := a.salesOrderRepo.Update(ctx, order.ID, map[string]any{"needs_manual_refund_review": true}); err != nil {
			return err
		}
		return enqueuePaymentRefundManualReviewRequired(ctx, a.outboxRepo, order.ID, li.ID, s.SKUID, s.Quantity, amount)
	})
	if err != nil {
		return err
	}
	return a.markApplied(ctx, marker, s.ID)
}

// markApplied persists that a row's effect already committed -- its own
// transaction, separate from the effect itself (tenant.WithTenant must
// never nest), so this is the one small window a crash can still land in:
// see replay's doc comment.
func (a *OrderChangeApplier) markApplied(ctx context.Context, marker changeMarker, id string) error {
	return tenant.WithTenant(ctx, a.db, func(ctx context.Context) error {
		return marker.MarkReplayed(ctx, id, time.Now())
	})
}
