package channels_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestBolTransporterCode_mapsEveryShippingCarrier(t *testing.T) {
	cases := []struct {
		name      string
		carrier   shippingDomain.Carrier
		recipient string
		want      string
	}{
		{"PostNL", shippingDomain.CarrierPostNL, "NL", "TNT"},
		{"PostNL to Belgium", shippingDomain.CarrierPostNL, "BE", "TNT"},
		{"bpost", shippingDomain.CarrierBpost, "BE", "BPOST_BE"},
		{"DPD to the Netherlands", shippingDomain.CarrierDPD, "NL", "DPD-NL"},
		{"DPD to Belgium", shippingDomain.CarrierDPD, "BE", "DPD-BE"},
		{"DPD to Belgium, lowercase country", shippingDomain.CarrierDPD, "be", "DPD-BE"},
		{"DPD with no recipient country", shippingDomain.CarrierDPD, "", "DPD-NL"},
		{"DHL For You", shippingDomain.CarrierDHLForYou, "NL", "DHLFORYOU"},
		{"DHL Parcel Connect", shippingDomain.CarrierDHLParcelConnect, "NL", "DHL"},
		{"DHL Europlus", shippingDomain.CarrierDHLEuroplus, "NL", "DHL"},
		{"UPS Standard", shippingDomain.CarrierUPSStandard, "NL", "UPS"},
		{"UPS Express Saver", shippingDomain.CarrierUPSExpressSaver, "NL", "UPS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := channelsDomain.BolTransporterCode(tc.carrier, tc.recipient)

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBolTransporterCode_unknownCarrier_returnsUnsupported(t *testing.T) {
	_, err := channelsDomain.BolTransporterCode(shippingDomain.Carrier(99), "NL")

	assert.ErrorIs(t, err, channelsDomain.ErrBolTransporterUnsupported)
}
