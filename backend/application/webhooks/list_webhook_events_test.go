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

type fakeWebhookEventRepoAdmin struct {
	events       []webhooks.WebhookEvent
	err          error
	gotProvider  string
	gotLimit     int
	captureCalls int
}

func (f *fakeWebhookEventRepoAdmin) Save(ctx context.Context, e *webhooks.WebhookEvent) error {
	panic("not used")
}
func (f *fakeWebhookEventRepoAdmin) DistinctTopics(ctx context.Context) ([]webhooks.TopicRef, error) {
	panic("not used")
}
func (f *fakeWebhookEventRepoAdmin) MarkDispatchResult(ctx context.Context, id string, status webhooks.WebhookEventStatus, dispatchError string) error {
	panic("not used")
}
func (f *fakeWebhookEventRepoAdmin) List(ctx context.Context, provider string, limit int) ([]webhooks.WebhookEvent, error) {
	f.captureCalls++
	f.gotProvider, f.gotLimit = provider, limit
	if f.err != nil {
		return nil, f.err
	}
	return f.events, nil
}

func TestListWebhookEventsUseCase_listsFromRepo(t *testing.T) {
	repo := &fakeWebhookEventRepoAdmin{events: []webhooks.WebhookEvent{
		{ID: "evt-1", Provider: "myparcel", Topic: "shipment_status_change", ReceivedAt: time.Now()},
	}}
	uc := appwebhooks.NewListWebhookEventsUseCase(newMockGormDB(t, 1), repo)

	events, err := uc.Execute(context.Background(), "", 50)

	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "evt-1", events[0].ID)
}

func TestListWebhookEventsUseCase_passesProviderFilterThrough(t *testing.T) {
	repo := &fakeWebhookEventRepoAdmin{}
	uc := appwebhooks.NewListWebhookEventsUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "myparcel", 50)

	require.NoError(t, err)
	assert.Equal(t, "myparcel", repo.gotProvider)
}

func TestListWebhookEventsUseCase_defaultsOutOfRangeLimit(t *testing.T) {
	repo := &fakeWebhookEventRepoAdmin{}
	uc := appwebhooks.NewListWebhookEventsUseCase(newMockGormDB(t, 2), repo)

	_, err := uc.Execute(context.Background(), "", 0)
	require.NoError(t, err)
	assert.Equal(t, 50, repo.gotLimit)

	_, err = uc.Execute(context.Background(), "", 500)
	require.NoError(t, err)
	assert.Equal(t, 50, repo.gotLimit)
}

func TestListWebhookEventsUseCase_propagatesRepoError(t *testing.T) {
	repo := &fakeWebhookEventRepoAdmin{err: errors.New("db down")}
	uc := appwebhooks.NewListWebhookEventsUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "", 50)

	require.Error(t, err)
}
