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
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo)

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
	uc := appwebhooks.NewProcessWebhookEventUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), webhooks.Envelope{EventID: "evt-1", TenantID: "tenant-1", Provider: "woocommerce"})

	require.Error(t, err)
}
