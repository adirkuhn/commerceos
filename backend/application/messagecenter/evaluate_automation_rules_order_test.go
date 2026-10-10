package messagecenter_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/locale"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func emailTemplate(subject, body string) mcDomain.MessageTemplate {
	return mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true, Subject: subject, Body: body}
}

func orderFixture() omsDomain.SalesOrder {
	return omsDomain.SalesOrder{
		ID: "order-1", Number: "1001", Status: omsDomain.SalesOrderStatusShipped, CurrencyCode: "EUR", TotalMinorUnits: 1498,
		CustomerID: strPtr("cust-1"), OccurredAt: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC),
		Shipping: omsDomain.Address{Postcode: "1012 PA"},
	}
}

func TestEvaluateAutomationRules_templatesSeeOrderAndStoreDetails(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo: &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": emailTemplate(
			"Order {{order_number}}: {{order_status_label}}",
			"{{store_name}}: {{#items}}{{quantity}}x {{name}};{{/items}} total {{order_total}}, track {{tracking_url}}")}},
		salesOrders: &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": orderFixture()}},
		customers:   &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", FirstName: "Ana", Email: "ana@x.nl"}}},
		lines:       map[string][]omsDomain.SalesOrderLineItem{"order-1": {{Name: "Guaraná", Quantity: 3, PriceMinorUnits: 150, TotalMinorUnits: 450}}},
		shipments: map[string][]shippingDomain.Shipment{"order-1": {
			{ID: "new", Carrier: shippingDomain.CarrierPostNL, Barcode: "3S2", TrackingURL: "https://t/3S2"},
		}},
		profile:     &settingsDomain.CompanyProfile{TradingName: "Cantinho Brasileiro"},
		settings:    &settingsDomain.TenantSettings{Language: locale.Dutch},
		conns:       &fakeConnectionRepo{},
		emailClient: email,
	})

	require.NoError(t, uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"}))

	require.Len(t, email.sent, 1)
	assert.Equal(t, "ana@x.nl", email.sent[0].To)
	assert.Equal(t, "Order 1001: Verzonden", email.sent[0].Subject)
	assert.Equal(t, "Cantinho Brasileiro: 3x Guaraná; total € 14,98, track https://t/3S2", email.sent[0].HTMLBody)
}

func TestEvaluateAutomationRules_guestOrderGoesToTheBillingEmail(t *testing.T) {
	order := orderFixture()
	order.CustomerID = nil
	order.Billing = omsDomain.Address{FirstName: "Bia", Email: "bia@x.nl"}
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "storefront.order.placed", TemplateID: "template-1", Enabled: true}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": emailTemplate("Thanks", "Hi {{customer_name}}")}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": order}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	require.NoError(t, uc.Execute(context.Background(), "storefront.order.placed", "evt-1", map[string]any{"sales_order_id": "order-1"}))

	require.Len(t, email.sent, 1)
	assert.Equal(t, "bia@x.nl", email.sent[0].To)
	assert.Equal(t, "Hi Bia", email.sent[0].HTMLBody)
}

func TestEvaluateAutomationRules_conditionsCanUseOrderDetails(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true,
		Conditions: []mcDomain.Condition{{Field: "order_status", Operator: mcDomain.OperatorEq, Value: "completed"}}}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": emailTemplate("x", "x")}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": orderFixture()}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", Email: "ana@x.nl"}}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	require.NoError(t, uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"}))

	assert.Empty(t, email.sent, "the order is shipped, the rule wants completed")
}

func TestEvaluateAutomationRules_orderCreatedFromAnySourceFindsTheOrderAndItsData(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.created", TemplateID: "template-1", Enabled: true,
		Conditions: []mcDomain.Condition{{Field: "source", Operator: mcDomain.OperatorNeq, Value: "woocommerce"}}}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": emailTemplate("New order {{order_number}}", "via {{source}}")}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": orderFixture()}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", Email: "ana@x.nl"}}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.created", "evt-1", map[string]any{
		"event": "oms.sales_order.created", "entity_id": "order-1", "data": map[string]any{"source": "pos", "number": "1001"},
	})

	require.NoError(t, err)
	require.Len(t, email.sent, 1)
	assert.Equal(t, "New order 1001", email.sent[0].Subject)
	assert.Equal(t, "via pos", email.sent[0].HTMLBody)
}

func TestEvaluateAutomationRules_deletedOrderStillSendsToExtras(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "storefront.order.placed", TemplateID: "template-1", Enabled: true,
		ExtraRecipients: []string{"shop@x.nl"}}
	email := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": emailTemplate("x", "{{store_name}}")}},
		conns:        &fakeConnectionRepo{},
		emailClient:  email,
	})

	require.NoError(t, uc.Execute(context.Background(), "storefront.order.placed", "evt-1", map[string]any{"sales_order_id": "gone"}))

	require.Len(t, email.sent, 1)
	assert.Equal(t, "shop@x.nl", email.sent[0].To)
	assert.Equal(t, "Cantinho", email.sent[0].HTMLBody)
}
