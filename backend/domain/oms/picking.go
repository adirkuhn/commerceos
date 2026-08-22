package oms

import "time"

type PickingSessionStatus string

const (
	PickingSessionStatusInProgress PickingSessionStatus = "in_progress"
	PickingSessionStatusPaused     PickingSessionStatus = "paused"
	PickingSessionStatusCompleted  PickingSessionStatus = "completed"
)

// PickingSession tracks one scanner-app session working a sales order on
// the warehouse floor -- StartOrResumePickingUseCase reuses an existing
// in_progress/paused session for the order rather than creating a new one,
// same "pausing one order to work another is just start/resume" mechanic
// as procurement.Booking. Unlike Booking, there's no warehouse-assignment
// step: SalesOrder.WarehouseID is always set (never nil), so a session
// never needs to backfill one before starting.
type PickingSession struct {
	ID           string
	TenantID     string
	SalesOrderID string
	Status       PickingSessionStatus
	StartedAt    time.Time
	PausedAt     *time.Time
	CompletedAt  *time.Time
	// CompletedBy is the paired session's UserID (see
	// iam.ScannerPairingCode) -- plain string, not a FK, same convention as
	// every other *By field in this codebase (lm2 user IDs aren't UUIDs).
	CompletedBy string
}

// PickScanActionType classifies what a staged PickScan row will do once
// the picking session is completed.
type PickScanActionType string

const (
	// PickScanActionPick confirms one of the order's existing line items
	// was picked -- no stock effect at replay time (stock was already
	// decremented the moment the order's Status entered a stock-reducing
	// state, see SalesOrderStatus.ReducesStock's doc). Purely a
	// confirmation/audit row.
	PickScanActionPick PickScanActionType = "pick"
	// PickScanActionSubstitute swaps an entire original line item's stock
	// for a different SKU -- LineItemID is the ORIGINAL line, SKUID is the
	// SUBSTITUTE. At replay: reverse the original line's full decrement,
	// decrement the substitute for real. The original line item row itself
	// is never mutated -- see CompletePickingUseCase's doc. Whole-line-item
	// quantity only, no partial substitution -- this codebase has no
	// partial-reversal primitive to build on.
	PickScanActionSubstitute PickScanActionType = "substitute"
	// PickScanActionFreebie adds a brand-new, real SalesOrderLineItem
	// priced at 0 and decrements its stock for real. LineItemID is nil
	// (nothing to reference yet); SKUID is the item being added.
	PickScanActionFreebie PickScanActionType = "freebie"
	// PickScanActionNewItem is Freebie's non-zero-priced sibling -- same
	// replay effect, PriceMinorUnits is picker-entered instead of forced 0.
	PickScanActionNewItem PickScanActionType = "new_item"
)

// PickScan is a STAGED action from the scanner app -- see BookingScan's
// doc comment for the "nothing is real until Complete replays it"
// rationale, which applies identically here: a session in progress can
// still be paused, resumed, or have the picker switch to another order,
// so nothing should touch real inventory or the order's line items until
// CompletePickingUseCase replays every row (in ScannedAt order) when the
// session finishes.
type PickScan struct {
	ID        string
	TenantID  string
	SessionID string
	// LineItemID is the SalesOrderLineItem this scan targets: the line
	// being picked (Pick), the ORIGINAL line being replaced (Substitute),
	// or nil (Freebie/NewItem -- nothing on the order yet).
	LineItemID *string
	// SKUID is the line's own SKU (Pick), the SUBSTITUTE sku (Substitute),
	// or the new SKU being added (Freebie/NewItem).
	SKUID      string
	ActionType PickScanActionType
	// Quantity is always a positive magnitude -- for Substitute, this must
	// equal the original line item's own Quantity (whole-line swap only,
	// see PickScanActionSubstitute's doc).
	Quantity int64
	// PriceMinorUnits is nil for Pick/Substitute (no real line item
	// created), 0 for Freebie, picker-entered for NewItem.
	PriceMinorUnits *int64
	// Reason is an optional user comment -- most useful on a substitute row.
	Reason    string
	ScannedAt time.Time
	ScannedBy string
	// ReplayedAt is nil until CompletePickingUseCase's replay actually
	// applies this row for real, then set once and never cleared -- see
	// BookingScan's identical field for the crash-safety rationale.
	ReplayedAt *time.Time
}
