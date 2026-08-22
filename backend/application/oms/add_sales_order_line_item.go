package oms

import (
	"context"
	"fmt"
	"time"

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
// No tax computation -- SubtotalTaxMinorUnits/TotalTaxMinorUnits are left
// 0, a deliberate simplification (no tax-rate lookup happens mid-pick;
// staff can correct via the existing header-edit flow if needed).
// TotalMinorUnits is a cached column on SalesOrder, never derived from
// summed line items (unlike procurement.PurchaseOrder's), so this use case
// must explicitly bump it -- nothing else will.
//
// Does not touch stock -- same "line-item write is one transaction, stock
// movement is the caller's separate call" split used everywhere else in
// this package (see ReceivePOSSalesOrderUseCase's doc).
type AddSalesOrderLineItemUseCase struct {
	db             *gorm.DB
	salesOrderRepo omsDomain.SalesOrderRepository
	lineItemRepo   omsDomain.SalesOrderLineItemRepository
	skuRepo        skuLookupRepo
}

func NewAddSalesOrderLineItemUseCase(db *gorm.DB, salesOrderRepo omsDomain.SalesOrderRepository, lineItemRepo omsDomain.SalesOrderLineItemRepository, skuRepo skuLookupRepo) *AddSalesOrderLineItemUseCase {
	return &AddSalesOrderLineItemUseCase{db: db, salesOrderRepo: salesOrderRepo, lineItemRepo: lineItemRepo, skuRepo: skuRepo}
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

		total := in.PriceMinorUnits * in.Quantity
		skuID := in.SKUID
		lineItem = omsDomain.SalesOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), SalesOrderID: in.SalesOrderID, SKUID: &skuID,
			Name: in.Name, SKUCode: sku.SKUCode, Barcode: sku.Barcode, Quantity: in.Quantity,
			StockStatus: omsDomain.StockStatusDecremented,
			// This use case's caller (CompletePickingUseCase) records the
			// real stock movement itself right after this returns -- see
			// its doc comment for why the two aren't one transaction.
			PriceMinorUnits: in.PriceMinorUnits, SubtotalMinorUnits: total, TotalMinorUnits: total,
			TaxClassID: sku.TaxClassID, CreatedAt: time.Now(),
		}
		if err := uc.lineItemRepo.Create(ctx, &lineItem); err != nil {
			return err
		}

		return uc.salesOrderRepo.Update(ctx, order.ID, map[string]any{
			"total_minor_units": order.TotalMinorUnits + total,
			"updated_at":        time.Now(),
		})
	})
	if err != nil {
		return nil, fmt.Errorf("addSalesOrderLineItem: %w", err)
	}
	return &lineItem, nil
}
