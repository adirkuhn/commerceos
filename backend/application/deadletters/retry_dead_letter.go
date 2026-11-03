package deadletters

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/deadletter"
	"github.com/adirkuhn/commerceos/domain/events"
)

// RetryDeadLetterUseCase publishes a dead letter's event again for the
// consumer that failed it (every consumer on the subject when unknown;
// each skips an event it already processed), then removes it from the
// dead-letter queue.
//
// Publishing to NATS can't share a transaction with the outbox, so the
// steps run in the order that never loses an event: publish, then delete,
// then audit. A failed publish leaves the dead letter in place. A failed
// delete after a publish is reported; retrying again is safe because
// consumers deduplicate by event ID.
type RetryDeadLetterUseCase struct {
	db        *gorm.DB
	store     getDeleter
	publisher deadletter.RetryPublisher
	outbox    eventrecord.Enqueuer
}

func NewRetryDeadLetterUseCase(db *gorm.DB, store getDeleter, publisher deadletter.RetryPublisher, outbox eventrecord.Enqueuer) *RetryDeadLetterUseCase {
	return &RetryDeadLetterUseCase{db: db, store: store, publisher: publisher, outbox: outbox}
}

func (uc *RetryDeadLetterUseCase) Execute(ctx context.Context, stream deadletter.Stream, seq uint64) error {
	if err := checkAccess(ctx, stream); err != nil {
		return fmt.Errorf("retryDeadLetter: %w", err)
	}
	d, err := uc.store.Get(ctx, stream, seq)
	if err != nil {
		return fmt.Errorf("retryDeadLetter: %w", err)
	}
	if err := uc.publisher.Republish(ctx, d); err != nil {
		return fmt.Errorf("retryDeadLetter: %w: %w", deadletter.ErrRetryFailed, err)
	}
	if err := uc.store.Delete(ctx, stream, seq); err != nil {
		return fmt.Errorf("retryDeadLetter: republished but not removed: %w", err)
	}
	if err := recordAction(ctx, uc.db, uc.outbox, events.DeadLetterRetried, d, auditData(d)); err != nil {
		return fmt.Errorf("retryDeadLetter: %w", err)
	}
	return nil
}
