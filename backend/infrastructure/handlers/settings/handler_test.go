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
	appapikeys "github.com/adirkuhn/commerceos/application/apikeys"
	appsettings "github.com/adirkuhn/commerceos/application/settings"
	apikeysDomain "github.com/adirkuhn/commerceos/domain/apikeys"
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

type stubListAPIKeys struct {
	out []apikeysDomain.APIKey
	err error
}

func (s *stubListAPIKeys) Execute(_ context.Context) ([]apikeysDomain.APIKey, error) {
	return s.out, s.err
}

type stubCreateAPIKey struct {
	out      *appapikeys.CreateAPIKeyResult
	err      error
	gotName  string
	gotDescr string
}

func (s *stubCreateAPIKey) Execute(_ context.Context, name, description string, _ *string) (*appapikeys.CreateAPIKeyResult, error) {
	s.gotName, s.gotDescr = name, description
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appapikeys.CreateAPIKeyResult{Name: name, Description: description}, nil
}

type stubRevokeAPIKey struct {
	err   error
	gotID string
}

func (s *stubRevokeAPIKey) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

func newTestHandler(get *stubGetTenantSettings, update *stubUpdateTenantSettings, apiKeyStubs ...any) *settingshandler.Handler {
	list := &stubListAPIKeys{}
	create := &stubCreateAPIKey{}
	revoke := &stubRevokeAPIKey{}
	getCompany := &stubGetCompanyProfile{}
	updateCompany := &stubUpdateCompanyProfile{}
	for _, s := range apiKeyStubs {
		switch v := s.(type) {
		case *stubListAPIKeys:
			list = v
		case *stubCreateAPIKey:
			create = v
		case *stubRevokeAPIKey:
			revoke = v
		case *stubGetCompanyProfile:
			getCompany = v
		case *stubUpdateCompanyProfile:
			updateCompany = v
		}
	}
	return settingshandler.NewHandler(get, update, list, create, revoke, settingshandler.NewCompanyProfileHandler(getCompany, updateCompany))
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
	h := newTestHandler(&stubGetTenantSettings{out: &settingsDomain.TenantSettings{DefaultCurrency: "USD", BundleConfidenceThreshold: 0.4, RecommendationsMinSupport: 3, RecommendationsLookbackDays: 90, UpdatedAt: time.Now()}}, &stubUpdateTenantSettings{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/settings/tenant", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "USD")
	assert.Contains(t, w.Body.String(), "bundle_confidence_threshold")
	assert.Contains(t, w.Body.String(), "recommendations_min_support")
	assert.Contains(t, w.Body.String(), "recommendations_lookback_days")
}

func TestUpdateTenantSettingsHandler_passesAllFields(t *testing.T) {
	update := &stubUpdateTenantSettings{}
	h := newTestHandler(&stubGetTenantSettings{}, update)
	body := map[string]any{
		"default_currency": "EUR", "timezone": "Europe/Amsterdam", "date_format": "DD/MM/YYYY", "language": "nl",
		"prices_include_tax": true, "default_tax_country": "NL",
		"inventory_sync_rules_enabled": true, "batch_expiry_threshold_days": 30,
		"bundle_confidence_threshold": 0.25, "recommendations_min_support": 3,
		"recommendations_lookback_days": 90,
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
	require.InDelta(t, 0.25, update.gotInput.BundleConfidenceThreshold, 0.0001)
	require.Equal(t, 3, update.gotInput.RecommendationsMinSupport)
	require.Equal(t, 90, update.gotInput.RecommendationsLookbackDays)
}

func TestUpdateTenantSettingsHandler_propagatesError(t *testing.T) {
	update := &stubUpdateTenantSettings{err: assert.AnError}
	h := newTestHandler(&stubGetTenantSettings{}, update)
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/settings/tenant", map[string]string{"default_currency": "EUR"})

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListApiKeysHandler_returnsKeys(t *testing.T) {
	list := &stubListAPIKeys{out: []apikeysDomain.APIKey{{ID: "key-1", Name: "My storefront"}}}
	h := newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, list)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/settings/api-keys", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "My storefront")
}

func TestCreateApiKeyHandler_returnsPlaintextKeyOnce(t *testing.T) {
	create := &stubCreateAPIKey{out: &appapikeys.CreateAPIKeyResult{ID: "key-1", Name: "My storefront", PlaintextKey: "topsecretkey"}}
	h := newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, create)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/settings/api-keys", map[string]string{"name": "My storefront"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "topsecretkey")
	assert.Equal(t, "My storefront", create.gotName)
}

func TestCreateApiKeyHandler_propagatesValidationError(t *testing.T) {
	create := &stubCreateAPIKey{err: apikeysDomain.ErrNameRequired}
	h := newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, create)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/settings/api-keys", map[string]string{"name": ""})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRevokeApiKeyHandler_revokesById(t *testing.T) {
	revoke := &stubRevokeAPIKey{}
	h := newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, revoke)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/settings/api-keys/key-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "key-1", revoke.gotID)
}

func TestRevokeApiKeyHandler_notFound(t *testing.T) {
	revoke := &stubRevokeAPIKey{err: apikeysDomain.ErrNotFound}
	h := newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, revoke)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/settings/api-keys/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateTenantSettingsHandler_passesPickingMode(t *testing.T) {
	update := &stubUpdateTenantSettings{}
	h := newTestHandler(&stubGetTenantSettings{}, update)

	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/settings/tenant", map[string]any{"picking_mode": "list"})

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "list", update.gotInput.PickingMode)
}

func TestUpdateTenantSettingsHandler_invalidPickingModeIsUnprocessable(t *testing.T) {
	update := &stubUpdateTenantSettings{err: settingsDomain.ErrInvalidPickingMode}
	h := newTestHandler(&stubGetTenantSettings{}, update)

	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/settings/tenant", map[string]any{"picking_mode": "list"})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetTenantSettingsHandler_returnsPickingMode(t *testing.T) {
	get := &stubGetTenantSettings{out: &settingsDomain.TenantSettings{TenantID: "t", PickingMode: settingsDomain.PickingModeList}}
	h := newTestHandler(get, &stubUpdateTenantSettings{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/settings/tenant", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"picking_mode":"list"`)
}
