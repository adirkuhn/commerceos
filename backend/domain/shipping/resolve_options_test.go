package shipping_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/shipping"
)

func TestResolveOptions_noSettingsNoOverride_usesHardcodedDefaults(t *testing.T) {
	out := shipping.ResolveOptions(nil, nil, shipping.CarrierPostNL)

	assert.Equal(t, shipping.CarrierPostNL, out.Carrier)
	assert.Equal(t, shipping.PackageTypePackage, out.PackageType)
	assert.Equal(t, shipping.LabelFormatPDF, out.LabelFormat)
	assert.False(t, out.SignatureRequired)
	assert.False(t, out.OnlyRecipient)
}

func TestResolveOptions_settingsOnly_usesConnectionDefaults(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{
		DefaultPackageType: shipping.PackageTypeMailboxPackage, DefaultLabelFormat: shipping.LabelFormatZPL,
		DefaultSignatureRequired: true, DefaultOnlyRecipient: true,
	}

	out := shipping.ResolveOptions(settings, nil, shipping.CarrierDPD)

	assert.Equal(t, shipping.PackageTypeMailboxPackage, out.PackageType)
	assert.Equal(t, shipping.LabelFormatZPL, out.LabelFormat)
	assert.True(t, out.SignatureRequired)
	assert.True(t, out.OnlyRecipient)
}

func TestResolveOptions_carrierOverride_winsOverConnectionSettings(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{
		DefaultPackageType: shipping.PackageTypePackage, DefaultSignatureRequired: false,
	}
	pkgType := shipping.PackageTypeDigitalStamp
	sig := true
	override := &shipping.ShippingCarrierOverride{Carrier: shipping.CarrierBpost, PackageType: &pkgType, SignatureRequired: &sig}

	out := shipping.ResolveOptions(settings, override, shipping.CarrierBpost)

	assert.Equal(t, shipping.PackageTypeDigitalStamp, out.PackageType)
	assert.True(t, out.SignatureRequired)
	// OnlyRecipient has no override set -- must still fall through to the
	// connection settings' own value (false, the zero value here), not get
	// clobbered by the override struct's own zero value.
	assert.False(t, out.OnlyRecipient)
}

func TestResolveOptions_insuranceLayersLikeOtherFields(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{InsureByDefault: true}
	no := false
	override := &shipping.ShippingCarrierOverride{Carrier: shipping.CarrierPostNL, InsureByDefault: &no}

	out := shipping.ResolveOptions(settings, override, shipping.CarrierPostNL)

	assert.False(t, out.Insure, "a non-nil override always wins over the connection default, even to turn it off")
}

func TestResolveOptions_insuranceFallsBackToConnectionDefaultWhenNoOverride(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{InsureByDefault: true}

	out := shipping.ResolveOptions(settings, nil, shipping.CarrierPostNL)

	assert.True(t, out.Insure)
}

func TestResolveOptions_insuredShipmentForcesSignatureAndOnlyRecipient(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{DefaultSignatureRequired: false, DefaultOnlyRecipient: false}
	yes := true
	override := &shipping.ShippingCarrierOverride{Carrier: shipping.CarrierPostNL, InsureByDefault: &yes}

	out := shipping.ResolveOptions(settings, override, shipping.CarrierPostNL)

	assert.True(t, out.SignatureRequired, "MyParcel requires signature whenever a shipment is insured")
	assert.True(t, out.OnlyRecipient, "MyParcel requires only_recipient whenever a shipment is insured")
}

func TestResolveOptions_notInsuredDoesNotForceSignatureOrOnlyRecipient(t *testing.T) {
	no := false
	override := &shipping.ShippingCarrierOverride{Carrier: shipping.CarrierPostNL, InsureByDefault: &no}

	out := shipping.ResolveOptions(nil, override, shipping.CarrierPostNL)

	assert.False(t, out.SignatureRequired)
	assert.False(t, out.OnlyRecipient)
}
