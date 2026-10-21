package shipping_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	shippinghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/shipping"
)

const (
	mpID = "11111111-1111-1111-1111-111111111111"
	pnID = "22222222-2222-2222-2222-222222222222"
)

type stubListShippingConnections struct {
	result *appshipping.ShippingConnections
	err    error
}

func (s *stubListShippingConnections) Execute(context.Context) (*appshipping.ShippingConnections, error) {
	return s.result, s.err
}

type stubSetDefaultShipping struct {
	got    *string
	called bool
	err    error
}

func (s *stubSetDefaultShipping) Execute(_ context.Context, id *string) error {
	s.called, s.got = true, id
	return s.err
}

func twoShippingConnections() *appshipping.ShippingConnections {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &appshipping.ShippingConnections{
		Connections: []appshipping.ShippingConnection{
			{Connection: integrationsDomain.Connection{ID: mpID, Provider: integrationsDomain.ProviderMyParcel, Status: integrationsDomain.StatusConnected, CreatedAt: created}},
			{Connection: integrationsDomain.Connection{ID: pnID, Provider: "postnl", Status: integrationsDomain.StatusConnected, CreatedAt: created.AddDate(0, 1, 0)},
				Missing: []string{appshipping.MissingSenderAddress}},
		},
		DefaultConnectionID: mpID,
	}
}

func preferencesRouter(list *stubListShippingConnections, set *stubSetDefaultShipping) http.Handler {
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{},
		shippinghandler.NewPreferencesHandler(list, set, &stubGetAddresses{}, &stubUpdateAddresses{}), nil)
	return newTestRouter(h)
}

func TestListShippingConnections_200MarksTheDefault(t *testing.T) {
	r := preferencesRouter(&stubListShippingConnections{result: twoShippingConnections()}, &stubSetDefaultShipping{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/shipping/connections", nil))

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data struct {
			Connections []struct {
				ID        string `json:"id"`
				Provider  string `json:"provider"`
				IsDefault bool   `json:"is_default"`
			} `json:"connections"`
			DefaultConnectionID    *string `json:"default_connection_id"`
			ConfiguredConnectionID *string `json:"configured_connection_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Connections, 2)
	assert.True(t, body.Data.Connections[0].IsDefault)
	assert.False(t, body.Data.Connections[1].IsDefault)
	assert.Equal(t, "postnl", body.Data.Connections[1].Provider)
	require.NotNil(t, body.Data.DefaultConnectionID)
	assert.Equal(t, mpID, *body.Data.DefaultConnectionID)
	assert.Nil(t, body.Data.ConfiguredConnectionID)
	assert.Contains(t, w.Body.String(), `"configured_connection_id":null`)
	assert.Contains(t, w.Body.String(), `"missing":[]`, "a ready connection lists nothing missing")
	assert.Contains(t, w.Body.String(), `"missing":["sender_address"]`)
}

func TestListShippingConnections_noneGivesNullDefault(t *testing.T) {
	r := preferencesRouter(&stubListShippingConnections{result: &appshipping.ShippingConnections{}}, &stubSetDefaultShipping{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/shipping/connections", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"connections":[]`)
	assert.Contains(t, w.Body.String(), `"default_connection_id":null`)
}

func patchPreferences(r http.Handler, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/shipping/preferences", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateShippingPreferences_200SetsAndReturnsRefreshedList(t *testing.T) {
	set := &stubSetDefaultShipping{}
	r := preferencesRouter(&stubListShippingConnections{result: twoShippingConnections()}, set)

	w := patchPreferences(r, map[string]any{"default_connection_id": pnID})

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, set.got)
	assert.Equal(t, pnID, *set.got)
	assert.Contains(t, w.Body.String(), `"connections":[`)
}

func TestUpdateShippingPreferences_nullClears(t *testing.T) {
	set := &stubSetDefaultShipping{}
	r := preferencesRouter(&stubListShippingConnections{result: twoShippingConnections()}, set)

	w := patchPreferences(r, map[string]any{"default_connection_id": nil})

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, set.called)
	assert.Nil(t, set.got)
}

func TestUpdateShippingPreferences_errorMapping(t *testing.T) {
	cases := map[string]struct {
		err  error
		want int
	}{
		"forbidden":   {shippingDomain.ErrShippingPreferencesForbidden, http.StatusForbidden},
		"not found":   {integrationsDomain.ErrConnectionNotFound, http.StatusNotFound},
		"cannot ship": {shippingDomain.ErrNotAShippingConnection, http.StatusUnprocessableEntity},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := preferencesRouter(&stubListShippingConnections{result: twoShippingConnections()}, &stubSetDefaultShipping{err: tc.err})

			w := patchPreferences(r, map[string]any{"default_connection_id": pnID})

			assert.Equal(t, tc.want, w.Code)
		})
	}
}

func TestUpdateShippingPreferences_400OnMalformedBody(t *testing.T) {
	set := &stubSetDefaultShipping{}
	r := preferencesRouter(&stubListShippingConnections{}, set)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/shipping/preferences", bytes.NewReader([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, set.called)
}
