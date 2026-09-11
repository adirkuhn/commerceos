// Command backfill-outbox-replay requeues outbox_messages rows stuck in
// Failed back to Pending so cmd/relay's next tick retries publishing them
// -- the relay only ever fetches Pending rows (see cmd/relay's
// FetchPending call), so a row that already exhausted its retry budget
// stays Failed forever on its own, even after whatever made it fail is
// fixed.
//
// Written for the 2026-09-11 incident where "shipping.>" was missing from
// the JetStream stream's subjects (see infrastructure/publishers/
// stream.go): every ShipmentFulfillmentSignal published under
// "shipping.*" failed with "no stream matches subject", and after 10
// attempts (cmd/relay's maxOutboxPublishRetries) landed in Failed. Once
// the stream fix is deployed, publishing those subjects works again --
// but the already-Failed rows still need this one-off requeue to actually
// go out, since the relay won't touch them on its own.
//
// -subject-prefix scopes the requeue to one event subject prefix at a
// time (ListFailedBySubjectPrefix is a prefix match, not "every Failed
// row") so replaying this incident can't accidentally sweep up an
// unrelated Failed row still awaiting manual triage for a different
// reason.
//
// Safe to run more than once -- a row already back to Pending (or since
// published) is no longer Failed, so a repeat run finds nothing to do for
// it.
//
// Usage:
//
//	go run ./cmd/backfill-outbox-replay -subject-prefix=shipping.              # requeue
//	go run ./cmd/backfill-outbox-replay -subject-prefix=shipping. -dry-run     # list only, no writes
package main

import (
	"context"
	"flag"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/domain/outbox"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/internal/config"
)

// outboxReplayRepo is outbox.Repository's would-be shape narrowed to what
// this tool needs (ISP, same reasoning as cmd/relay's relayOutboxRepo) --
// neither method belongs on the shared domain/outbox.Repository interface
// since every other caller only ever Enqueues.
type outboxReplayRepo interface {
	ListFailedBySubjectPrefix(ctx context.Context, prefix string) ([]outbox.Message, error)
	ResetToPending(ctx context.Context, id uint) error
}

func main() {
	subjectPrefix := flag.String("subject-prefix", "", "requeue Failed rows whose subject starts with this prefix, e.g. \"shipping.\" (required)")
	dryRun := flag.Bool("dry-run", false, "list matching Failed rows without requeuing them")
	flag.Parse()

	if *subjectPrefix == "" {
		log.Fatalf("-subject-prefix is required -- refusing to guess at which incident this replay is for")
	}

	cfg := config.Load()
	db, err := gorm.Open(postgres.Open(cfg.DBDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	repo := repositories.NewOutboxRepository(db)
	ran, err := run(context.Background(), repo, *subjectPrefix, *dryRun)
	if err != nil {
		log.Fatalf("replay: %v", err)
	}

	if ran == 0 {
		log.Printf("no Failed rows found with subject prefix %q", *subjectPrefix)
	}
}

func run(ctx context.Context, repo outboxReplayRepo, subjectPrefix string, dryRun bool) (int, error) {
	failed, err := repo.ListFailedBySubjectPrefix(ctx, subjectPrefix)
	if err != nil {
		return 0, err
	}

	for _, m := range failed {
		if dryRun {
			log.Printf("[dry-run] would requeue id=%d subject=%s event_id=%s retry_count=%d occurred_at=%s",
				m.ID, m.Subject, m.EventID, m.RetryCount, m.OccurredAt)
			continue
		}
		if err := repo.ResetToPending(ctx, m.ID); err != nil {
			log.Printf("requeue id=%d subject=%s event_id=%s: %v", m.ID, m.Subject, m.EventID, err)
			continue
		}
		log.Printf("requeued id=%d subject=%s event_id=%s (was retry_count=%d)", m.ID, m.Subject, m.EventID, m.RetryCount)
	}

	return len(failed), nil
}
