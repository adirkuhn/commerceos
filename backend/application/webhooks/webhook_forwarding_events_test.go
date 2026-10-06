package webhooks_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appwebhooks "github.com/adirkuhn/commerceos/application/webhooks"
	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreateForwardRule_RecordsCreatedWithTargetHostOnly(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}

	_, err := appwebhooks.NewCreateForwardRuleUseCase(newMockGormDB(t, 1), &fakeForwardRuleRepoAdmin{}, outbox).
		Execute(context.Background(), "woocommerce", "order.updated", "https://hooks.example.com/in?token=s3cret")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "webhook_forwarding.forward_rule.created", e.Subject)
	assert.Equal(t, "rule-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"provider": "woocommerce", "topic": "order.updated", "target_host": "hooks.example.com"}, e.Data())
}

func TestDeleteForwardRule_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}

	require.NoError(t, appwebhooks.NewDeleteForwardRuleUseCase(newMockGormDB(t, 1), &fakeForwardRuleRepoAdmin{}, outbox).Execute(context.Background(), "rule-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "webhook_forwarding.forward_rule.deleted", e.Subject)
	assert.Equal(t, "rule-1", e.Payload["entity_id"])
}

func TestRetryWebhookFallbackEvent_RecordsRepublished(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &listingFallbackRepo{events: []webhooks.FallbackEvent{{ID: "fb-1", Provider: "myparcel", Status: webhooks.FallbackStatusPendingRepublish}}}

	require.NoError(t, appwebhooks.NewRetryWebhookFallbackEventUseCase(newMockGormDB(t, 2), repo, &fakePublisher{}, outbox).Execute(context.Background(), "fb-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "webhook_forwarding.fallback_event.republished", e.Subject)
	assert.Equal(t, map[string]any{"provider": "myparcel"}, e.Data())
}

func TestAbandonWebhookFallbackEvent_RecordsAbandoned(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &listingFallbackRepo{events: []webhooks.FallbackEvent{{ID: "fb-1", Provider: "myparcel", Status: webhooks.FallbackStatusPendingRepublish}}}

	require.NoError(t, appwebhooks.NewAbandonWebhookFallbackEventUseCase(newMockGormDB(t, 1), repo, outbox).Execute(context.Background(), "fb-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "webhook_forwarding.fallback_event.abandoned", e.Subject)
	assert.Equal(t, "fb-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"provider": "myparcel"}, e.Data())
}

func TestAbandonAllWebhookFallbackEvents_RecordsBulkAbandoned(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &listingFallbackRepo{events: []webhooks.FallbackEvent{
		{ID: "a", Status: webhooks.FallbackStatusPendingRepublish}, {ID: "b", Status: webhooks.FallbackStatusPendingRepublish},
	}}

	_, err := appwebhooks.NewAbandonAllWebhookFallbackEventsUseCase(newMockGormDB(t, 1), repo, outbox).
		Execute(tenant.WithTenantID(context.Background(), "tenant-1"), "")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "webhook_forwarding.fallback_event.bulk_abandoned", e.Subject)
	assert.Equal(t, "tenant-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"from_status": string(webhooks.FallbackStatusPendingRepublish), "count": float64(2)}, e.Data())
}

func TestWebhookAdminUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}

	_, err := appwebhooks.NewCreateForwardRuleUseCase(newMockGormDB(t, 1), &fakeForwardRuleRepoAdmin{}, failing).
		Execute(context.Background(), "woocommerce", "", "https://hooks.example.com")
	assert.ErrorIs(t, err, boom)
	assert.ErrorIs(t, appwebhooks.NewDeleteForwardRuleUseCase(newMockGormDB(t, 1), &fakeForwardRuleRepoAdmin{}, failing).Execute(context.Background(), "r"), boom)
}
