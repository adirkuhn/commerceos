package shipping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/shipping"
)

func TestCancelOutcome_Note(t *testing.T) {
	assert.Equal(t, "Shipment deleted on MyParcel.", shipping.CancelOutcome{DeletedOnCarrier: true}.Note())
	assert.Equal(t,
		"MyParcel could not delete this shipment (label already printed). It has been cancelled in CommerceOS so a corrected shipment can be created, but the old shipment/label may still exist on MyParcel's own side.",
		shipping.CancelOutcome{CarrierError: "label already printed"}.Note())
}

func TestCancelOutcome_NoteForALabelTheCarrierCantDelete(t *testing.T) {
	assert.Equal(t,
		"The carrier can't delete a pre-announced label. It costs nothing as long as it's never scanned and disappears after 7 days -- don't hand over the old label. The shipment is cancelled in CommerceOS so a corrected one can be created.",
		shipping.CancelOutcome{LabelKept: true}.Note())
}
