package orderdifferences_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	ginmiddleware "github.com/oapi-codegen/gin-middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genorderdifferences "github.com/adirkuhn/commerceos/api/orderdifferences"
	appchannels "github.com/adirkuhn/commerceos/application/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	orderdifferenceshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/orderdifferences"
)

func init() { gin.SetMode(gin.TestMode) }

type stubList struct {
	out             []omsDomain.OpenLineDiscrepancy
	total           int64
	err             error
	gotPage, gotPer int
}

func (s *stubList) Execute(_ context.Context, page, perPage int) ([]omsDomain.OpenLineDiscrepancy, int64, error) {
	s.gotPage, s.gotPer = page, perPage
	return s.out, s.total, s.err
}

type stubDismiss struct {
	err error
	got string
}

func (s *stubDismiss) Execute(_ context.Context, id string) error {
	s.got = id
	return s.err
}

type stubAudit struct {
	out *appchannels.AuditWooOrderDifferencesResult
	err error
	got appchannels.AuditWooOrderDifferencesInput
}

func (s *stubAudit) Execute(_ context.Context, in appchannels.AuditWooOrderDifferencesInput) (*appchannels.AuditWooOrderDifferencesResult, error) {
	s.got = in
	return s.out, s.err
}

type stubs struct {
	list    *stubList
	dismiss *stubDismiss
	audit   *stubAudit
}

func newRouter(t *testing.T, s stubs) *gin.Engine {
	t.Helper()
	if s.list == nil {
		s.list = &stubList{}
	}
	if s.dismiss == nil {
		s.dismiss = &stubDismiss{}
	}
	if s.audit == nil {
		s.audit = &stubAudit{out: &appchannels.AuditWooOrderDifferencesResult{}}
	}
	spec, err := genorderdifferences.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genorderdifferences.RegisterHandlers(g, orderdifferenceshandler.NewHandler(s.list, s.dismiss, s.audit))
	return r
}

func do(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	r.ServeHTTP(w, req)
	return w
}

func TestListOrderDifferences_returnsContractShape(t *testing.T) {
	detected := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	occurred := time.Date(2026, 10, 6, 10, 20, 0, 0, time.UTC)
	list := &stubList{total: 1, out: []omsDomain.OpenLineDiscrepancy{{
		LineDiscrepancy: omsDomain.LineDiscrepancy{
			ID: "d-1", SalesOrderID: "so-1", Kind: omsDomain.LineDiscrepancyMissing, SKUCode: "7896015976606",
			Name: "Mistura de Pão de Queijo - Pinduca - 500g", ChannelQuantity: 1, DetectedAt: detected,
		},
		Order: omsDomain.SalesOrder{
			ID: "so-1", Number: "25922", CurrencyCode: "EUR", OccurredAt: occurred,
			Billing: omsDomain.Address{FirstName: "Ana", LastName: "Silva"},
		},
	}}}

	w := do(newRouter(t, stubs{list: list}), http.MethodGet, "/api/v1/order-differences?page=2&per_page=10", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 2, list.gotPage)
	assert.Equal(t, 10, list.gotPer)
	var body struct {
		Data genorderdifferences.OrderDifferencePage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, int64(1), body.Data.Total)
	require.Len(t, body.Data.Differences, 1)
	assert.Equal(t, genorderdifferences.OrderDifference{
		Id: "d-1", SalesOrderId: "so-1", OrderNumber: "25922", CustomerName: "Ana Silva", OrderOccurredAt: occurred,
		Kind: genorderdifferences.Missing, SkuCode: "7896015976606", Name: "Mistura de Pão de Queijo - Pinduca - 500g",
		ChannelQuantity: 1, CurrencyCode: "EUR", DetectedAt: detected,
	}, body.Data.Differences[0])
}

func TestListOrderDifferences_defaultsPaging(t *testing.T) {
	list := &stubList{}

	w := do(newRouter(t, stubs{list: list}), http.MethodGet, "/api/v1/order-differences", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, list.gotPage)
	assert.Equal(t, 20, list.gotPer)
	assert.JSONEq(t, `{"data":{"differences":[],"total":0},"error":null}`, w.Body.String())
}

func TestListOrderDifferences_perPageAboveMax_is400(t *testing.T) {
	w := do(newRouter(t, stubs{}), http.MethodGet, "/api/v1/order-differences?per_page=500", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDismissOrderDifference(t *testing.T) {
	dismiss := &stubDismiss{}

	w := do(newRouter(t, stubs{dismiss: dismiss}), http.MethodPost, "/api/v1/order-differences/d-1/dismiss", "")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "d-1", dismiss.got)
}

func TestDismissOrderDifference_errors(t *testing.T) {
	for err, status := range map[error]int{
		fmt.Errorf("dismiss: %w", omsDomain.ErrLineDiscrepancyNotFound): http.StatusNotFound,
		fmt.Errorf("dismiss: %w", omsDomain.ErrLineDiscrepancyNotOpen):  http.StatusUnprocessableEntity,
		errors.New("pq: down"): http.StatusInternalServerError,
	} {
		w := do(newRouter(t, stubs{dismiss: &stubDismiss{err: err}}), http.MethodPost, "/api/v1/order-differences/d-1/dismiss", "")

		assert.Equal(t, status, w.Code, err.Error())
		assert.NotContains(t, w.Body.String(), "pq:")
	}
}

func TestAuditWooOrderDifferences_runsOneBatch(t *testing.T) {
	audit := &stubAudit{out: &appchannels.AuditWooOrderDifferencesResult{Checked: 25, NextCursor: "c-2"}}

	w := do(newRouter(t, stubs{audit: audit}), http.MethodPost, "/api/v1/integrations/connections/conn-1/audit-woo-order-differences", `{"cursor":"c-1"}`)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, appchannels.AuditWooOrderDifferencesInput{ConnectionID: "conn-1", Cursor: "c-1"}, audit.got)
	assert.JSONEq(t, `{"data":{"checked":25,"errors":[],"next_cursor":"c-2"},"error":null}`, w.Body.String())
}

func TestAuditWooOrderDifferences_errors(t *testing.T) {
	for err, status := range map[error]int{
		fmt.Errorf("audit: %w", integrationsDomain.ErrConnectionNotFound):   http.StatusNotFound,
		fmt.Errorf("audit: %w", integrationsDomain.ErrProviderNotSupported): http.StatusUnprocessableEntity,
		fmt.Errorf("audit: %w", appchannels.ErrInvalidBackfillCursor):       http.StatusUnprocessableEntity,
	} {
		w := do(newRouter(t, stubs{audit: &stubAudit{err: err}}), http.MethodPost, "/api/v1/integrations/connections/conn-1/audit-woo-order-differences", `{}`)

		assert.Equal(t, status, w.Code, err.Error())
	}
}

func TestAuditWooOrderDifferences_missingBody_is400(t *testing.T) {
	w := do(newRouter(t, stubs{}), http.MethodPost, "/api/v1/integrations/connections/conn-1/audit-woo-order-differences", "")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
