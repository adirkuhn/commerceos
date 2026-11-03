package deadletters

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/deadletter"
	"github.com/adirkuhn/commerceos/domain/events"
)

// DiscardDeadLetterUseCase deletes one dead letter without processing it.
// The audit event keeps the failure it was discarded with.
type DiscardDeadLetterUseCase struct {
	db     *gorm.DB
	store  getDeleter
	outbox eventrecord.Enqueuer
}

func NewDiscardDeadLetterUseCase(db *gorm.DB, store getDeleter, outbox eventrecord.Enqueuer) *DiscardDeadLetterUseCase {
	return &DiscardDeadLetterUseCase{db: db, store: store, outbox: outbox}
}

func (uc *DiscardDeadLetterUseCase) Execute(ctx context.Context, stream deadletter.Stream, seq uint64) error {
	if err := checkAccess(ctx, stream); err != nil {
		return fmt.Errorf("discardDeadLetter: %w", err)
	}
	d, err := uc.store.Get(ctx, stream, seq)
	if err != nil {
		return fmt.Errorf("discardDeadLetter: %w", err)
	}
	if err := uc.store.Delete(ctx, stream, seq); err != nil {
		return fmt.Errorf("discardDeadLetter: %w", err)
	}
	data := auditData(d)
	data["error"] = d.Error
	if err := recordAction(ctx, uc.db, uc.outbox, events.DeadLetterDiscarded, d, data); err != nil {
		return fmt.Errorf("discardDeadLetter: %w", err)
	}
	return nil
}
