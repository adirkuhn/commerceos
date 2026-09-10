package shipping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestIsCarrierConfirmed(t *testing.T) {
	cases := map[string]bool{
		"":                  false,
		"registered":        false,
		"handed_to_carrier": true,
		"sorting":           true,
		"distribution":      true,
		"delivered":         true,
	}
	for statusMain, want := range cases {
		assert.Equal(t, want, shippingDomain.IsCarrierConfirmed(statusMain), "statusMain=%q", statusMain)
	}
}
