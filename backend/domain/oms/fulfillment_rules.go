package oms

// IsPickable reports whether an order in this status may still be
// physically collected from the shelves -- the statuses the picking queue
// offers, and the ones packing accepts for an order that skipped the
// picking module entirely (see SalesOrder.CanStartPacking). Pending is
// excluded: it's WooCommerce's "Pending payment" -- no payment received
// yet, so nothing is shipped until it moves to Processing. OnHold stays
// pickable: stock is already reduced and the store confirms the payment
// itself (e.g. a bank transfer).
func (s SalesOrderStatus) IsPickable() bool {
	switch s {
	case SalesOrderStatusProcessing, SalesOrderStatusOnHold:
		return true
	default:
		return false
	}
}

// PickableStatuses lists every status IsPickable accepts -- for callers
// that need the set itself (a repository's SQL filter), so the rule lives
// in one place.
func PickableStatuses() []SalesOrderStatus {
	return []SalesOrderStatus{SalesOrderStatusProcessing, SalesOrderStatusOnHold}
}

// IsFulfillable reports whether orders from this source go through picking
// and packing -- a POS sale is handed over at the counter, there's nothing
// to pick, pack or ship.
func (s SalesOrderSource) IsFulfillable() bool {
	return s != SalesOrderSourcePOS
}

// source is where the order came from (its origin, COM-15) -- passed in,
// the order itself holds no integration data.
//
// CanStartPacking decides whether a packing session may start for this
// order: not packed, not a POS sale, and still pending/processing/on-hold --
// picked or not (picking is optional). Status, not Fulfilled, is what says
// an order hasn't left yet: Fulfilled is also set automatically when an
// order completes or the carrier confirms a shipment, and those must never
// be packed. An open picking session on the order is the use case's call
// (StartOrResumePackingUseCase takes it over when its lock allows), not
// this rule's.
func (o SalesOrder) CanStartPacking(source SalesOrderSource) error {
	if o.Packed {
		return ErrOrderAlreadyPacked
	}
	if !source.IsFulfillable() {
		return ErrPOSOrderNotFulfillable
	}
	if !o.Status.IsPickable() {
		return ErrOrderNotPackable
	}
	return nil
}

// CanStartPicking is CanStartPacking's picking-side counterpart.
// packingInProgress is whether the order has an in_progress packing session.
func (o SalesOrder) CanStartPicking(source SalesOrderSource, packingInProgress bool) error {
	if !source.IsFulfillable() {
		return ErrPOSOrderNotFulfillable
	}
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
func (o SalesOrder) CanMarkPicked(source SalesOrderSource, packingInProgress bool) error {
	if err := o.CanStartPicking(source, packingInProgress); err != nil {
		return err
	}
	if !o.Status.IsPickable() {
		return ErrOrderNotPickable
	}
	return nil
}
