package shipping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestMyParcelStatusLabel_knownCodes(t *testing.T) {
	cases := map[int]string{
		1:  "Pending - concept",
		3:  "Enroute - handed to carrier",
		7:  "Delivered - at recipient",
		16: "Expired",
		17: "Cancelled",
		19: "Delivered - at agreed location",
		30: "Inactive - concept",
		38: "Inactive - delivered - package picked up",
	}
	for code, want := range cases {
		assert.Equal(t, want, shippingDomain.MyParcelStatusLabel(code))
	}
}

func TestMyParcelStatusLabel_unknownCodeFallsBackToRawCode(t *testing.T) {
	assert.Equal(t, "Unknown status (25)", shippingDomain.MyParcelStatusLabel(25))
	assert.Equal(t, "Unknown status (0)", shippingDomain.MyParcelStatusLabel(0))
}
