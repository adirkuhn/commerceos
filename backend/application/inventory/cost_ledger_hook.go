package inventory

import (
	"context"
	"time"
)

// CostLedgerMovementInput mirrors application/pms's
// RecordCostLedgerMovementInput -- a local copy, not an import (that
// package already imports application/inventory -- see
// application/pms/recalculate_cost_prices.go's own doc comment on that
// exact cycle), wired via an adapter in the composition root instead. See
// SKUCostReceiptInput's doc comment for the same pattern.
type CostLedgerMovementInput struct {
	SKUID         string
	QuantityDelta int64
	// SourceType is one of pms's SKUCostPriceLedgerSource* string values
	// ("damaged", "lost", "adjustment") -- kept as a plain string here so
	// this package doesn't need to import pms's domain package either.
	SourceType    string
	ReferenceType string
	ReferenceID   string
	OccurredAt    time.Time
}

// costLedgerRecorder is the narrow slice of application/pms's
// RecordCostLedgerMovementUseCase this package needs (ISP) -- kept local
// rather than imported, same cross-domain-avoidance reasoning as
// skuCostUpdaterForReceiveStock.
type costLedgerRecorder interface {
	Execute(ctx context.Context, in CostLedgerMovementInput) error
}
