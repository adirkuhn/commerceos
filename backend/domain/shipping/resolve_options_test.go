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
		DefaultPackageType: shipping.PackageTypePackage, DefaultLabelFormat: shipping.LabelFormatZPL,
		DefaultSignatureRequired: true, DefaultOnlyRecipient: true,
	}

	out := shipping.ResolveOptions(settings, nil, shipping.CarrierDPD)

	assert.Equal(t, shipping.PackageTypePackage, out.PackageType)
	assert.Equal(t, shipping.LabelFormatZPL, out.LabelFormat)
	assert.True(t, out.SignatureRequired)
	assert.True(t, out.OnlyRecipient)
}

func TestResolveOptions_carrierOverride_winsOverConnectionSettings(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{
		DefaultPackageType: shipping.PackageTypePackage, DefaultSignatureRequired: false,
	}
	pkgType := shipping.PackageTypePackage
	sig := true
	override := &shipping.ShippingCarrierOverride{Carrier: shipping.CarrierBpost, PackageType: &pkgType, SignatureRequired: &sig}

	out := shipping.ResolveOptions(settings, override, shipping.CarrierBpost)

	assert.Equal(t, shipping.PackageTypePackage, out.PackageType)
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
	settings := &shipping.ShippingConnectionSettings{DefaultPackageType: shipping.PackageTypePackage, InsureByDefault: true}

	out := shipping.ResolveOptions(settings, nil, shipping.CarrierPostNL)

	assert.True(t, out.Insure)
}

func TestResolveOptions_insuredShipmentForcesSignatureAndOnlyRecipient(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{
		DefaultPackageType: shipping.PackageTypePackage, DefaultSignatureRequired: false, DefaultOnlyRecipient: false,
	}
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

// TestResolveOptions_mailboxPackage_forcesOptionsOff pins a real MyParcel
// 422 (error 3705, "Verzendingsopties kunnen niet worden gebruikt met
// Brievenbuspakje" -- "shipment options cannot be used with Mailbox
// package"): Mailbox package is an unregistered/administrative mail product
// on MyParcel's side and cannot carry signature, only_recipient or insurance
// at all, regardless of what the connection/carrier defaults say.
func TestResolveOptions_mailboxPackage_forcesOptionsOff(t *testing.T) {
	settings := &shipping.ShippingConnectionSettings{
		DefaultPackageType: shipping.PackageTypeMailboxPackage,
		DefaultSignatureRequired: true, DefaultOnlyRecipient: true, InsureByDefault: true,
	}

	out := shipping.ResolveOptions(settings, nil, shipping.CarrierPostNL)

	assert.Equal(t, shipping.PackageTypeMailboxPackage, out.PackageType)
	assert.False(t, out.SignatureRequired, "Mailbox package must not carry signature_required")
	assert.False(t, out.OnlyRecipient, "Mailbox package must not carry only_recipient")
	assert.False(t, out.Insure, "Mailbox package must not carry insurance")
}

// TestResolveOptions_otherNonPackageTypes_doNotForceOptionsOff pins the
// correction after TestResolveOptions_mailboxPackage_forcesOptionsOff's
// gate turned out to be over-broad: only Mailbox package's inability to
// carry insurance/signature/only_recipient is confirmed by a real MyParcel
// error. Digital stamp/Pallet/Small package/Envelope were previously lumped
// into the same restriction by assumption -- since dropped (see
// GateOptionsForPackageType's doc comment) after a real shipment showed
// Small package silently lost its insurance despite no MyParcel rejection.
func TestResolveOptions_otherNonPackageTypes_doNotForceOptionsOff(t *testing.T) {
	unrestrictedTypes := []shipping.PackageType{
		shipping.PackageTypeDigitalStamp, shipping.PackageTypePallet,
		shipping.PackageTypeSmallPackage, shipping.PackageTypeEnvelope,
	}
	for _, pt := range unrestrictedTypes {
		settings := &shipping.ShippingConnectionSettings{
			DefaultPackageType: pt, DefaultSignatureRequired: true, DefaultOnlyRecipient: true, InsureByDefault: true,
		}

		out := shipping.ResolveOptions(settings, nil, shipping.CarrierPostNL)

		assert.Equal(t, pt, out.PackageType)
		assert.True(t, out.SignatureRequired, "package type %d must keep signature_required", pt)
		assert.True(t, out.OnlyRecipient, "package type %d must keep only_recipient", pt)
		assert.True(t, out.Insure, "package type %d must keep insurance", pt)
	}
}

// A per-carrier override picking a non-Package type must still win the gate
// even when the override itself also tries to force signature/insurance on
// -- the gate is a hard MyParcel API constraint, not a preference any layer
// (including an explicit override) can opt back into.
func TestResolveOptions_carrierOverridePicksNonPackageType_stillForcesOptionsOff(t *testing.T) {
	pkgType := shipping.PackageTypeMailboxPackage
	sig, onlyRecipient, insure := true, true, true
	override := &shipping.ShippingCarrierOverride{
		Carrier: shipping.CarrierPostNL, PackageType: &pkgType,
		SignatureRequired: &sig, OnlyRecipient: &onlyRecipient, InsureByDefault: &insure,
	}

	out := shipping.ResolveOptions(nil, override, shipping.CarrierPostNL)

	assert.Equal(t, shipping.PackageTypeMailboxPackage, out.PackageType)
	assert.False(t, out.SignatureRequired)
	assert.False(t, out.OnlyRecipient)
	assert.False(t, out.Insure)
}
