package procurement

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ReverseGoodsReceivedCostInput is one procurement.goods.receipt_corrected.
// Quantity is the correction's (negative: stock taken back).
type ReverseGoodsReceivedCostInput struct {
	EventID                 string
	SKUID                   string
	PurchaseOrderLineItemID string
	Quantity                int64
	At                      time.Time
}

// ReverseGoodsReceivedCostUseCase takes received stock back off a purchase
// order line in the SKU's cost history by supplier (COM-11) -- a reversal
// row at the line's cost (procurementDomain.ReversalEntry); history is
// never edited. A fully reversed line drops out, so the cost price falls
// back to the receipt before it (through procurement.supplier_cost.recorded).
// Idempotent: the row is keyed by the event. Nothing standing is a no-op.
type ReverseGoodsReceivedCostUseCase struct {
	db     *gorm.DB
	costs  procurementDomain.SupplierCostRepository
	outbox eventrecord.Enqueuer
}

func NewReverseGoodsReceivedCostUseCase(db *gorm.DB, costs procurementDomain.SupplierCostRepository, outbox eventrecord.Enqueuer) *ReverseGoodsReceivedCostUseCase {
	return &ReverseGoodsReceivedCostUseCase{db: db, costs: costs, outbox: outbox}
}

func (uc *ReverseGoodsReceivedCostUseCase) Execute(ctx context.Context, in ReverseGoodsReceivedCostInput) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		all, err := uc.costs.ListByLine(ctx, in.PurchaseOrderLineItemID)
		if err != nil {
			return err
		}
		var line []procurementDomain.SupplierCostEntry
		for _, e := range all {
			if e.SKUID == in.SKUID {
				line = append(line, e)
			}
		}
		e, ok := procurementDomain.ReversalEntry(line, -in.Quantity, in.At)
		if !ok {
			return nil
		}
		e.SourceKey = "reversal:" + in.EventID
		if err := uc.costs.Append(ctx, []procurementDomain.SupplierCostEntry{e}); err != nil {
			return err
		}
		return recordSupplierCostRecorded(ctx, uc.outbox, e)
	})
	if err != nil {
		return fmt.Errorf("reverseGoodsReceivedCost: %w", err)
	}
	return nil
}
