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

type stubListRules struct {
	out []mcDomain.AutomationRule
	err error
}

func (s *stubListRules) Execute(_ context.Context) ([]mcDomain.AutomationRule, error) {
	return s.out, s.err
}

type stubCreateRule struct {
	out *mcDomain.AutomationRule
	err error
	got appmc.CreateAutomationRuleInput
}

func (s *stubCreateRule) Execute(_ context.Context, in appmc.CreateAutomationRuleInput) (*mcDomain.AutomationRule, error) {
	s.got = in
	return s.out, s.err
}

type stubUpdateRule struct {
	out *mcDomain.AutomationRule
	err error
	got appmc.UpdateAutomationRuleInput
}

func (s *stubUpdateRule) Execute(_ context.Context, in appmc.UpdateAutomationRuleInput) (*mcDomain.AutomationRule, error) {
	s.got = in
	return s.out, s.err
}

type stubDeleteRule struct {
	err   error
	gotID string
}

func (s *stubDeleteRule) Execute(_ context.Context, id string) error {
	s.gotID = id
	return s.err
}

func newHandler(
	list *stubListTemplates, create *stubCreateTemplate, update *stubUpdateTemplate, del *stubDeleteTemplate,
	listRules *stubListRules, createRule *stubCreateRule, updateRule *stubUpdateRule, deleteRule *stubDeleteRule,
) *mchandler.Handler {
	if list == nil {
		list = &stubListTemplates{}
	}
	if create == nil {
		create = &stubCreateTemplate{}
	}
	if update == nil {
		update = &stubUpdateTemplate{}
	}
	if del == nil {
		del = &stubDeleteTemplate{}
	}
	if listRules == nil {
		listRules = &stubListRules{}
	}
	if createRule == nil {
		createRule = &stubCreateRule{}
	}
	if updateRule == nil {
		updateRule = &stubUpdateRule{}
	}
	if deleteRule == nil {
		deleteRule = &stubDeleteRule{}
	}
	return mchandler.NewHandler(list, create, update, del, listRules, createRule, updateRule, deleteRule)
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
	h := newHandler(list, nil, nil, nil, nil, nil, nil, nil)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/message-templates", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Order confirmed")
}

func TestCreateMessageTemplateHandler_success(t *testing.T) {
	create := &stubCreateTemplate{out: &mcDomain.MessageTemplate{ID: "template-1", Name: "New", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := newHandler(nil, create, nil, nil, nil, nil, nil, nil)
	body := `{"name":"New","category":"purchase","channels":["email"],"body":"hi"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/message-templates", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "New", create.got.Name)
	assert.Equal(t, mcDomain.CategoryPurchase, create.got.Category)
}

func TestCreateMessageTemplateHandler_invalidInput(t *testing.T) {
	create := &stubCreateTemplate{err: mcDomain.ErrTemplateNameRequired}
	h := newHandler(nil, create, nil, nil, nil, nil, nil, nil)
	body := `{"name":"","category":"purchase","channels":["email"],"body":"hi"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/message-templates", body)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateMessageTemplateHandler_success(t *testing.T) {
	update := &stubUpdateTemplate{out: &mcDomain.MessageTemplate{ID: "template-1", Name: "Updated", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := newHandler(nil, nil, update, nil, nil, nil, nil, nil)
	body := `{"name":"Updated","category":"purchase","channels":["sms"],"body":"hi","enabled":true}`
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/message-templates/template-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated")
}

func TestUpdateMessageTemplateHandler_notFound(t *testing.T) {
	update := &stubUpdateTemplate{err: mcDomain.ErrTemplateNotFound}
	h := newHandler(nil, nil, update, nil, nil, nil, nil, nil)
	body := `{"name":"Updated","category":"purchase","channels":["sms"],"body":"hi","enabled":true}`
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/message-templates/missing", body)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteMessageTemplateHandler_success(t *testing.T) {
	del := &stubDeleteTemplate{}
	h := newHandler(nil, nil, nil, del, nil, nil, nil, nil)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/message-templates/template-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "template-1", del.gotID)
}

func TestDeleteMessageTemplateHandler_notFound(t *testing.T) {
	del := &stubDeleteTemplate{err: mcDomain.ErrTemplateNotFound}
	h := newHandler(nil, nil, nil, del, nil, nil, nil, nil)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/message-templates/missing", "")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListAutomationRulesHandler_returnsTheList(t *testing.T) {
	list := &stubListRules{out: []mcDomain.AutomationRule{
		{ID: "rule-1", Name: "Collect point notice", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	h := newHandler(nil, nil, nil, nil, list, nil, nil, nil)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/automation-rules", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Collect point notice")
}

func TestCreateAutomationRuleHandler_success(t *testing.T) {
	create := &stubCreateRule{out: &mcDomain.AutomationRule{ID: "rule-1", Name: "New rule", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := newHandler(nil, nil, nil, nil, nil, create, nil, nil)
	body := `{"name":"New rule","trigger_subject":"shipping.shipment.fulfillment_signal","template_id":"template-1","conditions":[{"field":"status_code","operator":"in","value":[8,19]}]}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/automation-rules", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "New rule", create.got.Name)
	assert.Equal(t, "shipping.shipment.fulfillment_signal", create.got.TriggerSubject)
	require.Len(t, create.got.Conditions, 1)
	assert.Equal(t, mcDomain.OperatorIn, create.got.Conditions[0].Operator)
}

func TestCreateAutomationRuleHandler_invalidInput(t *testing.T) {
	create := &stubCreateRule{err: mcDomain.ErrInvalidTriggerSubject}
	h := newHandler(nil, nil, nil, nil, nil, create, nil, nil)
	body := `{"name":"New rule","trigger_subject":"bogus","template_id":"template-1"}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/automation-rules", body)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateAutomationRuleHandler_success(t *testing.T) {
	update := &stubUpdateRule{out: &mcDomain.AutomationRule{ID: "rule-1", Name: "Updated rule", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := newHandler(nil, nil, nil, nil, nil, nil, update, nil)
	body := `{"name":"Updated rule","trigger_subject":"shipping.shipment.fulfillment_signal","template_id":"template-1","enabled":true}`
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/automation-rules/rule-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated rule")
}

func TestUpdateAutomationRuleHandler_notFound(t *testing.T) {
	update := &stubUpdateRule{err: mcDomain.ErrRuleNotFound}
	h := newHandler(nil, nil, nil, nil, nil, nil, update, nil)
	body := `{"name":"Updated rule","trigger_subject":"shipping.shipment.fulfillment_signal","template_id":"template-1","enabled":true}`
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/automation-rules/missing", body)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteAutomationRuleHandler_success(t *testing.T) {
	del := &stubDeleteRule{}
	h := newHandler(nil, nil, nil, nil, nil, nil, nil, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/automation-rules/rule-1", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rule-1", del.gotID)
}

func TestDeleteAutomationRuleHandler_notFound(t *testing.T) {
	del := &stubDeleteRule{err: mcDomain.ErrRuleNotFound}
	h := newHandler(nil, nil, nil, nil, nil, nil, nil, del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/automation-rules/missing", "")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateAutomationRuleHandler_passesExtraRecipients(t *testing.T) {
	create := &stubCreateRule{out: &mcDomain.AutomationRule{ID: "rule-1", ExtraRecipients: []string{"ops@example.nl"}, CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := newHandler(nil, nil, nil, nil, nil, create, nil, nil)
	body := `{"name":"R","trigger_subject":"storefront.order.placed","template_id":"template-1","extra_recipients":["ops@example.nl"]}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/automation-rules", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"ops@example.nl"}, create.got.ExtraRecipients)
	assert.Contains(t, w.Body.String(), `"extra_recipients":["ops@example.nl"]`)
}

func TestUpdateAutomationRuleHandler_passesExtraRecipients(t *testing.T) {
	update := &stubUpdateRule{out: &mcDomain.AutomationRule{ID: "rule-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}}
	h := newHandler(nil, nil, nil, nil, nil, nil, update, nil)
	body := `{"name":"R","trigger_subject":"storefront.order.placed","template_id":"template-1","enabled":true,"extra_recipients":["a@example.nl","b@example.nl"]}`
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/automation-rules/rule-1", body)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"a@example.nl", "b@example.nl"}, update.got.ExtraRecipients)
	assert.Contains(t, w.Body.String(), `"extra_recipients":[]`, "a rule without extras lists none, not null")
}

func TestCreateAutomationRuleHandler_invalidExtraRecipientIsUnprocessable(t *testing.T) {
	for _, domainErr := range []error{mcDomain.ErrInvalidExtraRecipient, mcDomain.ErrTooManyExtraRecipients} {
		create := &stubCreateRule{err: domainErr}
		h := newHandler(nil, nil, nil, nil, nil, create, nil, nil)
		body := `{"name":"R","trigger_subject":"storefront.order.placed","template_id":"template-1","extra_recipients":["ops@example.nl"]}`
		w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/automation-rules", body)

		require.Equal(t, http.StatusUnprocessableEntity, w.Code, domainErr.Error())
	}
}

func TestCreateAutomationRuleHandler_malformedExtraRecipientIsUnprocessable(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, &stubCreateRule{}, nil, nil)
	body := `{"name":"R","trigger_subject":"storefront.order.placed","template_id":"template-1","extra_recipients":["not-an-email"]}`
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/automation-rules", body)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListMessageTriggersHandler_listsVariablesAndSampleData(t *testing.T) {
	h := newHandler(nil, nil, nil, nil, nil, nil, nil, nil)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/message-triggers", "")

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"subject":"storefront.order.placed"`)
	assert.Contains(t, body, `"audience":"customer"`)
	assert.Contains(t, body, `"has_order":true`)
	assert.Contains(t, body, `"name":"order_number"`)
	assert.Contains(t, body, `"type":"list"`)
	assert.Contains(t, body, `"subject":"inventory.stock.close_to_expiry_reported"`)
	assert.Contains(t, body, `"audience":"staff"`)
	assert.Contains(t, body, `"sample_data":{`)
}
