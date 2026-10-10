package channels

import (
	"fmt"
	"github.com/adirkuhn/commerceos/domain/errs"
	"strings"

	"github.com/adirkuhn/commerceos/domain/shipping"
)

// ErrBolTransporterUnsupported means a shipment's carrier has no Bol
// transporter code, so Bol can't be told about it automatically -- the
// merchant has to confirm that order by hand in Bol's seller portal.
var ErrBolTransporterUnsupported = errs.New("BOL_TRANSPORTER_UNSUPPORTED", "carrier has no Bol transporter code")

// bolTransporterRule picks a Bol transporter code for one carrier. Most
// carriers map to a single code; DPD is the exception, with one code per
// receiving country.
type bolTransporterRule func(recipientCountry string) string

func fixedBolTransporter(code string) bolTransporterRule {
	return func(string) string { return code }
}

func dpdBolTransporter(recipientCountry string) string {
	if strings.EqualFold(recipientCountry, "BE") {
		return "DPD-BE"
	}
	return "DPD-NL"
}

// bolTransporterRules is Bol's own transporter code table (Retailer API,
// "Transporters") for every carrier shipping.Carrier knows. DHL Parcel
// Connect and Europlus have no service-specific entry there, so they use
// Bol's generic DHL (DHL eCommerce) code.
var bolTransporterRules = map[shipping.Carrier]bolTransporterRule{
	shipping.CarrierPostNL:           fixedBolTransporter("TNT"),
	shipping.CarrierBpost:            fixedBolTransporter("BPOST_BE"),
	shipping.CarrierDPD:              dpdBolTransporter,
	shipping.CarrierDHLForYou:        fixedBolTransporter("DHLFORYOU"),
	shipping.CarrierDHLParcelConnect: fixedBolTransporter("DHL"),
	shipping.CarrierDHLEuroplus:      fixedBolTransporter("DHL"),
	shipping.CarrierUPSStandard:      fixedBolTransporter("UPS"),
	shipping.CarrierUPSExpressSaver:  fixedBolTransporter("UPS"),
}

// BolTransporterCode returns the code Bol expects in a shipment's
// transport.transporterCode for a parcel sent with carrier to
// recipientCountry (ISO 3166-1 alpha-2).
func BolTransporterCode(carrier shipping.Carrier, recipientCountry string) (string, error) {
	rule, ok := bolTransporterRules[carrier]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrBolTransporterUnsupported, carrier)
	}
	return rule(recipientCountry), nil
}
