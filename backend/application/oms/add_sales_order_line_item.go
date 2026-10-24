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

type AddSalesOrderLineItemInput struct {
	SalesOrderID string
	SKUID        string
	// Name is picker-entered (or SKU-derived by the caller before this
	// input is built) -- this use case doesn't infer it, same "don't infer
	// what the caller already knows" split as the rest of this package's
	// line-item construction.
	Name            string
	Quantity        int64
	PriceMinorUnits int64
}

// AddSalesOrderLineItemUseCase appends a new line item to an ALREADY
// EXISTING sales order -- unlike every other line-item write in this
// package, which happens once at order creation (see SalesOrderLineItem's
// "immutable once created" doc). Needed for CompletePickingUseCase's
// freebie/new_item replay: a picker can add a product to an order mid-pick
// that was never on the original order at all.
//
// The price is the shop's own, so its VAT is split the way the shop prices
// its catalog (ShopLinePricer). TotalMinorUnits/TaxTotalMinorUnits are
// cached columns on SalesOrder, never derived from summed line items
// (unlike procurement.PurchaseOrder's), so this use case must explicitly
// bump them -- nothing else will.
//
// Does not touch stock -- same "line-item write is one transaction, stock
// movement is the caller's separate call" split used everywhere else in
// this package (see ReceivePOSSalesOrderUseCase's doc).
type AddSalesOrderLineItemUseCase struct {
	outbox         eventrecord.Enqueuer
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	skuRepo        skuLookupRepo
	pricer         addedLinePricer
}

type addedLinePricer interface {
	PriceLine(ctx context.Context, taxClassID *string, quantity, unitPriceMinorUnits int64) (omsDomain.LineAmounts, bool, error)
}

func NewAddSalesOrderLineItemUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, skuRepo skuLookupRepo, pricer addedLinePricer, outbox eventrecord.Enqueuer) *AddSalesOrderLineItemUseCase {
	return &AddSalesOrderLineItemUseCase{outbox: outbox, db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, skuRepo: skuRepo, pricer: pricer}
}

func (uc *AddSalesOrderLineItemUseCase) Execute(ctx context.Context, in AddSalesOrderLineItemInput) (*omsDomain.SalesOrderLineItem, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("addSalesOrderLineItem: %w", omsDomain.ErrInvalidLineItemQuantity)
	}
	if in.PriceMinorUnits < 0 {
		return nil, fmt.Errorf("addSalesOrderLineItem: %w", omsDomain.ErrInvalidPrice)
	}

	var lineItem omsDomain.SalesOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		order, err := uc.salesOrderRepo.FindByID(ctx, in.SalesOrderID)
		if err != nil {
			return err
		}
		sku, err := uc.skuRepo.FindByID(ctx, in.SKUID)
		if err != nil {
			return err
		}

		amounts, includesTax, err := uc.pricer.PriceLine(ctx, sku.TaxClassID, in.Quantity, in.PriceMinorUnits)
		if err != nil {
			return err
		}
		skuID := in.SKUID
		lineItem = omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: in.SalesOrderID, SKUID: &skuID,
			Name: in.Name, SKUCode: sku.SKUCode, Barcode: sku.Barcode, Quantity: in.Quantity,
			StockStatus: omsDomain.StockStatusDecremented,
			// This use case's caller (CompletePickingUseCase) records the
			// real stock movement itself right after this returns -- see
			// its doc comment for why the two aren't one transaction.
			PriceMinorUnits: in.PriceMinorUnits, SubtotalMinorUnits: amounts.Subtotal, SubtotalTaxMinorUnits: amounts.SubtotalTax,
			TotalMinorUnits: amounts.Total, TotalTaxMinorUnits: amounts.TotalTax, TotalIncludesTax: includesTax,
			TaxClassID: sku.TaxClassID, CreatedAt: time.Now(),
		}
		if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
			return err
		}

		newTotal := order.TotalMinorUnits + lineItem.TotalInclTax()
		newTax := order.TaxTotalMinorUnits + lineItem.TotalTaxMinorUnits
		if err := uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{
			"total_minor_units":     newTotal,
			"tax_total_minor_units": newTax,
			"updated_at":            time.Now(),
		}); err != nil {
			return err
		}
		if err := eventrecord.Record(ctx, uc.outbox, events.SalesOrderLineItemAdded, lineItem.ID, eventrecord.Data{
			"sales_order_id": order.ID, "sku_id": in.SKUID, "quantity": lineItem.Quantity,
			"price_minor_units": lineItem.PriceMinorUnits, "total_minor_units": lineItem.TotalMinorUnits,
		}); err != nil {
			return err
		}
		return recordSalesOrderUpdated(ctx, uc.outbox, order.ID,
			eventrecord.NewChanges().Set("total_minor_units", order.TotalMinorUnits, newTotal).
				Set("tax_total_minor_units", order.TaxTotalMinorUnits, newTax), OrderChangeReasonLineItemAdded)
	})
	if err != nil {
		return nil, fmt.Errorf("addSalesOrderLineItem: %w", err)
	}
	return &lineItem, nil
}
