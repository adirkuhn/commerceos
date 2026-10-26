package oms

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// EditSalesOrderInput is the admin order form sent again for an existing
// order: everything it shows, lines included. An empty Number keeps the
// order's own.
type EditSalesOrderInput struct {
	ID           string
	Number       string
	WarehouseID  string
	CustomerID   *string
	Billing      omsDomain.Address
	Shipping     omsDomain.Address
	CustomerNote string
	OccurredAt   time.Time

	ShippingTotalMinorUnits int64
	FeeLabel                string
	FeeTotalMinorUnits      int64
	TaxTotalMinorUnits      int64
	TotalMinorUnits         int64

	LineItems []CreateSalesOrderLineItemInput
}

type activePickingFinder interface {
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PickingSession, error)
}

type activePackingFinder interface {
	FindActiveBySalesOrder(ctx context.Context, salesOrderID string) (*omsDomain.PackingSession, error)
}

// EditSalesOrderUseCase lets staff correct a manual order before it is
// picked (see SalesOrder.Editable): its lines are replaced, not patched,
// the way the form sends them. An order holding stock gives it all back
// first and takes it again for the new lines -- the same steps as a
// status change out of and back into a stock-reducing status, so a short
// line lands in Stock issues the usual way.
//
// Like DeleteSalesOrderUseCase, the stock is given back before the lines
// change hands: if that fails the order is left as it was.
type EditSalesOrderUseCase struct {
	db              *gorm.DB
	salesOrderRepo  omsDomain.SalesOrderRepository
	lineItemRepo    omsDomain.SalesOrderLineItemRepository
	linkRepo        omsDomain.SalesOrderLineItemStockMovementRepository
	skuRepo         skuLookupRepo
	allocate        recordSaleMovementUC
	reverseMovement movementReverser
	picking         activePickingFinder
	packing         activePackingFinder
	outbox          eventrecord.Enqueuer
}

func NewEditSalesOrderUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, linkRepo omsDomain.SalesOrderLineItemStockMovementRepository, skuRepo skuLookupRepo, allocate recordSaleMovementUC, reverseMovement movementReverser, picking activePickingFinder, packing activePackingFinder, outbox eventrecord.Enqueuer) *EditSalesOrderUseCase {
	return &EditSalesOrderUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, linkRepo: linkRepo, skuRepo: skuRepo,
		allocate: allocate, reverseMovement: reverseMovement, picking: picking, packing: packing, outbox: outbox}
}

func (uc *EditSalesOrderUseCase) Execute(ctx context.Context, in EditSalesOrderInput) (*CreateSalesOrderResult, error) {
	if len(in.LineItems) == 0 {
		return nil, fmt.Errorf("editSalesOrder: %w", omsDomain.ErrEmptyLineItems)
	}
	order, oldLines, err := uc.load(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("editSalesOrder: %w", err)
	}
	if order.StockDecremented {
		if err := restockLineItems(ctx, uc.db, uc.lineItemRepo, uc.salesOrderRepo, uc.reverseMovement, uc.linkRepo, uc.outbox,
			order.ID, order.WarehouseID, oldLines); err != nil {
			return nil, fmt.Errorf("editSalesOrder: %w", err)
		}
		order.NeedsStockReview = false
	}

	edited, lines, err := uc.replace(ctx, *order, in)
	if err != nil {
		return nil, fmt.Errorf("editSalesOrder: %w", err)
	}
	result := &CreateSalesOrderResult{SalesOrder: edited, LineItems: lines}
	if !edited.StockDecremented {
		return result, nil
	}
	pending, err := allocateLineItems(ctx, uc.db, uc.lineItemRepo, uc.linkRepo, uc.skuRepo, uc.allocate, uc.outbox,
		edited.WarehouseID, lines, fmt.Sprintf("manual sale %s edited", edited.ID), time.Now())
	if err != nil {
		return nil, fmt.Errorf("editSalesOrder: %w", err)
	}
	if len(pending) > 0 {
		if err := setOrderNeedsStockReview(ctx, uc.db, uc.salesOrderRepo, uc.outbox, edited.ID, true, OrderChangeReasonStockShortfall); err != nil {
			return nil, fmt.Errorf("editSalesOrder: %w", err)
		}
		result.SalesOrder.NeedsStockReview = true
		for _, i := range pending {
			result.LineItems[i].StockStatus = omsDomain.StockStatusPending
		}
	}
	return result, nil
}

// load reads the order and its lines, refusing one that is no longer
// editable or that someone is picking or packing right now.
func (uc *EditSalesOrderUseCase) load(ctx context.Context, id string) (*omsDomain.SalesOrder, []omsDomain.SalesOrderLineItem, error) {
	var order *omsDomain.SalesOrder
	var lines []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		if order, err = uc.salesOrderRepo.FindByID(ctx, id); err != nil {
			return err
		}
		if err := order.Editable(); err != nil {
			return err
		}
		if picking, err := uc.picking.FindActiveBySalesOrder(ctx, id); err != nil {
			return err
		} else if picking != nil {
			return omsDomain.ErrOrderPickingInProgress
		}
		if packing, err := uc.packing.FindActiveBySalesOrder(ctx, id); err != nil {
			return err
		} else if packing != nil {
			return omsDomain.ErrOrderPackingInProgress
		}
		lines, err = uc.lineItemRepo.ListBySalesOrder(ctx, id)
		return err
	})
	return order, lines, err
}

// replace writes the new lines and order fields in one transaction.
func (uc *EditSalesOrderUseCase) replace(ctx context.Context, before omsDomain.SalesOrder, in EditSalesOrderInput) (omsDomain.SalesOrder, []omsDomain.SalesOrderLineItem, error) {
	after := before
	if in.Number != "" {
		after.Number = in.Number
	}
	after.WarehouseID, after.CustomerID, after.Billing, after.Shipping, after.CustomerNote = in.WarehouseID, in.CustomerID, in.Billing, in.Shipping, in.CustomerNote
	after.ShippingTotalMinorUnits, after.FeeLabel, after.FeeTotalMinorUnits = in.ShippingTotalMinorUnits, in.FeeLabel, in.FeeTotalMinorUnits
	after.TaxTotalMinorUnits, after.TotalMinorUnits = in.TaxTotalMinorUnits, in.TotalMinorUnits
	if !in.OccurredAt.IsZero() {
		after.OccurredAt = in.OccurredAt
	}
	after.UpdatedAt = time.Now()

	fields := map[string]any{
		"warehouse_id": after.WarehouseID, "customer_id": after.CustomerID, "customer_note": after.CustomerNote, "occurred_at": after.OccurredAt,
		"shipping_total_minor_units": after.ShippingTotalMinorUnits, "fee_label": after.FeeLabel, "fee_total_minor_units": after.FeeTotalMinorUnits,
		"tax_total_minor_units": after.TaxTotalMinorUnits, "total_minor_units": after.TotalMinorUnits, "updated_at": after.UpdatedAt,
	}
	if in.Number != "" {
		fields["number"] = in.Number
	}
	setBillingFields(fields, after.Billing)
	setShippingFields(fields, after.Shipping)

	var lines []omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.lineItemRepo.DeleteBySalesOrder(ctx, before.ID); err != nil {
			return err
		}
		if err := uc.salesOrderRepo.Update(ctx, before.ID, fields); err != nil {
			return err
		}
		for _, li := range in.LineItems {
			line := newOrderLineItem(ctx, uc.skuRepo, after.ID, li, after.UpdatedAt)
			if err := uc.lineItemRepo.Create(ctx, &line); err != nil {
				return err
			}
			lines = append(lines, line)
		}
		return eventrecord.Record(ctx, uc.outbox, events.SalesOrderEdited, before.ID, eventrecord.NewChanges().
			Set("number", before.Number, after.Number).
			Set("warehouse_id", before.WarehouseID, after.WarehouseID).
			Set("customer_id", before.CustomerID, after.CustomerID).
			Set("shipping_total_minor_units", before.ShippingTotalMinorUnits, after.ShippingTotalMinorUnits).
			Set("fee_total_minor_units", before.FeeTotalMinorUnits, after.FeeTotalMinorUnits).
			Set("tax_total_minor_units", before.TaxTotalMinorUnits, after.TaxTotalMinorUnits).
			Set("total_minor_units", before.TotalMinorUnits, after.TotalMinorUnits),
			eventrecord.Data{"line_count": len(lines)})
	})
	return after, lines, err
}
