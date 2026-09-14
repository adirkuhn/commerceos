package webhooks_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

func TestListWebhookFallbackEventsUseCase_listsFromRepo(t *testing.T) {
	repo := &listingFallbackRepo{events: []webhooks.FallbackEvent{{ID: "fb-1", Provider: "woocommerce", Status: webhooks.FallbackStatusPendingRepublish}}}
	uc := appwebhooks.NewListWebhookFallbackEventsUseCase(newMockGormDB(t, 1), repo)

	out, err := uc.Execute(context.Background(), webhooks.FallbackStatusPendingRepublish)

	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "fb-1", out[0].ID)
}

func TestRetryWebhookFallbackEventUseCase_republishesAndMarksStatus(t *testing.T) {
	repo := &listingFallbackRepo{
		events: []webhooks.FallbackEvent{{ID: "fb-1", Provider: "woocommerce", Payload: []byte(`{"id":1}`), Status: webhooks.FallbackStatusPendingRepublish, ReceivedAt: time.Now()}},
	}
	pub := &fakePublisher{}
	uc := appwebhooks.NewRetryWebhookFallbackEventUseCase(newMockGormDB(t, 2), repo, pub)

	err := uc.Execute(context.Background(), "fb-1")

	require.NoError(t, err)
	assert.Equal(t, 1, pub.calls)
	require.Len(t, repo.statusUpdates, 1)
	assert.Equal(t, webhooks.FallbackStatusRepublished, repo.statusUpdates["fb-1"])
}

// TestRetryWebhookFallbackEventUseCase_publishesEnvelopeShapedLikeHotPath
// pins that retry republishes the same Envelope{EventID, TenantID,
// Provider, ReceivedAt, RawPayload, Headers} shape ReceiveWebhookUseCase's
// hot path publishes, not the bare raw payload -- a consumer expecting the
// envelope must not break just because a message came through retry
// instead of the hot path.
func TestRetryWebhookFallbackEventUseCase_publishesEnvelopeShapedLikeHotPath(t *testing.T) {
	receivedAt := time.Now().UTC().Truncate(time.Second)
	repo := &listingFallbackRepo{
		events: []webhooks.FallbackEvent{{
			ID: "fb-1", TenantID: "tenant-1", Provider: "woocommerce", EventID: "evt-1",
			Payload: []byte(`{"id":1}`), Headers: map[string]string{"X-Test": "1"},
			Status: webhooks.FallbackStatusPendingRepublish, ReceivedAt: receivedAt,
		}},
	}
	pub := &fakePublisher{}
	uc := appwebhooks.NewRetryWebhookFallbackEventUseCase(newMockGormDB(t, 2), repo, pub)

	err := uc.Execute(context.Background(), "fb-1")

	require.NoError(t, err)
	require.Equal(t, "webhooks.received.woocommerce", pub.lastSubject)

	var got webhooks.Envelope
	require.NoError(t, json.Unmarshal(pub.lastData, &got))
	assert.Equal(t, "evt-1", got.EventID)
	assert.Equal(t, "tenant-1", got.TenantID)
	assert.Equal(t, "woocommerce", got.Provider)
	assert.Equal(t, receivedAt, got.ReceivedAt)
	assert.Equal(t, []byte(`{"id":1}`), got.RawPayload)
	assert.Equal(t, "1", got.Headers["X-Test"])
}

func TestRetryWebhookFallbackEventUseCase_leavesStatusPendingOnPublishFailure(t *testing.T) {
	repo := &listingFallbackRepo{
		events: []webhooks.FallbackEvent{{ID: "fb-1", Provider: "woocommerce", Payload: []byte(`{}`), Status: webhooks.FallbackStatusPendingRepublish}},
	}
	pub := &fakePublisher{failCount: 99}
	uc := appwebhooks.NewRetryWebhookFallbackEventUseCase(newMockGormDB(t, 1), repo, pub)

	err := uc.Execute(context.Background(), "fb-1")

	require.Error(t, err)
	assert.Empty(t, repo.statusUpdates)
}

func TestAbandonWebhookFallbackEventUseCase_marksAbandoned(t *testing.T) {
	repo := &listingFallbackRepo{events: []webhooks.FallbackEvent{{ID: "fb-1"}}}
	uc := appwebhooks.NewAbandonWebhookFallbackEventUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "fb-1")

	require.NoError(t, err)
	assert.Equal(t, webhooks.FallbackStatusAbandoned, repo.statusUpdates["fb-1"])
}

func TestAbandonWebhookFallbackEventUseCase_propagatesNotFound(t *testing.T) {
	repo := &listingFallbackRepo{notFound: true}
	uc := appwebhooks.NewAbandonWebhookFallbackEventUseCase(newMockGormDB(t, 1), repo)

	err := uc.Execute(context.Background(), "missing")

	assert.ErrorIs(t, err, webhooks.ErrFallbackEventNotFound)
}

// listingFallbackRepo is a richer fake than fakeFallbackRepo (which only
// supports Save) -- needed for the admin use cases' Get/List/UpdateStatus.
type listingFallbackRepo struct {
	events        []webhooks.FallbackEvent
	getByID       bool
	notFound      bool
	statusUpdates map[string]webhooks.FallbackStatus
}

func (r *listingFallbackRepo) Save(ctx context.Context, e *webhooks.FallbackEvent) error { return nil }

func (r *listingFallbackRepo) Get(ctx context.Context, id string) (*webhooks.FallbackEvent, error) {
	if r.notFound {
		return nil, webhooks.ErrFallbackEventNotFound
	}
	for _, e := range r.events {
		if e.ID == id {
			ee := e
			return &ee, nil
		}
	}
	return nil, errors.New("not found in fake")
}

func (r *listingFallbackRepo) List(ctx context.Context, status webhooks.FallbackStatus) ([]webhooks.FallbackEvent, error) {
	return r.events, nil
}

func (r *listingFallbackRepo) UpdateStatus(ctx context.Context, id string, status webhooks.FallbackStatus) error {
	if r.notFound {
		return webhooks.ErrFallbackEventNotFound
	}
	if r.statusUpdates == nil {
		r.statusUpdates = map[string]webhooks.FallbackStatus{}
	}
	r.statusUpdates[id] = status
	return nil
}
