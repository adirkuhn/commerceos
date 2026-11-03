package pms

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// LastReceiptCost is the last paid goods receipt still standing in a SKU's
// cost history by supplier -- a local mirror of procurement's
// LastSupplierCost (application/procurement imports this package, so the
// reverse import would cycle; the composition root adapts it).
type LastReceiptCost struct {
	LandedUnitCostMinorUnits int64
	PurchaseOrderLineItemID  string
}

// lastReceiptCostLookup reads the SKU's last receipt from the supplier cost
// history (ISP). It opens its own transaction, so it is called before ours
// -- tenant.WithTenant must never nest.
type lastReceiptCostLookup interface {
	LastReceiptCost(ctx context.Context, skuID string) (*LastReceiptCost, error)
}

type skuCostPriceWriter interface {
	FindByID(ctx context.Context, skuID string) (*pmsDomain.SKU, error)
	UpdateCostPriceFromReceipt(ctx context.Context, skuID string, costPriceMinorUnits int64) error
}

// SyncCostPriceFromHistoryUseCase sets a SKU's cost price to the last
// receipt still standing in its cost history by supplier (COM-11): after a
// receipt is reversed (falls back to the one before), after a receipt is
// repriced by a correction, and as the daily reconciliation. A manual cost
// price wins; no paid receipt in the history changes nothing.
type SyncCostPriceFromHistoryUseCase struct {
	db     *gorm.DB
	skus   skuCostPriceWriter
	last   lastReceiptCostLookup
	outbox eventrecord.Enqueuer
}

func NewSyncCostPriceFromHistoryUseCase(db *gorm.DB, skus skuCostPriceWriter, last lastReceiptCostLookup, outbox eventrecord.Enqueuer) *SyncCostPriceFromHistoryUseCase {
	return &SyncCostPriceFromHistoryUseCase{db: db, skus: skus, last: last, outbox: outbox}
}

func (uc *SyncCostPriceFromHistoryUseCase) Execute(ctx context.Context, skuID string) error {
	last, err := uc.last.LastReceiptCost(ctx, skuID)
	if err != nil {
		return fmt.Errorf("syncCostPriceFromHistory: %w", err)
	}
	if last == nil {
		return nil
	}
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skus.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		return applyLastReceiptCost(ctx, uc.skus, uc.outbox, *sku, *last)
	})
	if err != nil {
		return fmt.Errorf("syncCostPriceFromHistory: %w", err)
	}
	return nil
}

// applyLastReceiptCost writes last as sku's cost price unless it is manual
// or already that, raising pms.sku.cost_price_changed.
func applyLastReceiptCost(ctx context.Context, skus skuCostPriceWriter, outbox eventrecord.Enqueuer, sku pmsDomain.SKU, last LastReceiptCost) error {
	if sku.CostPriceSource == pmsDomain.SKUCostPriceSourceManual {
		return nil
	}
	if sku.CostPriceMinorUnits != nil && *sku.CostPriceMinorUnits == last.LandedUnitCostMinorUnits {
		return nil
	}
	cost := last.LandedUnitCostMinorUnits
	if err := skus.UpdateCostPriceFromReceipt(ctx, sku.ID, cost); err != nil {
		return err
	}
	return recordCostPriceChanged(ctx, outbox, sku, &cost, pmsDomain.SKUCostPriceSourceComputed,
		CostPriceChangeReasonReceipt, ReceiptReferencePurchaseOrderLine, last.PurchaseOrderLineItemID)
}
