package oms_test

import (
	"net/http"
	"testing"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartOrResumePacking_success(t *testing.T) {
	var start *stubStartOrResumePacking
	w := doRequest(routerWithPacking(func(s *packingStubs) { start = s.start }),
		"POST", "/api/v1/sales-orders/so-1/packing", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "so-1", start.gotOrderID)
	assert.Contains(t, w.Body.String(), "pack-session-1")
}

func TestStartOrResumePacking_notPackableIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.start.err = omsDomain.ErrOrderNotPackable }),
		"POST", "/api/v1/sales-orders/so-1/packing", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestStartOrResumePacking_pickingInProgressIsConflict(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.start.err = omsDomain.ErrOrderPickingInProgress }),
		"POST", "/api/v1/sales-orders/so-1/packing", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestStartOrResumePacking_lockedIsConflict(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.start.err = omsDomain.ErrPackingSessionLocked }),
		"POST", "/api/v1/sales-orders/so-1/packing", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestTouchPackingSession_success(t *testing.T) {
	var touch *stubTouchPackingSession
	w := doRequest(routerWithPacking(func(s *packingStubs) { touch = s.touch }),
		"POST", "/api/v1/packing-sessions/session-1/touch", nil)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "session-1", touch.gotSessionID)
}

func TestTouchPackingSession_lockedIsConflict(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.touch.err = omsDomain.ErrPackingSessionLocked }),
		"POST", "/api/v1/packing-sessions/session-1/touch", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetPackingSession_success(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) {
		s.get.out = &appoms.PackingSessionDetail{Session: omsDomain.PackingSession{ID: "session-1"}}
	}), "GET", "/api/v1/packing-sessions/session-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "session-1")
}

func TestGetPackingSession_notFound(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.get.err = omsDomain.ErrPackingSessionNotFound }),
		"GET", "/api/v1/packing-sessions/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestResetPackingSession_success(t *testing.T) {
	var reset *stubResetPackingSession
	w := doRequest(routerWithPacking(func(s *packingStubs) { reset = s.reset }),
		"POST", "/api/v1/packing-sessions/session-1/reset", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", reset.gotID)
}

func TestResetPackingSession_alreadyCompletedIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.reset.err = omsDomain.ErrPackingSessionAlreadyCompleted }),
		"POST", "/api/v1/packing-sessions/session-1/reset", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestScanPackItem_success(t *testing.T) {
	var scan *stubScanPackItem
	body := map[string]any{"sku_id": "sku-1", "quantity": 1}
	w := doRequest(routerWithPacking(func(s *packingStubs) { scan = s.scan }),
		"POST", "/api/v1/packing-sessions/session-1/scan", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "session-1", scan.gotIn.SessionID)
	assert.Equal(t, "sku-1", scan.gotIn.SKUID)
	assert.Equal(t, int64(1), scan.gotIn.Quantity)
}

func TestScanPackItem_mismatchIsUnprocessable(t *testing.T) {
	body := map[string]any{"sku_id": "sku-1", "quantity": 1}
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.scan.err = omsDomain.ErrPackScanSKUMismatch }),
		"POST", "/api/v1/packing-sessions/session-1/scan", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeletePackScan_success(t *testing.T) {
	var deleteScan *stubDeletePackScan
	w := doRequest(routerWithPacking(func(s *packingStubs) { deleteScan = s.deleteScan }),
		"DELETE", "/api/v1/packing-sessions/session-1/scans/scan-1", nil)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "session-1", deleteScan.gotSessionID)
	assert.Equal(t, "scan-1", deleteScan.gotScanID)
}

func TestDeletePackScan_notFound(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.deleteScan.err = omsDomain.ErrPackScanNotFound }),
		"DELETE", "/api/v1/packing-sessions/session-1/scans/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestCompletePacking_success(t *testing.T) {
	var complete *stubCompletePacking
	w := doRequest(routerWithPacking(func(s *packingStubs) { complete = s.complete }),
		"POST", "/api/v1/packing-sessions/session-1/complete", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", complete.gotIn.SessionID)
	assert.Contains(t, w.Body.String(), "completed")
}

func TestCompletePacking_notActiveIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.complete.err = omsDomain.ErrPackingSessionNotActive }),
		"POST", "/api/v1/packing-sessions/session-1/complete", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
