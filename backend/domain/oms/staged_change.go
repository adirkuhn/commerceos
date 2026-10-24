package oms

import "time"

// ChangeAction is an order change staff record while working an order --
// replacing a line, adding an item (freebie or paid), or refunding part of a
// line. See StagedChange.
type ChangeAction string

const (
	ChangeSubstitute ChangeAction = "substitute"
	ChangeFreebie    ChangeAction = "freebie"
	ChangeNewItem    ChangeAction = "new_item"
	ChangeRefund     ChangeAction = "refund"
)

// StagedChange is an order change recorded during a picking or packing
// session and applied for real (stock, line items, refunds) only when the
// session completes -- resetting the session discards it. Both PickScan and
// PackScan rows carry changes; this is their common shape, so one applier
// (application/oms orderChangeApplier) serves both.
type StagedChange struct {
	ID                 string
	LineItemID         *string
	SKUID              string
	Action             ChangeAction
	Quantity           int64
	OriginalQtyCovered *int64
	ReturnToStock      bool
	PriceMinorUnits    *int64
	Reason             string
	ScannedAt          time.Time
	// ScannedBy is the lm2 UserID who recorded the change.
	ScannedBy  string
	ReplayedAt *time.Time
}

// Change returns the order change this scan records -- false for a plain
// pick, which changes nothing.
func (s PickScan) Change() (StagedChange, bool) {
	switch s.ActionType {
	case PickScanActionSubstitute, PickScanActionFreebie, PickScanActionNewItem, PickScanActionRefund:
	default:
		return StagedChange{}, false
	}
	return StagedChange{
		ID: s.ID, LineItemID: s.LineItemID, SKUID: s.SKUID, Action: ChangeAction(s.ActionType), Quantity: s.Quantity,
		OriginalQtyCovered: s.OriginalQtyCovered, ReturnToStock: s.ReturnToStock, PriceMinorUnits: s.PriceMinorUnits,
		Reason: s.Reason, ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy, ReplayedAt: s.ReplayedAt,
	}, true
}
