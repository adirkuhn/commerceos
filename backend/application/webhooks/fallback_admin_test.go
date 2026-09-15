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

	out, err := uc.Execute(context.Background(), webhooks.FallbackStatusPendingRepublish, 1, 20)

	require.NoError(t, err)
	require.Len(t, out.Events, 1)
	assert.Equal(t, "fb-1", out.Events[0].ID)
}

func TestListWebhookFallbackEventsUseCase_translatesPageToLimitOffsetAndReturnsTotal(t *testing.T) {
	repo := &listingFallbackRepo{
		events: []webhooks.FallbackEvent{{ID: "fb-1"}},
		total:  47,
	}
	uc := appwebhooks.NewListWebhookFallbackEventsUseCase(newMockGormDB(t, 1), repo)

	out, err := uc.Execute(context.Background(), webhooks.FallbackStatusPendingRepublish, 3, 10)

	require.NoError(t, err)
	assert.Equal(t, 10, repo.gotLimit)
	assert.Equal(t, 20, repo.gotOffset, "page 3 at 10/page -> offset 20")
	assert.EqualValues(t, 47, out.Total)
}

func TestListWebhookFallbackEventsUseCase_defaultsAndClampsPagination(t *testing.T) {
	repo := &listingFallbackRepo{}
	uc := appwebhooks.NewListWebhookFallbackEventsUseCase(newMockGormDB(t, 2), repo)

	_, err := uc.Execute(context.Background(), webhooks.FallbackStatusPendingRepublish, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 20, repo.gotLimit, "perPage<=0 defaults to 20")
	assert.Equal(t, 0, repo.gotOffset, "page<=0 defaults to page 1 -> offset 0")

	_, err = uc.Execute(context.Background(), webhooks.FallbackStatusPendingRepublish, 1, 10000)
	require.NoError(t, err)
	assert.Equal(t, 200, repo.gotLimit, "perPage is clamped to the max")
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

func TestAbandonAllWebhookFallbackEventsUseCase_abandonsEveryMatchingRow(t *testing.T) {
	repo := &listingFallbackRepo{events: []webhooks.FallbackEvent{
		{ID: "fb-1", Status: webhooks.FallbackStatusPendingRepublish},
		{ID: "fb-2", Status: webhooks.FallbackStatusPendingRepublish},
		{ID: "fb-3", Status: webhooks.FallbackStatusAbandoned},
	}}
	uc := appwebhooks.NewAbandonAllWebhookFallbackEventsUseCase(newMockGormDB(t, 1), repo)

	count, err := uc.Execute(context.Background(), webhooks.FallbackStatusPendingRepublish)

	require.NoError(t, err)
	assert.EqualValues(t, 2, count)
	assert.Equal(t, webhooks.FallbackStatusAbandoned, repo.events[0].Status)
	assert.Equal(t, webhooks.FallbackStatusAbandoned, repo.events[1].Status)
}

func TestAbandonAllWebhookFallbackEventsUseCase_defaultsToPendingRepublish(t *testing.T) {
	repo := &listingFallbackRepo{events: []webhooks.FallbackEvent{
		{ID: "fb-1", Status: webhooks.FallbackStatusPendingRepublish},
	}}
	uc := appwebhooks.NewAbandonAllWebhookFallbackEventsUseCase(newMockGormDB(t, 1), repo)

	count, err := uc.Execute(context.Background(), "")

	require.NoError(t, err)
	assert.EqualValues(t, 1, count)
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
	total         int64
	getByID       bool
	notFound      bool
	statusUpdates map[string]webhooks.FallbackStatus
	gotLimit      int
	gotOffset     int
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

func (r *listingFallbackRepo) List(ctx context.Context, status webhooks.FallbackStatus, limit, offset int) ([]webhooks.FallbackEvent, int64, error) {
	r.gotLimit, r.gotOffset = limit, offset
	return r.events, r.total, nil
}

func (r *listingFallbackRepo) UpdateStatusForAll(ctx context.Context, fromStatus, toStatus webhooks.FallbackStatus) (int64, error) {
	var n int64
	for i := range r.events {
		if r.events[i].Status == fromStatus {
			r.events[i].Status = toStatus
			n++
		}
	}
	return n, nil
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
