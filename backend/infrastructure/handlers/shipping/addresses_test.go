package shipping_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	shippinghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/shipping"
)

type stubGetAddresses struct {
	result *appshipping.ShippingAddresses
}

func (s *stubGetAddresses) Execute(context.Context) (*appshipping.ShippingAddresses, error) {
	if s.result == nil {
		return &appshipping.ShippingAddresses{}, nil
	}
	return s.result, nil
}

type stubUpdateAddresses struct {
	got    appshipping.UpdateShippingAddressesInput
	called bool
	err    error
}

func (s *stubUpdateAddresses) Execute(_ context.Context, in appshipping.UpdateShippingAddressesInput) error {
	s.called, s.got = true, in
	return s.err
}

func addressesRouter(get *stubGetAddresses, update *stubUpdateAddresses) http.Handler {
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{},
		shippinghandler.NewPreferencesHandler(&stubListShippingConnections{}, &stubSetDefaultShipping{}, get, update), nil)
	return newTestRouter(h)
}

func TestGetShippingAddresses_200(t *testing.T) {
	company := &shippingDomain.ShippingAddress{Company: "Cantinho", Street: "Damrak", Number: "1", PostalCode: "1012AB", City: "Amsterdam", CC: "NL"}
	r := addressesRouter(&stubGetAddresses{result: &appshipping.ShippingAddresses{CompanyAddress: company}}, &stubUpdateAddresses{})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/shipping/addresses", nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sender_address":null`)
	assert.Contains(t, w.Body.String(), `"return_address":null`)
	assert.Contains(t, w.Body.String(), `"company":"Cantinho"`)
}

func putAddresses(r http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipping/addresses", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateShippingAddresses_200PassesBothAndReturnsSaved(t *testing.T) {
	update := &stubUpdateAddresses{}
	r := addressesRouter(&stubGetAddresses{}, update)

	w := putAddresses(r, `{"sender_address":{"company":"WH","street":"Kade","number":"9","number_suffix":"B","postal_code":"3011AA","city":"Rotterdam","cc":"NL","email":"wh@example.com"},"return_address":null}`)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, update.got.Sender)
	assert.Equal(t, shippingDomain.ShippingAddress{Company: "WH", Street: "Kade", Number: "9", NumberSuffix: "B", PostalCode: "3011AA", City: "Rotterdam", CC: "NL", Email: "wh@example.com"}, *update.got.Sender)
	assert.Nil(t, update.got.Return)
}

func TestUpdateShippingAddresses_errorMapping(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"forbidden": {shippingDomain.ErrShippingPreferencesForbidden, http.StatusForbidden},
		"invalid":   {shippingDomain.ErrShippingAddressInvalid, http.StatusUnprocessableEntity},
	} {
		t.Run(name, func(t *testing.T) {
			r := addressesRouter(&stubGetAddresses{}, &stubUpdateAddresses{err: tc.err})

			w := putAddresses(r, `{"sender_address":null,"return_address":null}`)

			assert.Equal(t, tc.want, w.Code)
		})
	}
}

func TestUpdateShippingAddresses_400OnMalformedBody(t *testing.T) {
	update := &stubUpdateAddresses{}
	w := putAddresses(addressesRouter(&stubGetAddresses{}, update), `{`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, update.called)
}
