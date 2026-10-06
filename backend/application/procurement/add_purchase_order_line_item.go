package procurement

import (
	"context"
	"fmt"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type AddPurchaseOrderLineItemInput struct {
	PurchaseOrderID string
	SKUID           *string
	RawSKU          string
	RawDescription  string
	Quantity        int64
	// UnitsPerPurchasedItem defaults to 1 (an ordinary line) when zero --
	// see PurchaseOrderLineItem's doc comment for the case/box scenario
	// that needs it set higher.
	UnitsPerPurchasedItem int64
	UnitCostMinorUnits    int64
	// LineType defaults to product (see LineItemType.IsProduct) when empty.
	LineType procurementDomain.LineItemType
}

type AddPurchaseOrderLineItemUseCase struct {
	db           *gorm.DB
	lineItemRepo procurementDomain.PurchaseOrderLineItemRepository
	outbox       eventrecord.Enqueuer
}

func NewAddPurchaseOrderLineItemUseCase(db *gorm.DB, lineItemRepo procurementDomain.PurchaseOrderLineItemRepository, outbox eventrecord.Enqueuer) *AddPurchaseOrderLineItemUseCase {
	return &AddPurchaseOrderLineItemUseCase{db: db, lineItemRepo: lineItemRepo, outbox: outbox}
}

func (uc *AddPurchaseOrderLineItemUseCase) Execute(ctx context.Context, in AddPurchaseOrderLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	if in.Quantity <= 0 {
		return nil, fmt.Errorf("addPurchaseOrderLineItem: %w", procurementDomain.ErrInvalidQuantity)
	}
	unitsPerPurchasedItem := in.UnitsPerPurchasedItem
	if unitsPerPurchasedItem == 0 {
		unitsPerPurchasedItem = 1
	} else if unitsPerPurchasedItem < 0 {
		return nil, fmt.Errorf("addPurchaseOrderLineItem: %w", procurementDomain.ErrInvalidUnitsPerPurchasedItem)
	}

	var li procurementDomain.PurchaseOrderLineItem
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.lineItemRepo.ListByPurchaseOrder(ctx, in.PurchaseOrderID)
		if err != nil {
			return err
		}
		// Append after whatever's already there -- max+1 rather than
		// len(existing) so a manually-added item still lands last even if an
		// earlier one was deleted, leaving a gap in the sequence.
		var nextPosition int64
		for _, item := range existing {
			if item.Position >= nextPosition {
				nextPosition = item.Position + 1
			}
		}

		now := time.Now()
		li = procurementDomain.PurchaseOrderLineItem{
			TenantID: tenant.IDFromContext(ctx), PurchaseOrderID: in.PurchaseOrderID, SKUID: in.SKUID, RawSKU: in.RawSKU,
			RawDescription: in.RawDescription, Position: nextPosition, Quantity: in.Quantity, UnitCostMinorUnits: in.UnitCostMinorUnits,
			UnitsPerPurchasedItem: unitsPerPurchasedItem,
			// A human is adding this line item directly -- no ambiguity to
			// review, confirmed immediately.
			LineType: in.LineType, LineTypeConfirmed: true,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := uc.lineItemRepo.Create(ctx, &li); err != nil {
			return err
		}
		return recordLineItemAdded(ctx, uc.outbox, li, LineItemSourceManual)
	})
	if err != nil {
		return nil, fmt.Errorf("addPurchaseOrderLineItem: %w", err)
	}
	return &li, nil
}
