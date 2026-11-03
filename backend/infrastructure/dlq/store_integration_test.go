//go:build integration

// Runs Store against a real JetStream (nats:2.10-alpine, the version
// docker-compose runs), one container for the whole package:
//
//	go test -tags=integration ./infrastructure/dlq/...
package dlq_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/adirkuhn/commerceos/domain/deadletter"
	"github.com/adirkuhn/commerceos/infrastructure/consumers"
	"github.com/adirkuhn/commerceos/infrastructure/dlq"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
)

var natsURL string

func TestMain(m *testing.M) {
	ctx := context.Background()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			Cmd:          []string{"-js"},
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForLog("Server is ready"),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "start nats:", err)
		os.Exit(1)
	}
	endpoint, err := c.PortEndpoint(ctx, "4222/tcp", "nats")
	if err != nil {
		fmt.Fprintln(os.Stderr, "nats endpoint:", err)
		os.Exit(1)
	}
	natsURL = endpoint
	code := m.Run()
	_ = c.Terminate(ctx) // best effort; the container is reaped by testcontainers anyway
	os.Exit(code)
}

// freshStreams connects and recreates the event, webhook and DLQ streams
// empty, so each test starts from a known state.
func freshStreams(t *testing.T) (*nats.Conn, nats.JetStreamContext, jetstream.JetStream) {
	t.Helper()
	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	legacy, err := nc.JetStream()
	require.NoError(t, err)
	for _, name := range []string{publishers.StreamName, publishers.DLQStreamName, publishers.WebhookStreamName, publishers.WebhookDLQStreamName} {
		if err := legacy.DeleteStream(name); err != nil && err != nats.ErrStreamNotFound {
			require.NoError(t, err)
		}
	}
	require.NoError(t, publishers.EnsureStream(legacy))
	require.NoError(t, publishers.EnsureDLQStream(legacy))
	require.NoError(t, publishers.EnsureWebhookStream(legacy))
	require.NoError(t, publishers.EnsureWebhookDLQStream(legacy))
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	return nc, legacy, js
}

func deadLetter(t *testing.T, legacy nats.JetStreamContext, subject, tenantID, consumer string) {
	t.Helper()
	m := &nats.Msg{Subject: dlq.Subject(subject), Data: []byte(fmt.Sprintf(`{"event_id":"e-%s-%s","tenant_id":%q}`, tenantID, consumer, tenantID))}
	if consumer != "" {
		m.Header = dlq.FailureHeader(consumer, subject, fmt.Errorf("boom"), 5, time.Now())
	}
	_, err := legacy.PublishMsg(m)
	require.NoError(t, err)
}

func TestStore_listFiltersPagesAndNeverConsumes(t *testing.T) {
	_, legacy, js := freshStreams(t)
	deadLetter(t, legacy, "oms.sales_order.status_changed", "t-1", "c1")
	deadLetter(t, legacy, "pms.product.changed", "t-2", "")
	deadLetter(t, legacy, "oms.sales_order.status_changed", "t-1", "c2")
	store := dlq.NewStore(js)
	ctx := context.Background()

	first, err := store.List(ctx, deadletter.Filter{Stream: deadletter.StreamEvents, Limit: 2})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	assert.Equal(t, uint64(3), first.Total)
	assert.Equal(t, "c1", first.Items[0].Consumer)
	assert.False(t, first.Items[1].ConsumerKnown(), "old dead letter without headers still lists")
	assert.Equal(t, "pms.product.changed", first.Items[1].Subject)

	second, err := store.List(ctx, deadletter.Filter{Stream: deadletter.StreamEvents, Limit: 2, After: first.NextCursor})
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	assert.Equal(t, "c2", second.Items[0].Consumer)
	assert.Zero(t, second.NextCursor)

	filtered, err := store.List(ctx, deadletter.Filter{Stream: deadletter.StreamEvents, TenantID: "t-1", Consumer: "c2"})
	require.NoError(t, err)
	require.Len(t, filtered.Items, 1)
	assert.Equal(t, uint64(3), filtered.Items[0].Sequence)

	again, err := store.List(ctx, deadletter.Filter{Stream: deadletter.StreamEvents})
	require.NoError(t, err)
	assert.Len(t, again.Items, 3, "reading consumed nothing")
}

func TestStore_getAndDeleteExactlyOne(t *testing.T) {
	_, legacy, js := freshStreams(t)
	deadLetter(t, legacy, "oms.sales_order.status_changed", "t-1", "c1")
	deadLetter(t, legacy, "oms.sales_order.status_changed", "t-1", "c2")
	store := dlq.NewStore(js)
	ctx := context.Background()

	got, err := store.Get(ctx, deadletter.StreamEvents, 1)
	require.NoError(t, err)
	assert.Equal(t, "c1", got.Consumer)
	assert.Equal(t, "boom", got.Error)
	assert.Equal(t, 5, got.NumDelivered)

	require.NoError(t, store.Delete(ctx, deadletter.StreamEvents, 1))

	_, err = store.Get(ctx, deadletter.StreamEvents, 1)
	assert.ErrorIs(t, err, deadletter.ErrNotFound)
	assert.ErrorIs(t, store.Delete(ctx, deadletter.StreamEvents, 1), deadletter.ErrNotFound)
	page, err := store.List(ctx, deadletter.Filter{Stream: deadletter.StreamEvents})
	require.NoError(t, err)
	require.Len(t, page.Items, 1, "only the deleted one is gone")
	assert.Equal(t, "c2", page.Items[0].Consumer)
}

func TestStore_webhookStream(t *testing.T) {
	_, legacy, js := freshStreams(t)
	_, err := legacy.Publish(dlq.Subject("webhooks.received.woocommerce"), []byte(`{"EventID":"sha","TenantID":"t-9"}`))
	require.NoError(t, err)

	page, err := dlq.NewStore(js).List(context.Background(), deadletter.Filter{Stream: deadletter.StreamWebhooks})

	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "webhooks.received.woocommerce", page.Items[0].Subject)
	assert.Equal(t, "t-9", page.Items[0].TenantID)
}

// TestStore_retryReachesOnlyTheFailedConsumer is the end-to-end promise of
// COM-14: two durable consumers on one subject, the retry runs only the one
// that failed; an old dead letter (consumer unknown) reaches both.
func TestStore_retryReachesOnlyTheFailedConsumer(t *testing.T) {
	_, legacy, js := freshStreams(t)
	const subject = "oms.sales_order.status_changed"
	var mu sync.Mutex
	got := map[string][]string{}
	for _, durable := range []string{"push-to-woo", "whatsapp-notify"} {
		durable := durable
		sub, err := legacy.Subscribe(subject, consumers.OnlyRetriesFor(durable, func(m *nats.Msg) {
			mu.Lock()
			got[durable] = append(got[durable], string(m.Data))
			mu.Unlock()
			require.NoError(t, m.Ack())
		}), nats.Durable(durable), nats.ManualAck())
		require.NoError(t, err)
		t.Cleanup(func() { _ = sub.Unsubscribe() })
	}
	store := dlq.NewStore(js)
	ctx := context.Background()

	require.NoError(t, store.Republish(ctx, deadletter.DeadLetter{Subject: subject, Consumer: "push-to-woo", Data: []byte(`targeted`)}))
	require.NoError(t, store.Republish(ctx, deadletter.DeadLetter{Subject: subject, Data: []byte(`everyone`)}))
	// Same data again: a fresh Nats-Msg-Id each time, so JetStream's
	// duplicate window doesn't drop a second retry.
	require.NoError(t, store.Republish(ctx, deadletter.DeadLetter{Subject: subject, Data: []byte(`everyone`)}))

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got["push-to-woo"]) == 3 && len(got["whatsapp-notify"]) == 2
	}, 5*time.Second, 50*time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{"targeted", "everyone", "everyone"}, got["push-to-woo"])
	assert.Equal(t, []string{"everyone", "everyone"}, got["whatsapp-notify"])
}
