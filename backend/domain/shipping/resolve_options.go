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
	}
	return out
}
