package procurement

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

// How a line item got its SKU -- data.source of
// procurement.purchase_order_line_item.linked.
const (
	LinkSourceManual        = "manual"
	LinkSourceSupplierAlias = "supplier_alias"
)

// PurchaseOrderChangeReasonLineItemsResolved: data.reason of a
// procurement.purchase_order.updated the system raised when the last
// unlinked product line got linked (needs_review -> draft).
const PurchaseOrderChangeReasonLineItemsResolved = "line_items_resolved"

// purchaseOrderChanges are the fields an update changed on a PO.
func purchaseOrderChanges(before, after procurementDomain.PurchaseOrder) eventrecord.Changes {
	return eventrecord.NewChanges().
		Set("supplier_id", before.SupplierID, after.SupplierID).
		Set("warehouse_id", before.WarehouseID, after.WarehouseID).
		Set("status", string(before.Status), string(after.Status)).
		Set("currency_code", before.CurrencyCode, after.CurrencyCode).
		Set("expected_date", before.ExpectedDate, after.ExpectedDate).
		Set("notes", before.Notes, after.Notes)
}

// How a line item was added -- data.source of
// procurement.purchase_order_line_item.added.
const (
	LineItemSourceManual            = "manual"
	LineItemSourceDuplicateDocument = "duplicate_document"
)

func recordLineItemAdded(ctx context.Context, outbox eventrecord.Enqueuer, li procurementDomain.PurchaseOrderLineItem, source string) error {
	return eventrecord.Record(ctx, outbox, events.PurchaseOrderLineItemAdded, li.ID, eventrecord.Data{
		"purchase_order_id": li.PurchaseOrderID, "sku_id": li.SKUID, "quantity": li.Quantity,
		"units_per_purchased_item": li.UnitsPerPurchasedItem, "unit_cost_minor_units": li.UnitCostMinorUnits,
		"line_type": string(li.LineType), "source": source,
	})
}

func recordLineItemLinked(ctx context.Context, outbox eventrecord.Enqueuer, before, after procurementDomain.PurchaseOrderLineItem, source string) error {
	return eventrecord.Record(ctx, outbox, events.PurchaseOrderLineItemLinked, after.ID,
		eventrecord.Data{"purchase_order_id": after.PurchaseOrderID, "source": source},
		eventrecord.NewChanges().
			Set("sku_id", before.SKUID, after.SKUID).
			Set("units_per_purchased_item", before.UnitsPerPurchasedItem, after.UnitsPerPurchasedItem))
}
