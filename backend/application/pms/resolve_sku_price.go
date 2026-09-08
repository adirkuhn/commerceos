package pms

import (
	"context"
	"time"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

// ResolvedSKUPrice is what a SalesChannel actually charges for a SKU in
// its own currency -- see ResolveSKUPrice.
type ResolvedSKUPrice struct {
	PriceMinorUnits     int64
	SalePriceMinorUnits *int64
	CurrencyCode        string
}

// ResolveSKUPrice computes sku's price in channelCurrency ("" meaning the
// tenant's default currency -- the pre-multi-currency behavior, always a
// straight passthrough of sku's own fields). Must be called from inside an
// existing tenant.WithTenant block (like resolveStripeConnection in
// application/storefront) -- it makes no transaction of its own so callers
// already computing a cart/checkout total in one transaction can call this
// per line without nesting.
//
// Resolution order: (1) sku.CurrencyCode itself needs no conversion; (2) a
// manual SKUCurrencyPrice override for channelCurrency wins if one exists;
// (3) otherwise fx.ExchangeRate converts sku's own price -- ErrExchangeRateRequired
// if none is configured (a misconfigured channel, since
// CreateSalesChannelUseCase already requires one to exist at creation
// time).
func ResolveSKUPrice(ctx context.Context, sku pmsDomain.SKU, channelCurrency string, overrideRepo pmsDomain.SKUCurrencyPriceRepository, fxRepo fxDomain.ExchangeRateRepository) (*ResolvedSKUPrice, error) {
	if channelCurrency == "" || channelCurrency == sku.CurrencyCode {
		return &ResolvedSKUPrice{PriceMinorUnits: sku.PriceMinorUnits, SalePriceMinorUnits: sku.SalePriceMinorUnits, CurrencyCode: sku.CurrencyCode}, nil
	}

	override, err := overrideRepo.FindBySKUAndCurrency(ctx, sku.ID, channelCurrency)
	if err != nil {
		return nil, err
	}
	if override != nil {
		salePrice := override.SalePriceMinorUnits
		if override.SaleEndDate != nil && override.SaleEndDate.Before(time.Now()) {
			salePrice = nil
		}
		return &ResolvedSKUPrice{PriceMinorUnits: override.PriceMinorUnits, SalePriceMinorUnits: salePrice, CurrencyCode: channelCurrency}, nil
	}

	rate, err := fxRepo.FindByCurrency(ctx, channelCurrency)
	if err != nil {
		return nil, err
	}
	if rate == nil {
		return nil, pmsDomain.ErrExchangeRateRequired
	}
	var convertedSale *int64
	if sku.SalePriceMinorUnits != nil {
		v := rate.Convert(*sku.SalePriceMinorUnits)
		convertedSale = &v
	}
	return &ResolvedSKUPrice{PriceMinorUnits: rate.Convert(sku.PriceMinorUnits), SalePriceMinorUnits: convertedSale, CurrencyCode: channelCurrency}, nil
}
