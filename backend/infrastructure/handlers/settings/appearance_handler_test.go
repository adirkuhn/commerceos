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

type stubGetAppearance struct {
	out *settingsDomain.Appearance
	err error
}

func (s *stubGetAppearance) Execute(context.Context) (*settingsDomain.Appearance, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	a := settingsDomain.DefaultAppearance("tenant-1")
	return &a, nil
}

// stubUpdateAppearance runs the real domain validation, so the 422 cases
// exercise the actual error values end to end.
type stubUpdateAppearance struct {
	err      error
	gotInput settingsDomain.AppearanceInput
}

func (s *stubUpdateAppearance) Execute(_ context.Context, in settingsDomain.AppearanceInput) (*settingsDomain.Appearance, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	a, err := settingsDomain.NewAppearance("tenant-1", in, time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC))
	if err != nil {
		return nil, fmt.Errorf("updateAppearance: %w", err)
	}
	return a, nil
}

func appearanceBody() map[string]any {
	return map[string]any{
		"default_look": "graphite", "corners": "round", "text_size": "large", "density": "compact",
	}
}

type appearanceEnvelope struct {
	Data *struct {
		DefaultLook string  `json:"default_look"`
		Corners     string  `json:"corners"`
		TextSize    string  `json:"text_size"`
		Density     string  `json:"density"`
		UpdatedAt   *string `json:"updated_at"`
	} `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeAppearance(t *testing.T, raw []byte) appearanceEnvelope {
	t.Helper()
	var body appearanceEnvelope
	require.NoError(t, json.Unmarshal(raw, &body))
	return body
}

func TestGetAppearance_neverSavedReturnsDefaultsWithNullUpdatedAt(t *testing.T) {
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}))

	w := doRequest(r, http.MethodGet, "/api/v1/settings/appearance", nil)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), `"updated_at":null`)
	body := decodeAppearance(t, w.Body.Bytes())
	require.NotNil(t, body.Data)
	assert.Equal(t, "wallet", body.Data.DefaultLook)
	for _, retired := range []string{"font", "accent_color", "sidebar_color"} {
		assert.NotContains(t, w.Body.String(), `"`+retired+`"`, "retired field")
	}
	assert.Equal(t, "normal", body.Data.Corners)
	assert.Equal(t, "normal", body.Data.TextSize)
	assert.Equal(t, "comfortable", body.Data.Density)
	assert.Nil(t, body.Data.UpdatedAt)
}

func TestGetAppearance_returnsSaved(t *testing.T) {
	saved := settingsDomain.Appearance{
		TenantID: "tenant-1", DefaultLook: settingsDomain.LookGraphite,
		Corners: settingsDomain.CornersSharp, TextSize: settingsDomain.TextSizeLarger, Density: settingsDomain.DensityCompact,
		UpdatedAt: time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC),
	}
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, &stubGetAppearance{out: &saved}))

	w := doRequest(r, http.MethodGet, "/api/v1/settings/appearance", nil)

	require.Equal(t, http.StatusOK, w.Code)
	body := decodeAppearance(t, w.Body.Bytes())
	assert.Equal(t, "graphite", body.Data.DefaultLook)
	assert.Equal(t, "larger", body.Data.TextSize)
	require.NotNil(t, body.Data.UpdatedAt)
	assert.Equal(t, "2026-10-01T08:00:00Z", *body.Data.UpdatedAt)
}

func TestGetAppearance_internalErrorIsNotLeaked(t *testing.T) {
	get := &stubGetAppearance{err: fmt.Errorf("db: %w", errSecretDB)}
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, get))

	w := doRequest(r, http.MethodGet, "/api/v1/settings/appearance", nil)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "secret")
}

func TestUpdateAppearance_passesInputAndReturnsSaved(t *testing.T) {
	update := &stubUpdateAppearance{}
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, update))

	w := doRequest(r, http.MethodPut, "/api/v1/settings/appearance", appearanceBody())

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, settingsDomain.AppearanceInput{
		DefaultLook: "graphite", Corners: "round", TextSize: "large", Density: "compact",
	}, update.gotInput)
	body := decodeAppearance(t, w.Body.Bytes())
	assert.Equal(t, "graphite", body.Data.DefaultLook)
	assert.Equal(t, "compact", body.Data.Density)
	require.NotNil(t, body.Data.UpdatedAt)
	assert.Equal(t, "2026-10-12T09:00:00Z", *body.Data.UpdatedAt)
}

func TestUpdateAppearance_malformedJSONIs400(t *testing.T) {
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}))

	w := doRequest(r, http.MethodPut, "/api/v1/settings/appearance", "not an object")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateAppearance_eachInvalidFieldIs422WithItsCode(t *testing.T) {
	cases := map[string]struct {
		field, value, code string
	}{
		"default look": {"default_look", "classic", "APPEARANCE_INVALID_LOOK"},
		"corners":      {"corners", "pill", "APPEARANCE_INVALID_CORNERS"},
		"text size":    {"text_size", "huge", "APPEARANCE_INVALID_TEXT_SIZE"},
		"density":      {"density", "cosy", "APPEARANCE_INVALID_DENSITY"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := appearanceBody()
			req[tc.field] = tc.value
			r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, &stubUpdateAppearance{}))

			w := doRequest(r, http.MethodPut, "/api/v1/settings/appearance", req)

			require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
			body := decodeAppearance(t, w.Body.Bytes())
			require.NotNil(t, body.Error)
			assert.Equal(t, tc.code, body.Error.Code)
			assert.NotEmpty(t, body.Error.Message)
		})
	}
}

func TestUpdateAppearance_internalErrorIsNotLeaked(t *testing.T) {
	update := &stubUpdateAppearance{err: fmt.Errorf("db: %w", errSecretDB)}
	r := newTestRouter(newTestHandler(&stubGetTenantSettings{}, &stubUpdateTenantSettings{}, update))

	w := doRequest(r, http.MethodPut, "/api/v1/settings/appearance", appearanceBody())

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "secret")
}
