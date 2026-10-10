package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type stubNotificationEmailSender struct {
	calls   []appnotifications.SendNotificationEmailInput
	tenants []string
	err     error
}

func (s *stubNotificationEmailSender) Execute(ctx context.Context, in appnotifications.SendNotificationEmailInput) error {
	s.calls = append(s.calls, in)
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func emailRequestedMsg(t *testing.T) *nats.Msg {
	t.Helper()
	data, err := json.Marshal(events.NotificationEmailRequested{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", Type: "import.failed", Recipients: []string{"ops@example.nl"},
		Params:  map[string]any{"error": "boom"},
		Subject: "Import failed", Body: "boom", Link: "/sync-log",
	})
	require.NoError(t, err)
	return &nats.Msg{Subject: "notifications.email.requested", Data: data}
}

func TestNotificationEmailConsumer_sendsForTheEventsTenantAndMarksProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	sender := &stubNotificationEmailSender{}
	c := consumers.NewNotificationEmailConsumer(processed, sender, &stubDLQPublisher{})

	c.Handle(emailRequestedMsg(t))

	require.Equal(t, []appnotifications.SendNotificationEmailInput{{
		Recipients: []string{"ops@example.nl"}, Type: "import.failed", Params: map[string]any{"error": "boom"},
		Subject: "Import failed", Body: "boom", Link: "/sync-log",
	}}, sender.calls)
	require.Equal(t, []string{"tenant-1"}, sender.tenants)
	require.Equal(t, []string{"evt-1|notification-email-send"}, processed.marked)
}

func TestNotificationEmailConsumer_duplicateDeliverySendsNothing(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|notification-email-send": true}}
	sender := &stubNotificationEmailSender{}
	c := consumers.NewNotificationEmailConsumer(processed, sender, &stubDLQPublisher{})

	c.Handle(emailRequestedMsg(t))

	require.Empty(t, sender.calls)
}

func TestNotificationEmailConsumer_failedSendStaysEligibleForRedelivery(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	sender := &stubNotificationEmailSender{err: errors.New("smtp down")}
	c := consumers.NewNotificationEmailConsumer(processed, sender, &stubDLQPublisher{})

	c.Handle(emailRequestedMsg(t))

	require.Len(t, sender.calls, 1)
	require.Empty(t, processed.marked)
}

func TestNotificationEmailConsumer_malformedPayloadIsDropped(t *testing.T) {
	sender := &stubNotificationEmailSender{}
	c := consumers.NewNotificationEmailConsumer(&stubProcessedEvents{processed: map[string]bool{}}, sender, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "notifications.email.requested", Data: []byte("nope")})

	require.Empty(t, sender.calls)
}
