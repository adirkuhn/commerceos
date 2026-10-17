package channels

import (
	"context"
	"fmt"
	"math"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

// bolVATCountry: Bol offers are priced with Dutch VAT.
const bolVATCountry = "NL"

// vatRateResolver is the one method Bol pricing needs from
// application/tax.TaxRateResolver (ISP). Called inside the caller's
// tenant.WithTenant.
type vatRateResolver interface {
	RateBasisPoints(ctx context.Context, taxClassID *string, country string) (int64, error)
}

// bolPricingConfig is the merchant's Bol price settings for one product --
// the saved BolProductOverride, or a draft being previewed.
type bolPricingConfig struct {
	CustomPriceMinorUnits         *int64
	BundlePricingEnabled          bool
	BundleTiers                   []channelsDomain.BolBundleTier
	CommissionOverrideBasisPoints *int32
}

func bolPricingConfigOf(o *channelsDomain.BolProductOverride) bolPricingConfig {
	if o == nil {
		return bolPricingConfig{}
	}
	return bolPricingConfig{
		CustomPriceMinorUnits: o.CustomPriceMinorUnits, BundlePricingEnabled: o.BundlePricingEnabled,
		BundleTiers: o.BundleTiers, CommissionOverrideBasisPoints: o.CommissionOverrideBasisPoints,
	}
}

// resolveBolPrices prices one SKU's offer (see channels.ResolveBolPrices),
// asking Bol for its fee at each candidate price unless the merchant set
// their own commission.
func resolveBolPrices(ctx context.Context, client integrationsDomain.BolClient, ean string, in channelsDomain.BolPriceInputs, cfg bolPricingConfig) ([]channelsDomain.BolResolvedPrice, error) {
	var tiers []channelsDomain.BolBundleTier
	if cfg.BundlePricingEnabled {
		tiers = cfg.BundleTiers
	}
	return channelsDomain.ResolveBolPrices(in, cfg.CustomPriceMinorUnits, tiers, bolFeeLookup(ctx, client, ean, cfg.CommissionOverrideBasisPoints))
}

func bolFeeLookup(ctx context.Context, client integrationsDomain.BolClient, ean string, commissionOverride *int32) func(int64) (channelsDomain.BolFee, error) {
	if commissionOverride != nil {
		fee := channelsDomain.BolFee{PercentBasisPoints: int64(*commissionOverride)}
		return func(int64) (channelsDomain.BolFee, error) { return fee, nil }
	}
	seen := map[int64]channelsDomain.BolFee{}
	return func(price int64) (channelsDomain.BolFee, error) {
		if fee, ok := seen[price]; ok {
			return fee, nil
		}
		c, err := client.GetCommission(ctx, ean, float64(price)/100)
		if err != nil {
			return channelsDomain.BolFee{}, fmt.Errorf("%w: %w", channelsDomain.ErrBolCommissionRequestFailed, err)
		}
		fee := channelsDomain.BolFee{
			FixedMinorUnits:    int64(math.Round(c.FixedAmount * 100)),
			PercentBasisPoints: int64(math.Round(c.Percentage * 100)),
		}
		seen[price] = fee
		return fee, nil
	}
}

func toBolOfferPrices(levels []channelsDomain.BolResolvedPrice) []integrationsDomain.BolPrice {
	out := make([]integrationsDomain.BolPrice, len(levels))
	for i, l := range levels {
		out[i] = integrationsDomain.BolPrice{Quantity: l.Quantity, UnitPrice: float64(l.UnitPriceMinorUnits) / 100}
	}
	return out
}
