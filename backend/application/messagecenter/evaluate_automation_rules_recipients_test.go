package messagecenter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func expiryRule(extras ...string) mcDomain.AutomationRule {
	return mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: mcDomain.TriggerStockCloseToExpiry, TemplateID: "template-1", Enabled: true, ExtraRecipients: extras}
}

func expiryTemplate() mcDomain.MessageTemplate {
	return mcDomain.MessageTemplate{
		ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true,
		Subject: "Stock expiring within {{window_days}} days",
		Body:    "<ul>{{#items}}<li>{{product_name}} ({{sku}}){{#sale_price}} sale {{sale_price}}{{/sale_price}}</li>{{/items}}</ul>",
	}
}

func expiryPayload() map[string]any {
	return map[string]any{
		"event_id": "evt-1", "tenant_id": "tenant-1", "window_days": float64(30),
		"items": []any{
			map[string]any{"product_name": "Pão de queijo & co", "sku": "PQ-1", "sale_price": "€ 3,00"},
			map[string]any{"product_name": "Guaraná", "sku": "G-1", "sale_price": ""},
		},
	}
}

func staffEvalUseCase(t *testing.T, rule mcDomain.AutomationRule, staff *stubCategoryRecipients, email *fakeEmailClient) *evalHarness {
	t.Helper()
	return &evalHarness{uc: newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": expiryTemplate()}},
		salesOrders:  &fakeSalesOrderRepo{},
		customers:    &fakeCustomerRepo{},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
		staff:        staff,
	})}
}

func sentTo(msgs []messaging.Message) []string {
	out := make([]string, len(msgs))
	for i, m := range msgs {
		out[i] = m.To
	}
	return out
}

func TestEvaluateAutomationRules_staffTrigger_sendsToCategoryRecipientsAndExtras(t *testing.T) {
	email := &fakeEmailClient{}
	staff := &stubCategoryRecipients{recipients: []string{"ops@example.nl"}}
	h := staffEvalUseCase(t, expiryRule("buyer@example.nl"), staff, email)

	err := h.uc.Execute(context.Background(), mcDomain.TriggerStockCloseToExpiry, "evt-1", expiryPayload())

	require.NoError(t, err)
	assert.Equal(t, notificationsDomain.CategoryStock, staff.gotCategory)
	assert.Equal(t, []string{"ops@example.nl", "buyer@example.nl"}, sentTo(email.sent))
	assert.Equal(t, "Stock expiring within 30 days", email.sent[0].Subject)
	assert.Equal(t, "<ul><li>Pão de queijo &amp; co (PQ-1) sale € 3,00</li><li>Guaraná (G-1)</li></ul>", email.sent[0].HTMLBody)
}

func TestEvaluateAutomationRules_staffEmailOff_stillSendsToExtras(t *testing.T) {
	email := &fakeEmailClient{}
	h := staffEvalUseCase(t, expiryRule("buyer@example.nl"), &stubCategoryRecipients{}, email)

	err := h.uc.Execute(context.Background(), mcDomain.TriggerStockCloseToExpiry, "evt-1", expiryPayload())

	require.NoError(t, err)
	assert.Equal(t, []string{"buyer@example.nl"}, sentTo(email.sent))
}

func TestEvaluateAutomationRules_staffEmailOffNoExtras_sendsNothing(t *testing.T) {
	email := &fakeEmailClient{}
	h := staffEvalUseCase(t, expiryRule(), &stubCategoryRecipients{}, email)

	err := h.uc.Execute(context.Background(), mcDomain.TriggerStockCloseToExpiry, "evt-1", expiryPayload())

	require.NoError(t, err)
	assert.Empty(t, email.sent)
}

func TestEvaluateAutomationRules_extraSameAsStaffRecipient_sentOnce(t *testing.T) {
	email := &fakeEmailClient{}
	h := staffEvalUseCase(t, expiryRule("OPS@example.nl"), &stubCategoryRecipients{recipients: []string{"ops@example.nl"}}, email)

	err := h.uc.Execute(context.Background(), mcDomain.TriggerStockCloseToExpiry, "evt-1", expiryPayload())

	require.NoError(t, err)
	assert.Equal(t, []string{"ops@example.nl"}, sentTo(email.sent))
}

func TestEvaluateAutomationRules_staffRecipientsError_propagates(t *testing.T) {
	boom := errors.New("prefs down")
	h := staffEvalUseCase(t, expiryRule("buyer@example.nl"), &stubCategoryRecipients{err: boom}, &fakeEmailClient{})

	err := h.uc.Execute(context.Background(), mcDomain.TriggerStockCloseToExpiry, "evt-1", expiryPayload())

	assert.ErrorIs(t, err, boom)
}

func TestEvaluateAutomationRules_customerTrigger_extrasGetACopy(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true,
		ExtraRecipients: []string{"shop@example.nl"}}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true, Subject: "Update", Body: "Order for {{customer_name}}"}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", FirstName: "Jane", Email: "jane@example.com"}}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	assert.Equal(t, []string{"jane@example.com", "shop@example.nl"}, sentTo(email.sent))
	assert.Equal(t, "Order for Jane", email.sent[1].HTMLBody, "the copy names the order's customer")
}

func TestEvaluateAutomationRules_noCustomer_stillSendsToExtras(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "pms.product.deleted", TemplateID: "template-1", Enabled: true,
		ExtraRecipients: []string{"shop@example.nl"}}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true, Body: "Deleted {{product_id}}"}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{},
		customers:    &fakeCustomerRepo{},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	err := uc.Execute(context.Background(), "pms.product.deleted", "evt-1", map[string]any{"product_id": "prod-1"})

	require.NoError(t, err)
	require.Len(t, email.sent, 1)
	assert.Equal(t, "shop@example.nl", email.sent[0].To)
	assert.Equal(t, "Deleted prod-1", email.sent[0].HTMLBody)
}

func TestEvaluateAutomationRules_extrasAreEmailOnly(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true,
		ExtraRecipients: []string{"shop@example.nl"}}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelWhatsApp}, Enabled: true, Body: "Hi"}
	evo := &fakeEvolutionAPIClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", PhoneE164: "31612345678"}}},
		conns:        &fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		evoClient:    evo,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	assert.Equal(t, "31612345678", evo.sentTo)
}

func TestEvaluateAutomationRules_whatsappBodyIsNotHTMLEscaped(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelWhatsApp}, Enabled: true, Body: "Hi {{customer_name}}"}
	evo := &fakeEvolutionAPIClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", FirstName: "Ana & Bia", PhoneE164: "31612345678"}}},
		conns:        &fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		evoClient:    evo,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	assert.Equal(t, "Hi Ana & Bia", evo.sentBody)
}
