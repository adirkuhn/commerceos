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

func TestStartOrResumePacking_posOrderIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.start.err = omsDomain.ErrPOSOrderNotFulfillable }),
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

// Order changes happen in packing -- substitute, refund, resolve-scan.

func TestSubstituteLineItem_success(t *testing.T) {
	var substitute *stubSubstituteLineItem
	body := map[string]any{
		"line_item_id": "li-1",
		"lines": []map[string]any{
			{"sku_id": "sku-a", "substitute_quantity": 3, "original_quantity_covered": 1, "return_to_stock": false},
			{"sku_id": "sku-b", "substitute_quantity": 4, "original_quantity_covered": 1, "return_to_stock": true},
		},
	}
	w := doRequest(routerWithPacking(func(s *packingStubs) { substitute = s.substitute }),
		"POST", "/api/v1/packing-sessions/session-1/substitute-batch", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "session-1", substitute.gotIn.SessionID)
	assert.Equal(t, "li-1", substitute.gotIn.LineItemID)
	require.Len(t, substitute.gotIn.Lines, 2)
	assert.Equal(t, "sku-a", substitute.gotIn.Lines[0].SKUID)
	assert.Equal(t, int64(3), substitute.gotIn.Lines[0].SubstituteQuantity)
	assert.Equal(t, int64(1), substitute.gotIn.Lines[0].OriginalQtyCovered)
	assert.False(t, substitute.gotIn.Lines[0].ReturnToStock)
	assert.True(t, substitute.gotIn.Lines[1].ReturnToStock)
	assert.Contains(t, w.Body.String(), "sku-a")
	assert.Contains(t, w.Body.String(), "sku-b")
}

func TestSubstituteLineItem_coverageMismatchIsUnprocessable(t *testing.T) {
	body := map[string]any{
		"line_item_id": "li-1",
		"lines":        []map[string]any{{"sku_id": "sku-a", "substitute_quantity": 1, "original_quantity_covered": 1}},
	}
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.substitute.err = omsDomain.ErrSubstitutionCoverageMismatch }),
		"POST", "/api/v1/packing-sessions/session-1/substitute-batch", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubstituteLineItem_emptyLinesIsUnprocessable(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "lines": []map[string]any{}}
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.substitute.err = omsDomain.ErrSubstituteLinesRequired }),
		"POST", "/api/v1/packing-sessions/session-1/substitute-batch", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestSubstituteLineItem_notFound(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "lines": []map[string]any{{"sku_id": "sku-a", "substitute_quantity": 1, "original_quantity_covered": 1}}}
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.substitute.err = omsDomain.ErrPackingSessionNotFound }),
		"POST", "/api/v1/packing-sessions/missing/substitute-batch", body)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRefundLineItem_success(t *testing.T) {
	var refund *stubRefundLineItem
	body := map[string]any{"line_item_id": "li-1", "quantity": 1, "reason": "damaged"}
	w := doRequest(routerWithPacking(func(s *packingStubs) { refund = s.refund }),
		"POST", "/api/v1/packing-sessions/session-1/refund", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "session-1", refund.gotIn.SessionID)
	assert.Equal(t, "li-1", refund.gotIn.LineItemID)
	assert.Equal(t, int64(1), refund.gotIn.Quantity)
	assert.Equal(t, "damaged", refund.gotIn.Reason)
	assert.Contains(t, w.Body.String(), "refund")
}

func TestRefundLineItem_invalidQuantityIsUnprocessable(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "quantity": 1}
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.refund.err = omsDomain.ErrInvalidRefundQuantity }),
		"POST", "/api/v1/packing-sessions/session-1/refund", body)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRefundLineItem_permissionDeniedIsForbidden(t *testing.T) {
	body := map[string]any{"line_item_id": "li-1", "quantity": 1}
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.refund.err = omsDomain.ErrPermissionDenied }),
		"POST", "/api/v1/packing-sessions/session-1/refund", body)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestResolvePackScanCode_success(t *testing.T) {
	var resolve *stubResolvePickScanCode
	w := doRequest(routerWithPacking(func(s *packingStubs) { resolve = s.resolve }),
		"POST", "/api/v1/packing-sessions/session-1/resolve-scan", map[string]any{"code": "BARCODE-1"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "session-1", resolve.gotSessionID)
	assert.Equal(t, "BARCODE-1", resolve.gotCode)
}

func TestResolvePackScanCode_notFound(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.resolve.err = omsDomain.ErrPackingSessionNotFound }),
		"POST", "/api/v1/packing-sessions/missing/resolve-scan", map[string]any{"code": "BARCODE-1"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestScanPackItem_passesExtraItemFields(t *testing.T) {
	var scan *stubScanPackItem
	body := map[string]any{"sku_id": "sku-9", "quantity": 1, "action_type": "new_item", "price_minor_units": 399, "reason": "Extra jar"}
	w := doRequest(routerWithPacking(func(s *packingStubs) { scan = s.scan }), "POST", "/api/v1/packing-sessions/session-1/scan", body)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, omsDomain.PackScanActionNewItem, scan.gotIn.ActionType)
	require.NotNil(t, scan.gotIn.PriceMinorUnits)
	assert.Equal(t, int64(399), *scan.gotIn.PriceMinorUnits)
	assert.Equal(t, "Extra jar", scan.gotIn.Reason)
}

func TestScanPackItem_invalidActionIsUnprocessable(t *testing.T) {
	w := doRequest(routerWithPacking(func(s *packingStubs) { s.scan.err = omsDomain.ErrInvalidPackScanActionType }),
		"POST", "/api/v1/packing-sessions/session-1/scan", map[string]any{"sku_id": "sku-1", "quantity": 1})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestScanPackItem_returnsActionFields(t *testing.T) {
	price := int64(399)
	w := doRequest(routerWithPacking(func(s *packingStubs) {
		s.scan.out = &appoms.ScanPackItemResult{Scan: omsDomain.PackScan{ID: "p1", SKUID: "sku-9", ActionType: omsDomain.PackScanActionNewItem, Quantity: 1, PriceMinorUnits: &price, Reason: "Extra jar"}}
	}), "POST", "/api/v1/packing-sessions/session-1/scan", map[string]any{"sku_id": "sku-9", "quantity": 1, "action_type": "new_item", "price_minor_units": 399})

	require.Equal(t, http.StatusCreated, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"action_type":"new_item"`)
	assert.Contains(t, body, `"line_item_id":null`)
	assert.Contains(t, body, `"price_minor_units":399`)
	assert.Contains(t, body, `"replayed_at":null`)
}

func TestCompletePacking_passesResolutionsAndReturnsShortfalls(t *testing.T) {
	var complete *stubCompletePacking
	w := doRequest(routerWithPacking(func(s *packingStubs) {
		complete = s.complete
		s.complete.out = &appoms.CompletePackingResult{StockShortfalls: []appoms.StockShortfall{{ScanID: "p1", SKUID: "sku-9", Name: "Jar", Needed: 3, Short: 2}}}
	}), "POST", "/api/v1/packing-sessions/session-1/complete", map[string]any{"resolutions": map[string]string{"p0": "with_item"}})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, map[string]appoms.StockShortfallResolution{"p0": appoms.StockShortfallWithItem}, complete.gotIn.Resolutions)
	body := w.Body.String()
	assert.Contains(t, body, `"completed":false`)
	assert.Contains(t, body, `"scan_id":"p1"`)
	assert.NotContains(t, body, `"session"`)
}
