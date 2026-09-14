package shipping_test

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
	gotIn  appshipping.UpdateShippingSettingsInput
}

func (s *stubUpdateSettings) Execute(_ context.Context, in appshipping.UpdateShippingSettingsInput) (*shippingDomain.ShippingConnectionSettings, error) {
	s.gotIn = in
	return s.result, s.err
}

type stubUpsertOverride struct {
	result *shippingDomain.ShippingCarrierOverride
	err    error
	gotIn  appshipping.UpsertCarrierOverrideInput
}

func (s *stubUpsertOverride) Execute(_ context.Context, in appshipping.UpsertCarrierOverrideInput) (*shippingDomain.ShippingCarrierOverride, error) {
	s.gotIn = in
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
	result  *integrationsDomain.LabelResult
	err     error
	gotSize string
}

func (s *stubGetLabel) Execute(_ context.Context, _ string, size string) (*integrationsDomain.LabelResult, error) {
	s.gotSize = size
	return s.result, s.err
}

type stubListShipments struct {
	result []shippingDomain.Shipment
	err    error
}

func (s *stubListShipments) Execute(context.Context, string) ([]shippingDomain.Shipment, error) {
	return s.result, s.err
}

type stubListTrackingEvents struct {
	result []shippingDomain.ShipmentStatusEvent
	err    error
}

func (s *stubListTrackingEvents) Execute(context.Context, string) ([]shippingDomain.ShipmentStatusEvent, error) {
	return s.result, s.err
}

type stubRefreshTracking struct {
	result shippingDomain.ShipmentStatusEvent
	err    error
}

func (s *stubRefreshTracking) Execute(context.Context, string) (shippingDomain.ShipmentStatusEvent, error) {
	return s.result, s.err
}

type stubExportMyParcelOrder struct {
	result *shippingDomain.MyParcelFulfilmentOrder
	err    error
}

func (s *stubExportMyParcelOrder) Execute(context.Context, appshipping.ExportMyParcelOrderInput) (*shippingDomain.MyParcelFulfilmentOrder, error) {
	return s.result, s.err
}

type stubLookupAddress struct {
	result         *shippingDomain.AddressLookupResult
	err            error
	gotPostalCode  string
	gotHouseNumber string
}

func (s *stubLookupAddress) Execute(_ context.Context, postalCode, houseNumber string) (*shippingDomain.AddressLookupResult, error) {
	s.gotPostalCode, s.gotHouseNumber = postalCode, houseNumber
	return s.result, s.err
}

type stubCancelShipment struct {
	result *appshipping.CancelShipmentResult
	err    error
	gotIn  appshipping.CancelShipmentInput
}

func (s *stubCancelShipment) Execute(_ context.Context, in appshipping.CancelShipmentInput) (*appshipping.CancelShipmentResult, error) {
	s.gotIn = in
	return s.result, s.err
}

type stubGetDeliveryEstimate struct {
	result []shippingDomain.DeliveryPriceOption
	err    error
	gotIn  appshipping.GetDeliveryPriceEstimateInput
}

func (s *stubGetDeliveryEstimate) Execute(_ context.Context, in appshipping.GetDeliveryPriceEstimateInput) ([]shippingDomain.DeliveryPriceOption, error) {
	s.gotIn = in
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
	h := shippinghandler.NewHandler(&stubConnect{conn: conn}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{"api_key": "test-key"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/myparcel/connect", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestUpdateShippingSettings_200WrapsResponseInDataEnvelope(t *testing.T) {
	settings := &shippingDomain.ShippingConnectionSettings{ConnectionID: "22222222-2222-2222-2222-222222222222", DefaultCarrier: 1, DefaultPackageType: 1, DefaultLabelFormat: "pdf"}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{result: settings}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{"default_carrier": 1, "default_package_type": 1, "default_label_format": "pdf"})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipping/connections/22222222-2222-2222-2222-222222222222/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &parsed))
	// Regression: this handler used to write the bare settings object with
	// no top-level "data" key, so the frontend's request<T>() -- which
	// always reads body.data -- silently resolved to undefined.
	require.Contains(t, parsed, "data")
	assert.NotNil(t, parsed["data"])
}

func TestUpdateShippingSettings_threadsExportOrderToMyParcelThrough(t *testing.T) {
	update := &stubUpdateSettings{result: &shippingDomain.ShippingConnectionSettings{}}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, update, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"default_carrier": 1, "default_package_type": 1, "default_label_format": "pdf", "export_order_to_myparcel": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipping/connections/22222222-2222-2222-2222-222222222222/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, update.gotIn.ExportOrderToMyParcel)
}

func TestUpsertCarrierOverride_200WrapsResponseInDataEnvelope(t *testing.T) {
	override := &shippingDomain.ShippingCarrierOverride{ConnectionID: "22222222-2222-2222-2222-222222222222", Carrier: 1}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{result: override}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{"package_type": nil, "signature_required": nil, "only_recipient": nil})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipping/connections/22222222-2222-2222-2222-222222222222/carrier-overrides/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &parsed))
	// Regression: same bare-object bug as UpdateShippingSettings above --
	// the frontend pushed the resulting `undefined` straight into its
	// overrides array, crashing the next render's .find() with
	// "Cannot read properties of undefined (reading 'carrier')".
	require.Contains(t, parsed, "data")
	assert.NotNil(t, parsed["data"])
}

func TestCreateShipment_201OnSuccess(t *testing.T) {
	shipment := &shippingDomain.Shipment{ID: "s-1", SalesOrderID: "order-1", Barcode: "3SMYPA1", StatusMain: "registered"}
	create := &stubCreateShipment{result: shipment}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, create, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
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
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, create, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
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

func TestCreateShipment_422OnUnsupportedPackageTypeForCarrier(t *testing.T) {
	create := &stubCreateShipment{err: shippingDomain.ErrUnsupportedPackageTypeForCarrier}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, create, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"sales_order_id": "11111111-1111-1111-1111-111111111111",
		"connection_id":  "22222222-2222-2222-2222-222222222222",
		"carrier":        1,
		"package_type":   4,
		"recipient":      map[string]any{"person": "Jane", "street": "Damrak", "city": "Amsterdam", "cc": "NL"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateShipment_502OnMyParcelRejection(t *testing.T) {
	create := &stubCreateShipment{err: assertAnError}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, create, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
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
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{result: label}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/label", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Equal(t, []byte("%PDF-fake"), w.Body.Bytes())
}

// TestGetShipmentLabel_passesSizeQueryParamThrough is a regression test for
// the A6-print feature: ?size=A6 on this GET request must reach the use
// case's Execute call, not get silently dropped.
func TestGetShipmentLabel_passesSizeQueryParamThrough(t *testing.T) {
	label := &integrationsDomain.LabelResult{Data: []byte("%PDF-fake"), ContentType: "application/pdf"}
	getLabel := &stubGetLabel{result: label}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, getLabel, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/label?size=A6", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "A6", getLabel.gotSize)
}

func TestRegisterMyParcelWebhook_409OnNoSecretSet(t *testing.T) {
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{err: shippingDomain.ErrWebhookSecretNotSet}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/connections/11111111-1111-1111-1111-111111111111/register-webhook", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdateShippingSettings_passesInsuranceAndFulfilmentConfigThrough(t *testing.T) {
	accountID, shopID := 111, 222
	settings := &shippingDomain.ShippingConnectionSettings{
		ConnectionID: "22222222-2222-2222-2222-222222222222", DefaultCarrier: 1, DefaultPackageType: 1, DefaultLabelFormat: "pdf",
		InsureByDefault: true, MyParcelAccountID: &accountID, MyParcelShopID: &shopID, AutoPrintEnabled: true,
	}
	update := &stubUpdateSettings{result: settings}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, update, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"default_carrier": 1, "default_package_type": 1, "default_label_format": "pdf",
		"insure_by_default": true, "myparcel_account_id": 111, "myparcel_shop_id": 222, "auto_print_enabled": true,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipping/connections/22222222-2222-2222-2222-222222222222/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, update.gotIn.InsureByDefault)
	require.NotNil(t, update.gotIn.MyParcelAccountID)
	assert.Equal(t, 111, *update.gotIn.MyParcelAccountID)
	assert.True(t, update.gotIn.AutoPrintEnabled)
	assert.Contains(t, w.Body.String(), `"auto_print_enabled":true`)
	assert.Contains(t, w.Body.String(), `"myparcel_account_id":111`)
}

func TestUpsertCarrierOverride_passesInsureByDefaultThrough(t *testing.T) {
	yes := true
	override := &shippingDomain.ShippingCarrierOverride{ConnectionID: "22222222-2222-2222-2222-222222222222", Carrier: 1, InsureByDefault: &yes}
	upsert := &stubUpsertOverride{result: override}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, upsert, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{"insure_by_default": true})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/shipping/connections/22222222-2222-2222-2222-222222222222/carrier-overrides/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, upsert.gotIn.InsureByDefault)
	assert.True(t, *upsert.gotIn.InsureByDefault)
	assert.Contains(t, w.Body.String(), `"insure_by_default":true`)
}

func TestExportMyParcelOrder_201OnSuccess(t *testing.T) {
	order := &shippingDomain.MyParcelFulfilmentOrder{
		ID: "fulfilment-1", ConnectionID: "22222222-2222-2222-2222-222222222222", SalesOrderID: "11111111-1111-1111-1111-111111111111",
		MyParcelUUID: "order-uuid-1", Status: "open",
	}
	export := &stubExportMyParcelOrder{result: order}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, export, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"carrier":   1,
		"recipient": map[string]any{"person": "Jane", "street": "Damrak", "city": "Amsterdam", "cc": "NL"},
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/shipping/connections/22222222-2222-2222-2222-222222222222/sales-orders/11111111-1111-1111-1111-111111111111/export-order",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "order-uuid-1")
}

func TestExportMyParcelOrder_502OnMyParcelRejection(t *testing.T) {
	export := &stubExportMyParcelOrder{err: assertAnError}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, export, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	body, _ := json.Marshal(map[string]any{
		"carrier":   1,
		"recipient": map[string]any{"person": "Jane", "street": "Damrak", "city": "Amsterdam", "cc": "NL"},
	})
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/shipping/connections/22222222-2222-2222-2222-222222222222/sales-orders/11111111-1111-1111-1111-111111111111/export-order",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
}

var assertAnError = &testError{"myparcel rejected the shipment"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func TestListShipmentTrackingEvents_200WithLabel(t *testing.T) {
	occurredAt := time.Now()
	listEvents := &stubListTrackingEvents{result: []shippingDomain.ShipmentStatusEvent{
		{ID: "evt-1", ShipmentID: "ship-1", Status: 3, StatusMain: "enroute", Source: shippingDomain.StatusEventSourceWebhook, OccurredAt: occurredAt},
	}}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, listEvents, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/tracking-events", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":3`)
	assert.Contains(t, w.Body.String(), `"status_label":"Enroute - handed to carrier"`)
	assert.Contains(t, w.Body.String(), `"source":"webhook"`)
}

func TestRefreshShipmentTracking_200OnSuccess(t *testing.T) {
	refresh := &stubRefreshTracking{result: shippingDomain.ShipmentStatusEvent{
		ID: "evt-2", ShipmentID: "ship-1", Status: 7, StatusMain: "delivered", Source: shippingDomain.StatusEventSourceManual,
	}}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, refresh, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/tracking-events/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status_label":"Delivered - at recipient"`)
	assert.Contains(t, w.Body.String(), `"source":"manual"`)
}

func TestRefreshShipmentTracking_404WhenShipmentNotFound(t *testing.T) {
	refresh := &stubRefreshTracking{err: shippingDomain.ErrShipmentNotFound}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, refresh, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/tracking-events/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRefreshShipmentTracking_502WhenProviderUnreachable(t *testing.T) {
	refresh := &stubRefreshTracking{err: assertAnError}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, refresh, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/tracking-events/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadGateway, w.Code)
}

func TestLookupAddress_200OnMatch(t *testing.T) {
	lookup := &stubLookupAddress{result: &shippingDomain.AddressLookupResult{Street: "Damrak", City: "Amsterdam"}}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, lookup, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/address-lookup?postal_code=1012AB&number=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1012AB", lookup.gotPostalCode)
	assert.Equal(t, "1", lookup.gotHouseNumber)
	assert.Contains(t, w.Body.String(), `"street":"Damrak"`)
	assert.Contains(t, w.Body.String(), `"city":"Amsterdam"`)
}

func TestLookupAddress_404WhenNotFound(t *testing.T) {
	lookup := &stubLookupAddress{err: shippingDomain.ErrAddressNotFound}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, lookup, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/address-lookup?postal_code=9999ZZ&number=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestLookupAddress_400WhenMissingRequiredQueryParam(t *testing.T) {
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/address-lookup?postal_code=1012AB", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCancelShipment_200ReportsOutcomeEvenWhenMyParcelRejected(t *testing.T) {
	cancel := &stubCancelShipment{result: &appshipping.CancelShipmentResult{
		DeletedOnMyParcel: false, Message: "MyParcel could not delete this shipment (422); marked as cancelled locally.",
	}}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, cancel, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "MyParcel rejecting the delete must still be a 200 -- staff can proceed regardless")
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", cancel.gotIn.ShipmentID)
	assert.Contains(t, w.Body.String(), `"deleted_on_myparcel":false`)
	assert.Contains(t, w.Body.String(), "marked as cancelled locally")
}

func TestCancelShipment_404WhenShipmentNotFound(t *testing.T) {
	cancel := &stubCancelShipment{err: shippingDomain.ErrShipmentNotFound}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, cancel, &stubGetDeliveryEstimate{})
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/shipping/shipments/11111111-1111-1111-1111-111111111111/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetDeliveryPriceEstimate_200WrapsPricedOptionsInDataEnvelope(t *testing.T) {
	estimate := &stubGetDeliveryEstimate{result: []shippingDomain.DeliveryPriceOption{
		{PriceComment: "standard", AmountMinorUnits: 1000, Currency: "EUR"},
	}}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, estimate)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/delivery-price-estimate?connection_id=22222222-2222-2222-2222-222222222222&cc=NL&postal_code=1012AB&number=1&carrier=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", estimate.gotIn.ConnectionID)
	assert.Equal(t, "NL", estimate.gotIn.CC)
	assert.Equal(t, "1012AB", estimate.gotIn.PostalCode)
	assert.Equal(t, "1", estimate.gotIn.Number)
	assert.Equal(t, shippingDomain.CarrierPostNL, estimate.gotIn.Carrier)
	assert.Contains(t, w.Body.String(), `"price_comment":"standard"`)
	assert.Contains(t, w.Body.String(), `"amount_minor_units":1000`)
}

func TestGetDeliveryPriceEstimate_422WhenCountryUnsupported(t *testing.T) {
	estimate := &stubGetDeliveryEstimate{err: shippingDomain.ErrDeliveryEstimateUnsupportedCountry}
	h := shippinghandler.NewHandler(&stubConnect{}, &stubRegisterWebhook{}, &stubGetSettings{}, &stubUpdateSettings{}, &stubUpsertOverride{}, &stubCreateShipment{}, &stubGetLabel{}, &stubListShipments{}, &stubExportMyParcelOrder{}, &stubListTrackingEvents{}, &stubRefreshTracking{}, &stubLookupAddress{}, &stubCancelShipment{}, estimate)
	r := newTestRouter(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/shipping/delivery-price-estimate?connection_id=22222222-2222-2222-2222-222222222222&cc=DE&postal_code=10115&number=1&carrier=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
