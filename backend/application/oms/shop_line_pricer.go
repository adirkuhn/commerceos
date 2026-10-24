package oms

import (
	"context"
	"fmt"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type shopPricingSettings interface {
	Get(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type shopTaxRates interface {
	RateBasisPoints(ctx context.Context, taxClassID *string, country string) (int64, error)
}

// ShopLinePricer prices a line at the shop's own price the way the shop
// prices its catalog: the tax class's VAT rate in the shop's country, inside
// or on top of the price per its prices-include-tax setting. It reads inside
// the caller's transaction (tenant.WithTenant).
type ShopLinePricer struct {
	settings shopPricingSettings
	rates    shopTaxRates
}

func NewShopLinePricer(settings shopPricingSettings, rates shopTaxRates) *ShopLinePricer {
	return &ShopLinePricer{settings: settings, rates: rates}
}

// PriceLine returns the line's amounts and whether its total includes the VAT.
func (p *ShopLinePricer) PriceLine(ctx context.Context, taxClassID *string, quantity, unitPriceMinorUnits int64) (omsDomain.LineAmounts, bool, error) {
	s, err := p.settings.Get(ctx)
	if err != nil {
		return omsDomain.LineAmounts{}, false, fmt.Errorf("shopLinePricer: %w", err)
	}
	rate, err := p.rates.RateBasisPoints(ctx, taxClassID, s.DefaultTaxCountry)
	if err != nil {
		return omsDomain.LineAmounts{}, false, fmt.Errorf("shopLinePricer: %w", err)
	}
	// A never-saved shop reads as zero values; its default is tax-inclusive.
	includesTax := s.PricesIncludeTax || s.TenantID == ""
	return omsDomain.PriceLine(quantity, unitPriceMinorUnits, rate, includesTax), includesTax, nil
}
