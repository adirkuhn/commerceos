package oms

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// Why a sales order or one of its lines changed -- data.reason of
// oms.sales_order.updated / oms.sales_order_line_item.updated.
const (
	OrderChangeReasonPickingCompleted  = "picking_completed"
	OrderChangeReasonPackingCompleted  = "packing_completed"
	OrderChangeReasonMarkedPicked      = "marked_picked"
	OrderChangeReasonLineItemAdded     = "line_item_added"
	OrderChangeReasonLineItemsLinked   = "line_items_linked"
	OrderChangeReasonStockShortfall    = "stock_shortfall"
	OrderChangeReasonStockTaken        = "stock_taken"
	OrderChangeReasonStockRestored     = "stock_restored"
	OrderChangeReasonStockReviewRepair = "stock_review_repair"
)

// How a sales order line got its SKU -- data.source of
// oms.sales_order_line_item.linked.
const (
	LineLinkSourceManual   = "manual"
	LineLinkSourcePickScan = "pick_scan"
)

func recordSalesOrderCreated(ctx context.Context, outbox eventrecord.Enqueuer, o omsDomain.SalesOrder, source omsDomain.SalesOrderSource, lineCount int) error {
	return eventrecord.Record(ctx, outbox, events.SalesOrderCreated, o.ID, eventrecord.Data{
		"source": string(source), "status": string(o.Status), "number": o.Number, "currency_code": o.CurrencyCode,
		"total_minor_units": o.TotalMinorUnits, "customer_id": o.CustomerID, "sales_channel_id": o.SalesChannelID,
		"warehouse_id": o.WarehouseID, "line_count": lineCount,
	})
}

// recordSalesOrderUpdated records changes on a sales order, skipping a
// no-op (nothing actually changed).
func recordSalesOrderUpdated(ctx context.Context, outbox eventrecord.Enqueuer, orderID string, changes eventrecord.Changes, reason string) error {
	if len(changes.Fields()) == 0 {
		return nil
	}
	return eventrecord.Record(ctx, outbox, events.SalesOrderUpdated, orderID, changes, eventrecord.Data{"reason": reason})
}

func recordLineItemStockStatus(ctx context.Context, outbox eventrecord.Enqueuer, li omsDomain.SalesOrderLineItem, status omsDomain.StockStatus, reason string) error {
	changes := eventrecord.NewChanges().Set("stock_status", string(li.StockStatus), string(status))
	if len(changes.Fields()) == 0 {
		return nil
	}
	return eventrecord.Record(ctx, outbox, events.SalesOrderLineItemUpdated, li.ID, changes,
		eventrecord.Data{"sales_order_id": li.SalesOrderID, "reason": reason})
}

func recordLineItemLinked(ctx context.Context, outbox eventrecord.Enqueuer, li omsDomain.SalesOrderLineItem, skuID, source string) error {
	return eventrecord.Record(ctx, outbox, events.SalesOrderLineItemLinked, li.ID,
		eventrecord.NewChanges().Set("sku_id", li.SKUID, skuID),
		eventrecord.Data{"sales_order_id": li.SalesOrderID, "source": source})
}

// setNeedsSKUReview writes an order's needs_sku_review flag and records it
// when it changed. Runs inside the caller's transaction.
func setNeedsSKUReview(ctx context.Context, repo omsDomain.SalesOrderRepository, outbox eventrecord.Enqueuer, orderID string, needs bool) error {
	order, err := repo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if err := repo.Update(ctx, orderID, map[string]any{"needs_sku_review": needs}); err != nil {
		return err
	}
	return recordSalesOrderUpdated(ctx, outbox, orderID,
		eventrecord.NewChanges().Set("needs_sku_review", order.NeedsSKUReview, needs), OrderChangeReasonLineItemsLinked)
}

func recordActivity(ctx context.Context, outbox eventrecord.Enqueuer, a omsDomain.SalesOrderActivity) error {
	return eventrecord.Record(ctx, outbox, events.SalesOrderActivityRecorded, a.ID, eventrecord.Data{
		"sales_order_id": a.SalesOrderID, "activity_type": a.ActivityType, "summary": a.Summary,
	})
}

func recordPickScan(ctx context.Context, outbox eventrecord.Enqueuer, action events.ActionType, s omsDomain.PickScan) error {
	return eventrecord.Record(ctx, outbox, action, s.ID, eventrecord.Data{
		"session_id": s.SessionID, "line_item_id": s.LineItemID, "sku_id": s.SKUID,
		"action_type": string(s.ActionType), "quantity": s.Quantity, "reason": s.Reason,
	})
}

func recordPackScan(ctx context.Context, outbox eventrecord.Enqueuer, action events.ActionType, s omsDomain.PackScan) error {
	return eventrecord.Record(ctx, outbox, action, s.ID, eventrecord.Data{
		"session_id": s.SessionID, "line_item_id": s.LineItemID, "sku_id": s.SKUID,
		"action_type": string(s.ActionType), "quantity": s.Quantity, "price_minor_units": s.PriceMinorUnits,
		"original_qty_covered": s.OriginalQtyCovered, "return_to_stock": s.ReturnToStock, "reason": s.Reason,
	})
}

func recordSession(ctx context.Context, outbox eventrecord.Enqueuer, action events.ActionType, sessionID, salesOrderID string, extra eventrecord.Data) error {
	data := eventrecord.Data{"sales_order_id": salesOrderID}
	for k, v := range extra {
		data[k] = v
	}
	return eventrecord.Record(ctx, outbox, action, sessionID, data)
}
