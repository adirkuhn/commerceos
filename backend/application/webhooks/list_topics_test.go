package webhooks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appintegrations "github.com/adirkuhn/commerceos/application/integrations"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	"github.com/adirkuhn/commerceos/domain/webhooks"
)

type fakeTopicsEventRepo struct {
	refs []webhooks.TopicRef
	err  error
}

func (f *fakeTopicsEventRepo) Save(ctx context.Context, e *webhooks.WebhookEvent) error {
	panic("not used")
}
func (f *fakeTopicsEventRepo) DistinctTopics(ctx context.Context) ([]webhooks.TopicRef, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.refs, nil
}

func (f *fakeTopicsEventRepo) List(ctx context.Context, provider string, limit int) ([]webhooks.WebhookEvent, error) {
	panic("not used")
}

func (f *fakeTopicsEventRepo) MarkDispatchResult(ctx context.Context, id string, status webhooks.WebhookEventStatus, dispatchError string) error {
	panic("not used")
}

type fakeSubscriptionsLister struct {
	result *appintegrations.ListAllProviderWebhooksResult
	err    error
}

func (f *fakeSubscriptionsLister) Execute(ctx context.Context) (*appintegrations.ListAllProviderWebhooksResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

// A provider outside integrationsDomain.KnownWebhookTopicsByProvider's
// baseline catalog -- isolates these assertions from that seeding so they
// only exercise the received/subscribed logic itself. Baseline seeding
// gets its own test below.
const customProvider = "customprovider"

func TestListAvailableWebhookTopicsUseCase_allEvents_isCatalogPlusReceivedTraffic(t *testing.T) {
	events := &fakeTopicsEventRepo{refs: []webhooks.TopicRef{
		{Provider: customProvider, Topic: "thing.updated"},
		{Provider: customProvider, Topic: "thing.created"},
	}}
	subs := &fakeSubscriptionsLister{result: &appintegrations.ListAllProviderWebhooksResult{}}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"thing.created", "thing.updated"}, out.AllEvents[customProvider])
}

func TestListAvailableWebhookTopicsUseCase_activeWebhooks_isCurrentlyActiveSubscriptionsOnly(t *testing.T) {
	events := &fakeTopicsEventRepo{}
	subs := &fakeSubscriptionsLister{result: &appintegrations.ListAllProviderWebhooksResult{
		Webhooks: []integrationsDomain.ProviderWebhook{
			{Provider: customProvider, Topic: "active.one", Status: "active"},
			{Provider: customProvider, Topic: "disabled.one", Status: "disabled"},
			{Provider: integrationsDomain.ProviderMyParcel, Topic: string(integrationsDomain.MyParcelHookShipmentStatusChange), Status: "active"},
		},
	}}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"active.one"}, out.ActiveWebhooks[customProvider])
	assert.Contains(t, out.ActiveWebhooks["myparcel"], string(integrationsDomain.MyParcelHookShipmentStatusChange))
}

func TestListAvailableWebhookTopicsUseCase_subscriptionsFailure_stillReturnsAllEvents(t *testing.T) {
	events := &fakeTopicsEventRepo{refs: []webhooks.TopicRef{{Provider: customProvider, Topic: "thing.updated"}}}
	subs := &fakeSubscriptionsLister{err: errors.New("provider api down")}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"thing.updated"}, out.AllEvents[customProvider])
	assert.Empty(t, out.ActiveWebhooks[customProvider])
}

func TestListAvailableWebhookTopicsUseCase_eventRepoFailure_propagatesError(t *testing.T) {
	events := &fakeTopicsEventRepo{err: errors.New("db down")}
	subs := &fakeSubscriptionsLister{result: &appintegrations.ListAllProviderWebhooksResult{}}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}

func TestListAvailableWebhookTopicsUseCase_seedsKnownCatalogEvenWithNoTrafficOrSubscriptions(t *testing.T) {
	events := &fakeTopicsEventRepo{}
	subs := &fakeSubscriptionsLister{result: &appintegrations.ListAllProviderWebhooksResult{}}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Contains(t, out.AllEvents["woocommerce"], string(integrationsDomain.WooTopicOrderUpdated))
	assert.Contains(t, out.AllEvents["myparcel"], string(integrationsDomain.MyParcelHookShipmentStatusChange))
	assert.Contains(t, out.AllEvents["myparcel"], string(integrationsDomain.MyParcelHookShopUpdated))
}
