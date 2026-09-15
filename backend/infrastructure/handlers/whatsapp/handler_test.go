package whatsapp_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genwhatsapp "github.com/adirkuhn/commerceos/api/whatsapp"
	whatsappDomain "github.com/adirkuhn/commerceos/domain/whatsapp"
	whatsapphandler "github.com/adirkuhn/commerceos/infrastructure/handlers/whatsapp"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListConversations struct {
	out []whatsappDomain.ConversationSummary
	err error
}

func (s *stubListConversations) Execute(_ context.Context) ([]whatsappDomain.ConversationSummary, error) {
	return s.out, s.err
}

type stubListConversationMessages struct {
	out      []whatsappDomain.Message
	err      error
	gotPhone string
}

func (s *stubListConversationMessages) Execute(_ context.Context, phoneE164 string) ([]whatsappDomain.Message, error) {
	s.gotPhone = phoneE164
	return s.out, s.err
}

type stubDeleteConversation struct {
	err      error
	gotPhone string
}

func (s *stubDeleteConversation) Execute(_ context.Context, phoneE164 string) error {
	s.gotPhone = phoneE164
	return s.err
}

func newTestHandler(list *stubListConversations, messages *stubListConversationMessages, opts ...*stubDeleteConversation) *whatsapphandler.Handler {
	del := &stubDeleteConversation{}
	if len(opts) > 0 {
		del = opts[0]
	}
	return whatsapphandler.NewHandler(list, messages, del)
}

func newTestRouter(h *whatsapphandler.Handler) *gin.Engine {
	r := gin.New()
	genwhatsapp.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func doRequest(rg *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func TestListWhatsAppConversationsHandler_returnsTheList(t *testing.T) {
	custID := "cust-1"
	list := &stubListConversations{out: []whatsappDomain.ConversationSummary{
		{
			PhoneE164: "31612345678", CustomerID: &custID, CustomerName: "Jane Doe",
			LastDirection: whatsappDomain.MessageDirectionOutbound, LastBody: "Widget - in stock: 7", LastMessageAt: time.Now(),
		},
	}}
	h := newTestHandler(list, &stubListConversationMessages{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/whatsapp/conversations")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Jane Doe")
	assert.Contains(t, w.Body.String(), "31612345678")
}

func TestListWhatsAppConversationsHandler_infraError(t *testing.T) {
	list := &stubListConversations{err: errors.New("db down")}
	h := newTestHandler(list, &stubListConversationMessages{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/whatsapp/conversations")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestListWhatsAppConversationMessagesHandler_returnsTheThread(t *testing.T) {
	messages := &stubListConversationMessages{out: []whatsappDomain.Message{
		{ID: "msg-1", PhoneE164: "31612345678", Direction: whatsappDomain.MessageDirectionInbound, Body: "STOCK ABC123", CreatedAt: time.Now()},
	}}
	h := newTestHandler(&stubListConversations{}, messages)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/whatsapp/conversations/31612345678/messages")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "31612345678", messages.gotPhone)
	assert.Contains(t, w.Body.String(), "STOCK ABC123")
}

func TestDeleteWhatsAppConversationHandler_success(t *testing.T) {
	del := &stubDeleteConversation{}
	h := newTestHandler(&stubListConversations{}, &stubListConversationMessages{}, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/whatsapp/conversations/31612345678/messages")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "31612345678", del.gotPhone)
}

func TestDeleteWhatsAppConversationHandler_infraError(t *testing.T) {
	del := &stubDeleteConversation{err: errors.New("db down")}
	h := newTestHandler(&stubListConversations{}, &stubListConversationMessages{}, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/whatsapp/conversations/31612345678/messages")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
