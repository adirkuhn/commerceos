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
// it's still in a pickable status. pickingInProgress is whether the order
// has an in_progress/paused picking session -- packing then has to wait, so
// two people never work the same order through both flows at once.
func (o SalesOrder) CanStartPacking(pickingInProgress bool) error {
	if o.Packed {
		return ErrOrderAlreadyPacked
	}
	if pickingInProgress {
		return ErrOrderPickingInProgress
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
