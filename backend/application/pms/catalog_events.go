package pms

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// Why a SKU's cost price changed -- data.reason of pms.sku.cost_price_changed.
const (
	CostPriceChangeReasonReceipt         = "receipt"
	CostPriceChangeReasonCorrection      = "correction"
	CostPriceChangeReasonManual          = "manual"
	CostPriceChangeReasonOverrideCleared = "override_cleared"
	CostPriceChangeReasonSet             = "set"
	CostPriceChangeReasonCaseLink        = "case_link"
	CostPriceChangeReasonMerge           = "merge"
)

// recordCostPriceChanged records a SKU's cost price change: before is the
// SKU as it was, after the cost and source it has now.
func recordCostPriceChanged(ctx context.Context, outbox eventrecord.Enqueuer, before pmsDomain.SKU, afterCost *int64,
	afterSource pmsDomain.SKUCostPriceSource, reason, referenceType, referenceID string) error {
	data := eventrecord.Data{"reason": reason}
	if referenceType != "" {
		data["reference_type"] = referenceType
	}
	if referenceID != "" {
		data["reference_id"] = referenceID
	}
	return eventrecord.Record(ctx, outbox, events.SKUCostPriceChanged, before.ID, data, eventrecord.NewChanges().
		Set("cost_price_minor_units", before.CostPriceMinorUnits, afterCost).
		Set("cost_price_source", string(before.CostPriceSource), string(afterSource)))
}

// recordInventoryCostChanged records a change to a SKU's inventory cost
// (moving weighted average of the stock on hand); before is the SKU as it
// was. reason is one of the CostPriceChangeReason* values.
func recordInventoryCostChanged(ctx context.Context, outbox eventrecord.Enqueuer, before pmsDomain.SKU, after int64,
	reason, referenceType, referenceID string) error {
	data := eventrecord.Data{"reason": reason}
	if referenceType != "" {
		data["reference_type"] = referenceType
	}
	if referenceID != "" {
		data["reference_id"] = referenceID
	}
	return eventrecord.Record(ctx, outbox, events.SKUInventoryCostChanged, before.ID, data, eventrecord.NewChanges().
		Set("inventory_cost_minor_units", before.InventoryCostMinorUnits, &after))
}
