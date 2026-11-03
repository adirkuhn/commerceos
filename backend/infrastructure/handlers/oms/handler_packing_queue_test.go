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
	out        *appoms.ListQueueResult
	err        error
	gotPage    int
	gotPerPage int
}

func (s *stubListPackingQueue) Execute(_ context.Context, page, perPage int) (*appoms.ListQueueResult, error) {
	s.gotPage, s.gotPerPage = page, perPage
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appoms.ListQueueResult{}, nil
}

type stubReleaseLock struct {
	err          error
	gotSessionID string
	gotUserID    string
}

func (s *stubReleaseLock) Execute(_ context.Context, sessionID, userID string) error {
	s.gotSessionID, s.gotUserID = sessionID, userID
	return s.err
}

type stubTakeOverPacking struct {
	err      error
	gotActor omsDomain.LockActor
}

func (s *stubTakeOverPacking) Execute(_ context.Context, sessionID string, actor omsDomain.LockActor) (*omsDomain.PackingSession, error) {
	s.gotActor = actor
	if s.err != nil {
		return nil, s.err
	}
	return &omsDomain.PackingSession{ID: sessionID, Status: omsDomain.PackingSessionStatusInProgress, LockedBy: &actor.UserID}, nil
}

type stubTakeOverPicking struct {
	err      error
	gotActor omsDomain.LockActor
}

func (s *stubTakeOverPicking) Execute(_ context.Context, sessionID string, actor omsDomain.LockActor) (*omsDomain.PickingSession, error) {
	s.gotActor = actor
	if s.err != nil {
		return nil, s.err
	}
	return &omsDomain.PickingSession{ID: sessionID, Status: omsDomain.PickingSessionStatusInProgress, LockedBy: &actor.UserID}, nil
}

type lockStubs struct {
	releasePacking  *stubReleaseLock
	releasePicking  *stubReleaseLock
	takeOverPacking *stubTakeOverPacking
	takeOverPicking *stubTakeOverPicking
	pickingQueue    *stubListPackingQueue
	queue           *stubListPackingQueue
	startPacking    *stubStartOrResumePacking
	startPicking    *stubStartOrResumePicking
	markPicked      *stubMarkOrderPicked
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
		&stubScanPackItem{}, &stubDeletePackScan{}, &stubCompletePacking{}, &stubBuildPickList{}, s.queue, s.markPicked, &stubResolvePickScanCode{},
		omshandler.SessionLockUseCases{ReleasePacking: s.releasePacking, ReleasePicking: s.releasePicking, TakeOverPacking: s.takeOverPacking, TakeOverPicking: s.takeOverPicking}, s.pickingQueue, &stubListSalesOrderLineChanges{}).WithOrigins(&stubOriginLister{})
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("jwt_claims", &middleware.Claims{UserID: "user-1", TenantID: "tenant-1", Role: role})
		c.Next()
	})
	genoms.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func newLockStubs() lockStubs {
	return lockStubs{queue: &stubListPackingQueue{}, startPacking: &stubStartOrResumePacking{}, startPicking: &stubStartOrResumePicking{}, markPicked: &stubMarkOrderPicked{},
		releasePacking: &stubReleaseLock{}, releasePicking: &stubReleaseLock{}, takeOverPacking: &stubTakeOverPacking{}, takeOverPicking: &stubTakeOverPicking{}, pickingQueue: &stubListPackingQueue{}}
}

func TestListPackingQueue_returnsEntriesWithLocks(t *testing.T) {
	since := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	s := newLockStubs()
	s.queue.out = &appoms.ListQueueResult{Total: 2, Entries: []omsDomain.QueueEntry{
		{Order: omsDomain.SalesOrder{ID: "so-1", Number: "1001"}, Lock: &omsDomain.QueueLock{Activity: omsDomain.QueueLockPicking, UserID: "picker", Since: since}},
		{Order: omsDomain.SalesOrder{ID: "so-2", Number: "1002"}},
	}}

	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "GET", "/api/v1/packing-queue?page=2&per_page=50", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 2, s.queue.gotPage)
	assert.Equal(t, 50, s.queue.gotPerPage)
	var resp genoms.OrderQueueResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotNil(t, resp.Data)
	assert.Equal(t, int64(2), resp.Data.Total)
	require.Len(t, resp.Data.Entries, 2)
	assert.Equal(t, "so-1", resp.Data.Entries[0].SalesOrder.Id)
	require.NotNil(t, resp.Data.Entries[0].Lock)
	assert.Equal(t, "picker", resp.Data.Entries[0].Lock.UserId)
	assert.Equal(t, genoms.QueueLockActivityPicking, resp.Data.Entries[0].Lock.Activity)
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

func TestReleaseSessionLocks_useTheCaller(t *testing.T) {
	s := newLockStubs()
	r := newTestRouterAs(middleware.RoleCustomerUser, s)

	require.Equal(t, http.StatusNoContent, doRequest(r, "POST", "/api/v1/packing-sessions/pack-1/release", nil).Code)
	require.Equal(t, http.StatusNoContent, doRequest(r, "POST", "/api/v1/picking-sessions/pick-1/release", nil).Code)

	assert.Equal(t, "pack-1", s.releasePacking.gotSessionID)
	assert.Equal(t, "user-1", s.releasePacking.gotUserID)
	assert.Equal(t, "pick-1", s.releasePicking.gotSessionID)
	assert.Equal(t, "user-1", s.releasePicking.gotUserID)
}

func TestReleaseSessionLock_notFound(t *testing.T) {
	s := newLockStubs()
	s.releasePacking.err = omsDomain.ErrPackingSessionNotFound

	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "POST", "/api/v1/packing-sessions/missing/release", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTakeOverSessions_passTheCallersAdminRights(t *testing.T) {
	s := newLockStubs()
	r := newTestRouterAs(middleware.RoleCustomerOwner, s)

	w := doRequest(r, "POST", "/api/v1/packing-sessions/pack-1/take-over", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"locked_by":"user-1"`)
	require.Equal(t, http.StatusOK, doRequest(r, "POST", "/api/v1/picking-sessions/pick-1/take-over", nil).Code)

	want := omsDomain.LockActor{UserID: "user-1", IsAdmin: true}
	assert.Equal(t, want, s.takeOverPacking.gotActor)
	assert.Equal(t, want, s.takeOverPicking.gotActor)
}

func TestTakeOverSession_lockedIsConflict(t *testing.T) {
	s := newLockStubs()
	s.takeOverPacking.err = omsDomain.ErrPackingSessionLocked
	s.takeOverPicking.err = omsDomain.ErrPickingSessionLocked
	r := newTestRouterAs(middleware.RoleCustomerUser, s)

	assert.Equal(t, http.StatusConflict, doRequest(r, "POST", "/api/v1/packing-sessions/pack-1/take-over", nil).Code)
	assert.Equal(t, http.StatusConflict, doRequest(r, "POST", "/api/v1/picking-sessions/pick-1/take-over", nil).Code)
}

func TestListPickingQueue_returnsEntriesWithLockAndInPacking(t *testing.T) {
	s := newLockStubs()
	s.pickingQueue.out = &appoms.ListQueueResult{Total: 1, Entries: []omsDomain.QueueEntry{
		{Order: omsDomain.SalesOrder{ID: "so-1"}, Lock: &omsDomain.QueueLock{Activity: omsDomain.QueueLockPacking, UserID: "packer", Since: time.Now()}, InPacking: true},
	}}

	w := doRequest(newTestRouterAs(middleware.RoleCustomerUser, s), "GET", "/api/v1/picking-queue?per_page=50", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 50, s.pickingQueue.gotPerPage)
	body := w.Body.String()
	assert.Contains(t, body, `"in_packing":true`)
	assert.Contains(t, body, `"activity":"packing"`)
}
