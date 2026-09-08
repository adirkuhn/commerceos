package storefront

import (
	"context"
	"strings"

	"gorm.io/gorm"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ShippingOption is one priced delivery choice a shopper can pick at
// checkout -- Title doubles as the option's identity (see CheckoutInput's
// doc comment on ShippingMethodTitle): MyParcel's own delivery-options
// response has no separate machine ID, just a human PriceComment
// ("Standard", "Evening"), and that's already unique per call.
type ShippingOption struct {
	Title            string
	AmountMinorUnits int64
	CurrencyCode     string
}

// shippingOptionsResolver resolves the tenant's MyParcel connection and
// carrier, then asks for priced delivery options for one address -- the
// one piece GetShippingOptionsUseCase (what checkout_form shows the
// shopper) and CheckoutCartUseCase (which re-resolves the same options
// server-side rather than ever trusting a client-supplied price) share.
// Best-effort throughout: any missing piece (no MyParcel connection, an
// unsupported country -- MyParcel's own NL/BE-only constraint, an address
// splitStreetAndNumber can't pull a house number out of, a failed
// MyParcel call) returns no options rather than an error.
type shippingOptionsResolver struct {
	db                   *gorm.DB
	connRepo             integrationsDomain.ConnectionRepository
	shippingSettingsRepo shippingDomain.ShippingConnectionSettingsRepository
	estimateUC           deliveryPriceEstimator
}

func (r *shippingOptionsResolver) resolve(ctx context.Context, shipping CheckoutAddressInput) []ShippingOption {
	if r.estimateUC == nil {
		return nil
	}
	// Checked up front, before ever opening a transaction to look up a
	// connection -- same short-circuit GetDeliveryPriceEstimateUseCase's
	// own Execute does; no point resolving a connection for a country
	// MyParcel could never price anyway.
	cc := strings.ToUpper(strings.TrimSpace(shipping.Country))
	if cc != "NL" && cc != "BE" {
		return nil
	}
	_, number, _ := splitStreetAndNumber(shipping.Address1, shipping.Address2)
	if number == "" {
		return nil
	}

	var connectionID string
	carrier := shippingDomain.CarrierPostNL
	err := tenant.WithTenant(ctx, r.db, func(ctx context.Context) error {
		conns, err := r.connRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		for _, c := range conns {
			if c.Provider == integrationsDomain.ProviderMyParcel {
				connectionID = c.ID
				break
			}
		}
		if connectionID == "" {
			return nil
		}
		settings, err := r.shippingSettingsRepo.FindByConnectionID(ctx, connectionID)
		if err != nil {
			return err
		}
		if settings != nil {
			carrier = settings.DefaultCarrier
		}
		return nil
	})
	if err != nil || connectionID == "" {
		return nil
	}

	// GetDeliveryPriceEstimateUseCase opens its own tenant.WithTenant --
	// called here, outside the transaction above, to avoid nesting one.
	options, err := r.estimateUC.Execute(ctx, appshipping.GetDeliveryPriceEstimateInput{
		ConnectionID: connectionID, CC: cc, PostalCode: shipping.Postcode, Number: number, Carrier: carrier,
	})
	if err != nil {
		return nil
	}
	out := make([]ShippingOption, len(options))
	for i, o := range options {
		out[i] = ShippingOption{Title: o.PriceComment, AmountMinorUnits: int64(o.AmountMinorUnits), CurrencyCode: o.Currency}
	}
	return out
}

// GetShippingOptionsUseCase is checkout_form's "what can I ship this to"
// call -- read-only, no persistence. An empty result is a normal state
// (see shippingOptionsResolver's doc comment for every reason that
// happens), not an error: the storefront-frontend falls back to shipping
// cost communicated some other way when there's nothing to show.
type GetShippingOptionsUseCase struct {
	resolver *shippingOptionsResolver
}

func NewGetShippingOptionsUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	shippingSettingsRepo shippingDomain.ShippingConnectionSettingsRepository,
	estimateUC deliveryPriceEstimator,
) *GetShippingOptionsUseCase {
	return &GetShippingOptionsUseCase{resolver: &shippingOptionsResolver{
		db: db, connRepo: connRepo, shippingSettingsRepo: shippingSettingsRepo, estimateUC: estimateUC,
	}}
}

func (uc *GetShippingOptionsUseCase) Execute(ctx context.Context, shipping CheckoutAddressInput) []ShippingOption {
	return uc.resolver.resolve(ctx, shipping)
}
