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

// CorrectSupplierCostInput is a cost correction referencing a purchase
// order line (pms.sku.inventory_cost_changed, reason correction).
type CorrectSupplierCostInput struct {
	EventID                   string
	SKUID                     string
	PurchaseOrderLineItemID   string
	AdjustmentValueMinorUnits int64
	At                        time.Time
}

// CorrectSupplierCostUseCase reprices a received purchase order line in the
// SKU's cost history by supplier (COM-11) -- e.g. a supplier invoice that
// corrects what the line cost: the line's quantity out at its old cost,
// back in at the new one (procurementDomain.CorrectionEntries). History is
// never edited. If that line is the SKU's last receipt, the cost price
// follows through procurement.supplier_cost.recorded. Idempotent: rows are
// keyed by the event. A line with nothing of this SKU standing is a no-op.
type CorrectSupplierCostUseCase struct {
	db     *gorm.DB
	costs  procurementDomain.SupplierCostRepository
	outbox eventrecord.Enqueuer
}

func NewCorrectSupplierCostUseCase(db *gorm.DB, costs procurementDomain.SupplierCostRepository, outbox eventrecord.Enqueuer) *CorrectSupplierCostUseCase {
	return &CorrectSupplierCostUseCase{db: db, costs: costs, outbox: outbox}
}

func (uc *CorrectSupplierCostUseCase) Execute(ctx context.Context, in CorrectSupplierCostInput) error {
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
		pair, ok := procurementDomain.CorrectionEntries(line, in.AdjustmentValueMinorUnits, in.At)
		if !ok {
			return nil
		}
		pair[0].SourceKey = "correction:" + in.EventID + ":out"
		pair[1].SourceKey = "correction:" + in.EventID + ":in"
		if err := uc.costs.Append(ctx, pair); err != nil {
			return err
		}
		return recordSupplierCostRecorded(ctx, uc.outbox, pair[1])
	})
	if err != nil {
		return fmt.Errorf("correctSupplierCost: %w", err)
	}
	return nil
}
