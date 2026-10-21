package shipping

import (
	"context"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

// companyProfileGetter is settings.CompanyProfileRepository narrowed to
// point of use (ISP) -- the company address is the fallback sender.
type companyProfileGetter interface {
	Get(ctx context.Context) (*settingsDomain.CompanyProfile, error)
}

// ShipmentAddresses is what a label prints as sender and as the return
// address for undeliverable parcels; either may be nil.
type ShipmentAddresses struct {
	Sender *integrationsDomain.CarrierAddress
	Return *integrationsDomain.CarrierAddress
}

// shipmentAddressSource is ShipmentAddressResolver at point of use.
type shipmentAddressSource interface {
	Resolve(ctx context.Context) (ShipmentAddresses, error)
}

// ShipmentAddressResolver picks the tenant's label addresses: its own
// shipping sender address, else the company profile address; the return
// address only when one was set. Runs inside the caller's transaction.
type ShipmentAddressResolver struct {
	prefs    shippingDomain.ShippingPreferencesRepository
	profiles companyProfileGetter
}

func NewShipmentAddressResolver(prefs shippingDomain.ShippingPreferencesRepository, profiles companyProfileGetter) *ShipmentAddressResolver {
	return &ShipmentAddressResolver{prefs: prefs, profiles: profiles}
}

func (r *ShipmentAddressResolver) Resolve(ctx context.Context) (ShipmentAddresses, error) {
	prefs, err := r.prefs.Get(ctx)
	if err != nil {
		return ShipmentAddresses{}, err
	}
	var out ShipmentAddresses
	if prefs != nil {
		out.Sender = carrierAddressOf(prefs.SenderAddress)
		out.Return = carrierAddressOf(prefs.ReturnAddress)
	}
	if out.Sender == nil {
		profile, err := r.profiles.Get(ctx)
		if err != nil {
			return ShipmentAddresses{}, err
		}
		out.Sender = companySender(profile)
	}
	return out, nil
}

func carrierAddressOf(a *shippingDomain.ShippingAddress) *integrationsDomain.CarrierAddress {
	if a == nil {
		return nil
	}
	return &integrationsDomain.CarrierAddress{
		Company: a.Company, Street: a.Street, Number: a.Number, NumberSuffix: a.NumberSuffix,
		PostalCode: a.PostalCode, City: a.City, CC: a.CC, Email: a.Email, Phone: a.Phone,
	}
}

// companySender is the company profile address as a label sender, nil
// when none was saved.
func companySender(p *settingsDomain.CompanyProfile) *integrationsDomain.CarrierAddress {
	if p == nil || p.Address.Street == "" || p.Address.PostalCode == "" {
		return nil
	}
	name := p.TradingName
	if name == "" {
		name = p.LegalName
	}
	return &integrationsDomain.CarrierAddress{
		Company: name, Street: p.Address.Street, Number: p.Address.HouseNumber, PostalCode: p.Address.PostalCode,
		City: p.Address.City, CC: p.Address.Country, Email: p.Email, Phone: p.Phone,
	}
}
