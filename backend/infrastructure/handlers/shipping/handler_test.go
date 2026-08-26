package shipping_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genshipping "github.com/adirkuhn/commerceos/api/shipping"
	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
	shippinghandler "github.com/adirkuhn/commerceos/infrastructure/handlers/shipping"
)

type stubConnect struct {
	conn *integrationsDomain.Connection
	err  error
}

func (s *stubConnect) Execute(context.Context, appintegrations.ConnectMyParcelInput) (*integrationsDomain.Connection, error) {
	return s.conn, s.err
}

type stubRegisterWebhook struct{ err error }

func (s *stubRegisterWebhook) Execute(context.Context, string) error { return s.err }

type stubGetSettings struct {
	result *appshipping.ShippingSettingsResult
	err    error
}

func (s *stubGetSettings) Execute(context.Context, string) (*appshipping.ShippingSettingsResult, error) {
	return s.result, s.err
}

type stubUpdateSettings struct {
	result *shippingDomain.ShippingConnectionSettings
	err    error
}

func (s *stubUpdateSettings) Execute(context.Context, appshipping.UpdateShippingSettingsInput) (*shippingDomain.ShippingConnectionSettings, error) {
	return s.result, s.err
}

type stubUpsertOverride struct {
	result *shippingDomain.ShippingCarrierOverride
	err    error
}

func (s *stubUpsertOverride) Execute(context.Context, appshipping.UpsertCarrierOverrideInput) (*shippingDomain.ShippingCarrierOverride, error) {
	return s.result, s.err
}

type stubCreateShipment struct {
	result *shippingDomain.Shipment
	err    error
	gotIn  appshipping.CreateShipmentInput
}

func (s *stubCreateShipment) Execute(_ context.Context, in appshipping.CreateShipmentInput) (*shippingDomain.Shipment, error) {
	s.gotIn = in
	return s.result, s.err
}

type stubGetLabel struct {
	result *integrationsDomain.LabelResult
	err    error
}

func (s *stubGetLabel) Execute(context.Context, string) (*integrationsDomain.LabelResult, error) {
	return s.result, s.err
}

type stubListShipments struct {
	result []shippingDomain.Shipment
	err    error
}

func (s *stubListShipments) Execute(context.Context, string) ([]shippingDomain.Shipment, error) {
	return s.result, s.err
}

func init() { gin.SetMode(gin.TestMode) }

// newTestRouter mirrors main.go's real wiring exactly: a root engine, an
// "/api/v1" group, then shipping's own sub-group off that (v1.Group("") in
// main.go) -- NOT a bare root router. Registering directly on a bare root
// router hid the double-prefix bug (BaseURL "/api/v1" applied on top of a
// group already mounted at "/api/v1") since every request in this file
// still happened to resolve.
func newTestRouter(h *shippinghandler.Handler) *gin.Engine {
	r := gin.New()
	v1 := r.Group("/api/v1")
	shippingGroup := v1.Group("")
	genshipping.RegisterHandlersWithOptions(shippingGroup, h, genshipping.GinServerOptions{})
	return r
}

func TestConnectMyParcel_201OnSuccess(t *testing.T) {
	conn := &integrationsDomain.Connection{ID: "c-1", Provider: "myparcel", Status: "connected"}
	h := shippinghandler.NewHandler(&stubConnect{conn: conn}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{"api_key": "test-key"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/myparcel/connect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestCreateShipment_201OnSuccess(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", SalesOrderID: "order-1", Barcode: "3SMYPA1", StatusMain: "registered"}
	create := &stubCreateShipment{result: shipment}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, create, &stubGetLabel{}, &stubListShipments{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"sales_order_id": "11111111-1111-1111-1111-111111111111",
		"connection_id":  "22222222-2222-2222-2222-222222222222",
		"carrier":        1,
		"recipient":      map[string]any{"person": "Jane", "street": "Damrak", "city": "Amsterdam", "cc": "NL"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", create.gotIn.SalesOrderID)
	assert.Equal(t, "Jane", create.gotIn.Recipient.Person)
}

func TestCreateShipment_400OnMissingRecipient(t *testing.T) {
	create := &stubCreateShipment{err: shippingDomain.ErrMissingRecipient}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, create, &stubGetLabel{}, &stubListShipments{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"sales_order_id": "11111111-1111-1111-1111-111111111111",
		"connection_id":  "22222222-2222-2222-2222-222222222222",
		"carrier":        1,
		"recipient":      map[string]any{"person": "Jane", "street": "Damrak", "city": "Amsterdam", "cc": "NL"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateShipment_502OnMyParcelRejection(t *testing.T) {
	create := &stubCreateShipment{err: assertAnError}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, create, &stubGetLabel{}, &stubListShipments{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"sales_order_id": "11111111-1111-1111-1111-111111111111",
		"connection_id":  "22222222-2222-2222-2222-222222222222",
		"carrier":        1,
		"recipient":      map[string]any{"person": "Jane", "street": "Damrak", "city": "Amsterdam", "cc": "NL"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
}

func TestGetShipmentLabel_streamsRawBytesWithContentType(t *testing.T) {
	label := &integrationsDomain.LabelResult{Data: []byte("%PDF-fake"), ContentType: "application/pdf"}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{result: label}, &stubListShipments{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/label", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Equal(t, []byte("%PDF-fake"), w.Body.Bytes())
}

func TestRegisterMyParcelWebhook_409OnNoSecretSet(t *testing.T) {
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{err: shippingDomain.ErrWebhookSecretNotSet}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/connections/11111111-1111-1111-1111-111111111111/register-webhook", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
}

var assertAnError = &testError{"myparcel rejected the shipment"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
