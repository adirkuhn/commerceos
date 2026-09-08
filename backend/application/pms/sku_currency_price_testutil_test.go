package pms_test

import (
	"context"

	fxDomain "github.com/adirkuhn/commerceos/domain/fx"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

type stubSKUCurrencyPriceRepo struct {
	byKey                         map[string]pmsDomain.SKUCurrencyPrice
	upserted                      *pmsDomain.SKUCurrencyPrice
	deletedSKUID, deletedCurrency string
	findErr                       error
}

func skuCurrencyKey(skuID, currencyCode string) string { return skuID + "|" + currencyCode }

func (s *stubSKUCurrencyPriceRepo) FindBySKUAndCurrency(_ context.Context, skuID, currencyCode string) (*pmsDomain.SKUCurrencyPrice, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if p, ok := s.byKey[skuCurrencyKey(skuID, currencyCode)]; ok {
		cp := p
		return &cp, nil
	}
	return nil, nil
}

func (s *stubSKUCurrencyPriceRepo) ListBySKU(_ context.Context, skuID string) ([]pmsDomain.SKUCurrencyPrice, error) {
	var out []pmsDomain.SKUCurrencyPrice
	for _, p := range s.byKey {
		if p.SKUID == skuID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *stubSKUCurrencyPriceRepo) Upsert(_ context.Context, p *pmsDomain.SKUCurrencyPrice) error {
	p.ID = "sku-currency-price-1"
	s.upserted = p
	return nil
}

func (s *stubSKUCurrencyPriceRepo) Delete(_ context.Context, skuID, currencyCode string) error {
	s.deletedSKUID, s.deletedCurrency = skuID, currencyCode
	return nil
}

type stubExchangeRateRepoPMS struct {
	byCurrency map[string]fxDomain.ExchangeRate
}

func (s *stubExchangeRateRepoPMS) FindByCurrency(_ context.Context, currencyCode string) (*fxDomain.ExchangeRate, error) {
	if r, ok := s.byCurrency[currencyCode]; ok {
		cp := r
		return &cp, nil
	}
	return nil, nil
}
func (s *stubExchangeRateRepoPMS) ListByTenant(_ context.Context) ([]fxDomain.ExchangeRate, error) {
	return nil, nil
}
func (s *stubExchangeRateRepoPMS) Upsert(_ context.Context, _ *fxDomain.ExchangeRate) error {
	return nil
}
func (s *stubExchangeRateRepoPMS) Delete(_ context.Context, _ string) error { return nil }
