package oms

// IsPickable reports whether an order in this status may still be
// physically collected from the shelves -- the statuses the picking queue
// offers, and the ones packing accepts for an order that skipped the
// picking module entirely (see SalesOrder.CanStartPacking).
func (s SalesOrderStatus) IsPickable() bool {
	switch s {
	case SalesOrderStatusPending, SalesOrderStatusProcessing, SalesOrderStatusOnHold:
		return true
	default:
		return false
	}
}

// PickableStatuses lists every status IsPickable accepts -- for callers
// that need the set itself (a repository's SQL filter), so the rule lives
// in one place.
func PickableStatuses() []SalesOrderStatus {
	return []SalesOrderStatus{SalesOrderStatusPending, SalesOrderStatusProcessing, SalesOrderStatusOnHold}
}

// CanStartPacking decides whether a packing session may start for this
// order. Picking is optional: an order may go straight to packing (items
// collected off a printed pick list, then scanned during packing) as long as
// it's still in a pickable status. An open picking session on the order is
// the use case's call (StartOrResumePackingUseCase takes it over when its
// lock allows), not this rule's.
func (o SalesOrder) CanStartPacking() error {
	if o.Packed {
		return ErrOrderAlreadyPacked
	}
	if !o.Fulfilled && !o.Status.IsPickable() {
		return ErrOrderNotPackable
	}
	return nil
}

// CanStartPicking is CanStartPacking's picking-side counterpart.
// packingInProgress is whether the order has an in_progress packing session.
func (o SalesOrder) CanStartPicking(packingInProgress bool) error {
	if o.Fulfilled {
		return ErrOrderAlreadyFulfilled
	}
	if o.Packed {
		return ErrOrderAlreadyPacked
	}
	if packingInProgress {
		return ErrOrderPackingInProgress
	}
	return nil
}

// HasPendingPickChanges reports whether a picking session still holds
// order changes (substitute/refund/freebie/new item) not yet applied by
// CompletePickingUseCase's replay -- plain picks change nothing.
func HasPendingPickChanges(scans []PickScan) bool {
	for _, s := range scans {
		if s.ActionType != PickScanActionPick && s.ReplayedAt == nil {
			return true
		}
	}
	return false
}

// IsOpen reports whether the session can still be worked (scanned into,
// completed) -- in_progress or paused.
func (s PickingSession) IsOpen() bool {
	return s.Status == PickingSessionStatusInProgress || s.Status == PickingSessionStatusPaused
}
