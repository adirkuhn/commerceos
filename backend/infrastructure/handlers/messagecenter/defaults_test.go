package messagecenter_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genmc "github.com/adirkuhn/commerceos/api/messagecenter"
	"github.com/adirkuhn/commerceos/domain/locale"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	mchandler "github.com/adirkuhn/commerceos/infrastructure/handlers/messagecenter"
)

type stubListDefaults struct {
	out []mcDomain.DefaultTemplate
	err error
}

func (s *stubListDefaults) Execute(context.Context) ([]mcDomain.DefaultTemplate, error) {
	return s.out, s.err
}

type stubResetTemplate struct {
	out *mcDomain.MessageTemplate
	err error
	got string
}

func (s *stubResetTemplate) Execute(_ context.Context, id string) (*mcDomain.MessageTemplate, error) {
	s.got = id
	return s.out, s.err
}

func defaultsHandler(list *stubListDefaults, reset *stubResetTemplate) *mchandler.Handler {
	return newHandler(nil, nil, nil, nil, nil, nil, nil, nil).WithDefaults(list, reset)
}

func TestListMessageTemplateDefaultsHandler_returnsContractShape(t *testing.T) {
	defaults := mcDomain.DefaultTemplates(locale.English)
	w := doRequest(newTestRouter(defaultsHandler(&stubListDefaults{out: defaults}, nil)), http.MethodGet, "/api/v1/message-template-defaults", "")

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Data []genmc.MessageTemplateDefault `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, len(defaults))
	first := body.Data[0]
	assert.Equal(t, mcDomain.DefaultOrderConfirmation, first.Key)
	assert.Equal(t, genmc.TriggerSubject("storefront.order.placed"), first.TriggerSubject)
	assert.Equal(t, defaults[0].Template.Name, first.Name)
	assert.Equal(t, genmc.Category("purchase"), first.Category)
	assert.Equal(t, []genmc.Channel{"email", "whatsapp"}, first.Channels)
	assert.Equal(t, defaults[0].Template.Body, first.Body)
	assert.Equal(t, defaults[0].Template.TextBody, first.TextBody)
}

func TestListMessageTemplateDefaultsHandler_errorIs500(t *testing.T) {
	w := doRequest(newTestRouter(defaultsHandler(&stubListDefaults{err: errors.New("db down")}, nil)), http.MethodGet, "/api/v1/message-template-defaults", "")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "db down")
}

func TestCreateMessageTemplateHandler_passesTheDefaultKey(t *testing.T) {
	create := &stubCreateTemplate{out: &mcDomain.MessageTemplate{ID: "template-1", Name: "New", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := newHandler(nil, create, nil, nil, nil, nil, nil, nil)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/message-templates",
		`{"name":"New","category":"purchase","channels":["email"],"body":"hi","default_key":"order_confirmation"}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "order_confirmation", create.got.DefaultKey)
}

func TestCreateMessageTemplateHandler_unknownDefaultIs422(t *testing.T) {
	h := newHandler(nil, &stubCreateTemplate{err: mcDomain.ErrUnknownDefaultTemplate}, nil, nil, nil, nil, nil, nil)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/message-templates",
		`{"name":"New","category":"purchase","channels":["email"],"body":"hi","default_key":"nope"}`)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"MESSAGE_TEMPLATE_DEFAULT_UNKNOWN"`)
}

func TestResetMessageTemplateHandler_returnsTheResetTemplate(t *testing.T) {
	reset := &stubResetTemplate{out: &mcDomain.MessageTemplate{ID: "tpl-1", Name: "Mine", Body: "<p>default</p>", DefaultKey: "order_confirmation"}}

	w := doRequest(newTestRouter(defaultsHandler(nil, reset)), http.MethodPost, "/api/v1/message-templates/tpl-1/reset", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "tpl-1", reset.got)
	assert.Contains(t, w.Body.String(), `"default_key":"order_confirmation"`)
	assert.Contains(t, w.Body.String(), `"design":null`)
}

func TestResetMessageTemplateHandler_errors(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{fmt.Errorf("resetMessageTemplate: %w", mcDomain.ErrTemplateNotFound), http.StatusNotFound, "MESSAGE_TEMPLATE_NOT_FOUND"},
		{fmt.Errorf("resetMessageTemplate: %w", mcDomain.ErrTemplateHasNoDefault), http.StatusUnprocessableEntity, "MESSAGE_TEMPLATE_HAS_NO_DEFAULT"},
	} {
		w := doRequest(newTestRouter(defaultsHandler(nil, &stubResetTemplate{err: tc.err})), http.MethodPost, "/api/v1/message-templates/tpl-1/reset", "")
		assert.Equal(t, tc.status, w.Code)
		assert.Contains(t, w.Body.String(), `"code":"`+tc.code+`"`)
	}
}
