package shipping_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippinghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/shipping"
)

type stubConnectPostNL struct {
	got appintegrations.ConnectPostNLInput
	err error
}

func (s *stubConnectPostNL) Execute(_ context.Context, in appintegrations.ConnectPostNLInput) (*integrationsDomain.Connection, error) {
	s.got = in
	if s.err != nil {
		return nil, s.err
	}
	return &integrationsDomain.Connection{ID: "c-pn", Provider: integrationsDomain.ProviderPostNL, Status: integrationsDomain.StatusConnected}, nil
}

func postnlRouter(connect *stubConnectPostNL) http.Handler {
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{},
		nil, shippinghandler.NewPostNLHandler(connect))
	return newTestRouter(h)
}

func postConnectPostNL(r http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/postnl/connect", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestConnectPostNL_201(t *testing.T) {
	connect := &stubConnectPostNL{}
	w := postConnectPostNL(postnlRouter(connect), `{"api_key":"k","customer_code":"DEVC","customer_number":"11223344","sandbox":true}`)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, appintegrations.ConnectPostNLInput{APIKey: "k", CustomerCode: "DEVC", CustomerNumber: "11223344", Sandbox: true}, connect.got)
	var body struct {
		Data struct {
			ConnectionID string `json:"connection_id"`
			Provider     string `json:"provider"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "c-pn", body.Data.ConnectionID)
	assert.Equal(t, "postnl", body.Data.Provider)
}

func TestConnectPostNL_422OnInvalidAccount(t *testing.T) {
	w := postConnectPostNL(postnlRouter(&stubConnectPostNL{err: integrationsDomain.ErrPostNLAccountInvalid}), `{"api_key":"k","customer_code":"DEVC","customer_number":"1","sandbox":false}`)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "POSTNL_ACCOUNT_INVALID")
}

func TestConnectPostNL_400OnMalformedBody(t *testing.T) {
	connect := &stubConnectPostNL{}
	w := postConnectPostNL(postnlRouter(connect), `{`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Empty(t, connect.got.APIKey)
}
