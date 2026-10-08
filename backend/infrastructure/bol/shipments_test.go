package bol_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/infrastructure/bol"
)

func writeToken(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 299})
}

// Pinned to Bol's Retailer API v10 spec: POST /retailer/shipments with a
// ShipmentRequest (orderItems + transport{transporterCode, trackAndTrace}),
// 202 + ProcessStatus.
func TestAdapter_CreateShipment_postsShipmentRequestAndReturnsProcessStatus(t *testing.T) {
	var gotMethod, gotAccept, gotContentType string
	var gotBody map[string]any
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeToken(w)
		case "/shipments":
			gotMethod = r.Method
			gotAccept = r.Header.Get("Accept")
			gotContentType = r.Header.Get("Content-Type")
			raw, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(raw, &gotBody))
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"processStatusId": "ps-77", "eventType": "CREATE_SHIPMENT", "status": "PENDING",
				"description": "Example", "createTimestamp": "2026-10-08T10:00:00+02:00",
			})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	status, err := bol.NewAdapter(client).CreateShipment(t.Context(), integrationsDomain.BolShipmentInput{
		OrderItems:        []integrationsDomain.BolShipmentItem{{OrderItemID: "2012345678", Quantity: 2}},
		ShipmentReference: "C000HLRF87",
		TransporterCode:   "TNT",
		TrackAndTrace:     "3SMYPA859699114",
	})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "application/vnd.retailer.v10+json", gotAccept)
	assert.Equal(t, "application/vnd.retailer.v10+json", gotContentType)
	assert.Equal(t, map[string]any{
		"orderItems":        []any{map[string]any{"orderItemId": "2012345678", "quantity": float64(2)}},
		"shipmentReference": "C000HLRF87",
		"transport":         map[string]any{"transporterCode": "TNT", "trackAndTrace": "3SMYPA859699114"},
	}, gotBody)
	assert.Equal(t, integrationsDomain.BolProcessStatus{ID: "ps-77", Status: "PENDING"}, status)
}

func TestAdapter_CreateShipment_validationRejection_isPermanent(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			writeToken(w)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"title":"Bad Request","violations":[{"name":"orderItems[0].orderItemId","reason":"Order item has already been shipped."}]}`))
	})

	_, err := bol.NewAdapter(client).CreateShipment(t.Context(), integrationsDomain.BolShipmentInput{
		OrderItems: []integrationsDomain.BolShipmentItem{{OrderItemID: "1", Quantity: 1}}, TransporterCode: "TNT",
	})

	require.ErrorIs(t, err, integrationsDomain.ErrBolRequestRejected)
	assert.Contains(t, err.Error(), "Order item has already been shipped.")
}

func TestAdapter_CreateShipment_serverOrRateLimitError_isRetryable(t *testing.T) {
	for _, code := range []int{http.StatusTooManyRequests, http.StatusUnauthorized, http.StatusInternalServerError} {
		_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/token" {
				writeToken(w)
				return
			}
			w.WriteHeader(code)
		})

		_, err := bol.NewAdapter(client).CreateShipment(t.Context(), integrationsDomain.BolShipmentInput{
			OrderItems: []integrationsDomain.BolShipmentItem{{OrderItemID: "1", Quantity: 1}}, TransporterCode: "TNT",
		})

		require.Error(t, err, code)
		assert.NotErrorIs(t, err, integrationsDomain.ErrBolRequestRejected, code)
	}
}

// Pinned to Bol's shared process-status v10 spec:
// GET /shared/process-status/{process-status-id}.
func TestAdapter_GetProcessStatus_readsStatusAndErrorMessage(t *testing.T) {
	var gotPath, gotAccept string
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			writeToken(w)
			return
		}
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"processStatusId": "ps-77", "eventType": "CREATE_SHIPMENT", "status": "FAILURE",
			"errorMessage": "Invalid track and trace code.", "description": "x", "createTimestamp": "2026-10-08T10:00:00+02:00",
		})
	})

	status, err := bol.NewAdapter(client).GetProcessStatus(t.Context(), "ps-77")

	require.NoError(t, err)
	assert.Equal(t, "/process-status/ps-77", gotPath)
	assert.Equal(t, "application/vnd.retailer.v10+json", gotAccept)
	assert.Equal(t, integrationsDomain.BolProcessStatus{ID: "ps-77", Status: "FAILURE", ErrorMessage: "Invalid track and trace code."}, status)
}
