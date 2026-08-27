package webhooks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

type fakeWebhookEventRepo struct {
	saved []webhooks.WebhookEvent
	err   error
}

func (f *fakeWebhookEventRepo) Save(ctx context.Context, e *webhooks.WebhookEvent) error {
	if f.err != nil {
		return f.err
	}
	e.ID = "webhook-event-" + e.EventID
	f.saved = append(f.saved, *e)
	return nil
}

type fakeForwardRuleRepo struct {
	rules []webhooks.ForwardRule
	err   error
}

func (f *fakeForwardRuleRepo) Create(ctx context.Context, r *webhooks.ForwardRule) error { return nil }
func (f *fakeForwardRuleRepo) List(ctx context.Context) ([]webhooks.ForwardRule, error)  { return f.rules, nil }
func (f *fakeForwardRuleRepo) Delete(ctx context.Context, id string) error               { return nil }
func (f *fakeForwardRuleRepo) Match(ctx context.Context, provider, topic string) ([]webhooks.ForwardRule, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []webhooks.ForwardRule
	for _, r := range f.rules {
		if r.Provider == provider && (r.Topic == topic || r.Topic == "") {
			out = append(out, r)
		}
	}
	return out, nil
}

type fakeForwardDeliveryRepo struct {
	created []webhooks.ForwardDelivery
	err     error
}

func (f *fakeForwardDeliveryRepo) Create(ctx context.Context, d *webhooks.ForwardDelivery) error {
	if f.err != nil {
		return f.err
	}
	d.ID = "delivery-" + d.RuleID
	f.created = append(f.created, *d)
	return nil
}
func (f *fakeForwardDeliveryRepo) ListDue(ctx context.Context, now time.Time, limit int) ([]webhooks.ForwardDelivery, error) {
	return nil, nil
}
func (f *fakeForwardDeliveryRepo) MarkDelivered(ctx context.Context, id string) error { return nil }
func (f *fakeForwardDeliveryRepo) MarkFailed(ctx context.Context, id string, errMsg string, nextAttemptAt time.Time, maxAttempts int) error {
	return nil
}
func (f *fakeForwardDeliveryRepo) ListRecent(ctx context.Context, limit int) ([]webhooks.ForwardDelivery, error) {
	return nil, nil
}

func TestProcessWebhookEventUseCase_persistsEnvelope(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil, nil, nil)

	err := uc.Execute(context.Background(), webhooks.Envelope{
		EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce",
		ReceivedAt: time.Now(), RawPayload: []byte(`{"id":1}`),
		Headers: map[string]string{"X-Wc-Webhook-Topic": "order.updated"},
	})

	require.NoError(t, err)
	require.Len(t, repo.saved, 1)
	assert.Equal(t, "tenant-1", repo.saved[0].TenantID)
	assert.Equal(t, "order.updated", repo.saved[0].Topic)
	assert.Equal(t, []byte(`{"id":1}`), repo.saved[0].Payload)
}

func TestProcessWebhookEventUseCase_propagatesRepoError(t *testing.T) {
	repo := &fakeWebhookEventRepo{err: errors.New("save failed")}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil, nil, nil)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce"})

	require.Error(t, err)
}

type fakeShipmentHandler struct {
	called   bool
	envelope webhooks.Envelope
	err      error
}

func (f *fakeShipmentHandler) Execute(_ context.Context, e webhooks.Envelope) error {
	f.called, f.envelope = true, e
	return f.err
}

func TestProcessWebhookEventUseCase_myParcelProvider_dispatchesToShipmentHandler(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	handler := &fakeShipmentHandler{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, handler, nil, nil)

	envelope := webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "myparcel", RawPayload: []byte(`{}`)}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.True(t, handler.called)
	assert.Equal(t, "evt-1", handler.envelope.EventID)
}

func TestProcessWebhookEventUseCase_otherProviders_neverDispatchToShipmentHandler(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	handler := &fakeShipmentHandler{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, handler, nil, nil)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce"})

	require.NoError(t, err)
	assert.False(t, handler.called, "the shipment handler is myparcel-only -- every other provider's event still just gets persisted")
}

func TestProcessWebhookEventUseCase_shipmentHandlerError_propagates(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	handler := &fakeShipmentHandler{err: errors.New("myparcel lookup failed")}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, handler, nil, nil)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "myparcel"})

	require.Error(t, err)
	require.Len(t, repo.saved, 1, "the event is still durably persisted even if the subsequent dispatch fails")
}

func TestProcessWebhookEventUseCase_enqueuesForwardDeliveryForMatchingRule(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	ruleRepo := &fakeForwardRuleRepo{rules: []webhooks.ForwardRule{
		{ID: "rule-1", Provider: "woocommerce", Topic: "order.updated", TargetURL: "https://example.com/hook", Active: true},
	}}
	deliveryRepo := &fakeForwardDeliveryRepo{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil, ruleRepo, deliveryRepo)

	err := uc.Execute(context.Background(), webhooks.Envelope{
		EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce", RawPayload: []byte(`{"id":1}`),
		Headers: map[string]string{"X-Wc-Webhook-Topic": "order.updated"},
	})

	require.NoError(t, err)
	require.Len(t, deliveryRepo.created, 1)
	d := deliveryRepo.created[0]
	assert.Equal(t, "rule-1", d.RuleID)
	assert.Equal(t, "webhook-event-evt-1", d.WebhookEventID)
	assert.Equal(t, "https://example.com/hook", d.TargetURL)
	assert.Equal(t, []byte(`{"id":1}`), d.Payload)
}

func TestProcessWebhookEventUseCase_noMatchingRule_enqueuesNothing(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	ruleRepo := &fakeForwardRuleRepo{rules: []webhooks.ForwardRule{
		{ID: "rule-1", Provider: "woocommerce", Topic: "product.updated", TargetURL: "https://example.com/hook", Active: true},
	}}
	deliveryRepo := &fakeForwardDeliveryRepo{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil, ruleRepo, deliveryRepo)

	err := uc.Execute(context.Background(), webhooks.Envelope{
		EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce",
		Headers: map[string]string{"X-Wc-Webhook-Topic": "order.updated"},
	})

	require.NoError(t, err)
	assert.Empty(t, deliveryRepo.created)
}

func TestProcessWebhookEventUseCase_providerWideRule_matchesEveryTopic(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	ruleRepo := &fakeForwardRuleRepo{rules: []webhooks.ForwardRule{
		{ID: "rule-1", Provider: "myparcel", Topic: "", TargetURL: "https://example.com/hook", Active: true},
	}}
	deliveryRepo := &fakeForwardDeliveryRepo{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil, ruleRepo, deliveryRepo)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "myparcel"})

	require.NoError(t, err)
	require.Len(t, deliveryRepo.created, 1)
}

func TestProcessWebhookEventUseCase_forwardDeliveryCreateError_propagates(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	ruleRepo := &fakeForwardRuleRepo{rules: []webhooks.ForwardRule{
		{ID: "rule-1", Provider: "woocommerce", Topic: "", TargetURL: "https://example.com/hook", Active: true},
	}}
	deliveryRepo := &fakeForwardDeliveryRepo{err: errors.New("insert failed")}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil, ruleRepo, deliveryRepo)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce"})

	require.Error(t, err)
}
