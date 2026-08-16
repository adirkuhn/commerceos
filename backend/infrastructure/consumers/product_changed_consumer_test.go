package consumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
)

type stubProcessedEvents struct {
	processed map[string]bool // eventID+consumer -> done
	isErr     error
	markErr   error
	marked    []string
}

func key(eventID, consumer string) string { return eventID + "|" + consumer }

func (s *stubProcessedEvents) IsProcessed(_ context.Context, eventID, consumer string) (bool, error) {
	if s.isErr != nil {
		return false, s.isErr
	}
	return s.processed[key(eventID, consumer)], nil
}
func (s *stubProcessedEvents) MarkProcessed(_ context.Context, eventID, consumer string) error {
	if s.markErr != nil {
		return s.markErr
	}
	s.marked = append(s.marked, key(eventID, consumer))
	return nil
}

type stubSyncer struct {
	calls []string
	err   error
}

func (s *stubSyncer) Execute(_ context.Context, productID string) error {
	s.calls = append(s.calls, productID)
	return s.err
}

func newMsg(t *testing.T, event events.ProductChanged) *nats.Msg {
	t.Helper()
	payload, err := json.Marshal(event)
	require.NoError(t, err)
	return &nats.Msg{Subject: event.EventName(), Data: payload}
}

func TestProductChangedConsumer_syncsAndMarksProcessedOnSuccess(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubSyncer{}
	c := consumers.NewProductChangedConsumer(processed, &stubSyncer{}, syncer, &stubDLQPublisher{})
	event := events.ProductChanged{
		Base:     events.Base{EventID: "evt-1", OccurredAt: time.Now()},
		TenantID: "tenant-1", ProductID: "product-1",
	}

	c.Handle(newMsg(t, event))

	require.Equal(t, []string{"product-1"}, syncer.calls)
	require.Equal(t, []string{"evt-1|product-changed-sync"}, processed.marked)
}

func TestProductChangedConsumer_skipsAlreadyProcessedEvents(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{"evt-1|product-changed-sync": true}}
	syncer := &stubSyncer{}
	c := consumers.NewProductChangedConsumer(processed, &stubSyncer{}, syncer, &stubDLQPublisher{})
	event := events.ProductChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", ProductID: "product-1"}

	c.Handle(newMsg(t, event))

	require.Empty(t, syncer.calls, "already-processed event must not re-run the sync")
}

func TestProductChangedConsumer_doesNotMarkProcessedWhenSyncFails(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubSyncer{err: errors.New("db down")}
	c := consumers.NewProductChangedConsumer(processed, &stubSyncer{}, syncer, &stubDLQPublisher{})
	event := events.ProductChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", ProductID: "product-1"}

	c.Handle(newMsg(t, event))

	require.Len(t, syncer.calls, 1)
	require.Empty(t, processed.marked, "a failed sync must stay eligible for redelivery, not look already-handled")
}

func TestProductChangedConsumer_dropsMalformedPayloadWithoutCallingSyncer(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	syncer := &stubSyncer{}
	c := consumers.NewProductChangedConsumer(processed, &stubSyncer{}, syncer, &stubDLQPublisher{})

	c.Handle(&nats.Msg{Subject: "pms.product.changed", Data: []byte("not json")})

	require.Empty(t, syncer.calls)
}

func TestProductChangedConsumer_runsEnrollerBeforeSyncer(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	enroller := &stubSyncer{}
	syncer := &stubSyncer{}
	c := consumers.NewProductChangedConsumer(processed, enroller, syncer, &stubDLQPublisher{})
	event := events.ProductChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", ProductID: "product-1"}

	c.Handle(newMsg(t, event))

	require.Equal(t, []string{"product-1"}, enroller.calls, "enroller must run so a brand-new product's first listing exists before syncer re-pushes")
	require.Equal(t, []string{"product-1"}, syncer.calls)
	require.Equal(t, []string{"evt-1|product-changed-sync"}, processed.marked)
}

func TestProductChangedConsumer_enrollerFailure_nacksWithoutRunningSyncerOrMarkingProcessed(t *testing.T) {
	processed := &stubProcessedEvents{processed: map[string]bool{}}
	enroller := &stubSyncer{err: errors.New("db down")}
	syncer := &stubSyncer{}
	c := consumers.NewProductChangedConsumer(processed, enroller, syncer, &stubDLQPublisher{})
	event := events.ProductChanged{Base: events.Base{EventID: "evt-1", OccurredAt: time.Now()}, TenantID: "tenant-1", ProductID: "product-1"}

	c.Handle(newMsg(t, event))

	require.Len(t, enroller.calls, 1)
	require.Empty(t, syncer.calls, "syncer must not run if enrollment failed")
	require.Empty(t, processed.marked)
}
