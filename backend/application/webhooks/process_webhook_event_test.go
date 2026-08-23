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
	f.saved = append(f.saved, *e)
	return nil
}

func TestProcessWebhookEventUseCase_persistsEnvelope(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil)

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
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, nil)

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
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, handler)

	envelope := webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "myparcel", RawPayload: []byte(`{}`)}
	err := uc.Execute(context.Background(), envelope)

	require.NoError(t, err)
	assert.True(t, handler.called)
	assert.Equal(t, "evt-1", handler.envelope.EventID)
}

func TestProcessWebhookEventUseCase_otherProviders_neverDispatchToShipmentHandler(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	handler := &fakeShipmentHandler{}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, handler)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce"})

	require.NoError(t, err)
	assert.False(t, handler.called, "the shipment handler is myparcel-only -- every other provider's event still just gets persisted")
}

func TestProcessWebhookEventUseCase_shipmentHandlerError_propagates(t *testing.T) {
	repo := &fakeWebhookEventRepo{}
	handler := &fakeShipmentHandler{err: errors.New("myparcel lookup failed")}
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo, handler)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "myparcel"})

	require.Error(t, err)
	require.Len(t, repo.saved, 1, "the event is still durably persisted even if the subsequent dispatch fails")
}
