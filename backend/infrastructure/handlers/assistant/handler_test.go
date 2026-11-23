package assistant_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	genassistant "github.com/adirkuhn/commerceos/api/assistant"
	appassistant "github.com/adirkuhn/commerceos/application/assistant"
	"github.com/adirkuhn/commerceos/domain/assistant"
	handler "github.com/adirkuhn/commerceos/infrastructure/handlers/assistant"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const (
	secret = "s"
	reqID  = "11111111-1111-4111-8111-111111111111"
)

var at = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

func sample(status assistant.Status) *assistant.Request {
	return &assistant.Request{ID: reqID, TenantID: "t-1", UserID: "u-1", UserRole: "customer_admin", Title: "Status of CO-12?",
		Status: status, Attempts: 0, NextAttemptAt: at, CreatedAt: at, UpdatedAt: at}
}

type stubCreate struct {
	gotText string
	gotCtx  context.Context
	err     error
}

func (s *stubCreate) Execute(ctx context.Context, text string) (*assistant.Request, error) {
	s.gotText, s.gotCtx = text, ctx
	if s.err != nil {
		return nil, s.err
	}
	return sample(assistant.StatusQueued), nil
}

type stubList struct {
	got assistant.ListFilter
	out *appassistant.RequestPage
	err error
}

func (s *stubList) Execute(_ context.Context, f assistant.ListFilter) (*appassistant.RequestPage, error) {
	s.got = f
	return s.out, s.err
}

type stubGet struct {
	out *appassistant.RequestDetail
	err error
}

func (s *stubGet) Execute(context.Context, string) (*appassistant.RequestDetail, error) { return s.out, s.err }

type stubReply struct {
	gotID, gotText string
	err            error
}

func (s *stubReply) Execute(_ context.Context, id, text string) (*assistant.Request, error) {
	s.gotID, s.gotText = id, text
	if s.err != nil {
		return nil, s.err
	}
	return sample(assistant.StatusQueued), nil
}

type stubCancel struct {
	gotID string
	err   error
}

func (s *stubCancel) Execute(_ context.Context, id string) (*assistant.Request, error) {
	s.gotID = id
	if s.err != nil {
		return nil, s.err
	}
	return sample(assistant.StatusCancelled), nil
}

type stubs struct {
	create *stubCreate
	list   *stubList
	get    *stubGet
	reply  *stubReply
	cancel *stubCancel
}

func newStubs() stubs {
	return stubs{create: &stubCreate{}, list: &stubList{out: &appassistant.RequestPage{}}, get: &stubGet{}, reply: &stubReply{}, cancel: &stubCancel{}}
}

func newRouter(t *testing.T, s stubs) *gin.Engine {
	t.Helper()
	r := gin.New()
	g := r.Group("/api/v1")
	g.Use(middleware.JWT([]byte(secret)))
	require.NoError(t, handler.Mount(g, handler.NewHandler(s.create, s.list, s.get, s.reply, s.cancel)))
	return r
}

func token(t *testing.T, role middleware.Role) string {
	t.Helper()
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.Claims{UserID: "u-1", TenantID: "t-1", Role: role,
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).SignedString([]byte(secret))
	require.NoError(t, err)
	return tok
}

// do sends the request as role and checks the response against the
// OpenAPI contract -- status, headers and body, error bodies included.
func do(t *testing.T, r *gin.Engine, method, path string, body any, role middleware.Role) *httptest.ResponseRecorder {
	t.Helper()
	var buf []byte
	if body != nil {
		var err error
		buf, err = json.Marshal(body)
		require.NoError(t, err)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(buf))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if role != "" {
		req.Header.Set("Authorization", "Bearer "+token(t, role))
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assertMatchesContract(t, method, path, buf, w)
	return w
}

func assertMatchesContract(t *testing.T, method, path string, body []byte, w *httptest.ResponseRecorder) {
	t.Helper()
	spec, err := genassistant.GetSpec()
	require.NoError(t, err)
	spec.Servers = nil
	router, err := gorillamux.NewRouter(spec)
	require.NoError(t, err)
	probe := httptest.NewRequest(method, path[len("/api/v1"):], bytes.NewReader(body))
	if body != nil {
		probe.Header.Set("Content-Type", "application/json")
	}
	route, pathParams, err := router.FindRoute(probe)
	if errors.Is(err, routers.ErrPathNotFound) {
		t.Fatalf("no contract for %s %s", method, path)
	}
	require.NoError(t, err)
	in := &openapi3filter.RequestValidationInput{Request: probe, PathParams: pathParams, Route: route,
		Options: &openapi3filter.Options{AuthenticationFunc: openapi3filter.NoopAuthenticationFunc, ExcludeRequestBody: true}}
	out := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: in, Status: w.Code, Header: w.Header(),
		Body:    io.NopCloser(bytes.NewReader(w.Body.Bytes())),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	require.NoError(t, openapi3filter.ValidateResponse(context.Background(), out), "response breaks the contract: %d %s", w.Code, w.Body.String())
}

func errorCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct{ Code string } `json:"error"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error.Code
}

func TestCreate_queuesAsTheCaller(t *testing.T) {
	s := newStubs()

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests", map[string]string{"text": "Status of CO-12?"}, middleware.RoleCustomerAdmin)

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, "Status of CO-12?", s.create.gotText)
	assert.Equal(t, "t-1", tenant.IDFromContext(s.create.gotCtx))
	assert.Equal(t, "u-1", tenant.UserIDFromContext(s.create.gotCtx))
	assert.Contains(t, w.Body.String(), `"status":"queued"`)
}

func TestCreate_contractViolationsAre422(t *testing.T) {
	for name, body := range map[string]any{
		"empty text":   map[string]string{"text": ""},
		"missing text": map[string]string{},
		"extra field":  map[string]any{"text": "hi", "admin": true},
	} {
		t.Run(name, func(t *testing.T) {
			s := newStubs()
			w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests", body, middleware.RoleCustomerOwner)

			assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
			assert.Equal(t, "VALIDATION_ERROR", errorCode(t, w))
			assert.Empty(t, s.create.gotText)
		})
	}
}

func TestCreate_domainValidationIs422WithItsCode(t *testing.T) {
	s := newStubs()
	s.create.err = assistant.ErrTextRequired

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests", map[string]string{"text": "   "}, middleware.RoleCustomerAdmin)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "ASSISTANT_TEXT_REQUIRED", errorCode(t, w))
}

func TestList_mapsFilterAndPage(t *testing.T) {
	s := newStubs()
	r := *sample(assistant.StatusQueued)
	r.Attempts, r.WaitReason, r.LastError = 2, assistant.WaitModelBusy, "429"
	s.list.out = &appassistant.RequestPage{Requests: []assistant.Request{r}, NextCursor: reqID}

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/assistant/requests?status=queued&limit=10&cursor="+reqID, nil, middleware.RoleSuperAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, assistant.ListFilter{Status: assistant.StatusQueued, Limit: 10, Cursor: reqID}, s.list.got)
	var body genassistant.AssistantRequestListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data.Requests, 1)
	assert.Equal(t, genassistant.ModelBusy, *body.Data.Requests[0].WaitReason)
	assert.Equal(t, "429", *body.Data.Requests[0].LastError)
	assert.Equal(t, reqID, *body.Data.NextCursor)
}

func TestList_unknownStatusIs422(t *testing.T) {
	w := do(t, newRouter(t, newStubs()), http.MethodGet, "/api/v1/assistant/requests?status=done", nil, middleware.RoleCustomerAdmin)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGet_returnsConversation(t *testing.T) {
	s := newStubs()
	s.get.out = &appassistant.RequestDetail{Request: *sample(assistant.StatusNeedsInput), Messages: []assistant.Message{
		{Seq: 1, Role: assistant.RoleUser, Content: "quote Ana 10 farofa", CreatedAt: at},
		{Seq: 2, Role: assistant.RoleAssistant, ToolCalls: []assistant.ToolCall{{ID: "c1", Name: "search", Arguments: `{"query":"Ana"}`}}, CreatedAt: at},
		{Seq: 3, Role: assistant.RoleTool, ToolCallID: "c1", ToolName: "search", CreatedAt: at},
		{Seq: 4, Role: assistant.RoleAssistant, Question: true, Content: "Which Ana?", CreatedAt: at},
		{Seq: 5, Role: assistant.RoleAssistant, Question: true, Notice: assistant.NoticeTooManySteps, CreatedAt: at},
	}}

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/assistant/requests/"+reqID, nil, middleware.RoleCustomerAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body genassistant.AssistantRequestDetailResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	msgs := body.Data.Messages
	require.Len(t, msgs, 5)
	assert.Equal(t, "search", (*msgs[1].ToolCalls)[0].Name)
	assert.Equal(t, "search", *msgs[2].ToolName)
	assert.True(t, msgs[3].Question)
	assert.Equal(t, genassistant.TooManySteps, *msgs[4].Notice)
	assert.Nil(t, msgs[0].Notice)
}

func TestGet_notFound(t *testing.T) {
	s := newStubs()
	s.get.err = assistant.ErrRequestNotFound

	w := do(t, newRouter(t, s), http.MethodGet, "/api/v1/assistant/requests/"+reqID, nil, middleware.RoleCustomerAdmin)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "ASSISTANT_REQUEST_NOT_FOUND", errorCode(t, w))
}

func TestReply(t *testing.T) {
	s := newStubs()

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests/"+reqID+"/reply", map[string]string{"text": "Ana Bakery"}, middleware.RoleCustomerAdmin)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, reqID, s.reply.gotID)
	assert.Equal(t, "Ana Bakery", s.reply.gotText)
}

func TestReply_wrongStateIs409(t *testing.T) {
	s := newStubs()
	s.reply.err = assistant.ErrInvalidTransition

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests/"+reqID+"/reply", map[string]string{"text": "x"}, middleware.RoleCustomerAdmin)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "ASSISTANT_INVALID_TRANSITION", errorCode(t, w))
}

func TestCancel(t *testing.T) {
	s := newStubs()

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests/"+reqID+"/cancel", nil, middleware.RoleCustomerOwner)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, reqID, s.cancel.gotID)
	assert.Contains(t, w.Body.String(), `"status":"cancelled"`)
}

func TestCancel_finishedIs409(t *testing.T) {
	s := newStubs()
	s.cancel.err = assistant.ErrNotCancellable

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests/"+reqID+"/cancel", nil, middleware.RoleCustomerAdmin)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestUnexpectedErrorIs500WithoutDetail(t *testing.T) {
	s := newStubs()
	s.cancel.err = errors.New("pq: connection refused at 10.0.0.5")

	w := do(t, newRouter(t, s), http.MethodPost, "/api/v1/assistant/requests/"+reqID+"/cancel", nil, middleware.RoleCustomerAdmin)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "10.0.0.5")
}

func TestAdminOnly(t *testing.T) {
	routes := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/v1/assistant/requests", nil},
		{http.MethodPost, "/api/v1/assistant/requests", map[string]string{"text": "hi"}},
		{http.MethodGet, "/api/v1/assistant/requests/" + reqID, nil},
		{http.MethodPost, "/api/v1/assistant/requests/" + reqID + "/reply", map[string]string{"text": "hi"}},
		{http.MethodPost, "/api/v1/assistant/requests/" + reqID + "/cancel", nil},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path+" as customer_user", func(t *testing.T) {
			s := newStubs()
			w := do(t, newRouter(t, s), rt.method, rt.path, rt.body, middleware.RoleCustomerUser)

			assert.Equal(t, http.StatusForbidden, w.Code)
			assert.Empty(t, s.create.gotText)
			assert.Empty(t, s.cancel.gotID)
		})
		t.Run(rt.method+" "+rt.path+" without token", func(t *testing.T) {
			w := do(t, newRouter(t, newStubs()), rt.method, rt.path, rt.body, "")

			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}
