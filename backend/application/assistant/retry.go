package assistant

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/assistant"
	"github.com/adirkuhn/commerceos/domain/events"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// RetryRequestUseCase makes a request waiting after a failed attempt run on
// the worker's next poll, with its backoff reset.
type RetryRequestUseCase struct {
	db       *gorm.DB
	requests assistant.RequestRepository
	outbox   eventrecord.Enqueuer
	now      func() time.Time
}

func NewRetryRequestUseCase(db *gorm.DB, requests assistant.RequestRepository, outbox eventrecord.Enqueuer, now func() time.Time) *RetryRequestUseCase {
	return &RetryRequestUseCase{db: db, requests: requests, outbox: outbox, now: now}
}

func (uc *RetryRequestUseCase) Execute(ctx context.Context, id string) (*assistant.Request, error) {
	var out *assistant.Request
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		r, err := uc.requests.GetForUpdate(ctx, id)
		if err != nil {
			return err
		}
		if err := r.RetryNow(uc.now()); err != nil {
			return err
		}
		if err := uc.requests.Update(ctx, r); err != nil {
			return err
		}
		out = r
		return eventrecord.Record(ctx, uc.outbox, events.AssistantRequestRetried, r.ID,
			eventrecord.Data{"wait_reason": string(r.WaitReason)})
	})
	if err != nil {
		return nil, fmt.Errorf("retryAssistantRequest: %w", err)
	}
	return out, nil
}
