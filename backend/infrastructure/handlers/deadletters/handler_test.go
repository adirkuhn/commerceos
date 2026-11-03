package deadletters_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gendeadletters "github.com/adirkuhn/commerceos/api/deadletters"
	"github.com/adirkuhn/commerceos/domain/deadletter"
	handler "github.com/adirkuhn/commerceos/infrastructure/handlers/deadletters"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

var errBoom = fmt.Errorf("boom")

type stubList struct {
	got  deadletter.Filter
	page deadletter.Page
	err  error
}

func (s *stubList) Execute(_ context.Context, f deadletter.Filter) (deadletter.Page, error) {
	s.got = f
	return s.page, s.err
}

type stubGet struct {
	out deadletter.DeadLetter
	err error
}

func (s *stubGet) Execute(_ context.Context, _ deadletter.Stream, _ uint64) (deadletter.DeadLetter, error) {
	return s.out, s.err
}

type stubAction struct {
	gotStream deadletter.Stream
	gotSeq    uint64
	called    bool
	err       error
}

func (s *stubAction) Execute(_ context.Context, stream deadletter.Stream, seq uint64) error {
	s.gotStream, s.gotSeq, s.called = stream, seq, true
	return s.err
}

type stubs struct {
	list    *stubList
	get     *stubGet
	retry   *stubAction
	discard *stubAction
}

func newStubs() stubs {
	return stubs{list: &stubList{}, get: &stubGet{}, retry: &stubAction{}, discard: &stubAction{}}
}

const secret = "s"

func newRouter(t *testing.T, s stubs) *gin.Engine {
	t.Helper()
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte(secret)))
	require.NoError(t, handler.Mount(g, handler.NewHandler(s.list, s.get, s.retry, s.discard)))
	return r
}

func token(t *testing.T, role middleware.Role) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{UserID: "u-1", TenantID: "t-1", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte(secret))
	require.NoError(t, err)
	return tok
}

// do sends the request and checks the response against the OpenAPI
// contract -- status, headers and body, error bodies included.
func do(t *testing.T, r *gin.Engine, method, path string, role middleware.Role) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if role != "" {
		req.Header.Set("Authorization", "Bearer "+token(t, role))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assertMatchesContract(t, req, w)
	return w
}

func assertMatchesContract(t *testing.T, req *http.Request, w *httptest.ResponseRecorder) {
	t.Helper()
	spec, err := gendeadletters.GetSpec()
	require.NoError(t, err)
	spec.Servers = nil
	router, err := gorillamux.NewRouter(spec)
	require.NoError(t, err)
	// Match the route without the /api/v1 prefix (servers stripped above).
	probe := req.Clone(context.Background())
	probe.URL.Path = req.URL.Path[len("/api/v1"):]
	route, pathParams, err := router.FindRoute(probe)
	if err == routers.ErrPathNotFound {
		t.Fatalf("no contract for %s %s", req.Method, req.URL.Path)
	}
	require.NoError(t, err)
	in := &openapi3filter.RequestValidationInput{Request: probe, PathParams: pathParams, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc}}
	out := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: in, Status: w.Code, Header: w.Header(),
		Body:    io.NopCloser(bytes.NewReader(w.Body.Bytes())),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	require.NoError(t, openapi3filter.ValidateResponse(context.Background(), out), "response breaks the contract: %d %s", w.Code, w.Body.String())
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body gendeadletters.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error.Code
}

func sample() deadletter.DeadLetter {
	at := time.Date(2026, 11, 3, 10, 0, 0, 0, time.UTC)
	return deadletter.DeadLetter{
		Stream: deadletter.StreamEvents, Sequence: 7, Subject: "oms.sales_order.status_changed",
		Consumer: "sales-order-status-changed-scheduler", EventID: "e-1", TenantID: "t-1",
		Error: "woo: 403", NumDelivered: 5, DeadLetteredAt: &at, StoredAt: at,
		Data: []byte(`{"event_id":"e-1"}`),
	}
}

func TestListDeadLetters(t *testing.T) {
	s := newStubs()
	old := sample()
	old.Sequence, old.Consumer, old.DeadLetteredAt, old.Error, old.NumDelivered = 8, "", nil, "", 0
	s.list.page = deadletter.Page{Items: []deadletter.DeadLetter{sample(), old}, NextCursor: 8, Total: 12}
	r := newRouter(t, s)

	w := do(t, r, http.MethodGet, "/api/v1/admin/dead-letters?stream=events&subject=oms.sales_order.status_changed&consumer=c1&tenant_id=t-1&cursor=3&limit=2", middleware.RoleSuperAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, deadletter.Filter{Stream: deadletter.StreamEvents, Subject: "oms.sales_order.status_changed",
		Consumer: "c1", TenantID: "t-1", After: 3, Limit: 2}, s.list.got)
	var resp gendeadletters.DeadLetterListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 2)
	assert.Equal(t, "sales-order-status-changed-scheduler", resp.Data.Items[0].Consumer)
	assert.True(t, resp.Data.Items[0].ConsumerKnown)
	assert.False(t, resp.Data.Items[1].ConsumerKnown)
	assert.Nil(t, resp.Data.Items[1].DeadLetteredAt)
	require.NotNil(t, resp.Data.NextCursor)
	assert.Equal(t, int64(8), *resp.Data.NextCursor)
	assert.Equal(t, int64(12), resp.Data.Total)
}

func TestListDeadLetters_lastPageHasNoCursor(t *testing.T) {
	s := newStubs()
	s.list.page = deadletter.Page{Items: []deadletter.DeadLetter{}}
	r := newRouter(t, s)

	w := do(t, r, http.MethodGet, "/api/v1/admin/dead-letters?stream=webhooks", middleware.RoleSuperAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp gendeadletters.DeadLetterListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Nil(t, resp.Data.NextCursor)
	assert.Equal(t, deadletter.StreamWebhooks, s.list.got.Stream)
}

func TestListDeadLetters_contractViolationsAre422(t *testing.T) {
	for _, q := range []string{"", "?stream=orders", "?stream=events&limit=0", "?stream=events&limit=500", "?stream=events&cursor=-1"} {
		t.Run(q, func(t *testing.T) {
			s := newStubs()
			w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/admin/dead-letters"+q, middleware.RoleSuperAdmin)

			assert.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
			assert.Equal(t, "VALIDATION_ERROR", errorCode(t, w))
		})
	}
}

func TestGetDeadLetter(t *testing.T) {
	s := newStubs()
	s.get.out = sample()
	r := newRouter(t, s)

	w := do(t, r, http.MethodGet, "/api/v1/admin/dead-letters/events/7", middleware.RoleSuperAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp gendeadletters.DeadLetterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, `{"event_id":"e-1"}`, resp.Data.Data)
	assert.Equal(t, "woo: 403", resp.Data.Error)
	assert.Equal(t, 5, resp.Data.NumDelivered)
}

func TestGetDeadLetter_notFound(t *testing.T) {
	s := newStubs()
	s.get.err = fmt.Errorf("getDeadLetter: %w", deadletter.ErrNotFound)

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/admin/dead-letters/events/7", middleware.RoleSuperAdmin)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "DEAD_LETTER_NOT_FOUND", errorCode(t, w))
}

func TestGetDeadLetter_badPathIs422(t *testing.T) {
	for _, path := range []string{"/api/v1/admin/dead-letters/orders/7", "/api/v1/admin/dead-letters/events/0", "/api/v1/admin/dead-letters/events/x"} {
		t.Run(path, func(t *testing.T) {
			w := do(t, newRouter(t, newStubs()), http.MethodGet, path, middleware.RoleSuperAdmin)

			assert.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
		})
	}
}

func TestRetryDeadLetter(t *testing.T) {
	s := newStubs()

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/admin/dead-letters/webhooks/9/retry", middleware.RoleSuperAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, deadletter.StreamWebhooks, s.retry.gotStream)
	assert.Equal(t, uint64(9), s.retry.gotSeq)
}

func TestRetryDeadLetter_publishFailureIs502(t *testing.T) {
	s := newStubs()
	s.retry.err = fmt.Errorf("retryDeadLetter: %w: %w", deadletter.ErrRetryFailed, fmt.Errorf("nats: timeout"))

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/admin/dead-letters/events/7/retry", middleware.RoleSuperAdmin)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Equal(t, "DEAD_LETTER_RETRY_FAILED", errorCode(t, w))
}

func TestRetryDeadLetter_notFound(t *testing.T) {
	s := newStubs()
	s.retry.err = deadletter.ErrNotFound

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/admin/dead-letters/events/7/retry", middleware.RoleSuperAdmin)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRetryDeadLetter_unexpectedErrorIs500(t *testing.T) {
	s := newStubs()
	s.retry.err = errBoom

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/admin/dead-letters/events/7/retry", middleware.RoleSuperAdmin)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "INTERNAL_ERROR", errorCode(t, w))
}

func TestDiscardDeadLetter(t *testing.T) {
	s := newStubs()

	w := do(t, newRouter(t, s), http.MethodDelete, "/api/v1/admin/dead-letters/events/7", middleware.RoleSuperAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, uint64(7), s.discard.gotSeq)
}

func TestDiscardDeadLetter_notFound(t *testing.T) {
	s := newStubs()
	s.discard.err = deadletter.ErrNotFound

	w := do(t, newRouter(t, s), http.MethodDelete, "/api/v1/admin/dead-letters/events/7", middleware.RoleSuperAdmin)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// Only super-admins reach these routes (COM-14): every other role is 403,
// no token is 401, and the use case never runs.
func TestRoutesAreSuperAdminOnly(t *testing.T) {
	routes := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/admin/dead-letters?stream=events"},
		{http.MethodGet, "/api/v1/admin/dead-letters/events/7"},
		{http.MethodPost, "/api/v1/admin/dead-letters/events/7/retry"},
		{http.MethodDelete, "/api/v1/admin/dead-letters/events/7"},
	}
	for _, rt := range routes {
		for _, role := range []middleware.Role{middleware.RoleCustomerOwner, middleware.RoleCustomerAdmin, middleware.RoleCustomerUser} {
			t.Run(rt.method+" "+rt.path+" as "+string(role), func(t *testing.T) {
				s := newStubs()
				w := do(t, newRouter(t, s), rt.method, rt.path, role)

				assert.Equal(t, http.StatusForbidden, w.Code)
				assert.False(t, s.retry.called || s.discard.called)
			})
		}
		t.Run(rt.method+" "+rt.path+" without token", func(t *testing.T) {
			w := do(t, newRouter(t, newStubs()), rt.method, rt.path, "")

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
