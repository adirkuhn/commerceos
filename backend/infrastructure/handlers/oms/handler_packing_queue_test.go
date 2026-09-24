package oms_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	omshandler "github.com/adirkuhn/commerceos/infrastructure/handlers/oms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubListPackingQueue struct {
	out        *appoms.ListPackingQueueResult
	err        error
	gotPage    int
	gotPerPage int
}

func (s *stubListPackingQueue) Execute(_ context.Context, page, perPage int) (*appoms.ListPackingQueueResult, error) {
	s.gotPage, s.gotPerPage = page, perPage
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.ListPackingQueueResult{}, nil
}

type lockStubs struct {
	queue        *stubListPackingQueue
	startPacking *stubStartOrResumePacking
	startPicking *stubStartOrResumePicking
	markPicked   *stubMarkOrderPicked
}

// newTestRouterAs injects claims the way middleware.JWT would, so the
// handler's role -> LockActor mapping is exercised.
func newTestRouterAs(role middleware.Role, s lockStubs) *gin.Engine {
	h := omshandler.NewHandler(&stubListSalesOrders{}, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{},
		&stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{},
		&stubAddSalesOrderLineItem{}, s.startPicking, &stubGetPickingSession{}, &stubTouchPickingSession{}, &stubResetPickingSession{},
		&stubScanPickItem{}, &stubSubstituteLineItem{}, &stubRefundLineItem{},
		&stubResolvePickScanCode{}, &stubDeletePickScan{}, &stubPausePicking{}, &stubCompletePicking{}, &stubBulkUpdateSalesOrderStatus{},
		&stubListSalesOrderActivities{},
		s.startPacking, &stubGetPackingSession{}, &stubTouchPackingSession{}, &stubResetPackingSession{},
		&stubScanPackItem{}, &stubDeletePackScan{}, &stubCompletePacking{}, &stubBuildPickList{}, s.queue, s.markPicked, &stubResolvePickScanCode{})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("jwt_claims", &middleware.Claims{UserID: "user-1", TenantID: "tenant-1", Role: role})
		c.Next()
	})
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func newLockStubs() lockStubs {
	return lockStubs{queue: &stubListPackingQueue{}, startPacking: &stubStartOrResumePacking{}, startPicking: &stubStartOrResumePicking{}, markPicked: &stubMarkOrderPicked{}}
}

func TestListPackingQueue_returnsEntriesWithLocks(t *testing.T) {
	since := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newLockStubs()
	s.queue.out = &appoms.ListPackingQueueResult{Total: 2, Entries: []omsDomain.PackingQueueEntry{
		{Order: omsDomain.SalesOrder{ID: "so-1", Number: "1001"}, Lock: &omsDomain.QueueLock{Activity: omsDomain.QueueLockPicking, UserID: "picker", Since: since}},
		{Order: omsDomain.SalesOrder{ID: "so-2", Number: "1002"}},
	}}

	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "GET", "/api/v1/packing-queue?page=2&per_page=50", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 2, s.queue.gotPage)
	assert.Equal(t, 50, s.queue.gotPerPage)
	var resp genoms.PackingQueueResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	assert.Equal(t, int64(2), resp.Data.Total)
	require.Len(t, resp.Data.Entries, 2)
	assert.Equal(t, "so-1", resp.Data.Entries[0].SalesOrder.Id)
	require.NotNil(t, resp.Data.Entries[0].Lock)
	assert.Equal(t, "picker", resp.Data.Entries[0].Lock.UserId)
	assert.Equal(t, genoms.Picking, resp.Data.Entries[0].Lock.Activity)
	assert.True(t, resp.Data.Entries[0].Lock.Since.Equal(since))
	assert.Contains(t, w.Body.String(), `"lock":null`, "an unlocked entry serializes lock as null")
}

func TestListPackingQueue_defaultsPaging(t *testing.T) {
	s := newLockStubs()
	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "GET", "/api/v1/packing-queue", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, s.queue.gotPage)
	assert.Equal(t, 20, s.queue.gotPerPage)
	assert.Contains(t, w.Body.String(), `"entries":[]`)
}

func TestStartPackingAndPicking_passTheCallersAdminRightsAsLockActor(t *testing.T) {
	cases := []struct {
		role    middleware.Role
		isAdmin bool
	}{
		{middleware.RoleSuperAdmin, true},
		{middleware.RoleCustomerOwner, true},
		{middleware.RoleCustomerAdmin, true},
		{middleware.RoleCustomerUser, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.role), func(t *testing.T) {
			s := newLockStubs()
			r := newTestRouterAs(tc.role, s)

			require.Equal(t, http.StatusOK, doRequest(r, "POST", "/api/v1/sales-orders/so-1/packing", nil).Code)
			require.Equal(t, http.StatusOK, doRequest(r, "POST", "/api/v1/sales-orders/so-1/picking", nil).Code)

			want := omsDomain.LockActor{UserID: "user-1", IsAdmin: tc.isAdmin}
			assert.Equal(t, want, s.startPacking.gotActor)
			assert.Equal(t, want, s.startPicking.gotActor)
		})
	}
}

func TestListSalesOrders_passesAwaitingPickingThrough(t *testing.T) {
	list := &stubListSalesOrders{out: &appoms.ListSalesOrdersResult{}}
	w := doRequest(newTestRouter(list, &stubGetSalesOrder{}, &stubCreateSalesOrder{}, &stubUpdateSalesOrder{}, &stubDeleteSalesOrder{}, &stubLinkLineItemSKU{}, &stubRetryStockDecrement{}, &stubForceAcceptStock{}),
		"GET", "/api/v1/sales-orders?awaiting_picking=true", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, list.gotInput.AwaitingPicking)
}

func TestMarkOrderPicked_posOrderIsUnprocessable(t *testing.T) {
	s := newLockStubs()
	s.markPicked.err = omsDomain.ErrPOSOrderNotFulfillable
	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "POST", "/api/v1/sales-orders/so-1/mark-picked", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestStartOrResumePacking_pendingPickingChangesIsConflict(t *testing.T) {
	s := newLockStubs()
	s.startPacking.err = omsDomain.ErrPickingSessionHasPendingChanges

	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "POST", "/api/v1/sales-orders/so-1/packing", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

type stubMarkOrderPicked struct {
	err        error
	gotOrderID string
	gotActor   omsDomain.LockActor
}

func (s *stubMarkOrderPicked) Execute(_ context.Context, salesOrderID string, actor omsDomain.LockActor) error {
	s.gotOrderID, s.gotActor = salesOrderID, actor
	return s.err
}

func TestMarkOrderPicked_success(t *testing.T) {
	s := newLockStubs()
	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "POST", "/api/v1/sales-orders/so-1/mark-picked", nil)

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "so-1", s.markPicked.gotOrderID)
	assert.Equal(t, omsDomain.LockActor{UserID: "user-1"}, s.markPicked.gotActor)
}

func TestMarkOrderPicked_errors(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{omsDomain.ErrSalesOrderNotFound, http.StatusNotFound},
		{omsDomain.ErrOrderAlreadyFulfilled, http.StatusUnprocessableEntity},
		{omsDomain.ErrOrderNotPickable, http.StatusUnprocessableEntity},
		{omsDomain.ErrOrderPackingInProgress, http.StatusConflict},
		{omsDomain.ErrOrderPickingInProgress, http.StatusConflict},
	}
	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			s := newLockStubs()
			s.markPicked.err = tc.err
			w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "POST", "/api/v1/sales-orders/so-1/mark-picked", nil)
			assert.Equal(t, tc.code, w.Code)
		})
	}
}
