package settings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gensettings "github.com/adirkuhn/commerceos/api/settings"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	settingshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/settings"
)

func init() { gin.SetMode(gin.TestMode) }

type stubGetTenantSettings struct {
	out *settingsDomain.TenantSettings
	err error
}

func (s *stubGetTenantSettings) Execute(_ context.Context) (*settingsDomain.TenantSettings, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &settingsDomain.TenantSettings{}, nil
}

type stubUpdateTenantSettings struct {
	out      *settingsDomain.TenantSettings
	err      error
	gotInput appsettings.UpdateTenantSettingsInput
}

func (s *stubUpdateTenantSettings) Execute(_ context.Context, in appsettings.UpdateTenantSettingsInput) (*settingsDomain.TenantSettings, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &settingsDomain.TenantSettings{
		DefaultCurrency: in.DefaultCurrency, Timezone: in.Timezone, DateFormat: in.DateFormat, Language: in.Language,
		PricesIncludeTax: in.PricesIncludeTax, DefaultTaxCountry: in.DefaultTaxCountry,
	}, nil
}

type stubSetStorefrontKey struct {
	out *appintegrations.SetStorefrontKeyResult
	err error
}

func (s *stubSetStorefrontKey) Execute(_ context.Context) (*appintegrations.SetStorefrontKeyResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appintegrations.SetStorefrontKeyResult{}, nil
}

func newTestHandler(get *stubGetTenantSettings, update *stubUpdateTenantSettings, setKey ...*stubSetStorefrontKey) *settingshandler.Handler {
	var key *stubSetStorefrontKey
	if len(setKey) > 0 {
		key = setKey[0]
	} else {
		key = &stubSetStorefrontKey{}
	}
	return settingshandler.NewHandler(get, update, key)
}

func newTestRouter(h *settingshandler.Handler) *gin.Engine {
	r := gin.New()
	gensettings.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestGetTenantSettingsHandler_returnsSettings(t *testing.T) {
	h := newTestHandler(&stubGetTenantSettings{out: &settingsDomain.TenantSettings{DefaultCurrency: "USD", UpdatedAt: time.Now()}}, &stubUpdateTenantSettings{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/settings/tenant", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "USD")
}

func TestUpdateTenantSettingsHandler_passesAllFields(t *testing.T) {
	update := &stubUpdateTenantSettings{}
	h := newTestHandler(&stubGetTenantSettings{}, update)
	body := map[string]any{
		"default_currency": "EUR", "timezone": "Europe/Amsterdam", "date_format": "DD/MM/YYYY", "language": "nl",
		"prices_include_tax": true, "default_tax_country": "NL",
		"inventory_sync_rules_enabled": true, "batch_expiry_threshold_days": 30,
	}
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/settings/tenant", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "EUR", update.gotInput.DefaultCurrency)
	require.Equal(t, "Europe/Amsterdam", update.gotInput.Timezone)
	require.Equal(t, "DD/MM/YYYY", update.gotInput.DateFormat)
	require.Equal(t, "nl", update.gotInput.Language)
	require.True(t, update.gotInput.PricesIncludeTax)
	require.Equal(t, "NL", update.gotInput.DefaultTaxCountry)
	require.True(t, update.gotInput.InventorySyncRulesEnabled)
	require.Equal(t, 30, update.gotInput.BatchExpiryThresholdDays)
}

func TestUpdateTenantSettingsHandler_propagatesError(t *testing.T) {
	update := &stubUpdateTenantSettings{err: assert.AnError}
	h := newTestHandler(&stubGetTenantSettings{}, update)
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/settings/tenant", map[string]string{"default_currency": "EUR"})

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSetStorefrontKeyHandler_returnsPlaintextKeyOnce(t *testing.T) {
	setKey := &stubSetStorefrontKey{out: &appintegrations.SetStorefrontKeyResult{PlaintextKey: "topsecretkey"}}
	h := newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, setKey)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/settings/storefront-key", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "topsecretkey")
}

func TestSetStorefrontKeyHandler_propagatesError(t *testing.T) {
	setKey := &stubSetStorefrontKey{err: assert.AnError}
	h := newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, setKey)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/settings/storefront-key", nil)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
