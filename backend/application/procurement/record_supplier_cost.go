package procurement

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// lineItemFinderForSupplierCost / purchaseOrderFinderForSupplierCost are the
// narrow lookups the supplier cost history needs (ISP).
type lineItemFinderForSupplierCost interface {
	FindByID(ctx context.Context, id string) (*procurementDomain.PurchaseOrderLineItem, error)
}

type purchaseOrderFinderForSupplierCost interface {
	FindByID(ctx context.Context, id string) (*procurementDomain.PurchaseOrder, error)
}

// RecordGoodsReceivedCostInput is one procurement.goods.received event.
type RecordGoodsReceivedCostInput struct {
	EventID                  string
	SKUID                    string
	PurchaseOrderLineItemID  string
	BatchID                  string
	Quantity                 int64
	LandedUnitCostMinorUnits int64
	ReceivedAt               time.Time
}

// RecordGoodsReceivedCostUseCase appends a goods receipt to the SKU's cost
// history by supplier (COM-11): who supplied it (the PO's supplier), at
// what price (the PO line's) and landed cost (fees included), in which
// currency. Idempotent: the row is keyed by the event, so a redelivery adds
// nothing. Raises procurement.supplier_cost.recorded for pms to follow the
// SKU's cost price.
type RecordGoodsReceivedCostUseCase struct {
	db     *gorm.DB
	lines  lineItemFinderForSupplierCost
	orders purchaseOrderFinderForSupplierCost
	costs  procurementDomain.SupplierCostRepository
	outbox eventrecord.Enqueuer
}

func NewRecordGoodsReceivedCostUseCase(db *gorm.DB, lines lineItemFinderForSupplierCost, orders purchaseOrderFinderForSupplierCost,
	costs procurementDomain.SupplierCostRepository, outbox eventrecord.Enqueuer) *RecordGoodsReceivedCostUseCase {
	return &RecordGoodsReceivedCostUseCase{db: db, lines: lines, orders: orders, costs: costs, outbox: outbox}
}

func (uc *RecordGoodsReceivedCostUseCase) Execute(ctx context.Context, in RecordGoodsReceivedCostInput) error {
	if in.Quantity <= 0 {
		return fmt.Errorf("recordGoodsReceivedCost: received quantity %d is not positive", in.Quantity)
	}
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		line, err := uc.lines.FindByID(ctx, in.PurchaseOrderLineItemID)
		if err != nil {
			return err
		}
		po, err := uc.orders.FindByID(ctx, line.PurchaseOrderID)
		if err != nil {
			return err
		}
		entry := procurementDomain.SupplierCostEntry{
			TenantID: tenant.IDFromContext(ctx), SKUID: in.SKUID, SupplierID: po.SupplierID,
			PurchaseOrderID: po.ID, PurchaseOrderLineItemID: line.ID,
			Kind: procurementDomain.SupplierCostKindReceipt, Quantity: in.Quantity,
			UnitCostMinorUnits: line.UnitCostMinorUnits, LandedUnitCostMinorUnits: in.LandedUnitCostMinorUnits,
			CurrencyCode: po.CurrencyCode, ReceivedAt: in.ReceivedAt, SourceKey: "goods_received:" + in.EventID,
		}
		if in.BatchID != "" {
			entry.BatchID = &in.BatchID
		}
		if err := uc.costs.Append(ctx, []procurementDomain.SupplierCostEntry{entry}); err != nil {
			return err
		}
		return recordSupplierCostRecorded(ctx, uc.outbox, entry)
	})
	if err != nil {
		return fmt.Errorf("recordGoodsReceivedCost: %w", err)
	}
	return nil
}

// recordSupplierCostRecorded raises procurement.supplier_cost.recorded for
// the row that best describes what was appended (for a correction pair,
// the repriced one).
func recordSupplierCostRecorded(ctx context.Context, outbox eventrecord.Enqueuer, e procurementDomain.SupplierCostEntry) error {
	return eventrecord.Record(ctx, outbox, events.SupplierCostRecorded, e.SKUID, eventrecord.Data{
		"kind": string(e.Kind), "purchase_order_id": e.PurchaseOrderID,
		"purchase_order_line_item_id": e.PurchaseOrderLineItemID, "supplier_id": e.SupplierID,
		"quantity": e.Quantity, "landed_unit_cost_minor_units": e.LandedUnitCostMinorUnits,
		"currency_code": e.CurrencyCode,
	})
}
