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
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

type evalDeps struct {
	ruleRepo     *stubAutomationRuleRepo
	templateRepo *stubMessageTemplateRepo
	salesOrders  *fakeSalesOrderRepo
	customers    *fakeCustomerRepo
	conns        *fakeConnectionRepo
	evoClient    *fakeEvolutionAPIClient
	emailClient  *fakeEmailClient
	emailFactory *fakeEmailClientFactory
	staff        *stubCategoryRecipients
	lines        map[string][]omsDomain.SalesOrderLineItem
	shipments    map[string][]shippingDomain.Shipment
	profile      *settingsDomain.CompanyProfile
	settings     *settingsDomain.TenantSettings
}

type fakeLineItems struct{ bySO map[string][]omsDomain.SalesOrderLineItem }

func (f fakeLineItems) ListBySalesOrder(_ context.Context, id string) ([]omsDomain.SalesOrderLineItem, error) {
	return f.bySO[id], nil
}

type fakeShipments struct{ bySO map[string][]shippingDomain.Shipment }

func (f fakeShipments) FindBySalesOrderID(_ context.Context, id string) ([]shippingDomain.Shipment, error) {
	return f.bySO[id], nil
}

type noCarrierOverrides struct{}

func (noCarrierOverrides) FindByConnectionAndCarrier(context.Context, string, shippingDomain.Carrier) (*shippingDomain.ShippingCarrierOverride, error) {
	return nil, nil
}

type fixedProfile struct{ p *settingsDomain.CompanyProfile }

func (f fixedProfile) Get(context.Context) (*settingsDomain.CompanyProfile, error) { return f.p, nil }

type fixedSettings struct{ s *settingsDomain.TenantSettings }

func (f fixedSettings) Get(context.Context) (*settingsDomain.TenantSettings, error) { return f.s, nil }

type evalHarness struct {
	uc *appmc.EvaluateAutomationRulesUseCase
}

type stubCategoryRecipients struct {
	recipients  []string
	err         error
	gotCategory notificationsDomain.Category
}

func (s *stubCategoryRecipients) Execute(_ context.Context, c notificationsDomain.Category) ([]string, error) {
	s.gotCategory = c
	return s.recipients, s.err
}

func newEvalUseCase(t *testing.T, d evalDeps) *appmc.EvaluateAutomationRulesUseCase {
	t.Helper()
	if d.evoClient == nil {
		d.evoClient = &fakeEvolutionAPIClient{}
	}
	if d.emailClient == nil {
		d.emailClient = &fakeEmailClient{}
	}
	if d.staff == nil {
		d.staff = &stubCategoryRecipients{}
	}
	if d.emailFactory == nil {
		d.emailFactory = &fakeEmailClientFactory{client: &fakeEmailClient{}}
	}
	if d.salesOrders == nil {
		d.salesOrders = &fakeSalesOrderRepo{}
	}
	if d.customers == nil {
		d.customers = &fakeCustomerRepo{}
	}
	if d.profile == nil {
		d.profile = &settingsDomain.CompanyProfile{TradingName: "Cantinho"}
	}
	if d.settings == nil {
		d.settings = &settingsDomain.TenantSettings{}
	}
	orders := appmc.OrderSources{
		SalesOrders: d.salesOrders, Customers: d.customers, LineItems: fakeLineItems{bySO: d.lines},
		Shipments: fakeShipments{bySO: d.shipments}, CarrierOverrides: noCarrierOverrides{},
	}
	store := appmc.StoreSources{CompanyProfile: fixedProfile{d.profile}, Settings: fixedSettings{d.settings}}
	return appmc.NewEvaluateAutomationRulesUseCase(
		newMockGormDB(t, 1), d.ruleRepo, d.templateRepo, orders, store, d.conns,
		&fakeEvolutionAPIClientFactory{client: d.evoClient}, testEncryptionSecret,
		d.emailFactory, d.emailClient, d.staff,
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

// TestEvaluateAutomationRules_sendsEmailViaHTMLBody is a regression test:
// smtp.Client.Send only reads Message.HTMLBody, never Message.Body (see
// infrastructure/messaging/smtp.Client.Send) -- an earlier version of this
// use case set Body instead, which the real SMTP client would silently
// send as an empty message.
func TestEvaluateAutomationRules_sendsEmailViaHTMLBody(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true, Subject: "Update", Body: "Hi {{customer_name}}"}
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
	require.Len(t, email.sent, 1)
	assert.Equal(t, "Update", email.sent[0].Subject)
	assert.Equal(t, "Hi Jane", email.sent[0].HTMLBody)
	assert.Empty(t, email.sent[0].Body, "Body must stay empty -- smtp.Client only reads HTMLBody")
}

// TestEvaluateAutomationRules_prefersTenantEmailConnectionOverFallback is
// the regression test for the per-tenant Email integration (see
// domain/integrations.ProviderEmail): when the tenant has connected one,
// EvaluateAutomationRulesUseCase must send through it -- decrypted
// password, host/port/username/from all threaded through correctly -- not
// through the injected platform-level fallback client.
func TestEvaluateAutomationRules_prefersTenantEmailConnectionOverFallback(t *testing.T) {
	rule := mcDomain.AutomationRule{ID: "rule-1", TriggerSubject: "oms.sales_order.status_changed", TemplateID: "template-1", Enabled: true}
	template := mcDomain.MessageTemplate{ID: "template-1", Channels: []messaging.Channel{messaging.ChannelEmail}, Enabled: true, Subject: "Update", Body: "Hi {{customer_name}}"}
	tenantClient := &fakeEmailClient{}
	factory := &fakeEmailClientFactory{client: tenantClient}
	fallback := &fakeEmailClient{}
	uc := newEvalUseCase(t, evalDeps{
		ruleRepo:     &stubAutomationRuleRepo{bySubject: []mcDomain.AutomationRule{rule}},
		templateRepo: &stubMessageTemplateRepo{byID: map[string]mcDomain.MessageTemplate{"template-1": template}},
		salesOrders:  &fakeSalesOrderRepo{byID: map[string]omsDomain.SalesOrder{"order-1": {ID: "order-1", CustomerID: strPtr("cust-1")}}},
		customers:    &fakeCustomerRepo{byID: map[string]customersDomain.Customer{"cust-1": {ID: "cust-1", FirstName: "Jane", Email: "jane@example.com"}}},
		conns:        &fakeConnectionRepo{conns: []integrationsDomain.Connection{emailConnection(t)}},
		emailFactory: factory,
		emailClient:  fallback,
	})

	err := uc.Execute(context.Background(), "oms.sales_order.status_changed", "evt-1", map[string]any{"sales_order_id": "order-1"})

	require.NoError(t, err)
	assert.Empty(t, fallback.sent, "must not use the platform-level fallback when a tenant connection exists")
	require.Len(t, tenantClient.sent, 1)
	assert.Equal(t, "jane@example.com", tenantClient.sent[0].To)
	assert.Equal(t, "smtp.mystore.com", factory.gotHost)
	assert.Equal(t, "587", factory.gotPort)
	assert.Equal(t, "smtp-user", factory.gotUsername)
	assert.Equal(t, "smtp-pass", factory.gotPassword)
	assert.Equal(t, "orders@mystore.com", factory.gotFrom)
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
