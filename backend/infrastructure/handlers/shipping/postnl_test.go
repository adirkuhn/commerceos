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
		nil, shippinghandler.NewPostNLHandler(connect, &stubGetPostNL{}, &stubUpdatePostNL{}))
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

const pnConnID = "33333333-3333-3333-3333-333333333333"

type stubGetPostNL struct{ err error }

func (s *stubGetPostNL) Execute(context.Context, string) (*integrationsDomain.Connection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &integrationsDomain.Connection{ID: pnConnID, Provider: integrationsDomain.ProviderPostNL, PostNLCustomerCode: "SLTQ", PostNLCustomerNumber: "10979997", PostNLSandbox: true}, nil
}

type stubUpdatePostNL struct {
	gotID string
	got   appintegrations.UpdatePostNLConnectionInput
	err   error
}

func (s *stubUpdatePostNL) Execute(_ context.Context, id string, in appintegrations.UpdatePostNLConnectionInput) (*integrationsDomain.Connection, error) {
	s.gotID, s.got = id, in
	if s.err != nil {
		return nil, s.err
	}
	return &integrationsDomain.Connection{ID: id, Provider: integrationsDomain.ProviderPostNL, PostNLCustomerCode: in.CustomerCode, PostNLCustomerNumber: in.CustomerNumber, PostNLSandbox: in.Sandbox}, nil
}

func postnlAccountRouter(get *stubGetPostNL, update *stubUpdatePostNL) http.Handler {
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{},
		nil, shippinghandler.NewPostNLHandler(&stubConnectPostNL{}, get, update))
	return newTestRouter(h)
}

func TestGetPostNLConnection_200WithoutTheKey(t *testing.T) {
	w := httptest.NewRecorder()
	postnlAccountRouter(&stubGetPostNL{}, &stubUpdatePostNL{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/shipping/postnl/connections/"+pnConnID, nil))

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"customer_code":"SLTQ"`)
	assert.Contains(t, w.Body.String(), `"sandbox":true`)
	assert.NotContains(t, w.Body.String(), "api_key")
}

func TestGetPostNLConnection_422ForOtherProvider(t *testing.T) {
	w := httptest.NewRecorder()
	postnlAccountRouter(&stubGetPostNL{err: integrationsDomain.ErrNotPostNLConnection}, &stubUpdatePostNL{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/shipping/postnl/connections/"+pnConnID, nil))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func putPostNL(r http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipping/postnl/connections/"+pnConnID, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdatePostNLConnection_200(t *testing.T) {
	update := &stubUpdatePostNL{}
	w := putPostNL(postnlAccountRouter(&stubGetPostNL{}, update), `{"api_key":"","customer_code":"SLTQ","customer_number":"10979997","sandbox":false}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, pnConnID, update.gotID)
	assert.Equal(t, appintegrations.UpdatePostNLConnectionInput{CustomerCode: "SLTQ", CustomerNumber: "10979997"}, update.got)
	assert.Contains(t, w.Body.String(), `"sandbox":false`)
}

func TestUpdatePostNLConnection_errors(t *testing.T) {
	for name, tc := range map[string]struct {
		err  error
		want int
	}{
		"invalid":   {integrationsDomain.ErrPostNLAccountInvalid, http.StatusUnprocessableEntity},
		"not found": {integrationsDomain.ErrConnectionNotFound, http.StatusNotFound},
		"other":     {integrationsDomain.ErrNotPostNLConnection, http.StatusUnprocessableEntity},
	} {
		t.Run(name, func(t *testing.T) {
			w := putPostNL(postnlAccountRouter(&stubGetPostNL{}, &stubUpdatePostNL{err: tc.err}), `{"api_key":"","customer_code":"SLTQ","customer_number":"1","sandbox":true}`)
			assert.Equal(t, tc.want, w.Code)
		})
	}
	assert.Equal(t, http.StatusBadRequest, putPostNL(postnlAccountRouter(&stubGetPostNL{}, &stubUpdatePostNL{}), `{`).Code)
}
