package shipping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func strPtr(s string) *string { return &s }

func TestBuildCarrierTrackingURL_substitutesBarcodeAndPercentEncodedPostcode(t *testing.T) {
	tmpl := strPtr("https://postnl.nl/tracktrace/?B={barcode}&P={postcode}&D=NL&T=C&L=EN")

	got := shippingDomain.BuildCarrierTrackingURL(tmpl, "3SMYPA827793433", "1251 SB")

	assert.Equal(t, "https://postnl.nl/tracktrace/?B=3SMYPA827793433&P=1251%20SB&D=NL&T=C&L=EN", got)
}

func TestBuildCarrierTrackingURL_templateWithNoPostcodePlaceholder(t *testing.T) {
	tmpl := strPtr("https://www.dpdgroup.com/nl/mydpd/my-parcels/search?lang=en&parcelNumber={barcode}")

	got := shippingDomain.BuildCarrierTrackingURL(tmpl, "05162815629376", "")

	assert.Equal(t, "https://www.dpdgroup.com/nl/mydpd/my-parcels/search?lang=en&parcelNumber=05162815629376", got)
}

func TestBuildCarrierTrackingURL_nilTemplate_returnsEmpty(t *testing.T) {
	got := shippingDomain.BuildCarrierTrackingURL(nil, "3SMYPA1", "1251 SB")

	assert.Empty(t, got)
}

func TestBuildCarrierTrackingURL_emptyTemplate_returnsEmpty(t *testing.T) {
	got := shippingDomain.BuildCarrierTrackingURL(strPtr(""), "3SMYPA1", "1251 SB")

	assert.Empty(t, got)
}

func TestBuildCarrierTrackingURL_noBarcodeYet_returnsEmpty(t *testing.T) {
	tmpl := strPtr("https://postnl.nl/tracktrace/?B={barcode}&P={postcode}")

	got := shippingDomain.BuildCarrierTrackingURL(tmpl, "", "1251 SB")

	assert.Empty(t, got, "nothing to link to without a barcode yet")
}
