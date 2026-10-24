package oms_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubShopSettings struct {
	out *settingsDomain.TenantSettings
	err error
}

func (s *stubShopSettings) Get(context.Context) (*settingsDomain.TenantSettings, error) {
	return s.out, s.err
}

type stubShopTaxRates struct {
	rate       int64
	err        error
	gotClass   *string
	gotCountry string
}

func (s *stubShopTaxRates) RateBasisPoints(_ context.Context, taxClassID *string, country string) (int64, error) {
	s.gotClass, s.gotCountry = taxClassID, country
	return s.rate, s.err
}

func TestShopLinePricer_splitsVATOutOfATaxInclusivePrice(t *testing.T) {
	rates := &stubShopTaxRates{rate: 900}
	pricer := appoms.NewShopLinePricer(&stubShopSettings{out: &settingsDomain.TenantSettings{TenantID: "t-1", PricesIncludeTax: true, DefaultTaxCountry: "NL"}}, rates)
	class := "reduced"

	amounts, includesTax, err := pricer.PriceLine(context.Background(), &class, 2, 199)

	require.NoError(t, err)
	assert.True(t, includesTax)
	assert.Equal(t, omsDomain.LineAmounts{Subtotal: 398, SubtotalTax: 33, Total: 398, TotalTax: 33}, amounts)
	assert.Equal(t, &class, rates.gotClass)
	assert.Equal(t, "NL", rates.gotCountry)
}

func TestShopLinePricer_addsVATOnTopOfATaxExclusivePrice(t *testing.T) {
	pricer := appoms.NewShopLinePricer(&stubShopSettings{out: &settingsDomain.TenantSettings{TenantID: "t-1", DefaultTaxCountry: "NL"}}, &stubShopTaxRates{rate: 2100})

	amounts, includesTax, err := pricer.PriceLine(context.Background(), nil, 1, 1000)

	require.NoError(t, err)
	assert.False(t, includesTax)
	assert.Equal(t, int64(210), amounts.TotalTax)
}

func TestShopLinePricer_aNeverSavedShopPricesIncludingVAT(t *testing.T) {
	pricer := appoms.NewShopLinePricer(&stubShopSettings{out: &settingsDomain.TenantSettings{}}, &stubShopTaxRates{})

	_, includesTax, err := pricer.PriceLine(context.Background(), nil, 1, 1000)

	require.NoError(t, err)
	assert.True(t, includesTax)
}

func TestShopLinePricer_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")
	_, _, err := appoms.NewShopLinePricer(&stubShopSettings{err: boom}, &stubShopTaxRates{}).PriceLine(context.Background(), nil, 1, 1)
	assert.ErrorIs(t, err, boom)

	_, _, err = appoms.NewShopLinePricer(&stubShopSettings{out: &settingsDomain.TenantSettings{}}, &stubShopTaxRates{err: boom}).PriceLine(context.Background(), nil, 1, 1)
	assert.ErrorIs(t, err, boom)
}
