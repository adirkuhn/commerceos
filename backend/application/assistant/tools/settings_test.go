package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/assistant/tools"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	taxDomain "github.com/adirkuhn/commerceos/domain/tax"
)

type stubTenantSettings struct {
	out *settingsDomain.TenantSettings
	err error
}

func (s stubTenantSettings) Execute(context.Context) (*settingsDomain.TenantSettings, error) {
	return s.out, s.err
}

type stubCompany struct {
	out *settingsDomain.CompanyProfile
	err error
}

func (s stubCompany) Execute(context.Context) (*settingsDomain.CompanyProfile, error) {
	return s.out, s.err
}

type stubTaxClasses struct {
	out []taxDomain.TaxClass
	err error
}

func (s stubTaxClasses) Execute(context.Context) ([]taxDomain.TaxClass, error) { return s.out, s.err }

type stubTaxRates struct {
	out []taxDomain.TaxRate
	err error
}

func (s stubTaxRates) Execute(context.Context) ([]taxDomain.TaxRate, error) { return s.out, s.err }

func TestGetTenantSettingsTool(t *testing.T) {
	tool := tools.NewGetTenantSettingsTool(stubTenantSettings{out: &settingsDomain.TenantSettings{
		DefaultCurrency: "EUR", Timezone: "Europe/Amsterdam", PricesIncludeTax: true, DefaultTaxCountry: "NL",
		LowStockThreshold: 5, ExpiryAlert: settingsDomain.ExpiryAlert{Enabled: true, Windows: []int{30, 7}},
	}})
	assertStrictDefinition(t, tool, "get_tenant_settings")

	got := call(t, tool, `{}`)

	assert.Equal(t, "EUR", got["default_currency"])
	assert.Equal(t, true, got["prices_include_tax"])
	assert.Equal(t, "NL", got["default_tax_country"])
	assert.EqualValues(t, 5, got["low_stock_threshold"])
	assert.Equal(t, []any{float64(30), float64(7)}, got["expiry_alert_windows_days"])
	assert.ErrorContains(t, callErr(t, tools.NewGetTenantSettingsTool(stubTenantSettings{err: errors.New("db down")}), `{}`), "db down")
}

func TestGetCompanyProfileTool(t *testing.T) {
	tool := tools.NewGetCompanyProfileTool(stubCompany{out: &settingsDomain.CompanyProfile{
		LegalName: "Sabor BV", VATNumber: "NL001", Email: "info@sabor.test",
		Address: settingsDomain.CompanyAddress{Street: "Main", HouseNumber: "1", PostalCode: "1000AA", City: "Utrecht", Country: "NL"},
	}})
	assertStrictDefinition(t, tool, "get_company_profile")

	got := call(t, tool, `{}`)

	assert.Equal(t, "Sabor BV", got["legal_name"])
	assert.Equal(t, "Main 1, 1000AA Utrecht, NL", got["address"])
	assert.ErrorContains(t, callErr(t, tools.NewGetCompanyProfileTool(stubCompany{err: errors.New("db down")}), `{}`), "db down")
}

func TestGetCompanyProfileTool_notSetUpYet(t *testing.T) {
	got := call(t, tools.NewGetCompanyProfileTool(stubCompany{}), `{}`)
	assert.Equal(t, false, got["configured"])
}

func TestListTaxSettingsTool(t *testing.T) {
	tool := tools.NewListTaxRatesTool(
		stubTaxClasses{out: []taxDomain.TaxClass{{ID: "tc-std", Name: "Standard", IsDefault: true}, {ID: "tc-low", Name: "Reduced"}}},
		stubTaxRates{out: []taxDomain.TaxRate{{TaxClassID: "tc-std", Country: "NL", RateBasisPoints: 2100}, {TaxClassID: "tc-low", Country: "NL", RateBasisPoints: 900}}},
	)
	assertStrictDefinition(t, tool, "list_tax_rates")

	got := call(t, tool, `{"country":"nl"}`)

	classes := got["tax_classes"].([]any)
	require.Len(t, classes, 2)
	std := classes[0].(map[string]any)
	assert.Equal(t, "Standard", std["name"])
	assert.Equal(t, true, std["is_default"])
	rates := std["rates"].([]any)
	assert.Equal(t, map[string]any{"country": "NL", "rate_percent": 21.0}, rates[0])

	got = call(t, tool, `{"country":"BE"}`)
	assert.Empty(t, got["tax_classes"].([]any)[0].(map[string]any)["rates"])
}

func TestListTaxSettingsTool_errors(t *testing.T) {
	assert.ErrorContains(t, callErr(t, tools.NewListTaxRatesTool(stubTaxClasses{err: errors.New("db down")}, stubTaxRates{}), `{}`), "db down")
	assert.ErrorContains(t, callErr(t, tools.NewListTaxRatesTool(stubTaxClasses{}, stubTaxRates{err: errors.New("rates down")}), `{}`), "rates down")
}
