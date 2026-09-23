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

// PackScan is one line item's worth of units placed into the box -- unlike
// PickScan there's no staged/replayed distinction (see PickScan's doc
// comment for why picking needs that): packing never touches stock or the
// order's line items, it's purely a box-contents record, so a row is simply
// real the moment it's created. LineItemID always references a real,
// already-SKU-linked SalesOrderLineItem -- by the time an order reaches
// packing, ScanPickItemUseCase's auto-link step has already resolved every
// line to a catalog SKU, so packing never needs picking's
// unmatched-line-item fallback.
type PackScan struct {
	ID        string
	TenantID  string
	SessionID string
	// LineItemID is the SalesOrderLineItem this scan packs units of.
	LineItemID string
	// SKUID is that line item's own SKU, denormalized here the same way
	// PickScan denormalizes it, so a scan's contents can be read without a
	// second lookup.
	SKUID     string
	Quantity  int64
	ScannedAt time.Time
	ScannedBy string
}
