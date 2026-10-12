package messagecenter_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

const statusChanged = "oms.sales_order.status_changed"

func historyDeps(template mcDomain.MessageTemplate, customerEmail string, history *recordedEmails, email *fakeEmailClient) evalDeps {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: statusChanged, TemplateID: "template-1", Enabled: true}
	return evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", FirstName: "Jane", Email: customerEmail}}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
		history:      history,
	}
}

func historyTemplate(enabled bool) mcDomain.MessageTemplate {
	return mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: enabled,
		Subject: "Order update", Body: "<p>Hi {{customer_name}}</p>"}
}

func TestEvaluateAutomationRules_historyRecordsASentEmail(t *testing.T) {
	history := &recordedEmails{}
	uc := newEvalUseCase(t, historyDeps(historyTemplate(true), "jane@example.com", history, &fakeEmailClient{}))

	require.NoError(t, uc.Execute(tenantCtx(), statusChanged, "evt-1", map[string]any{"sales_order_id": "order-1"}))

	require.Len(t, history.got, 1)
	assert.Equal(t, mcDomain.SentEmail{
		Source: mcDomain.SentEmailSourceAutomation, Recipient: "jane@example.com", Subject: "Order update", HTMLBody: "<p>Hi Jane</p>",
		Status: mcDomain.SentEmailSent, RuleID: "rule-1", TemplateID: "template-1", TriggerSubject: statusChanged,
	}, history.got[0])
}

func TestEvaluateAutomationRules_historyRecordsAFailedEmail(t *testing.T) {
	history := &recordedEmails{}
	uc := newEvalUseCase(t, historyDeps(historyTemplate(true), "jane@example.com", history, &fakeEmailClient{err: errors.New("smtp: 535 auth failed")}))

	err := uc.Execute(tenantCtx(), statusChanged, "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.Error(t, err, "still redelivered for a retry")
	require.Len(t, history.got, 1)
	assert.Equal(t, mcDomain.SentEmailFailed, history.got[0].Status)
	assert.Equal(t, "smtp: 535 auth failed", history.got[0].Reason)
}

func TestEvaluateAutomationRules_historyRecordsASkipWithNobodyToSendTo(t *testing.T) {
	history, email := &recordedEmails{}, &fakeEmailClient{}
	uc := newEvalUseCase(t, historyDeps(historyTemplate(true), "", history, email))

	require.NoError(t, uc.Execute(tenantCtx(), statusChanged, "evt-1", map[string]any{"sales_order_id": "order-1"}))

	assert.Empty(t, email.sent)
	require.Len(t, history.got, 1)
	assert.Equal(t, mcDomain.SentEmail{
		Source: mcDomain.SentEmailSourceAutomation, Subject: "Order update", Status: mcDomain.SentEmailSkipped, Reason: mcDomain.SkipNoRecipient,
		RuleID: "rule-1", TemplateID: "template-1", TriggerSubject: statusChanged,
	}, history.got[0])
}

func TestEvaluateAutomationRules_historyRecordsASkipForADisabledTemplate(t *testing.T) {
	history := &recordedEmails{}
	uc := newEvalUseCase(t, historyDeps(historyTemplate(false), "jane@example.com", history, &fakeEmailClient{}))

	require.NoError(t, uc.Execute(tenantCtx(), statusChanged, "evt-1", map[string]any{"sales_order_id": "order-1"}))

	require.Len(t, history.got, 1)
	assert.Equal(t, mcDomain.SentEmailSkipped, history.got[0].Status)
	assert.Equal(t, mcDomain.SkipTemplateDisabled, history.got[0].Reason)
}

func TestEvaluateAutomationRules_whatsAppIsNotInTheEmailHistory(t *testing.T) {
	history := &recordedEmails{}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelWhatsApp}, Enabled: true, TextBody: "Hi"}
	d := historyDeps(template, "jane@example.com", history, &fakeEmailClient{})
	d.customers = &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", PhoneE164: "31612345678"}}}
	uc := newEvalUseCase(t, d)

	require.NoError(t, uc.Execute(tenantCtx(), statusChanged, "evt-1", map[string]any{"sales_order_id": "order-1"}))

	assert.Empty(t, history.got)
}

