package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/webhooks"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubWebhookProcessor struct {
	calls []webhooks.Envelope
	err   error
}

func (s *stubWebhookProcessor) Execute(_ context.Context, envelope webhooks.Envelope) error {
	s.calls = append(s.calls, envelope)
	return s.err
}

func newWebhookMsg(t *testing.T, envelope webhooks.Envelope) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(envelope)
	require.NoError(t, err)
	return &nats.Msg{Subject: "webhooks.received." + envelope.Provider, Data: payload}
}

func webhookDedupeKey(eventID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(eventID)).String()
}

func TestWebhookConsumer_processesAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	processor := &stubWebhookProcessor{}
	c := consumers.NewWebhookConsumer(processed, processor, &stubDLQPublisher{})
	envelope := webhooks.Envelope{EventID: "hash-abc", TenantID: "tenant-1", Provider: "woocommerce", ReceivedAt: time.Now(), RawPayload: []byte(`{}`)}

	c.Handle(newWebhookMsg(t, envelope))

	require.Len(t, processor.calls, 1)
	require.Equal(t, "tenant-1", processor.calls[0].TenantID)
	require.Equal(t, []string{key(webhookDedupeKey("hash-abc"), "webhook-processor")}, processed.marked)
}

func TestWebhookConsumer_skipsAlreadyProcessedEvent(t *testing.T) {
	dedupe := webhookDedupeKey("hash-abc")
	processed := &stubProcessedEvents{processed: map[string]bool{key(dedupe, "webhook-processor"): true}}
	processor := &stubWebhookProcessor{}
	c := consumers.NewWebhookConsumer(processed, processor, &stubDLQPublisher{})
	envelope := webhooks.Envelope{EventID: "hash-abc", TenantID: "tenant-1", Provider: "woocommerce"}

	c.Handle(newWebhookMsg(t, envelope))

	require.Empty(t, processor.calls)
}

func TestWebhookConsumer_dropsMalformedPayload(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	processor := &stubWebhookProcessor{}
	c := consumers.NewWebhookConsumer(processed, processor, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "webhooks.received.woocommerce", Data: []byte("not json")})

	require.Empty(t, processor.calls)
}

func TestWebhookConsumer_leavesUnprocessedOnProcessorError(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	processor := &stubWebhookProcessor{err: errors.New("db down")}
	c := consumers.NewWebhookConsumer(processed, processor, &stubDLQPublisher{})
	envelope := webhooks.Envelope{EventID: "hash-abc", TenantID: "tenant-1", Provider: "woocommerce"}

	c.Handle(newWebhookMsg(t, envelope))

	require.Empty(t, processed.marked)
}
