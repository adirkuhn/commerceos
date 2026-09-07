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
	// PickScanActionSubstitute swaps part (or all) of an original line
	// item's quantity for a different SKU -- LineItemID is the ORIGINAL
	// line, SKUID is the SUBSTITUTE. One original line can be covered by
	// several Substitute rows at once (e.g. 1 unit replaced by 3 of one
	// cheaper SKU and 4 of another) -- see SubstituteLineItemUseCase, the
	// only use case allowed to create these rows (ScanPickItemUseCase
	// rejects this action type). At replay: each row's own
	// OriginalQtyCovered is reversed from the original line's stock only if
	// its own ReturnToStock is true (summed once per line across all its
	// Substitute rows), then its own substitute SKU is decremented for
	// real. The original line item row itself is never mutated -- see
	// CompletePickingUseCase's doc.
	PickScanActionSubstitute PickScanActionType = "substitute"
	// PickScanActionFreebie adds a brand-new, real SalesOrderLineItem
	// priced at 0 and decrements its stock for real. LineItemID is nil
	// (nothing to reference yet); SKUID is the item being added.
	PickScanActionFreebie PickScanActionType = "freebie"
	// PickScanActionNewItem is Freebie's non-zero-priced sibling -- same
	// replay effect, PriceMinorUnits is picker-entered instead of forced 0.
	PickScanActionNewItem PickScanActionType = "new_item"
	// PickScanActionRefund stages refunding part or all of an existing line
	// item's quantity, when staff choose not to substitute it (see
	// RefundLineItemUseCase, the only use case allowed to create these
	// rows). LineItemID is the line being refunded, SKUID is that line's
	// own SKU (never a different one -- a refund never introduces a new
	// product). Quantity is the amount being refunded -- positive, at most
	// the line's own Quantity, need not equal it (see this doc for
	// Quantity's Refund case). At replay: if the order has a known
	// PaymentProvider/PaymentExternalID, CompletePickingUseCase attempts a
	// real refund through that provider for Quantity * the line's own
	// PriceMinorUnits; on success the order's Status becomes
	// PartiallyRefunded. Either no provider info or the API call itself
	// failing instead sets SalesOrder.NeedsManualRefundReview, leaving
	// Status untouched -- a refund never blocks picking completion. Either
	// way the line item's RefundedQuantity/RefundedAmountMinorUnits
	// accumulate and the line item row itself is never otherwise mutated.
	PickScanActionRefund PickScanActionType = "refund"
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
	// Quantity is always a positive magnitude -- for Substitute, this is
	// the SUBSTITUTE SKU's own quantity being added (caller-supplied via
	// SubstituteLineItemUseCase), not the original line's quantity. For
	// Refund, this is the quantity of the ORIGINAL line item being
	// refunded -- at most that line's own Quantity, need not equal it
	// (partial-by-quantity, see PickScanActionRefund's doc).
	Quantity int64
	// OriginalQtyCovered is set only for Substitute -- how much of the
	// original line item's own Quantity this particular row covers. Every
	// Substitute row for a given original line, summed, must equal that
	// line's Quantity exactly (enforced by SubstituteLineItemUseCase at
	// creation, not at replay). Nil for every other action type.
	OriginalQtyCovered *int64
	// ReturnToStock is set only for Substitute -- whether this row's own
	// OriginalQtyCovered should be added back to the original SKU's stock
	// at replay. Defaults false: a substitution normally happens because
	// the original stock figure was wrong (it isn't really there), so
	// nothing is added back unless staff explicitly say so. Ignored for
	// every other action type.
	ReturnToStock bool
	// PriceMinorUnits is nil for Pick/Substitute (no real line item
	// created), 0 for Freebie, picker-entered for NewItem.
	PriceMinorUnits *int64
	// Reason is an optional user comment on a Pick/Substitute row -- for
	// Freebie/NewItem it doubles as the new SalesOrderLineItem's Name
	// (picker-entered description of what was added), since PickScan has
	// no separate Name field of its own. May be empty either way -- the
	// line item still shows its SKUCode/Barcode.
	Reason    string
	ScannedAt time.Time
	ScannedBy string
	// ReplayedAt is nil until CompletePickingUseCase's replay actually
	// applies this row for real, then set once and never cleared -- see
	// BookingScan's identical field for the crash-safety rationale.
	ReplayedAt *time.Time
}
