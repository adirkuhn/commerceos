package messagecenter_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

type evalDeps struct {
	ruleRepo     *stubAutomationRuleRepo
	templateRepo *stubMessageTemplateRepo
	salesOrders  *fakeSalesOrderRepo
	customers    *fakeCustomerRepo
	conns        *fakeConnectionRepo
	evoClient    *fakeEvolutionAPIClient
	emailClient  *fakeEmailClient
}

func newEvalUseCase(t *testing.T, d evalDeps) *appmc.EvaluateAutomationRulesUseCase {
	t.Helper()
	if d.evoClient == nil {
		d.evoClient = &fakeEvolutionAPIClient{}
	}
	if d.emailClient == nil {
		d.emailClient = &fakeEmailClient{}
	}
	return appmc.NewEvaluateAutomationRulesUseCase(
		newMockGormDB(t, 1), d.ruleRepo, d.templateRepo, d.salesOrders, d.customers, d.conns,
		&fakeEvolutionAPIClientFactory{client: d.evoClient}, testEncryptionSecret, d.emailClient,
	)
}

// TestEvaluateAutomationRules_userExample_collectPointShipment mirrors the
// user's own automation example: a shipment left at a collection point
// (MyParcel status 8/19) sends a WhatsApp message naming where it is.
func TestEvaluateAutomationRules_userExample_collectPointShipment(t *testing.T) {
	rule := mcDomain.AutomationRule{
		ID: "rule-1", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1", Enabled: true,
		Conditions: []mcDomain.Condition{{Field: "status_code", Operator: mcDomain.OperatorIn, Value: []any{float64(8), float64(19)}}},
	}
	template := mcDomain.MessageTemplate{
		ID: "template-1", Channels: []messaging.Channel{messaging.ChannelWhatsApp}, Enabled: true,
		Body: "Hi {{customer_name}}, your order is at a collection point (code {{status_code}}).",
	}
	evo := &fakeEvolutionAPIClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", FirstName: "Jane", PhoneE164: "31612345678"}}},
		conns:        &fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		evoClient:    evo,
	})

	err := uc.Execute(context.Background(), "shipping.shipment.fulfillment_signal", "evt-1", map[string]any{
		"sales_order_id": "order-1", "status_code": float64(8), "kind": "shipment_status",
	})

	require.NoError(t, err)
	assert.Equal(t, "31612345678", evo.sentTo)
	assert.Equal(t, "Hi Jane, your order is at a collection point (code 8).", evo.sentBody)
}

func TestEvaluateAutomationRules_conditionDoesNotMatch_sendsNothing(t *testing.T) {
	rule := mcDomain.AutomationRule{
		ID: "rule-1", TriggerSubject: "shipping.shipment.fulfillment_signal", TemplateID: "template-1", Enabled: true,
		Conditions: []mcDomain.Condition{{Field: "status_code", Operator: mcDomain.OperatorIn, Value: []any{float64(8), float64(19)}}},
	}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelWhatsApp}, Enabled: true, Body: "x"}
	evo := &fakeEvolutionAPIClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", PhoneE164: "31612345678"}}},
		conns:        &fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		evoClient:    evo,
	})

	err := uc.Execute(context.Background(), "shipping.shipment.fulfillment_signal", "evt-1", map[string]any{
		"sales_order_id": "order-1", "status_code": float64(7),
	})

	require.NoError(t, err)
	assert.Empty(t, evo.sentTo)
}

func TestEvaluateAutomationRules_noRulesForSubject_sendsNothing(t *testing.T) {
	evo := &fakeEvolutionAPIClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{},
		templateRepo: &stubMessageTemplateRepo{},
		salesOrders:  &fakeSalesOrderRepo{},
		customers:    &fakeCustomerRepo{},
		conns:        &fakeConnectionRepo{},
		evoClient:    evo,
	})

	err := uc.Execute(context.Background(), "shipping.shipment.fulfillment_signal", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	assert.Empty(t, evo.sentTo)
}

// TestEvaluateAutomationRules_noSalesOrderIDInPayload_sendsNothing covers a
// merchant/ops trigger subject (e.g. pms.product.deleted) with no customer
// recipient to resolve -- the rule can still match, but there's nothing to
// send to, so it's silently skipped rather than erroring.
func TestEvaluateAutomationRules_noSalesOrderIDInPayload_sendsNothing(t *testing.T) {
	rule := mcDomain.AutomationRule{
		ID: "rule-1", TriggerSubject: "pms.product.deleted", TemplateID: "template-1", Enabled: true,
	}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true, Body: "x"}
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
	assert.Empty(t, email.sent)
}

func TestEvaluateAutomationRules_disabledTemplate_sendsNothing(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: false, Body: "x"}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", Email: "jane@example.com"}}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	assert.Empty(t, email.sent)
}

func TestEvaluateAutomationRules_channelOverride_usesOverrideNotTemplateChannels(t *testing.T) {
	override := messaging.ChannelEmail
	rule := mcDomain.AutomationRule{
		ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true,
		ChannelOverride: &override,
	}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelWhatsApp}, Enabled: true, Subject: "Update", Body: "Hi {{customer_name}}"}
	email := &fakeEmailClient{}
	evo := &fakeEvolutionAPIClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", FirstName: "Jane", Email: "jane@example.com", PhoneE164: "31612345678"}}},
		conns:        &fakeConnectionRepo{conns: []integrationsDomain.Connection{whatsappConnection(t)}},
		emailClient:  email,
		evoClient:    evo,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	require.Len(t, email.sent, 1)
	assert.Equal(t, "jane@example.com", email.sent[0].To)
	assert.Empty(t, evo.sentTo)
}

func TestEvaluateAutomationRules_customerWithNoDestinationForChannel_sendsNothing(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true, Body: "x"}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", Email: ""}}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	assert.Empty(t, email.sent)
}
