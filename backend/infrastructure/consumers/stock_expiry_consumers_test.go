package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// --- bell ---

type stubExpiryBell struct {
	calls   [][2]int
	tenants []string
	err     error
}

func (s *stubExpiryBell) Execute(ctx context.Context, windowDays, itemCount int) error {
	s.calls = append(s.calls, [2]int{windowDays, itemCount})
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func expiryReportMsg(t *testing.T) *nats.Msg {
	t.Helper()
	ev := events.StockCloseToExpiryReported{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", WindowDays: 30, ItemCount: 4}
	payload, err := json.Marshal(ev)
	require.NoError(t, err)
	return &nats.Msg{Subject: ev.EventName(), Data: payload}
}

func TestStockExpiryBellConsumer_notifiesUnderEventTenantAndMarksProcessed(t *testing.T) {
	processed, bell := &stubProcessedEvents{processed: map[string]bool{}}, &stubExpiryBell{}
	c := consumers.NewStockExpiryBellConsumer(processed, bell, &stubDLQPublisher{})

	c.Handle(expiryReportMsg(t))

	assert.Equal(t, [][2]int{{30, 4}}, bell.calls)
	assert.Equal(t, []string{"tenant-1"}, bell.tenants)
	assert.Equal(t, []string{"evt-1|stock-close-to-expiry-bell"}, processed.marked)
}

func TestStockExpiryBellConsumer_skipsDuplicates(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|stock-close-to-expiry-bell": true}}
	bell := &stubExpiryBell{}
	consumers.NewStockExpiryBellConsumer(processed, bell, &stubDLQPublisher{}).Handle(expiryReportMsg(t))

	assert.Empty(t, bell.calls)
}

func TestStockExpiryBellConsumer_failureIsNotMarkedProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	consumers.NewStockExpiryBellConsumer(processed, &stubExpiryBell{err: errors.New("db down")}, &stubDLQPublisher{}).Handle(expiryReportMsg(t))

	assert.Empty(t, processed.marked)
}

// --- Message Center defaults on settings change ---

type stubDefaultsEnsurer struct {
	tenants []string
	err     error
}

func (s *stubDefaultsEnsurer) Execute(ctx context.Context) error {
	s.tenants = append(s.tenants, tenant.IDFromContext(ctx))
	return s.err
}

func settingsUpdatedMsg(t *testing.T, changes map[string]events.FieldChange) *nats.Msg {
	t.Helper()
	ev := events.EntityAction{Base: events.Base{EventID: "evt-2", OccurredAt: time.Now()}, Type: events.TenantSettingsUpdated,
		TenantID: "tenant-1", EntityID: "tenant-1", Changes: changes}
	payload, err := json.Marshal(ev)
	require.NoError(t, err)
	return &nats.Msg{Subject: ev.EventName(), Data: payload}
}

func TestStockExpiryDefaultsConsumer_turningTheAlertOnEnsuresDefaults(t *testing.T) {
	processed, ensurer := &stubProcessedEvents{processed: map[string]bool{}}, &stubDefaultsEnsurer{}
	c := consumers.NewStockExpiryDefaultsConsumer(processed, ensurer, &stubDLQPublisher{})

	c.Handle(settingsUpdatedMsg(t, map[string]events.FieldChange{"expiry_alert_enabled": {Before: false, After: true}}))

	assert.Equal(t, []string{"tenant-1"}, ensurer.tenants)
	assert.Equal(t, []string{"evt-2|stock-expiry-message-defaults"}, processed.marked)
}

func TestStockExpiryDefaultsConsumer_ignoresOtherChanges(t *testing.T) {
	for name, changes := range map[string]map[string]events.FieldChange{
		"other field": {"timezone": {Before: "", After: "Europe/Amsterdam"}},
		"turned off":  {"expiry_alert_enabled": {Before: true, After: false}},
		"no changes":  nil,
	} {
		t.Run(name, func(t *testing.T) {
			processed, ensurer := &stubProcessedEvents{processed: map[string]bool{}}, &stubDefaultsEnsurer{}
			consumers.NewStockExpiryDefaultsConsumer(processed, ensurer, &stubDLQPublisher{}).Handle(settingsUpdatedMsg(t, changes))

			assert.Empty(t, ensurer.tenants)
			assert.Len(t, processed.marked, 1, "acked and marked, nothing to retry")
		})
	}
}

func TestStockExpiryDefaultsConsumer_failureIsNotMarkedProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	c := consumers.NewStockExpiryDefaultsConsumer(processed, &stubDefaultsEnsurer{err: errors.New("db down")}, &stubDLQPublisher{})

	c.Handle(settingsUpdatedMsg(t, map[string]events.FieldChange{"expiry_alert_enabled": {Before: false, After: true}}))

	assert.Empty(t, processed.marked)
}
