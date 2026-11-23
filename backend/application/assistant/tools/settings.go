package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/adirkuhn/commerceos/domain/assistant"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type tenantSettingsGetter interface {
	Execute(ctx context.Context) (*settingsDomain.TenantSettings, error)
}

type companyProfileGetter interface {
	Execute(ctx context.Context) (*settingsDomain.CompanyProfile, error)
}

// GetTenantSettingsTool reads the shop's general settings.
type GetTenantSettingsTool struct{ get tenantSettingsGetter }

func NewGetTenantSettingsTool(get tenantSettingsGetter) *GetTenantSettingsTool {
	return &GetTenantSettingsTool{get: get}
}

func (t *GetTenantSettingsTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name: "get_tenant_settings",
		Description: "Read the shop's general settings: default currency, timezone, language, date format, whether prices " +
			"include tax and the default tax country, stock thresholds, expiry alerts, picking mode, target markup.",
		Parameters: object(map[string]any{}),
	}
}

func (t *GetTenantSettingsTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct{}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	s, err := t.get.Execute(ctx)
	if err != nil {
		return "", err
	}
	return toJSON(struct {
		DefaultCurrency                string `json:"default_currency"`
		Timezone                       string `json:"timezone"`
		Language                       string `json:"language"`
		DateFormat                     string `json:"date_format"`
		PricesIncludeTax               bool   `json:"prices_include_tax"`
		DefaultTaxCountry              string `json:"default_tax_country"`
		InventorySyncRulesEnabled      bool   `json:"inventory_sync_rules_enabled"`
		BatchExpiryThresholdDays       int    `json:"batch_expiry_threshold_days"`
		LowStockThreshold              int    `json:"low_stock_threshold"`
		DefaultTargetMarkupBasisPoints int    `json:"default_target_markup_basis_points"`
		PickingMode                    string `json:"picking_mode"`
		ExpiryAlertEnabled             bool   `json:"expiry_alert_enabled"`
		ExpiryAlertWindowsDays         []int  `json:"expiry_alert_windows_days"`
		UpdatedAt                      string `json:"updated_at"`
	}{
		DefaultCurrency: s.DefaultCurrency, Timezone: s.Timezone, Language: string(s.Language), DateFormat: string(s.DateFormat),
		PricesIncludeTax: s.PricesIncludeTax, DefaultTaxCountry: s.DefaultTaxCountry,
		InventorySyncRulesEnabled: s.InventorySyncRulesEnabled, BatchExpiryThresholdDays: s.BatchExpiryThresholdDays,
		LowStockThreshold: s.LowStockThreshold, DefaultTargetMarkupBasisPoints: s.DefaultTargetMarkupBasisPoints,
		PickingMode: string(s.PickingMode), ExpiryAlertEnabled: s.ExpiryAlert.Enabled, ExpiryAlertWindowsDays: s.ExpiryAlert.Windows,
		UpdatedAt: formatTime(s.UpdatedAt),
	})
}

// GetCompanyProfileTool reads the shop's own business details.
type GetCompanyProfileTool struct{ get companyProfileGetter }

func NewGetCompanyProfileTool(get companyProfileGetter) *GetCompanyProfileTool {
	return &GetCompanyProfileTool{get: get}
}

func (t *GetCompanyProfileTool) Definition() assistant.ToolDefinition {
	return assistant.ToolDefinition{
		Name:        "get_company_profile",
		Description: "Read the shop's own business details used on invoices and emails: legal/trading name, VAT and chamber of commerce numbers, contact, address, IBAN.",
		Parameters:  object(map[string]any{}),
	}
}

func (t *GetCompanyProfileTool) Call(ctx context.Context, raw json.RawMessage) (string, error) {
	var args struct{}
	if err := decode(raw, &args); err != nil {
		return "", err
	}
	p, err := t.get.Execute(ctx)
	if err != nil {
		return "", err
	}
	if p == nil {
		return toJSON(map[string]any{"configured": false})
	}
	return toJSON(struct {
		Configured        bool   `json:"configured"`
		LegalName         string `json:"legal_name"`
		TradingName       string `json:"trading_name,omitempty"`
		VATNumber         string `json:"vat_number,omitempty"`
		ChamberOfCommerce string `json:"chamber_of_commerce,omitempty"`
		Email             string `json:"email,omitempty"`
		Phone             string `json:"phone,omitempty"`
		Address           string `json:"address,omitempty"`
		IBAN              string `json:"iban,omitempty"`
	}{
		Configured: true, LegalName: p.LegalName, TradingName: p.TradingName, VATNumber: p.VATNumber,
		ChamberOfCommerce: p.ChamberOfCommerce, Email: p.Email, Phone: p.Phone, Address: companyAddress(p.Address), IBAN: p.IBAN,
	})
}

func companyAddress(a settingsDomain.CompanyAddress) string {
	parts := []string{strings.TrimSpace(a.Street + " " + a.HouseNumber), strings.TrimSpace(a.PostalCode + " " + a.City), a.Country}
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}
