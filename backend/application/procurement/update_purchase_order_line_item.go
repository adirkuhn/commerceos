package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type UpdatePurchaseOrderLineItemInput struct {
	ID             string
	SKUID          *string
	RawSKU         string
	RawDescription string
	Quantity       int64
	// UnitsPerPurchasedItem defaults to 1 (an ordinary line) when zero --
	// see PurchaseOrderLineItem's doc comment for the case/box scenario
	// that needs it set higher.
	UnitsPerPurchasedItem int64
	UnitCostMinorUnits    int64
	// LineType defaults to product (see LineItemType.IsProduct) when empty.
	LineType procurementDomain.LineItemType
}

type UpdatePurchaseOrderLineItemUseCase struct {
	db           *gorm.DB
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	outbox       eventrecord.Enqueuer
}

func NewUpdatePurchaseOrderLineItemUseCase(db *gorm.DB, lineItemRepo procurementDomain.PurchaseOrderLineItemRepository, outbox eventrecord.Enqueuer) *UpdatePurchaseOrderLineItemUseCase {
	return &UpdatePurchaseOrderLineItemUseCase{db: db, lineItemRepo: lineItemRepo, outbox: outbox}
}

func (uc *UpdatePurchaseOrderLineItemUseCase) Execute(ctx context.Context, in UpdatePurchaseOrderLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("updatePurchaseOrderLineItem: %w", procurementDomain.ErrInvalidQuantity)
	}
	unitsPerPurchasedItem := in.UnitsPerPurchasedItem
	if unitsPerPurchasedItem == 0 {
		unitsPerPurchasedItem = 1
	} else if unitsPerPurchasedItem < 0 {
		return nil, fmt.Errorf("updatePurchaseOrderLineItem: %w", procurementDomain.ErrInvalidUnitsPerPurchasedItem)
	}

	var li procurementDomain.PurchaseOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.lineItemRepo.FindByID(ctx, in.ID)
		if err != nil {
			return err
		}
		before := *existing
		existing.SKUID = in.SKUID
		existing.RawSKU = in.RawSKU
		existing.RawDescription = in.RawDescription
		existing.Quantity = in.Quantity
		existing.UnitsPerPurchasedItem = unitsPerPurchasedItem
		existing.UnitCostMinorUnits = in.UnitCostMinorUnits
		existing.LineType = in.LineType
		// An explicit save through this form is a human touch -- overrides
		// any unreviewed alias guess this item may have carried in with,
		// and confirms the (possibly AI-suggested) line type.
		existing.SKUAutoLinkedFromAlias = false
		existing.LineTypeConfirmed = true
		existing.UpdatedAt = time.Now()
		if err := uc.lineItemRepo.Update(ctx, existing); err != nil {
			return err
		}
		li = *existing
		return eventrecord.Record(ctx, uc.outbox, events.PurchaseOrderLineItemUpdated, li.ID,
			eventrecord.Data{"purchase_order_id": li.PurchaseOrderID}, eventrecord.NewChanges().
				Set("sku_id", before.SKUID, li.SKUID).
				Set("raw_sku", before.RawSKU, li.RawSKU).
				Set("raw_description", before.RawDescription, li.RawDescription).
				Set("quantity", before.Quantity, li.Quantity).
				Set("units_per_purchased_item", before.UnitsPerPurchasedItem, li.UnitsPerPurchasedItem).
				Set("unit_cost_minor_units", before.UnitCostMinorUnits, li.UnitCostMinorUnits).
				Set("line_type", string(before.LineType), string(li.LineType)))
	})
	if err != nil {
		return nil, fmt.Errorf("updatePurchaseOrderLineItem: %w", err)
	}
	return &li, nil
}
