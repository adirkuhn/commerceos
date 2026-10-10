package bolconfirmations_test

import (
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

	genbolconfirmations "github.com/adirkuhn/commerceos/api/bolconfirmations"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	bolconfirmationshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/bolconfirmations"
)

func init() { gin.SetMode(gin.TestMode) }

type stubList struct {
	out []channelsDomain.BolShipmentConfirmation
	err error
	got string
}

func (s *stubList) Execute(_ context.Context, salesOrderID string) ([]channelsDomain.BolShipmentConfirmation, error) {
	s.got = salesOrderID
	return s.out, s.err
}

// newTestRouter mounts the handler behind the same OpenAPI request
// validator cmd/api/main.go uses, so the contract itself is under test.
func newTestRouter(t *testing.T, list *stubList) *gin.Engine {
	t.Helper()
	spec, err := genbolconfirmations.GetSpec()
	require.NoError(t, err)
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(ginmiddleware.OapiRequestValidatorWithOptions(spec, &ginmiddleware.Options{SilenceServersWarning: true}))
	genbolconfirmations.RegisterHandlers(g, bolconfirmationshandler.NewHandler(list))
	return r
}

func get(t *testing.T, r *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w
}

func TestListBolShipmentConfirmations_returnsContractShape(t *testing.T) {
	created := time.Date(2026, 10, 7, 14, 11, 0, 0, time.UTC)
	resolved := created.Add(2 * time.Minute)
	list := &stubList{out: []channelsDomain.BolShipmentConfirmation{
		{
			ID: "conf-2", ShipmentID: "ship-1", ExternalOrderID: "C000HLRF87", TransporterCode: "TNT",
			Action: channelsDomain.BolActionUpdateTransport, TransportID: "t-1",
			TrackAndTrace: "3SMYPA859699114", ProcessStatusID: "ps-2", Status: channelsDomain.BolConfirmationFailure,
			FailureReason: channelsDomain.BolFailureRejected, ErrorMessage: "Invalid track and trace code.", CreatedAt: created, ResolvedAt: &resolved,
		},
		{
			ID: "conf-1", ShipmentID: "ship-1", ExternalOrderID: "C000HLRF87", TransporterCode: "TNT",
			Action:        channelsDomain.BolActionCreateShipment,
			TrackAndTrace: "3SMYPA859699114", ProcessStatusID: "ps-1", Status: channelsDomain.BolConfirmationPending,
			CreatedAt: created,
		},
	}}

	w := get(t, newTestRouter(t, list), "/api/v1/sales-orders/so-1/bol-shipment-confirmations")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "so-1", list.got)
	var body struct {
		Data  []genbolconfirmations.BolShipmentConfirmation `json:"data"`
		Error any                                           `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Nil(t, body.Error)
	require.Len(t, body.Data, 2)
	assert.Equal(t, genbolconfirmations.BolShipmentConfirmation{
		Id: "conf-2", ShipmentId: "ship-1", ExternalOrderId: "C000HLRF87", TransporterCode: "TNT",
		Action: genbolconfirmations.UpdateTransport, TransportId: "t-1",
		TrackAndTrace: "3SMYPA859699114", ProcessStatusId: "ps-2", Status: genbolconfirmations.Failure,
		FailureReason: "rejected", ErrorMessage: "Invalid track and trace code.", NeedsManualAction: true, CreatedAt: created, ResolvedAt: &resolved,
	}, body.Data[0])
	assert.Equal(t, genbolconfirmations.Pending, body.Data[1].Status)
	assert.Equal(t, genbolconfirmations.CreateShipment, body.Data[1].Action)
	assert.Empty(t, body.Data[1].TransportId)
	assert.False(t, body.Data[1].NeedsManualAction)
	assert.Nil(t, body.Data[1].ResolvedAt)
}

func TestListBolShipmentConfirmations_noneYet_returnsEmptyArray(t *testing.T) {
	w := get(t, newTestRouter(t, &stubList{out: []channelsDomain.BolShipmentConfirmation{}}), "/api/v1/sales-orders/so-1/bol-shipment-confirmations")

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"data":[],"error":null}`, w.Body.String())
}

func TestListBolShipmentConfirmations_unknownOrder_is404(t *testing.T) {
	list := &stubList{err: fmt.Errorf("listBolShipmentConfirmations: %w", omsDomain.ErrSalesOrderNotFound)}

	w := get(t, newTestRouter(t, list), "/api/v1/sales-orders/missing/bol-shipment-confirmations")

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"SALES_ORDER_NOT_FOUND"`)
}

func TestListBolShipmentConfirmations_unexpectedError_is500WithoutLeakingIt(t *testing.T) {
	list := &stubList{err: errors.New("pq: connection refused")}

	w := get(t, newTestRouter(t, list), "/api/v1/sales-orders/so-1/bol-shipment-confirmations")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "connection refused")
}
