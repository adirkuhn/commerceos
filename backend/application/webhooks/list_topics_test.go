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

func TestListAvailableWebhookTopicsUseCase_unionsReceivedAndSubscribedTopics(t *testing.T) {
	events := &fakeTopicsEventRepo{refs: []webhooks.TopicRef{
		{Provider: "woocommerce", Topic: "order.updated"},
		{Provider: "woocommerce", Topic: "order.created"},
	}}
	subs := &fakeSubscriptionsLister{result: &appintegrations.ListAllProviderWebhooksResult{
		Webhooks: []integrationsDomain.ProviderWebhook{
			{Provider: integrationsDomain.ProviderWooCommerce, Topic: "product.updated"},
			{Provider: integrationsDomain.ProviderMyParcel, Topic: "shipment_status_change"},
		},
	}}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"order.created", "order.updated", "product.updated"}, out["woocommerce"])
	assert.Equal(t, []string{"shipment_status_change"}, out["myparcel"])
}

func TestListAvailableWebhookTopicsUseCase_dedupesAcrossBothSources(t *testing.T) {
	events := &fakeTopicsEventRepo{refs: []webhooks.TopicRef{{Provider: "woocommerce", Topic: "order.updated"}}}
	subs := &fakeSubscriptionsLister{result: &appintegrations.ListAllProviderWebhooksResult{
		Webhooks: []integrationsDomain.ProviderWebhook{{Provider: integrationsDomain.ProviderWooCommerce, Topic: "order.updated"}},
	}}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"order.updated"}, out["woocommerce"])
}

func TestListAvailableWebhookTopicsUseCase_subscriptionsFailure_stillReturnsReceivedTopics(t *testing.T) {
	events := &fakeTopicsEventRepo{refs: []webhooks.TopicRef{{Provider: "woocommerce", Topic: "order.updated"}}}
	subs := &fakeSubscriptionsLister{err: errors.New("provider api down")}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	out, err := uc.Execute(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"order.updated"}, out["woocommerce"])
}

func TestListAvailableWebhookTopicsUseCase_eventRepoFailure_propagatesError(t *testing.T) {
	events := &fakeTopicsEventRepo{err: errors.New("db down")}
	subs := &fakeSubscriptionsLister{result: &appintegrations.ListAllProviderWebhooksResult{}}
	uc := appwebhooks.NewListAvailableWebhookTopicsUseCase(newMockGormDB(t, 1), events, subs)

	_, err := uc.Execute(context.Background())

	require.Error(t, err)
}
