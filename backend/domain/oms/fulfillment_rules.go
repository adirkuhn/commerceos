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

// IsPackable reports whether orders from this source go through packing --
// a POS sale is handed over at the counter, there's nothing to pack or ship.
func (s SalesOrderSource) IsPackable() bool {
	return s != SalesOrderSourcePOS
}

// CanStartPacking decides whether a packing session may start for this
// order: not packed, not a POS sale, and still pending/processing/on-hold --
// picked or not (picking is optional). Status, not Fulfilled, is what says
// an order hasn't left yet: Fulfilled is also set automatically when an
// order completes or the carrier confirms a shipment, and those must never
// be packed. An open picking session on the order is the use case's call
// (StartOrResumePackingUseCase takes it over when its lock allows), not
// this rule's.
func (o SalesOrder) CanStartPacking() error {
	if o.Packed {
		return ErrOrderAlreadyPacked
	}
	if !o.Source.IsPackable() {
		return ErrPOSOrderNotPackable
	}
	if !o.Status.IsPickable() {
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

// CanMarkPicked decides whether the whole order may be marked picked in one
// action (the picking module's list mode): the same rules as starting a
// picking session, and the order must still be in a pickable status.
func (o SalesOrder) CanMarkPicked(packingInProgress bool) error {
	if err := o.CanStartPicking(packingInProgress); err != nil {
		return err
	}
	if !o.Status.IsPickable() {
		return ErrOrderNotPickable
	}
	return nil
}
