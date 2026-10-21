package shipping

import "fmt"

// CancelOutcome is what happened when a shipment was cancelled: whether the
// carrier itself deleted it, or rejected the delete (CarrierError, the
// carrier's own wording) and it was only cancelled in CommerceOS. Clients
// show it translated from these fields; Note is the English text.
type CancelOutcome struct {
	DeletedOnCarrier bool
	CarrierError     string
	// LabelKept: the carrier has no delete for a pre-announced label
	// (PostNL). Not a refusal -- the label costs nothing unless scanned.
	LabelKept bool
}

// Note is the outcome in English, for logs, events and older clients.
func (o CancelOutcome) Note() string {
	if o.DeletedOnCarrier {
		return "Shipment deleted on MyParcel."
	}
	if o.LabelKept {
		return "The carrier can't delete a pre-announced label. It costs nothing as long as it's never scanned and disappears after 7 days -- don't hand over the old label. The shipment is cancelled in CommerceOS so a corrected one can be created."
	}
	return fmt.Sprintf(
		"MyParcel could not delete this shipment (%s). It has been cancelled in CommerceOS so a corrected shipment can be created, but the old shipment/label may still exist on MyParcel's own side.",
		o.CarrierError,
	)
}
