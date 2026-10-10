package settings_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
)

type stubGetCompanyProfile struct {
	out *settingsDomain.CompanyProfile
	err error
}

func (s *stubGetCompanyProfile) Execute(context.Context) (*settingsDomain.CompanyProfile, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &settingsDomain.CompanyProfile{}, nil
}

type stubUpdateCompanyProfile struct {
	err      error
	gotInput settingsDomain.CompanyProfileInput
}

func (s *stubUpdateCompanyProfile) Execute(_ context.Context, in settingsDomain.CompanyProfileInput) (*settingsDomain.CompanyProfile, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	return &settingsDomain.CompanyProfile{
		LegalName: in.LegalName, VATNumber: in.VATNumber, Address: in.Address, IBAN: in.IBAN,
		UpdatedAt: time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC),
	}, nil
}

func companyBody() map[string]any {
	return map[string]any{
		"legal_name": "Cantinho B.V.", "trading_name": "", "vat_number": "NL123456789B01",
		"chamber_of_commerce": "12345678", "email": "", "phone": "", "iban": "",
		"address": map[string]any{"street": "Damrak", "house_number": "1", "postal_code": "1012 LG", "city": "Amsterdam", "country": "NL"},
	}
}

type companyEnvelope struct {
	Data struct {
		LegalName string `json:"legal_name"`
		VATNumber string `json:"vat_number"`
		Address   struct {
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"address"`
		UpdatedAt string `json:"updated_at"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestGetCompanyProfile_returnsProfile(t *testing.T) {
	get := &stubGetCompanyProfile{out: &settingsDomain.CompanyProfile{
		LegalName: "Cantinho B.V.", Address: settingsDomain.CompanyAddress{City: "Amsterdam", Country: "NL"},
	}}
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, get))

	w := doRequest(r, http.MethodGet, "/api/v1/settings/company-profile", nil)

	require.Equal(t, http.StatusOK, w.Code)
	var body companyEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "Cantinho B.V.", body.Data.LegalName)
	assert.Equal(t, "Amsterdam", body.Data.Address.City)
}

func TestGetCompanyProfile_internalErrorIsNotLeaked(t *testing.T) {
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, &stubGetCompanyProfile{err: fmt.Errorf("db: %w", errSecretDB)}))

	w := doRequest(r, http.MethodGet, "/api/v1/settings/company-profile", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "secret")
}

var errSecretDB = fmt.Errorf("secret connection string")

func TestUpdateCompanyProfile_passesInputAndReturnsSaved(t *testing.T) {
	update := &stubUpdateCompanyProfile{}
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, update))

	w := doRequest(r, http.MethodPut, "/api/v1/settings/company-profile", companyBody())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "Cantinho B.V.", update.gotInput.LegalName)
	assert.Equal(t, "NL", update.gotInput.Address.Country)
	assert.Equal(t, "12345678", update.gotInput.ChamberOfCommerce)
	var body companyEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "NL123456789B01", body.Data.VATNumber)
	assert.Equal(t, "2026-09-24T10:00:00Z", body.Data.UpdatedAt)
}

func TestUpdateCompanyProfile_mapsEachValidationErrorToItsCode(t *testing.T) {
	cases := map[error]string{
		settingsDomain.ErrLegalNameRequired:   "COMPANY_LEGAL_NAME_REQUIRED",
		settingsDomain.ErrInvalidCompanyEmail: "COMPANY_INVALID_EMAIL",
		settingsDomain.ErrInvalidVATNumber:    "COMPANY_INVALID_VAT_NUMBER",
		settingsDomain.ErrInvalidIBAN:         "COMPANY_INVALID_IBAN",
		settingsDomain.ErrInvalidCountryCode:  "COMPANY_INVALID_COUNTRY",
	}
	for domainErr, code := range cases {
		t.Run(code, func(t *testing.T) {
			update := &stubUpdateCompanyProfile{err: fmt.Errorf("updateCompanyProfile: %w", domainErr)}
			r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, update))

			w := doRequest(r, http.MethodPut, "/api/v1/settings/company-profile", companyBody())

			require.Equal(t, http.StatusUnprocessableEntity, w.Code)
			var body companyEnvelope
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			require.NotNil(t, body.Error)
			assert.Equal(t, code, body.Error.Code)
			assert.Equal(t, domainErr.Error(), body.Error.Message)
		})
	}
}
