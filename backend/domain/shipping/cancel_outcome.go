package shipping

import "fmt"

// CancelOutcome is what happened when a shipment was cancelled: whether the
// carrier itself deleted it, or rejected the delete (CarrierError, the
// carrier's own wording) and it was only cancelled in CommerceOS. Clients
// show it translated from these fields; Note is the English text.
type CancelOutcome struct {
	DeletedOnCarrier bool
	CarrierError     string
}

// Note is the outcome in English, for logs, events and older clients.
func (o CancelOutcome) Note() string {
	if o.DeletedOnCarrier {
		return "Shipment deleted on MyParcel."
	}
	return fmt.Sprintf(
		"MyParcel could not delete this shipment (%s). It has been cancelled in CommerceOS so a corrected shipment can be created, but the old shipment/label may still exist on MyParcel's own side.",
		o.CarrierError,
	)
}
