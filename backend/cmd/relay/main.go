package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/publishers"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/config"
	"github.com/adirkuhn/commerceos/internal/db"
)

// maxOutboxPublishRetries caps how many genuine per-message publish
// failures (not NATS/JetStream connectivity errors -- see
// isConnectivityErr) a row tolerates before the relay gives up on it and
// marks it Failed for manual triage. A connectivity error never counts
// against this budget: the row just stays Pending and is retried on the
// next tick for as long as NATS/JetStream is unreachable, so an outage of
// any length never strands a row on its own.
const maxOutboxPublishRetries = 10

// relayOutboxRepo is outbox.Repository narrowed to what the relay's publish
// loop needs (ISP) -- RecordPublishFailure isn't part of the shared
// domain/outbox.Repository interface (dozens of unrelated use cases across
// the app only ever Enqueue; adding a relay-specific method there would
// force every one of their test stubs to implement it for no reason).
type relayOutboxRepo interface {
	FetchPending(ctx context.Context, limit int) ([]outbox.Message, error)
	MarkPublished(ctx context.Context, id uint) error
	RecordPublishFailure(ctx context.Context, id uint, maxRetries int) error
}

// jsPublisher is nats.JetStreamContext narrowed to point of use (ISP) --
// lets publish be unit tested without a real NATS connection.
type jsPublisher interface {
	Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error)
}

// connChecker is *nats.Conn narrowed to point of use (ISP), same reason.
type connChecker interface {
	IsConnected() bool
}

func main() {
	cfg := config.Load()

	// RetryOnFailedConnect+MaxReconnects(-1): if NATS/JetStream is down at
	// startup or drops mid-run, Connect returns immediately (does not
	// block) and the client keeps retrying in the background forever --
	// the relay stays up and simply skips ticks (see publish's IsConnected
	// check) instead of crash-looping via log.Fatalf until an operator
	// notices.
	nc, err := nats.Connect(cfg.NatsURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Printf("relay: nats disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) { log.Println("relay: nats reconnected") }),
	)
	if err != nil {
		log.Fatalf("nats connect: %v", err)
	}
	defer nc.Drain()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}
	if err := publishers.EnsureStream(js); err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	db, err := db.Open(cfg.DBDsn, 10)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	repo := repositories.NewOutboxRepository(db)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Println("relay started")
	for {
		select {
		case <-ctx.Done():
			log.Println("relay shutting down")
			return
		case <-ticker.C:
			publish(ctx, repo, js, nc)
		}
	}
}

func publish(ctx context.Context, repo relayOutboxRepo, js jsPublisher, nc connChecker) {
	if !nc.IsConnected() {
		log.Println("relay: nats not connected, skipping this tick")
		return
	}

	msgs, err := repo.FetchPending(ctx, 50)
	if err != nil {
		log.Printf("fetch pending: %v", err)
		return
	}
	for _, m := range msgs {
		if _, err := js.Publish(m.Subject, m.Payload); err != nil {
			if isConnectivityErr(err) {
				log.Printf("publish %s: connectivity error, will retry next tick: %v", m.EventID, err)
				continue
			}
			log.Printf("publish %s: %v", m.EventID, err)
			if failErr := repo.RecordPublishFailure(ctx, m.ID, maxOutboxPublishRetries); failErr != nil {
				log.Printf("record publish failure %s: %v", m.EventID, failErr)
			}
			continue
		}
		if err := repo.MarkPublished(ctx, m.ID); err != nil {
			// Row stays pending and is republished next tick -- harmless
			// (consumers dedupe on EventID) but must be visible.
			log.Printf("mark published %s: %v", m.EventID, err)
		}
	}
}

// isConnectivityErr distinguishes "NATS/JetStream itself is unreachable"
// from "this specific message failed to publish" -- only the latter counts
// against maxOutboxPublishRetries. A false negative here (treating a
// connectivity error as a message failure) just means a row burns retries
// it shouldn't during an outage; a false positive (treating a real message
// failure as connectivity) means it retries forever instead of eventually
// landing in Failed for triage -- so this stays a narrow, explicit allowlist
// rather than a broad heuristic.
func isConnectivityErr(err error) bool {
	return errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrTimeout) ||
		errors.Is(err, nats.ErrNoResponders) ||
		errors.Is(err, context.DeadlineExceeded)
}
