package oms

import "time"

type PackingSessionStatus string

const (
	PackingSessionStatusInProgress PackingSessionStatus = "in_progress"
	PackingSessionStatusCompleted  PackingSessionStatus = "completed"
)

// PackingSession tracks one warehouse-floor session boxing up an already-
// picked sales order and getting it ready to ship -- the second half of
// fulfillment, carved out from PickingSession so a different person can pick
// an order and hand it off to someone else to pack (or the same person can
// go straight from Finish Picking into packing in one flow). Only startable
// once the order's own Fulfilled flag is true (see
// StartOrResumePackingUseCase) -- packing has nothing to do until picking
// says the order is done. Unlike PickingSession there's no Paused state:
// packing is a single short box-and-label pass, not something staff leave
// mid-way for hours the way a big multi-line pick sometimes is.
type PackingSession struct {
	ID           string
	TenantID     string
	SalesOrderID string
	Status       PackingSessionStatus
	StartedAt    time.Time
	CompletedAt  *time.Time
	// CompletedBy is the caller's lm2 UserID, plain string not a FK, same
	// convention as every other *By field in this codebase.
	CompletedBy string
	// LockedBy/LockedAt mirror PickingSession's own fields exactly -- see
	// that type's doc comment. StartOrResumePackingUseCase/
	// TouchPackingSessionUseCase enforce the same SessionLock rules so two
	// people can't pack the same order at once.
	LockedBy    *string
	LockedAt    *time.Time
	LockedSince *time.Time
}

// Lock is the session's lock state, for SessionLock's rules.
func (s PackingSession) Lock() SessionLock {
	return SessionLock{HolderID: s.LockedBy, HeartbeatAt: s.LockedAt, Since: s.LockedSince}
}

// PackScanActionType classifies a PackScan row. A plain pack only records
// box contents and is real the moment it's created; every other action is
// an order change (see StagedChange), staged until CompletePackingUseCase
// applies it -- packing is where substitutes, extra items and refunds happen
// now that picking is optional.
type PackScanActionType string

const (
	PackScanActionPack       PackScanActionType = "pack"
	PackScanActionSubstitute PackScanActionType = "substitute"
	PackScanActionFreebie    PackScanActionType = "freebie"
	PackScanActionNewItem    PackScanActionType = "new_item"
	PackScanActionRefund     PackScanActionType = "refund"
)

// PackScan is one row recorded while packing an order: units put in the box
// (pack), or an order change (substitute, freebie, new item, refund). Field
// meanings for changes mirror PickScan's exactly.
type PackScan struct {
	ID        string
	TenantID  string
	SessionID string
	// LineItemID is the SalesOrderLineItem this row packs, substitutes or
	// refunds -- nil for a freebie/new item, which becomes a new line only
	// when the session completes.
	LineItemID *string
	// SKUID is the SKU physically packed (the substitute's own SKU for a
	// substitute), denormalized so a row reads without a second lookup.
	SKUID      string
	ActionType PackScanActionType
	Quantity   int64
	// OriginalQtyCovered is how much of the original line a substitute
	// replaces -- see PickScan.OriginalQtyCovered.
	OriginalQtyCovered *int64
	// ReturnToStock: a substitute puts the original units back on the shelf.
	ReturnToStock bool
	// PriceMinorUnits is a new item's per-unit price; nil for a freebie.
	PriceMinorUnits *int64
	Reason          string
	ScannedAt       time.Time
	ScannedBy       string
	// ReplayedAt is when CompletePackingUseCase applied this change -- nil
	// for a plain pack, or a change not applied yet.
	ReplayedAt *time.Time
}

// Change returns the order change this row records -- false for a plain
// pack, which changes nothing.
func (s PackScan) Change() (StagedChange, bool) {
	switch s.ActionType {
	case PackScanActionSubstitute, PackScanActionFreebie, PackScanActionNewItem, PackScanActionRefund:
	default:
		return StagedChange{}, false
	}
	return StagedChange{
		ID: s.ID, LineItemID: s.LineItemID, SKUID: s.SKUID, Action: ChangeAction(s.ActionType), Quantity: s.Quantity,
		OriginalQtyCovered: s.OriginalQtyCovered, ReturnToStock: s.ReturnToStock, PriceMinorUnits: s.PriceMinorUnits,
		Reason: s.Reason, ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy, ReplayedAt: s.ReplayedAt,
	}, true
}

// Covers is how much of which line item this row accounts for: packed units,
// the original quantity a substitute replaces, or refunded units. Freebies
// and new items cover no existing line (0).
func (s PackScan) Covers() (lineItemID string, qty int64) {
	if s.LineItemID == nil {
		return "", 0
	}
	switch s.ActionType {
	case PackScanActionPack, PackScanActionRefund:
		return *s.LineItemID, s.Quantity
	case PackScanActionSubstitute:
		if s.OriginalQtyCovered == nil {
			return *s.LineItemID, 0
		}
		return *s.LineItemID, *s.OriginalQtyCovered
	default:
		return "", 0
	}
}
