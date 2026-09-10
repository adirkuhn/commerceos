package shipping

// ResolvedOptions is the outcome of layering a carrier-specific override
// over a connection's global defaults over this package's own hardcoded
// fallback -- see ResolveOptions.
type ResolvedOptions struct {
	Carrier           Carrier
	PackageType       PackageType
	LabelFormat       string
	SignatureRequired bool
	OnlyRecipient     bool
	// Insure is the resolved on/off decision only -- this package is a pure
	// function with no I/O, so it can't know the SalesOrder's value to
	// compute an actual amount. CreateShipmentUseCase looks up that value
	// and computes the insured amount itself when this is true.
	Insure bool
}

// ResolveOptions layers, in priority order: a ShippingCarrierOverride field
// (if non-nil), then the connection's ShippingConnectionSettings (if a
// settings row exists), then this package's own hardcoded default -- pure
// function, no I/O, so CreateShipmentUseCase can unit test the layering
// without a real repository. settings/override may both be nil (a
// connection with no configuration at all yet, and/or no override for this
// specific carrier).
func ResolveOptions(settings *ShippingConnectionSettings, override *ShippingCarrierOverride, carrier Carrier) ResolvedOptions {
	out := ResolvedOptions{
		Carrier:     carrier,
		PackageType: PackageTypePackage,
		LabelFormat: LabelFormatPDF,
	}
	if settings != nil {
		out.PackageType = settings.DefaultPackageType
		out.LabelFormat = settings.DefaultLabelFormat
		out.SignatureRequired = settings.DefaultSignatureRequired
		out.OnlyRecipient = settings.DefaultOnlyRecipient
		out.Insure = settings.InsureByDefault
	}
	if override != nil {
		if override.PackageType != nil {
			out.PackageType = *override.PackageType
		}
		if override.SignatureRequired != nil {
			out.SignatureRequired = *override.SignatureRequired
		}
		if override.OnlyRecipient != nil {
			out.OnlyRecipient = *override.OnlyRecipient
		}
		if override.InsureByDefault != nil {
			out.Insure = *override.InsureByDefault
		}
	}
	// MyParcel requires only_recipient and signature whenever a shipment is
	// insured (see MyParcel's shipment-options docs) -- enforced here, not
	// left as tribal knowledge callers must remember.
	if out.Insure {
		out.SignatureRequired = true
		out.OnlyRecipient = true
	}
	// Every non-Package box type (Mailbox package, Digital stamp, Small
	// package, Envelope, Unfranked) is an unregistered/administrative mail
	// product on MyParcel's side and cannot carry signature/only_recipient/
	// insurance at all -- confirmed by a real 422 (error 3705,
	// "Verzendingsopties kunnen niet worden gebruikt met Brievenbuspakje")
	// for Mailbox package specifically. This gate always wins, even over an
	// explicit override that tries to force one of these on alongside a
	// non-Package type, since it's a hard API constraint, not a preference.
	if out.PackageType != PackageTypePackage {
		out.SignatureRequired = false
		out.OnlyRecipient = false
		out.Insure = false
	}
	return out
}
