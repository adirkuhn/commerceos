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
