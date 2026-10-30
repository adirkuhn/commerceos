package moneybird_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genmoneybird "github.com/adirkuhn/commerceos/api/moneybird"
	appmb "github.com/adirkuhn/commerceos/application/moneybird"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mb "github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/domain/oms"
	moneybirdhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/moneybird"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

const jwtSecret = "test-secret"

const orderID = "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b"

type stubs struct {
	connectIn  appmb.ConnectInput
	connectErr error
	setup      *appmb.Setup
	setupErr   error
	updateIn   appmb.UpdateSettingsInput
	updateErr  error
	sentOrder  string
	sentQuote  string
	sendErr    error
	status     *appmb.DocumentStatus
	statusType mb.SourceType
}

func (s *stubs) Connect(_ context.Context, in appmb.ConnectInput) (*integrationsDomain.Connection, error) {
	s.connectIn = in
	if s.connectErr != nil {
		return nil, s.connectErr
	}
	return &integrationsDomain.Connection{ID: "conn-1"}, nil
}
func (s *stubs) Setup(context.Context) (*appmb.Setup, error) { return s.setup, s.setupErr }
func (s *stubs) Update(_ context.Context, in appmb.UpdateSettingsInput) (*mb.Settings, error) {
	s.updateIn = in
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	out := mb.Settings{ConnectionID: "conn-1", AutoSync: in.AutoSync != nil && *in.AutoSync}
	return &out, nil
}
func (s *stubs) SendOrder(_ context.Context, id string) (*mb.Document, error) {
	s.sentOrder = id
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	return &mb.Document{SourceType: mb.SourceSalesOrder, SourceID: id, MoneybirdID: "inv-1", Reference: "1234", Status: mb.DocumentSynced, PaymentRegistered: true, SyncedAt: time.Now()}, nil
}
func (s *stubs) SendQuote(_ context.Context, id string) (*mb.Document, error) {
	s.sentQuote = id
	if s.sendErr != nil {
		return nil, s.sendErr
	}
	return &mb.Document{SourceType: mb.SourceQuote, SourceID: id, MoneybirdID: "est-1", Status: mb.DocumentSynced, SyncedAt: time.Now()}, nil
}
func (s *stubs) Status(_ context.Context, t mb.SourceType, _ string) (*appmb.DocumentStatus, error) {
	s.statusType = t
	return s.status, nil
}

func router(t *testing.T, s *stubs) *gin.Engine {
	t.Helper()
	spec, err := genmoneybird.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte(jwtSecret)))
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	moneybirdhandler.RegisterRoutes(g, moneybirdhandler.NewHandler(moneybirdhandler.UseCases{
		Connect: connectFn(s.Connect), Setup: setupFn(s.Setup), Update: updateFn(s.Update),
		SendOrder: sendFn(s.SendOrder), SendQuote: sendFn(s.SendQuote), Status: statusFn(s.Status),
	}))
	return r
}

type connectFn func(context.Context, appmb.ConnectInput) (*integrationsDomain.Connection, error)

func (f connectFn) Execute(ctx context.Context, in appmb.ConnectInput) (*integrationsDomain.Connection, error) {
	return f(ctx, in)
}

type setupFn func(context.Context) (*appmb.Setup, error)

func (f setupFn) Execute(ctx context.Context) (*appmb.Setup, error) { return f(ctx) }

type updateFn func(context.Context, appmb.UpdateSettingsInput) (*mb.Settings, error)

func (f updateFn) Execute(ctx context.Context, in appmb.UpdateSettingsInput) (*mb.Settings, error) {
	return f(ctx, in)
}

type sendFn func(context.Context, string) (*mb.Document, error)

func (f sendFn) Execute(ctx context.Context, id string) (*mb.Document, error) { return f(ctx, id) }

type statusFn func(context.Context, mb.SourceType, string) (*appmb.DocumentStatus, error)

func (f statusFn) Execute(ctx context.Context, t mb.SourceType, id string) (*appmb.DocumentStatus, error) {
	return f(ctx, t, id)
}

func jwtFor(t *testing.T, role middleware.Role) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{
		UserID: "user-1", TenantID: "tenant-1", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	s, err := tok.SignedString([]byte(jwtSecret))
	require.NoError(t, err)
	return s
}

func call(t *testing.T, r *gin.Engine, method, path, body string, role middleware.Role) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if role != "" {
		req.Header.Set("Authorization", "Bearer "+jwtFor(t, role))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error.Code
}

func TestConnect(t *testing.T) {
	s := &stubs{}
	w := call(t, router(t, s), http.MethodPost, "/api/v1/integrations/moneybird/connect", `{"api_token":"tok-1"}`, middleware.RoleCustomerOwner)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "tok-1", s.connectIn.APIToken)
	assert.JSONEq(t, `{"data":{"connection_id":"conn-1"},"error":null}`, w.Body.String())
}

func TestConnect_errors(t *testing.T) {
	s := &stubs{}
	r := router(t, s)

	assert.Equal(t, http.StatusUnauthorized, call(t, r, http.MethodPost, "/api/v1/integrations/moneybird/connect", `{"api_token":"x"}`, "").Code)
	assert.Equal(t, http.StatusForbidden, call(t, r, http.MethodPost, "/api/v1/integrations/moneybird/connect", `{"api_token":"x"}`, middleware.RoleCustomerUser).Code)
	assert.Equal(t, http.StatusBadRequest, call(t, r, http.MethodPost, "/api/v1/integrations/moneybird/connect", `{}`, middleware.RoleCustomerOwner).Code, "contract: api_token required")

	s.connectErr = mb.ErrTokenInvalid
	w := call(t, r, http.MethodPost, "/api/v1/integrations/moneybird/connect", `{"api_token":"bad"}`, middleware.RoleCustomerOwner)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "MONEYBIRD_TOKEN_INVALID", errorCode(t, w))
}

func TestGetSetup(t *testing.T) {
	s := &stubs{setup: &appmb.Setup{
		ConnectionID: "conn-1",
		Settings: mb.Settings{ConnectionID: "conn-1", AdministrationID: "adm-1", AdministrationName: "Cantinho", DefaultLedgerAccountID: "led-1",
			TaxRates: []mb.TaxRateMapping{{RateBasisPoints: 900, MoneybirdTaxRateID: "tr-9"}}},
		Administrations: []mb.Administration{{ID: "adm-1", Name: "Cantinho", Currency: "EUR"}},
		TaxRates:        []mb.TaxRate{{ID: "tr-9", Name: "9%", Percentage: "9.0"}},
		LedgerAccounts:  []mb.LedgerAccount{{ID: "led-1", Name: "Omzet"}},
		PaymentMethods:  []string{"cash"},
	}}

	w := call(t, router(t, s), http.MethodGet, "/api/v1/integrations/moneybird", "", middleware.RoleCustomerUser)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.JSONEq(t, `{"data":{
		"settings":{"connection_id":"conn-1","administration_id":"adm-1","administration_name":"Cantinho","default_ledger_account_id":"led-1","auto_sync":false,
			"tax_rates":[{"rate_basis_points":900,"moneybird_tax_rate_id":"tr-9"}],"payment_methods":[]},
		"administrations":[{"id":"adm-1","name":"Cantinho","currency":"EUR"}],
		"tax_rates":[{"id":"tr-9","name":"9%","percentage":"9.0"}],
		"ledger_accounts":[{"id":"led-1","name":"Omzet"}],
		"payment_methods":["cash"]},"error":null}`, w.Body.String())
}

func TestGetSetup_notConnected(t *testing.T) {
	w := call(t, router(t, &stubs{setupErr: mb.ErrNotConnected}), http.MethodGet, "/api/v1/integrations/moneybird", "", middleware.RoleCustomerOwner)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "MONEYBIRD_NOT_CONNECTED", errorCode(t, w))
}

func TestUpdateSettings_sendsOnlyTheFieldsGiven(t *testing.T) {
	s := &stubs{}
	w := call(t, router(t, s), http.MethodPatch, "/api/v1/integrations/moneybird/settings", `{"auto_sync":true}`, middleware.RoleCustomerAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, s.updateIn.AutoSync)
	assert.True(t, *s.updateIn.AutoSync)
	assert.Nil(t, s.updateIn.TaxRates)
	assert.Nil(t, s.updateIn.PaymentMethods)
	assert.Nil(t, s.updateIn.AdministrationID)
}

func TestUpdateSettings_mappings(t *testing.T) {
	s := &stubs{}
	w := call(t, router(t, s), http.MethodPatch, "/api/v1/integrations/moneybird/settings",
		`{"tax_rates":[{"rate_basis_points":2100,"moneybird_tax_rate_id":"tr-21"}],"payment_methods":[{"method":"bacs","moneybird_payment_method":"bank_transfer"}]}`,
		middleware.RoleCustomerOwner)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []mb.TaxRateMapping{{RateBasisPoints: 2100, MoneybirdTaxRateID: "tr-21"}}, *s.updateIn.TaxRates)
	assert.Equal(t, []mb.PaymentMethodMapping{{Method: "bacs", MoneybirdPaymentMethod: "bank_transfer"}}, *s.updateIn.PaymentMethods)
}

func TestUpdateSettings_errors(t *testing.T) {
	s := &stubs{}
	r := router(t, s)
	assert.Equal(t, http.StatusForbidden, call(t, r, http.MethodPatch, "/api/v1/integrations/moneybird/settings", `{"auto_sync":true}`, middleware.RoleCustomerUser).Code)
	assert.Equal(t, http.StatusBadRequest, call(t, r, http.MethodPatch, "/api/v1/integrations/moneybird/settings",
		`{"payment_methods":[{"method":"bacs","moneybird_payment_method":"bitcoin"}]}`, middleware.RoleCustomerOwner).Code, "contract: unknown Moneybird method")

	s.updateErr = fmt.Errorf("wrapped: %w", appmb.ErrUnknownMoneybirdID)
	w := call(t, r, http.MethodPatch, "/api/v1/integrations/moneybird/settings", `{"default_ledger_account_id":"x"}`, middleware.RoleCustomerOwner)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "MONEYBIRD_UNKNOWN_ID", errorCode(t, w))
}

func TestSendSalesOrder(t *testing.T) {
	s := &stubs{}
	w := call(t, router(t, s), http.MethodPost, "/api/v1/integrations/moneybird/sales-orders/"+orderID+"/send", "", middleware.RoleCustomerOwner)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, orderID, s.sentOrder)
	var body struct {
		Data struct {
			MoneybirdID       string `json:"moneybird_id"`
			Status            string `json:"status"`
			PaymentRegistered bool   `json:"payment_registered"`
			SourceType        string `json:"source_type"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "inv-1", body.Data.MoneybirdID)
	assert.Equal(t, "synced", body.Data.Status)
	assert.True(t, body.Data.PaymentRegistered)
	assert.Equal(t, "sales_order", body.Data.SourceType)
}

func TestSendSalesOrder_errors(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{oms.ErrSalesOrderNotFound, http.StatusNotFound, "SALES_ORDER_NOT_FOUND"},
		{mb.ErrNotConnected, http.StatusNotFound, "MONEYBIRD_NOT_CONNECTED"},
		{fmt.Errorf("x: %w", mb.ErrDuplicateReference), http.StatusConflict, "MONEYBIRD_DUPLICATE_REFERENCE"},
		{fmt.Errorf("x: %w", mb.ErrTaxRateNotMapped), http.StatusUnprocessableEntity, "MONEYBIRD_TAX_RATE_NOT_MAPPED"},
		{fmt.Errorf("x: %w", mb.ErrPaymentMethodNotMapped), http.StatusUnprocessableEntity, "MONEYBIRD_PAYMENT_METHOD_NOT_MAPPED"},
		{fmt.Errorf("x: %w", mb.ErrTotalsDontAddUp), http.StatusUnprocessableEntity, "MONEYBIRD_TOTALS_DONT_ADD_UP"},
		{mb.ErrNotConfigured, http.StatusUnprocessableEntity, "MONEYBIRD_NOT_CONFIGURED"},
		{fmt.Errorf("x: %w", mb.ErrRequestFailed), http.StatusBadGateway, "MONEYBIRD_REQUEST_FAILED"},
		{mb.ErrRateLimited, http.StatusServiceUnavailable, "MONEYBIRD_RATE_LIMITED"},
	}
	for _, tc := range cases {
		w := call(t, router(t, &stubs{sendErr: tc.err}), http.MethodPost, "/api/v1/integrations/moneybird/sales-orders/"+orderID+"/send", "", middleware.RoleCustomerOwner)
		assert.Equal(t, tc.status, w.Code, tc.code)
		assert.Equal(t, tc.code, errorCode(t, w))
	}
	r := router(t, &stubs{})
	assert.Equal(t, http.StatusForbidden, call(t, r, http.MethodPost, "/api/v1/integrations/moneybird/sales-orders/"+orderID+"/send", "", middleware.RoleCustomerUser).Code)
	assert.Equal(t, http.StatusUnauthorized, call(t, r, http.MethodPost, "/api/v1/integrations/moneybird/sales-orders/"+orderID+"/send", "", "").Code)
	assert.Equal(t, http.StatusBadRequest, call(t, r, http.MethodPost, "/api/v1/integrations/moneybird/sales-orders/not-a-uuid/send", "", middleware.RoleCustomerOwner).Code)
}

func TestSendQuote_locked(t *testing.T) {
	s := &stubs{}
	w := call(t, router(t, s), http.MethodPost, "/api/v1/integrations/moneybird/quotes/"+orderID+"/send", "", middleware.RoleCustomerOwner)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, orderID, s.sentQuote)

	s.sendErr = fmt.Errorf("x: %w", mb.ErrDocumentLocked)
	w = call(t, router(t, s), http.MethodPost, "/api/v1/integrations/moneybird/quotes/"+orderID+"/send", "", middleware.RoleCustomerOwner)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "MONEYBIRD_DOCUMENT_LOCKED", errorCode(t, w))
}

func TestDocumentStatus(t *testing.T) {
	s := &stubs{status: &appmb.DocumentStatus{AdministrationID: "adm-1", AutoSync: true}}
	w := call(t, router(t, s), http.MethodGet, "/api/v1/integrations/moneybird/quotes/"+orderID, "", middleware.RoleCustomerUser)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, mb.SourceQuote, s.statusType)
	assert.JSONEq(t, `{"data":{"administration_id":"adm-1","auto_sync":true,"document":null},"error":null}`, w.Body.String())

	s.status = &appmb.DocumentStatus{AdministrationID: "adm-1", Document: &mb.Document{SourceType: mb.SourceSalesOrder, SourceID: orderID, Status: mb.DocumentFailed, Error: "boom"}}
	w = call(t, router(t, s), http.MethodGet, "/api/v1/integrations/moneybird/sales-orders/"+orderID, "", middleware.RoleCustomerUser)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, mb.SourceSalesOrder, s.statusType)
	assert.Contains(t, w.Body.String(), `"error":"boom"`)
}

func TestUpdateSettings_ordersWithoutAPaymentMethod(t *testing.T) {
	s := &stubs{}
	w := call(t, router(t, s), http.MethodPatch, "/api/v1/integrations/moneybird/settings",
		`{"payment_methods":[{"method":"","moneybird_payment_method":"bank_transfer"}]}`, middleware.RoleCustomerOwner)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, []mb.PaymentMethodMapping{{Method: "", MoneybirdPaymentMethod: "bank_transfer"}}, *s.updateIn.PaymentMethods)
}
