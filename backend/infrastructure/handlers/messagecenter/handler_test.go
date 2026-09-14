package messagecenter_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genmc "github.com/adirkuhn/commerceos/api/messagecenter"
	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	mchandler "github.com/adirkuhn/commerceos/infrastructure/handlers/messagecenter"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListTemplates struct {
	out []mcDomain.MessageTemplate
	err error
}

func (s *stubListTemplates) Execute(_ context.Context) ([]mcDomain.MessageTemplate, error) {
	return s.out, s.err
}

type stubCreateTemplate struct {
	out *mcDomain.MessageTemplate
	err error
	got appmc.CreateMessageTemplateInput
}

func (s *stubCreateTemplate) Execute(_ context.Context, in appmc.CreateMessageTemplateInput) (*mcDomain.MessageTemplate, error) {
	s.got = in
	return s.out, s.err
}

type stubUpdateTemplate struct {
	out *mcDomain.MessageTemplate
	err error
}

func (s *stubUpdateTemplate) Execute(_ context.Context, _ appmc.UpdateMessageTemplateInput) (*mcDomain.MessageTemplate, error) {
	return s.out, s.err
}

type stubDeleteTemplate struct {
	err   error
	gotID string
}

func (s *stubDeleteTemplate) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

func newTestRouter(h *mchandler.Handler) *gin.Engine {
	r := gin.New()
	genmc.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestListMessageTemplatesHandler_returnsTheList(t *testing.T) {
	list := &stubListTemplates{out: []mcDomain.MessageTemplate{
		{ID: "template-1", Name: "Order confirmed", Category: mcDomain.CategoryPurchase, Body: "hi", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	h := mchandler.NewHandler(list, &stubCreateTemplate{}, &stubUpdateTemplate{}, &stubDeleteTemplate{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/message-templates", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Order confirmed")
}

func TestCreateMessageTemplateHandler_success(t *testing.T) {
	create := &stubCreateTemplate{out: &mcDomain.MessageTemplate{ID: "template-1", Name: "New", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := mchandler.NewHandler(&stubListTemplates{}, create, &stubUpdateTemplate{}, &stubDeleteTemplate{})
	body := `{"name":"New","category":"purchase","channels":["email"],"body":"hi"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/message-templates", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "New", create.got.Name)
	assert.Equal(t, mcDomain.CategoryPurchase, create.got.Category)
}

func TestCreateMessageTemplateHandler_invalidInput(t *testing.T) {
	create := &stubCreateTemplate{err: mcDomain.ErrTemplateNameRequired}
	h := mchandler.NewHandler(&stubListTemplates{}, create, &stubUpdateTemplate{}, &stubDeleteTemplate{})
	body := `{"name":"","category":"purchase","channels":["email"],"body":"hi"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/message-templates", body)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateMessageTemplateHandler_success(t *testing.T) {
	update := &stubUpdateTemplate{out: &mcDomain.MessageTemplate{ID: "template-1", Name: "Updated", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := mchandler.NewHandler(&stubListTemplates{}, &stubCreateTemplate{}, update, &stubDeleteTemplate{})
	body := `{"name":"Updated","category":"purchase","channels":["sms"],"body":"hi","enabled":true}`
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/message-templates/template-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated")
}

func TestUpdateMessageTemplateHandler_notFound(t *testing.T) {
	update := &stubUpdateTemplate{err: mcDomain.ErrTemplateNotFound}
	h := mchandler.NewHandler(&stubListTemplates{}, &stubCreateTemplate{}, update, &stubDeleteTemplate{})
	body := `{"name":"Updated","category":"purchase","channels":["sms"],"body":"hi","enabled":true}`
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/message-templates/missing", body)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteMessageTemplateHandler_success(t *testing.T) {
	del := &stubDeleteTemplate{}
	h := mchandler.NewHandler(&stubListTemplates{}, &stubCreateTemplate{}, &stubUpdateTemplate{}, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/message-templates/template-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "template-1", del.gotID)
}

func TestDeleteMessageTemplateHandler_notFound(t *testing.T) {
	del := &stubDeleteTemplate{err: mcDomain.ErrTemplateNotFound}
	h := mchandler.NewHandler(&stubListTemplates{}, &stubCreateTemplate{}, &stubUpdateTemplate{}, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/message-templates/missing", "")

	require.Equal(t, http.StatusNotFound, w.Code)
}
